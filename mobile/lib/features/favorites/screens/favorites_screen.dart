import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../../core/theme/lyo_tokens.dart';
import '../../auth/providers/auth_notifier.dart';
import '../../home/widgets/lyo_artwork_tile.dart';
import '../../track/utils/track_colors.dart';
import '../providers/favorites_notifier.dart';

class FavoritesScreen extends StatefulWidget {
  const FavoritesScreen({super.key});

  @override
  State<FavoritesScreen> createState() => _FavoritesScreenState();
}

class _FavoritesScreenState extends State<FavoritesScreen> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final token = context.read<AuthNotifier>().accessToken;
      if (token != null) {
        context.read<FavoritesNotifier>().load(token);
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    final bg = dark ? lyoBgDark : lyoBgLight;
    final textPrimary = dark ? lyoTextDark : lyoTextLight;
    final textSub = dark ? lyoSubDark : lyoSubLight;
    final notifier = context.watch<FavoritesNotifier>();

    return DefaultTabController(
      length: 3,
      child: Scaffold(
        backgroundColor: bg,
        appBar: AppBar(
          backgroundColor: bg,
          elevation: 0,
          title: Text(
            'Favorites',
            style: TextStyle(
              color: textPrimary,
              fontSize: lyoH1,
              fontWeight: FontWeight.w700,
            ),
          ),
          bottom: TabBar(
            indicatorColor: lyoAccent,
            labelColor: lyoAccent,
            unselectedLabelColor: textSub,
            tabs: const [
              Tab(text: 'Tracks'),
              Tab(text: 'Streams'),
              Tab(text: 'Playlists'),
            ],
          ),
        ),
        body: switch (notifier.status) {
          FavoritesStatus.loading || FavoritesStatus.idle => Center(
            child: Semantics(
              label: 'Loading favorites',
              liveRegion: true,
              child: const CircularProgressIndicator(color: lyoAccent),
            ),
          ),
          FavoritesStatus.error => Center(
            child: Semantics(
              liveRegion: true,
              child: Text(
                notifier.error ?? 'Something went wrong',
                style: const TextStyle(color: lyoError),
              ),
            ),
          ),
          FavoritesStatus.ready => TabBarView(
            children: [
              _TrackList(textPrimary: textPrimary, textSub: textSub),
              _StreamList(textPrimary: textPrimary, textSub: textSub),
              _PlaylistList(textPrimary: textPrimary, textSub: textSub),
            ],
          ),
        },
      ),
    );
  }
}

class _EmptyState extends StatelessWidget {
  const _EmptyState({
    required this.icon,
    required this.label,
    required this.textSub,
  });
  final IconData icon;
  final String label;
  final Color textSub;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Semantics(
        label: label,
        excludeSemantics: true,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 48, color: lyoAccent.withValues(alpha: 0.4)),
            const SizedBox(height: lyoGapM),
            Text(
              label,
              style: TextStyle(color: textSub, fontSize: lyoBody1),
            ),
          ],
        ),
      ),
    );
  }
}

class _TrackList extends StatelessWidget {
  const _TrackList({required this.textPrimary, required this.textSub});
  final Color textPrimary;
  final Color textSub;

  @override
  Widget build(BuildContext context) {
    final tracks = context.watch<FavoritesNotifier>().tracks;
    if (tracks.isEmpty) {
      return _EmptyState(
        icon: Icons.music_off_outlined,
        label: 'No favorite tracks yet',
        textSub: textSub,
      );
    }
    return ListView.separated(
      padding: const EdgeInsets.all(lyoGapM),
      itemCount: tracks.length,
      separatorBuilder: (_, _) => const SizedBox(height: lyoGapS),
      itemBuilder: (context, i) {
        final t = tracks[i];
        final colors = trackColors(t.id);
        return Semantics(
          button: true,
          label: t.title,
          hint: 'Play this track',
          excludeSemantics: true,
          onTap: () => context.push('/recorded-player/${t.id}'),
          child: GestureDetector(
            onTap: () => context.push('/recorded-player/${t.id}'),
            behavior: HitTestBehavior.opaque,
            child: Row(
              children: [
                LyoArtworkTile(
                  size: 48,
                  radius: 10,
                  color1: colors[0],
                  color2: colors[1],
                ),
                const SizedBox(width: lyoGapM),
                Expanded(
                  child: Text(
                    t.title,
                    style: TextStyle(
                      color: textPrimary,
                      fontWeight: FontWeight.w600,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
              ],
            ),
          ),
        );
      },
    );
  }
}

class _StreamList extends StatelessWidget {
  const _StreamList({required this.textPrimary, required this.textSub});
  final Color textPrimary;
  final Color textSub;

  @override
  Widget build(BuildContext context) {
    final streams = context.watch<FavoritesNotifier>().streams;
    if (streams.isEmpty) {
      return _EmptyState(
        icon: Icons.radio,
        label: 'No favorite streams yet',
        textSub: textSub,
      );
    }
    return ListView.separated(
      padding: const EdgeInsets.all(lyoGapM),
      itemCount: streams.length,
      separatorBuilder: (_, _) => const SizedBox(height: lyoGapS),
      itemBuilder: (context, i) {
        final s = streams[i];
        return Semantics(
          button: true,
          label: s.title,
          hint: 'Open this live stream',
          excludeSemantics: true,
          onTap: () => context.push('/player/${s.id}'),
          child: GestureDetector(
            onTap: () => context.push('/player/${s.id}'),
            behavior: HitTestBehavior.opaque,
            child: Row(
              children: [
                const Icon(Icons.podcasts, color: lyoAccent),
                const SizedBox(width: lyoGapM),
                Expanded(
                  child: Text(
                    s.title,
                    style: TextStyle(
                      color: textPrimary,
                      fontWeight: FontWeight.w600,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
              ],
            ),
          ),
        );
      },
    );
  }
}

class _PlaylistList extends StatelessWidget {
  const _PlaylistList({required this.textPrimary, required this.textSub});
  final Color textPrimary;
  final Color textSub;

  @override
  Widget build(BuildContext context) {
    final playlists = context.watch<FavoritesNotifier>().playlists;
    if (playlists.isEmpty) {
      return _EmptyState(
        icon: Icons.queue_music_outlined,
        label: 'No favorite playlists yet',
        textSub: textSub,
      );
    }
    return ListView.separated(
      padding: const EdgeInsets.all(lyoGapM),
      itemCount: playlists.length,
      separatorBuilder: (_, _) => const SizedBox(height: lyoGapS),
      itemBuilder: (context, i) {
        final p = playlists[i];
        return Semantics(
          button: true,
          label: p.title,
          hint: 'Open this playlist',
          excludeSemantics: true,
          onTap: () => context.push('/playlists/${p.id}'),
          child: GestureDetector(
            onTap: () => context.push('/playlists/${p.id}'),
            behavior: HitTestBehavior.opaque,
            child: Row(
              children: [
                const Icon(Icons.queue_music, color: lyoAccent),
                const SizedBox(width: lyoGapM),
                Expanded(
                  child: Text(
                    p.title,
                    style: TextStyle(
                      color: textPrimary,
                      fontWeight: FontWeight.w600,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
              ],
            ),
          ),
        );
      },
    );
  }
}
