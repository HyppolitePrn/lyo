import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/lyo_tokens.dart';
import '../../auth/providers/auth_notifier.dart';
import '../models/stream_model.dart';
import '../services/player_service.dart';

class StreamListScreen extends ConsumerStatefulWidget {
  const StreamListScreen({super.key});

  @override
  ConsumerState<StreamListScreen> createState() => _StreamListScreenState();
}

class _StreamListScreenState extends ConsumerState<StreamListScreen> {
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
      final token = ref.read(authNotifierProvider).accessToken ?? '';
      final svc = PlayerService(ref.read(apiClientProvider));
      final streams = await svc.listLive(token);
      if (mounted) setState(() => _streams = streams);
    } catch (e) {
      if (mounted) setState(() => _error = e.toString());
    } finally {
      if (mounted) setState(() => _isLoading = false);
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
      return const Center(
        child: CircularProgressIndicator(color: lyoAccent),
      );
    }
    if (_error != null) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(lyoGapL),
          child: Text(
            _error!,
            style: const TextStyle(color: lyoError),
            textAlign: TextAlign.center,
          ),
        ),
      );
    }
    if (_streams.isEmpty) {
      return Center(
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
            Icon(Icons.chevron_right, color: textSub),
          ],
        ),
      ),
    );
  }
}
