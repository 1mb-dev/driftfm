/**
 * Drift FM - Event Bus
 * Lightweight pub/sub for loose coupling between modules
 */

class EventBus {
  constructor() {
    this.listeners = new Map();
  }

  /**
   * Subscribe to an event
   * @param {string} event - Event name
   * @param {Function} callback - Handler function
   * @returns {Function} Unsubscribe function
   */
  on(event, callback) {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set());
    }
    this.listeners.get(event).add(callback);

    // Return unsubscribe function
    return () => this.off(event, callback);
  }

  /**
   * Unsubscribe from an event
   * @param {string} event - Event name
   * @param {Function} callback - Handler function to remove
   */
  off(event, callback) {
    const eventListeners = this.listeners.get(event);
    if (eventListeners) {
      eventListeners.delete(callback);
    }
  }

  /**
   * Emit an event to all subscribers
   * @param {string} event - Event name
   * @param {*} data - Data to pass to handlers
   */
  emit(event, data) {
    const eventListeners = this.listeners.get(event);
    if (eventListeners) {
      eventListeners.forEach(callback => {
        try {
          callback(data);
        } catch (err) {
          console.error(`[EventBus] Error in handler for "${event}":`, err);
        }
      });
    }
  }

}

// Singleton instance
export const events = new EventBus();

/**
 * Event types used in Drift FM (active events only)
 *
 * Settings events:
 * - 'settings:changed' { key, value }  — theme, instrumental, showLyrics, volume
 * - 'settings:opened'  {}
 * - 'settings:closed'  {}
 *
 * About events:
 * - 'about:opened'     {}
 * - 'about:closed'     {}
 *
 * Lyrics events:
 * - 'lyrics:opened'    {}
 * - 'lyrics:closed'    {}
 */
