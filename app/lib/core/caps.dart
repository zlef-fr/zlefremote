/// What this computer can and cannot do — and why.
///
/// Design rule for the whole app: **never hide a feature because the other side
/// can't do it.** A missing tab teaches the user nothing; a visible, disabled
/// tab that says "this computer's agent was built without screen capture,
/// update it to 1.2 or newer" teaches them exactly what to change. So every
/// capability below is always rendered, and this file is what turns the
/// welcome handshake into a state the UI can explain.
library;

enum ZrCapState {
  /// usable right now
  ready,

  /// the computer genuinely cannot do this (no backlight, no clipboard tool…)
  hostLacks,

  /// the computer could, with a newer agent — the feature postdates its build
  agentOld,

  /// this phone or this Android version is the limit, not the computer
  phoneLacks,

  /// available once the user grants an Android permission
  needsPermission,
}

/// Stable ids — also the i18n key stems (`cap_<id>_title`, `cap_<id>_<state>`).
enum ZrCap {
  screen,
  brightness,
  brightnessPerScreen,
  brightnessMethod,
  clipboard,
  keyHold,
  multiMonitor,
  lockScreenControls,
  volumeKeys,
  backgroundSession,
}

class ZrCapabilities {
  const ZrCapabilities({
    required this.states,
    required this.screens,
    required this.brightScreens,
    required this.brightBackends,
    required this.activeBackend,
    required this.brightness,
    required this.agentLooksOld,
  });

  final Map<ZrCap, ZrCapState> states;

  /// Monitors the computer can stream, in agent order.
  final List<ZrScreenInfo> screens;

  /// Monitors whose brightness can be set (a different set from [screens] —
  /// software gamma reaches external displays a backlight never will).
  final List<ZrBrightScreen> brightScreens;

  /// Mechanisms available to change brightness, when there is a real choice.
  final List<ZrBrightBackend> brightBackends;
  final String? activeBackend;

  /// Current level of the primary display, -1 when the computer won't say.
  final int brightness;

  /// The welcome carried none of the 1.7 capability keys.
  final bool agentLooksOld;

  ZrCapState operator [](ZrCap c) => states[c] ?? ZrCapState.hostLacks;
  bool ready(ZrCap c) => this[c] == ZrCapState.ready;

  static const empty = ZrCapabilities(
    states: {},
    screens: [],
    brightScreens: [],
    brightBackends: [],
    activeBackend: null,
    brightness: -1,
    agentLooksOld: false,
  );

  /// Reads the agent's `welcome` frame.
  ///
  /// Version inference: the agent does not report its version, but the shape of
  /// the frame dates it. `keyhold` and `clip` were both added in agent 1.7.0
  /// and Go always marshals every key of the capability map, so their absence
  /// means "older than 1.7" — enough to tell the user to update rather than
  /// leaving them to guess why the clipboard card is inert.
  factory ZrCapabilities.fromWelcome(Map<String, dynamic> w) {
    final cap = (w['cap'] as Map?)?.cast<String, dynamic>() ?? const {};
    final old = !cap.containsKey('keyhold') && !cap.containsKey('clip');

    final screens = ((w['screens'] as List?) ?? const [])
        .whereType<Map>()
        .map((m) => ZrScreenInfo(
              width: (m['w'] as num?)?.toInt() ?? 0,
              height: (m['h'] as num?)?.toInt() ?? 0,
            ))
        .toList();

    final brightScreens = ((w['brights'] as List?) ?? const [])
        .whereType<Map>()
        .map((m) => ZrBrightScreen(
              name: (m['name'] as String?) ?? '',
              level: (m['v'] as num?)?.toInt() ?? -1,
            ))
        .toList();

    final backends = ((w['backends'] as List?) ?? const [])
        .whereType<Map>()
        .map((m) => ZrBrightBackend(
              id: (m['id'] as String?) ?? '',
              label: (m['label'] as String?) ?? '',
              software: (m['kind'] as String?) == 'software',
            ))
        .toList();

    final canScreen = cap['screen'] == true;
    final canBright = cap['bright'] == true;

    return ZrCapabilities(
      states: {
        ZrCap.screen: canScreen
            ? ZrCapState.ready
            : (old ? ZrCapState.agentOld : ZrCapState.hostLacks),
        ZrCap.brightness: canBright ? ZrCapState.ready : ZrCapState.hostLacks,
        ZrCap.brightnessPerScreen: !canBright
            ? ZrCapState.hostLacks
            : (brightScreens.length > 1
                ? ZrCapState.ready
                : ZrCapState.hostLacks),
        ZrCap.brightnessMethod: !canBright
            ? ZrCapState.hostLacks
            : (backends.length > 1 ? ZrCapState.ready : ZrCapState.hostLacks),
        ZrCap.clipboard: cap['clip'] == true
            ? ZrCapState.ready
            : (old ? ZrCapState.agentOld : ZrCapState.hostLacks),
        ZrCap.keyHold: cap['keyhold'] == true
            ? ZrCapState.ready
            : ZrCapState.agentOld,
        ZrCap.multiMonitor:
            screens.length > 1 ? ZrCapState.ready : ZrCapState.hostLacks,
      },
      screens: screens,
      brightScreens: brightScreens,
      brightBackends: backends,
      activeBackend: w['backend'] as String?,
      brightness: (w['bright'] as num?)?.toInt() ?? -1,
      agentLooksOld: old,
    );
  }

  ZrCapabilities withPhoneCaps({
    required ZrCapState lockScreen,
    required ZrCapState volumeKeys,
  }) =>
      ZrCapabilities(
        states: {
          ...states,
          ZrCap.lockScreenControls: lockScreen,
          ZrCap.volumeKeys: volumeKeys,
        },
        screens: screens,
        brightScreens: brightScreens,
        brightBackends: brightBackends,
        activeBackend: activeBackend,
        brightness: brightness,
        agentLooksOld: agentLooksOld,
      );

  /// Applies a `brightend` reply: switching mechanism changes both the set of
  /// controllable screens and their levels, so the whole brightness block
  /// re-syncs rather than drifting.
  ZrCapabilities withBrightnessBackend(Map<String, dynamic> reply) {
    final brights = ((reply['brights'] as List?) ?? const [])
        .whereType<Map>()
        .map((m) => ZrBrightScreen(
              name: (m['name'] as String?) ?? '',
              level: (m['v'] as num?)?.toInt() ?? -1,
            ))
        .toList();
    return ZrCapabilities(
      states: {
        ...states,
        ZrCap.brightnessPerScreen:
            brights.length > 1 ? ZrCapState.ready : ZrCapState.hostLacks,
      },
      screens: screens,
      brightScreens: brights.isEmpty ? brightScreens : brights,
      brightBackends: brightBackends,
      activeBackend: (reply['backend'] as String?) ?? activeBackend,
      brightness: (reply['bright'] as num?)?.toInt() ?? brightness,
      agentLooksOld: agentLooksOld,
    );
  }
}

class ZrScreenInfo {
  const ZrScreenInfo({required this.width, required this.height});
  final int width;
  final int height;
  String get label => width > 0 ? '$width×$height' : '';
}

class ZrBrightScreen {
  const ZrBrightScreen({required this.name, required this.level});
  final String name;
  final int level;
  ZrBrightScreen copyWith({int? level}) =>
      ZrBrightScreen(name: name, level: level ?? this.level);
}

class ZrBrightBackend {
  const ZrBrightBackend({
    required this.id,
    required this.label,
    required this.software,
  });
  final String id;
  final String label;

  /// Software gamma: reaches external monitors but washes colours out.
  final bool software;
}
