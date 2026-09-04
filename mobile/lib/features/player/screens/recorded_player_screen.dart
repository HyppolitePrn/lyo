import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../../core/theme/lyo_tokens.dart';
import '../../auth/providers/auth_notifier.dart';
import '../../track/utils/track_colors.dart';
import '../providers/recorded_player_notifier.dart';
import '../widgets/volume_control.dart';

String _fmtClock(Duration d) {
  final m = d.inMinutes.remainder(60).toString().padLeft(1, '0');
  final s = d.inSeconds.remainder(60).toString().padLeft(2, '0');
  return '$m:$s';
}

class RecordedPlayerScreen extends StatefulWidget {
  const RecordedPlayerScreen({required this.episodeId, super.key});

  // episodeId is a track ID.
  final String episodeId;

  @override
  State<RecordedPlayerScreen> createState() => _RecordedPlayerScreenState();
}

class _RecordedPlayerScreenState extends State<RecordedPlayerScreen> {
  @override
  void initState() {
    super.initState();
    // Uses the app-wide notifier so playback survives leaving this screen —
    // loadIfNeeded() no-ops if this track is already loaded/playing. Deferred
    // to after the frame because HomeScreen watches the same notifier, and
    // notifying it mid-build permanently freezes the listening element.
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) {
        return;
      }
      final token = context.read<AuthNotifier>().accessToken;
      context.read<RecordedPlayerNotifier>().loadIfNeeded(
        widget.episodeId,
        token,
      );
    });
  }

  @override
  Widget build(BuildContext context) => const _RecordedPlayerView();
}

class _RecordedPlayerView extends StatelessWidget {
  const _RecordedPlayerView();

  @override
  Widget build(BuildContext context) {
    final notifier = context.watch<RecordedPlayerNotifier>();
    final dark = Theme.of(context).brightness == Brightness.dark;
    final bg = dark ? lyoBgDark : lyoBgLight;
    final textPrimary = dark ? lyoTextDark : lyoTextLight;
    final textSub = dark ? lyoSubDark : lyoSubLight;

    return Scaffold(
      backgroundColor: bg,
      appBar: AppBar(
        backgroundColor: bg,
        elevation: 0,
        iconTheme: IconThemeData(color: textPrimary),
        title: Text(
          notifier.track?.title ?? 'Episode Player',
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
          child: notifier.isLoading
              ? Center(
                  child: Semantics(
                    label: 'Loading episode',
                    liveRegion: true,
                    child: const CircularProgressIndicator(
                      color: lyoAccent,
                      strokeWidth: 2,
                    ),
                  ),
                )
              : notifier.error != null
              ? Center(
                  child: Semantics(
                    liveRegion: true,
                    label: notifier.error!,
                    excludeSemantics: true,
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Icon(
                          Icons.headphones,
                          size: 64,
                          color: lyoAccent.withValues(alpha: 0.4),
                        ),
                        const SizedBox(height: lyoGapM),
                        Text(
                          notifier.error!,
                          style: TextStyle(color: textSub, fontSize: lyoBody1),
                        ),
                      ],
                    ),
                  ),
                )
              : Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    _Artwork(id: notifier.track!.id),
                    const SizedBox(height: lyoGapXXL),
                    Text(
                      notifier.track!.title,
                      style: TextStyle(
                        color: textPrimary,
                        fontSize: lyoH1,
                        fontWeight: FontWeight.w700,
                      ),
                      textAlign: TextAlign.center,
                    ),
                    const SizedBox(height: lyoGapS),
                    Text(
                      notifier.track!.artist?.isNotEmpty == true
                          ? notifier.track!.artist!
                          : 'Unknown artist',
                      style: TextStyle(color: textSub, fontSize: lyoBody2),
                      textAlign: TextAlign.center,
                    ),
                    const SizedBox(height: lyoGapXXL),
                    SliderTheme(
                      data: SliderTheme.of(context).copyWith(
                        trackHeight: 3,
                        thumbShape: const RoundSliderThumbShape(
                          enabledThumbRadius: 6,
                        ),
                      ),
                      child: Slider(
                        label: _fmtClock(notifier.position),
                        semanticFormatterCallback: (v) => _fmtClock(
                          Duration(milliseconds: v.round()),
                        ),
                        activeColor: lyoAccent,
                        inactiveColor: dark ? lyoBorderDark : lyoBorderLight,
                        min: 0,
                        max: notifier.duration.inMilliseconds > 0
                            ? notifier.duration.inMilliseconds.toDouble()
                            : 1,
                        value: notifier.position.inMilliseconds
                            .clamp(
                              0,
                              notifier.duration.inMilliseconds > 0
                                  ? notifier.duration.inMilliseconds
                                  : 1,
                            )
                            .toDouble(),
                        onChanged: (v) => context
                            .read<RecordedPlayerNotifier>()
                            .seek(Duration(milliseconds: v.round())),
                      ),
                    ),
                    Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 4),
                      child: Semantics(
                        label:
                            '${_fmtClock(notifier.position)} elapsed '
                            'of ${_fmtClock(notifier.duration)}',
                        excludeSemantics: true,
                        child: Row(
                          mainAxisAlignment: MainAxisAlignment.spaceBetween,
                          children: [
                            Text(
                              _fmtClock(notifier.position),
                              style: TextStyle(
                                color: textSub,
                                fontSize: lyoSmall,
                              ),
                            ),
                            Text(
                              _fmtClock(notifier.duration),
                              style: TextStyle(
                                color: textSub,
                                fontSize: lyoSmall,
                              ),
                            ),
                          ],
                        ),
                      ),
                    ),
                    const SizedBox(height: lyoGapM),
                    const VolumeControl(),
                    const SizedBox(height: lyoGapL),
                    Semantics(
                      button: true,
                      label: notifier.isPlaying ? 'Pause' : 'Play',
                      onTap: () => context
                          .read<RecordedPlayerNotifier>()
                          .togglePlayPause(),
                      child: GestureDetector(
                        onTap: () => context
                            .read<RecordedPlayerNotifier>()
                            .togglePlayPause(),
                        child: Container(
                          width: 72,
                          height: 72,
                          decoration: const BoxDecoration(
                            color: lyoAccent,
                            shape: BoxShape.circle,
                            boxShadow: [lyoCtaGlow],
                          ),
                          child: Icon(
                            notifier.isPlaying ? Icons.pause : Icons.play_arrow,
                            color: Colors.white,
                            size: 36,
                          ),
                        ),
                      ),
                    ),
                  ],
                ),
        ),
      ),
    );
  }
}

class _Artwork extends StatelessWidget {
  const _Artwork({required this.id});
  final String id;

  @override
  Widget build(BuildContext context) {
    final colors = trackColors(id);
    return ExcludeSemantics(
      child: Container(
        width: 220,
        height: 220,
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(lyoRadiusCard),
          gradient: LinearGradient(
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
            colors: colors,
          ),
          boxShadow: const [lyoArtworkShadow],
        ),
        child: const Icon(Icons.headphones, size: 80, color: Colors.white),
      ),
    );
  }
}
