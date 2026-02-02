/**
 * Drift FM - About Panel Module
 * Handles about panel display and interactions
 */

import { events } from '../core/events.js';
import { storage } from '../core/storage.js';
import { applyTheme, resolveTheme } from '../core/theme.js';
import { trapFocus } from '../core/a11y.js';

const THEME_CYCLE = ['auto', 'light', 'dark'];

// SVG path data for theme icons (static, no user input)
const THEME_ICONS = {
  auto: '<circle cx="12" cy="12" r="5" stroke="currentColor" stroke-width="2" fill="none"/><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>',
  light: '<circle cx="12" cy="12" r="5" fill="currentColor"/><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>',
  dark: '<path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round" stroke-linejoin="round"/>'
};

/**
 * About panel manager
 * Emits: 'about:opened', 'about:closed'
 */
export class AboutManager {
  constructor() {
    // DOM elements
    this.panel = document.getElementById('about-panel');
    this.btn = document.getElementById('about-btn');
    this.btnPlaying = document.getElementById('about-btn-playing');
    this.closeBtn = document.getElementById('about-close');
    this.themeBtn = document.getElementById('about-theme-btn');
    this.themeIcon = document.getElementById('about-theme-icon');

    // Theme state (validate stored value)
    const stored = storage.getTheme();
    this.themePref = THEME_CYCLE.includes(stored) ? stored : 'auto';

    // Focus management
    this._opener = null;
    this._trapFocus = null;

    // Bind focus trap for event listener removal
    this.handleFocusTrap = (e) => trapFocus(this.panel, e);
  }

  /**
   * Initialize event listeners
   */
  init() {
    const handler = (e) => {
      e.preventDefault();
      this.toggle();
    };

    if (this.btn) this.btn.addEventListener('click', handler);
    if (this.btnPlaying) this.btnPlaying.addEventListener('click', handler);

    if (this.closeBtn) {
      this.closeBtn.addEventListener('click', () => this.close());
    }

    if (this.themeBtn) {
      this.themeBtn.addEventListener('click', () => this.cycleTheme());
    }

    // Stay in sync when theme changes from settings panel
    events.on('settings:changed', ({ key, value }) => {
      if (key === 'theme' && value !== this.themePref) {
        this.themePref = value;
        this.updateThemeIcon();
      }
    });

    this.updateThemeIcon();
  }

  // Theme cycling

  cycleTheme() {
    const idx = THEME_CYCLE.indexOf(this.themePref);
    this.themePref = THEME_CYCLE[(idx + 1) % THEME_CYCLE.length];

    storage.setTheme(this.themePref);
    applyTheme(resolveTheme(this.themePref));
    this.updateThemeIcon();
    events.emit('settings:changed', { key: 'theme', value: this.themePref });
  }

  updateThemeIcon() {
    if (!this.themeIcon) return;
    // Safe: THEME_ICONS values are hardcoded SVG, not user input
    this.themeIcon.innerHTML = THEME_ICONS[this.themePref] || THEME_ICONS.auto;
    this.themeBtn?.setAttribute('aria-label', `Theme: ${this.themePref}`);
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

    // Emit event so other panels can close
    events.emit('about:opened', {});

    this._opener = document.activeElement;
    this.panel.hidden = false;

    // Set up focus trap
    this._trapFocus = this.handleFocusTrap;
    this.panel.addEventListener('keydown', this._trapFocus);

    // Focus first focusable element
    const firstFocusable = this.panel.querySelector('button, a, input, [tabindex="0"]');
    if (firstFocusable) {
      firstFocusable.focus();
    }
  }

  close() {
    if (!this.panel) return;

    // Remove focus trap
    if (this._trapFocus) {
      this.panel.removeEventListener('keydown', this._trapFocus);
      this._trapFocus = null;
    }

    this.panel.hidden = true;

    // Restore focus to opener
    if (this._opener) {
      this._opener.focus();
      this._opener = null;
    }

    events.emit('about:closed', {});
  }

  // State Accessors

  isOpen() {
    return this.panel && !this.panel.hidden;
  }
}
