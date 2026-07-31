import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:mobile_scanner/mobile_scanner.dart';
import 'package:provider/provider.dart';

import '../core/devices.dart';
import '../core/i18n/i18n.dart';
import '../core/target.dart';
import 'control_screen.dart';
import 'theme.dart';
import 'widgets.dart';

/// Pairing. The camera is the whole screen and the scanner runs on open — the
/// web client could only offer this on one browser engine, behind a button,
/// and had to fall back to "paste a link" everywhere else.
class AddDeviceScreen extends StatefulWidget {
  const AddDeviceScreen({super.key});

  @override
  State<AddDeviceScreen> createState() => _AddDeviceScreenState();
}

class _AddDeviceScreenState extends State<AddDeviceScreen> {
  final _controller = MobileScannerController(
    formats: const [BarcodeFormat.qrCode],
    detectionSpeed: DetectionSpeed.noDuplicates,
  );
  final _paste = TextEditingController();
  bool _handled = false;
  String? _pasteError;

  @override
  void dispose() {
    _controller.dispose();
    _paste.dispose();
    super.dispose();
  }

  Future<void> _onDetect(BarcodeCapture capture) async {
    if (_handled) return;
    for (final barcode in capture.barcodes) {
      final target = ZrTarget.parse(barcode.rawValue ?? '');
      if (target != null) {
        _handled = true;
        HapticFeedback.mediumImpact();
        await _connect(target);
        return;
      }
    }
  }

  Future<void> _connect(ZrTarget target) async {
    final store = context.read<ZrDeviceStore>();
    final device =
        await store.upsert(ZrDevice.fromTarget(await target.resolved()));
    if (!mounted) return;
    Navigator.of(context).pushReplacement(MaterialPageRoute(
      builder: (_) => ControlScreen(device: device),
    ));
  }

  Future<void> _connectPasted() async {
    final target = ZrTarget.parse(_paste.text);
    if (target == null) {
      setState(() => _pasteError = L10n.of(context).t('add_bad_link'));
      return;
    }
    setState(() => _pasteError = null);
    await _connect(target);
  }

