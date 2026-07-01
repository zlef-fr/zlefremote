// Live screen view. Receives the computer's screen as chunked JPEG frames
// (reassembled here), draws them to a canvas, and turns touches on that canvas
// into absolute pointer moves + clicks — the phone becomes a touchscreen for
// the PC. Frames arrive already E2EE-decrypted as {t:'f', i,s,n,w,h,d}.
const ZRScreen = (() => {
  let send = () => {};
  let canvas = null, ctx = null, hud = null, msgEl = null, stageEl = null;
  let active = false, tuning = false;

  // quality presets → view params sent to the agent
  const PRESETS = {
    low: { fps: 5, q: 40, scale: 40 },
    med: { fps: 8, q: 55, scale: 65 },
    high: { fps: 12, q: 72, scale: 100 },
  };
  let preset = localStorage.getItem('zr_view_q') || 'med';
  if (!PRESETS[preset]) preset = 'med';

  // reassembly of the in-flight frame
  let asm = null;           // { id, n, got, parts:[] }
  let decoding = false, pending = null; // latest complete bytes waiting to decode
  let view = { ox: 0, oy: 0, w: 0, h: 0 }; // drawn image rect in CSS px (hit-test)

  // fps meter
  let fpsCount = 0, fpsAt = 0, lastFps = 0;

  function b64uDec(str) {
    str = str.replace(/-/g, '+').replace(/_/g, '/');
    while (str.length % 4) str += '=';
    const bin = atob(str);
    const out = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
    return out;
  }

  function config(opts) {
    send = opts.send || send;
    canvas = opts.canvas; ctx = canvas.getContext('2d', { alpha: false });
    hud = opts.hud; msgEl = opts.msg; stageEl = opts.stage;
    attachTouch();
  }

  // ── frame reassembly + draw ────────────────────────────────────────────────
  function onFrame(c) {
    if (!active) return;
    if (!asm || asm.id !== c.i) asm = { id: c.i, n: c.n, got: 0, parts: new Array(c.n), w: c.w, h: c.h };
    if (c.s < 0 || c.s >= asm.n || asm.parts[c.s]) return;
    asm.parts[c.s] = b64uDec(c.d);
    asm.got++;
    if (asm.got < asm.n) return;
    // whole frame in hand → concat the raw JPEG bytes
    let total = 0; for (const p of asm.parts) total += p.length;
    const bytes = new Uint8Array(total);
    let off = 0; for (const p of asm.parts) { bytes.set(p, off); off += p.length; }
    const w = asm.w, h = asm.h; asm = null;
    hideMsg();
    present(bytes, w, h);
  }

  function present(bytes, w, h) {
    // coalesce: if a decode is already running, only keep the newest frame
    if (decoding) { pending = { bytes, w, h }; return; }
    decoding = true;
    const blob = new Blob([bytes], { type: 'image/jpeg' });
    createImageBitmap(blob).then((bmp) => {
      draw(bmp, w, h);
      bmp.close && bmp.close();
      tickFps();
    }).catch(() => {}).finally(() => {
      decoding = false;
      if (pending) { const p = pending; pending = null; present(p.bytes, p.w, p.h); }
    });
  }

  function draw(bmp, sw, sh) {
    const rect = canvas.getBoundingClientRect();
    const cw = rect.width, ch = rect.height;
    if (!cw || !ch) return;
    const dpr = window.devicePixelRatio || 1;
    const bw = Math.round(cw * dpr), bh = Math.round(ch * dpr);
    if (canvas.width !== bw || canvas.height !== bh) { canvas.width = bw; canvas.height = bh; }
    const scale = Math.min(cw / sw, ch / sh);
    const dw = sw * scale, dh = sh * scale;
    const ox = (cw - dw) / 2, oy = (ch - dh) / 2;
    view = { ox, oy, w: dw, h: dh };
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.fillStyle = '#000';
    ctx.fillRect(0, 0, cw, ch);
    ctx.drawImage(bmp, ox, oy, dw, dh);
  }

  function tickFps() {
    fpsCount++;
    const now = performance.now();
    if (!fpsAt) fpsAt = now;
    if (now - fpsAt >= 1000) { lastFps = Math.round((fpsCount * 1000) / (now - fpsAt)); fpsCount = 0; fpsAt = now; }
    if (hud) hud.textContent = `${lastFps || fpsCount} fps · ${labelFor(preset)}`;
  }

  // ── activation ─────────────────────────────────────────────────────────────
  function start() {
    active = true; asm = null; pending = null;
    showMsg(ZRScreen._t ? ZRScreen._t('screen_wait') : 'Waiting for the screen…');
    send(Object.assign({ t: 'view', on: true }, PRESETS[preset]));
  }
  function stop() {
    if (!active) return;
    active = false;
    send({ t: 'view', on: false });
    if (ctx) { const r = canvas.getBoundingClientRect(); ctx.setTransform(1,0,0,1,0,0); ctx.clearRect(0, 0, canvas.width, canvas.height); }
    if (hud) hud.textContent = '';
  }
  function setActive(on) { on ? start() : stop(); }
  function isActive() { return active; }

  function setPreset(p) {
    if (!PRESETS[p]) return;
    preset = p; localStorage.setItem('zr_view_q', p);
    if (active) send(Object.assign({ t: 'view', on: true }, PRESETS[preset])); // retune live
    if (hud && lastFps) hud.textContent = `${lastFps} fps · ${labelFor(preset)}`;
  }
  function getPreset() { return preset; }
  function labelFor(p) { return ZRScreen._t ? ZRScreen._t('q_' + p) : p; }

  function onErr(reason) {
    if (!active) return;
    showMsg(ZRScreen._t ? ZRScreen._t(reason === 'unsupported' ? 'screen_unsup' : 'screen_err') : 'Screen unavailable.');
  }

  function showMsg(text) { if (msgEl) { msgEl.textContent = text; msgEl.hidden = false; } }
  function hideMsg() { if (msgEl) msgEl.hidden = true; }

  // ── touch → absolute pointer / click ───────────────────────────────────────
  function normAt(clientX, clientY) {
    const rect = canvas.getBoundingClientRect();
    const cx = clientX - rect.left, cy = clientY - rect.top;
    const nx = view.w ? (cx - view.ox) / view.w : 0;
    const ny = view.h ? (cy - view.oy) / view.h : 0;
    return { nx: Math.min(1, Math.max(0, nx)), ny: Math.min(1, Math.max(0, ny)) };
  }

  function attachTouch() {
    const TAP_MS = 260, SLOP = 12;
    let startPos = null, startT = 0, moved = 0, last = null, n2 = false;
    let moveThrottle = 0;
    let lastTapT = 0, lastTapPos = null;

    const opts = { passive: false };
    canvas.addEventListener('touchstart', (e) => {
      e.preventDefault();
      if (e.touches.length === 2) { n2 = true; return; }
      const t = e.touches[0];
      startPos = { x: t.clientX, y: t.clientY }; last = startPos; startT = Date.now(); moved = 0; n2 = e.touches.length > 1;
    }, opts);

    canvas.addEventListener('touchmove', (e) => {
      e.preventDefault();
      if (e.touches.length !== 1 || !last) return;
      const t = e.touches[0];
      moved += Math.hypot(t.clientX - last.x, t.clientY - last.y);
      last = { x: t.clientX, y: t.clientY };
      const now = Date.now();
      if (now - moveThrottle >= 33) { // ~30 Hz live cursor follow
        moveThrottle = now;
        const p = normAt(t.clientX, t.clientY); send({ t: 'mvabs', nx: p.nx, ny: p.ny });
      }
    }, opts);

    canvas.addEventListener('touchend', (e) => {
      e.preventDefault();
      const dt = Date.now() - startT;
      const quick = dt < TAP_MS && moved < SLOP;
      if (n2) { // two-finger tap → right click at last point
        if (quick && last) { const p = normAt(last.x, last.y); send({ t: 'clickabs', nx: p.nx, ny: p.ny, b: 'right' }); haptic(10); }
        if (e.touches.length === 0) n2 = false;
        return;
      }
      if (quick && last) {
        const p = normAt(last.x, last.y);
        // double-tap → double click
        const now = Date.now();
        const dbl = lastTapPos && (now - lastTapT < 320) && Math.hypot(last.x - lastTapPos.x, last.y - lastTapPos.y) < 24;
        send({ t: 'clickabs', nx: p.nx, ny: p.ny, b: 'left', double: !!dbl });
        haptic(dbl ? 16 : 8);
        lastTapT = dbl ? 0 : now; lastTapPos = dbl ? null : last;
      }
      if (e.touches.length === 0) { startPos = null; last = null; n2 = false; }
    }, opts);
    canvas.addEventListener('touchcancel', () => { startPos = null; last = null; n2 = false; }, opts);
  }
  function haptic(ms) { try { navigator.vibrate && navigator.vibrate(ms); } catch {} }

  return { config, onFrame, onErr, setActive, isActive, setPreset, getPreset, start, stop, presets: PRESETS };
})();
