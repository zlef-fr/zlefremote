import 'dart:async';

import 'package:app_links/app_links.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:provider/provider.dart';

import 'core/devices.dart';
import 'core/i18n/i18n.dart';
import 'core/settings.dart';
import 'core/target.dart';
import 'ui/control_screen.dart';
import 'ui/devices_screen.dart';
import 'ui/theme.dart';
import 'ui/widgets.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  SystemChrome.setSystemUIOverlayStyle(const SystemUiOverlayStyle(
    statusBarColor: Colors.transparent,
    statusBarIconBrightness: Brightness.light,
    systemNavigationBarColor: Z.bg,
    systemNavigationBarIconBrightness: Brightness.light,
  ));

  final settings = await ZrSettings.load();
  final devices = ZrDeviceStore();
  await devices.load();

  runApp(MultiProvider(
    providers: [
      ChangeNotifierProvider.value(value: settings),
      ChangeNotifierProvider.value(value: devices),
    ],
    child: const ZlefRemoteApp(),
  ));
}

class ZlefRemoteApp extends StatefulWidget {
  const ZlefRemoteApp({super.key});

  @override
  State<ZlefRemoteApp> createState() => _ZlefRemoteAppState();
}

class _ZlefRemoteAppState extends State<ZlefRemoteApp> {
  final _navigator = GlobalKey<NavigatorState>();
  final _links = AppLinks();
  StreamSubscription<Uri>? _linkSub;

  @override
  void initState() {
    super.initState();
    _listenForPairingLinks();
  }

  /// A pairing URL opened anywhere on the phone — the system camera reading the
  /// agent's QR, a link in a chat, a shared note — lands here instead of in a
  /// browser tab. The key rides in the fragment and Android hands us the whole
  /// URI, fragment included.
  void _listenForPairingLinks() {
    _linkSub = _links.uriLinkStream.listen(_openTarget, onError: (_) {});
    unawaited(_links.getInitialLink().then((uri) {
      if (uri != null) _openTarget(uri);
    }));
  }

  Future<void> _openTarget(Uri uri) async {
    final navigator = _navigator.currentState;
    if (navigator == null) return;
    final target = ZrTarget.parse(uri.toString());
    if (target == null) {
      // a bare https://remote.zlef.fr/… with no key is the marketing site, not
      // a pairing link — say so rather than opening a dead session.
      final context = navigator.context;
      zToast(context, L10n.of(context).t('add_bad_link'));
      return;
    }
    final store = navigator.context.read<ZrDeviceStore>();
    final device =
        await store.upsert(ZrDevice.fromTarget(await target.resolved()));
    if (!mounted) return;
    navigator.push(MaterialPageRoute(
      builder: (_) => ControlScreen(device: device),
    ));
  }

  @override
  void dispose() {
    _linkSub?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final settings = context.watch<ZrSettings>();
    return MaterialApp(
      navigatorKey: _navigator,
      title: 'ZlefRemote',
      debugShowCheckedModeBanner: false,
      theme: Z.theme,
      locale: settings.language == null ? null : Locale(settings.language!),
      supportedLocales: L10n.supported,
      localizationsDelegates: const [
        L10nDelegate(),
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      home: const DevicesScreen(),
    );
  }
}