  Future<void> _pasteFromClipboard() async {
    final l = L10n.of(context);
    final data = await Clipboard.getData(Clipboard.kTextPlain);
    final text = data?.text?.trim() ?? '';
    if (!mounted) return;
    if (text.isEmpty) {
      zToast(context, l.t('add_clipboard_empty'));
      return;
    }
    _paste.text = text;
    await _connectPasted();
  }

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    return Scaffold(
      backgroundColor: Z.bg,
      appBar: AppBar(
        backgroundColor: Z.bg,
        surfaceTintColor: Colors.transparent,
        title: Text(l.t('add_title'), style: Z.title),
        iconTheme: const IconThemeData(color: Z.inkSoft),
      ),
      body: SafeArea(
        top: false,
        child: ListView(
          padding: const EdgeInsets.fromLTRB(Z.s4, Z.s2, Z.s4, Z.s6),
          children: [
            Text(l.t('add_scan_title'),
                style: Z.body.copyWith(fontWeight: FontWeight.w600)),
            const SizedBox(height: Z.s1),
            Text(l.t('add_scan_hint'), style: Z.bodySoft),
            const SizedBox(height: Z.s3),
            _Viewfinder(controller: _controller, onDetect: _onDetect),
            const SizedBox(height: Z.s5),
            Text(l.t('add_paste_title'),
                style: Z.body.copyWith(fontWeight: FontWeight.w600)),
            const SizedBox(height: Z.s2),
            TextField(
              controller: _paste,
              style: Z.body.copyWith(fontSize: 15),
              minLines: 1,
              maxLines: 3,
              keyboardType: TextInputType.url,
              autocorrect: false,
              onSubmitted: (_) => _connectPasted(),
              decoration: InputDecoration(
                hintText: l.t('add_paste_ph'),
                hintStyle: Z.mono.copyWith(color: Z.inkFaint),
                filled: true,
                fillColor: Z.surface2,
                errorText: _pasteError,
                errorStyle: Z.label.copyWith(color: Z.danger),
                enabledBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(Z.rMd),
                  borderSide: const BorderSide(color: Z.line),
                ),
                focusedBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(Z.rMd),
                  borderSide: const BorderSide(color: Z.oliveMid),
                ),
                errorBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(Z.rMd),
                  borderSide: const BorderSide(color: Z.danger),
                ),
                focusedErrorBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(Z.rMd),
                  borderSide: const BorderSide(color: Z.danger),
                ),
              ),
            ),
            const SizedBox(height: Z.s3),
            Row(
              children: [
                Expanded(
                  child: ZButton(
                    label: l.t('add_paste_from_clipboard'),
                    kind: ZButtonKind.ghost,
                    icon: Icons.content_paste_rounded,
                    expand: true,
                    onPressed: _pasteFromClipboard,
                  ),
                ),
                const SizedBox(width: Z.s3),
                ZButton(label: l.t('add_connect'), onPressed: _connectPasted),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

/// The camera stage. Its three states — running, starting, refused — are all
/// rendered; a blank rectangle would leave the user guessing.
class _Viewfinder extends StatelessWidget {
  const _Viewfinder({required this.controller, required this.onDetect});

  final MobileScannerController controller;
  final void Function(BarcodeCapture) onDetect;

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    return ClipRRect(
      borderRadius: BorderRadius.circular(Z.rLg),
      child: AspectRatio(
        aspectRatio: 1,
        child: ValueListenableBuilder<MobileScannerState>(
          valueListenable: controller,
          builder: (context, state, _) {
            final error = state.error;
            if (error != null) {
              return _CameraProblem(
                title: error.errorCode == MobileScannerErrorCode.permissionDenied
                    ? l.t('add_camera_denied')
                    : l.t('cap_unavailable'),
                body: error.errorCode == MobileScannerErrorCode.permissionDenied
                    ? l.t('add_camera_denied_sub')
                    : (error.errorDetails?.message ?? error.errorCode.name),
              );
            }
            return Stack(
              fit: StackFit.expand,
              children: [
                MobileScanner(controller: controller, onDetect: onDetect),
                const _ViewfinderFrame(),
                if (!state.isRunning)
                  Container(
                    color: Z.bg.withValues(alpha: .7),
                    alignment: Alignment.center,
                    child: const SizedBox(
                      width: 26,
                      height: 26,
                      child: CircularProgressIndicator(
                        strokeWidth: 2,
                        valueColor: AlwaysStoppedAnimation(Z.oliveSoft),
                      ),
                    ),
                  ),
                Positioned(
                  right: Z.s3,
                  bottom: Z.s3,
                  child: ZButton(
                    label: l.t('add_torch'),
                    kind: ZButtonKind.ghost,
                    size: ZButtonSize.sm,
                    icon: state.torchState == TorchState.on
                        ? Icons.flashlight_on_rounded
                        : Icons.flashlight_off_rounded,
                    onPressed: controller.toggleTorch,
                  ),
                ),
              ],
            );
          },
        ),
      ),
    );
  }
}

class _ViewfinderFrame extends StatelessWidget {
  const _ViewfinderFrame();

  @override
  Widget build(BuildContext context) => IgnorePointer(
        child: CustomPaint(painter: _FramePainter(), size: Size.infinite),
      );
}

class _FramePainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final side = size.shortestSide * .68;
    final rect = Rect.fromCenter(
      center: Offset(size.width / 2, size.height / 2),
      width: side,
      height: side,
    );
    // dim everything but the target square
    final overlay = Path.combine(
      PathOperation.difference,
      Path()..addRect(Offset.zero & size),
      Path()..addRRect(RRect.fromRectAndRadius(rect, const Radius.circular(Z.rLg))),
    );
    canvas.drawPath(overlay, Paint()..color = Z.bg.withValues(alpha: .55));

    // four olive corner ticks, no full frame — lighter, and it reads as aim
    final tick = Paint()
      ..color = Z.oliveSoft
      ..style = PaintingStyle.stroke
      ..strokeWidth = 3
      ..strokeCap = StrokeCap.square;
    const arm = 26.0;
    for (final (corner, dx, dy) in [
      (rect.topLeft, 1.0, 1.0),
      (rect.topRight, -1.0, 1.0),
      (rect.bottomLeft, 1.0, -1.0),
      (rect.bottomRight, -1.0, -1.0),
    ]) {
      canvas.drawLine(corner, corner.translate(arm * dx, 0), tick);
      canvas.drawLine(corner, corner.translate(0, arm * dy), tick);
    }
  }

  @override
  bool shouldRepaint(_FramePainter old) => false;
}

class _CameraProblem extends StatelessWidget {
  const _CameraProblem({required this.title, required this.body});
  final String title;
  final String body;

  @override
  Widget build(BuildContext context) => Container(
        color: Z.surface1,
        padding: const EdgeInsets.all(Z.s5),
        alignment: Alignment.center,
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.no_photography_outlined,
                size: 34, color: Z.inkMuted),
            const SizedBox(height: Z.s3),
            Text(title,
                style: Z.body.copyWith(fontWeight: FontWeight.w600),
                textAlign: TextAlign.center),
            const SizedBox(height: Z.s2),
            Text(body, style: Z.bodySoft, textAlign: TextAlign.center),
          ],
        ),
      );
}
