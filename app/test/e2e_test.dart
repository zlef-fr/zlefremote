import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:zlefremote/core/caps.dart';
import 'package:zlefremote/core/conn.dart';
import 'package:zlefremote/core/devices.dart';
import 'package:zlefremote/core/i18n/i18n.dart';
import 'package:zlefremote/core/session.dart';
import 'package:zlefremote/core/settings.dart';
import 'package:zlefremote/ui/add_device_screen.dart';
import 'package:zlefremote/ui/control_screen.dart';
import 'package:zlefremote/ui/devices_screen.dart';
import 'package:zlefremote/ui/lock_surface.dart';
import 'package:zlefremote/ui/settings_screen.dart';
import 'package:zlefremote/ui/theme.dart';
import 'package:zlefremote/ui/widgets.dart';

/// End-to-end passes over every screen, in portrait and landscape, in both
/// locales — driven the way a thumb drives them: taps, drags, multi-touch,
/// scrolls, expanders, dialogs.
///
/// The point is not coverage for its own sake. A release APK renders a widget
/// that throws during build as a featureless grey rectangle — no message, no
/// red screen — so a broken pane is indistinguishable from a badly designed
/// one. Every step here asserts `takeException()` is null, which is the only
/// way that class of bug shows up before a device does.
///
/// Every layout assertion is also an overflow assertion: Flutter reports a
/// RenderFlex overflow as an exception in tests, so "no exception after this
/// interaction" means "nothing was clipped or cut off either".
void main() {
  late ZrSettings settings;

  /// Flutter reports a layout failure to FlutterError.onError with the full
  /// creator chain, but `takeException()` hands back only the summary line.
  /// Keeping the details means a failure names the widget instead of leaving
  /// the next person to bisect the tree by hand.
  final diagnostics = <String>[];
  FlutterExceptionHandler? previousOnError;

  setUp(() async {
    SharedPreferences.setMockInitialValues({});
    settings = await ZrSettings.load();
    diagnostics.clear();
    previousOnError = FlutterError.onError;
    FlutterError.onError = (details) {
      diagnostics.add(details.toString());
      previousOnError?.call(details);
    };
  });

  tearDown(() {
    FlutterError.onError = previousOnError;
  });

  /// No exception since the last check — and if there was one, say which
  /// widget caused it.
  void expectClean(WidgetTester tester) {
    final error = tester.takeException();
    if (error != null && diagnostics.isNotEmpty) {
      // printed, not just attached: the reporter drops `reason` on the floor,
      // and the creator chain is the whole value of catching this here.
      // ignore: avoid_print
      print('\n>>> LAYOUT FAILURE DETAIL\n${diagnostics.last}\n<<<\n');
    }
    expect(error, isNull);
  }

  // ── harness ────────────────────────────────────────────────────────────────

  const viewports = <String, Size>{
    'portrait': Size(1080, 2340),
    'landscape': Size(2340, 1080),
    'small portrait': Size(720, 1520),
  };

  const locales = [Locale('en'), Locale('fr')];

  /// A session that records what it would have sent instead of dialling out.
  final sent = <Map<String, dynamic>>[];

  ZrSession session({
    bool paired = true,
    Map<String, dynamic>? welcome,
    String name = 'workstation',
  }) {
    final s = _RecordingSession(sent);
    s.state = paired ? ZrConnState.paired : ZrConnState.connecting;
    s.hostName = paired ? name : '';
    s.caps = ZrCapabilities.fromWelcome(welcome ?? _fullyCapable);
    return s;
  }

  /// Pump past every animation the DA specifies (≤360 ms) plus the reveal
  /// stagger, so a test never asserts against a half-played transition.
  Future<void> settle(WidgetTester tester) async {
    for (var i = 0; i < 8; i++) {
      await tester.pump(const Duration(milliseconds: 120));
    }
  }

  /// Providers sit ABOVE MaterialApp, exactly as main.dart wires them. It
  /// matters: a modal sheet is a route on the root navigator, so providers
  /// nested inside `home:` would be out of its scope and every sheet would
  /// render an ErrorWidget instead of settings.
  Widget app(Widget home, Locale locale, {ZrDeviceStore? store}) => MultiProvider(
        providers: [
          ChangeNotifierProvider.value(value: settings),
          ChangeNotifierProvider.value(value: store ?? _StubStore([])),
        ],
        child: MaterialApp(
          theme: Z.theme,
          locale: locale,
          supportedLocales: L10n.supported,
          localizationsDelegates: const [
            L10nDelegate(),
            GlobalMaterialLocalizations.delegate,
            GlobalWidgetsLocalizations.delegate,
            GlobalCupertinoLocalizations.delegate,
          ],
          home: home,
        ),
      );

  /// Bring a widget into view before touching it. A phone screen shows part of
  /// a pane; a test that taps blind is testing the viewport, not the app.
  Future<void> reveal(WidgetTester tester, Finder target) async {
    if (target.evaluate().isNotEmpty) {
      await tester.ensureVisible(target);
      await settle(tester);
      return;
    }
    // Not built yet: it is below the fold of a lazy list. Try each scrollable
    // on screen rather than guessing which one owns it, and give up quietly —
    // the assertion that follows should report "not found", not a scroll error
    // from the harness.
    for (final scrollable in find.byType(Scrollable).evaluate().toList()) {
      try {
        await tester.scrollUntilVisible(
          target,
          200,
          scrollable: find.byWidget(scrollable.widget),
          maxScrolls: 40,
        );
        await settle(tester);
        return;
      } catch (_) {
        // wrong scrollable — try the next
      }
    }
    await settle(tester);
  }

  Future<void> sized(WidgetTester tester, Size size) async {
    tester.view.physicalSize = size;
    tester.view.devicePixelRatio = 2.75;
    addTearDown(tester.view.reset);
  }

  // ── devices screen ─────────────────────────────────────────────────────────

  for (final viewport in viewports.entries) {
    for (final locale in locales) {
      final tag = '${locale.languageCode} · ${viewport.key}';

      testWidgets('devices: empty state and its call to action — $tag',
          (tester) async {
        await sized(tester, viewport.value);
        final store = _StubStore([]);
        await tester.pumpWidget(app(const DevicesScreen(), locale, store: store));
        await settle(tester);

        final l = L10n(locale);
        expect(find.text(l.t('devices_empty_title')), findsOneWidget);
        expect(find.text(l.t('devices_empty_cta')), findsWidgets);
        expectClean(tester);
      });

      testWidgets('devices: list, per-device menu, rename and remove — $tag',
          (tester) async {
        await sized(tester, viewport.value);
        final store = _StubStore([
          ZrDevice(keyB64: 'a' * 43, name: 'workstation', os: 'linux'),
          ZrDevice(keyB64: 'b' * 43, name: 'laptop', os: 'windows'),
        ]);
        await tester.pumpWidget(app(const DevicesScreen(), locale, store: store));
        await settle(tester);

        expect(find.text('workstation'), findsOneWidget);
        expect(find.text('laptop'), findsOneWidget);

        // the ⋯ menu opens a sheet with rename + remove
        await reveal(tester, find.byIcon(Icons.more_horiz_rounded).first);
        await tester.tap(find.byIcon(Icons.more_horiz_rounded).first);
        await settle(tester);
        final l = L10n(locale);
        expect(find.text(l.t('rename')), findsOneWidget);
        expect(find.text(l.t('remove')), findsOneWidget);

        // remove → confirmation dialog, cancel leaves the device alone
        await reveal(tester, find.text(l.t('remove')));
        await tester.tap(find.text(l.t('remove')));
        await settle(tester);
        expect(find.text(l.t('remove_title')), findsOneWidget);
        await tester.tap(find.text(l.t('cancel')));
        await settle(tester);
        expect(store.devices.length, 2);
        expectClean(tester);
      });

      testWidgets('add device: paste field rejects a non-pairing link — $tag',
          (tester) async {
        await sized(tester, viewport.value);
        await tester.pumpWidget(app(const AddDeviceScreen(), locale));
        await settle(tester);

        final l = L10n(locale);
        await reveal(tester, find.byType(TextField));
        await tester.enterText(find.byType(TextField), 'https://example.com');
        await reveal(tester, find.text(l.t('add_connect')));
        await tester.tap(find.text(l.t('add_connect')));
        await settle(tester);
        expect(find.text(l.t('add_bad_link')), findsOneWidget);
        expectClean(tester);
      });

      testWidgets('settings: sections, language chips, notes stay folded — $tag',
          (tester) async {
        await sized(tester, viewport.value);
        await tester.pumpWidget(app(const SettingsScreen(), locale));
        await settle(tester);

        final l = L10n(locale);
        // an explanation is available but not on screen unasked
        expect(find.text(l.t('haptics_note')), findsNothing);
        await tester.tap(find.byIcon(Icons.info_outline_rounded).first);
        await settle(tester);
        expectClean(tester);

        // language chips switch without rebuilding into an error
        await reveal(tester, find.text('Français'));
        await tester.tap(find.text('Français'));
        await settle(tester);
        expect(settings.language, 'fr');
        expectClean(tester);
      });

      // ── control screen ───────────────────────────────────────────────────

      testWidgets('control: every mode in the dock renders — $tag',
          (tester) async {
        await sized(tester, viewport.value);
        final s = session();
        await tester.pumpWidget(app(
          ControlScreen(device: _device, session: s),
          locale,
        ));
        await settle(tester);

        final l = L10n(locale);
        for (final tab in [
          l.t('tab_screen'),
          l.t('tab_keys'),
          l.t('tab_media'),
          l.t('tab_pad'),
        ]) {
          await tester.tap(find.text(tab));
          await settle(tester);
            expectClean(tester);
        }
        s.dispose();
      });

      testWidgets('control: trackpad drag moves, tap clicks — $tag',
          (tester) async {
        await sized(tester, viewport.value);
        sent.clear();
        final s = session();
        await tester.pumpWidget(app(
          ControlScreen(device: _device, session: s),
          locale,
        ));
        await settle(tester);

        final pad = tester.getCenter(find.byType(CustomPaint).first);
        final drag = await tester.startGesture(pad);
        for (var i = 0; i < 6; i++) {
          await drag.moveBy(const Offset(12, 7));
          await tester.pump(const Duration(milliseconds: 16));
        }
        await drag.up();
        await settle(tester);
        expect(sent.where((c) => c['t'] == 'mv'), isNotEmpty,
            reason: 'a drag must move the pointer');

        sent.clear();
        await tester.tapAt(pad);
        await settle(tester);
        expect(sent.where((c) => c['t'] == 'click' && c['b'] == 'left'),
            isNotEmpty, reason: 'a tap must click');
        expectClean(tester);
        s.dispose();
      });

      testWidgets('control: two fingers scroll, three switch app — $tag',
          (tester) async {
        await sized(tester, viewport.value);
        sent.clear();
        final s = session();
        await tester.pumpWidget(app(
          ControlScreen(device: _device, session: s),
          locale,
        ));
        await settle(tester);

        final pad = tester.getCenter(find.byType(CustomPaint).first);

        // two fingers → scroll
        final f1 = await tester.startGesture(pad.translate(-30, 0));
        final f2 = await tester.startGesture(pad.translate(30, 0));
        for (var i = 0; i < 5; i++) {
          await f1.moveBy(const Offset(0, 14));
          await f2.moveBy(const Offset(0, 14));
          await tester.pump(const Duration(milliseconds: 16));
        }
        await f1.up();
        await f2.up();
        await settle(tester);
        expect(sent.where((c) => c['t'] == 'scroll'), isNotEmpty,
            reason: 'two fingers must scroll');

        // three fingers left → previous app, as an intent
        sent.clear();
        final g1 = await tester.startGesture(pad.translate(-40, 0));
        final g2 = await tester.startGesture(pad);
        final g3 = await tester.startGesture(pad.translate(40, 0));
        for (var i = 0; i < 6; i++) {
          await g1.moveBy(const Offset(-14, 0));
          await g2.moveBy(const Offset(-14, 0));
          await g3.moveBy(const Offset(-14, 0));
          await tester.pump(const Duration(milliseconds: 16));
        }
        await g1.up();
        await g2.up();
        await g3.up();
        await settle(tester);
        final swipe = sent.where((c) => c['t'] == 'gesture');
        expect(swipe, isNotEmpty, reason: 'three fingers must switch app');
        expect(swipe.first['g'], 'app-prev');
        expectClean(tester);
        s.dispose();
      });

      testWidgets('control: two-finger flick goes back, pinch zooms — $tag',
          (tester) async {
        await sized(tester, viewport.value);
        sent.clear();
        final s = session();
        await tester.pumpWidget(app(
          ControlScreen(device: _device, session: s),
          locale,
        ));
        await settle(tester);

        final pad = tester.getCenter(find.byType(CustomPaint).first);

        // a quick sideways flick is history, not a horizontal scroll — and the
        // sideways scroll must have been withheld, or the page moved too.
        final f1 = await tester.startGesture(pad.translate(-30, 20));
        final f2 = await tester.startGesture(pad.translate(30, 20));
        for (var i = 0; i < 6; i++) {
          await f1.moveBy(const Offset(18, 0));
          await f2.moveBy(const Offset(18, 0));
          await tester.pump(const Duration(milliseconds: 16));
        }
        await f1.up();
        await f2.up();
        await settle(tester);
        final nav = sent.where((c) => c['t'] == 'gesture');
        expect(nav, isNotEmpty, reason: 'a sideways flick must go back');
        expect(nav.first['g'], 'nav-back');
        expect(
            sent.where((c) => c['t'] == 'scroll' && (c['dx'] as int) != 0),
            isEmpty,
            reason: 'the flick must not also scroll sideways');

        // spreading two fingers is zoom, not scroll
        sent.clear();
        final p1 = await tester.startGesture(pad.translate(-20, 0));
        final p2 = await tester.startGesture(pad.translate(20, 0));
        for (var i = 0; i < 8; i++) {
          await p1.moveBy(const Offset(-14, 0));
          await p2.moveBy(const Offset(14, 0));
          await tester.pump(const Duration(milliseconds: 16));
        }
        await p1.up();
        await p2.up();
        await settle(tester);
        expect(sent.where((c) => c['g'] == 'zoom-in'), isNotEmpty,
            reason: 'a pinch out must zoom in');
        expect(sent.where((c) => c['t'] == 'scroll'), isEmpty,
            reason: 'a pinch must not scroll');
        expectClean(tester);
        s.dispose();
      });

      testWidgets('control: an old agent still gets the swipe, as a chord — '
          '$tag', (tester) async {
        await sized(tester, viewport.value);
        sent.clear();
        // agent 1.8 and older: no cap.gesture, so the phone resolves the chord
        final s = session(welcome: _noGestureAgent);
        await tester.pumpWidget(app(
          ControlScreen(device: _device, session: s),
          locale,
        ));
        await settle(tester);

        final pad = tester.getCenter(find.byType(CustomPaint).first);
        final g1 = await tester.startGesture(pad.translate(-40, 0));
        final g2 = await tester.startGesture(pad);
        final g3 = await tester.startGesture(pad.translate(40, 0));
        for (var i = 0; i < 6; i++) {
          await g1.moveBy(const Offset(14, 0));
          await g2.moveBy(const Offset(14, 0));
          await g3.moveBy(const Offset(14, 0));
          await tester.pump(const Duration(milliseconds: 16));
        }
        await g1.up();
        await g2.up();
        await g3.up();
        await settle(tester);
        expect(sent.where((c) => c['t'] == 'gesture'), isEmpty,
            reason: 'an old agent would drop the verb');
        final chord = sent.where((c) => c['t'] == 'key' && c['k'] == 'tab');
        expect(chord, isNotEmpty, reason: 'the swipe must still switch app');
        expect((chord.first['mods'] as List).contains('alt'), isTrue);
        expectClean(tester);
        s.dispose();
      });

      testWidgets('control: keyboard types, F-keys unfold and fire — $tag',
          (tester) async {
        await sized(tester, viewport.value);
        sent.clear();
        final s = session();
        await tester.pumpWidget(app(
          ControlScreen(device: _device, session: s),
          locale,
        ));
        await settle(tester);

        final l = L10n(locale);
        await tester.tap(find.text(l.t('tab_keys')));
        await settle(tester);

        await tester.enterText(find.byType(TextField).first, 'hello');
        await settle(tester);
        expect(sent.where((c) => c['t'] == 'text'), isNotEmpty,
            reason: 'typing must reach the computer');

        // F-keys live behind their section
        expect(find.text('F7'), findsNothing);
        await reveal(tester, find.text(l.t('fkeys').toUpperCase()));
        await tester.tap(find.text(l.t('fkeys').toUpperCase()));
        await settle(tester);
        expect(find.text('F1'), findsOneWidget);
        expect(find.text('F12'), findsOneWidget);

        sent.clear();
        await reveal(tester, find.text('F1'));
        await tester.tap(find.text('F1'));
        await settle(tester);
        expect(sent.where((c) => c['t'] == 'key' && c['k'] == 'f1'), isNotEmpty);

        // shortcuts fold open too
        await reveal(tester, find.text(l.t('shortcuts').toUpperCase()));
        await tester.tap(find.text(l.t('shortcuts').toUpperCase()));
        await settle(tester);
        expectClean(tester);
        s.dispose();
      });

      testWidgets('control: media transport and brightness — $tag',
          (tester) async {
        await sized(tester, viewport.value);
        sent.clear();
        final s = session();
        await tester.pumpWidget(app(
          ControlScreen(device: _device, session: s),
          locale,
        ));
        await settle(tester);

        final l = L10n(locale);
        await tester.tap(find.text(l.t('tab_media')));
        await settle(tester);

        await reveal(tester, find.text(l.t('play_pause')));
        await tester.tap(find.text(l.t('play_pause')));
        await settle(tester);
        expect(sent.where((c) => c['t'] == 'media' && c['k'] == 'playpause'),
            isNotEmpty);

        // per-screen brightness chips are present for a two-monitor host
        expect(find.text(l.t('bright_all')), findsOneWidget);
        await reveal(tester, find.byType(Slider));
        await tester.drag(find.byType(Slider), const Offset(60, 0));
        await settle(tester);
        expect(sent.where((c) => c['t'] == 'bright'), isNotEmpty);
        expectClean(tester);
        s.dispose();
      });

      testWidgets('control: an incapable computer explains itself — $tag',
          (tester) async {
        await sized(tester, viewport.value);
        final s = session(welcome: _capableOfNothing);
        await tester.pumpWidget(app(
          ControlScreen(device: _device, session: s),
          locale,
        ));
        await settle(tester);

        final l = L10n(locale);
        await tester.tap(find.text(l.t('tab_screen')));
        await settle(tester);

        // the tab is still there, the feature is named, the reason is one tap
        expect(find.text(l.capTitle(ZrCap.screen)), findsOneWidget);
        expect(find.text(l.t('cap_unavailable')), findsWidgets);
        expect(find.text(l.capReason(ZrCap.screen, ZrCapState.hostLacks)!),
            findsNothing);
        await reveal(tester, find.text(l.capTitle(ZrCap.screen)));
        await tester.tap(find.text(l.capTitle(ZrCap.screen)));
        await settle(tester);
        expect(find.text(l.capReason(ZrCap.screen, ZrCapState.hostLacks)!),
            findsOneWidget);
        expectClean(tester);
        s.dispose();
      });

      testWidgets('control: disconnected overlay names the cause — $tag',
          (tester) async {
        await sized(tester, viewport.value);
        final s = session(paired: false);
        s.state = ZrConnState.error;
        s.error = ZrConnError.noSuchRoom;
        await tester.pumpWidget(app(
          ControlScreen(device: _device, session: s),
          locale,
        ));
        await settle(tester);

        final l = L10n(locale);
        expect(find.text(l.t('err_room_title')), findsOneWidget);
        expect(find.text(l.t('try_again')), findsOneWidget);
        expectClean(tester);
        s.dispose();
      });

      testWidgets('session settings: update button and capability report — $tag',
          (tester) async {
        await sized(tester, viewport.value);
        sent.clear();
        final s = session();
        await tester.pumpWidget(app(
          ControlScreen(device: _device, session: s),
          locale,
        ));
        await settle(tester);

        await tester.tap(find.byIcon(Icons.tune_rounded));
        await settle(tester);

        final l = L10n(locale);
        await reveal(tester, find.text(l.t('agent_update')));
        expect(find.text(l.t('agent_update')), findsOneWidget);
        await tester.tap(find.text(l.t('agent_update')));
        await settle(tester);
        expect(sent.where((c) => c['t'] == 'update'), isNotEmpty);
        expectClean(tester);
        s.dispose();
      });

      // ── lock screen ──────────────────────────────────────────────────────

      testWidgets('lock surface: restricted set, no trackpad — $tag',
          (tester) async {
        await sized(tester, viewport.value);
        sent.clear();
        final s = session();
        await tester.pumpWidget(app(
          ChangeNotifierProvider<ZrSession>.value(
            value: s,
            child: const Scaffold(body: LockSurface()),
          ),
          locale,
        ));
        await settle(tester);

        await tester.tap(find.byIcon(Icons.play_arrow_rounded));
        await settle(tester);
        expect(sent.where((c) => c['t'] == 'media'), isNotEmpty);
        final l = L10n(locale);
        expect(find.text(l.t('lock_unlock_for_more')), findsOneWidget);
        expectClean(tester);
        s.dispose();
      });

      testWidgets('lock: full control is off until confirmed — $tag',
          (tester) async {
        await sized(tester, viewport.value);
        await tester.pumpWidget(app(const SettingsScreen(), locale));
        await settle(tester);

        final l = L10n(locale);
        expect(settings.lockFullControl, isFalse);

        await reveal(tester, find.text(l.t('lock_full')));
        final toggle = find.ancestor(
          of: find.text(l.t('lock_full')),
          matching: find.byType(ZRow),
        );
        await reveal(
            tester, find.descendant(of: toggle, matching: find.byType(ZToggle)));
        await tester.tap(
          find.descendant(of: toggle, matching: find.byType(ZToggle)),
        );
        await settle(tester);

        // a dialog, not a silent flip
        expect(find.text(l.t('lock_full_confirm_title')), findsOneWidget);
        await tester.tap(find.text(l.t('cancel')));
        await settle(tester);
        expect(settings.lockFullControl, isFalse,
            reason: 'cancelling must not enable it');

        await tester.tap(
          find.descendant(of: toggle, matching: find.byType(ZToggle)),
        );
        await settle(tester);
        await tester.tap(find.text(l.t('lock_full_confirm_ok')));
        await settle(tester);
        expect(settings.lockFullControl, isTrue);
        expect(find.text(l.t('lock_full_warning')), findsOneWidget,
            reason: 'while it is on, it says so');
        expectClean(tester);
      });
    }
  }
}

