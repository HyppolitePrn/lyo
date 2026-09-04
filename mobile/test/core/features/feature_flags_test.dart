import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import 'package:mobile/core/api/api_client.dart';
import 'package:mobile/core/features/feature_flags_provider.dart';
import 'package:mobile/core/features/feature_flags_service.dart';

class MockFeatureFlagsService extends Mock implements FeatureFlagsService {}

void main() {
  late MockFeatureFlagsService mockSvc;
  late FeatureFlags flags;

  setUp(() {
    mockSvc = MockFeatureFlagsService();
    flags = FeatureFlags(service: mockSvc);
  });

  test('serves defaults before the first load', () {
    expect(flags.isEnabled('live_streaming'), isTrue);
    expect(flags.isEnabled('chat_websocket'), isFalse);
    expect(flags.loaded, isFalse);
  });

  test('the server answer overrides the defaults', () async {
    when(mockSvc.fetch).thenAnswer((_) async => {
          'live_streaming': false,
          'chat_websocket': true,
        });

    var notified = 0;
    flags.addListener(() => notified++);
    await flags.load();

    expect(flags.isEnabled('live_streaming'), isFalse);
    expect(flags.isEnabled('chat_websocket'), isTrue);
    expect(flags.loaded, isTrue);
    expect(notified, 1);
  });

  test('flags the server omits keep their default', () async {
    when(mockSvc.fetch).thenAnswer((_) async => {'chat_websocket': true});

    await flags.load();

    expect(flags.isEnabled('playlists'), isTrue);
  });

  // An unreachable API must not hide working features.
  test('a failed load keeps the defaults', () async {
    when(mockSvc.fetch).thenThrow(const ApiException(0, 'offline'));

    await flags.load();

    expect(flags.isEnabled('playlists'), isTrue);
    expect(flags.loaded, isFalse);
  });

  test('applyToggle updates the flag and notifies', () async {
    var notified = 0;
    flags.addListener(() => notified++);

    flags.applyToggle('chat_websocket', true);

    expect(flags.isEnabled('chat_websocket'), isTrue);
    expect(notified, 1);
  });

  test('applyToggle to the current value notifies nobody', () {
    var notified = 0;
    flags.addListener(() => notified++);

    flags.applyToggle('playlists', true);

    expect(notified, 0);
  });

  test('an unknown flag is off', () {
    expect(flags.isEnabled('nope'), isFalse);
  });
}
