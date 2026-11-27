import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'session.dart';

/// Per-phone preferences. Nothing here is a secret, so plain preferences —
/// device keys live in [ZrDeviceStore] behind the Keystore instead.
class ZrSettings extends ChangeNotifier {
  ZrSettings._(this._prefs);

  final SharedPreferences _prefs;

  static Future<ZrSettings> load() async =>
      ZrSettings._(await SharedPreferences.getInstance());

  // ── pointer ────────────────────────────────────────────────────────────────
  double get pointerSpeed => _prefs.getDouble('pointer_speed') ?? 1.6;
  set pointerSpeed(double v) => _set('pointer_speed', v);

  double get scrollSpeed => _prefs.getDouble('scroll_speed') ?? 1.0;
  set scrollSpeed(double v) => _set('scroll_speed', v);

  bool get naturalScroll => _prefs.getBool('natural_scroll') ?? true;
  set naturalScroll(bool v) => _set('natural_scroll', v);

  /// The multi-finger verbs — three-finger swipes, the sideways two-finger
  /// flick, pinch to zoom. Scrolling and the two-finger right-click are not in
  /// here: those are how a trackpad works, not extras. Off leaves the pad
  /// strictly pointer + scroll, for anyone whose grip keeps firing them.
  bool get gestures => _prefs.getBool('gestures') ?? true;
  set gestures(bool v) => _set('gestures', v);

  // ── feedback ───────────────────────────────────────────────────────────────
  bool get haptics => _prefs.getBool('haptics') ?? true;
  set haptics(bool v) => _set('haptics', v);

  bool get keepAwake => _prefs.getBool('keep_awake') ?? true;
  set keepAwake(bool v) => _set('keep_awake', v);

  // ── native surfaces ────────────────────────────────────────────────────────

  /// Media controls in the notification shade and on the lock screen. Unlike
  /// the web client's silent-audio trick, this holds no audio focus and never
  /// pauses what the phone itself is playing — so it defaults to on.
  bool get lockScreenControls => _prefs.getBool('lock_controls') ?? true;
  set lockScreenControls(bool v) => _set('lock_controls', v);

  /// Put the WHOLE remote over the lock screen — trackpad, keyboard and all —
  /// instead of the playback-only surface. Off by default and confirmed once,
  /// because it hands full control of the computer to anyone holding the phone.
  bool get lockFullControl => _prefs.getBool('lock_full_control') ?? false;
  set lockFullControl(bool v) => _set('lock_full_control', v);

  /// Send the phone's volume rocker to the computer while a session is open.
  bool get volumeKeys => _prefs.getBool('volume_keys') ?? false;
  set volumeKeys(bool v) => _set('volume_keys', v);

  // ── live view ──────────────────────────────────────────────────────────────
  ZrViewQuality get viewQuality =>
      ZrViewQualityParams.parse(_prefs.getString('view_quality'));
  set viewQuality(ZrViewQuality v) => _set('view_quality', v.name);

  // ── language ───────────────────────────────────────────────────────────────
  /// 'en', 'fr', or null to follow the phone.
  String? get language => _prefs.getString('language');
  set language(String? v) {
    if (v == null) {
      _prefs.remove('language');
      notifyListeners();
    } else {
      _set('language', v);
    }
  }

  void _set(String key, Object value) {
    switch (value) {
      case final bool v:
        _prefs.setBool(key, v);
      case final double v:
        _prefs.setDouble(key, v);
      case final String v:
        _prefs.setString(key, v);
      case final int v:
        _prefs.setInt(key, v);
    }
    notifyListeners();
  }
}
