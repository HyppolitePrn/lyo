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

  AuthNotifier makeNotifier() => AuthNotifier(apiClient: mockApi);

  const validTokens = {
    'access_token': 'access-abc',
    'refresh_token': 'refresh-xyz',
  };

  group('signIn', () {
    test('success — sets isAuthenticated true', () async {
      when(
        () => mockApi.post('/auth/login', any()),
      ).thenAnswer((_) async => validTokens);

      final notifier = makeNotifier();

      final result =
          await notifier.signIn('user@example.com', 'password123');

      expect(result, isTrue);
      expect(notifier.isAuthenticated, isTrue);
      expect(notifier.error, isNull);
    });

    test('invalid credentials — sets error, stays unauthenticated', () async {
      when(
        () => mockApi.post('/auth/login', any()),
      ).thenThrow(const ApiException(401, 'Invalid credentials'));

      final notifier = makeNotifier();

      final result =
          await notifier.signIn('user@example.com', 'wrongpassword');

      expect(result, isFalse);
      expect(notifier.isAuthenticated, isFalse);
      expect(notifier.error, 'Invalid credentials');
    });

    test('network failure — sets generic error message', () async {
      when(
        () => mockApi.post('/auth/login', any()),
      ).thenThrow(Exception('connection refused'));

      final notifier = makeNotifier();

      final result =
          await notifier.signIn('user@example.com', 'password123');

      expect(result, isFalse);
      expect(notifier.isAuthenticated, isFalse);
      expect(notifier.error, contains('network'));
    });

    test('clears previous error on new attempt', () async {
      when(() => mockApi.post('/auth/login', any()))
          .thenThrow(const ApiException(401, 'Invalid credentials'));

      final notifier = makeNotifier();

      await notifier.signIn('user@example.com', 'bad');

      expect(notifier.error, isNotNull);

      when(() => mockApi.post('/auth/login', any()))
          .thenAnswer((_) async => validTokens);

      await notifier.signIn('user@example.com', 'password123');

      expect(notifier.error, isNull);
    });
  });

  group('register', () {
    test('success — sets isAuthenticated true', () async {
      when(
        () => mockApi.post('/auth/register', any()),
      ).thenAnswer((_) async => validTokens);

      final notifier = makeNotifier();

      final result = await notifier.register(
          'johndoe', 'john@example.com', 'password123');

      expect(result, isTrue);
      expect(notifier.isAuthenticated, isTrue);
      expect(notifier.error, isNull);
    });

    test('duplicate username — sets error from backend', () async {
      when(
        () => mockApi.post('/auth/register', any()),
      ).thenThrow(const ApiException(409, 'Username already taken'));

      final notifier = makeNotifier();

      final result = await notifier.register(
          'johndoe', 'john@example.com', 'password123');

      expect(result, isFalse);
      expect(notifier.isAuthenticated, isFalse);
      expect(notifier.error, 'Username already taken');
    });

    test('duplicate email — sets error from backend', () async {
      when(
        () => mockApi.post('/auth/register', any()),
      ).thenThrow(const ApiException(409, 'Email already registered'));

      final notifier = makeNotifier();

      final result = await notifier.register(
          'newuser', 'existing@example.com', 'password123');

      expect(result, isFalse);
      expect(notifier.error, 'Email already registered');
    });

    test('network failure — sets generic error message', () async {
      when(
        () => mockApi.post('/auth/register', any()),
      ).thenThrow(Exception('connection refused'));

      final notifier = makeNotifier();

      final result = await notifier.register(
          'johndoe', 'john@example.com', 'password123');

      expect(result, isFalse);
      expect(notifier.error, contains('network'));
    });
  });

  group('continueAnonymously', () {
    test('sets isAnonymous true without touching isAuthenticated', () {
      final notifier = makeNotifier();

      notifier.continueAnonymously();

      expect(notifier.isAnonymous, isTrue);
      expect(notifier.isAuthenticated, isFalse);
      expect(notifier.hasAccess, isTrue);
    });
  });
}
