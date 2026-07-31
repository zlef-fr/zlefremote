import 'dart:async';
import 'dart:ui' as ui;

import 'package:flutter/foundation.dart';

import 'caps.dart';
import 'conn.dart';
import 'crypto.dart';
import 'devices.dart';

/// Quality of the live screen stream. The agent clamps these, so the presets
/// are advice, not a contract.
enum ZrViewQuality { low, balanced, sharp }

extension ZrViewQualityParams on ZrViewQuality {
  Map<String, dynamic> get params => switch (this) {
        ZrViewQuality.low => {'fps': 5, 'q': 40, 'scale': 40},
        ZrViewQuality.balanced => {'fps': 8, 'q': 55, 'scale': 65},
        ZrViewQuality.sharp => {'fps': 12, 'q': 72, 'scale': 100},
      };
  String get id => name;
  static ZrViewQuality parse(String? s) => ZrViewQuality.values
      .firstWhere((q) => q.name == s, orElse: () => ZrViewQuality.balanced);
}

/// State of an agent self-update the phone asked for.
enum ZrAgentUpdate { idle, running, done, alreadyCurrent, failed }

/// Why the live view has nothing to show.
enum ZrViewProblem { none, unsupported, captureFailed }

/// One live link to one computer: state, capabilities, and every command the
/// UI can send. Owns the transport; the widgets own none of it.
class ZrSession extends ChangeNotifier {
  ZrSession({required this.device, required this.onDeviceLearned});

  final ZrDevice device;

  /// Called once the computer identifies itself, so the store can remember its
  /// real name and OS instead of the placeholder from the pairing screen.
  final void Function(ZrDevice updated) onDeviceLearned;

  ZrConn? _conn;
  StreamSubscription<ZrConnState>? _stateSub;
  StreamSubscription<Map<String, dynamic>>? _cmdSub;
  StreamSubscription<ZrConnError>? _errSub;

  ZrConnState state = ZrConnState.idle;
  ZrConnError? error;
  String hostName = '';
  String hostOs = '';
  ZrCapabilities caps = ZrCapabilities.empty;

  /// Round-trip time to the computer, null until the first pong.
  Duration? latency;
  int _pingId = 0;
  final _pingSentAt = <int, DateTime>{};
  Timer? _pingTimer;

  /// Text the computer put on its clipboard while we were watching.
  String? hostClipboard;

  /// Progress of an agent update triggered from here.
  ZrAgentUpdate agentUpdate = ZrAgentUpdate.idle;
  String? agentUpdateVersion;
  String? agentUpdateError;

  // ── live view ──────────────────────────────────────────────────────────────
  bool viewing = false;
  int display = 0;
  ZrViewQuality quality = ZrViewQuality.balanced;
  ZrViewProblem viewProblem = ZrViewProblem.none;
  ui.Image? frame;
  int fps = 0;

  _FrameAssembly? _asm;
  bool _decoding = false;
  Uint8List? _pendingFrame;
  int _fpsCount = 0;
  DateTime _fpsSince = DateTime.now();

  bool get isPaired => state == ZrConnState.paired;

  Future<void> start() async {
    final target = await device.target();
    final crypto = await ZrCrypto.fromKeyB64(device.keyB64);
    final conn = ZrConn(target: target, crypto: crypto);
    _conn = conn;
    _stateSub = conn.states.listen((s) {
      state = s;
      if (s != ZrConnState.paired) latency = null;
      notifyListeners();
    });
    _errSub = conn.errors.listen((e) {
      error = e;
      notifyListeners();
    });
    _cmdSub = conn.commands.listen(_onCommand);
    await conn.connect();
  }

  void retry() {
    error = null;
    notifyListeners();
    _conn?.retry();
  }

  // ── inbound ────────────────────────────────────────────────────────────────

