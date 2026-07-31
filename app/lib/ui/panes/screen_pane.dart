import 'dart:math' as math;
import 'dart:ui' as ui;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../../core/caps.dart';
import '../../core/i18n/i18n.dart';
import '../../core/session.dart';
import '../../core/settings.dart';
import '../theme.dart';
import '../widgets.dart';

/// The computer's screen, live, as a touchscreen.
///
/// New here versus the web client: **pinch to zoom and pan**. A 4K desktop
/// letterboxed into a phone is unreadable at 1×, and the old client had no way
/// to get closer — so taps were guesses. Zoom is local; only the resulting
/// absolute pointer position crosses the wire.
class ScreenPane extends StatefulWidget {
  const ScreenPane({super.key});

  @override
  State<ScreenPane> createState() => _ScreenPaneState();
}

class _ScreenPaneState extends State<ScreenPane> {
  static const _maxZoom = 5.0;
  static const _tapWindow = Duration(milliseconds: 260);
  static const _tapSlop = 12.0;

  final _pointers = <int, Offset>{};
  Offset? _lastSingle;
  double _travelled = 0;
  DateTime _touchStart = DateTime.now();
  DateTime? _lastTapAt;
  Offset? _lastTapAt2;
  bool _multi = false;
  DateTime _lastMoveSent = DateTime.fromMillisecondsSinceEpoch(0);

  double _zoom = 1;
  Offset _pan = Offset.zero;
  double? _pinchStartDistance;
  double _pinchStartZoom = 1;
  Offset? _pinchStartMid;
  Offset _panAtPinchStart = Offset.zero;

  Size _stage = Size.zero;

  ZrSession get _session => context.read<ZrSession>();

  void _haptic([bool strong = false]) {
    if (!context.read<ZrSettings>().haptics) return;
    strong ? HapticFeedback.mediumImpact() : HapticFeedback.selectionClick();
  }

  // ── geometry ───────────────────────────────────────────────────────────────

  Rect _imageRect(ui.Image image) {
    if (_stage.isEmpty) return Rect.zero;
    final base = math.min(
      _stage.width / image.width,
      _stage.height / image.height,
    );
    final w = image.width * base * _zoom;
    final h = image.height * base * _zoom;
    final center = Offset(_stage.width / 2, _stage.height / 2) + _pan;
    return Rect.fromCenter(center: center, width: w, height: h);
  }

  /// Keeps the picture from being flung off screen: at 1× it stays centred, and
  /// zoomed in it can never expose more than the stage on any side.
  void _clampPan(ui.Image image) {
    final rect = _imageRect(image);
    if (rect.width <= _stage.width) {
      _pan = Offset(0, _pan.dy);
    } else {
      final slack = (rect.width - _stage.width) / 2;
      _pan = Offset(_pan.dx.clamp(-slack, slack), _pan.dy);
    }
    if (rect.height <= _stage.height) {
      _pan = Offset(_pan.dx, 0);
    } else {
      final slack = (rect.height - _stage.height) / 2;
      _pan = Offset(_pan.dx, _pan.dy.clamp(-slack, slack));
    }
  }

  Offset? _normalize(Offset local, ui.Image? image) {
    if (image == null) return null;
    final rect = _imageRect(image);
    if (rect.width == 0 || rect.height == 0) return null;
    return Offset(
      ((local.dx - rect.left) / rect.width).clamp(0.0, 1.0),
      ((local.dy - rect.top) / rect.height).clamp(0.0, 1.0),
    );
  }

  // ── touch ──────────────────────────────────────────────────────────────────

  void _onDown(PointerDownEvent e) {
    _pointers[e.pointer] = e.localPosition;
    if (_pointers.length == 1) {
      _lastSingle = e.localPosition;
      _travelled = 0;
      _touchStart = DateTime.now();
      _multi = false;
    } else if (_pointers.length == 2) {
      _multi = true;
      final points = _pointers.values.toList();
      _pinchStartDistance = (points[0] - points[1]).distance;
      _pinchStartZoom = _zoom;
      _pinchStartMid = (points[0] + points[1]) / 2;
      _panAtPinchStart = _pan;
    }
  }

