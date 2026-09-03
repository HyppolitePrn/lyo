import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import 'package:mobile/core/api/api_client.dart';
import 'package:mobile/features/auth/providers/auth_notifier.dart';
import 'package:mobile/features/auth/services/token_store.dart';

class MockApiClient extends Mock implements ApiClient {}

// Builds a signature-less JWT whose payload carries a real `exp`/`role`. The
// app only ever decodes the payload — the backend verifies the signature.
String fakeJwt({required Duration expiresIn, String role = 'user'}) {
  String seg(Map<String, dynamic> m) =>
      base64Url.encode(utf8.encode(jsonEncode(m))).replaceAll('=', '');
  final exp = DateTime.now().toUtc().add(expiresIn).millisecondsSinceEpoch;
  return '${seg({'alg': 'HS256'})}.'
      '${seg({'role': role, 'exp': exp ~/ 1000})}.sig';
}

void main() {
  late MockApiClient mockApi;
  late InMemoryTokenStore store;

  setUp(() {
    mockApi = MockApiClient();
    store = InMemoryTokenStore();
  });

  AuthNotifier makeNotifier() =>
      AuthNotifier(apiClient: mockApi, tokenStore: store);

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

  group('forgotPassword', () {
    test('success — sets resetEmailSent true', () async {
      when(
        () => mockApi.post('/auth/forgot-password', any()),
      ).thenAnswer((_) async => <String, dynamic>{});

      final notifier = makeNotifier();

      final result = await notifier.forgotPassword('user@example.com');

      expect(result, isTrue);
      expect(notifier.resetEmailSent, isTrue);
      expect(notifier.error, isNull);
    });

    test('backend error — sets error from backend', () async {
      when(
        () => mockApi.post('/auth/forgot-password', any()),
      ).thenThrow(const ApiException(400, 'Invalid email'));

      final notifier = makeNotifier();

      final result = await notifier.forgotPassword('bad');

      expect(result, isFalse);
      expect(notifier.resetEmailSent, isFalse);
      expect(notifier.error, 'Invalid email');
    });

    test('network failure — sets generic error message', () async {
      when(
        () => mockApi.post('/auth/forgot-password', any()),
      ).thenThrow(Exception('connection refused'));

      final notifier = makeNotifier();

      final result = await notifier.forgotPassword('user@example.com');

      expect(result, isFalse);
      expect(notifier.error, contains('network'));
    });
  });

  group('resetPassword', () {
    test('success — sets resetPasswordSuccess true', () async {
      when(
        () => mockApi.post('/auth/reset-password', any()),
      ).thenAnswer((_) async => <String, dynamic>{});

      final notifier = makeNotifier();

      final result = await notifier.resetPassword('tok', 'newpassword123');

      expect(result, isTrue);
      expect(notifier.resetPasswordSuccess, isTrue);
      expect(notifier.error, isNull);
    });

    test('invalid or expired token — sets error from backend', () async {
      when(
        () => mockApi.post('/auth/reset-password', any()),
      ).thenThrow(const ApiException(400, 'invalid or expired token'));

      final notifier = makeNotifier();

      final result = await notifier.resetPassword('bad-tok', 'newpassword123');

      expect(result, isFalse);
      expect(notifier.resetPasswordSuccess, isFalse);
      expect(notifier.error, 'invalid or expired token');
    });

    test('network failure — sets generic error message', () async {
      when(
        () => mockApi.post('/auth/reset-password', any()),
      ).thenThrow(Exception('connection refused'));

      final notifier = makeNotifier();

      final result = await notifier.resetPassword('tok', 'newpassword123');

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

  group('session persistence', () {
    test('signIn stores the token pair for the next launch', () async {
      when(
        () => mockApi.post('/auth/login', any()),
      ).thenAnswer((_) async => validTokens);

      await makeNotifier().signIn('user@example.com', 'password123');

      expect(await store.read(), (access: 'access-abc', refresh: 'refresh-xyz'));
    });

    test('signOut clears the stored pair', () async {
      await store.write(access: 'access-abc', refresh: 'refresh-xyz');
      final notifier = makeNotifier();

      notifier.signOut();
      await pumpEventQueue();

      expect(await store.read(), isNull);
      expect(notifier.accessToken, isNull);
    });
  });

  group('restoreSession', () {
    test('no stored pair — reports no session', () async {
      final notifier = makeNotifier();

      expect(await notifier.restoreSession(), isFalse);
      expect(notifier.isAuthenticated, isFalse);
    });

    test('still-valid access token — restores without a refresh call',
        () async {
      final access = fakeJwt(expiresIn: const Duration(minutes: 10));
      await store.write(access: access, refresh: 'refresh-xyz');
      when(() => mockApi.get(any(), token: any(named: 'token')))
          .thenThrow(const ApiException(500, 'profile unavailable'));

      final notifier = makeNotifier();
      addTearDown(notifier.dispose);

      expect(await notifier.restoreSession(), isTrue);
      expect(notifier.isAuthenticated, isTrue);
      expect(notifier.accessToken, access);
      verifyNever(() => mockApi.post('/auth/refresh', any()));
    });

    test('expired access token — refreshes and adopts the new pair', () async {
      await store.write(
        access: fakeJwt(expiresIn: const Duration(minutes: -5)),
        refresh: 'refresh-xyz',
      );
      final fresh = fakeJwt(expiresIn: const Duration(minutes: 15));
      when(() => mockApi.post('/auth/refresh', any())).thenAnswer(
        (_) async => {'access_token': fresh, 'refresh_token': 'refresh-2'},
      );

      final notifier = makeNotifier();
      addTearDown(notifier.dispose);

      expect(await notifier.restoreSession(), isTrue);
      expect(notifier.accessToken, fresh);
      expect(await store.read(), (access: fresh, refresh: 'refresh-2'));
    });

    test('expired refresh token — clears the stored pair', () async {
      await store.write(
        access: fakeJwt(expiresIn: const Duration(minutes: -5)),
        refresh: 'refresh-expired',
      );
      when(() => mockApi.post('/auth/refresh', any()))
          .thenThrow(const ApiException(401, 'invalid or expired refresh token'));

      final notifier = makeNotifier();

      expect(await notifier.restoreSession(), isFalse);
      expect(notifier.isAuthenticated, isFalse);
      expect(await store.read(), isNull);
    });

    test('server unreachable — keeps the stored pair for a later retry',
        () async {
      await store.write(
        access: fakeJwt(expiresIn: const Duration(minutes: -5)),
        refresh: 'refresh-xyz',
      );
      when(() => mockApi.post('/auth/refresh', any()))
          .thenThrow(const ApiException(0, 'Cannot reach the server.'));

      final notifier = makeNotifier();

      expect(await notifier.restoreSession(), isFalse);
      expect(await store.read(), isNotNull);
    });
  });

  group('ensureFreshSession', () {
    Future<AuthNotifier> signedIn(String access) async {
      when(() => mockApi.post('/auth/login', any())).thenAnswer(
        (_) async => {'access_token': access, 'refresh_token': 'refresh-xyz'},
      );
      when(() => mockApi.get(any(), token: any(named: 'token')))
          .thenThrow(const ApiException(500, 'profile unavailable'));
      final notifier = makeNotifier();
      await notifier.signIn('user@example.com', 'password123');
      return notifier;
    }

    test('token still fresh — no refresh call', () async {
      final notifier =
          await signedIn(fakeJwt(expiresIn: const Duration(minutes: 10)));
      addTearDown(notifier.dispose);

      await notifier.ensureFreshSession();

      verifyNever(() => mockApi.post('/auth/refresh', any()));
    });

    test('token expired while suspended — refreshes on resume', () async {
      final notifier =
          await signedIn(fakeJwt(expiresIn: const Duration(seconds: -1)));
      addTearDown(notifier.dispose);
      final fresh = fakeJwt(expiresIn: const Duration(minutes: 15));
      when(() => mockApi.post('/auth/refresh', any())).thenAnswer(
        (_) async => {'access_token': fresh, 'refresh_token': 'refresh-2'},
      );

      await notifier.ensureFreshSession();

      expect(notifier.accessToken, fresh);
    });

    test('concurrent callers share a single refresh request', () async {
      final notifier =
          await signedIn(fakeJwt(expiresIn: const Duration(seconds: -1)));
      addTearDown(notifier.dispose);
      when(() => mockApi.post('/auth/refresh', any())).thenAnswer((_) async {
        await Future<void>.delayed(const Duration(milliseconds: 10));
        return {
          'access_token': fakeJwt(expiresIn: const Duration(minutes: 15)),
          'refresh_token': 'refresh-2',
        };
      });

      await Future.wait([
        notifier.ensureFreshSession(),
        notifier.ensureFreshSession(),
      ]);

      verify(() => mockApi.post('/auth/refresh', any())).called(1);
    });

    test('anonymous session — nothing to refresh', () async {
      final notifier = makeNotifier();
      notifier.continueAnonymously();

      await notifier.ensureFreshSession();

      verifyNever(() => mockApi.post('/auth/refresh', any()));
    });
  });
}