  void _onCommand(Map<String, dynamic> c) {
    switch (c['t']) {
      case 'welcome':
        _onWelcome(c);
      case 'f':
        _onFrameChunk(c);
      case 'viewerr':
        viewProblem = c['reason'] == 'unsupported'
            ? ZrViewProblem.unsupported
            : ZrViewProblem.captureFailed;
        notifyListeners();
      case 'brightend':
        caps = caps.withBrightnessBackend(c);
        notifyListeners();
      case 'clip':
        hostClipboard = c['s'] as String?;
        notifyListeners();
      case 'updating':
        agentUpdate = ZrAgentUpdate.running;
        notifyListeners();
      case 'updated':
        agentUpdate = c['current'] == true
            ? ZrAgentUpdate.alreadyCurrent
            : ZrAgentUpdate.done;
        agentUpdateVersion = c['v'] as String?;
        notifyListeners();
      case 'updateerr':
        agentUpdate = ZrAgentUpdate.failed;
        agentUpdateError = c['reason'] as String?;
        notifyListeners();
      case 'pong':
        final sent = _pingSentAt.remove((c['i'] as num?)?.toInt() ?? -1);
        if (sent != null) {
          latency = DateTime.now().difference(sent);
          notifyListeners();
        }
    }
  }

  void _onWelcome(Map<String, dynamic> w) {
    hostName = (w['name'] as String?) ?? '';
    hostOs = (w['os'] as String?) ?? '';
    caps = ZrCapabilities.fromWelcome(w);
    if (display >= caps.screens.length) display = 0;
    _conn?.markPaired();
    onDeviceLearned(device.copyWith(name: hostName, os: hostOs));
    _startPinging();
    // the computer's clipboard is only worth watching if it can read one
    if (caps.ready(ZrCap.clipboard)) send({'t': 'clipwatch', 'on': true});
    if (viewing) _sendView();
    notifyListeners();
  }

  void _startPinging() {
    _pingTimer?.cancel();
    _ping();
    _pingTimer = Timer.periodic(const Duration(seconds: 5), (_) => _ping());
  }

  void _ping() {
    if (!isPaired) return;
    final id = ++_pingId;
    _pingSentAt[id] = DateTime.now();
    // a pong that never comes must not leak; anything older than a few seconds
    // is a lost round trip, not a slow one.
    _pingSentAt.removeWhere(
        (_, at) => DateTime.now().difference(at) > const Duration(seconds: 15));
    send({'t': 'ping', 'i': id});
  }

  // ── screen frames ──────────────────────────────────────────────────────────

  void _onFrameChunk(Map<String, dynamic> c) {
    if (!viewing) return;
    final id = (c['i'] as num?)?.toInt() ?? 0;
    final total = (c['n'] as num?)?.toInt() ?? 1;
    final index = (c['s'] as num?)?.toInt() ?? 0;
    if (_asm == null || _asm!.id != id) {
      _asm = _FrameAssembly(id: id, total: total);
    }
    final asm = _asm!;
    if (index < 0 || index >= asm.total || asm.parts[index] != null) return;
    try {
      asm.parts[index] = b64uDecode(c['d'] as String);
    } catch (_) {
      return;
    }
    asm.got++;
    if (asm.got < asm.total) return;
    _asm = null;
    viewProblem = ZrViewProblem.none;
    _present(asm.join());
  }

  /// Decode is the expensive step. If one is already running we keep only the
  /// newest frame — showing a stale screen is worse than skipping it.
  Future<void> _present(Uint8List jpeg) async {
    if (_decoding) {
      _pendingFrame = jpeg;
      return;
    }
    _decoding = true;
    try {
      final codec = await ui.instantiateImageCodec(jpeg);
      final decoded = await codec.getNextFrame();
      frame?.dispose();
      frame = decoded.image;
      codec.dispose();
      _tickFps();
      notifyListeners();
    } catch (_) {
    } finally {
      _decoding = false;
      final next = _pendingFrame;
      _pendingFrame = null;
      if (next != null && viewing) unawaited(_present(next));
    }
  }

  void _tickFps() {
    _fpsCount++;
    final elapsed = DateTime.now().difference(_fpsSince);
    if (elapsed.inMilliseconds >= 1000) {
      fps = (_fpsCount * 1000 / elapsed.inMilliseconds).round();
      _fpsCount = 0;
      _fpsSince = DateTime.now();
    }
  }

