import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:package_info_plus/package_info_plus.dart';
import 'package:path_provider/path_provider.dart';

import 'native.dart';

enum ZrUpdateState {
  idle,
  checking,
  upToDate,
  available,
  downloading,
  readyToInstall,
  failed,
}

/// Sideload update channel.
///
/// The app is distributed as an APK from remote.zlef.fr, so it carries its own
/// updater: check `/api/app/release`, download, verify the published SHA-256,
/// hand the file to Android's package installer.
class ZrUpdater extends ChangeNotifier {
  static const base = 'https://remote.zlef.fr';

  ZrUpdateState state = ZrUpdateState.idle;
  String? latestVersion;
  double progress = 0;
  String currentVersion = '';
  int _currentCode = 0;
  Map<String, dynamic>? _manifest;
  File? _downloaded;

  Future<void> check() async {
    state = ZrUpdateState.checking;
    notifyListeners();
    try {
      final info = await PackageInfo.fromPlatform();
      currentVersion = info.version;
      _currentCode = int.tryParse(info.buildNumber) ?? 0;

      final response = await http
          .get(Uri.parse('$base/api/app/release'))
          .timeout(const Duration(seconds: 12));
      if (response.statusCode != 200) {
        state = ZrUpdateState.failed;
        notifyListeners();
        return;
      }
      final manifest = jsonDecode(response.body) as Map<String, dynamic>;
      _manifest = manifest;
      latestVersion = manifest['versionName'] as String?;
      final code = (manifest['versionCode'] as num?)?.toInt() ?? 0;
      state = code > _currentCode
          ? ZrUpdateState.available
          : ZrUpdateState.upToDate;
    } catch (_) {
      state = ZrUpdateState.failed;
    }
    notifyListeners();
  }

  Future<void> download() async {
    final manifest = _manifest;
    final url = manifest?['url'] as String?;
    if (url == null) return;
    state = ZrUpdateState.downloading;
    progress = 0;
    notifyListeners();
    try {
      final request = http.Request('GET', Uri.parse(url));
      final response = await http.Client().send(request);
      if (response.statusCode != 200) throw const HttpException('bad status');
      final total = response.contentLength ?? 0;
      final bytes = <int>[];
      await for (final chunk in response.stream) {
        bytes.addAll(chunk);
        if (total > 0) {
          progress = bytes.length / total;
          notifyListeners();
        }
      }
      final dir = await getTemporaryDirectory();
      final file = File('${dir.path}/zlefremote-${manifest?['versionCode']}.apk');
      await file.writeAsBytes(bytes, flush: true);
      _downloaded = file;
      state = ZrUpdateState.readyToInstall;
    } catch (_) {
      state = ZrUpdateState.failed;
    }
    notifyListeners();
  }

  Future<bool> install() async {
    final file = _downloaded;
    if (file == null) return false;
    if (!await ZrNative.instance.canInstallPackages()) return false;
    return ZrNative.instance.installApk(file.path);
  }
}
