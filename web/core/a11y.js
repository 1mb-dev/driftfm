/**
 * Drift FM - Accessibility Utilities
 * Shared focus management helpers used by panel modules
 */

/**
 * Handle focus trap within a panel element.
 * Cycles focus between first and last focusable children on Tab.
 * @param {HTMLElement} panel - Container element to trap focus within
 * @param {KeyboardEvent} e - Keydown event
 */
export function trapFocus(panel, e) {
  if (e.key !== 'Tab') return;

  const focusables = panel.querySelectorAll(
    'button, a, input, [tabindex="0"], [href]'
  );
  if (focusables.length === 0) return;

  const first = focusables[0];
  const last = focusables[focusables.length - 1];

  if (e.shiftKey && document.activeElement === first) {
    e.preventDefault();
    last.focus();
  } else if (!e.shiftKey && document.activeElement === last) {
    e.preventDefault();
    first.focus();
  }
}
