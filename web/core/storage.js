/**
 * Drift FM - Storage Module
 * Centralized localStorage access with typed getters/setters
 */

const KEYS = {
  theme: 'drift-theme',
  instrumental: 'drift-instrumental',
  showLyrics: 'drift-show-lyrics',
  lyricsOpen: 'drift-lyrics-open',
  lyricsExpanded: 'drift-lyrics-expanded',
  volume: 'drift-volume',
  lastMood: 'drift-last-mood'
};

export const storage = {
  // Theme: 'dark', 'light', or 'auto' (default)
  getTheme() {
    return localStorage.getItem(KEYS.theme) || 'auto';
  },
  setTheme(value) {
    if (value === 'auto') {
      localStorage.removeItem(KEYS.theme);
    } else {
      localStorage.setItem(KEYS.theme, value);
    }
  },

  // Instrumental filter: boolean
  getInstrumental() {
    return localStorage.getItem(KEYS.instrumental) === 'true';
  },
  setInstrumental(value) {
    localStorage.setItem(KEYS.instrumental, String(value));
  },

  // Show lyrics button: boolean (default: true)
  getShowLyrics() {
    return localStorage.getItem(KEYS.showLyrics) !== 'false';
  },
  setShowLyrics(value) {
    localStorage.setItem(KEYS.showLyrics, String(value));
  },

  // Lyrics panel open state: boolean
  getLyricsOpen() {
    return localStorage.getItem(KEYS.lyricsOpen) === 'true';
  },
  setLyricsOpen(value) {
    localStorage.setItem(KEYS.lyricsOpen, String(value));
  },

  // Lyrics panel expanded state: boolean
  getLyricsExpanded() {
    return localStorage.getItem(KEYS.lyricsExpanded) === 'true';
  },
  setLyricsExpanded(value) {
    localStorage.setItem(KEYS.lyricsExpanded, String(value));
  },

  // Volume: 0-100
  getVolume() {
    const val = parseInt(localStorage.getItem(KEYS.volume) || '100', 10);
    return isNaN(val) ? 100 : Math.max(0, Math.min(100, val));
  },
  setVolume(value) {
    localStorage.setItem(KEYS.volume, String(value));
  },

  // Last mood: string (mood name)
  getLastMood() {
    return localStorage.getItem(KEYS.lastMood);
  },
  setLastMood(value) {
    localStorage.setItem(KEYS.lastMood, value);
  }
};
