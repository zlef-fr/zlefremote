// A physical keyboard event → the key names the agent's injector knows
// (robotgo keyNames). Shared by the desktop remote (/d/) and the phone client
// (/r/) when it is opened on a computer, so both forward the same keys.
//
// Three shapes of key:
//  - printable, no Ctrl/Alt/Meta: the browser already composed the character
//    (layout, dead keys, AltGr) → the caller sends it as TEXT;
//  - named (Escape, Tab, F1…, arrows, numpad…): a key event;
//  - a chord (Ctrl/Alt/Meta + anything): a key event whose name must survive
//    the modifiers. Ctrl+Shift+T reports key 'T', Ctrl+Shift+1 reports '!' on
//    QWERTY: letters keep the layout's letter (Ctrl+A on AZERTY is the A key),
//    everything else falls back to the physical key.
const ZRKeys = (() => {
  const KEYS = {
    Enter: 'enter', Escape: 'escape', Tab: 'tab',
    Backspace: 'backspace', Delete: 'delete', Insert: 'insert',
    Home: 'home', End: 'end', PageUp: 'pageup', PageDown: 'pagedown',
    ArrowUp: 'up', ArrowDown: 'down', ArrowLeft: 'left', ArrowRight: 'right',
    CapsLock: 'capslock', PrintScreen: 'printscreen', ContextMenu: 'menu',
    NumLock: 'num_lock',
  };
  const CODES = {
    NumpadEnter: 'enter',
    Numpad0: 'num0', Numpad1: 'num1', Numpad2: 'num2', Numpad3: 'num3',
    Numpad4: 'num4', Numpad5: 'num5', Numpad6: 'num6', Numpad7: 'num7',
    Numpad8: 'num8', Numpad9: 'num9', NumpadAdd: 'num_plus',
    NumpadSubtract: 'num_minus', NumpadMultiply: 'num_asterisk',
    NumpadDivide: 'num_slash',
  };
  const PUNCT = {
    Minus: '-', Equal: '=', BracketLeft: '[', BracketRight: ']', Backslash: '\\',
    Semicolon: ';', Quote: "'", Comma: ',', Period: '.', Slash: '/', Backquote: '`',
    Space: 'space', IntlBackslash: '\\',
  };
  const MODCODES = {
    ShiftLeft: 'shift', ShiftRight: 'shift', ControlLeft: 'ctrl',
    AltLeft: 'alt', AltRight: 'alt', MetaLeft: 'cmd', MetaRight: 'cmd',
  };

  // a key that never produces a character: Escape, F5, the numpad…
  function named(e) {
    if (CODES[e.code]) return CODES[e.code];
    if (KEYS[e.key]) return KEYS[e.key];
    if (/^F([1-9]|1[0-9]|2[0-4])$/.test(e.key)) return e.key.toLowerCase();
    return null;
  }

  // the physical key, named the US way: the other computer's layout decides
  // what it means
  function physical(e) {
    const c = e.code;
    if (/^Key[A-Z]$/.test(c)) return c.slice(3).toLowerCase();
    if (/^Digit[0-9]$/.test(c)) return c.slice(5);
    if (PUNCT[c]) return PUNCT[c];
    return named(e) || MODCODES[c] || null;
  }

  // the key to press under Ctrl/Alt/Meta
  function chord(e) {
    const n = named(e);
    if (n) return n;
    const k = e.key || '';
    if (/^[a-z]$/i.test(k)) return k.toLowerCase();
    if (k.length === 1 && k.charCodeAt(0) > 32 && k.charCodeAt(0) < 127 && !e.shiftKey) return k;
    return physical(e);
  }

  function isPrintable(e) {
    if (!e.key || e.key.length !== 1) return false;
    if (e.metaKey) return false;
    // AltGr reports as Ctrl+Alt on Windows, yet it composes a real character
    if (e.getModifierState && e.getModifierState('AltGraph')) return true;
    return !e.ctrlKey && !e.altKey;
  }

  function isModifier(k) {
    return k === 'Shift' || k === 'Control' || k === 'Alt' || k === 'Meta' || k === 'AltGraph' || k === 'OS';
  }

  // modifiers in the `key` verb's vocabulary (agent modMap: meta → cmd)
  function mods(e) {
    const m = [];
    if (e.shiftKey) m.push('shift');
    if (e.ctrlKey) m.push('ctrl');
    if (e.altKey) m.push('alt');
    if (e.metaKey) m.push('meta');
    return m;
  }

  // how many of Shift/Ctrl/Alt/Meta are down (a modifier's own keydown counts)
  function modCount(e) { return (e.shiftKey ? 1 : 0) + (e.ctrlKey ? 1 : 0) + (e.altKey ? 1 : 0) + (e.metaKey ? 1 : 0); }

  return { named, physical, chord, isPrintable, isModifier, mods, modCount };
})();
