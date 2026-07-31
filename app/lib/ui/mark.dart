import 'package:flutter/material.dart';

import 'theme.dart';

/// The app mark: a screen with a pointer in it, drawn in geometric strokes with
/// an angular notch cut out of the top-right corner — the same notch language
/// as the zlef.fr wordmark. No asset, no font, scales to any size.
class ZrMark extends StatelessWidget {
  const ZrMark({super.key, this.size = 28, this.color, this.accent});

  final double size;
  final Color? color;
  final Color? accent;

  @override
  Widget build(BuildContext context) => CustomPaint(
        size: Size.square(size),
        painter: _MarkPainter(
          color: color ?? Z.ink,
          accent: accent ?? Z.oliveSoft,
        ),
      );
}

class _MarkPainter extends CustomPainter {
  _MarkPainter({required this.color, required this.accent});

  final Color color;
  final Color accent;

  @override
  void paint(Canvas canvas, Size size) {
    final u = size.width / 32; // design grid is 32×32
    final stroke = Paint()
      ..color = color
      ..style = PaintingStyle.stroke
      ..strokeWidth = 2.4 * u
      ..strokeJoin = StrokeJoin.miter;

    // screen outline, top-right corner notched off
    final screen = Path()
      ..moveTo(3 * u, 5 * u)
      ..lineTo(22 * u, 5 * u)
      ..lineTo(29 * u, 12 * u)
      ..lineTo(29 * u, 23 * u)
      ..lineTo(3 * u, 23 * u)
      ..close();
    canvas.drawPath(screen, stroke);

    // the pointer inside it — solid olive, the one filled shape
    final arrow = Path()
      ..moveTo(13 * u, 10 * u)
      ..lineTo(22 * u, 16 * u)
      ..lineTo(17.6 * u, 17 * u)
      ..lineTo(19.6 * u, 20.6 * u)
      ..lineTo(17 * u, 21.8 * u)
      ..lineTo(15.1 * u, 18.2 * u)
      ..lineTo(12 * u, 20.6 * u)
      ..close();
    canvas.drawPath(arrow, Paint()..color = accent);

    // stand
    canvas.drawLine(
      Offset(13 * u, 27 * u),
      Offset(21 * u, 27 * u),
      stroke..strokeWidth = 2.4 * u,
    );
  }

  @override
  bool shouldRepaint(_MarkPainter old) =>
      old.color != color || old.accent != accent;
}
