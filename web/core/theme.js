/**
 * Drift FM - Theme Utilities
 * Shared theme application logic used by settings and about panels
 */

/**
 * Apply a resolved theme ('dark' or 'light') to the document.
 * Updates the data-theme attribute and browser chrome color.
 */
export function applyTheme(theme) {
  document.documentElement.dataset.theme = theme;
  const themeColor = theme === 'dark' ? '#0a0a0a' : '#fafafa';
  document.getElementById('theme-color')?.setAttribute('content', themeColor);
}

/**
 * Resolve a theme preference to an effective theme.
 * 'auto' resolves to system preference; 'dark'/'light' pass through.
 */
export function resolveTheme(pref) {
  if (pref === 'auto') {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }
  return pref;
}
