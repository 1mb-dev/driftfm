/**
 * Tests for events.js (EventBus)
 */

import { describe, it, expect, spy } from './runner.js';
import { events } from '../core/events.js';

export function runEventsTests() {
  describe('EventBus.on/emit', () => {
    it('calls listener when event emitted', () => {
      const handler = spy();
      events.on('test:basic', handler);
      events.emit('test:basic', { value: 42 });

      expect(handler.called).toBeTruthy();
      expect(handler.calledWith).toEqual({ value: 42 });

      events.off('test:basic', handler);
    });

    it('calls multiple listeners', () => {
      const handler1 = spy();
      const handler2 = spy();

      events.on('test:multi', handler1);
      events.on('test:multi', handler2);
      events.emit('test:multi', 'data');

      expect(handler1.callCount).toBe(1);
      expect(handler2.callCount).toBe(1);

      events.off('test:multi', handler1);
      events.off('test:multi', handler2);
    });

    it('does not call listener for different event', () => {
      const handler = spy();
      events.on('test:a', handler);
      events.emit('test:b', 'data');

      expect(handler.called).toBeFalsy();

      events.off('test:a', handler);
    });
  });

  describe('EventBus.off', () => {
    it('removes listener', () => {
      const handler = spy();
      events.on('test:remove', handler);
      events.off('test:remove', handler);
      events.emit('test:remove', 'data');

      expect(handler.called).toBeFalsy();
    });

    it('returns unsubscribe function from on()', () => {
      const handler = spy();
      const unsub = events.on('test:unsub', handler);
      unsub();
      events.emit('test:unsub', 'data');

      expect(handler.called).toBeFalsy();
    });
  });

  describe('EventBus error handling', () => {
    it('continues to other listeners if one throws', () => {
      const badHandler = () => { throw new Error('oops'); };
      const goodHandler = spy();

      events.on('test:error', badHandler);
      events.on('test:error', goodHandler);

      // Should not throw
      events.emit('test:error', 'data');

      expect(goodHandler.called).toBeTruthy();

      events.off('test:error', badHandler);
      events.off('test:error', goodHandler);
    });
  });
}
