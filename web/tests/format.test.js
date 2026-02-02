/**
 * Tests for format.js
 */

import { describe, it, expect } from './runner.js';
import { formatTime, formatEnergy, formatIntensity, getTrackDisplayName } from '../utils/format.js';

export function runFormatTests() {
  describe('formatTime', () => {
    it('formats 0 seconds', () => {
      expect(formatTime(0)).toBe('0:00');
    });

    it('formats seconds under a minute', () => {
      expect(formatTime(45)).toBe('0:45');
    });

    it('formats whole minutes', () => {
      expect(formatTime(120)).toBe('2:00');
    });

    it('formats mixed minutes and seconds', () => {
      expect(formatTime(185)).toBe('3:05');
    });

    it('pads single-digit seconds', () => {
      expect(formatTime(61)).toBe('1:01');
    });
  });

  describe('formatEnergy', () => {
    it('returns Balanced for null', () => {
      expect(formatEnergy(null)).toBe('Balanced');
    });

    it('returns Balanced for undefined', () => {
      expect(formatEnergy(undefined)).toBe('Balanced');
    });

    it('formats low energy', () => {
      expect(formatEnergy('low')).toBe('◐ Low Energy');
    });

    it('formats medium energy', () => {
      expect(formatEnergy('medium')).toBe('◑ Medium Energy');
    });

    it('formats high energy', () => {
      expect(formatEnergy('high')).toBe('● High Energy');
    });

    it('returns Balanced for unknown value', () => {
      expect(formatEnergy('extreme')).toBe('Balanced');
    });
  });

  describe('formatIntensity', () => {
    it('returns Moderate for null', () => {
      expect(formatIntensity(null)).toBe('Moderate');
    });

    it('returns Moderate for undefined', () => {
      expect(formatIntensity(undefined)).toBe('Moderate');
    });

    it('formats low intensity (1-3)', () => {
      expect(formatIntensity(1)).toBe('○○○ Gentle');
      expect(formatIntensity(3)).toBe('○○○ Gentle');
    });

    it('formats medium intensity (4-6)', () => {
      expect(formatIntensity(4)).toBe('◐◐◐ Moderate');
      expect(formatIntensity(6)).toBe('◐◐◐ Moderate');
    });

    it('formats high intensity (7-10)', () => {
      expect(formatIntensity(7)).toBe('●●● Intense');
      expect(formatIntensity(10)).toBe('●●● Intense');
    });
  });

  describe('getTrackDisplayName', () => {
    it('uses title when available', () => {
      expect(getTrackDisplayName({ title: 'Calm Waters' })).toBe('Calm Waters');
    });

    it('returns title as-is (server cleans during import)', () => {
      expect(getTrackDisplayName({ title: 'My Track' })).toBe('My Track');
    });

    it('falls back to file_path when no title', () => {
      const result = getTrackDisplayName({ file_path: 'focus/deep_work.mp3' });
      expect(result).toBe('Deep Work');
    });

    it('cleans numeric suffixes from file_path fallback', () => {
      const result = getTrackDisplayName({ file_path: 'focus/calm-waters-186803.mp3' });
      expect(result).toBe('Calm Waters');
    });
  });
}
