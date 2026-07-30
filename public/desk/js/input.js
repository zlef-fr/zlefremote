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
// control, and its chords drive this page — including the chords no browser
// would ever let us capture (Ctrl+Alt+Del, Alt+Tab, Super).
const ZRDeskInput = (() => {
  let stage = null, send = () => {}, view = null, onAction = () => {};
  let driving = false, keyHold = false, raw = false;
  let nx = .5, ny = .5, pendingMove = false, rafId = 0;
  let scrollAcc = 0, scrollAccX = 0;
  let textBuf = '';
  const heldKeys = new Set();
  const heldButtons = new Set();
  const sentMods = new Set();
  let menuHeld = false, menuUsed = false;
  let keyboardLocked = false;

  const BUTTONS = { 0: 'left', 1: 'middle', 2: 'right' };

  // browser key → the names the agent's injector knows (robotgo keyNames)
  const KEYS = {
    Enter: 'enter', NumpadEnter: 'enter', Escape: 'escape', Tab: 'tab',
    Backspace: 'backspace', Delete: 'delete', Insert: 'insert',
    Home: 'home', End: 'end', PageUp: 'pageup', PageDown: 'pagedown',
    ArrowUp: 'up', ArrowDown: 'down', ArrowLeft: 'left', ArrowRight: 'right',
    CapsLock: 'capslock', PrintScreen: 'printscreen', ContextMenu: 'menu',
    NumLock: 'num_lock',
  };
  const CODES = {
    Numpad0: 'num0', Numpad1: 'num1', Numpad2: 'num2', Numpad3: 'num3',
    Numpad4: 'num4', Numpad5: 'num5', Numpad6: 'num6', Numpad7: 'num7',
    Numpad8: 'num8', Numpad9: 'num9', NumpadAdd: 'num_plus',
    NumpadSubtract: 'num_minus', NumpadMultiply: 'num_asterisk',
    NumpadDivide: 'num_slash',
  };

  function keyNameOf(e) {
    if (KEYS[e.key]) return KEYS[e.key];
    if (CODES[e.code]) return CODES[e.code];
    if (/^F([1-9]|1[0-2])$/.test(e.key)) return e.key.toLowerCase();
    return null;
  }

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
    if (!stage) return;
    // unadjustedMovement asks for raw device deltas (no OS pointer acceleration)
    // so a flick here matches a flick over there; older browsers ignore it.
    const req = stage.requestPointerLock({ unadjustedMovement: true });
    if (req && typeof req.catch === 'function') req.catch(() => stage.requestPointerLock());
  }
  function release() { if (document.pointerLockElement) document.exitPointerLock(); else setDriving(false); }
  function toggle() { driving ? release() : take(); }

  function onLockChange() { setDriving(document.pointerLockElement === stage); }

  function setDriving(on) {
    if (driving === on) return;
    driving = on;
    if (!on) releaseAll();
    view && view.setCursor(nx, ny, on);
    onAction(on ? 'grab' : 'ungrab');
  }

  // Never leave a key or button held on the other machine.
  function releaseAll() {
    heldButtons.forEach((b) => send({ t: 'up', b }));
    heldButtons.clear();
    heldKeys.forEach((k) => { if (keyHold) send({ t: 'kup', k }); });
    heldKeys.clear();
    releaseMods();
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
    if (menuHeld) { e.preventDefault(); chord(e); return; }
    if (!driving) return;
    e.preventDefault();

    if (e.repeat && !raw && isPrintable(e)) { queueText(e.key); return; }
    if (e.key === 'Dead' || e.key === 'Unidentified') return;   // composing
    if (isModifier(e.key)) return;                              // pressed lazily

    if (!raw && isPrintable(e)) {
      // the character the browser composed IS what the user typed — send it as
      // text so the host injects it in ITS layout
      releaseMods();
      queueText(e.key);
      return;
    }

    const name = raw ? rawName(e) : keyNameOf(e);
    if (!name) return;
    syncMods(e);
    if (keyHold) { send({ t: 'kdown', k: name }); heldKeys.add(name); }
    else send({ t: 'key', k: name, mods: modList(e) });   // agent < 1.7.0
  }

  function onKeyUp(e) {
    if (e.code === 'ControlRight') {
      e.preventDefault();
      menuHeld = false;
      if (!menuUsed) toggle();     // a bare tap flips control
      return;
    }
    if (!driving) return;
    e.preventDefault();
    if (isModifier(e.key)) { releaseModIfUp(e); return; }
    const name = raw ? rawName(e) : keyNameOf(e);
    if (!name || !heldKeys.has(name)) return;
    heldKeys.delete(name);
    if (keyHold) send({ t: 'kup', k: name });
  }

  // raw mode: forward the physical key, letting the other computer's layout
  // decide the character. Right for games, wrong for typing across layouts.
  function rawName(e) {
    const c = e.code;
    if (/^Key[A-Z]$/.test(c)) return c.slice(3).toLowerCase();
    if (/^Digit[0-9]$/.test(c)) return c.slice(5);
    if (CODES[c]) return CODES[c];
    const punct = {
      Minus: '-', Equal: '=', BracketLeft: '[', BracketRight: ']', Backslash: '\\',
      Semicolon: ';', Quote: "'", Comma: ',', Period: '.', Slash: '/', Backquote: '`',
      Space: 'space', IntlBackslash: '\\',
    };
    if (punct[c]) return punct[c];
    if (KEYS[e.key]) return KEYS[e.key];
    if (/^F([1-9]|1[0-2])$/.test(e.key)) return e.key.toLowerCase();
    const mods = { ShiftLeft: 'shift', ShiftRight: 'shift', ControlLeft: 'ctrl', AltLeft: 'alt', AltRight: 'alt', MetaLeft: 'cmd', MetaRight: 'cmd' };
    return mods[c] || null;
  }

  function isPrintable(e) {
    if (!e.key || e.key.length !== 1) return false;
    if (e.metaKey) return false;
    // AltGr reports as Ctrl+Alt on Windows, yet it composes a real character
    if (e.getModifierState && e.getModifierState('AltGraph')) return true;
    return !e.ctrlKey && !e.altKey;
  }
  function isModifier(k) { return k === 'Shift' || k === 'Control' || k === 'Alt' || k === 'Meta' || k === 'AltGraph'; }

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

  function modList(e) {
    const m = [];
    if (e.shiftKey) m.push('shift');
    if (e.ctrlKey) m.push('ctrl');
    if (e.altKey) m.push('alt');
    if (e.metaKey) m.push('meta');
    return m;
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
    const name = { Shift: 'shift', Control: 'ctrl', Alt: 'alt', Meta: 'cmd' }[e.key];
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
