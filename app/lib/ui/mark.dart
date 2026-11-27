import 'package:flutter/material.dart';

/// The app mark.
///
/// This is the product's own icon (assets/icon.png, the same file the web
/// client ships as public/app/icons/icon-512.png) rather than a redrawing of
/// it, so the two can never drift apart.
class ZrMark extends StatelessWidget {
  const ZrMark({super.key, this.size = 28, this.opacity = 1});

  final double size;

  /// Dimmed for decorative use (the empty state), full strength in chrome.
  final double opacity;

  @override
  Widget build(BuildContext context) => Opacity(
        opacity: opacity,
        child: Image.asset(
          'assets/icon.png',
          width: size,
          height: size,
          filterQuality: FilterQuality.medium,
        ),
      );
}
