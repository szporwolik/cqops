// Applies the saved theme before the stylesheets load to avoid a flash of
// the default theme. Kept as an external file so the CSP does not need
// 'unsafe-inline' for scripts.
(function () {
  try {
    var t = localStorage.getItem('cqops-theme');
    if (t && t !== 'bright') document.documentElement.classList.add(t);
  } catch (e) {}
})();