  // ── outbound ───────────────────────────────────────────────────────────────

  void send(Map<String, dynamic> command) {
    if (state == ZrConnState.paired || state == ZrConnState.linking) {
      unawaited(_conn?.send(command) ?? Future.value());
    }
  }

  void moveBy(double dx, double dy) =>
      send({'t': 'mv', 'dx': dx.round(), 'dy': dy.round()});

  void moveAbs(double nx, double ny) => send({'t': 'mvabs', 'nx': nx, 'ny': ny});

  void click(String button, {bool double = false}) =>
      send({'t': 'click', 'b': button, 'double': double});

  void clickAbs(double nx, double ny,
          {String button = 'left', bool double = false}) =>
      send({
        't': 'clickabs',
        'nx': nx,
        'ny': ny,
        'b': button,
        'double': double,
      });

  void buttonDown(String button) => send({'t': 'down', 'b': button});
  void buttonUp(String button) => send({'t': 'up', 'b': button});

  void scroll(double dx, double dy) =>
      send({'t': 'scroll', 'dx': dx.round(), 'dy': dy.round()});

  void key(String name, {List<String> mods = const []}) =>
      send({'t': 'key', 'k': name, 'mods': mods});

  void type(String text) => send({'t': 'text', 's': text});

  void media(String key) => send({'t': 'media', 'k': key});

  void setBrightness(int percent, {int screen = -1}) =>
      send({'t': 'bright', 'v': percent, 'bd': screen});

  void setBrightnessBackend(String id) => send({'t': 'brightend', 'be': id});

  /// Push text from this phone onto the computer's clipboard.
  void pushClipboard(String text) => send({'t': 'clip', 's': text});

  /// Ask for the computer's clipboard right now.
  void pullClipboard() => send({'t': 'clipget'});

  /// Ask the agent to update itself. The phone is the surface that knows the
  /// agent is old — it is the one comparing capabilities — so it is the right
  /// place to fix it from.
  void updateAgent() {
    agentUpdate = ZrAgentUpdate.running;
    agentUpdateError = null;
    notifyListeners();
    send({'t': 'update'});
  }

  // ── live view control ──────────────────────────────────────────────────────

  void startView() {
    if (viewing) return;
    viewing = true;
    viewProblem = ZrViewProblem.none;
    _sendView();
    notifyListeners();
  }

  void stopView() {
    if (!viewing) return;
    viewing = false;
    send({'t': 'view', 'on': false});
    frame?.dispose();
    frame = null;
    fps = 0;
    _asm = null;
    _pendingFrame = null;
    notifyListeners();
  }

  void setQuality(ZrViewQuality q) {
    quality = q;
    if (viewing) _sendView(); // the agent retunes a running stream in place
    notifyListeners();
  }

  void setDisplay(int index) {
    if (index == display) return;
    display = index;
    _asm = null; // half-assembled frames belong to the previous monitor
    _pendingFrame = null;
    if (viewing) _sendView();
    notifyListeners();
  }

  void _sendView() =>
      send({'t': 'view', 'on': true, 'd': display, ...quality.params});

  @override
  void dispose() {
    _pingTimer?.cancel();
    if (viewing) send({'t': 'view', 'on': false});
    if (caps.ready(ZrCap.clipboard)) send({'t': 'clipwatch', 'on': false});
    _stateSub?.cancel();
    _cmdSub?.cancel();
    _errSub?.cancel();
    unawaited(_conn?.dispose() ?? Future.value());
    frame?.dispose();
    frame = null;
    super.dispose();
  }
}

class _FrameAssembly {
  _FrameAssembly({required this.id, required this.total})
      : parts = List<Uint8List?>.filled(total, null);
  final int id;
  final int total;
  final List<Uint8List?> parts;
  int got = 0;

  Uint8List join() {
    final size = parts.fold<int>(0, (n, p) => n + (p?.length ?? 0));
    final out = Uint8List(size);
    var offset = 0;
    for (final p in parts) {
      if (p == null) continue;
      out.setAll(offset, p);
      offset += p.length;
    }
    return out;
  }
}
