import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../../core/api/api_client.dart';
import '../../../core/features/feature_flags_provider.dart';
import '../../../core/theme/lyo_tokens.dart';
import '../../auth/providers/auth_notifier.dart';
import '../../favorites/providers/favorites_notifier.dart';
import '../../favorites/widgets/favorite_button.dart';
import '../../playlist/widgets/add_to_playlist_sheet.dart';
import '../../track/models/track_model.dart';
import '../../track/services/track_service.dart';
import '../../track/utils/track_colors.dart';
import '../widgets/lyo_artwork_tile.dart';

String _fmtDuration(int seconds) {
  final m = seconds ~/ 60;
  if (m < 1) {
    return '${seconds}s';
  }
  return '$m min';
}

class BrowseTab extends StatefulWidget {
  const BrowseTab({super.key});

  @override
  State<BrowseTab> createState() => _BrowseTabState();
}

class _BrowseTabState extends State<BrowseTab> {
  static const _limit = 20;
  final _trackSvc = const TrackService(ApiClient());
  final _scrollController = ScrollController();

  final List<Track> _tracks = [];
  int _page = 1;
  bool _isLoading = false;
  bool _hasMore = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _scrollController.addListener(_onScroll);
    _load(reset: true);
  }

  @override
  void dispose() {
    _scrollController.removeListener(_onScroll);
    _scrollController.dispose();
    super.dispose();
  }

  void _onScroll() {
    if (!_isLoading &&
        _hasMore &&
        _scrollController.position.pixels >=
            _scrollController.position.maxScrollExtent - 200) {
      _load();
    }
  }

  Future<void> _load({bool reset = false}) async {
    setState(() {
      _isLoading = true;
      if (reset) {
        _error = null;
      }
    });

    final page = reset ? 1 : _page + 1;
    try {
      final token = context.read<AuthNotifier>().accessToken;
      final items =
          await _trackSvc.listTracks(page: page, limit: _limit, token: token);
      if (!mounted) {
        return;
      }
      setState(() {
        if (reset) {
          _tracks.clear();
        }
        _tracks.addAll(items);
        _page = page;
        _hasMore = items.length == _limit;
        _isLoading = false;
        _error = null;
      });
    } catch (_) {
      if (!mounted) {
        return;
      }
      setState(() {
        _isLoading = false;
        if (_tracks.isEmpty) {
          _error = 'Could not load tracks';
        }
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    final textSub = dark ? lyoSubDark : lyoSubLight;

    if (_error != null && _tracks.isEmpty) {
      return SafeArea(
        child: Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(Icons.grid_view_outlined,
                  size: 48, color: lyoAccent.withValues(alpha: 0.4)),
              const SizedBox(height: lyoGapM),
              Text(_error!, style: TextStyle(color: textSub, fontSize: lyoBody1)),
              const SizedBox(height: lyoGapM),
              TextButton(
                onPressed: () => _load(reset: true),
                child: const Text('Retry'),
              ),
            ],
          ),
        ),
      );
    }

    return SafeArea(
      child: RefreshIndicator(
        color: lyoAccent,
        onRefresh: () => _load(reset: true),
        child: _tracks.isEmpty
            ? ListView(
                children: [
                  SizedBox(
                    height: 400,
                    child: Center(
                      child: _isLoading
                          ? const CircularProgressIndicator(
                              color: lyoAccent, strokeWidth: 2)
                          : Column(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                Icon(Icons.grid_view_outlined,
                                    size: 48,
                                    color: lyoAccent.withValues(alpha: 0.4)),
                                const SizedBox(height: lyoGapM),
                                Text('No tracks yet',
                                    style: TextStyle(
                                        color: textSub, fontSize: lyoBody1)),
                              ],
                            ),
                    ),
                  ),
                ],
              )
            : ListView.separated(
                controller: _scrollController,
                padding: const EdgeInsets.fromLTRB(20, 16, 20, 100),
                itemCount: _tracks.length + (_hasMore ? 1 : 0),
                separatorBuilder: (_, _) =>
                    Divider(color: dark ? lyoBorderDark : lyoBorderLight, height: 1),
                itemBuilder: (context, i) {
                  if (i >= _tracks.length) {
                    return const Padding(
                      padding: EdgeInsets.symmetric(vertical: 20),
                      child: Center(
                        child: CircularProgressIndicator(
                            color: lyoAccent, strokeWidth: 2),
                      ),
                    );
                  }
                  final t = _tracks[i];
                  return _TrackTile(
                    track: t,
                    dark: dark,
                    onTap: () => context.push('/recorded-player/${t.id}'),
                  );
                },
              ),
      ),
    );
  }
}

class _TrackTile extends StatelessWidget {
  const _TrackTile({required this.track, required this.dark, required this.onTap});

  final Track track;
  final bool dark;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final textPrimary = dark ? lyoTextDark : lyoTextLight;
    final textSub = dark ? lyoSubDark : lyoSubLight;
    final colors = trackColors(track.id);
    final flags = context.watch<FeatureFlags>();
    final favoritesEnabled = flags.isEnabled('favorites');
    final favorites = favoritesEnabled ? context.watch<FavoritesNotifier>() : null;
    final playlistsEnabled = flags.isEnabled('playlists');

    return GestureDetector(
      onTap: onTap,
      behavior: HitTestBehavior.opaque,
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 10),
        child: Row(
          children: [
            LyoArtworkTile(size: 52, radius: 10, color1: colors[0], color2: colors[1]),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    track.title,
                    style: TextStyle(
                      fontSize: lyoBody2,
                      fontWeight: FontWeight.w600,
                      color: textPrimary,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                  const SizedBox(height: 3),
                  Text(
                    [
                      if (track.artist?.isNotEmpty == true) track.artist,
                      _fmtDuration(track.durationSeconds),
                    ].join(' · '),
                    style: TextStyle(fontSize: lyoBody2 - 2, color: textSub),
                  ),
                ],
              ),
            ),
            if (playlistsEnabled)
              IconButton(
                icon: const Icon(Icons.playlist_add, size: 22, color: lyoSubDark),
                onPressed: () => showAddToPlaylistSheet(context, track.id),
              ),
            if (favorites != null)
              FavoriteButton(
                isFavorited: favorites.isTrackFavorited(track.id),
                onTap: () {
                  final token = context.read<AuthNotifier>().accessToken;
                  if (token != null) {
                    favorites.toggleTrack(track.id, token);
                  }
                },
              ),
            const SizedBox(width: 4),
            const Icon(Icons.play_circle_outline, size: 22, color: lyoAccent),
          ],
        ),
      ),
    );
  }
}
