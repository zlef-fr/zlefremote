import 'dart:async';

import 'package:flutter/services.dart';

/// Things only a real app can do.
///
/// Each of these is a capability the browser flatly denied the old PWA:
///
///  * a **foreground service with a MediaStyle notification**, so the computer's
///    transport controls sit on the lock screen for real. The web client had to
///    loop a silent audio file to get a media card, which stole audio focus and
///    paused the phone's own music — it shipped default-off for that reason.
///    This one holds no audio focus at all.
///  * the connection surviving a screen-off or an app switch, instead of the
///    browser freezing timers and dropping the socket.
///  * the **volume rocker** driving the computer's volume.
///  * a wake lock that is actually honoured.
///
/// Everything is best-effort: if a channel call fails the app keeps working
/// without that surface, and the settings screen says which one is missing.
class ZrNative {
  ZrNative._();
  static final instance = ZrNative._();

  static const _channel = MethodChannel('fr.zlef.remote/native');
  static const _events = EventChannel('fr.zlef.remote/events');

  Stream<ZrNativeEvent>? _stream;

  /// Media-button presses from the notification / lock screen, and volume-key
  /// presses while capture is on.
  Stream<ZrNativeEvent> get events => _stream ??= _events
      .receiveBroadcastStream()
      .map(ZrNativeEvent._parse)
      .where((e) => e != null)
      .cast<ZrNativeEvent>();

  /// Starts the foreground service. [host] is shown as the notification title.
  Future<bool> startSession(String host) => _call('startSession', {'host': host});

  Future<bool> updateSession({required String host, required bool connected}) =>
      _call('updateSession', {'host': host, 'connected': connected});

  Future<bool> stopSession() => _call('stopSession');

  Future<bool> setKeepAwake(bool on) => _call('setKeepAwake', {'on': on});

  Future<bool> setVolumeKeyCapture(bool on) =>
      _call('setVolumeKeyCapture', {'on': on});

  /// POST_NOTIFICATIONS is required from Android 13 for the media card to be
  /// visible at all. Returns true when it is granted (or not needed).
  Future<bool> hasNotificationPermission() =>
      _call('hasNotificationPermission');

  Future<bool> requestNotificationPermission() =>
      _call('requestNotificationPermission');

  /// Whether the keyguard is up right now.
  Future<bool> isDeviceLocked() => _call('isDeviceLocked');

  /// True when the OS has background-restricted this app. On that setting the
  /// foreground service survives but its socket does not: the session dies
  /// silently the moment the screen goes off.
  Future<bool> isBackgroundRestricted() => _call('isBackgroundRestricted');

  /// Opens this app's system settings page, where the restriction is lifted.
  Future<bool> openAppSettings() => _call('openAppSettings');

  /// Hands an APK to the package installer (sideload update channel).
  Future<bool> installApk(String path) => _call('installApk', {'path': path});

  Future<bool> canInstallPackages() => _call('canInstallPackages');

  Future<bool> _call(String method, [Map<String, dynamic>? args]) async {
    try {
      final ok = await _channel.invokeMethod<bool>(method, args);
      return ok ?? false;
    } on PlatformException {
      return false;
    } on MissingPluginException {
      return false;
    }
  }
}

enum ZrNativeEventKind { media, volumeUp, volumeDown, stopRequested, keyguard }

class ZrNativeEvent {
  const ZrNativeEvent(this.kind, [this.value]);
  final ZrNativeEventKind kind;

  /// For [ZrNativeEventKind.media]: the agent's media verb
  /// (`playpause` | `next` | `prev` | `volup` | `voldown` | `mute`).
  final String? value;

  static ZrNativeEvent? _parse(dynamic raw) {
    if (raw is! Map) return null;
    return switch (raw['type']) {
      'media' => ZrNativeEvent(ZrNativeEventKind.media, raw['key'] as String?),
      'volumeUp' => const ZrNativeEvent(ZrNativeEventKind.volumeUp),
      'volumeDown' => const ZrNativeEvent(ZrNativeEventKind.volumeDown),
      'stop' => const ZrNativeEvent(ZrNativeEventKind.stopRequested),
      'keyguard' => ZrNativeEvent(
          ZrNativeEventKind.keyguard,
          raw['locked'] == true ? 'locked' : 'unlocked',
        ),
      _ => null,
    };
  }
}
