/**
 * Analytics module — listen event reporting.
 *
 * reportListen: sends listen events to /api/tracks/:id/play (SQLite)
 */

/**
 * Report a listen event to the server.
 * @param {number} trackId
 * @param {object} data - { event, listen_seconds, mood, position, weight? }
 * @param {object} options - { beacon: boolean }
 */
export function reportListen(trackId, data, options = {}) {
  const url = `/api/tracks/${trackId}/play`;
  const payload = JSON.stringify(data);
  try {
    if (options.beacon && navigator.sendBeacon) {
      navigator.sendBeacon(url, new globalThis.Blob([payload], { type: 'application/json' }));
      return;
    }
    fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: payload,
      keepalive: true
    }).catch(err => console.warn('Listen report error:', err.message));
  } catch (err) {
    console.warn('Listen report error:', err.message);
  }
}
