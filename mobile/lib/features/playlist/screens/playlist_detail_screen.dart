import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../../core/theme/lyo_tokens.dart';
import '../../auth/providers/auth_notifier.dart';
import '../../home/widgets/lyo_artwork_tile.dart';
import '../../track/models/track_model.dart';
import '../../track/utils/track_colors.dart';
import '../models/playlist_model.dart';
import '../providers/playlist_notifier.dart';

class PlaylistDetailScreen extends StatefulWidget {
  const PlaylistDetailScreen({required this.playlistId, super.key});
  final String playlistId;

  @override
  State<PlaylistDetailScreen> createState() => _PlaylistDetailScreenState();
}

class _PlaylistDetailScreenState extends State<PlaylistDetailScreen> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final token = context.read<AuthNotifier>().accessToken;
      context.read<PlaylistNotifier>().loadOne(widget.playlistId, token: token);
    });
  }

  Future<void> _removeTrack(String trackId) async {
    final token = context.read<AuthNotifier>().accessToken;
    if (token == null) {
      return;
    }
    await context.read<PlaylistNotifier>().removeTrack(
          playlistId: widget.playlistId,
          trackId: trackId,
          token: token,
        );
  }

  Future<void> _deletePlaylist() async {
    final token = context.read<AuthNotifier>().accessToken;
    if (token == null) {
      return;
    }
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: const Text('Delete playlist?'),
        content: const Text('This cannot be undone.'),
        actions: [
          TextButton(
              onPressed: () => Navigator.of(dialogContext).pop(false),
              child: const Text('Cancel')),
          TextButton(
              onPressed: () => Navigator.of(dialogContext).pop(true),
              child: const Text('Delete')),
        ],
      ),
    );
    if (confirmed != true || !mounted) {
      return;
    }
    final ok = await context
        .read<PlaylistNotifier>()
        .delete(widget.playlistId, token);
    if (ok && mounted) {
      context.pop();
    }
  }

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    final bg = dark ? lyoBgDark : lyoBgLight;
    final textPrimary = dark ? lyoTextDark : lyoTextLight;
    final textSub = dark ? lyoSubDark : lyoSubLight;
    final notifier = context.watch<PlaylistNotifier>();
    final playlist = notifier.selected;

    return Scaffold(
      backgroundColor: bg,
      appBar: AppBar(
        backgroundColor: bg,
        elevation: 0,
        title: Text(playlist?.title ?? 'Playlist',
            style: TextStyle(
                color: textPrimary, fontSize: lyoH1, fontWeight: FontWeight.w700)),
        actions: [
          if (playlist != null)
            IconButton(
              icon: const Icon(Icons.delete_outline, color: lyoError),
              onPressed: _deletePlaylist,
            ),
        ],
      ),
      body: switch (notifier.status) {
        PlaylistStatus.loading || PlaylistStatus.idle =>
          const Center(child: CircularProgressIndicator(color: lyoAccent)),
        PlaylistStatus.error => Center(
            child: Text(notifier.error ?? 'Something went wrong',
                style: const TextStyle(color: lyoError))),
        PlaylistStatus.ready => _Body(
            playlist: playlist,
            textSub: textSub,
            textPrimary: textPrimary,
            onRemoveTrack: _removeTrack,
          ),
      },
    );
  }
}

class _Body extends StatelessWidget {
  const _Body({
    required this.playlist,
    required this.textSub,
    required this.textPrimary,
    required this.onRemoveTrack,
  });

  final Playlist? playlist;
  final Color textSub;
  final Color textPrimary;
  final ValueChanged<String> onRemoveTrack;

  @override
  Widget build(BuildContext context) {
    final tracks = playlist?.tracks ?? const <Track>[];
    if (tracks.isEmpty) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.music_off_outlined,
                size: 48, color: lyoAccent.withValues(alpha: 0.4)),
            const SizedBox(height: lyoGapM),
            Text('No tracks in this playlist yet',
                style: TextStyle(color: textSub, fontSize: lyoBody1)),
          ],
        ),
      );
    }
    return ListView.separated(
      padding: const EdgeInsets.fromLTRB(20, 16, 20, 100),
      itemCount: tracks.length,
      separatorBuilder: (_, _) => const SizedBox(height: 4),
      itemBuilder: (context, i) {
        final t = tracks[i];
        final colors = trackColors(t.id);
        return Padding(
          padding: const EdgeInsets.symmetric(vertical: 6),
          child: Row(
            children: [
              LyoArtworkTile(
                  size: 48, radius: 10, color1: colors[0], color2: colors[1]),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(t.title,
                        style: TextStyle(
                            fontSize: lyoBody2,
                            fontWeight: FontWeight.w600,
                            color: textPrimary),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis),
                    if (t.artist?.isNotEmpty == true)
                      Text(t.artist!,
                          style: TextStyle(fontSize: lyoBody2 - 2, color: textSub)),
                  ],
                ),
              ),
              IconButton(
                icon: const Icon(Icons.remove_circle_outline, color: lyoSubDark),
                onPressed: () => onRemoveTrack(t.id),
              ),
            ],
          ),
        );
      },
    );
  }
}
