import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../../core/theme/lyo_tokens.dart';
import '../../auth/providers/auth_notifier.dart';
import '../models/playlist_model.dart';
import '../providers/playlist_notifier.dart';

class PlaylistsScreen extends StatefulWidget {
  const PlaylistsScreen({super.key});

  @override
  State<PlaylistsScreen> createState() => _PlaylistsScreenState();
}

class _PlaylistsScreenState extends State<PlaylistsScreen> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final token = context.read<AuthNotifier>().accessToken;
      if (token != null) {
        context.read<PlaylistNotifier>().loadMine(token);
      }
    });
  }

  Future<void> _createPlaylist() async {
    final titleController = TextEditingController();
    final descController = TextEditingController();
    var isPublic = false;

    final created = await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      backgroundColor: Theme.of(context).brightness == Brightness.dark
          ? lyoSurfaceDark
          : lyoSurfaceLight,
      builder: (sheetContext) {
        return StatefulBuilder(
          builder: (sheetContext, setSheetState) {
            return Padding(
              padding: EdgeInsets.only(
                left: lyoGapL,
                right: lyoGapL,
                top: lyoGapL,
                bottom: MediaQuery.of(sheetContext).viewInsets.bottom + lyoGapL,
              ),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  const Text(
                    'New playlist',
                    style: TextStyle(
                        fontSize: lyoH2, fontWeight: FontWeight.w700),
                  ),
                  const SizedBox(height: lyoGapL),
                  TextField(
                    controller: titleController,
                    decoration: const InputDecoration(labelText: 'Title'),
                    autofocus: true,
                  ),
                  const SizedBox(height: lyoGapM),
                  TextField(
                    controller: descController,
                    decoration:
                        const InputDecoration(labelText: 'Description (optional)'),
                  ),
                  const SizedBox(height: lyoGapM),
                  SwitchListTile(
                    contentPadding: EdgeInsets.zero,
                    title: const Text('Public'),
                    value: isPublic,
                    onChanged: (v) => setSheetState(() => isPublic = v),
                  ),
                  const SizedBox(height: lyoGapM),
                  ElevatedButton(
                    onPressed: titleController.text.trim().isEmpty
                        ? null
                        : () => Navigator.of(sheetContext).pop(true),
                    child: const Text('Create'),
                  ),
                ],
              ),
            );
          },
        );
      },
    );

    if (created != true || !mounted) {
      return;
    }
    final token = context.read<AuthNotifier>().accessToken;
    if (token == null) {
      return;
    }
    await context.read<PlaylistNotifier>().create(
          title: titleController.text.trim(),
          token: token,
          description:
              descController.text.trim().isEmpty ? null : descController.text.trim(),
          isPublic: isPublic,
        );
  }

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    final bg = dark ? lyoBgDark : lyoBgLight;
    final textPrimary = dark ? lyoTextDark : lyoTextLight;
    final textSub = dark ? lyoSubDark : lyoSubLight;
    final surface = dark ? lyoSurfaceDark : lyoSurfaceLight;
    final notifier = context.watch<PlaylistNotifier>();

    return Scaffold(
      backgroundColor: bg,
      appBar: AppBar(
        backgroundColor: bg,
        elevation: 0,
        title: Text('My Playlists',
            style: TextStyle(
                color: textPrimary, fontSize: lyoH1, fontWeight: FontWeight.w700)),
        actions: [
          IconButton(
            icon: const Icon(Icons.add, color: lyoAccent),
            onPressed: _createPlaylist,
          ),
        ],
      ),
      body: switch (notifier.status) {
        PlaylistStatus.loading || PlaylistStatus.idle =>
          const Center(child: CircularProgressIndicator(color: lyoAccent)),
        PlaylistStatus.error => Center(
            child: Text(notifier.error ?? 'Something went wrong',
                style: const TextStyle(color: lyoError))),
        PlaylistStatus.ready when notifier.playlists.isEmpty => Center(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(Icons.queue_music_outlined,
                    size: 48, color: lyoAccent.withValues(alpha: 0.4)),
                const SizedBox(height: lyoGapM),
                Text('No playlists yet',
                    style: TextStyle(color: textSub, fontSize: lyoBody1)),
              ],
            ),
          ),
        PlaylistStatus.ready => ListView.separated(
            padding: const EdgeInsets.all(lyoGapM),
            itemCount: notifier.playlists.length,
            separatorBuilder: (_, _) => const SizedBox(height: lyoGapS),
            itemBuilder: (context, i) {
              final Playlist p = notifier.playlists[i];
              return GestureDetector(
                onTap: () => context.push('/playlists/${p.id}'),
                child: Container(
                  padding: const EdgeInsets.all(lyoGapM),
                  decoration: BoxDecoration(
                    color: surface,
                    borderRadius: BorderRadius.circular(lyoRadiusCard),
                  ),
                  child: Row(
                    children: [
                      const Icon(Icons.queue_music, color: lyoAccent),
                      const SizedBox(width: lyoGapM),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(p.title,
                                style: TextStyle(
                                    color: textPrimary,
                                    fontSize: lyoBody1,
                                    fontWeight: FontWeight.w600)),
                            const SizedBox(height: 2),
                            Text('${p.trackIds.length} tracks',
                                style: TextStyle(color: textSub, fontSize: lyoCaption)),
                          ],
                        ),
                      ),
                      Icon(Icons.chevron_right, color: textSub),
                    ],
                  ),
                ),
              );
            },
          ),
      },
    );
  }
}
