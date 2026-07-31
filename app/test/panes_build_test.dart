import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:zlefremote/core/caps.dart';
import 'package:zlefremote/core/devices.dart';
import 'package:zlefremote/core/i18n/i18n.dart';
import 'package:zlefremote/core/session.dart';
import 'package:zlefremote/core/settings.dart';
import 'package:zlefremote/ui/panes/keys_pane.dart';
import 'package:zlefremote/ui/panes/media_pane.dart';
import 'package:zlefremote/ui/panes/screen_pane.dart';
import 'package:zlefremote/ui/panes/touchpad_pane.dart';
import 'package:zlefremote/ui/theme.dart';

/// Every pane must build on a phone-sized viewport, in both locales, for a
/// computer that can do everything and for one that can do nothing.
///
/// This exists because a build error in a release APK renders as a featureless
/// grey rectangle — no message, no red screen — so a pane that throws looks
/// exactly like a pane that was designed badly. The first version of the
/// keyboard pane shipped that way.
void main() {
  late ZrSettings settings;

  setUp(() async {
    SharedPreferences.setMockInitialValues({});
    settings = await ZrSettings.load();
  });

  ZrSession sessionWith(Map<String, dynamic> welcome) {
    final session = ZrSession(
      device: const ZrDevice(keyB64: 'x', name: 'test'),
      onDeviceLearned: (_) {},
    );
    session.caps = ZrCapabilities.fromWelcome(welcome);
    return session;
  }

  Widget host(Widget pane, ZrSession session, Locale locale) => MaterialApp(
        theme: Z.theme,
        locale: locale,
        supportedLocales: L10n.supported,
        localizationsDelegates: const [
          L10nDelegate(),
          GlobalMaterialLocalizations.delegate,
          GlobalWidgetsLocalizations.delegate,
          GlobalCupertinoLocalizations.delegate,
        ],
        home: MultiProvider(
          providers: [
            ChangeNotifierProvider.value(value: settings),
            ChangeNotifierProvider.value(value: session),
          ],
          child: Scaffold(body: pane),
        ),
      );

  final welcomes = {
    'fully capable': {
      'name': 'pc',
      'os': 'linux',
      'screen': {'w': 1920, 'h': 1080},
      'screens': [
        {'w': 1920, 'h': 1080},
        {'w': 2560, 'h': 1440},
      ],
      'cap': {'screen': true, 'bright': true, 'keyhold': true, 'clip': true},
      'bright': 60,
      'brights': [
        {'name': 'eDP-1', 'v': 60},
        {'name': 'HDMI-1', 'v': 80},
      ],
      'backends': [
        {'id': 'brightnessctl', 'label': 'Backlight', 'kind': 'hardware'},
        {'id': 'xrandr', 'label': 'Gamma', 'kind': 'software'},
      ],
      'backend': 'brightnessctl',
    },
    'capable of nothing': {
      'name': 'pc',
      'os': 'linux',
      'screen': {'w': 1920, 'h': 1080},
      'screens': <Map<String, int>>[],
      'cap': {'screen': false, 'bright': false, 'keyhold': false, 'clip': false},
    },
    'old agent': {
      'name': 'pc',
      'os': 'linux',
      'screen': {'w': 1920, 'h': 1080},
      'screens': <Map<String, int>>[],
      'cap': {'screen': false, 'bright': false},
    },
  };

  final panes = <String, Widget>{
    'touchpad': const TouchpadPane(),
    'screen': const ScreenPane(),
    'keys': const KeysPane(),
    'media': const MediaPane(),
  };

  const viewports = {
    'portrait': Size(1080, 2340),
    'landscape': Size(2340, 1080),
    // a small phone: 360pt wide is where long French labels bite
    'narrow': Size(990, 1980),
  };

  for (final viewport in viewports.entries) {
  for (final locale in [const Locale('en'), const Locale('fr')]) {
    for (final welcome in welcomes.entries) {
      for (final pane in panes.entries) {
        testWidgets(
          '${pane.key} — ${welcome.key} · ${locale.languageCode} · ${viewport.key}',
          (tester) async {
            tester.view.physicalSize = viewport.value;
            tester.view.devicePixelRatio = 2.75;
            addTearDown(tester.view.reset);

            await tester.pumpWidget(
              host(pane.value, sessionWith(welcome.value), locale),
            );
            await tester.pump(const Duration(milliseconds: 400));

            expect(tester.takeException(), isNull);
          },
        );
      }
    }
  }
  }
}
