import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../../core/gestures.dart';
import '../../core/i18n/i18n.dart';
import '../../core/session.dart';
import '../../core/settings.dart';
import '../theme.dart';

/// The trackpad.
///
/// Gesture vocabulary is the one people already know from a laptop: drag moves,
/// tap clicks, tap-then-press drags, two fingers scroll, two-finger tap
/// right-clicks, a sideways two-finger flick goes back/forward, pinch zooms,
/// and three fingers switch app / open the overview / show the desktop, with a
/// three-finger tap for the middle button a phone has no room for. A dedicated
/// rail on the right scrolls with one finger, because two-finger scrolling on a
/// phone held one-handed is a stretch.
///
/// Every multi-finger verb travels as an INTENT (see core/gestures.dart) — the
/// computer owns the keyboard shortcut, because it is the one that knows
/// whether it is a Mac.
class TouchpadPane extends StatefulWidget {
  const TouchpadPane({super.key});

  @override
  State<TouchpadPane> createState() => _TouchpadPaneState();
}

class _TouchpadPaneState extends State<TouchpadPane>
    with TickerProviderStateMixin {
  static const _tapSlop = 10.0;
  static const _tapWindow = Duration(milliseconds: 220);
  static const _tapDragWindow = Duration(milliseconds: 300);

  /// A sideways two-finger flick is back/forward; a slower sideways drag is a
  /// horizontal scroll. Only time and distance separate them, so the sideways
  /// scroll is withheld for [_flickWindow] and then either flushed or spent.
  static const _flickWindow = Duration(milliseconds: 300);
  static const _flickDistance = 64.0;
  static const _flickVerticalGiveUp = 26.0;

  /// Fingers that change their spread by this much are pinching, not scrolling;
  /// each further [_pinchStep] of spread is one zoom notch.
  static const _pinchTrigger = 34.0;
  static const _pinchStep = 56.0;
  static const _pinchStepsPerFrame = 3;

  /// A three-finger swipe fires once it has travelled this far.
  static const _swipeThreshold = 46.0;

  final _pointers = <int, Offset>{};
  Offset? _last;
  double _travelled = 0;
  DateTime _gestureStart = DateTime.now();
  DateTime? _lastTapAt;
  bool _dragging = false;
  bool _twoFinger = false;

  /// Travel of the two-finger gesture itself — [_travelled] only grows under a
  /// single finger, so without this a fast scroll inside the tap window reads
  /// as a two-finger tap and right-clicks.
  double _twoTravel = 0;
  DateTime _twoFingerStart = DateTime.now();
  Offset? _twoFingerStartMid;
  double? _twoStartSpread;

  /// Sideways travel held back from the scroll while the flick is still
  /// possible, and whether it still is.
  double _flickDx = 0;
  bool _flickOpen = false;
  bool _flickFired = false;

  /// Spread at which the last zoom notch fired; null until a pinch is declared.
  double? _pinchAnchor;

  /// A three-finger gesture is a swipe, not a pointer move — once it starts,
  /// nothing else on the pad may act on those fingers.
  bool _threeFinger = false;
  bool _threeFingerFired = false;
  Offset? _threeFingerStart;
  Offset? _lastMidpoint;
  Offset _scrollRemainder = Offset.zero;
  bool _dragLock = false;
  bool _hintShown = true;

  // touch feedback
  Offset? _glow;
  bool _glowGrabbing = false;
  final _ripples = <_Ripple>[];

  ZrSession get _session => context.read<ZrSession>();
  ZrSettings get _settings => context.read<ZrSettings>();

  void _haptic([int intensity = 0]) {
    if (!_settings.haptics) return;
    switch (intensity) {
      case 0:
        HapticFeedback.selectionClick();
      case 1:
        HapticFeedback.lightImpact();
      default:
        HapticFeedback.mediumImpact();
    }
  }

  @override
  void dispose() {
    for (final r in _ripples) {
      r.controller.dispose();
    }
    super.dispose();
  }

  // ── gestures ───────────────────────────────────────────────────────────────

  void _onDown(PointerDownEvent e) {
    _pointers[e.pointer] = e.localPosition;
    if (_pointers.length == 1) {
      _last = e.localPosition;
      _travelled = 0;
      _gestureStart = DateTime.now();
      _twoFinger = false;
      final recentTap = _lastTapAt != null &&
          DateTime.now().difference(_lastTapAt!) < _tapDragWindow;
      setState(() {
        _glow = e.localPosition;
        _glowGrabbing = recentTap || _dragging;
        if (_hintShown) _hintShown = false;
      });
      // tap, then press again → the computer starts a drag
      if (recentTap && !_dragging) {
        _dragging = true;
        _session.buttonDown('left');
        _haptic(1);
      }
    } else if (_pointers.length == 2) {
      _twoFinger = true;
      _lastMidpoint = _midpoint();
      _twoFingerStartMid = _lastMidpoint;
      _twoFingerStart = DateTime.now();
      _twoStartSpread = _spread();
      _twoTravel = 0;
      _flickDx = 0;
      _flickOpen = _settings.gestures;
      _flickFired = false;
      _pinchAnchor = null;
      setState(() => _glow = null);
    } else if (_pointers.length == 3) {
      _threeFinger = true;
      _threeFingerFired = false;
      _threeFingerStart = _centroid();
      _twoFinger = false;
      _lastMidpoint = null;
      setState(() => _glow = null);
    }
  }

  void _onMove(PointerMoveEvent e) {
    if (!_pointers.containsKey(e.pointer)) return;
    _pointers[e.pointer] = e.localPosition;

    if (_threeFinger && _pointers.length >= 3) {
      _trackThreeFinger();
      return;
    }

    if (_twoFinger && _pointers.length >= 2) {
      _trackTwoFinger();
      return;
    }

    if (_pointers.length == 1 && _last != null) {
      final delta = e.localPosition - _last!;
      _travelled += delta.distance;
      _last = e.localPosition;
      setState(() => _glow = e.localPosition);
      final dx = _accelerate(delta.dx);
      final dy = _accelerate(delta.dy);
      if (dx.abs() >= 1 || dy.abs() >= 1) _session.moveBy(dx, dy);
    }
  }

  void _onUp(PointerUpEvent e) {
    final wasCount = _pointers.length;
    _pointers.remove(e.pointer);
    final held = DateTime.now().difference(_gestureStart);
    final quick = held < _tapWindow && _travelled < _tapSlop;

    // a three-finger gesture owns the whole touch: lifting a finger ends it,
    // and none of the click/scroll paths below may claim the remainder.
    if (_threeFinger) {
      if (wasCount == 3 && !_threeFingerFired && quick) {
        // three-finger tap → middle click, the mouse button phones don't have
        _session.click('middle');
        _haptic(1);
      }
      if (_pointers.isEmpty) {
        _threeFinger = false;
        _threeFingerFired = false;
        _threeFingerStart = null;
        _endGesture();
      }
      return;
    }

    if (_dragging && _pointers.isEmpty) {
      _dragging = false;
      _session.buttonUp('left');
      _endGesture();
      return;
    }

    if (wasCount == 2) {
      // a sideways flick that never became a scroll is back/forward
      if (_flickOpen && !_flickFired && _flickDx.abs() >= _flickDistance) {
        _flickFired = true;
        _session
            .gesture(_flickDx > 0 ? ZrGesture.navBack : ZrGesture.navForward);
        _haptic(2);
      } else if (quick &&
          _twoTravel < _tapSlop &&
          _pinchAnchor == null &&
          !_flickFired) {
        _session.click('right');
        if (_last != null) _ripple(_last!, right: true);
        _haptic(1);
      }
      // _twoFinger stays set until every finger is up: the one still down must
      // not start moving the pointer, nor count as a one-finger tap.
      if (_pointers.isEmpty) _endGesture();
      return;
    }

    if (wasCount == 1 && _pointers.isEmpty && !_twoFinger && quick) {
      _session.click('left');
      if (_last != null) _ripple(_last!);
      _haptic();
      _lastTapAt = DateTime.now();
    }
    if (_pointers.isEmpty) _endGesture();
  }

  void _onCancel(PointerCancelEvent e) {
    _pointers.remove(e.pointer);
    if (_pointers.isEmpty) {
      if (_dragging) {
        _dragging = false;
        _session.buttonUp('left');
      }
      _endGesture();
    }
  }

  void _endGesture() {
    _twoFinger = false;
    _threeFinger = false;
    _threeFingerFired = false;
    _threeFingerStart = null;
    _lastMidpoint = null;
    _twoFingerStartMid = null;
    _twoStartSpread = null;
    _pinchAnchor = null;
    _flickOpen = false;
    _flickFired = false;
    _flickDx = 0;
    _twoTravel = 0;
    _scrollRemainder = Offset.zero;
    setState(() {
      _glow = null;
      _glowGrabbing = false;
    });
  }

  Offset _midpoint() {
    final points = _pointers.values.toList();
    return (points[0] + points[1]) / 2;
  }

  /// Distance between the first two fingers — the pinch signal.
  double _spread() {
    final points = _pointers.values.toList();
    if (points.length < 2) return 0;
    return (points[0] - points[1]).distance;
  }

  Offset _centroid() {
    final points = _pointers.values.toList();
    var sum = Offset.zero;
    for (final p in points) {
      sum += p;
    }
    return sum / points.length.toDouble();
  }

  /// Two fingers do three things, and they have to be told apart live:
  /// pinching zooms, a quick sideways flick is back/forward, everything else
  /// scrolls. Pinch wins as soon as the spread changes decisively; the sideways
  /// component of a scroll is withheld until the flick window closes, so a
  /// flick that turned into a drag still scrolls the distance it covered.
  void _trackTwoFinger() {
    final mid = _midpoint();
    final previous = _lastMidpoint;
    final spread = _spread();
    final startSpread = _twoStartSpread;

    if (_settings.gestures &&
        _pinchAnchor == null &&
        startSpread != null &&
        (spread - startSpread).abs() > _pinchTrigger) {
      _pinchAnchor =
          startSpread + (spread > startSpread ? _pinchTrigger : -_pinchTrigger);
      _flickOpen = false;
      _flickDx = 0;
      _haptic(1);
    }
    if (_pinchAnchor != null) {
      _pinchTick(spread);
      _lastMidpoint = mid;
      return;
    }

    if (previous != null) {
      final delta = mid - previous;
      _twoTravel += delta.distance;
      var dx = delta.dx;
      if (_flickOpen) {
        _flickDx += dx;
        final elapsed = DateTime.now().difference(_twoFingerStart);
        final vertical = ((mid - (_twoFingerStartMid ?? mid)).dy).abs();
        if (elapsed > _flickWindow || vertical > _flickVerticalGiveUp) {
          // not a flick after all — hand the withheld sideways scroll back
          dx = _flickDx;
          _flickOpen = false;
          _flickDx = 0;
        } else {
          dx = 0;
        }
      }
      final speed = _settings.scrollSpeed;
      final direction = _settings.naturalScroll ? 1 : -1;
      _scrollRemainder += Offset(dx * speed, delta.dy * speed * direction);
      final sx = _scrollRemainder.dx.truncateToDouble();
      final sy = _scrollRemainder.dy.truncateToDouble();
      if (sx != 0 || sy != 0) {
        _session.scroll(sx, sy);
        _scrollRemainder -= Offset(sx, sy);
      }
    }
    _lastMidpoint = mid;
  }

  /// One zoom notch per [_pinchStep] of spread, capped per frame so a fast
  /// pinch can't machine-gun a dozen Ctrl+= at the computer.
  void _pinchTick(double spread) {
    var fired = 0;
    while (fired < _pinchStepsPerFrame &&
        (spread - _pinchAnchor!).abs() >= _pinchStep) {
      final out = spread > _pinchAnchor!;
      _pinchAnchor = _pinchAnchor! + (out ? _pinchStep : -_pinchStep);
      _session.gesture(out ? ZrGesture.zoomIn : ZrGesture.zoomOut);
      fired++;
    }
    if (fired > 0) _haptic();
  }

  /// Three fingers = the desktop gestures a trackpad has and a phone doesn't:
  /// swipe left/right switches window, up opens the overview, down shows the
  /// desktop. One shot per gesture — a swipe is a verb, not a stream.
  void _trackThreeFinger() {
    if (_threeFingerFired || !_settings.gestures) return;
    final start = _threeFingerStart;
    if (start == null) return;
    final delta = _centroid() - start;
    if (delta.distance < _swipeThreshold) return;

    _threeFingerFired = true;
    _haptic(2);
    if (delta.dx.abs() > delta.dy.abs()) {
      _session.gesture(delta.dx > 0 ? ZrGesture.appNext : ZrGesture.appPrev);
    } else if (delta.dy < 0) {
      _session.gesture(ZrGesture.overview);
    } else {
      _session.gesture(ZrGesture.showDesktop);
    }
  }

  /// Small movements stay precise, big ones cover ground — the same curve the
  /// web client used, so muscle memory carries over.
  double _accelerate(double delta) {
    final speed = _settings.pointerSpeed;
    final magnitude = delta.abs();
    final boost = magnitude > 8
        ? 1.7
        : magnitude > 3
            ? 1.2
            : 1.0;
    return delta * speed * boost;
  }

  void _ripple(Offset at, {bool right = false}) {
    final controller = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 480),
    );
    final ripple = _Ripple(at, right, controller);
    setState(() => _ripples.add(ripple));
    controller.forward().whenComplete(() {
      if (!mounted) {
        controller.dispose();
        return;
      }
      setState(() => _ripples.remove(ripple));
      controller.dispose();
    });
  }

  // ── build ──────────────────────────────────────────────────────────────────

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    return Padding(
      padding: const EdgeInsets.fromLTRB(Z.s3, 0, Z.s3, Z.s3),
      child: Column(
        children: [
          Expanded(
            child: Row(
              children: [
                Expanded(child: _pad(l)),
                const SizedBox(width: Z.s2),
                _ScrollRail(
                  onScroll: (dy) {
                    final direction = _settings.naturalScroll ? 1 : -1;
                    final amount =
                        (dy * direction * _settings.scrollSpeed * 1.3).round();
                    if (amount != 0) _session.scroll(0, amount.toDouble());
                  },
                  label: l.t('scroll_rail'),
                ),
              ],
            ),
          ),
          const SizedBox(height: Z.s3),
          _buttons(l),
        ],
      ),
    );
  }

  Widget _pad(L10n l) => Listener(
        onPointerDown: _onDown,
        onPointerMove: _onMove,
        onPointerUp: _onUp,
        onPointerCancel: _onCancel,
        behavior: HitTestBehavior.opaque,
        child: Container(
          decoration: BoxDecoration(
            color: Z.surface1,
            borderRadius: BorderRadius.circular(Z.rLg),
            border: Border.all(color: Z.line),
          ),
          child: ClipRRect(
            borderRadius: BorderRadius.circular(Z.rLg),
            child: Stack(
              fit: StackFit.expand,
              children: [
                CustomPaint(
                  painter: _PadPainter(
                    glow: _glow,
                    grabbing: _glowGrabbing,
                    ripples: _ripples,
                  ),
                ),
                if (_hintShown)
                  Align(
                    alignment: Alignment.center,
                    child: Padding(
                      padding: const EdgeInsets.symmetric(horizontal: Z.s5),
                      child: Text(
                        l.t('pad_hint'),
                        textAlign: TextAlign.center,
                        style: Z.bodySoft
                            .copyWith(fontSize: 15, color: Z.inkFaint),
                      ),
                    ),
                  ),
                if (_dragLock)
                  Align(
                    alignment: Alignment.topCenter,
                    child: Padding(
                      padding: const EdgeInsets.all(Z.s3),
                      child: Container(
                        padding: const EdgeInsets.symmetric(
                            horizontal: Z.s3, vertical: 6),
                        decoration: BoxDecoration(
                          color: Z.oliveDeep,
                          borderRadius: BorderRadius.circular(Z.rSm),
                          border: Border.all(color: Z.oliveMid),
                        ),
                        child: Text(l.t('drag_lock_on'),
                            style: Z.label.copyWith(color: Z.oliveBright)),
                      ),
                    ),
                  ),
              ],
            ),
          ),
        ),
      );

  Widget _buttons(L10n l) => Row(
        children: [
          Expanded(
            child: _PadButton(
              label: l.t('btn_left'),
              onTap: () {
                _session.click('left');
                _haptic();
              },
            ),
          ),
          const SizedBox(width: Z.s2),
          Expanded(
            child: _PadButton(
              label: l.t('btn_mid'),
              onTap: () {
                _session.click('middle');
                _haptic();
              },
            ),
          ),
          const SizedBox(width: Z.s2),
          Expanded(
            child: _PadButton(
              label: l.t('btn_right'),
              onTap: () {
                _session.click('right');
                _haptic();
              },
            ),
          ),
          const SizedBox(width: Z.s2),
          Expanded(
            child: _PadButton(
              label: l.t('drag_lock'),
              icon: Icons.pan_tool_alt_outlined,
              active: _dragLock,
              onTap: () {
                setState(() => _dragLock = !_dragLock);
                if (_dragLock) {
                  _session.buttonDown('left');
                } else {
                  _session.buttonUp('left');
                }
                _haptic(2);
              },
            ),
          ),
        ],
      );
}

