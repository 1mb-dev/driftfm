/**
 * Drift FM - Mood Radio Player
 *
 * States: idle -> playing
 * - idle: mood selector visible, player bar hidden
 * - playing: mood visual + player bar visible
 */

import { formatTime, formatEnergy, formatIntensity, getTrackDisplayName } from './utils/format.js';
import { storage } from './core/storage.js';
import { events } from './core/events.js';
import { reportListen } from './core/listen-reporter.js';
import { SettingsManager } from './ui/settings.js';
import { LyricsManager } from './ui/lyrics.js';
import { AboutManager } from './ui/about.js';
import { Galaxy } from './galaxy.js';

const KNOWN_MOODS = [
  { name: 'focus', display_name: 'Focus' },
  { name: 'calm', display_name: 'Calm' },
  { name: 'late_night', display_name: 'Late Night' },
  { name: 'energize', display_name: 'Energize' }
];

class DriftFMPlayer {
  constructor() {
    this.audioCurrent = document.getElementById('audio-current');
    this.audioNext = document.getElementById('audio-next');
    this.playBtn = document.getElementById('play-btn');
    this.skipBtn = document.getElementById('skip-btn');
    this.playIcon = document.getElementById('play-icon');
    this.pauseIcon = document.getElementById('pause-icon');
    this.trackName = document.getElementById('track-name');
    this.moodIndicator = document.getElementById('mood-indicator');
    this.moodGalaxy = document.getElementById('mood-galaxy');
    this.playerBar = document.getElementById('player-bar');
    this.progressFill = document.getElementById('progress-fill');
    this.progressRail = document.getElementById('progress-rail');
    this.currentTimeEl = document.getElementById('current-time');
    this.srAnnouncer = document.getElementById('sr-announcer');

    // Settings module (handles drawer, theme, volume, toggles)
    this.settings = new SettingsManager();

    // Moodlet discovery elements
    this.moodletToast = document.getElementById('moodlet-toast');
    this.moodletDeeperLabel = document.getElementById('moodlet-deeper');

    // Lyrics panel module
    this.lyrics = new LyricsManager();

    // About panel module
    this.about = new AboutManager();

    // Resume button in header
    this.resumeBtn = document.getElementById('resume-btn');
    this.resumeMoodEl = document.getElementById('resume-mood');

    // Galaxy layout engine
    this.galaxy = new Galaxy();

    this.isPlaying = false;
    this.currentMood = 'focus';
    this.moods = [];
    this.playlist = [];
    this.currentIndex = -1;
    this.currentTrack = null;
    this.isSelecting = false; // Prevent rapid selection race conditions
    this.errorRetryTimeout = null; // Track error retry timeout for cleanup
    this.errorRetryCount = 0; // Consecutive error retries (capped at 3)

    // Moodlet discovery state
    this.sessionPlayCount = 0; // Tracks played this session (per mood)
    this.moodletShownForMood = {}; // { focus: true, calm: false, ... }
    this.moodletTimeout = null; // Auto-dismiss timeout

    // Listen event tracking
    this.playStartTime = -1; // audio.currentTime when play started (-1 = not tracking)

    // Skip friction state (progressive soft friction)
    this.skipCount = 0; // Skips in current friction window
    this.skipResetTimeout = null; // Timer to reset skip count after engagement

    // Track info cycling state
    this.metadataCycleIndex = 0;
    this.metadataCycleInterval = null;

    // Bound handlers for proper removal
    this._onTimeUpdate = () => this.updateProgress();
    this._onEnded = () => this.onTrackEnded();
    this._onError = (e) => this.handleError(e);

    this.bindEvents();
  }

