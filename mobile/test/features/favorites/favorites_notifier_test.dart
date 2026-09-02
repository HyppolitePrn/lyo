import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import 'package:mobile/core/api/api_client.dart';
import 'package:mobile/features/favorites/providers/favorites_notifier.dart';
import 'package:mobile/features/favorites/services/favorites_service.dart';
import 'package:mobile/features/track/models/track_model.dart';

class MockFavoritesService extends Mock implements FavoritesService {}

void main() {
  late MockFavoritesService mockSvc;
  late FavoritesNotifier notifier;

  setUp(() {
    mockSvc = MockFavoritesService();
    notifier = FavoritesNotifier(favoritesService: mockSvc);
  });

  final track = Track(
    id: 't-1',
    broadcasterId: 'b-1',
    title: 'Song',
    audioUrl: 'https://x/y.mp3',
    durationSeconds: 120,
    createdAt: DateTime(2026),
  );

  test('load populates lists and favorite id sets', () async {
    when(() => mockSvc.getFavorites('tok')).thenAnswer(
      (_) async => Favorites(tracks: [track], streams: const [], playlists: const []),
    );

    await notifier.load('tok');

    expect(notifier.status, FavoritesStatus.ready);
    expect(notifier.tracks, [track]);
    expect(notifier.isTrackFavorited('t-1'), isTrue);
    expect(notifier.isTrackFavorited('other'), isFalse);
  });

  test('load failure sets error status', () async {
    when(() => mockSvc.getFavorites('tok'))
        .thenThrow(const ApiException(500, 'server error'));

    await notifier.load('tok');

    expect(notifier.status, FavoritesStatus.error);
    expect(notifier.error, 'server error');
  });

  test('toggleTrack optimistically favorites then confirms', () async {
    when(() => mockSvc.favoriteTrack('t-1', 'tok')).thenAnswer((_) async {});

    final future = notifier.toggleTrack('t-1', 'tok');
    expect(notifier.isTrackFavorited('t-1'), isTrue);
    await future;

    verify(() => mockSvc.favoriteTrack('t-1', 'tok')).called(1);
    expect(notifier.isTrackFavorited('t-1'), isTrue);
  });

  test('toggleTrack reverts optimistic update on failure', () async {
    when(() => mockSvc.favoriteTrack('t-1', 'tok'))
        .thenThrow(const ApiException(404, 'not found'));

    await notifier.toggleTrack('t-1', 'tok');

    expect(notifier.isTrackFavorited('t-1'), isFalse);
  });

  test('toggleTrack unfavorites an already-favorited track', () async {
    when(() => mockSvc.getFavorites('tok')).thenAnswer(
      (_) async => Favorites(tracks: [track], streams: const [], playlists: const []),
    );
    await notifier.load('tok');

    when(() => mockSvc.unfavoriteTrack('t-1', 'tok')).thenAnswer((_) async {});

    await notifier.toggleTrack('t-1', 'tok');

    expect(notifier.isTrackFavorited('t-1'), isFalse);
    expect(notifier.tracks, isEmpty);
  });
}
