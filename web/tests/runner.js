/**
 * Drift FM - Minimal Test Runner
 * Lightweight assertion library for browser-based tests
 */

let passed = 0;
let failed = 0;
let currentSuite = '';
let currentBeforeEach = null;

export function describe(name, fn) {
  currentSuite = name;
  const previousBeforeEach = currentBeforeEach;
  currentBeforeEach = null;
  console.group(`📦 ${name}`);
  fn();
  console.groupEnd();
  currentBeforeEach = previousBeforeEach;
}

export function beforeEach(fn) {
  currentBeforeEach = fn;
}

export function it(name, fn) {
  try {
    if (currentBeforeEach) currentBeforeEach();
    fn();
    passed++;
    console.log(`  ✓ ${name}`);
  } catch (err) {
    failed++;
    console.error(`  ✗ ${name}`);
    console.error(`    ${err.stack || err.message}`);
  }
}

export function expect(actual) {
  return {
    toBe(expected) {
      if (actual !== expected) {
        throw new Error(`Expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
      }
    },
    toEqual(expected) {
      if (JSON.stringify(actual) !== JSON.stringify(expected)) {
        throw new Error(`Expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
      }
    },
    toBeTruthy() {
      if (!actual) {
        throw new Error(`Expected truthy, got ${JSON.stringify(actual)}`);
      }
    },
    toBeFalsy() {
      if (actual) {
        throw new Error(`Expected falsy, got ${JSON.stringify(actual)}`);
      }
    },
    toBeGreaterThan(n) {
      if (!(actual > n)) {
        throw new Error(`Expected ${actual} to be greater than ${n}`);
      }
    },
    toBeLessThanOrEqual(n) {
      if (!(actual <= n)) {
        throw new Error(`Expected ${actual} to be <= ${n}`);
      }
    },
    toBeNull() {
      if (actual !== null) {
        throw new Error(`Expected null, got ${JSON.stringify(actual)}`);
      }
    },
    toHaveBeenCalled() {
      if (!actual.called) {
        throw new Error('Expected function to have been called');
      }
    },
    toHaveBeenCalledWith(expected) {
      if (!actual.calledWith || JSON.stringify(actual.calledWith) !== JSON.stringify(expected)) {
        throw new Error(`Expected call with ${JSON.stringify(expected)}, got ${JSON.stringify(actual.calledWith)}`);
      }
    }
  };
}

export function spy() {
  const fn = function(...args) {
    fn.called = true;
    fn.callCount++;
    fn.calledWith = args.length === 1 ? args[0] : args;
    fn.calls.push(args);
  };
  fn.called = false;
  fn.callCount = 0;
  fn.calledWith = undefined;
  fn.calls = [];
  fn.reset = () => {
    fn.called = false;
    fn.callCount = 0;
    fn.calledWith = undefined;
    fn.calls = [];
  };
  return fn;
}

export function summary() {
  console.log('');
  console.log(`Tests: ${passed} passed, ${failed} failed`);
  return { passed, failed };
}

export function reset() {
  passed = 0;
  failed = 0;
}
