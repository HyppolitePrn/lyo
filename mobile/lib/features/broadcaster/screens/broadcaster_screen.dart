import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/features/feature_flags_provider.dart';
import '../../../core/theme/lyo_tokens.dart';
import '../../auth/widgets/auth_error_banner.dart';
import '../../auth/widgets/lyo_text_field.dart';
import '../providers/broadcaster_notifier.dart';

class BroadcasterScreen extends ConsumerStatefulWidget {
  const BroadcasterScreen({super.key});

  @override
  ConsumerState<BroadcasterScreen> createState() => _BroadcasterScreenState();
}

class _BroadcasterScreenState extends ConsumerState<BroadcasterScreen> {
  final _titleCtrl = TextEditingController();
  final _descCtrl = TextEditingController();
  final _formKey = GlobalKey<FormState>();

  // Timer for live duration display
  int _liveSeconds = 0;
  DateTime? _liveStartedAt;

  @override
  void dispose() {
    _titleCtrl.dispose();
    _descCtrl.dispose();
    super.dispose();
  }

  Future<void> _goLive() async {
    if (!(_formKey.currentState?.validate() ?? false)) return;
    await ref.read(broadcasterNotifierProvider.notifier).startBroadcast(
          _titleCtrl.text.trim(),
          _descCtrl.text.trim().isEmpty ? null : _descCtrl.text.trim(),
        );
    _liveStartedAt = DateTime.now();
    _tickTimer();
  }

  Future<void> _endStream() async {
    await ref.read(broadcasterNotifierProvider.notifier).stopBroadcast();
    _liveStartedAt = null;
    setState(() => _liveSeconds = 0);
  }

  void _tickTimer() {
    Future.delayed(const Duration(seconds: 1), () {
      if (!mounted) return;
      final status = ref.read(broadcasterNotifierProvider).status;
      if (status == BroadcasterStatus.live && _liveStartedAt != null) {
        setState(() {
          _liveSeconds =
              DateTime.now().difference(_liveStartedAt!).inSeconds;
        });
        _tickTimer();
      }
    });
  }

  String _formatDuration(int secs) {
    final m = secs ~/ 60;
    final s = secs % 60;
    return '${m.toString().padLeft(2, '0')}:${s.toString().padLeft(2, '0')}';
  }

  @override
  Widget build(BuildContext context) {
    final flags = ref.watch(featureFlagsProvider);
    if (!flags.isEnabled('live_streaming')) return const SizedBox.shrink();

    final broadcaster = ref.watch(broadcasterNotifierProvider);
    final dark = Theme.of(context).brightness == Brightness.dark;
    final bg = dark ? lyoBgDark : lyoBgLight;
    final textPrimary = dark ? lyoTextDark : lyoTextLight;
    final textSub = dark ? lyoSubDark : lyoSubLight;
    final isLive = broadcaster.status == BroadcasterStatus.live;
    final isBusy = broadcaster.status == BroadcasterStatus.creating ||
        broadcaster.status == BroadcasterStatus.ending;

    return Scaffold(
      backgroundColor: bg,
      appBar: AppBar(
        backgroundColor: bg,
        elevation: 0,
        iconTheme: IconThemeData(color: textPrimary),
        title: Text(
          'Go Live',
          style: TextStyle(
            color: textPrimary,
            fontSize: lyoH1,
            fontWeight: FontWeight.w700,
          ),
        ),
      ),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.symmetric(
              horizontal: lyoPadHMain, vertical: lyoGapXL),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              if (broadcaster.error != null)
                AuthErrorBanner(message: broadcaster.error!),
              if (isLive) ...[
                _LiveBadge(duration: _formatDuration(_liveSeconds)),
                const SizedBox(height: lyoGapXXL),
              ],
              Form(
                key: _formKey,
                child: Column(
                  children: [
                    LyoTextField(
                      controller: _titleCtrl,
                      hint: 'Stream title',
                      label: 'Title',
                      textInputAction: TextInputAction.next,
                      validator: (v) =>
                          (v == null || v.trim().isEmpty) ? 'Title is required' : null,
                    ),
                    const SizedBox(height: lyoGapL),
                    LyoTextField(
                      controller: _descCtrl,
                      hint: 'What are you broadcasting? (optional)',
                      label: 'Description',
                      textInputAction: TextInputAction.done,
                    ),
                  ],
                ),
              ),
              const SizedBox(height: lyoGapXXXL),
              _ActionButton(
                isLive: isLive,
                isBusy: isBusy,
                onGoLive: _goLive,
                onEnd: _endStream,
                textSub: textSub,
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _LiveBadge extends StatelessWidget {
  const _LiveBadge({required this.duration});
  final String duration;

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        Container(
          width: 10,
          height: 10,
          decoration: const BoxDecoration(
            color: Colors.redAccent,
            shape: BoxShape.circle,
          ),
        ),
        const SizedBox(width: lyoGapS),
        const Text(
          'LIVE',
          style: TextStyle(
            color: Colors.redAccent,
            fontSize: lyoCaption,
            fontWeight: FontWeight.w700,
            letterSpacing: 1.5,
          ),
        ),
        const SizedBox(width: lyoGapL),
        Text(
          duration,
          style: const TextStyle(
            color: lyoAccent,
            fontSize: lyoBody1,
            fontWeight: FontWeight.w600,
          ),
        ),
      ],
    );
  }
}

class _ActionButton extends StatelessWidget {
  const _ActionButton({
    required this.isLive,
    required this.isBusy,
    required this.onGoLive,
    required this.onEnd,
    required this.textSub,
  });

  final bool isLive;
  final bool isBusy;
  final VoidCallback onGoLive;
  final VoidCallback onEnd;
  final Color textSub;

  @override
  Widget build(BuildContext context) {
    if (isBusy) {
      return const Center(
        child: CircularProgressIndicator(color: lyoAccent),
      );
    }

    if (isLive) {
      return OutlinedButton(
        onPressed: onEnd,
        style: OutlinedButton.styleFrom(
          side: const BorderSide(color: lyoError),
          foregroundColor: lyoError,
          padding: const EdgeInsets.symmetric(vertical: lyoGapM),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(lyoRadiusBtn),
          ),
        ),
        child: const Text(
          'End Stream',
          style: TextStyle(fontSize: lyoBody1, fontWeight: FontWeight.w600),
        ),
      );
    }

    return ElevatedButton(
      onPressed: onGoLive,
      style: ElevatedButton.styleFrom(
        backgroundColor: lyoAccent,
        foregroundColor: Colors.white,
        padding: const EdgeInsets.symmetric(vertical: lyoGapM),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(lyoRadiusBtn),
        ),
        elevation: 0,
      ),
      child: const Text(
        'Go Live',
        style: TextStyle(
            fontSize: lyoBody1, fontWeight: FontWeight.w700, letterSpacing: 0.5),
      ),
    );
  }
}
