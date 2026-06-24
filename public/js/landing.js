// reveal-on-scroll + gentle parallax for the hero art.
// Progressive enhancement: the hidden state only applies once we add `.js`, so
// if this script never runs the content is fully visible (never a blank page).
(function () {
  document.documentElement.classList.add('js');

  const io = new IntersectionObserver((entries) => {
    for (const e of entries) {
      if (e.isIntersecting) { e.target.classList.add('in'); io.unobserve(e.target); }
    }
  }, { threshold: 0.1, rootMargin: '0px 0px -8% 0px' });

  const reveals = document.querySelectorAll('.reveal');
  reveals.forEach((el, i) => {
    el.style.transitionDelay = (Math.min(i % 4, 3) * 70) + 'ms';
    io.observe(el);
  });
  // Safety net: if anything is still hidden after a beat (e.g. observer quirks
  // on some mobile browsers), reveal it so content is never stuck invisible.
  setTimeout(() => reveals.forEach((el) => el.classList.add('in')), 3000);

  const art = document.querySelector('.hero-art');
  if (art && matchMedia('(hover: hover)').matches && !matchMedia('(prefers-reduced-motion: reduce)').matches) {
    window.addEventListener('mousemove', (e) => {
      const cx = (e.clientX / innerWidth - 0.5);
      const cy = (e.clientY / innerHeight - 0.5);
      art.style.transform = `translate3d(${cx * 14}px, ${cy * 10}px, 0)`;
    }, { passive: true });
  }
})();
