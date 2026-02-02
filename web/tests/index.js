/**
 * Drift FM - Test Suite Entry Point
 * Run all tests and report results
 */

import { summary, reset } from './runner.js';
import { runFormatTests } from './format.test.js';
import { runEventsTests } from './events.test.js';
import { runStorageTests } from './storage.test.js';

console.log('╔══════════════════════════════════════╗');
console.log('║     Drift FM Frontend Test Suite     ║');
console.log('╚══════════════════════════════════════╝');
console.log('');

reset();

// Run all test suites
runFormatTests();
runEventsTests();
runStorageTests();

// Print summary
const { passed, failed } = summary();

// Set exit status for CI
if (typeof window !== 'undefined') {
  window.testResults = { passed, failed };
  if (failed === 0) {
    console.log('\n✅ All tests passed!');
  } else {
    console.log(`\n❌ ${failed} test(s) failed`);
  }
}
