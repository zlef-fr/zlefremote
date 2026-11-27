/// Multi-touch gestures, as intents.
///
/// The phone recognises the *shape* of a gesture — three fingers went left, two
/// fingers pinched apart — and the computer decides what that means, because
/// the chord depends on the desktop: Alt+Tab switches windows on Linux and
/// Windows and does nothing on a Mac. So [ZrGesture] is what crosses the wire
/// (`{t:'gesture', g:'app-next'}`) and agent 1.9+ resolves it.
///
/// [fallbackChord] exists only for older agents: they don't know the verb, so
/// the phone sends the chord itself and gets the mapping right for the OS the
/// agent reported. It is a copy of the agent's table, deliberately small and
/// deliberately frozen — new gestures go in the agent, not here.
library;

class ZrGesture {
  /// three fingers left / right
  static const appNext = 'app-next';
  static const appPrev = 'app-prev';

  /// three fingers up / down
  static const overview = 'overview';
  static const showDesktop = 'show-desktop';

  /// two fingers flicked sideways — browser/file-manager history
  static const navBack = 'nav-back';
  static const navForward = 'nav-forward';

  /// pinch on the trackpad
  static const zoomIn = 'zoom-in';
  static const zoomOut = 'zoom-out';

  static const workspaceNext = 'workspace-next';
  static const workspacePrev = 'workspace-prev';
}

class ZrChord {
  const ZrChord(this.key, [this.mods = const []]);
  final String key;
  final List<String> mods;
}

const _linux = <String, ZrChord>{
  ZrGesture.appNext: ZrChord('tab', ['alt']),
  ZrGesture.appPrev: ZrChord('tab', ['alt', 'shift']),
  ZrGesture.overview: ZrChord('meta'),
  ZrGesture.showDesktop: ZrChord('d', ['meta']),
  ZrGesture.navBack: ZrChord('left', ['alt']),
  ZrGesture.navForward: ZrChord('right', ['alt']),
  ZrGesture.zoomIn: ZrChord('=', ['ctrl']),
  ZrGesture.zoomOut: ZrChord('-', ['ctrl']),
  ZrGesture.workspaceNext: ZrChord('right', ['ctrl', 'alt']),
  ZrGesture.workspacePrev: ZrChord('left', ['ctrl', 'alt']),
};

const _windows = <String, ZrChord>{
  ZrGesture.appNext: ZrChord('tab', ['alt']),
  ZrGesture.appPrev: ZrChord('tab', ['alt', 'shift']),
  ZrGesture.overview: ZrChord('tab', ['meta']), // Task View
  ZrGesture.showDesktop: ZrChord('d', ['meta']),
  ZrGesture.navBack: ZrChord('left', ['alt']),
  ZrGesture.navForward: ZrChord('right', ['alt']),
  ZrGesture.zoomIn: ZrChord('=', ['ctrl']),
  ZrGesture.zoomOut: ZrChord('-', ['ctrl']),
  ZrGesture.workspaceNext: ZrChord('right', ['ctrl', 'meta']),
  ZrGesture.workspacePrev: ZrChord('left', ['ctrl', 'meta']),
};

const _darwin = <String, ZrChord>{
  ZrGesture.appNext: ZrChord('tab', ['meta']),
  ZrGesture.appPrev: ZrChord('tab', ['meta', 'shift']),
  ZrGesture.overview: ZrChord('up', ['ctrl']),
  ZrGesture.showDesktop: ZrChord('f11'),
  ZrGesture.navBack: ZrChord('[', ['meta']),
  ZrGesture.navForward: ZrChord(']', ['meta']),
  ZrGesture.zoomIn: ZrChord('=', ['meta']),
  ZrGesture.zoomOut: ZrChord('-', ['meta']),
  ZrGesture.workspaceNext: ZrChord('right', ['ctrl']),
  ZrGesture.workspacePrev: ZrChord('left', ['ctrl']),
};

/// The chord to send to an agent too old to understand the intent. Null when
/// this OS has no sensible binding — better nothing than a wrong keystroke on
/// someone's desktop.
ZrChord? fallbackChord(String gesture, String hostOs) {
  final table = switch (hostOs) {
    'darwin' => _darwin,
    'windows' => _windows,
    _ => _linux,
  };
  return table[gesture];
}