class _PadButton extends StatefulWidget {
  const _PadButton({
    required this.label,
    required this.onTap,
    this.icon,
    this.active = false,
  });

  final String label;
  final VoidCallback onTap;
  final IconData? icon;
  final bool active;

  @override
  State<_PadButton> createState() => _PadButtonState();
}

class _PadButtonState extends State<_PadButton> {
  bool _down = false;

  @override
  Widget build(BuildContext context) => GestureDetector(
        onTapDown: (_) => setState(() => _down = true),
        onTapUp: (_) => setState(() => _down = false),
        onTapCancel: () => setState(() => _down = false),
        onTap: widget.onTap,
        behavior: HitTestBehavior.opaque,
        child: AnimatedContainer(
          duration: Z.fast,
          curve: Z.ease,
          height: 54,
          decoration: BoxDecoration(
            color: widget.active
                ? Z.olive
                : (_down ? Z.surface3 : Z.surface2),
            borderRadius: BorderRadius.circular(Z.rMd),
            border: Border.all(color: widget.active ? Z.oliveMid : Z.line),
          ),
          alignment: Alignment.center,
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              if (widget.icon != null)
                Icon(widget.icon,
                    size: 18,
                    color: widget.active ? Z.oliveBright : Z.inkSoft),
              Text(
                widget.label,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: Z.label.copyWith(
                  color: widget.active ? Z.oliveBright : Z.inkSoft,
                ),
              ),
            ],
          ),
        ),
      );
}

