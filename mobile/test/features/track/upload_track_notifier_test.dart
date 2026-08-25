import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import 'package:mobile/core/api/api_client.dart';
import 'package:mobile/features/track/models/track_model.dart';
import 'package:mobile/features/track/providers/upload_track_notifier.dart';
import 'package:mobile/features/track/services/track_service.dart';

class MockTrackService extends Mock implements TrackService {}

class FakeFile extends Fake implements File {}

void main() {
  late MockTrackService mockSvc;
  late UploadTrackNotifier notifier;

  setUpAll(() {
    registerFallbackValue(FakeFile());
  });

  setUp(() {
    mockSvc = MockTrackService();
    notifier = UploadTrackNotifier(trackService: mockSvc);
  });

  final file = File('song.mp3');
  final createdTrack = Track(
    id: 't-1',
    broadcasterId: 'b-1',
    title: 'Song',
    audioUrl: 'https://bucket.s3.region.amazonaws.com/tracks/b-1/song.mp3',
    durationSeconds: 0,
    createdAt: DateTime(2026),
  );

  test('upload without a selected file sets an error and stays idle',
      () async {
    await notifier.upload(title: 'Song', token: 'tok');

    expect(notifier.status, UploadTrackStatus.idle);
    expect(notifier.error, isNotNull);
    verifyNever(() => mockSvc.uploadTrack(
          file: any(named: 'file'),
          title: any(named: 'title'),
          artist: any(named: 'artist'),
          token: any(named: 'token'),
        ));
  });

  test('upload success — sets success status and the uploaded track',
      () async {
    notifier.selectFile(file);
    when(() => mockSvc.uploadTrack(
          file: file,
          title: 'Song',
          artist: any(named: 'artist'),
          token: 'tok',
        )).thenAnswer((_) async => createdTrack);

    await notifier.upload(title: 'Song', token: 'tok');

    expect(notifier.status, UploadTrackStatus.success);
    expect(notifier.uploadedTrack, createdTrack);
    expect(notifier.error, isNull);
  });

  test('upload failure — sets error status and message from ApiException',
      () async {
    notifier.selectFile(file);
    when(() => mockSvc.uploadTrack(
          file: file,
          title: 'Song',
          artist: any(named: 'artist'),
          token: 'tok',
        )).thenThrow(const ApiException(500, 'Upload failed (500)'));

    await notifier.upload(title: 'Song', token: 'tok');

    expect(notifier.status, UploadTrackStatus.error);
    expect(notifier.error, 'Upload failed (500)');
  });

  test('reset clears file, track and error', () {
    notifier.selectFile(file);
    notifier.reset();

    expect(notifier.status, UploadTrackStatus.idle);
    expect(notifier.selectedFile, isNull);
    expect(notifier.uploadedTrack, isNull);
    expect(notifier.error, isNull);
  });
}
