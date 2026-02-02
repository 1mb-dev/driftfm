// Theme initialization — runs blocking before CSS to prevent flash
(function() {
  try {
    const stored = localStorage.getItem('drift-theme');
    const systemDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    let theme;
    if (stored === 'dark' || stored === 'light') {
      theme = stored;
    } else {
      theme = systemDark ? 'dark' : 'light';
    }
    document.documentElement.dataset.theme = theme;
    const themeColor = theme === 'dark' ? '#0a0a0a' : '#fafafa';
    const el = document.getElementById('theme-color');
    if (el) el.setAttribute('content', themeColor);
    const lastMood = localStorage.getItem('drift-last-mood');
    if (lastMood) {
      document.documentElement.dataset.splashMood = lastMood;
    }
  } catch (e) {
    console.warn('[Theme] Init failed, using CSS defaults:', e.message);
  }
})();
