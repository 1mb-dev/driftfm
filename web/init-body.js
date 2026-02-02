// Set last mood on body early so --color-accent token resolves for splash glow
(function() {
  const m = document.documentElement.dataset.splashMood;
  if (m) document.body.dataset.mood = m;
})();
