import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../../core/theme/lyo_tokens.dart';
import '../../auth/providers/auth_notifier.dart';
import '../providers/playlist_notifier.dart';

Future<void> showAddToPlaylistSheet(BuildContext context, String trackId) {
  final dark = Theme.of(context).brightness == Brightness.dark;
  final token = context.read<AuthNotifier>().accessToken;
  if (token == null) {
    return Future.value();
  }
  final notifier = context.read<PlaylistNotifier>();
  if (notifier.status != PlaylistStatus.ready) {
    notifier.loadMine(token);
  }

  return showModalBottomSheet(
    context: context,
    backgroundColor: dark ? lyoSurfaceDark : lyoSurfaceLight,
    builder: (sheetContext) {
      return ChangeNotifierProvider.value(
        value: notifier,
        child: Consumer<PlaylistNotifier>(
          builder: (consumerContext, n, _) {
            final textPrimary = dark ? lyoTextDark : lyoTextLight;
            final textSub = dark ? lyoSubDark : lyoSubLight;
            return SafeArea(
              child: Padding(
                padding: const EdgeInsets.all(lyoGapL),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    Text('Add to playlist',
                        style: TextStyle(
                            fontSize: lyoH2,
                            fontWeight: FontWeight.w700,
                            color: textPrimary)),
                    const SizedBox(height: lyoGapM),
                    if (n.status == PlaylistStatus.loading)
                      const Padding(
                        padding: EdgeInsets.symmetric(vertical: lyoGapL),
                        child: Center(
                            child: CircularProgressIndicator(color: lyoAccent)),
                      )
                    else if (n.playlists.isEmpty)
                      Padding(
                        padding: const EdgeInsets.symmetric(vertical: lyoGapL),
                        child: Text('No playlists yet. Create one first.',
                            style: TextStyle(color: textSub)),
                      )
                    else
                      Flexible(
                        child: ListView.builder(
                          shrinkWrap: true,
                          itemCount: n.playlists.length,
                          itemBuilder: (itemContext, i) {
                            final p = n.playlists[i];
                            return ListTile(
                              leading: const Icon(Icons.queue_music, color: lyoAccent),
                              title: Text(p.title,
                                  style: TextStyle(color: textPrimary)),
                              onTap: () async {
                                await n.addTrack(
                                    playlistId: p.id,
                                    trackId: trackId,
                                    token: token);
                                if (sheetContext.mounted) {
                                  Navigator.of(sheetContext).pop();
                                }
                              },
                            );
                          },
                        ),
                      ),
                  ],
                ),
              ),
            );
          },
        ),
      );
    },
  );
}
