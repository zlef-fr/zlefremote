// Your mouse and keyboard → the other computer.
//
// Pointer: taking control locks the pointer (Pointer Lock API), which gives
// unbounded relative motion and hides the local cursor. We integrate that into
// our own position and send it as an ABSOLUTE normalized point, so a dropped
// frame can never leave the two cursors permanently offset.
//
// Keyboard: the browser hands us `event.key` with the layout, dead keys and
// AltGr composition ALREADY applied — so printable characters travel as text
// (the agent injects them layout-correctly) and only non-character keys and
// chords travel as key events. Modifiers are pressed on the host lazily, right
// before a chord or a click needs them: holding Shift/AltGr over there while
// text is being injected would corrupt what you typed.
//
// Right Ctrl is the local menu key (never forwarded): tap = take/give back
// control, double tap = black the screen out until the next double tap, and
// its chords drive this page. Ctrl+Esc always gives control back.
//
// Taking control in Chromium also goes fullscreen, because that is the only
// state in which the Keyboard Lock API works — and without it the browser and
// the OS keep Escape, the Windows key, Alt+Tab and Ctrl+W for themselves.
const ZRDeskInput = (() => {
  let stage = null, send = () => {}, view = null, onAction = () => {};
  let driving = false, keyHold = false, raw = false;
  let nx = .5, ny = .5, pendingMove = false, rafId = 0;
  let scrollAcc = 0, scrollAccX = 0;
  let textBuf = '';
  const heldKeys = new Map();      // e.code → name pressed on the host
  const heldButtons = new Set();
  const sentMods = new Set();
  let menuHeld = false, menuUsed = false, lastTap = 0, leaveTimer = 0, drivingBeforeTap = false;
  let keyboardLocked = false;
  let autoFull = false;            // fullscreen we entered ourselves, on take()
  let blackout = false, wasDriving = false, wantFull = false;
  let soloMod = null;              // a modifier pressed with nothing else (yet)

  const BUTTONS = { 0: 'left', 1: 'middle', 2: 'right' };
  const DOUBLE_TAP_MS = 320;
  const canLockKeys = !!(navigator.keyboard && navigator.keyboard.lock);

  function config(opts) {
    stage = opts.stage;
    send = opts.send || send;
    view = opts.view;
    onAction = opts.onAction || onAction;

    stage.addEventListener('mousedown', onMouseDown);
    window.addEventListener('mouseup', onMouseUp);
    window.addEventListener('mousemove', onMouseMove);
    stage.addEventListener('wheel', onWheel, { passive: false });
    stage.addEventListener('contextmenu', (e) => e.preventDefault());
    window.addEventListener('keydown', onKeyDown, true);
    window.addEventListener('keyup', onKeyUp, true);
    window.addEventListener('blur', () => { if (driving) release(); });
    document.addEventListener('pointerlockchange', onLockChange);
    document.addEventListener('pointerlockerror', () => onAction('lockfail'));
    document.addEventListener('fullscreenchange', onFullscreenChange);
  }

  // ── taking / giving back control ──────────────────────────────────────────
  function take() {
    if (!stage || blackout) return;
    wantFull = canLockKeys && !document.fullscreenElement;
    // unadjustedMovement asks for raw device deltas (no OS pointer acceleration)
    // so a flick here matches a flick over there; older browsers ignore it.
    const req = stage.requestPointerLock({ unadjustedMovement: true });
    if (req && typeof req.catch === 'function') req.catch(() => stage.requestPointerLock());
  }
  // Fullscreen only once the pointer is locked: requestFullscreen consumes the
  // user activation, and Chrome checks pointer lock's gesture asynchronously —
  // asking for both in the same handler, in either order, loses the lock.
  function enterAutoFull() {
    if (!wantFull) return;
    wantFull = false;
    if (document.fullscreenElement) return;
    autoFull = true;
    document.documentElement.requestFullscreen().catch(() => { autoFull = false; });
  }
  function release() { if (document.pointerLockElement) document.exitPointerLock(); else setDriving(false); }
  function toggle() { driving ? release() : take(); }

  // the one way out that always works: control back, fullscreen off
  function escape() {
    if (blackout) return;
    release();
    autoFull = false;
    if (document.fullscreenElement) document.exitFullscreen().catch(() => {});
  }

  function leaveAutoFull() {
    if (!autoFull) return;
    autoFull = false;
    if (document.fullscreenElement) document.exitFullscreen().catch(() => {});
  }

  function onLockChange() {
    const on = document.pointerLockElement === stage;
    setDriving(on);
    if (on) enterAutoFull();
  }

  function setDriving(on) {
    if (driving === on) return;
    driving = on;
    if (!on) {
      releaseAll();
      // Esc (unlocked browsers), Right Ctrl, focus loss… all land here. Wait
      // out the double-tap window first: if this release was the first half of
      // a blackout, leaving fullscreen only to re-enter it would flash.
      clearTimeout(leaveTimer);
      leaveTimer = setTimeout(() => { if (!blackout && !driving) leaveAutoFull(); }, DOUBLE_TAP_MS);
    }
    view && view.setCursor(nx, ny, on);
    onAction(on ? 'grab' : 'ungrab');
  }

  // Right Ctrl, twice: the screen goes black (this page, not the other
  // computer) and stays black until the next double tap. Control is handed
  // back meanwhile so nothing typed blind lands over there, and taken again on
  // the way out if you had it.
  function toggleBlackout(drivingBefore) {
    if (!blackout) {
      wasDriving = drivingBefore;
      blackout = true;
      release();
      // a black window is not a black screen
      if (canLockKeys && !document.fullscreenElement) {
        autoFull = true;
        document.documentElement.requestFullscreen().catch(() => { autoFull = false; });
      }
      onAction('blackout');
    } else {
      blackout = false;
      onAction('unblackout');
      if (wasDriving) take(); else leaveAutoFull();
    }
  }

  // Never leave a key or button held on the other machine.
  function releaseAll() {
    heldButtons.forEach((b) => send({ t: 'up', b }));
    heldButtons.clear();
    heldKeys.forEach((k) => { if (keyHold) send({ t: 'kup', k }); });
    heldKeys.clear();
    releaseMods();
    soloMod = null;
  }

  // ── pointer ───────────────────────────────────────────────────────────────
  function onMouseMove(e) {
    if (!driving) return;
    const r = view.rect();
    if (!r.w || !r.h) return;
    nx = clamp01(nx + e.movementX / r.w);
    ny = clamp01(ny + e.movementY / r.h);
    view.setCursor(nx, ny, true);
    queueMove();
  }

  // one move per animation frame: the host injects at most one per captured
  // frame anyway, and a flick shouldn't flood a mobile uplink
  function queueMove() {
    pendingMove = true;
    if (rafId) return;
    rafId = requestAnimationFrame(() => {
      rafId = 0;
      if (!pendingMove) return;
      pendingMove = false;
      send({ t: 'mvabs', nx: round6(nx), ny: round6(ny) });
    });
  }

  function onMouseDown(e) {
    if (!driving) return;   // the first click is what takes control (app.js)
    e.preventDefault();
    const b = BUTTONS[e.button];
    if (!b) return;
    soloMod = false;  // Alt+drag, Meta+click: the modifier was not alone
    syncMods(e);
    send({ t: 'mvabs', nx: round6(nx), ny: round6(ny) });   // pin before pressing
    send({ t: 'down', b });
    heldButtons.add(b);
  }

  function onMouseUp(e) {
    const b = BUTTONS[e.button];
    if (!b || !heldButtons.has(b)) return;
    e.preventDefault();
    send({ t: 'up', b });
    heldButtons.delete(b);
  }

  function onWheel(e) {
    if (!driving) return;
    e.preventDefault();
    // deltaMode: 0 = pixels, 1 = lines, 2 = pages → notches
    const div = e.deltaMode === 0 ? 100 : e.deltaMode === 1 ? 3 : 1;
    // browser: down is positive. robotgo: up is positive.
    scrollAcc += -(e.deltaY / div) * 3;
    scrollAccX += (e.deltaX / div) * 3;
    const dy = Math.trunc(scrollAcc), dx = Math.trunc(scrollAccX);
    if (!dx && !dy) return;
    scrollAcc -= dy; scrollAccX -= dx;
    send({ t: 'scroll', dx, dy });
  }

  // ── keyboard ──────────────────────────────────────────────────────────────
  function onKeyDown(e) {
    // Right Ctrl: the local menu key. Held = nothing is forwarded.
    if (e.code === 'ControlRight') {
      e.preventDefault();
      if (!menuHeld) { menuHeld = true; menuUsed = false; releaseAll(); }
      return;
    }
    if (blackout) { e.preventDefault(); return; }
    if (menuHeld) { e.preventDefault(); chord(e); return; }
    // Ctrl+Esc: give control back, from anywhere
    if (e.key === 'Escape' && e.ctrlKey) { e.preventDefault(); escape(); return; }
    if (!driving) return;
    e.preventDefault();

    if (e.repeat && !raw && ZRKeys.isPrintable(e)) { queueText(e.key); return; }
    if (e.key === 'Dead' || e.key === 'Unidentified') return;   // composing
    if (ZRKeys.isModifier(e.key)) {                             // pressed lazily
      if (!e.repeat) soloMod = ZRKeys.modCount(e) === 1 && heldKeys.size === 0 ? e.code : false;
      return;
    }
    soloMod = false;

    if (!raw && ZRKeys.isPrintable(e)) {
      // the character the browser composed IS what the user typed — send it as
      // text so the host injects it in ITS layout
      releaseMods();
      queueText(e.key);
      return;
    }

    // Ctrl+C, Ctrl+Shift+T, Win+E… need a name that survives the modifiers
    const chorded = e.ctrlKey || e.altKey || e.metaKey;
    const name = raw ? ZRKeys.physical(e) : chorded ? ZRKeys.chord(e) : ZRKeys.named(e);
    if (!name) return;
    flushText();
    syncMods(e);
    if (keyHold) {
      // auto-repeat re-sends the press: the host repeats like a local keyboard
      send({ t: 'kdown', k: name });
      heldKeys.set(e.code, name);
    } else {
      send({ t: 'key', k: name, mods: ZRKeys.mods(e) });   // agent < 1.7.0
    }
  }

  function onKeyUp(e) {
    if (e.code === 'ControlRight') {
      e.preventDefault();
      menuHeld = false;
      if (menuUsed) return;
      // a bare tap flips control at once (pointer lock must be requested from
      // inside the key event, a timer is refused); a second tap inside the
      // window turns the pair into a blackout and undoes the first one
      const now = performance.now();
      if (lastTap && now - lastTap < DOUBLE_TAP_MS) {
        lastTap = 0;
        toggleBlackout(drivingBeforeTap);
        return;
      }
      lastTap = now;
      drivingBeforeTap = driving;
      if (!blackout) toggle();
      return;
    }
    if (blackout) { e.preventDefault(); return; }
    if (!driving) return;
    e.preventDefault();
    if (ZRKeys.isModifier(e.key)) {
      // the Windows key (or left Alt) pressed and released alone is a key of
      // its own: Start menu / launcher / menu bar over there
      if (soloMod === e.code && (e.code === 'MetaLeft' || e.code === 'MetaRight' || e.code === 'AltLeft')) {
        send({ t: 'key', k: e.code === 'AltLeft' ? 'alt' : 'meta', mods: [] });
      }
      soloMod = null;
      releaseModIfUp(e);
      return;
    }
    const name = heldKeys.get(e.code);
    if (!name) return;
    heldKeys.delete(e.code);
    if (keyHold) send({ t: 'kup', k: name });
  }

  // text is batched per frame: typing fast shouldn't mean one frame per letter
  function queueText(ch) {
    textBuf += ch;
    if (textBuf.length === 1) requestAnimationFrame(flushText);
  }
  function flushText() {
    if (!textBuf) return;
    send({ t: 'text', s: textBuf });
    textBuf = '';
  }

  // press on the host exactly the modifiers held here, right before an event
  // that needs them
  function syncMods(e) {
    if (!keyHold) return;
    const want = new Set();
    if (e.shiftKey) want.add('shift');
    if (e.ctrlKey) want.add('ctrl');
    if (e.altKey) want.add('alt');
    if (e.metaKey) want.add('cmd');
    want.forEach((m) => { if (!sentMods.has(m)) { send({ t: 'kdown', k: m }); sentMods.add(m); } });
    sentMods.forEach((m) => { if (!want.has(m)) { send({ t: 'kup', k: m }); sentMods.delete(m); } });
  }

  function releaseModIfUp(e) {
    const name = { Shift: 'shift', Control: 'ctrl', Alt: 'alt', Meta: 'cmd', OS: 'cmd' }[e.key];
    if (!name || !sentMods.has(name)) return;
    send({ t: 'kup', k: name });
    sentMods.delete(name);
  }

  function releaseMods() {
    sentMods.forEach((m) => { if (keyHold) send({ t: 'kup', k: m }); });
    sentMods.clear();
  }

  // ── Right Ctrl chords ─────────────────────────────────────────────────────
  function chord(e) {
    menuUsed = true;
    lastTap = 0;   // a chord is not a tap
    switch (e.code) {
      case 'KeyF': onAction('fullscreen'); break;
      case 'KeyP': onAction('quality'); break;
      case 'KeyM': onAction('monitor'); break;
      case 'KeyK': onAction('raw'); break;
      case 'KeyC': onAction('clip'); break;
      case 'KeyQ': onAction('quit'); break;
      case 'Slash': onAction('help'); break;
      // the chords a browser can never hand us — send them as commands instead
      case 'Delete': send({ t: 'key', k: 'delete', mods: ['ctrl', 'alt'] }); break;
      case 'Tab': send({ t: 'key', k: 'tab', mods: ['alt'] }); break;
      case 'KeyS': send({ t: 'key', k: 'cmd', mods: [] }); break;
      case 'Escape': send({ t: 'key', k: 'escape', mods: [] }); break;
      default: menuUsed = e.code !== 'ControlRight'; break;
    }
  }

  // ── keyboard lock (Chromium, fullscreen only) ─────────────────────────────
  // With it, Ctrl+W / Ctrl+T / Alt+Tab / Escape reach this page instead of the
  // browser or the OS. Without it those stay local — the chords above cover the
  // ones that matter.
  async function onFullscreenChange() {
    const fs = !!document.fullscreenElement;
    if (!fs) autoFull = false;
    try {
      if (fs && navigator.keyboard && navigator.keyboard.lock) {
        await navigator.keyboard.lock();
        keyboardLocked = true;
      } else if (!fs && navigator.keyboard && navigator.keyboard.unlock) {
        navigator.keyboard.unlock();
        keyboardLocked = false;
      }
    } catch { keyboardLocked = false; }
    onAction('capture');
  }

  function clamp01(v) { return v < 0 ? 0 : v > 1 ? 1 : v; }
  function round6(v) { return Math.round(v * 1e6) / 1e6; }

  return {
    config, take, release, toggle,
    driving: () => driving,
    setKeyHold: (v) => { keyHold = !!v; },
    setRaw: (v) => { releaseAll(); raw = !!v; },
    raw: () => raw,
    keyboardLocked: () => keyboardLocked,
    canLockKeys: () => canLockKeys,
    blackedOut: () => blackout,
    // place the pointer where the user clicked before control was taken
    warpFromClient(clientX, clientY) {
      const r = view.rect();
      const box = stage.getBoundingClientRect();
      if (!r.w || !r.h) return;
      nx = clamp01((clientX - box.left - r.ox) / r.w);
      ny = clamp01((clientY - box.top - r.oy) / r.h);
      view.setCursor(nx, ny, true);
      send({ t: 'mvabs', nx: round6(nx), ny: round6(ny) });
    },
    center() { nx = .5; ny = .5; view.setCursor(nx, ny, driving); },
    releaseAll,
  };
})();
