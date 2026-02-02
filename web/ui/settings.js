/**
 * Drift FM - Settings Module
 * Handles settings drawer, theme, volume, and preferences
 */

import { storage } from '../core/storage.js';
import { events } from '../core/events.js';
import { applyTheme, resolveTheme } from '../core/theme.js';
import { trapFocus } from '../core/a11y.js';

/**
 * Settings drawer manager
 * Emits: 'settings:opened', 'settings:closed', 'settings:changed'
 */
export class SettingsManager {
  constructor() {
    // DOM elements
    this.drawer = document.getElementById('settings-drawer');
    this.settingsBtn = document.getElementById('settings-btn');
    this.closeBtn = document.getElementById('settings-close');
    this.themeToggle = document.getElementById('theme-toggle');
    this.instrumentalToggle = document.getElementById('instrumental-toggle');
    this.showLyricsToggle = document.getElementById('show-lyrics-toggle');
    this.volumeSlider = document.getElementById('volume-slider');

    // State
    this.themePref = storage.getTheme();
    this.instrumentalOnly = storage.getInstrumental();
    this.showLyricsButton = storage.getShowLyrics();
    this.volume = storage.getVolume();

    // Focus management
    this._settingsOpener = null;
    this._trapFocus = null;

    // Bind focus trap for event listener removal
    this.handleFocusTrap = (e) => trapFocus(this.drawer, e);
  }

  /**
   * Initialize event listeners
   * @param {Object} options - Configuration options
   * @param {Function} options.onVolumeChange - Callback for volume changes (audio elements)
   */
  init(options = {}) {
    this.onVolumeChange = options.onVolumeChange || (() => {});

    // Settings button toggle
    if (this.settingsBtn) {
      this.settingsBtn.addEventListener('click', () => this.toggle());
    }

    // Close button
    if (this.closeBtn) {
      this.closeBtn.addEventListener('click', () => this.close());
    }

    // Theme toggle
    if (this.themeToggle) {
      this.themeToggle.addEventListener('change', (e) => {
        if (e.target.type === 'radio' && e.target.name === 'theme') {
          this.setTheme(e.target.value);
        }
      });
    }

    // System theme changes (when using auto)
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
      if (this.themePref === 'auto') {
        applyTheme(resolveTheme('auto'));
      }
    });

    // Instrumental toggle
    if (this.instrumentalToggle) {
      this.instrumentalToggle.addEventListener('change', () => {
        this.instrumentalOnly = this.instrumentalToggle.checked;
        storage.setInstrumental(this.instrumentalOnly);
        events.emit('settings:changed', { key: 'instrumental', value: this.instrumentalOnly });
      });
    }

    // Show lyrics toggle
    if (this.showLyricsToggle) {
      this.showLyricsToggle.addEventListener('change', () => {
        this.showLyricsButton = this.showLyricsToggle.checked;
        storage.setShowLyrics(this.showLyricsButton);
        events.emit('settings:changed', { key: 'showLyrics', value: this.showLyricsButton });
      });
    }

    // Volume slider
    if (this.volumeSlider) {
      this.volumeSlider.addEventListener('input', () => {
        this.setVolume(parseInt(this.volumeSlider.value, 10));
      });
    }

    // Initialize UI state
    this.initializeUI();
  }

  /**
   * Initialize UI elements with stored values
   */
  initializeUI() {
    // Apply stored theme
    applyTheme(resolveTheme(this.themePref));
    this.updateThemeToggle();

    // Set toggle states
    if (this.instrumentalToggle) {
      this.instrumentalToggle.checked = this.instrumentalOnly;
    }
    if (this.showLyricsToggle) {
      this.showLyricsToggle.checked = this.showLyricsButton;
    }

    // Set volume
    this.setVolume(this.volume);
    if (this.volumeSlider) {
      this.volumeSlider.value = this.volume;
    }
  }

  // Theme Management

  setTheme(pref) {
    this.themePref = pref;
    storage.setTheme(pref);
    applyTheme(resolveTheme(pref));
    this.updateThemeToggle();
    events.emit('settings:changed', { key: 'theme', value: pref });
  }

  updateThemeToggle() {
    if (!this.themeToggle) return;
    const radios = this.themeToggle.querySelectorAll('input[type="radio"]');
    radios.forEach(radio => {
      radio.checked = radio.value === this.themePref;
    });
  }

  // Volume Management

  setVolume(value) {
    this.volume = Math.max(0, Math.min(100, value));
    storage.setVolume(this.volume);
    // Notify audio elements via callback
    try {
      this.onVolumeChange(this.volume / 100);
    } catch (err) {
      console.error('[Settings] Volume callback error:', err);
    }
    events.emit('settings:changed', { key: 'volume', value: this.volume });
  }

  // Drawer Management

  toggle() {
    if (this.drawer.hidden) {
      this.open();
    } else {
      this.close();
    }
  }

  open() {
    // Emit event so other panels can close
    events.emit('settings:opened', {});

    this._settingsOpener = document.activeElement;
    this.drawer.hidden = false;

    // Set up focus trap
    this._trapFocus = this.handleFocusTrap;
    this.drawer.addEventListener('keydown', this._trapFocus);

    // Focus first focusable element
    const firstFocusable = this.drawer.querySelector('button, input, [tabindex="0"]');
    if (firstFocusable) {
      firstFocusable.focus();
    }
  }

  close() {
    // Remove focus trap
    if (this._trapFocus) {
      this.drawer.removeEventListener('keydown', this._trapFocus);
      this._trapFocus = null;
    }

    this.drawer.hidden = true;

    // Restore focus to opener
    if (this._settingsOpener) {
      this._settingsOpener.focus();
      this._settingsOpener = null;
    }

    events.emit('settings:closed', {});
  }

  // State Accessors

  isOpen() {
    return !this.drawer.hidden;
  }

  getInstrumentalOnly() {
    return this.instrumentalOnly;
  }

  getShowLyricsButton() {
    return this.showLyricsButton;
  }

  getVolume() {
    return this.volume;
  }
}
