import 'package:flutter/material.dart';

/// The zlef.fr design authority (da.zlef.fr), expressed for a touch surface.
///
/// Dark only, near-black ground, two deep desaturated accents (olive leaf,
/// vintage grape), light rounding (3/5/8/12), flat crisp components — no
/// gradients, no coloured glow, no bounce. Type never below 16, controls never
/// below 44. Grape appears atmospherically, never as a UI fill.
abstract final class Z {
  // ── surfaces ───────────────────────────────────────────────────────────────
  static const bg = Color(0xFF06060A);
  static const surface1 = Color(0xFF0E0E13);
  static const surface2 = Color(0xFF15151C);
  static const surface3 = Color(0xFF1D1D25);

  // ── ink ────────────────────────────────────────────────────────────────────
  static const ink = Color(0xFFE9EAE2);
  static const inkSoft = Color(0xFFB6B7AD);
  static const inkMuted = Color(0xFF7D7E76);
  static const inkFaint = Color(0xFF54554F);

  // ── olive leaf (primary) ───────────────────────────────────────────────────
  static const oliveDeep = Color(0xFF2C3212);
  static const olive = Color(0xFF3E4618);
  static const oliveMid = Color(0xFF59642A);
  static const oliveSoft = Color(0xFF9DAE50);
  static const oliveBright = Color(0xFFBDCE74);

  // ── vintage grape (secondary) ──────────────────────────────────────────────
  static const grapeDeep = Color(0xFF382B39);
  static const grape = Color(0xFF4C3B4D);
  static const grapeMid = Color(0xFF6B5470);
  static const grapeSoft = Color(0xFFB095B3);

  // ── semantic ───────────────────────────────────────────────────────────────
  static const success = Color(0xFF6F9A3A);
  static const warning = Color(0xFFC79A3E);
  static const danger = Color(0xFFC2566A);

  // ── lines ──────────────────────────────────────────────────────────────────
  static const line = Color(0x12FFFFFF); // rgba(255,255,255,.07)
  static const lineStrong = Color(0x21FFFFFF); // rgba(255,255,255,.13)
  static const scrim = Color(0xCC06060A);

  // ── geometry ───────────────────────────────────────────────────────────────
  static const rSm = 3.0; // badges, chips
  static const rMd = 5.0; // buttons, inputs
  static const rLg = 8.0; // cards
  static const rXl = 12.0; // sheets, stages

  static const s1 = 4.0;
  static const s2 = 8.0;
  static const s3 = 12.0;
  static const s4 = 16.0;
  static const s5 = 24.0;
  static const s6 = 32.0;
  static const s7 = 48.0;

  /// Minimum touch target. "Never too small, never too big."
  static const tap = 48.0;

  // ── motion ─────────────────────────────────────────────────────────────────
  static const ease = Cubic(.22, .61, .36, 1);
  static const fast = Duration(milliseconds: 140);
  static const normal = Duration(milliseconds: 220);
  static const slow = Duration(milliseconds: 360);

  // ── type ───────────────────────────────────────────────────────────────────
  static const _sans = [
    'Roboto',
    'system-ui',
    '-apple-system',
    'Segoe UI',
    'Helvetica',
    'Arial',
  ];
  static const _mono = ['monospace'];

  static const eyebrow = TextStyle(
    fontSize: 13,
    height: 1.3,
    fontWeight: FontWeight.w700,
    letterSpacing: 1.8,
    color: oliveSoft,
    fontFamilyFallback: _sans,
  );
  static const label = TextStyle(
    fontSize: 14.5,
    height: 1.4,
    fontWeight: FontWeight.w600,
    color: inkSoft,
    fontFamilyFallback: _sans,
  );
  static const body = TextStyle(
    fontSize: 17,
    height: 1.5,
    color: ink,
    fontFamilyFallback: _sans,
  );
  static const bodySoft = TextStyle(
    fontSize: 16,
    height: 1.55,
    color: inkSoft,
    fontFamilyFallback: _sans,
  );
  static const title = TextStyle(
    fontSize: 22,
    height: 1.25,
    fontWeight: FontWeight.w700,
    letterSpacing: -0.4,
    color: ink,
    fontFamilyFallback: _sans,
  );
  static const display = TextStyle(
    fontSize: 30,
    height: 1.15,
    fontWeight: FontWeight.w700,
    letterSpacing: -0.7,
    color: ink,
    fontFamilyFallback: _sans,
  );
  static const mono = TextStyle(
    fontSize: 14,
    height: 1.4,
    color: inkSoft,
    fontFamilyFallback: _mono,
  );

  /// The one Material theme in the app — everything visible is built from the
  /// widgets in ui/widgets.dart, so this exists to keep framework-owned pixels
  /// (text selection, cursors, sheets) on palette.
  static ThemeData get theme {
    const scheme = ColorScheme.dark(
      primary: oliveSoft,
      onPrimary: bg,
      secondary: grapeSoft,
      onSecondary: bg,
      surface: surface1,
      onSurface: ink,
      error: danger,
      onError: bg,
    );
    return ThemeData(
      useMaterial3: true,
      brightness: Brightness.dark,
      colorScheme: scheme,
      scaffoldBackgroundColor: bg,
      canvasColor: bg,
      splashFactory: NoSplash.splashFactory, // flat: no bloom on press
      highlightColor: Colors.transparent,
      textSelectionTheme: const TextSelectionThemeData(
        cursorColor: oliveSoft,
        selectionColor: Color(0x553E4618),
        selectionHandleColor: oliveSoft,
      ),
      snackBarTheme: const SnackBarThemeData(
        backgroundColor: surface3,
        contentTextStyle: TextStyle(color: ink, fontSize: 16),
        behavior: SnackBarBehavior.floating,
      ),
      pageTransitionsTheme: const PageTransitionsTheme(
        builders: {
          TargetPlatform.android: _ZPageTransitions(),
          TargetPlatform.iOS: _ZPageTransitions(),
        },
      ),
      textTheme: const TextTheme(bodyMedium: body),
    );
  }
}

/// Page motion: a short rise + fade on the DA easing. No slide-from-edge, no
/// bounce — pages arrive, they don't swoop.
class _ZPageTransitions extends PageTransitionsBuilder {
  const _ZPageTransitions();

  @override
  Widget buildTransitions<T>(
    PageRoute<T> route,
    BuildContext context,
    Animation<double> animation,
    Animation<double> secondaryAnimation,
    Widget child,
  ) {
    final curved = CurvedAnimation(parent: animation, curve: Z.ease);
    return FadeTransition(
      opacity: curved,
      child: SlideTransition(
        position: Tween(begin: const Offset(0, .03), end: Offset.zero)
            .animate(curved),
        child: child,
      ),
    );
  }
}
