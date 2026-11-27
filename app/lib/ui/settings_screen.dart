import 'package:flutter/material.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:provider/provider.dart';

import '../core/i18n/i18n.dart';
import '../core/settings.dart';
import '../core/updater.dart';
import 'settings_controls.dart';
import 'theme.dart';
import 'widgets.dart';

class SettingsScreen extends StatefulWidget {
  const SettingsScreen({super.key});

  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  final _updater = ZrUpdater();
  String _version = '';

  @override
  void initState() {
    super.initState();
    PackageInfo.fromPlatform().then((info) {
      if (mounted) {
        setState(() => _version = '${info.version} (${info.buildNumber})');
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final settings = context.watch<ZrSettings>();

    return Scaffold(
      backgroundColor: Z.bg,
      appBar: AppBar(
        backgroundColor: Z.bg,
        surfaceTintColor: Colors.transparent,
        title: Text(l.t('settings'), style: Z.title),
        iconTheme: const IconThemeData(color: Z.inkSoft),
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(Z.s4, 0, Z.s4, Z.s7),
        children: [
          const SettingsControls(),
          ZSectionTitle(l.t('section_language')),
          Wrap(
            spacing: Z.s2,
            children: [
              ZChip(
                label: l.t('language_auto'),
                selected: settings.language == null,
                onTap: () => settings.language = null,
              ),
              ZChip(
                label: 'English',
                selected: settings.language == 'en',
                onTap: () => settings.language = 'en',
              ),
              ZChip(
                label: 'Français',
                selected: settings.language == 'fr',
                onTap: () => settings.language = 'fr',
              ),
            ],
          ),
          ZSectionTitle(l.t('section_about')),
          ZCard(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                ZRow(
                  title: l.t('version'),
                  trailing: Text(_version, style: Z.mono),
                ),
                const Divider(height: 1, color: Z.line),
                const SizedBox(height: Z.s3),
                Text(l.t('e2ee_long'), style: Z.bodySoft),
              ],
            ),
          ),
          const SizedBox(height: Z.s3),
          ListenableBuilder(
            listenable: _updater,
            builder: (context, _) => _UpdateCard(updater: _updater),
          ),
        ],
      ),
    );
  }
}

class _UpdateCard extends StatelessWidget {
  const _UpdateCard({required this.updater});
  final ZrUpdater updater;

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    return ZCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(l.t('update_title').toUpperCase(), style: Z.eyebrow),
          const SizedBox(height: Z.s2),
          Text(
            switch (updater.state) {
              ZrUpdateState.checking => l.t('update_checking'),
              ZrUpdateState.upToDate => l.t('update_current'),
              ZrUpdateState.available =>
                l.f('update_available', {'v': updater.latestVersion ?? ''}),
              ZrUpdateState.downloading => l.t('update_downloading'),
              ZrUpdateState.readyToInstall => l.t('update_install'),
              ZrUpdateState.failed => l.t('update_failed'),
              ZrUpdateState.idle => l.t('update_check'),
            },
            style: Z.bodySoft,
          ),
          if (updater.state == ZrUpdateState.downloading) ...[
            const SizedBox(height: Z.s3),
            LinearProgressIndicator(
              value: updater.progress > 0 ? updater.progress : null,
              backgroundColor: Z.surface3,
              valueColor: const AlwaysStoppedAnimation(Z.oliveMid),
              minHeight: 6,
            ),
          ],
          const SizedBox(height: Z.s3),
          ZButton(
            label: switch (updater.state) {
              ZrUpdateState.available => l.t('update_download'),
              ZrUpdateState.readyToInstall => l.t('update_install'),
              _ => l.t('update_check'),
            },
            kind: updater.state == ZrUpdateState.available ||
                    updater.state == ZrUpdateState.readyToInstall
                ? ZButtonKind.primary
                : ZButtonKind.ghost,
            loading: updater.state == ZrUpdateState.checking ||
                updater.state == ZrUpdateState.downloading,
            expand: true,
            onPressed: () async {
              switch (updater.state) {
                case ZrUpdateState.available:
                  await updater.download();
                case ZrUpdateState.readyToInstall:
                  final ok = await updater.install();
                  if (!ok && context.mounted) {
                    zToast(context, l.t('update_needs_permission'));
                  }
                default:
                  await updater.check();
              }
            },
          ),
        ],
      ),
    );
  }
}
