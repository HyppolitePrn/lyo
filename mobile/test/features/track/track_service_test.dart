import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import 'package:mobile/core/api/api_client.dart';
import 'package:mobile/features/track/services/track_service.dart';

class MockApiClient extends Mock implements ApiClient {}

void main() {
  late MockApiClient mockApi;
  late TrackService service;

  setUp(() {
    mockApi = MockApiClient();
    service = TrackService(mockApi);
  });

  const trackJson = {
    'id': 't-1',
    'broadcaster_id': 'b-1',
    'title': 'Track One',
    'artist': 'Artist One',
    'audio_url': 'https://bucket.s3.region.amazonaws.com/tracks/b-1/x.mp3',
    'duration_seconds': 180,
    'created_at': '2026-01-01T00:00:00.000Z',
  };

  group('listTracks', () {
    test('requests page/limit and maps items', () async {
      when(() => mockApi.get(any(), token: any(named: 'token')))
          .thenAnswer((_) async => {'items': [trackJson]});

      final tracks = await service.listTracks(page: 2, limit: 10);

      final captured = verify(
        () => mockApi.get(captureAny(), token: any(named: 'token')),
      ).captured.single as String;
      expect(captured, contains('page=2'));
      expect(captured, contains('limit=10'));
      expect(tracks, hasLength(1));
      expect(tracks.single.id, 't-1');
      expect(tracks.single.artist, 'Artist One');
    });

    test('omits broadcaster_id when not provided', () async {
      when(() => mockApi.get(any(), token: any(named: 'token')))
          .thenAnswer((_) async => {'items': <dynamic>[]});

      await service.listTracks();

      final captured = verify(
        () => mockApi.get(captureAny(), token: any(named: 'token')),
      ).captured.single as String;
      expect(captured, isNot(contains('broadcaster_id')));
    });
  });

  test('getTrack fetches by id', () async {
    when(() => mockApi.get('/tracks/t-1', token: any(named: 'token')))
        .thenAnswer((_) async => trackJson);

    final track = await service.getTrack('t-1');

    expect(track.id, 't-1');
    expect(track.title, 'Track One');
  });

  test('createTrack posts metadata and returns the created track', () async {
    when(() => mockApi.post('/tracks', any(), token: any(named: 'token')))
        .thenAnswer((_) async => trackJson);

    final track = await service.createTrack(
      title: 'Track One',
      audioUrl: trackJson['audio_url'] as String,
      token: 'tok',
    );

    expect(track.id, 't-1');
    final body = verify(
      () => mockApi.post('/tracks', captureAny(), token: any(named: 'token')),
    ).captured.single as Map<String, dynamic>;
    expect(body['title'], 'Track One');
    expect(body.containsKey('artist'), isFalse);
  });

  test('deleteTrack calls DELETE on the track path', () async {
    when(() => mockApi.delete(any(), token: any(named: 'token')))
        .thenAnswer((_) async {});

    await service.deleteTrack('t-1', 'tok');

    verify(() => mockApi.delete('/tracks/t-1', token: 'tok')).called(1);
  });
}
