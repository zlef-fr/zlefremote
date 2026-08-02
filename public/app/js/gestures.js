// Multi-touch gestures, as intents.
//
// The phone recognises the SHAPE of a gesture — three fingers went left, two
// fingers pinched apart. The computer decides what it means, because the chord
// depends on the desktop: Alt+Tab switches windows on Linux and Windows and
// does nothing on a Mac. So `{t:'gesture', g:'app-next'}` is what crosses the
// wire and agent 1.9+ resolves it.
//
// The chord table below is only for older agents, which would drop a verb they
// don't know. It mirrors the agent's own table; new gestures go there, not here.
const ZRGestures = (() => {
  const G = {
    appNext: 'app-next', appPrev: 'app-prev',
    overview: 'overview', showDesktop: 'show-desktop',
    navBack: 'nav-back', navForward: 'nav-forward',
    zoomIn: 'zoom-in', zoomOut: 'zoom-out',
  };

  const linux = {
    'app-next': ['tab', ['alt']],
    'app-prev': ['tab', ['alt', 'shift']],
    'overview': ['meta', []],
    'show-desktop': ['d', ['meta']],
    'nav-back': ['left', ['alt']],
    'nav-forward': ['right', ['alt']],
    'zoom-in': ['=', ['ctrl']],
    'zoom-out': ['-', ['ctrl']],
  };
  const windows = Object.assign({}, linux, {
    'overview': ['tab', ['meta']], // Task View
  });
  const darwin = {
    'app-next': ['tab', ['meta']],
    'app-prev': ['tab', ['meta', 'shift']],
    'overview': ['up', ['ctrl']],       // Mission Control
    'show-desktop': ['f11', []],
    'nav-back': ['[', ['meta']],        // works in Safari, Chrome and Finder
    'nav-forward': [']', ['meta']],
    'zoom-in': ['=', ['meta']],
    'zoom-out': ['-', ['meta']],
  };

  // set from the welcome frame: which OS the agent runs, and whether it speaks
  // the gesture verb at all
  let host = { os: 'linux', resolves: false };
  function setHost(os, resolves) { host = { os: os || 'linux', resolves: !!resolves }; }

  // The frame to send for an intent — the verb when the agent understands it,
  // its chord otherwise, null when this OS has no sensible binding (better
  // nothing than a wrong keystroke on someone's desktop).
  function frame(id) {
    if (host.resolves) return { t: 'gesture', g: id };
    const table = host.os === 'darwin' ? darwin : host.os === 'windows' ? windows : linux;
    const chord = table[id];
    return chord ? { t: 'key', k: chord[0], mods: chord[1] } : null;
  }

  return { G, setHost, frame };
})();
