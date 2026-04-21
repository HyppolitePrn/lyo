import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import 'package:mobile/core/api/api_client.dart';
import 'package:mobile/features/auth/providers/auth_notifier.dart';

class MockApiClient extends Mock implements ApiClient {}

void main() {
  late MockApiClient mockApi;

  setUp(() {
    mockApi = MockApiClient();
  });

  ProviderContainer makeContainer() => ProviderContainer(
        overrides: [apiClientProvider.overrideWithValue(mockApi)],
      );

  const validTokens = {
    'access_token': 'access-abc',
    'refresh_token': 'refresh-xyz',
  };

  group('signIn', () {
    test('success — sets isAuthenticated true', () async {
      when(
        () => mockApi.post('/auth/login', any()),
      ).thenAnswer((_) async => validTokens);

      final container = makeContainer();
      addTearDown(container.dispose);

      final result = await container
          .read(authNotifierProvider.notifier)
          .signIn('user@example.com', 'password123');

      expect(result, isTrue);
      expect(container.read(authNotifierProvider).isAuthenticated, isTrue);
      expect(container.read(authNotifierProvider).error, isNull);
    });

    test('invalid credentials — sets error, stays unauthenticated', () async {
      when(
        () => mockApi.post('/auth/login', any()),
      ).thenThrow(const ApiException(401, 'Invalid credentials'));

      final container = makeContainer();
      addTearDown(container.dispose);

      final result = await container
          .read(authNotifierProvider.notifier)
          .signIn('user@example.com', 'wrongpassword');

      expect(result, isFalse);
      expect(container.read(authNotifierProvider).isAuthenticated, isFalse);
      expect(
        container.read(authNotifierProvider).error,
        'Invalid credentials',
      );
    });

    test('network failure — sets generic error message', () async {
      when(
        () => mockApi.post('/auth/login', any()),
      ).thenThrow(Exception('connection refused'));

      final container = makeContainer();
      addTearDown(container.dispose);

      final result = await container
          .read(authNotifierProvider.notifier)
          .signIn('user@example.com', 'password123');

      expect(result, isFalse);
      expect(container.read(authNotifierProvider).isAuthenticated, isFalse);
      expect(
        container.read(authNotifierProvider).error,
        contains('network'),
      );
    });

    test('clears previous error on new attempt', () async {
      when(() => mockApi.post('/auth/login', any()))
          .thenThrow(const ApiException(401, 'Invalid credentials'));

      final container = makeContainer();
      addTearDown(container.dispose);

      await container
          .read(authNotifierProvider.notifier)
          .signIn('user@example.com', 'bad');

      expect(container.read(authNotifierProvider).error, isNotNull);

      when(() => mockApi.post('/auth/login', any()))
          .thenAnswer((_) async => validTokens);

      await container
          .read(authNotifierProvider.notifier)
          .signIn('user@example.com', 'password123');

      expect(container.read(authNotifierProvider).error, isNull);
    });
  });

  group('register', () {
    test('success — sets isAuthenticated true', () async {
      when(
        () => mockApi.post('/auth/register', any()),
      ).thenAnswer((_) async => validTokens);

      final container = makeContainer();
      addTearDown(container.dispose);

      final result = await container
          .read(authNotifierProvider.notifier)
          .register('johndoe', 'john@example.com', 'password123');

      expect(result, isTrue);
      expect(container.read(authNotifierProvider).isAuthenticated, isTrue);
      expect(container.read(authNotifierProvider).error, isNull);
    });

    test('duplicate username — sets error from backend', () async {
      when(
        () => mockApi.post('/auth/register', any()),
      ).thenThrow(const ApiException(409, 'Username already taken'));

      final container = makeContainer();
      addTearDown(container.dispose);

      final result = await container
          .read(authNotifierProvider.notifier)
          .register('johndoe', 'john@example.com', 'password123');

      expect(result, isFalse);
      expect(container.read(authNotifierProvider).isAuthenticated, isFalse);
      expect(
        container.read(authNotifierProvider).error,
        'Username already taken',
      );
    });

    test('duplicate email — sets error from backend', () async {
      when(
        () => mockApi.post('/auth/register', any()),
      ).thenThrow(const ApiException(409, 'Email already registered'));

      final container = makeContainer();
      addTearDown(container.dispose);

      final result = await container
          .read(authNotifierProvider.notifier)
          .register('newuser', 'existing@example.com', 'password123');

      expect(result, isFalse);
      expect(
        container.read(authNotifierProvider).error,
        'Email already registered',
      );
    });

    test('network failure — sets generic error message', () async {
      when(
        () => mockApi.post('/auth/register', any()),
      ).thenThrow(Exception('connection refused'));

      final container = makeContainer();
      addTearDown(container.dispose);

      final result = await container
          .read(authNotifierProvider.notifier)
          .register('johndoe', 'john@example.com', 'password123');

      expect(result, isFalse);
      expect(
        container.read(authNotifierProvider).error,
        contains('network'),
      );
    });
  });

  group('continueAnonymously', () {
    test('sets isAnonymous true without touching isAuthenticated', () {
      final container = makeContainer();
      addTearDown(container.dispose);

      container.read(authNotifierProvider.notifier).continueAnonymously();

      final state = container.read(authNotifierProvider);
      expect(state.isAnonymous, isTrue);
      expect(state.isAuthenticated, isFalse);
      expect(state.hasAccess, isTrue);
    });
  });
}