/// One-finger scrolling. Narrow, full height, always where the thumb already is.
class _ScrollRail extends StatefulWidget {
  const _ScrollRail({required this.onScroll, required this.label});

  final void Function(double dy) onScroll;
  final String label;

  @override
  State<_ScrollRail> createState() => _ScrollRailState();
}

class _ScrollRailState extends State<_ScrollRail> {
  double? _last;
  bool _active = false;

  @override
  Widget build(BuildContext context) => Listener(
        onPointerDown: (e) {
          _last = e.localPosition.dy;
          setState(() => _active = true);
        },
        onPointerMove: (e) {
          final previous = _last;
          if (previous == null) return;
          widget.onScroll(e.localPosition.dy - previous);
          _last = e.localPosition.dy;
        },
        onPointerUp: (_) {
          _last = null;
          setState(() => _active = false);
        },
        onPointerCancel: (_) {
          _last = null;
          setState(() => _active = false);
        },
        behavior: HitTestBehavior.opaque,
        child: AnimatedContainer(
          duration: Z.fast,
          curve: Z.ease,
          width: 46,
          decoration: BoxDecoration(
            color: _active ? Z.surface3 : Z.surface1,
            borderRadius: BorderRadius.circular(Z.rLg),
            border: Border.all(color: _active ? Z.oliveMid : Z.line),
          ),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(Icons.keyboard_arrow_up_rounded,
                  color: _active ? Z.oliveSoft : Z.inkFaint),
              const SizedBox(height: Z.s2),
              RotatedBox(
                quarterTurns: 3,
                child: Text(
                  widget.label.toUpperCase(),
                  style: Z.eyebrow.copyWith(
                    fontSize: 11,
                    color: _active ? Z.oliveSoft : Z.inkFaint,
                  ),
                ),
              ),
              const SizedBox(height: Z.s2),
              Icon(Icons.keyboard_arrow_down_rounded,
                  color: _active ? Z.oliveSoft : Z.inkFaint),
            ],
          ),
        ),
      );
}

