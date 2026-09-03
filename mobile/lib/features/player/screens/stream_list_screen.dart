import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../../core/api/api_client.dart';
import '../../../core/features/feature_flags_provider.dart';
import '../../../core/theme/lyo_tokens.dart';
import '../../auth/providers/auth_notifier.dart';
import '../../favorites/providers/favorites_notifier.dart';
import '../../favorites/widgets/favorite_button.dart';
import '../models/stream_model.dart';
import '../services/player_service.dart';

class StreamListScreen extends StatefulWidget {
  const StreamListScreen({super.key});

  @override
  State<StreamListScreen> createState() => _StreamListScreenState();
}

class _StreamListScreenState extends State<StreamListScreen> {
  List<LiveStream> _streams = [];
  bool _isLoading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _isLoading = true;
      _error = null;
    });
    try {
      final token = context.read<AuthNotifier>().accessToken ?? '';
      const svc = PlayerService(ApiClient());
      final streams = await svc.listLive(token);
      if (mounted) {
        setState(() => _streams = streams);
      }
    } catch (e) {
      if (mounted) {
        setState(() => _error = e.toString());
      }
    } finally {
      if (mounted) {
        setState(() => _isLoading = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    final bg = dark ? lyoBgDark : lyoBgLight;
    final textPrimary = dark ? lyoTextDark : lyoTextLight;

    return Scaffold(
      backgroundColor: bg,
      appBar: AppBar(
        backgroundColor: bg,
        elevation: 0,
        title: Text(
          'Live Streams',
          style: TextStyle(
            color: textPrimary,
            fontSize: lyoH1,
            fontWeight: FontWeight.w700,
          ),
        ),
        actions: [
          IconButton(
            tooltip: 'Refresh the list',
            icon: const Icon(Icons.refresh, color: lyoAccent),
            onPressed: _load,
          ),
        ],
      ),
      body: RefreshIndicator(
        color: lyoAccent,
        onRefresh: _load,
        child: _buildBody(context),
      ),
    );
  }

  Widget _buildBody(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    final surface = dark ? lyoSurfaceDark : lyoSurfaceLight;
    final textPrimary = dark ? lyoTextDark : lyoTextLight;
    final textSub = dark ? lyoSubDark : lyoSubLight;

    if (_isLoading) {
      return Center(
        child: Semantics(
          label: 'Loading live streams',
          liveRegion: true,
          child: const CircularProgressIndicator(color: lyoAccent),
        ),
      );
    }
    if (_error != null) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(lyoGapL),
          child: Semantics(
            liveRegion: true,
            child: Text(
              _error!,
              style: const TextStyle(color: lyoError),
              textAlign: TextAlign.center,
            ),
          ),
        ),
      );
    }
    if (_streams.isEmpty) {
      return Center(
        child: Semantics(
          label: 'No live streams right now.',
          excludeSemantics: true,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(
                Icons.radio,
                size: 64,
                color: lyoAccent.withValues(alpha: 0.4),
              ),
              const SizedBox(height: lyoGapM),
              Text(
                'No live streams right now.',
                style: TextStyle(
                  color: lyoAccent.withValues(alpha: 0.7),
                  fontSize: lyoBody1,
                ),
              ),
            ],
          ),
        ),
      );
    }
    return ListView.separated(
      padding: const EdgeInsets.all(lyoGapM),
      itemCount: _streams.length,
      separatorBuilder: (_, _) => const SizedBox(height: lyoGapS),
      itemBuilder: (context, i) => _StreamTile(
        stream: _streams[i],
        surface: surface,
        textPrimary: textPrimary,
        textSub: textSub,
      ),
    );
  }
}

class _StreamTile extends StatelessWidget {
  const _StreamTile({
    required this.stream,
    required this.surface,
    required this.textPrimary,
    required this.textSub,
  });

  final LiveStream stream;
  final Color surface;
  final Color textPrimary;
  final Color textSub;

  @override
  Widget build(BuildContext context) {
    final favoritesEnabled = context.watch<FeatureFlags>().isEnabled(
      'favorites',
    );
    final favorites = favoritesEnabled
        ? context.watch<FavoritesNotifier>()
        : null;

    return GestureDetector(
      onTap: () => context.push('/player/${stream.id}'),
      child: Container(
        padding: const EdgeInsets.all(lyoGapM),
        decoration: BoxDecoration(
          color: surface,
          borderRadius: BorderRadius.circular(lyoRadiusCard),
        ),
        child: Row(
          children: [
            Container(
              width: 10,
              height: 10,
              decoration: const BoxDecoration(
                color: Colors.redAccent,
                shape: BoxShape.circle,
              ),
            ),
            const SizedBox(width: lyoGapM),
            Expanded(
              child: Semantics(
                button: true,
                label: stream.description?.isNotEmpty == true
                    ? 'Live: ${stream.title}, ${stream.description}'
                    : 'Live: ${stream.title}',
                hint: 'Open this live stream',
                excludeSemantics: true,
                onTap: () => context.push('/player/${stream.id}'),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      stream.title,
                      style: TextStyle(
                        color: textPrimary,
                        fontSize: lyoBody1,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    if (stream.description != null) ...[
                      const SizedBox(height: 2),
                      Text(
                        stream.description!,
                        style: TextStyle(color: textSub, fontSize: lyoCaption),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                    ],
                  ],
                ),
              ),
            ),
            if (favorites != null)
              FavoriteButton(
                isFavorited: favorites.isStreamFavorited(stream.id),
                onTap: () {
                  final token = context.read<AuthNotifier>().accessToken;
                  if (token != null) {
                    favorites.toggleStream(stream.id, token);
                  }
                },
              ),
            ExcludeSemantics(
              child: Icon(Icons.chevron_right, color: textSub),
            ),
          ],
        ),
      ),
    );
  }
}
