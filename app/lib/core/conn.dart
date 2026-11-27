import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:web_socket_channel/io.dart';

import 'crypto.dart';
import 'target.dart';

enum ZrConnState {
  idle,

  /// socket opening
  connecting,

  /// socket open, room joined — waiting for the computer's welcome
  linking,

  /// welcome received, commands flow
  paired,

  /// dropped mid-session, retrying
  reconnecting,

  /// the computer ended the session
  closed,

  /// unrecoverable without user action
  error,
}

/// Why a connection failed, in terms the UI can explain.
enum ZrConnError {
  /// the relay has no such room: the computer is off, asleep, or its agent quit
  noSuchRoom,

  /// four phones are already on this computer
  roomFull,

  /// the relay or the LAN address could not be reached at all
  unreachable,

  /// the LAN agent is not on this network (typical when you leave the house)
  lanUnreachable,
}

/// The transport. One socket, two shapes:
///
///   relay — dial `wss://remote.zlef.fr/ws`, `{t:'join',room}`, then frames
///   LAN   — dial the agent's own `wss://<ip>:<port>/ws` directly
///
/// Both carry `{t:'data',payload:<sealed>}`. Everything above this layer speaks
/// plain Dart maps; sealing and opening happen here and nowhere else.
///
/// LAN note: the agent serves a certificate it signed itself (a browser cannot
/// use WebCrypto over plain http, so the agent mints one for its own addresses).
/// We accept that certificate for private addresses only. It buys the transport,
/// not the trust: every frame inside is still sealed with the link key, so a
/// forged certificate yields nothing but ciphertext.
class ZrConn {
  ZrConn({required this.target, required this.crypto});

  final ZrTarget target;
  final ZrCrypto crypto;

  IOWebSocketChannel? _channel;
  StreamSubscription<dynamic>? _sub;
  Timer? _keepalive;
  Timer? _retry;
  bool _disposed = false;
  int _attempt = 0;

  ZrConnState _state = ZrConnState.idle;
  ZrConnState get state => _state;

  final _states = StreamController<ZrConnState>.broadcast();
  final _commands = StreamController<Map<String, dynamic>>.broadcast();
  final _errors = StreamController<ZrConnError>.broadcast();

  Stream<ZrConnState> get states => _states.stream;

  /// Decrypted host→phone frames (welcome, screen chunks, clipboard, pong…).
  Stream<Map<String, dynamic>> get commands => _commands.stream;
  Stream<ZrConnError> get errors => _errors.stream;

  void _setState(ZrConnState s) {
    if (_disposed || _state == s) return;
    _state = s;
    _states.add(s);
  }

  /// Marks the link live. Called by the session once the welcome lands, so
  /// "paired" always means "this computer answered", never just "socket open".
  void markPaired() => _setState(ZrConnState.paired);

  Future<void> connect() async {
    if (_disposed) return;
    _retry?.cancel();
    _setState(_attempt == 0 ? ZrConnState.connecting : ZrConnState.reconnecting);

    try {
      final client = HttpClient()
        ..connectionTimeout = const Duration(seconds: 8);
      final lanHost = target.lanHost;
      if (lanHost != null) {
        client.badCertificateCallback =
            (cert, host, port) => _isPrivateHost(host);
      }
      final socket = await WebSocket.connect(
        target.wsUrl,
        customClient: client,
      ).timeout(const Duration(seconds: 12));
      if (_disposed) {
        await socket.close();
        return;
      }
      final channel = IOWebSocketChannel(socket);
      _channel = channel;
      _sub = channel.stream.listen(
        _onMessage,
        onDone: _onDone,
        onError: (_) => _onDone(),
        cancelOnError: false,
      );

      if (target.transport == ZrTransport.relay) {
        _raw({'t': 'join', 'room': target.room});
      } else {
        _afterLink(); // the LAN agent is already listening on the other end
      }
      _startKeepalive();
    } catch (_) {
      _channel = null;
      _fail(target.transport == ZrTransport.lan
          ? ZrConnError.lanUnreachable
          : ZrConnError.unreachable);
    }
  }