const _device = ZrDevice(keyB64: 'z', name: 'workstation', os: 'linux');

const _fullyCapable = <String, dynamic>{
  'name': 'workstation',
  'os': 'linux',
  'screen': {'w': 1920, 'h': 1080},
  'screens': [
    {'w': 1920, 'h': 1080},
    {'w': 2560, 'h': 1440},
  ],
  'cap': {
    'screen': true,
    'bright': true,
    'keyhold': true,
    'clip': true,
    'update': true,
    'gesture': true,
  },
  'gestures': ['app-next', 'app-prev', 'overview', 'show-desktop',
    'nav-back', 'nav-forward', 'zoom-in', 'zoom-out'],
  'bright': 60,
  'brights': [
    {'name': 'eDP-1', 'v': 60},
    {'name': 'HDMI-1', 'v': 80},
  ],
};

/// Agent 1.8: everything but the gesture vocabulary, so the phone has to send
/// the keyboard shortcut itself.
const _noGestureAgent = <String, dynamic>{
  'name': 'workstation',
  'os': 'linux',
  'screen': {'w': 1920, 'h': 1080},
  'screens': [
    {'w': 1920, 'h': 1080},
  ],
  'cap': {
    'screen': true,
    'bright': true,
    'keyhold': true,
    'clip': true,
    'update': true,
  },
};

const _capableOfNothing = <String, dynamic>{
  'name': 'workstation',
  'os': 'linux',
  'screen': {'w': 1920, 'h': 1080},
  'screens': <Map<String, int>>[],
  'cap': {
    'screen': false,
    'bright': false,
    'keyhold': false,
    'clip': false,
    'update': false,
  },
};

/// Records commands instead of sealing them onto a socket.
class _RecordingSession extends ZrSession {
  _RecordingSession(this.log)
      : super(device: _device, onDeviceLearned: _ignore);

  final List<Map<String, dynamic>> log;

  static void _ignore(ZrDevice _) {}

  @override
  void send(Map<String, dynamic> command) => log.add(command);
}

/// An in-memory device store — the real one talks to the Keystore.
class _StubStore extends ZrDeviceStore {
  _StubStore(this._devices) {
    loaded = true;
  }

  final List<ZrDevice> _devices;

  @override
  List<ZrDevice> get devices => _devices;

  @override
  Future<void> load() async {
    loaded = true;
  }
}
