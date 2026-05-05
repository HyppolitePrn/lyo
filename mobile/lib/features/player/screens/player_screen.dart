import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/theme/lyo_tokens.dart';
import '../providers/player_notifier.dart';

class PlayerScreen extends ConsumerStatefulWidget {
  const PlayerScreen({super.key, required this.streamId});

  final String streamId;

  @override
  ConsumerState<PlayerScreen> createState() => _PlayerScreenState();
}

class _PlayerScreenState extends ConsumerState<PlayerScreen> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      ref.read(playerNotifierProvider.notifier).connect(widget.streamId);
    });
  }

  @override
  void dispose() {
    ref.read(playerNotifierProvider.notifier).disconnect();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final player = ref.watch(playerNotifierProvider);
    final dark = Theme.of(context).brightness == Brightness.dark;
    final bg = dark ? lyoBgDark : lyoBgLight;
    final surface = dark ? lyoSurfaceDark : lyoSurfaceLight;
    final textPrimary = dark ? lyoTextDark : lyoTextLight;
    final textSub = dark ? lyoSubDark : lyoSubLight;

    return Scaffold(
      backgroundColor: bg,
      appBar: AppBar(
        backgroundColor: bg,
        elevation: 0,
        iconTheme: IconThemeData(color: textPrimary),
        title: Text(
          player.stream?.title ?? 'Live Stream',
          style: TextStyle(
            color: textPrimary,
            fontSize: lyoH1,
            fontWeight: FontWeight.w700,
          ),
        ),
      ),
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: lyoPadHMain),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              _ArtworkPlaceholder(surface: surface),
              const SizedBox(height: lyoGapXXL),
              if (player.stream != null) ...[
                Text(
                  player.stream!.title,
                  style: TextStyle(
                    color: textPrimary,
                    fontSize: lyoH1,
                    fontWeight: FontWeight.w700,
                  ),
                  textAlign: TextAlign.center,
                ),
                const SizedBox(height: lyoGapS),
                Text(
                  player.stream!.description ?? 'Live broadcast',
                  style: TextStyle(color: textSub, fontSize: lyoBody2),
                  textAlign: TextAlign.center,
                ),
              ],
              const SizedBox(height: lyoGapXXXL),
              _StatusIndicator(status: player.status),
              const SizedBox(height: lyoGapXL),
              _ControlButton(
                status: player.status,
                onConnect: () => ref
                    .read(playerNotifierProvider.notifier)
                    .connect(widget.streamId),
                onDisconnect: () =>
                    ref.read(playerNotifierProvider.notifier).disconnect(),
              ),
              if (player.error != null) ...[
                const SizedBox(height: lyoGapM),
                Text(
                  player.error!,
                  style: const TextStyle(color: lyoError, fontSize: lyoCaption),
                  textAlign: TextAlign.center,
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

class _ArtworkPlaceholder extends StatelessWidget {
  const _ArtworkPlaceholder({required this.surface});
  final Color surface;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 220,
      height: 220,
      decoration: BoxDecoration(
        color: surface,
        borderRadius: BorderRadius.circular(lyoRadiusCard),
        boxShadow: const [lyoArtworkShadow],
      ),
      child: const Icon(Icons.radio, size: 80, color: lyoAccent),
    );
  }
}

class _StatusIndicator extends StatelessWidget {
  const _StatusIndicator({required this.status});
  final PlayerStatus status;

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    final textSub = dark ? lyoSubDark : lyoSubLight;

    return Row(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        if (status == PlayerStatus.connecting ||
            status == PlayerStatus.playing) ...[
          Container(
            width: 8,
            height: 8,
            decoration: const BoxDecoration(
              color: Colors.redAccent,
              shape: BoxShape.circle,
            ),
          ),
          const SizedBox(width: lyoGapS),
        ],
        Text(
          switch (status) {
            PlayerStatus.idle => 'Tap play to listen',
            PlayerStatus.connecting => 'Connecting…',
            PlayerStatus.playing => 'LIVE',
            PlayerStatus.error => 'Disconnected',
          },
          style: TextStyle(
            color: status == PlayerStatus.playing ? Colors.redAccent : textSub,
            fontSize: lyoCaption,
            fontWeight: FontWeight.w600,
            letterSpacing: 1.2,
          ),
        ),
      ],
    );
  }
}

class _ControlButton extends StatelessWidget {
  const _ControlButton({
    required this.status,
    required this.onConnect,
    required this.onDisconnect,
  });

  final PlayerStatus status;
  final VoidCallback onConnect;
  final VoidCallback onDisconnect;

  @override
  Widget build(BuildContext context) {
    final isActive = status == PlayerStatus.playing ||
        status == PlayerStatus.connecting;

    return GestureDetector(
      onTap: isActive ? onDisconnect : onConnect,
      child: Container(
        width: 72,
        height: 72,
        decoration: BoxDecoration(
          color: lyoAccent,
          shape: BoxShape.circle,
          boxShadow: const [lyoCtaGlow],
        ),
        child: status == PlayerStatus.connecting
            ? const Padding(
                padding: EdgeInsets.all(20),
                child: CircularProgressIndicator(
                    color: Colors.white, strokeWidth: 2),
              )
            : Icon(
                isActive ? Icons.stop : Icons.play_arrow,
                color: Colors.white,
                size: 36,
              ),
      ),
    );
  }
}
