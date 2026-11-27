import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import 'crypto.dart';
import 'target.dart';

/// One remembered computer.
///
/// The whole record is a credential: whoever holds [keyB64] can drive that
/// machine. It therefore lives in the Android Keystore rather than in plain
/// preferences — the single biggest difference from the web client, whose
/// localStorage was readable by anything running on the origin and wiped by
/// "clear site data".
class ZrDevice {
  const ZrDevice({
    required this.keyB64,
    required this.name,
    this.os = '',
    this.room,
    this.lanOrigin,
    this.persistent = false,
    this.addedAt,
    this.lastUsed,
  });

  final String keyB64;
  final String name;
  final String os;
  final String? room;
  final String? lanOrigin;

  /// The agent runs with `--remember`, so its room is stable and this device
  /// reconnects in one tap. Without it a pairing is good for one session only.
  final bool persistent;

  final DateTime? addedAt;
  final DateTime? lastUsed;

  bool get isLan => lanOrigin != null;

  /// Stable identity for de-duplication. Persistent devices are keyed by their
  /// derived room (the same computer always lands on the same one); everything
  /// else by where it was plus a key prefix.
  String get id {
    if (persistent && room != null) return 'p:$room';
    if (isLan) return 'l:$lanOrigin:${keyB64.substring(0, 8)}';
    return 'e:${room ?? '?'}:${keyB64.substring(0, 8)}';
  }

  Future<ZrTarget> target() async {
    if (isLan) {
      return ZrTarget(
        transport: ZrTransport.lan,
        keyB64: keyB64,
        lanOrigin: lanOrigin,
        persistent: persistent,
      );
    }
    // a saved device stores only the key; the room is re-derived every time so
    // it keeps finding the computer after either side restarts.
    final derived =
        persistent ? await ZrCrypto.deriveRoom(keyB64) : (room ?? '');
    return ZrTarget(
      transport: ZrTransport.relay,
      keyB64: keyB64,
      room: derived,
      persistent: persistent,
    );
  }

  ZrDevice copyWith({
    String? name,
    String? os,
    String? room,
    DateTime? lastUsed,
  }) =>
      ZrDevice(
        keyB64: keyB64,
        name: name?.isNotEmpty == true ? name! : this.name,
        os: os?.isNotEmpty == true ? os! : this.os,
        room: room ?? this.room,
        lanOrigin: lanOrigin,
        persistent: persistent,
        addedAt: addedAt,
        lastUsed: lastUsed ?? this.lastUsed,
      );

  Map<String, dynamic> toJson() => {
        'key': keyB64,
        'name': name,
        'os': os,
        if (room != null) 'room': room,
        if (lanOrigin != null) 'lan': lanOrigin,
        'persistent': persistent,
        'addedAt': addedAt?.millisecondsSinceEpoch,
        'lastUsed': lastUsed?.millisecondsSinceEpoch,
      };

  static ZrDevice? fromJson(Map<String, dynamic> j) {
    final key = j['key'];
    if (key is! String || key.isEmpty) return null;
    DateTime? at(dynamic v) => v is num
        ? DateTime.fromMillisecondsSinceEpoch(v.toInt())
        : null;
    return ZrDevice(
      keyB64: key,
      name: (j['name'] as String?) ?? '',
      os: (j['os'] as String?) ?? '',
      room: j['room'] as String?,
      lanOrigin: j['lan'] as String?,
      persistent: j['persistent'] == true,
      addedAt: at(j['addedAt']),
      lastUsed: at(j['lastUsed']),
    );
  }

  static ZrDevice fromTarget(ZrTarget t, {String name = ''}) => ZrDevice(
        keyB64: t.keyB64,
        name: name,
        room: t.room,
        lanOrigin: t.lanOrigin,
        persistent: t.persistent,
        addedAt: DateTime.now(),
      );
}

/// The saved-computer list. Load once at boot, mutate through here.
class ZrDeviceStore extends ChangeNotifier {
  ZrDeviceStore({FlutterSecureStorage? storage})
      : _storage = storage ??
            const FlutterSecureStorage(
              aOptions: AndroidOptions(encryptedSharedPreferences: true),
            );

  static const _slot = 'zr_devices_v1';

  final FlutterSecureStorage _storage;
  List<ZrDevice> _devices = const [];
  bool loaded = false;

  /// Most recently used first — the one you want is almost always the last one.
  List<ZrDevice> get devices {
    final list = [..._devices];
    list.sort((a, b) => (b.lastUsed ?? b.addedAt ?? DateTime(0))
        .compareTo(a.lastUsed ?? a.addedAt ?? DateTime(0)));
    return list;
  }

  Future<void> load() async {
    try {
      final raw = await _storage.read(key: _slot);
      if (raw != null && raw.isNotEmpty) {
        final decoded = jsonDecode(raw);
        if (decoded is List) {
          _devices = decoded
              .whereType<Map>()
              .map((m) => ZrDevice.fromJson(m.cast<String, dynamic>()))
              .whereType<ZrDevice>()
              .toList();
        }
      }
    } catch (_) {
      _devices = const [];
    }
    loaded = true;
    notifyListeners();
  }

  Future<void> _flush() async {
    await _storage.write(
      key: _slot,
      value: jsonEncode(_devices.map((d) => d.toJson()).toList()),
    );
    notifyListeners();
  }

  Future<ZrDevice> upsert(ZrDevice device) async {
    final i = _devices.indexWhere((d) => d.id == device.id);
    final merged = i >= 0
        ? device.copyWith(
            name: device.name.isNotEmpty ? device.name : _devices[i].name,
            os: device.os.isNotEmpty ? device.os : _devices[i].os,
            lastUsed: DateTime.now(),
          )
        : device.copyWith(lastUsed: DateTime.now());
    if (i >= 0) {
      _devices[i] = merged;
    } else {
      _devices = [..._devices, merged];
    }
    await _flush();
    return merged;
  }

  Future<void> touch(ZrDevice device) async {
    final i = _devices.indexWhere((d) => d.id == device.id);
    if (i < 0) return;
    _devices[i] = _devices[i].copyWith(lastUsed: DateTime.now());
    await _flush();
  }

  Future<void> rename(ZrDevice device, String name) async {
    final i = _devices.indexWhere((d) => d.id == device.id);
    if (i < 0) return;
    _devices[i] = _devices[i].copyWith(name: name);
    await _flush();
  }

  Future<void> remove(ZrDevice device) async {
    _devices = _devices.where((d) => d.id != device.id).toList();
    await _flush();
  }

  bool contains(ZrDevice device) => _devices.any((d) => d.id == device.id);
}
