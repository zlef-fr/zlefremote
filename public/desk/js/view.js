// The picture. Reassembles the agent's chunked JPEG frames, decodes them off
// the main thread (createImageBitmap) and paints them letterboxed into the
// canvas — plus the remote cursor, which the host's capture does not include.
//
// The image arrives at 10–20 fps but the pointer moves at display rate, so the
// last decoded bitmap is kept and re-blitted on every animation frame while you
// are driving: the cursor stays smooth over a slow picture.
const ZRDeskView = (() => {
  let canvas = null, ctx = null, send = () => {};
  let onStats = () => {};

  // quality presets → the agent clamps fps 1..20, q 20..90, scale 20..100
  const PRESETS = {
    fast: { fps: 20, q: 45, scale: 50 },
    balanced: { fps: 14, q: 60, scale: 75 },
    sharp: { fps: 10, q: 82, scale: 100 },
  };
  const ORDER = ['fast', 'balanced', 'sharp'];
  let preset = localStorage.getItem('zrd_quality');
  if (!PRESETS[preset]) preset = 'balanced';

  let active = false;
  let screens = [], display = 0;

  let asm = null;                        // in-flight frame { id, n, got, parts }
  let decoding = false, pending = null;  // newest complete JPEG waiting to decode
  let bitmap = null, bmpW = 0, bmpH = 0;
  let view = { ox: 0, oy: 0, w: 0, h: 0 };

  let cursor = { x: .5, y: .5, on: false };
  let rafId = 0, dirty = false;

  // stats
  let fpsCount = 0, fpsAt = 0, fps = 0, bytes = 0, bytesAt = 0, kbps = 0, lastFrameAt = 0;

  function b64uDec(str) {
    str = str.replace(/-/g, '+').replace(/_/g, '/');
    while (str.length % 4) str += '=';
    const bin = atob(str);
    const out = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
    return out;
  }

  function config(opts) {
    canvas = opts.canvas;
    ctx = canvas.getContext('2d', { alpha: false });
    send = opts.send || send;
    onStats = opts.onStats || onStats;
    window.addEventListener('resize', () => { dirty = true; schedule(); });
  }

  // ── frames ────────────────────────────────────────────────────────────────
  function frame(c) {
    if (!active) return;
    if (!asm || asm.id !== c.i) asm = { id: c.i, n: c.n, got: 0, parts: new Array(c.n), w: c.w, h: c.h };
    if (c.s < 0 || c.s >= asm.n || asm.parts[c.s]) return;
    const chunk = b64uDec(c.d);
    asm.parts[c.s] = chunk;
    asm.got++;
    noteBytes(chunk.length);
    if (asm.got < asm.n) return;

    let total = 0;
    for (const p of asm.parts) total += p.length;
    const raw = new Uint8Array(total);
    let off = 0;
    for (const p of asm.parts) { raw.set(p, off); off += p.length; }
    const w = asm.w, h = asm.h;
    asm = null;
    decode(raw, w, h);
  }

  function decode(raw, w, h) {
    // a decode already running? keep only the newest frame — a stale picture is
    // worth less than the latency of catching up
    if (decoding) { pending = { raw, w, h }; return; }
    decoding = true;
    createImageBitmap(new Blob([raw], { type: 'image/jpeg' })).then((bmp) => {
      if (bitmap && bitmap.close) bitmap.close();
      bitmap = bmp; bmpW = w; bmpH = h;
      tickFps();
      dirty = true;
      schedule();
    }).catch(() => {}).finally(() => {
      decoding = false;
      if (pending) { const p = pending; pending = null; decode(p.raw, p.w, p.h); }
    });
  }

  // ── painting ──────────────────────────────────────────────────────────────
  function schedule() {
    if (rafId) return;
    rafId = requestAnimationFrame(() => { rafId = 0; render(); });
  }

  function render() {
    if (!ctx || !bitmap) return;
    const rect = canvas.getBoundingClientRect();
    const cw = rect.width, ch = rect.height;
    if (!cw || !ch) return;
    const dpr = window.devicePixelRatio || 1;
    const bw = Math.round(cw * dpr), bh = Math.round(ch * dpr);
    if (canvas.width !== bw || canvas.height !== bh) { canvas.width = bw; canvas.height = bh; }

    const scale = Math.min(cw / bmpW, ch / bmpH);
    const dw = bmpW * scale, dh = bmpH * scale;
    const ox = (cw - dw) / 2, oy = (ch - dh) / 2;
    view = { ox, oy, w: dw, h: dh };

    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.fillStyle = '#000';
    ctx.fillRect(0, 0, cw, ch);
    ctx.drawImage(bitmap, ox, oy, dw, dh);
    if (cursor.on) drawCursor(ox + cursor.x * dw, oy + cursor.y * dh);
    dirty = false;
    // keep re-blitting while the cursor is live so it tracks at display rate
    if (cursor.on) schedule();
  }

  // the same arrow the native client draws: olive fill, dark outline so it stays
  // readable over a white remote window
  function drawCursor(x, y) {
    const path = () => {
      ctx.beginPath();
      ctx.moveTo(x, y);
      ctx.lineTo(x, y + 17);
      ctx.lineTo(x + 4.5, y + 13);
      ctx.lineTo(x + 8, y + 20);
      ctx.lineTo(x + 11, y + 18.5);
      ctx.lineTo(x + 7.5, y + 11.5);
      ctx.lineTo(x + 13, y + 11);
      ctx.closePath();
    };
    path();
    ctx.lineWidth = 3; ctx.lineJoin = 'round';
    ctx.strokeStyle = 'rgba(6,6,10,.85)';
    ctx.stroke();
    ctx.fillStyle = '#bdce74';
    ctx.fill();
  }

  function setCursor(nx, ny, on) {
    cursor = { x: nx, y: ny, on: !!on };
    schedule();
  }

  // ── stats ─────────────────────────────────────────────────────────────────
  function tickFps() {
    const now = performance.now();
    lastFrameAt = now;
    fpsCount++;
    if (!fpsAt) fpsAt = now;
    if (now - fpsAt >= 1000) { fps = Math.round((fpsCount * 1000) / (now - fpsAt)); fpsCount = 0; fpsAt = now; }
    onStats(stats());
  }
  function noteBytes(n) {
    const now = performance.now();
    bytes += n;
    if (!bytesAt) bytesAt = now;
    if (now - bytesAt >= 1000) { kbps = Math.round(bytes / 1024 / ((now - bytesAt) / 1000)); bytes = 0; bytesAt = now; }
  }
  function stats() {
    const stale = lastFrameAt > 0 && performance.now() - lastFrameAt > 2000;
    return { fps, kbps, stale };
  }

  // ── stream control ────────────────────────────────────────────────────────
  function viewCmd() {
    return Object.assign({ t: 'view', on: true, d: display }, PRESETS[preset]);
  }
  function start() { active = true; asm = null; pending = null; send(viewCmd()); }
  function stop() {
    if (!active) return;
    active = false;
    send({ t: 'view', on: false });
  }
  function reset() {
    asm = null; pending = null;
    if (bitmap && bitmap.close) bitmap.close();
    bitmap = null;
    if (ctx) { ctx.setTransform(1, 0, 0, 1, 0, 0); ctx.clearRect(0, 0, canvas.width, canvas.height); }
  }

  function cyclePreset() {
    preset = ORDER[(ORDER.indexOf(preset) + 1) % ORDER.length];
    localStorage.setItem('zrd_quality', preset);
    if (active) send(viewCmd());   // retune live
    return preset;
  }
  function getPreset() { return preset; }

  function setScreens(list) {
    screens = Array.isArray(list) ? list : [];
    if (display >= Math.max(1, screens.length)) display = 0;
  }
  function nextDisplay() {
    if (screens.length < 2) return display;
    display = (display + 1) % screens.length;
    reset();
    if (active) send(viewCmd());
    return display;
  }
  function getDisplay() { return display; }
  function getScreens() { return screens; }
  function rect() { return view; }
  function hasImage() { return !!bitmap; }

  return {
    config, frame, start, stop, reset, setCursor, rect, stats, hasImage,
    cyclePreset, getPreset, setScreens, nextDisplay, getDisplay, getScreens,
  };
})();
