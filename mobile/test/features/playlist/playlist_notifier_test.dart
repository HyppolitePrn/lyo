import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import 'package:mobile/core/api/api_client.dart';
import 'package:mobile/features/playlist/models/playlist_model.dart';
import 'package:mobile/features/playlist/providers/playlist_notifier.dart';
import 'package:mobile/features/playlist/services/playlist_service.dart';

class MockPlaylistService extends Mock implements PlaylistService {}

void main() {
  late MockPlaylistService mockSvc;
  late PlaylistNotifier notifier;

  setUp(() {
    mockSvc = MockPlaylistService();
    notifier = PlaylistNotifier(playlistService: mockSvc);
  });

  final playlist = Playlist(
    id: 'p-1',
    ownerId: 'u-1',
    title: 'My Mix',
    trackIds: const [],
    isPublic: false,
    createdAt: DateTime(2026),
    updatedAt: DateTime(2026),
  );

  test('loadMine success sets ready status and playlists', () async {
    when(() => mockSvc.listPlaylists(token: any(named: 'token')))
        .thenAnswer((_) async => [playlist]);

    await notifier.loadMine('tok');

    expect(notifier.status, PlaylistStatus.ready);
    expect(notifier.playlists, [playlist]);
  });

  test('loadMine failure sets error status and message', () async {
    when(() => mockSvc.listPlaylists(token: any(named: 'token')))
        .thenThrow(const ApiException(500, 'server error'));

    await notifier.loadMine('tok');

    expect(notifier.status, PlaylistStatus.error);
    expect(notifier.error, 'server error');
  });

  test('create prepends the new playlist on success', () async {
    when(() => mockSvc.createPlaylist(
          title: any(named: 'title'),
          token: any(named: 'token'),
          description: any(named: 'description'),
          isPublic: any(named: 'isPublic'),
        )).thenAnswer((_) async => playlist);

    final ok = await notifier.create(title: 'My Mix', token: 'tok');

    expect(ok, isTrue);
    expect(notifier.playlists, [playlist]);
  });

  test('delete removes the playlist from the list', () async {
    when(() => mockSvc.listPlaylists(token: any(named: 'token')))
        .thenAnswer((_) async => [playlist]);
    await notifier.loadMine('tok');

    when(() => mockSvc.deletePlaylist('p-1', 'tok'))
        .thenAnswer((_) async {});

    final ok = await notifier.delete('p-1', 'tok');

    expect(ok, isTrue);
    expect(notifier.playlists, isEmpty);
  });

  test('addTrack replaces the playlist with the updated one', () async {
    final updated = Playlist(
      id: 'p-1',
      ownerId: 'u-1',
      title: 'My Mix',
      trackIds: const ['t-1'],
      isPublic: false,
      createdAt: DateTime(2026),
      updatedAt: DateTime(2026),
    );
    when(() => mockSvc.listPlaylists(token: any(named: 'token')))
        .thenAnswer((_) async => [playlist]);
    await notifier.loadMine('tok');

    when(() => mockSvc.addTrack(
          playlistId: 'p-1',
          trackId: 't-1',
          token: 'tok',
        )).thenAnswer((_) async => updated);

    final ok = await notifier.addTrack(
        playlistId: 'p-1', trackId: 't-1', token: 'tok');

    expect(ok, isTrue);
    expect(notifier.playlists.single.trackIds, ['t-1']);
  });
}
