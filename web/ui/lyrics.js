/**
 * Drift FM - Lyrics Panel Module
 * Handles lyrics display, panel state, and surrender-philosophy messages
 */

import { storage } from '../core/storage.js';
import { events } from '../core/events.js';

// Surrender-philosophy messages for instrumental/no-lyrics tracks
const SURRENDER_MESSAGES = [
  'Let the music speak.',
  'Words would only get in the way.',
  'Some things are best left unspoken.',
  'Close your eyes. Just drift.',
  'The melody is the message.',
  'No lyrics needed.',
  'Pure sound. Pure feeling.',
  'Let it wash over you.',
  'No lyrics. Just vibes.'
];

/**
 * Lyrics panel manager
 * Emits: 'lyrics:opened', 'lyrics:closed'
 */
export class LyricsManager {
  constructor() {
    // DOM elements
    this.panel = document.getElementById('lyrics-panel');
    this.content = document.getElementById('lyrics-content');
    this.btn = document.getElementById('lyrics-btn');
    this.closeBtn = document.getElementById('lyrics-close');
    this.header = this.panel?.querySelector('.lyrics-panel__header');

    // State
    this.isExpanded = storage.getLyricsExpanded();
    this.viewingMode = storage.getLyricsOpen();
    this._updateSeq = 0; // Sequence counter for stale update protection
    this._lastSurrenderIndex = -1;

    // Callbacks (set via init)
    this.onAnnounce = () => {};
    this.getShowLyricsButton = () => true;
    this.closeSettings = () => {};
  }

  /**
   * Initialize event listeners
   * @param {Object} options - Configuration options
   * @param {Function} options.onAnnounce - Callback for screen reader announcements
   * @param {Function} options.getShowLyricsButton - Returns whether lyrics button should show
   * @param {Function} options.closeSettings - Closes settings panel
   */
  init(options = {}) {
    this.onAnnounce = options.onAnnounce || this.onAnnounce;
    this.getShowLyricsButton = options.getShowLyricsButton || this.getShowLyricsButton;
    this.closeSettings = options.closeSettings || this.closeSettings;

    // Lyrics button click
    if (this.btn) {
      this.btn.addEventListener('click', () => this.toggle());
    }

    // Close button
    if (this.closeBtn) {
      this.closeBtn.addEventListener('click', () => this.close());
    }

    // Double-tap header to toggle expanded state
    if (this.header) {
      this.header.addEventListener('dblclick', () => this.toggleExpanded());
    }
  }

  // Panel Management

  toggle() {
    if (!this.panel) return;

    if (this.panel.hidden) {
      this.open();
    } else {
      this.close();
    }
  }

  open() {
    if (!this.panel) return;
    // Only open when playing
    if (document.body.dataset.state !== 'playing') return;

    // Close settings if open (panels are mutually exclusive)
    this.closeSettings();

    this.panel.hidden = false;
    this.viewingMode = true;
    storage.setLyricsOpen(true);

    if (this.btn) {
      this.btn.dataset.active = 'true';
    }

    // Restore expanded state from preference
    if (this.isExpanded) {
      this.panel.dataset.expanded = 'true';
    }

    events.emit('lyrics:opened', {});
  }

  close(persist = true) {
    if (!this.panel) return;

    this.panel.hidden = true;
    delete this.panel.dataset.expanded; // Clean up expanded state on close
    if (persist) {
      this.viewingMode = false;
      storage.setLyricsOpen(false);
    }

    if (this.btn) {
      this.btn.dataset.active = 'false';
    }

    events.emit('lyrics:closed', {});
  }

  toggleExpanded() {
    if (!this.panel) return;

    this.isExpanded = !this.isExpanded;
    if (this.isExpanded) {
      this.panel.dataset.expanded = 'true';
    } else {
      delete this.panel.dataset.expanded;
    }
    storage.setLyricsExpanded(this.isExpanded);
    this.onAnnounce(this.isExpanded ? 'Lyrics panel expanded' : 'Lyrics panel collapsed');
  }

  // Content Management

  /**
   * Update lyrics display for a track
   * @param {Object|null} track - Track object with lyrics property
   */
  updateDisplay(track) {
    const hasLyrics = track && track.lyrics;
    const shouldShowButton = hasLyrics && this.getShowLyricsButton();

    // Show/hide lyrics button based on whether track has lyrics AND user preference
    if (this.btn) {
      this.btn.hidden = !shouldShowButton;
    }

    // Update lyrics content with fade transition
    if (this.content) {
      // Use sequence counter for robust stale-update protection
      this._updateSeq++;
      const seq = this._updateSeq;

      // Fade out current content
      this.content.classList.add('lyrics-panel__content--loading');

      // After fade out, update content and fade in
      setTimeout(() => {
        // Guard against stale update (user skipped to another track)
        if (this._updateSeq !== seq) return;

        if (hasLyrics) {
          this.content.textContent = track.lyrics;
          this.content.classList.remove('lyrics-panel__content--empty');
        } else {
          // Show rotating surrender-philosophy message
          this.content.textContent = this._getSurrenderMessage();
          this.content.classList.add('lyrics-panel__content--empty');
        }

        // Scroll to top for new lyrics
        this.content.scrollTop = 0;

        // Fade in new content
        this.content.classList.remove('lyrics-panel__content--loading');
      }, 200); // Match CSS transition duration
    }
  }

  /**
   * Clear lyrics immediately when starting to load a new track
   * Called before updateDisplay to prevent showing stale lyrics
   */
  clearForTrackChange() {
    if (this.content) {
      this.content.classList.add('lyrics-panel__content--loading');
    }
  }

  _getSurrenderMessage() {
    // Pick a random message, avoiding immediate repeat
    let index;
    do {
      index = Math.floor(Math.random() * SURRENDER_MESSAGES.length);
    } while (index === this._lastSurrenderIndex && SURRENDER_MESSAGES.length > 1);
    this._lastSurrenderIndex = index;
    return SURRENDER_MESSAGES[index];
  }

  // State Accessors

  isOpen() {
    return this.panel && !this.panel.hidden;
  }

  isInViewingMode() {
    return this.viewingMode;
  }
}
