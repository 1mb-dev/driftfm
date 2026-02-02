/**
 * Tests for storage.js
 * Note: Uses real localStorage with beforeEach cleanup for isolation
 */

import { describe, it, expect, beforeEach } from './runner.js';
import { storage } from '../core/storage.js';

// Test keys to avoid collision with real app state
const TEST_KEYS = [
  'drift-theme',
  'drift-instrumental',
  'drift-show-lyrics',
  'drift-lyrics-open',
  'drift-lyrics-expanded',
  'drift-volume',
  'drift-last-mood'
];

function clearTestStorage() {
  TEST_KEYS.forEach(key => localStorage.removeItem(key));
}

export function runStorageTests() {
  describe('storage.theme', () => {
    beforeEach(clearTestStorage);

    it('defaults to auto when unset', () => {
      expect(storage.getTheme()).toBe('auto');
    });

    it('stores and retrieves dark theme', () => {
      storage.setTheme('dark');
      expect(storage.getTheme()).toBe('dark');
    });

    it('stores and retrieves light theme', () => {
      storage.setTheme('light');
      expect(storage.getTheme()).toBe('light');
    });

    it('removes key when set to auto', () => {
      storage.setTheme('dark');
      storage.setTheme('auto');
      expect(storage.getTheme()).toBe('auto');
      expect(localStorage.getItem('drift-theme')).toBeNull();
    });
  });

  describe('storage.instrumental', () => {
    beforeEach(clearTestStorage);

    it('defaults to false when unset', () => {
      expect(storage.getInstrumental()).toBe(false);
    });

    it('stores true', () => {
      storage.setInstrumental(true);
      expect(storage.getInstrumental()).toBe(true);
    });

    it('stores false', () => {
      storage.setInstrumental(true);
      storage.setInstrumental(false);
      expect(storage.getInstrumental()).toBe(false);
    });
  });

  describe('storage.showLyrics', () => {
    beforeEach(clearTestStorage);

    it('defaults to true when unset', () => {
      expect(storage.getShowLyrics()).toBe(true);
    });

    it('stores false', () => {
      storage.setShowLyrics(false);
      expect(storage.getShowLyrics()).toBe(false);
    });

    it('stores true', () => {
      storage.setShowLyrics(false);
      storage.setShowLyrics(true);
      expect(storage.getShowLyrics()).toBe(true);
    });
  });

  describe('storage.volume', () => {
    beforeEach(clearTestStorage);

    it('defaults to 100 when unset', () => {
      expect(storage.getVolume()).toBe(100);
    });

    it('stores and retrieves volume', () => {
      storage.setVolume(75);
      expect(storage.getVolume()).toBe(75);
    });

    it('clamps volume to 0-100 range on get', () => {
      localStorage.setItem('drift-volume', '-10');
      expect(storage.getVolume()).toBe(0);

      localStorage.setItem('drift-volume', '150');
      expect(storage.getVolume()).toBe(100);
    });

    it('returns default for invalid value', () => {
      localStorage.setItem('drift-volume', 'abc');
      expect(storage.getVolume()).toBe(100);
    });
  });

  describe('storage.lastMood', () => {
    beforeEach(clearTestStorage);

    it('returns null when unset', () => {
      expect(storage.getLastMood()).toBeNull();
    });

    it('stores and retrieves mood', () => {
      storage.setLastMood('focus');
      expect(storage.getLastMood()).toBe('focus');
    });
  });

  describe('storage.lyricsOpen', () => {
    beforeEach(clearTestStorage);

    it('defaults to false when unset', () => {
      expect(storage.getLyricsOpen()).toBe(false);
    });

    it('stores true', () => {
      storage.setLyricsOpen(true);
      expect(storage.getLyricsOpen()).toBe(true);
    });
  });

  describe('storage.lyricsExpanded', () => {
    beforeEach(clearTestStorage);

    it('defaults to false when unset', () => {
      expect(storage.getLyricsExpanded()).toBe(false);
    });

    it('stores true', () => {
      storage.setLyricsExpanded(true);
      expect(storage.getLyricsExpanded()).toBe(true);
    });
  });
}