  bindEvents() {
    this.playBtn.addEventListener('click', () => this.togglePlay());
    if (this.skipBtn) {
      this.skipBtn.addEventListener('click', () => this.skipTrack());
    }

    // Bind audio events (use bound handlers for proper removal on swap)
    this.audioCurrent.addEventListener('timeupdate', this._onTimeUpdate);
    this.audioCurrent.addEventListener('ended', this._onEnded);
    this.audioCurrent.addEventListener('error', this._onError);

    // Mood indicator click returns to idle state
    this.moodIndicator.addEventListener('click', () => this.returnToIdle());
    this.moodIndicator.addEventListener('keydown', (e) => {
      if (e.code === 'Enter' || e.code === 'Space') {
        e.preventDefault();
        this.returnToIdle();
      }
    });

    // Progress rail seek (click and keyboard)
    this.progressRail.addEventListener('click', (e) => this.seek(e));
    this.progressRail.addEventListener('keydown', (e) => this.handleProgressKeydown(e));

    // Keyboard controls
    document.addEventListener('keydown', (e) => {
      // Don't intercept when focused on inputs
      if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return;

      // About panel toggle works in both idle and playing states
      if (e.code === 'KeyI') {
        e.preventDefault();
        this.about.toggle();
        return;
      }

      // Escape closes about panel in any state
      if (e.code === 'Escape' && this.about.isOpen()) {
        this.about.close();
        return;
      }

      if (document.body.dataset.state === 'playing') {
        switch (e.code) {
          case 'Space':
            e.preventDefault();
            this.togglePlay();
            break;
          case 'KeyN':
            e.preventDefault();
            this.skipTrack();
            break;
          case 'KeyM':
            e.preventDefault();
            this.toggleMute();
            break;
          case 'KeyL':
            e.preventDefault();
            this.lyrics.toggle();
            break;
          case 'Escape':
            // Close panels in order: settings, lyrics, then return to idle
            if (this.settings.isOpen()) {
              this.settings.close();
            } else if (this.lyrics.isOpen()) {
              this.lyrics.close();
            } else {
              this.returnToIdle();
            }
            break;
        }
      }
    });

    // Click outside to close panels (robust whitelist approach)
    // Only close panels when clicking on actual background elements
    // This prevents accidental closes when clicking on other UI elements
    document.addEventListener('click', (e) => {
      const backgroundElements = ['mood-visual', 'app', 'app__header', 'mood-galaxy'];
      const isBackground = backgroundElements.some(cls => e.target.classList.contains(cls));

      if (!isBackground) return; // Only process clicks on background

      // Close panels when clicking on background
      if (this.about.isOpen()) {
        this.about.close();
      }
      if (this.settings.isOpen()) {
        this.settings.close();
      }
      if (this.lyrics.isOpen()) {
        this.lyrics.close();
      }
    });

    // Initialize settings module
    this.settings.init({
      onVolumeChange: (volumeDecimal) => {
        this.audioCurrent.volume = volumeDecimal;
        this.audioNext.volume = volumeDecimal;
      }
    });

    // Listen for settings events
    events.on('settings:opened', () => this.lyrics.close(false));
    events.on('settings:changed', async ({ key }) => {
      if (key === 'instrumental') {
        // If currently playing, refetch playlist and continue from next track
        if (this.currentMood && document.body.dataset.state === 'playing') {
          await this.fetchPlaylist();
          this.currentIndex = -1;
        }
      } else if (key === 'showLyrics') {
        // Update lyrics button visibility if currently playing
        if (this.currentTrack) {
          this.lyrics.updateDisplay(this.currentTrack);
        }
      }
    });

    // Initialize lyrics module
    this.lyrics.init({
      onAnnounce: (msg) => this.announce(msg),
      getShowLyricsButton: () => this.settings.getShowLyricsButton(),
      closeSettings: () => this.settings.close()
    });

    // Initialize about panel
    this.about.init();

    // About panel mutual exclusivity
    events.on('about:opened', () => {
      this.settings.close();
      this.lyrics.close(false);
    });

    // Moodlet discovery events
    if (this.moodletToast) {
      this.moodletToast.addEventListener('click', (e) => {
        const option = e.target.closest('[data-direction]');
        if (option) {
          this.handleMoodletSelection(option.dataset.direction);
        }
        const dismiss = e.target.closest('.moodlet-toast__dismiss');
        if (dismiss) {
          this.hideMoodlet();
        }
      });
    }
  }

  // Close all overlay panels (for state transitions)
  closeAllPanels() {
    this.about.close();
    this.settings.close();
    this.lyrics.close(false); // Don't persist - state transition, not user action
  }

