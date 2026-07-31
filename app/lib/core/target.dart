import 'crypto.dart';

/// How to reach one computer.
///
/// The agent prints exactly one pairing URL, in one of two shapes:
///
///   relay  `https://remote.zlef.fr/r/<ROOM>#k=<key>[&p=1]`
///   LAN    `https://<ip>:<port>/#k=<key>`
///
/// The key never leaves the fragment, so a relay URL is safe to send over any
/// channel: the server it names can read the room code but not the traffic.
/// `&p=1` marks a `--remember` agent — its room is derived from its key, which
/// is what makes a saved device reconnectable without a new scan.
enum ZrTransport { relay, lan }

class ZrTarget {
  const ZrTarget({
    required this.transport,
    required this.keyB64,
    this.room,
    this.lanOrigin,
    this.persistent = false,
  });

  final ZrTransport transport;
  final String keyB64;

  /// Relay room code (uppercase). Null for a LAN target.
  final String? room;

  /// `https://host:port` of the agent's own server. Null for a relay target.
  final String? lanOrigin;

  /// The agent runs with `--remember`: stable room, so this device is savable.
  final bool persistent;

  /// WebSocket endpoint for this target.
  String get wsUrl {
    if (transport == ZrTransport.lan) {
      final origin = lanOrigin!;
      final scheme = origin.startsWith('https') ? 'wss' : 'ws';
      return '$scheme://${origin.replaceFirst(RegExp(r'^https?://'), '')}/ws';
    }
    return 'wss://$relayHost/ws';
  }

  /// Host of the LAN agent, used to scope the self-signed-certificate exception.
  String? get lanHost {
    if (lanOrigin == null) return null;
    return Uri.tryParse(lanOrigin!)?.host;
  }

  static const relayHost = 'remote.zlef.fr';

  /// Parses a pairing link (from a QR code, a deep link, or pasted text).
  /// Returns null when the string carries no usable key.
  static ZrTarget? parse(String raw) {
    final s = raw.trim();
    if (s.isEmpty) return null;

    final keyMatch = RegExp(r'[#&?]k=([A-Za-z0-9\-_]+)').firstMatch(s);
    if (keyMatch == null) return null;
    final key = keyMatch.group(1)!;
    // reject anything that isn't a 256-bit key before we build a device from it
    try {
      if (ZrCrypto.b64uDecode(key).length != 32) return null;
    } catch (_) {
      return null;
    }

    final persistent = RegExp(r'[#&]p=1\b').hasMatch(s);

    // both remote surfaces (/r/ phone, /d/ desktop) name the same room
    final roomMatch = RegExp(r'/[rd]/([A-Za-z0-9]{4,8})').firstMatch(s);
    if (roomMatch != null) {
      return ZrTarget(
        transport: ZrTransport.relay,
        keyB64: key,
        room: roomMatch.group(1)!.toUpperCase(),
        persistent: persistent,
      );
    }

    // no room in the path → the agent's own LAN server
    final uri = Uri.tryParse(s.split('#').first);
    if (uri != null && uri.hasAuthority) {
      final port = uri.hasPort ? ':${uri.port}' : '';
      return ZrTarget(
        transport: ZrTransport.lan,
        keyB64: key,
        lanOrigin: '${uri.scheme}://${uri.host}$port',
        persistent: persistent,
      );
    }

    // a bare key (pasted on its own) is only actionable for a remembered agent,
    // whose room we can re-derive; the caller resolves that asynchronously.
    return ZrTarget(
      transport: ZrTransport.relay,
      keyB64: key,
      persistent: persistent,
    );
  }

  /// Fills in a missing relay room by re-deriving it from the key. A saved
  /// device stores only the key, so this runs on every reconnect.
  Future<ZrTarget> resolved() async {
    if (transport != ZrTransport.relay || (room != null && room!.isNotEmpty)) {
      return this;
    }
    return ZrTarget(
      transport: transport,
      keyB64: keyB64,
      room: await ZrCrypto.deriveRoom(keyB64),
      persistent: persistent,
    );
  }

  ZrTarget copyWith({String? room, bool? persistent}) => ZrTarget(
        transport: transport,
        keyB64: keyB64,
        room: room ?? this.room,
        lanOrigin: lanOrigin,
        persistent: persistent ?? this.persistent,
      );
}