  void _onMove(PointerMoveEvent e, ui.Image? image) {
    if (!_pointers.containsKey(e.pointer)) return;
    _pointers[e.pointer] = e.localPosition;

    if (_pointers.length >= 2) {
      final points = _pointers.values.toList();
      final distance = (points[0] - points[1]).distance;
      final mid = (points[0] + points[1]) / 2;
      final start = _pinchStartDistance;
      if (start != null && start > 0 && image != null) {
        setState(() {
          _zoom = (_pinchStartZoom * (distance / start)).clamp(1.0, _maxZoom);
          _pan = _panAtPinchStart + (mid - (_pinchStartMid ?? mid));
          _clampPan(image);
        });
      }
      return;
    }

    if (_pointers.length == 1 && _lastSingle != null) {
      _travelled += (e.localPosition - _lastSingle!).distance;
      _lastSingle = e.localPosition;
      // ~30 Hz is plenty for a cursor the eye is tracking on a 14 fps picture
      final now = DateTime.now();
      if (now.difference(_lastMoveSent).inMilliseconds >= 33) {
        _lastMoveSent = now;
        final n = _normalize(e.localPosition, image);
        if (n != null) _session.moveAbs(n.dx, n.dy);
      }
    }
  }

  void _onUp(PointerUpEvent e, ui.Image? image) {
    final wasCount = _pointers.length;
    _pointers.remove(e.pointer);
    final quick = DateTime.now().difference(_touchStart) < _tapWindow &&
        _travelled < _tapSlop;

    if (wasCount == 2 && quick && _lastSingle != null) {
      final n = _normalize(_lastSingle!, image);
      if (n != null) {
        _session.clickAbs(n.dx, n.dy, button: 'right');
        _haptic(true);
      }
    } else if (wasCount == 1 && !_multi && quick && _lastSingle != null) {
      final n = _normalize(_lastSingle!, image);
      if (n != null) {
        final now = DateTime.now();
        final isDouble = _lastTapAt != null &&
            _lastTapAt2 != null &&
            now.difference(_lastTapAt!) < const Duration(milliseconds: 320) &&
            (_lastSingle! - _lastTapAt2!).distance < 24;
        _session.clickAbs(n.dx, n.dy, double: isDouble);
        _haptic(isDouble);
        _lastTapAt = isDouble ? null : now;
        _lastTapAt2 = isDouble ? null : _lastSingle;
      }
    }

    if (_pointers.isEmpty) {
      _multi = false;
      _pinchStartDistance = null;
    }
  }

  // ── build ──────────────────────────────────────────────────────────────────

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final session = context.watch<ZrSession>();
    final state = session.caps[ZrCap.screen];

    if (state != ZrCapState.ready) {
      // The tab is still here, and so is the explanation. Hiding it would only
      // teach the user that the app is inconsistent.
      return SingleChildScrollView(
        padding: const EdgeInsets.all(Z.s4),
        child: ZReveal(
          child: ZCapabilityNotice(cap: ZrCap.screen, state: state),
        ),
      );
    }