  async init() {
    try {
      await this.fetchMoods();
      this.moodGalaxy.classList.remove('mood-galaxy--loading');
      this.renderMoodSelector();

      // Handle ?mood= deep link
      const params = new URLSearchParams(window.location.search);
      const deepMood = params.get('mood');
      if (deepMood) {
        history.replaceState(null, '', window.location.pathname);
        if (this.moods.some(m => m.name === deepMood)) {
          await this.selectMood(deepMood);
          return;
        }
      }

      // Pre-fetch default mood playlist
      await this.fetchPlaylist();
    } catch (err) {
      console.error('Init error:', err);
      this.moodGalaxy.classList.remove('mood-galaxy--loading');
      // Still render static moods for demo
      this.renderFallbackMoods();
    } finally {
      document.body.classList.add('app--loaded');
    }
  }

  announce(message) {
    // Announce to screen readers via live region
    if (this.srAnnouncer) {
      this.srAnnouncer.textContent = message;
    }
  }

  async fetchMoods() {
    const response = await fetch('/api/moods');
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }
    this.moods = await response.json();

    // Merge with known moods so empty moods still show in galaxy
    const apiNames = new Set(this.moods.map(m => m.name));
    for (const known of KNOWN_MOODS) {
      if (!apiNames.has(known.name)) {
        this.moods.push({ ...known, track_count: 0, total_minutes: 0 });
      }
    }