class _Ripple {
  _Ripple(this.at, this.right, this.controller);
  final Offset at;
  final bool right;
  final AnimationController controller;
}

/// Dot grid + a puck that follows the finger + click ripples. The surface has
/// to answer the touch instantly: the computer's cursor is far away, and this
/// is the only local proof the gesture registered.
class _PadPainter extends CustomPainter {
  _PadPainter({required this.glow, required this.grabbing, required this.ripples})
      : super(
          repaint: Listenable.merge(ripples.map((r) => r.controller).toList()),
        );

  final Offset? glow;
  final bool grabbing;
  final List<_Ripple> ripples;

  @override
  void paint(Canvas canvas, Size size) {
    const spacing = 26.0;
    final dot = Paint()..color = Z.lineStrong;
    for (var x = spacing / 2; x < size.width; x += spacing) {
      for (var y = spacing / 2; y < size.height; y += spacing) {
        canvas.drawCircle(Offset(x, y), 1.1, dot);
      }
    }

    final puck = glow;
    if (puck != null) {
      canvas.drawCircle(
        puck,
        grabbing ? 34 : 28,
        Paint()
          ..color = (grabbing ? Z.grape : Z.olive).withValues(alpha: .45),
      );
      canvas.drawCircle(
        puck,
        grabbing ? 34 : 28,
        Paint()
          ..color = grabbing ? Z.grapeSoft : Z.oliveSoft
          ..style = PaintingStyle.stroke
          ..strokeWidth = 1.5,
      );
    }

    for (final ripple in ripples) {
      final t = Curves.easeOut.transform(ripple.controller.value);
      canvas.drawCircle(
        ripple.at,
        18 + 42 * t,
        Paint()
          ..color = (ripple.right ? Z.grapeSoft : Z.oliveSoft)
              .withValues(alpha: (1 - t) * .8)
          ..style = PaintingStyle.stroke
          ..strokeWidth = math.max(1, 3 * (1 - t)),
      );
    }
  }

  @override
  bool shouldRepaint(_PadPainter old) =>
      old.glow != glow ||
      old.grabbing != grabbing ||
      old.ripples.length != ripples.length;
}
