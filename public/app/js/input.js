// Touch trackpad + scroll gesture recognizer. Emits high-level commands through
// the provided send() function. Works with touch (phone) and mouse (desktop test).
const ZRInput = (() => {
  let send = () => {};
  let cfg = { sensitivity: 1.6, natural: true, scrollSpeed: 1.0 };

  function attach(pad, sendFn, getCfg) {
    send = sendFn;
    if (getCfg) cfg = getCfg;

    let pointers = new Map();        // active touches
    let last = null;                 // {x,y} of primary finger
    let moved = 0;                   // total travel of a gesture
    let startT = 0;
    let tapPending = false;          // a recent quick tap (for tap-drag)
    let tapTimer = null;
    let dragging = false;            // physical drag in progress
    let scrollAccum = { x: 0, y: 0 };
    let twoFinger = false;
    let lastScroll = null;
    const TAP_MS = 220, TAP_SLOP = 10, DRAGLOCK = { on: false };

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
        last = { x: p.x, y: p.y }; moved = 0; startT = Date.now();
        twoFinger = false;
        // tap-then-press → start a drag
        if (tapPending) { dragging = true; send({ t: 'down', b: 'left' }); }
      } else if (n === 2) {
        twoFinger = true; lastScroll = midpoint();
      }
    }

    function move(e) {
      for (const p of e.changedTouches || [e]) {
        if (pointers.has(p.identifier ?? 'm')) pointers.set(p.identifier ?? 'm', { x: p.clientX, y: p.clientY });
      }
      if (twoFinger && pointers.size >= 2) {
        const mid = midpoint();
        if (lastScroll) {
          const dx = mid.x - lastScroll.x, dy = mid.y - lastScroll.y;
          scrollAccum.x += dx; scrollAccum.y += dy;
          const sx = Math.trunc(scrollAccum.x), sy = Math.trunc(scrollAccum.y);
          if (sx || sy) {
            const dir = cfg.natural ? 1 : -1;
            send({ t: 'scroll', dx: sx * (cfg.scrollSpeed||1), dy: sy * dir * (cfg.scrollSpeed||1) });
            scrollAccum.x -= sx; scrollAccum.y -= sy;
          }
        }
        lastScroll = mid;
        return;
      }
      if (pointers.size === 1 && last) {
        const p = first();
        const dx = p.x - last.x, dy = p.y - last.y;
        moved += Math.hypot(dx, dy);
        last = { x: p.x, y: p.y };
        const mx = Math.round(accel(dx)), my = Math.round(accel(dy));
        if (mx || my) send({ t: 'mv', dx: mx, dy: my });
      }
    }

    function up(e) {
      const wasN = pointers.size;
      for (const p of e.changedTouches || [e]) pointers.delete(p.identifier ?? 'm');
      const dt = Date.now() - startT;

      if (dragging && pointers.size === 0) { dragging = false; send({ t: 'up', b: 'left' }); return; }

      if (wasN === 2 && pointers.size <= 1 && moved < TAP_SLOP && dt < TAP_MS && !twoFingerMoved()) {
        // two-finger tap → right click
        send({ t: 'click', b: 'right' });
        twoFinger = false;
        return;
      }

      if (wasN === 1 && pointers.size === 0 && !twoFinger) {
        if (moved < TAP_SLOP && dt < TAP_MS) {
          // quick tap → left click, and arm tap-drag window
          send({ t: 'click', b: 'left' });
          tapPending = true;
          clearTimeout(tapTimer);
          tapTimer = setTimeout(() => { tapPending = false; }, 300);
        }
      }
      if (pointers.size === 0) { twoFinger = false; lastScroll = null; }
    }

    function first() { return pointers.values().next().value; }
    function midpoint() {
      const it = [...pointers.values()];
      return { x: (it[0].x + it[1].x) / 2, y: (it[0].y + it[1].y) / 2 };
    }
    function twoFingerMoved() { return moved >= TAP_SLOP; }

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