    // Set default mood to first available or 'focus'
    if (this.moods.length > 0) {
      const focusMood = this.moods.find(m => m.name === 'focus');
      this.currentMood = focusMood ? 'focus' : this.moods[0].name;
    }
  }

  renderMoodSelector() {
    // Clear existing mood buttons (preserve resume pill and galaxy anchor)
    const children = Array.from(this.moodGalaxy.children);
    for (const child of children) {
      if (child.id !== 'resume-btn' && !child.classList.contains('galaxy-anchor')) {
        this.moodGalaxy.removeChild(child);
      }
    }

    // Check for last session to resume
    const lastMood = storage.getLastMood();
    const lastMoodData = lastMood ? this.moods.find(m => m.name === lastMood) : null;

    // Update resume pill
    if (this.resumeBtn && this.resumeMoodEl) {
      if (lastMoodData) {
        this.resumeMoodEl.textContent = lastMoodData.display_name;
        this.resumeBtn.hidden = false;
        this.resumeBtn.setAttribute('aria-label', `Resume ${lastMoodData.display_name} mood`);
        // Bind click handler (remove old first to prevent duplicates)
        this.resumeBtn.onclick = () => this.selectMood(lastMood);
      } else {
        this.resumeBtn.hidden = true;
      }
    }

    const orbData = [];
    const anchor = this.moodGalaxy.querySelector('.galaxy-anchor');
    for (const mood of this.moods) {
      const btn = document.createElement('button');
      btn.className = 'mood-space';
      btn.dataset.mood = mood.name;
      btn.title = `${mood.track_count} tracks, ${Math.round(mood.total_minutes)} min`;
      btn.setAttribute('aria-label', `Play ${mood.display_name} mood`);
      btn.addEventListener('click', () => this.selectMood(mood.name));

      // Inner wrapper: carries breathing animation (separates from JS positioning)
      const inner = document.createElement('span');
      inner.className = 'mood-space__inner';
      inner.textContent = mood.display_name;
      btn.appendChild(inner);

      // Insert before anchor so orbs are children 2-7 (after resume-btn,
      // before anchor), matching nth-child drift selectors in mood-space.css
      if (anchor) {
        this.moodGalaxy.insertBefore(btn, anchor);
      } else {
        this.moodGalaxy.appendChild(btn);
      }
      orbData.push({ el: btn, trackCount: mood.track_count || 1 });
    }

    // Initialize galaxy layout
    this.galaxy.init(this.moodGalaxy, orbData);
    if (!this.galaxy.reducedMotion) {
      this.moodGalaxy.classList.add('mood-galaxy--positioned');
      // Update bounds on resize (remove old handler to prevent accumulation)
      if (this._resizeHandler) {
        window.removeEventListener('resize', this._resizeHandler);
      }
      this._resizeHandler = () => {
        this.galaxy.updateBounds();
      };
      window.addEventListener('resize', this._resizeHandler);
    }
  }

  renderFallbackMoods() {
    this.moods = KNOWN_MOODS.map(m => ({ ...m, track_count: 0, total_minutes: 0 }));
    this.renderMoodSelector();
  }

  async selectMood(mood) {
    // Prevent rapid selection race conditions
    if (this.isSelecting) return;
    this.isSelecting = true;

    // Report skip for current track when switching moods while playing
    if (this.currentTrack && this.playStartTime >= 0 && document.body.dataset.state === 'playing') {
      const listenSeconds = this.audioCurrent.currentTime - this.playStartTime;
      this.notifyListen(this.currentTrack.id, 'skip', listenSeconds, true);
    }

    // Save as last mood for resume feature
    storage.setLastMood(mood);

    // Close any open panels
    this.closeAllPanels();

    // Reset session state for new mood
    this.sessionPlayCount = 0;
    this.skipCount = 0;
    if (this.skipResetTimeout) {
      clearTimeout(this.skipResetTimeout);
      this.skipResetTimeout = null;
    }
    this.hideMoodlet();

    const space = this.moodGalaxy.querySelector(`[data-mood="${mood}"]`);

    try {
      this.currentMood = mood;
      this.currentIndex = -1;
      this.playlist = [];

      // Start expansion animation
      this.moodGalaxy.classList.add('mood-galaxy--selecting');
      if (space) {
        space.classList.add('mood-space--expanding');
      }

      // Update mood indicator
      const moodData = this.moods.find(m => m.name === mood);
      this.moodIndicator.textContent = moodData ? moodData.display_name : mood;

      // Fetch playlist and animate concurrently
      const [playlistResult] = await Promise.allSettled([
        this.fetchPlaylist(),
        new Promise(resolve => setTimeout(resolve, 300)) // expansion animation
      ]);

      // Update body data attribute and transition to playing state
      document.body.dataset.mood = mood;
      this.enterPlayingState();

      // Handle fetch failure with user feedback
      if (playlistResult.status === 'rejected') {
        console.error('Playlist fetch error:', playlistResult.reason);
        this.trackName.textContent = 'Could not load tracks. Tap mood to retry.';
        this.announce('Could not load tracks. Please try again.');
        return;
      }

      await this.playNext();
    } finally {
      // Always clean up animation classes and reset flag
      this.moodGalaxy.classList.remove('mood-galaxy--selecting');
      if (space) {
        space.classList.remove('mood-space--expanding');
      }
      this.isSelecting = false;
    }
  }

  enterPlayingState() {
    document.body.dataset.state = 'playing';
    this.playerBar.hidden = false;
    // Announce state change for screen readers
    const moodData = this.moods.find(m => m.name === this.currentMood);
    const moodName = moodData ? moodData.display_name : this.currentMood;
    this.announce(`Entering ${moodName} mood`);
  }

  returnToIdle() {
    // Report skip for current track when returning to idle
    if (this.currentTrack && this.playStartTime >= 0) {
      const listenSeconds = this.audioCurrent.currentTime - this.playStartTime;
      this.notifyListen(this.currentTrack.id, 'skip', listenSeconds, true);
    }

    // Close all overlay panels
    this.closeAllPanels();

    // Stop metadata cycling
    this.stopMetadataCycle();

    // Clear any pending error retry timeout
    if (this.errorRetryTimeout) {
      clearTimeout(this.errorRetryTimeout);
      this.errorRetryTimeout = null;
    }

    // Clear skip friction timer
    if (this.skipResetTimeout) {
      clearTimeout(this.skipResetTimeout);
      this.skipResetTimeout = null;
    }

    // Pause both audio elements
    this.audioCurrent.pause();
    this.audioNext.pause();
    this.isPlaying = false;
    this.isSelecting = false;
    this.updatePlayButton();

    // Reset state
    document.body.dataset.state = 'idle';
    document.body.dataset.mood = '';
    this.playerBar.hidden = true;
    this.currentIndex = -1;
    this.trackName.textContent = 'Ready';
    this.progressFill.style.width = '0%';
    this.currentTimeEl.textContent = '0:00';
    this.progressRail.setAttribute('aria-valuenow', '0');

    // Reset moodlet state
    this.hideMoodlet();
    this.sessionPlayCount = 0;

    // Update resume pill to reflect the session we just left
    const lastMood = storage.getLastMood();
    const lastMoodData = lastMood ? this.moods.find(m => m.name === lastMood) : null;
    if (this.resumeBtn && this.resumeMoodEl && lastMoodData) {
      this.resumeMoodEl.textContent = lastMoodData.display_name;
      this.resumeBtn.hidden = false;
      this.resumeBtn.setAttribute('aria-label', `Resume ${lastMoodData.display_name} mood`);
      this.resumeBtn.onclick = () => this.selectMood(lastMood);
    }

    // Resume galaxy layout
    this.galaxy.updateBounds();

    // Announce state change for screen readers
    this.announce('Playback stopped. Select a mood to continue.');
  }

  async fetchPlaylist() {
    const params = this.settings.getInstrumentalOnly() ? '?instrumental=true' : '';
    const url = `/api/moods/${this.currentMood}/playlist${params}`;

    const response = await fetch(url);
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }
    this.playlist = await response.json();
  }

  togglePlay() {
    // Guard: don't play without a mood selected
    if (document.body.dataset.state !== 'playing') {
      return;
    }

    if (this.isPlaying) {
      this.pause();
    } else {
      this.play();
    }
  }

  async play() {
    if (this.playlist.length === 0) {
      return;
    }

    if (this.currentIndex < 0) {
      await this.playNext();
      return;
    }

    try {
      await this.audioCurrent.play();
      this.isPlaying = true;
      this.updatePlayButton();
      this.updateMediaSessionState();
    } catch (err) {
      console.error('Play failed:', err);
    }
  }

  pause() {
    this.audioCurrent.pause();
    this.isPlaying = false;
    this.updatePlayButton();
    this.updateMediaSessionState();
  }

  async playNext() {
    if (this.playlist.length === 0) {
      this.trackName.textContent = 'New sounds coming soon';
      this.announce('No tracks available for this mood');
      return;
    }

    this.currentIndex++;
    if (this.currentIndex >= this.playlist.length) {
      this.shufflePlaylist();
      this.currentIndex = 0;
    }

    const track = this.playlist[this.currentIndex];
    this.currentTrack = track;

    // Immediately fade out old lyrics to prevent stale content
    this.lyrics.clearForTrackChange();

    // Remove events from current before swap
    this.audioCurrent.removeEventListener('timeupdate', this._onTimeUpdate);
    this.audioCurrent.removeEventListener('ended', this._onEnded);
    this.audioCurrent.removeEventListener('error', this._onError);

    // Swap audio elements for gapless playback
    const temp = this.audioCurrent;
    this.audioCurrent = this.audioNext;
    this.audioNext = temp;

    // Bind events to new current element
    this.audioCurrent.addEventListener('timeupdate', this._onTimeUpdate);
    this.audioCurrent.addEventListener('ended', this._onEnded);
    this.audioCurrent.addEventListener('error', this._onError);

    this.audioCurrent.src = track.audio_url || `/audio/${track.file_path}`;
    const trackDisplayName = getTrackDisplayName(track);
    this.trackName.textContent = trackDisplayName;
    this.startMetadataCycle(track);

    try {
      await this.audioCurrent.play();
      this.isPlaying = true;
      this.errorRetryCount = 0; // Reset on successful play
      this.updatePlayButton();
      this.updateMediaSession(track);
      this.lyrics.updateDisplay(track);
      this.playStartTime = this.audioCurrent.currentTime;
      this.notifyListen(track.id, 'play');
      this.preloadNext();
      // Announce track change for screen readers
      this.announce(`Now playing: ${trackDisplayName}`);

      // Restore lyrics panel if user had it open (only on first track of session)
      if (this.sessionPlayCount === 0 && this.lyrics.isInViewingMode()) {
        this.lyrics.open();
      }

      // Track session play count and trigger moodlet discovery
      this.sessionPlayCount++;
      if (this.sessionPlayCount === 3 && !this.moodletShownForMood[this.currentMood]) {
        this.showMoodlet();
      }

      // Start skip friction reset timer (resets after 90s of uninterrupted play)
      this.startSkipResetTimer();
    } catch (err) {
      console.error('Play next failed:', err);
      // Handle autoplay block (common on mobile)
      if (err.name === 'NotAllowedError') {
        this.trackName.textContent = 'Tap to drift';
        this.isPlaying = false;
        this.updatePlayButton();
      }
    }
  }

  preloadNext() {
    // Guard against empty playlist (% 0 returns NaN)
    if (this.playlist.length === 0) return;

    const nextIndex = (this.currentIndex + 1) % this.playlist.length;
    const nextTrack = this.playlist[nextIndex];
    if (nextTrack) {
      this.audioNext.src = nextTrack.audio_url || `/audio/${nextTrack.file_path}`;
      this.audioNext.load();
    }
  }

  shufflePlaylist() {
    for (let i = this.playlist.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      [this.playlist[i], this.playlist[j]] = [this.playlist[j], this.playlist[i]];
    }
  }

  updateProgress() {
    const current = this.audioCurrent.currentTime;
    const dur = this.audioCurrent.duration;
    if (dur > 0) {
      const percent = (current / dur) * 100;
      this.progressFill.style.width = `${percent}%`;
      this.progressRail.setAttribute('aria-valuenow', Math.round(percent));
      this.currentTimeEl.textContent = formatTime(current);
    }
  }

  seek(e) {
    const rect = this.progressRail.getBoundingClientRect();
    const percent = (e.clientX - rect.left) / rect.width;
    const dur = this.audioCurrent.duration;
    if (dur > 0) {
      this.audioCurrent.currentTime = percent * dur;
    }
  }

  handleProgressKeydown(e) {
    const dur = this.audioCurrent.duration;
    if (!dur || dur === 0) return;

    const step = 5; // seconds
    const largeStep = 30; // seconds

    switch (e.code) {
      case 'ArrowRight':
        e.preventDefault();
        this.audioCurrent.currentTime = Math.min(dur, this.audioCurrent.currentTime + step);
        break;
      case 'ArrowLeft':
        e.preventDefault();
        this.audioCurrent.currentTime = Math.max(0, this.audioCurrent.currentTime - step);
        break;
      case 'ArrowUp':
        e.preventDefault();
        this.audioCurrent.currentTime = Math.min(dur, this.audioCurrent.currentTime + largeStep);
        break;
      case 'ArrowDown':
        e.preventDefault();
        this.audioCurrent.currentTime = Math.max(0, this.audioCurrent.currentTime - largeStep);
        break;
      case 'Home':
        e.preventDefault();
        this.audioCurrent.currentTime = 0;
        break;
      case 'End':
        e.preventDefault();
        this.audioCurrent.currentTime = dur;
        break;
    }
  }

  updatePlayButton() {
    if (this.isPlaying) {
      this.playIcon.classList.add('hidden');
      this.pauseIcon.classList.remove('hidden');
      this.playBtn.setAttribute('aria-label', 'Pause');
    } else {
      this.playIcon.classList.remove('hidden');
      this.pauseIcon.classList.add('hidden');
      this.playBtn.setAttribute('aria-label', 'Play');
    }
  }

  // Track metadata cycling (title → mood → energy → intensity)
  startMetadataCycle(track) {
    this.stopMetadataCycle();
    this.metadataCycleIndex = 0;

    // Cycle every 4 seconds
    this.metadataCycleInterval = setInterval(() => {
      this.metadataCycleIndex = (this.metadataCycleIndex + 1) % 4;
      this.trackName.textContent = this.getMetadataForCycle(track, this.metadataCycleIndex);
    }, 4000);
  }

  stopMetadataCycle() {
    if (this.metadataCycleInterval) {
      clearInterval(this.metadataCycleInterval);
      this.metadataCycleInterval = null;
    }
  }

  getMetadataForCycle(track, index) {
    const moodData = this.moods.find(m => m.name === this.currentMood);
    const moodName = moodData ? moodData.display_name : this.currentMood;

    switch (index) {
      case 0:
        return getTrackDisplayName(track);
      case 1:
        return moodName;
      case 2:
        return formatEnergy(track.energy);
      case 3:
        return formatIntensity(track.intensity);
      default:
        return getTrackDisplayName(track);
    }
  }

  notifyListen(trackId, eventType, listenSeconds = 0, useBeacon = false) {
    // Double-count guard: skip if already reported for this play
    if (this.playStartTime < 0 && eventType !== 'play') return;

    const data = {
      event: eventType,
      listen_seconds: Math.max(0, Math.round(listenSeconds)),
      mood: this.currentMood,
      position: this.currentIndex,
    };
    reportListen(trackId, data, { beacon: useBeacon });

    // Mark as reported to prevent double-counting
    if (eventType !== 'play') {
      this.playStartTime = -1;
    }
  }

  onTrackEnded() {
    if (this.currentTrack && this.playStartTime >= 0) {
      const listenSeconds = this.audioCurrent.currentTime - this.playStartTime;
      this.notifyListen(this.currentTrack.id, 'complete', listenSeconds);
    }
    this.playNext();
  }

  // Moodlet Discovery Methods

  showMoodlet() {
    if (!this.moodletToast) return;

    // Mark as shown for this mood
    this.moodletShownForMood[this.currentMood] = true;

    // Update label to match current mood
    const moodData = this.moods.find(m => m.name === this.currentMood);
    const moodName = moodData ? moodData.display_name : this.currentMood;
    if (this.moodletDeeperLabel) {
      this.moodletDeeperLabel.textContent = `Deeper ${moodName}`;
    }

    // Show toast
    this.moodletToast.hidden = false;
    // Trigger reflow for animation
    void this.moodletToast.offsetHeight;
    this.moodletToast.dataset.visible = 'true';

    // Auto-dismiss after 10 seconds
    this.moodletTimeout = setTimeout(() => {
      this.hideMoodlet();
    }, 10000);

    this.announce('Moodlet discovery: Want to go deeper or lighter?');
  }

  hideMoodlet() {
    if (!this.moodletToast) return;

    // Clear auto-dismiss timeout
    if (this.moodletTimeout) {
      clearTimeout(this.moodletTimeout);
      this.moodletTimeout = null;
    }

    // Hide with animation
    this.moodletToast.dataset.visible = 'false';

    // Hide completely after animation
    setTimeout(() => {
      this.moodletToast.hidden = true;
    }, 300);
  }

  handleMoodletSelection(direction) {
    this.hideMoodlet();
    // Refetch playlist with fresh shuffle
    this.fetchPlaylist().then(() => {
      const message = direction === 'deeper'
        ? 'Going deeper. Playlist updated.'
        : 'Lighter flow. Playlist updated.';
      this.announce(message);
    }).catch(err => {
      console.error('Failed to update playlist:', err);
    });
  }

  handleError(e) {
    this.errorRetryCount++;
    console.error(`Audio error (attempt ${this.errorRetryCount}):`, e);
    if (this.errorRetryCount >= 3) {
      this.trackName.textContent = 'Unable to play. Try another mood.';
      this.announce('Playback unavailable. Select a different mood.');
      this.errorRetryCount = 0;
      return;
    }

    this.trackName.textContent = 'Lost the signal...';
    this.announce('Audio error. Reconnecting...');

    // Store timeout ID so it can be cleared if user returns to idle
    this.errorRetryTimeout = setTimeout(() => {
      this.errorRetryTimeout = null;
      this.trackName.textContent = 'Reconnecting...';
      this.playNext();
    }, 1000);
  }

  // Skip to next track (with progressive soft friction)
  skipTrack() {
    if (this.playlist.length === 0) return;

    // Report skip event for current track
    if (this.currentTrack && this.playStartTime >= 0) {
      const listenSeconds = this.audioCurrent.currentTime - this.playStartTime;
      this.notifyListen(this.currentTrack.id, 'skip', listenSeconds);
    }

    // Increment skip count and clear any pending reset
    this.skipCount++;
    if (this.skipResetTimeout) {
      clearTimeout(this.skipResetTimeout);
      this.skipResetTimeout = null;
    }

    // Progressive friction based on skip count
    if (this.skipCount === 3) {
      // Visual nudge on 3rd skip
      this.showSkipNudge();
    } else if (this.skipCount >= 4) {
      // Philosophy toast on 4th+ skip
      this.showSkipPhilosophyToast();
    }

    this.playNext();
  }

  // Visual nudge when approaching skip limit
  showSkipNudge() {
    if (!this.skipBtn) return;
    this.skipBtn.classList.add('player-bar__skip--nudge');
    setTimeout(() => {
      this.skipBtn.classList.remove('player-bar__skip--nudge');
    }, 600);
  }

  // Philosophy reminder toast
  showSkipPhilosophyToast() {
    // Create toast if it doesn't exist
    let toast = document.getElementById('skip-philosophy-toast');
    if (!toast) {
      toast = document.createElement('div');
      toast.id = 'skip-philosophy-toast';
      toast.className = 'skip-toast';
      toast.setAttribute('role', 'status');
      toast.setAttribute('aria-live', 'polite');
      const span = document.createElement('span');
      span.textContent = 'Drifting works best when you let go';
      toast.appendChild(span);
      document.body.appendChild(toast);
    }

    // Show toast
    toast.hidden = false;
    toast.dataset.visible = 'true';

    // Auto-hide after 3s
    setTimeout(() => {
      toast.dataset.visible = 'false';
      setTimeout(() => {
        toast.hidden = true;
      }, 300);
    }, 3000);
  }

  // Reset skip counter after engagement (called after 90s of uninterrupted play)
  startSkipResetTimer() {
    if (this.skipResetTimeout) {
      clearTimeout(this.skipResetTimeout);
    }
    this.skipResetTimeout = setTimeout(() => {
      this.skipCount = 0;
      this.skipResetTimeout = null;
    }, 90000); // 90 seconds
  }

  // Toggle mute
  toggleMute() {
    this.audioCurrent.muted = !this.audioCurrent.muted;
    this.audioNext.muted = this.audioCurrent.muted;
    this.announce(this.audioCurrent.muted ? 'Muted' : 'Unmuted');
  }

  // Media Session API - update system media controls
  updateMediaSession(track) {
    if (!('mediaSession' in navigator)) return;

    const moodData = this.moods.find(m => m.name === this.currentMood);
    const moodName = moodData ? moodData.display_name : this.currentMood;

    // Extract title from track (use cleaned display name)
    const title = getTrackDisplayName(track);
    const artist = track.artist || 'Drift FM';

    navigator.mediaSession.metadata = new MediaMetadata({
      title: title,
      artist: artist,
      album: `${moodName} Radio`,
      artwork: [
        { src: '/icons/icon-192.png', sizes: '192x192', type: 'image/png' },
        { src: '/icons/icon-512.png', sizes: '512x512', type: 'image/png' }
      ]
    });

    // Set up action handlers
    navigator.mediaSession.setActionHandler('play', () => this.play());
    navigator.mediaSession.setActionHandler('pause', () => this.pause());
    navigator.mediaSession.setActionHandler('nexttrack', () => this.skipTrack());
    navigator.mediaSession.setActionHandler('previoustrack', null); // No previous track support

    // Update playback state
    navigator.mediaSession.playbackState = this.isPlaying ? 'playing' : 'paused';
  }

  // Update media session playback state
  updateMediaSessionState() {
    if ('mediaSession' in navigator) {
      navigator.mediaSession.playbackState = this.isPlaying ? 'playing' : 'paused';
    }
  }
}

// Initialize
document.addEventListener('DOMContentLoaded', () => {
  const player = new DriftFMPlayer();
  player.init();
  window.driftfm = player;
});
