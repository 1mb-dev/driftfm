/**
 * Drift FM - Formatting Utilities
 * Pure functions for text and data formatting
 */

/**
 * Format seconds as M:SS
 * @param {number} seconds
 * @returns {string}
 */
export function formatTime(seconds) {
  const mins = Math.floor(seconds / 60);
  const secs = Math.floor(seconds % 60);
  return `${mins}:${secs.toString().padStart(2, '0')}`;
}

/**
 * Format energy level for display
 * @param {string} energy - 'low', 'medium', 'high', or null
 * @returns {string}
 */
export function formatEnergy(energy) {
  if (!energy) return 'Balanced';
  const labels = {
    'low': '◐ Low Energy',
    'medium': '◑ Medium Energy',
    'high': '● High Energy'
  };
  return labels[energy] || 'Balanced';
}

/**
 * Format intensity level for display
 * @param {number} intensity - 1-10 or null
 * @returns {string}
 */
export function formatIntensity(intensity) {
  if (!intensity && intensity !== 0) return 'Moderate';
  if (intensity <= 3) return '○○○ Gentle';
  if (intensity <= 6) return '◐◐◐ Moderate';
  return '●●● Intense';
}

/**
 * Get display name for a track.
 * Uses title if set, otherwise cleans up the file path.
 * @param {Object} track
 * @returns {string}
 */
export function getTrackDisplayName(track) {
  if (track.title) return track.title;
  // Fallback: extract filename, strip extension and clean up
  const filename = (track.file_path || '').split('/').pop() || 'Untitled';
  return filename
    .replace(/\.mp3$/, '')
    .replace(/[-_]\d{4,}.*$/, '')
    .replace(/[-_]/g, ' ')
    .replace(/\b\w/g, c => c.toUpperCase())
    .replace(/\s+/g, ' ')
    .trim() || 'Untitled';
}
