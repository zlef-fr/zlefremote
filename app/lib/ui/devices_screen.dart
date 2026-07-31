import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../core/devices.dart';
import '../core/i18n/i18n.dart';
import 'add_device_screen.dart';
import 'control_screen.dart';
import 'mark.dart';
import 'settings_screen.dart';
import 'theme.dart';
import 'widgets.dart';

/// Home: the computers this phone can drive.
///
/// One tap reconnects. There is no "connect" step to hunt for and no QR to
/// rescan, because a remembered computer's room is re-derived from the key we
/// already hold.
class DevicesScreen extends StatelessWidget {
  const DevicesScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final store = context.watch<ZrDeviceStore>();
    final devices = store.devices;

    return Scaffold(
      backgroundColor: Z.bg,
      body: SafeArea(
        child: Column(
          children: [
            _Header(onSettings: () => _openSettings(context)),
            Expanded(
              child: devices.isEmpty
                  ? _Empty(onAdd: () => _openAdd(context))
                  : ListView.separated(
                      padding: const EdgeInsets.fromLTRB(Z.s4, Z.s2, Z.s4, 120),
                      itemCount: devices.length + 1,
                      separatorBuilder: (_, _) =>
                          const SizedBox(height: Z.s3),
                      itemBuilder: (context, i) {
                        if (i == 0) {
                          return Padding(
                            padding: const EdgeInsets.only(bottom: Z.s2),
                            child: Text(l.t('devices_sub'), style: Z.bodySoft),
                          );
                        }
                        final device = devices[i - 1];
                        return ZReveal(
                          index: i,
                          child: _DeviceCard(
                            device: device,
                            onOpen: () => _open(context, device),
                          ),
                        );
                      },
                    ),
            ),
          ],
        ),
      ),
      floatingActionButton: _AddButton(onTap: () => _openAdd(context)),
    );
  }

  void _open(BuildContext context, ZrDevice device) {
    context.read<ZrDeviceStore>().touch(device);
    Navigator.of(context).push(MaterialPageRoute(
      builder: (_) => ControlScreen(device: device),
    ));
  }

  void _openAdd(BuildContext context) => Navigator.of(context).push(
        MaterialPageRoute(builder: (_) => const AddDeviceScreen()),
      );

  void _openSettings(BuildContext context) => Navigator.of(context).push(
        MaterialPageRoute(builder: (_) => const SettingsScreen()),
      );
}

class _Header extends StatelessWidget {
  const _Header({required this.onSettings});
  final VoidCallback onSettings;

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    return Padding(
      padding: const EdgeInsets.fromLTRB(Z.s4, Z.s4, Z.s2, Z.s2),
      child: Row(
        children: [
          const ZrMark(size: 30),
          const SizedBox(width: Z.s3),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(l.t('devices_title'), style: Z.display),
              ],
            ),
          ),
          IconButton(
            onPressed: onSettings,
            tooltip: l.t('settings'),
            icon: const Icon(Icons.tune_rounded, color: Z.inkSoft, size: 26),
          ),
        ],
      ),
    );
  }
}

class _DeviceCard extends StatelessWidget {
  const _DeviceCard({required this.device, required this.onOpen});

  final ZrDevice device;
  final VoidCallback onOpen;

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final name = device.name.isNotEmpty ? device.name : l.t('unknown_computer');

