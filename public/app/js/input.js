// Touch trackpad gesture recognizer. Emits high-level commands through the
// provided send() function. Works with touch (phone) and mouse (desktop test).
//
// The vocabulary is a laptop trackpad's: one finger moves, tap clicks,
// tap-then-press drags, two fingers scroll, a two-finger tap right-clicks, a
// quick sideways two-finger flick is back/forward, pinch zooms, and three
// fingers switch app / open the overview / show the desktop, with a three-finger
// tap for the middle button. Multi-finger verbs travel as intents
// (js/gestures.js) so the computer picks the shortcut for its own desktop.
const ZRInput = (() => {
  let send = () => {};
  let fx = () => {};               // visual-feedback hook: fx(kind, x, y) in pad-local px
  let cfg = { sensitivity: 1.6, natural: true, scrollSpeed: 1.0, gestures: true };

  function attach(pad, sendFn, getCfg, fxFn) {
    send = sendFn;
    if (getCfg) cfg = getCfg;
    if (fxFn) fx = fxFn;

    let pointers = new Map();        // active touches
    let last = null;                 // {x,y} of primary finger
    let moved = 0;                   // total travel of a one-finger gesture
    let startT = 0;
    let tapPending = false;          // a recent quick tap (for tap-drag)
    let tapTimer = null;
    let dragging = false;            // physical drag in progress
    let scrollAccum = { x: 0, y: 0 };
    let twoFinger = false;
    let lastScroll = null;
    let rect = null;                 // pad bounds, captured per gesture
    const local = (p) => rect ? { x: p.x - rect.left, y: p.y - rect.top } : { x: p.x, y: p.y };
    const TAP_MS = 220, TAP_SLOP = 10, DRAGLOCK = { on: false };

    // two-finger state: its own travel (moved only grows under one finger, so
    // without this a fast scroll inside the tap window reads as a tap and
    // right-clicks), the withheld sideways flick, and the pinch anchor.
    let twoTravel = 0, twoStartT = 0, twoStartMid = null, twoStartSpread = null;
    let flickDx = 0, flickOpen = false, flickFired = false;
    let pinchAnchor = null;
    const FLICK_MS = 300, FLICK_DIST = 64, FLICK_GIVE_UP = 26;
    const PINCH_TRIGGER = 34, PINCH_STEP = 56, PINCH_MAX_PER_FRAME = 3;

    // three-finger state: a swipe owns the touch, so nothing else may act on it
    let threeFinger = false, threeFired = false, threeStart = null;
    const SWIPE_DIST = 46;

    const gesturesOn = () => cfg.gestures !== false;

    function gesture(id) {
      const frame = ZRGestures.frame(id);
      if (frame) send(frame);
    }

    function accel(d) {
      const s = cfg.sensitivity || 1.6;
      const mag = Math.abs(d);
      const boost = mag > 8 ? 1.7 : mag > 3 ? 1.2 : 1;
      return d * s * boost;
    }

    function down(e) {
      for (const p of e.changedTouches || [e]) {
        pointers.set(p.identifier ?? 'm', { x: p.clientX, y: p.clientY });
      }
      const n = pointers.size;
      if (n === 1) {
        const p = first();
        rect = pad.getBoundingClientRect();
        last = { x: p.x, y: p.y }; moved = 0; startT = Date.now();
        twoFinger = false;
        const lp = local(p); fx(dragging || tapPending ? 'grab' : 'start', lp.x, lp.y);
        // tap-then-press → start a drag
        if (tapPending) { dragging = true; send({ t: 'down', b: 'left' }); }
      } else if (n === 2) {
        twoFinger = true; lastScroll = midpoint(); fx('end');
        twoStartMid = lastScroll; twoStartT = Date.now(); twoStartSpread = spread();
        twoTravel = 0; flickDx = 0; flickOpen = gesturesOn(); flickFired = false;
        pinchAnchor = null;
      } else if (n === 3) {
        threeFinger = true; threeFired = false; threeStart = centroid();
        twoFinger = false; lastScroll = null; fx('end');
      }
    }

    function move(e) {
      for (const p of e.changedTouches || [e]) {
        if (pointers.has(p.identifier ?? 'm')) pointers.set(p.identifier ?? 'm', { x: p.clientX, y: p.clientY });
      }
      if (threeFinger && pointers.size >= 3) { trackThree(); return; }
      if (twoFinger && pointers.size >= 2) { trackTwo(); return; }
      if (pointers.size === 1 && last) {
        const p = first();
        const dx = p.x - last.x, dy = p.y - last.y;
        moved += Math.hypot(dx, dy);
        last = { x: p.x, y: p.y };
        const lp = local(p); fx('move', lp.x, lp.y);
        const mx = Math.round(accel(dx)), my = Math.round(accel(dy));
        if (mx || my) send({ t: 'mv', dx: mx, dy: my });
      }
    }

    // Two fingers do three things, told apart live: pinching zooms, a quick
    // sideways flick is back/forward, everything else scrolls. The sideways
    // component is withheld while the flick is still possible, then either
    // flushed as scroll or spent as history.
    function trackTwo() {
      const mid = midpoint(), sp = spread();
      if (gesturesOn() && pinchAnchor === null && twoStartSpread !== null &&
          Math.abs(sp - twoStartSpread) > PINCH_TRIGGER) {
        pinchAnchor = twoStartSpread + (sp > twoStartSpread ? PINCH_TRIGGER : -PINCH_TRIGGER);
        flickOpen = false; flickDx = 0;
      }
      if (pinchAnchor !== null) {
        let fired = 0;
        while (fired < PINCH_MAX_PER_FRAME && Math.abs(sp - pinchAnchor) >= PINCH_STEP) {
          const out = sp > pinchAnchor;
          pinchAnchor += out ? PINCH_STEP : -PINCH_STEP;
          gesture(out ? ZRGestures.G.zoomIn : ZRGestures.G.zoomOut);
          fired++;
        }
        lastScroll = mid;
        return;
      }
      if (lastScroll) {
        let dx = mid.x - lastScroll.x;
        const dy = mid.y - lastScroll.y;
        twoTravel += Math.hypot(dx, dy);
        if (flickOpen) {
          flickDx += dx;
          const vertical = Math.abs(mid.y - (twoStartMid ? twoStartMid.y : mid.y));
          if (Date.now() - twoStartT > FLICK_MS || vertical > FLICK_GIVE_UP) {
            // not a flick after all — hand the withheld sideways scroll back
            dx = flickDx; flickOpen = false; flickDx = 0;
          } else {
            dx = 0;
          }
        }
        scrollAccum.x += dx; scrollAccum.y += dy;
        const sx = Math.trunc(scrollAccum.x), sy = Math.trunc(scrollAccum.y);
        if (sx || sy) {
          const dir = cfg.natural ? 1 : -1;
          send({ t: 'scroll', dx: sx * (cfg.scrollSpeed || 1), dy: sy * dir * (cfg.scrollSpeed || 1) });
          scrollAccum.x -= sx; scrollAccum.y -= sy;
        }
      }
      lastScroll = mid;
    }

    // Three fingers = the desktop gestures a trackpad has and a phone doesn't.
    // One shot per gesture: a swipe is a verb, not a stream.
    function trackThree() {
      if (threeFired || !gesturesOn() || !threeStart) return;
      const c = centroid();
      const dx = c.x - threeStart.x, dy = c.y - threeStart.y;
      if (Math.hypot(dx, dy) < SWIPE_DIST) return;
      threeFired = true;
      if (Math.abs(dx) > Math.abs(dy)) {
        gesture(dx > 0 ? ZRGestures.G.appNext : ZRGestures.G.appPrev);
      } else {
        gesture(dy < 0 ? ZRGestures.G.overview : ZRGestures.G.showDesktop);
      }
    }

    function up(e) {
      const wasN = pointers.size;
      for (const p of e.changedTouches || [e]) pointers.delete(p.identifier ?? 'm');
      const dt = Date.now() - startT;

      // a three-finger gesture owns the whole touch: none of the click/scroll
      // paths below may claim what's left of it
      if (threeFinger) {
        if (wasN === 3 && !threeFired && moved < TAP_SLOP && dt < TAP_MS) {
          // three-finger tap → middle click, the button a phone has no room for
          send({ t: 'click', b: 'middle' });
        }
        if (pointers.size === 0) endGesture();
        return;
      }

      if (dragging && pointers.size === 0) { dragging = false; send({ t: 'up', b: 'left' }); fx('end'); return; }

      if (wasN === 2) {
        if (flickOpen && !flickFired && Math.abs(flickDx) >= FLICK_DIST) {
          flickFired = true;
          gesture(flickDx > 0 ? ZRGestures.G.navBack : ZRGestures.G.navForward);
        } else if (dt < TAP_MS && moved < TAP_SLOP && twoTravel < TAP_SLOP &&
                   pinchAnchor === null && !flickFired) {
          // two-finger tap → right click
          send({ t: 'click', b: 'right' });
          if (last && rect) fx('tap', last.x - rect.left, last.y - rect.top, 'right');
        }
        // twoFinger stays set until every finger is up: the one still down must
        // not move the pointer, nor count as a one-finger tap
        if (pointers.size === 0) endGesture();
        return;
      }

      if (wasN === 1 && pointers.size === 0 && !twoFinger) {
        if (moved < TAP_SLOP && dt < TAP_MS) {
          // quick tap → left click, and arm tap-drag window
          send({ t: 'click', b: 'left' });
          if (last && rect) fx('tap', last.x - rect.left, last.y - rect.top);
          tapPending = true;
          clearTimeout(tapTimer);
          tapTimer = setTimeout(() => { tapPending = false; }, 300);
        }
      }
      if (pointers.size === 0) endGesture();
    }

    function endGesture() {
      twoFinger = false; lastScroll = null;
      threeFinger = false; threeFired = false; threeStart = null;
      twoStartMid = null; twoStartSpread = null; twoTravel = 0;
      flickOpen = false; flickFired = false; flickDx = 0;
      pinchAnchor = null;
      scrollAccum = { x: 0, y: 0 };
      fx('end');
    }

    function first() { return pointers.values().next().value; }
    function midpoint() {
      const it = [...pointers.values()];
      return { x: (it[0].x + it[1].x) / 2, y: (it[0].y + it[1].y) / 2 };
    }
    function spread() {
      const it = [...pointers.values()];
      if (it.length < 2) return 0;
      return Math.hypot(it[0].x - it[1].x, it[0].y - it[1].y);
    }
    function centroid() {
      const it = [...pointers.values()];
      let x = 0, y = 0;
      for (const p of it) { x += p.x; y += p.y; }
      return { x: x / it.length, y: y / it.length };
    }

    const opts = { passive: false };
    pad.addEventListener('touchstart', (e) => { e.preventDefault(); down(e); }, opts);
    pad.addEventListener('touchmove', (e) => { e.preventDefault(); move(e); }, opts);
    pad.addEventListener('touchend', (e) => { e.preventDefault(); up(e); }, opts);
    pad.addEventListener('touchcancel', (e) => { up(e); }, opts);

    // mouse fallback (desktop testing)
    let mdown = false;
    pad.addEventListener('mousedown', (e) => { mdown = true; down({ changedTouches: null, clientX: e.clientX, clientY: e.clientY }); });
    pad.addEventListener('mousemove', (e) => { if (mdown) move({ changedTouches: null, clientX: e.clientX, clientY: e.clientY }); });
    window.addEventListener('mouseup', (e) => { if (mdown) { mdown = false; up({ changedTouches: null, clientX: e.clientX, clientY: e.clientY }); } });

    return { setDragLock: (v) => { DRAGLOCK.on = v; } };
  }

  return { attach };
})();
