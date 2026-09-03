import 'package:flutter/material.dart';

import '../models/home_models.dart';
import 'lyo_artwork_tile.dart';

String _fmtElapsed(Duration d) {
  final m = d.inMinutes.remainder(60).toString();
  final s = d.inSeconds.remainder(60).toString().padLeft(2, '0');
  return '$m:$s';
}

class MiniPlayer extends StatelessWidget {
  const MiniPlayer({
    required this.state,
    required this.onTap,
    required this.onToggle,
    required this.onDismiss,
    super.key,
  });

  final MiniPlayerState state;
  final VoidCallback onTap;
  final VoidCallback onToggle;
  final VoidCallback onDismiss;

  @override
  Widget build(BuildContext context) {
    final progress = state.duration.inMilliseconds > 0
        ? (state.position.inMilliseconds / state.duration.inMilliseconds).clamp(
            0.0,
            1.0,
          )
        : 0.0;

    final elapsed = state.duration > Duration.zero
        ? ', ${_fmtElapsed(state.position)} of ${_fmtElapsed(state.duration)}'
        : '';

    return Semantics(
      container: true,
      label: 'Mini player',
      child: GestureDetector(
        onTap: onTap,
        behavior: HitTestBehavior.opaque,
        child: ClipRRect(
          borderRadius: BorderRadius.circular(16),
          child: Container(
            decoration: BoxDecoration(
              color: const Color(0xFF2A2724),
              boxShadow: [
                BoxShadow(
                  color: Colors.black.withValues(alpha: 0.4),
                  blurRadius: 32,
                  offset: const Offset(0, 8),
                ),
              ],
            ),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Padding(
                  padding: const EdgeInsets.fromLTRB(14, 10, 14, 8),
                  child: Row(
                    children: [
                      LyoArtworkTile(
                        size: 42,
                        radius: 8,
                        color1: state.artColor1,
                        color2: state.artColor2,
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Semantics(
                          button: true,
                          label:
                              '${state.trackTitle}, ${state.showName}$elapsed',
                          hint: 'Open the full player',
                          excludeSemantics: true,
                          onTap: onTap,
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              Text(
                                state.trackTitle,
                                style: const TextStyle(
                                  fontSize: 13,
                                  fontWeight: FontWeight.w600,
                                  color: Colors.white,
                                ),
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                              ),
                              Row(
                                children: [
                                  Flexible(
                                    child: Text(
                                      state.showName,
                                      style: TextStyle(
                                        fontSize: 11,
                                        color: Colors.white.withValues(
                                          alpha: 0.5,
                                        ),
                                      ),
                                      maxLines: 1,
                                      overflow: TextOverflow.ellipsis,
                                    ),
                                  ),
                                  if (state.duration > Duration.zero) ...[
                                    const SizedBox(width: 6),
                                    Text(
                                      '· ${_fmtElapsed(state.position)} / ${_fmtElapsed(state.duration)}',
                                      style: TextStyle(
                                        fontSize: 11,
                                        color: Colors.white.withValues(
                                          alpha: 0.4,
                                        ),
                                      ),
                                    ),
                                  ],
                                ],
                              ),
                            ],
                          ),
                        ),
                      ),
                      IconButton(
                        tooltip: state.isPlaying ? 'Pause' : 'Play',
                        icon: Icon(
                          state.isPlaying ? Icons.pause : Icons.play_arrow,
                          color: Colors.white,
                          size: 22,
                        ),
                        onPressed: onToggle,
                      ),
                      IconButton(
                        tooltip: 'Close mini player',
                        padding: EdgeInsets.zero,
                        constraints: const BoxConstraints(
                          minWidth: 32,
                          minHeight: 32,
                        ),
                        icon: Icon(
                          Icons.close,
                          color: Colors.white.withValues(alpha: 0.4),
                          size: 16,
                        ),
                        onPressed: onDismiss,
                      ),
                    ],
                  ),
                ),
                ExcludeSemantics(
                  child: SizedBox(
                    height: 3,
                    child: LinearProgressIndicator(
                      value: progress,
                      backgroundColor: Colors.white.withValues(alpha: 0.08),
                      valueColor: const AlwaysStoppedAnimation(
                        Color(0xFFE8B84A),
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