    return ZCard(
      onTap: onOpen,
      padding: const EdgeInsets.all(Z.s3),
      child: Row(
        children: [
          Container(
            width: 46,
            height: 46,
            decoration: BoxDecoration(
              color: Z.surface2,
              borderRadius: BorderRadius.circular(Z.rMd),
              border: Border.all(color: Z.line),
            ),
            child: Icon(_osIcon(device.os), color: Z.oliveSoft, size: 24),
          ),
          const SizedBox(width: Z.s3),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(name,
                    style: Z.body.copyWith(fontWeight: FontWeight.w600),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis),
                const SizedBox(height: 2),
                Row(
                  children: [
                    Text(
                      '${l.t('last_used')} · ${l.relativeTime(device.lastUsed)}',
                      style: Z.mono.copyWith(color: Z.inkMuted),
                    ),
                    if (device.isLan) ...[
                      const SizedBox(width: Z.s2),
                      _Tag(l.t('lan_device'), Icons.wifi_rounded),
                    ] else if (!device.persistent) ...[
                      const SizedBox(width: Z.s2),
                      _Tag(l.t('one_shot_device'), Icons.timer_outlined),
                    ],
                  ],
                ),
              ],
            ),
          ),
          IconButton(
            onPressed: () => _menu(context),
            icon: const Icon(Icons.more_horiz_rounded, color: Z.inkMuted),
          ),
        ],
      ),
    );
  }

  static IconData _osIcon(String os) => switch (os.toLowerCase()) {
        'windows' || 'win' => Icons.window_rounded,
        'darwin' || 'mac' => Icons.laptop_mac_rounded,
        'linux' => Icons.terminal_rounded,
        _ => Icons.desktop_windows_rounded,
      };

  Future<void> _menu(BuildContext context) async {
    final l = L10n.of(context);
    final store = context.read<ZrDeviceStore>();
    await showZSheet<void>(
      context,
      title: device.name.isNotEmpty ? device.name : l.t('unknown_computer'),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (!device.persistent && !device.isLan) ...[
            ZCard(
              dimmed: true,
              child: Text(l.t('one_shot_note'), style: Z.bodySoft),
            ),
            const SizedBox(height: Z.s4),
          ],
          ZButton(
            label: l.t('rename'),
            kind: ZButtonKind.ghost,
            icon: Icons.edit_outlined,
            expand: true,
            onPressed: () {
              Navigator.of(context).pop();
              _rename(context, store);
            },
          ),
          const SizedBox(height: Z.s3),
          ZButton(
            label: l.t('remove'),
            kind: ZButtonKind.destructive,
            icon: Icons.delete_outline_rounded,
            expand: true,
            onPressed: () {
              Navigator.of(context).pop();
              _confirmRemove(context, store);
            },
          ),
        ],
      ),
    );
  }

  Future<void> _rename(BuildContext context, ZrDeviceStore store) async {
    final l = L10n.of(context);
    final controller = TextEditingController(text: device.name);
    final name = await showDialog<String>(
      context: context,
      builder: (context) => AlertDialog(
        backgroundColor: Z.surface1,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(Z.rLg),
          side: const BorderSide(color: Z.lineStrong),
        ),
        title: Text(l.t('rename_title'), style: Z.title),
        content: TextField(
          controller: controller,
          autofocus: true,
          style: Z.body,
          decoration: InputDecoration(
            hintText: l.t('rename_hint'),
            hintStyle: Z.bodySoft.copyWith(color: Z.inkFaint),
            enabledBorder: const OutlineInputBorder(
              borderSide: BorderSide(color: Z.lineStrong),
            ),
            focusedBorder: const OutlineInputBorder(
              borderSide: BorderSide(color: Z.oliveMid),
            ),
          ),
        ),
        actions: [
          ZButton(
            label: l.t('cancel'),
            kind: ZButtonKind.ghost,
            size: ZButtonSize.sm,
            onPressed: () => Navigator.of(context).pop(),
          ),
          ZButton(
            label: l.t('save'),
            size: ZButtonSize.sm,
            onPressed: () => Navigator.of(context).pop(controller.text.trim()),
          ),
        ],
      ),
    );
    if (name != null && name.isNotEmpty) await store.rename(device, name);
  }

  Future<void> _confirmRemove(
      BuildContext context, ZrDeviceStore store) async {
    final l = L10n.of(context);
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        backgroundColor: Z.surface1,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(Z.rLg),
          side: const BorderSide(color: Z.lineStrong),
        ),
        title: Text(l.t('remove_title'), style: Z.title),
        content: Text(l.t('remove_body'), style: Z.bodySoft),
        actions: [
          ZButton(
            label: l.t('cancel'),
            kind: ZButtonKind.secondary,
            size: ZButtonSize.sm,
            onPressed: () => Navigator.of(context).pop(false),
          ),
          ZButton(
            label: l.t('remove'),
            kind: ZButtonKind.destructive,
            size: ZButtonSize.sm,
            onPressed: () => Navigator.of(context).pop(true),
          ),
        ],
      ),
    );
    if (ok == true) await store.remove(device);
  }
}

class _Tag extends StatelessWidget {
  const _Tag(this.label, this.icon);
  final String label;
  final IconData icon;

  @override
  Widget build(BuildContext context) => Container(
        padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
        decoration: BoxDecoration(
          color: Z.surface3,
          borderRadius: BorderRadius.circular(Z.rSm),
          border: Border.all(color: Z.line),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 12, color: Z.inkMuted),
            const SizedBox(width: 4),
            Text(label,
                style: Z.mono.copyWith(fontSize: 11, color: Z.inkMuted)),
          ],
        ),
      );
}

class _Empty extends StatelessWidget {
  const _Empty({required this.onAdd});
  final VoidCallback onAdd;

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    return Center(
      child: ZReveal(
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: Z.s5),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const ZrMark(size: 88, opacity: .55),
              const SizedBox(height: Z.s5),
              Text(l.t('devices_empty_title'),
                  style: Z.title, textAlign: TextAlign.center),
              const SizedBox(height: Z.s2),
              Text(l.t('devices_empty_sub'),
                  style: Z.bodySoft, textAlign: TextAlign.center),
              const SizedBox(height: Z.s5),
              ZButton(
                label: l.t('devices_empty_cta'),
                icon: Icons.qr_code_scanner_rounded,
                size: ZButtonSize.lg,
                onPressed: onAdd,
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _AddButton extends StatelessWidget {
  const _AddButton({required this.onTap});
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    return Padding(
      padding: const EdgeInsets.only(bottom: Z.s2, right: Z.s1),
      child: ZButton(
        label: l.t('add_device'),
        icon: Icons.qr_code_scanner_rounded,
        size: ZButtonSize.lg,
        onPressed: onTap,
      ),
    );
  }
}