  void _afterLink() {
    _setState(ZrConnState.linking);
    // the handshake IS the proof we hold the key: a wrong key can't be opened,
    // so the computer simply never answers.
    send({'t': 'hello', 'v': 1, 'ua': 'zlefremote-android'});
  }

  void _onMessage(dynamic raw) {
    Map<String, dynamic> msg;
    try {
      final decoded = jsonDecode(raw is String ? raw : utf8.decode(raw));
      if (decoded is! Map<String, dynamic>) return;
      msg = decoded;
    } catch (_) {
      return;
    }
    switch (msg['t']) {
      case 'joined':
        _attempt = 0;
        _afterLink();
      case 'data':
        _openFrame(msg['payload']);
      case 'closed':
        _setState(ZrConnState.closed);
        _teardown();
      case 'error':
        _fail(switch (msg['error']) {
          'no_such_room' => ZrConnError.noSuchRoom,
          'room_full' => ZrConnError.roomFull,
          _ => ZrConnError.unreachable,
        });
      case 'pong':
        break;
    }
  }

  Future<void> _openFrame(dynamic payload) async {
    if (payload is! String) return;
    try {
      final cmd = await crypto.open(payload);
      if (!_disposed) _commands.add(cmd);
    } catch (_) {
      // wrong key or tampered frame — drop it, exactly like the agent does
    }
  }

  /// Seals and sends one command. No-op while the socket is down, so callers
  /// never have to guard: a lost mouse move is better than a queue that
  /// replays your last gesture minutes later.
  Future<void> send(Map<String, dynamic> command) async {
    final channel = _channel;
    if (channel == null) return;
    try {
      _raw({'t': 'data', 'payload': await crypto.seal(command)});
    } catch (_) {}
  }

  void _raw(Map<String, dynamic> envelope) {
    try {
      _channel?.sink.add(jsonEncode(envelope));
    } catch (_) {}
  }

  void _startKeepalive() {
    _keepalive?.cancel();
    _keepalive = Timer.periodic(const Duration(seconds: 25), (_) {
      _raw({'t': 'ping'});
    });
  }

  void _onDone() {
    if (_disposed || _state == ZrConnState.closed) return;
    _keepalive?.cancel();
    _channel = null;
    if (_state == ZrConnState.paired ||
        _state == ZrConnState.linking ||
        _state == ZrConnState.reconnecting) {
      _scheduleRetry();
    } else {
      _fail(ZrConnError.unreachable);
    }
  }

  void _scheduleRetry() {
    _setState(ZrConnState.reconnecting);
    _attempt++;
    // back off, but stay responsive: a phone that just walked back into Wi-Fi
    // should reconnect in about a second, not after a minute of doubling.
    final delayMs = [700, 1200, 2000, 3500, 5000][_attempt.clamp(1, 5) - 1];
    _retry?.cancel();
    _retry = Timer(Duration(milliseconds: delayMs), connect);
  }

  void _fail(ZrConnError e) {
    _setState(ZrConnState.error);
    if (!_disposed) _errors.add(e);
    _teardown();
  }

  void _teardown() {
    _keepalive?.cancel();
    _retry?.cancel();
    _sub?.cancel();
    _sub = null;
    try {
      _channel?.sink.close();
    } catch (_) {}
    _channel = null;
  }

  /// Retry after a failure the user acknowledged.
  void retry() {
    _attempt = 0;
    connect();
  }

  Future<void> dispose() async {
    _disposed = true;
    _teardown();
    await _states.close();
    await _commands.close();
    await _errors.close();
  }

  /// RFC1918 / link-local / loopback / mDNS — the only places an agent can be
  /// serving its own certificate.
  static bool _isPrivateHost(String host) {
    if (host == 'localhost' || host.endsWith('.local')) return true;
    final ip = InternetAddress.tryParse(host);
    if (ip == null) return false;
    if (ip.isLoopback || ip.isLinkLocal) return true;
    if (ip.type == InternetAddressType.IPv4) {
      final b = ip.rawAddress;
      return b[0] == 10 ||
          (b[0] == 172 && b[1] >= 16 && b[1] <= 31) ||
          (b[0] == 192 && b[1] == 168);
    }
    // IPv6 unique-local fc00::/7
    return ip.rawAddress[0] & 0xfe == 0xfc;
  }
}