    return Padding(
      padding: const EdgeInsets.fromLTRB(Z.s3, 0, Z.s3, Z.s3),
      child: Column(
        children: [
          Expanded(child: _stageWidget(l, session)),
          const SizedBox(height: Z.s2),
          _controls(l, session),
        ],
      ),
    );
  }

  Widget _stageWidget(L10n l, ZrSession session) {
    final image = session.frame;
    return LayoutBuilder(
      builder: (context, constraints) {
        _stage = Size(constraints.maxWidth, constraints.maxHeight);
        return ClipRRect(
          borderRadius: BorderRadius.circular(Z.rLg),
          child: Container(
            color: Colors.black,
            child: Stack(
              fit: StackFit.expand,
              children: [
                if (image != null)
                  Listener(
                    onPointerDown: _onDown,
                    onPointerMove: (e) => _onMove(e, image),
                    onPointerUp: (e) => _onUp(e, image),
                    onPointerCancel: (e) {
                      _pointers.remove(e.pointer);
                      if (_pointers.isEmpty) _multi = false;
                    },
                    behavior: HitTestBehavior.opaque,
                    child: CustomPaint(
                      painter: _ScreenPainter(
                        image: image,
                        rect: _imageRect(image),
                      ),
                    ),
                  ),
                if (image == null)
                  Center(
                    child: session.viewing
                        ? Column(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              const SizedBox(
                                width: 26,
                                height: 26,
                                child: CircularProgressIndicator(
                                  strokeWidth: 2,
                                  valueColor:
                                      AlwaysStoppedAnimation(Z.oliveSoft),
                                ),
                              ),
                              const SizedBox(height: Z.s3),
                              Text(
                                session.viewProblem == ZrViewProblem.captureFailed
                                    ? l.t('screen_failed')
                                    : l.t('screen_waiting'),
                                style: Z.bodySoft,
                                textAlign: TextAlign.center,
                              ),
                            ],
                          )
                        : ZButton(
                            label: l.t('screen_start'),
                            icon: Icons.play_arrow_rounded,
                            size: ZButtonSize.lg,
                            onPressed: session.startView,
                          ),
                  ),
                if (image != null)
                  Positioned(
                    left: Z.s2,
                    top: Z.s2,
                    child: _Hud(
                      text: '${session.fps} fps'
                          '${_zoom > 1.02 ? ' · ${_zoom.toStringAsFixed(1)}×' : ''}',
                    ),
                  ),
                if (image != null && _zoom > 1.02)
                  Positioned(
                    right: Z.s2,
                    top: Z.s2,
                    child: ZButton(
                      label: l.t('zoom_reset'),
                      kind: ZButtonKind.ghost,
                      size: ZButtonSize.sm,
                      icon: Icons.fit_screen_rounded,
                      onPressed: () => setState(() {
                        _zoom = 1;
                        _pan = Offset.zero;
                      }),
                    ),
                  ),
                if (image != null)
                  Positioned(
                    left: 0,
                    right: 0,
                    bottom: 0,
                    child: IgnorePointer(
                      child: Container(
                        padding: const EdgeInsets.symmetric(
                            horizontal: Z.s3, vertical: 6),
                        color: Z.bg.withValues(alpha: .55),
                        child: Text(
                          l.t('screen_hint'),
                          textAlign: TextAlign.center,
                          style: Z.mono.copyWith(fontSize: 11, color: Z.inkMuted),
                        ),
                      ),
                    ),
                  ),
              ],
            ),
          ),
        );
      },
    );
  }

  Widget _controls(L10n l, ZrSession session) {
    final settings = context.read<ZrSettings>();
    final multiMonitor = session.caps[ZrCap.multiMonitor];
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SingleChildScrollView(
          scrollDirection: Axis.horizontal,
          child: Row(
            children: [
              for (final quality in ZrViewQuality.values) ...[
                ZChip(
                  label: switch (quality) {
                    ZrViewQuality.low => l.t('q_low'),
                    ZrViewQuality.balanced => l.t('q_balanced'),
                    ZrViewQuality.sharp => l.t('q_sharp'),
                  },
                  selected: session.quality == quality,
                  onTap: () {
                    session.setQuality(quality);
                    settings.viewQuality = quality;
                  },
                ),
                const SizedBox(width: Z.s2),
              ],
              if (session.viewing) ...[
                const SizedBox(width: Z.s2),
                ZButton(
                  label: l.t('screen_stop'),
                  kind: ZButtonKind.ghost,
                  size: ZButtonSize.sm,
                  icon: Icons.stop_rounded,
                  onPressed: session.stopView,
                ),
              ],
            ],
          ),
        ),
        if (multiMonitor == ZrCapState.ready) ...[
          const SizedBox(height: Z.s2),
          SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            child: Row(
              children: [
                for (var i = 0; i < session.caps.screens.length; i++) ...[
                  ZChip(
                    icon: Icons.monitor_rounded,
                    label: '${l.t('display')} ${i + 1}',
                    sub: session.caps.screens[i].label,
                    selected: session.display == i,
                    onTap: () {
                      setState(() {
                        _zoom = 1;
                        _pan = Offset.zero;
                      });
                      session.setDisplay(i);
                    },
                  ),
                  const SizedBox(width: Z.s2),
                ],
              ],
            ),
          ),
        ],
      ],
    );
  }
}

class _Hud extends StatelessWidget {
  const _Hud({required this.text});
  final String text;

  @override
  Widget build(BuildContext context) => Container(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
        decoration: BoxDecoration(
          color: Z.bg.withValues(alpha: .7),
          borderRadius: BorderRadius.circular(Z.rSm),
          border: Border.all(color: Z.line),
        ),
        child: Text(text, style: Z.mono.copyWith(fontSize: 11.5)),
      );
}

class _ScreenPainter extends CustomPainter {
  _ScreenPainter({required this.image, required this.rect});

  final ui.Image image;
  final Rect rect;

  @override
  void paint(Canvas canvas, Size size) {
    canvas.drawRect(Offset.zero & size, Paint()..color = Colors.black);
    canvas.drawImageRect(
      image,
      Rect.fromLTWH(0, 0, image.width.toDouble(), image.height.toDouble()),
      rect,
      Paint()..filterQuality = FilterQuality.medium,
    );
  }

  @override
  bool shouldRepaint(_ScreenPainter old) =>
      old.image != image || old.rect != rect;
}
