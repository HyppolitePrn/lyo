import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import 'package:mobile/core/api/api_client.dart';
import 'package:mobile/features/profile/providers/account_notifier.dart';
import 'package:mobile/features/user/services/user_service.dart';

class MockApiClient extends Mock implements ApiClient {}

void main() {
  late MockApiClient mockApi;
  late AccountNotifier notifier;

  setUp(() {
    mockApi = MockApiClient();
    notifier = AccountNotifier(userService: UserService(mockApi));
  });

  test('a successful delete reports true and clears the busy flag', () async {
    when(() => mockApi.delete(any(), token: any(named: 'token')))
        .thenAnswer((_) async {});

    expect(await notifier.deleteAccount('jwt'), isTrue);
    expect(notifier.isDeleting, isFalse);
    expect(notifier.error, isNull);
  });

  test('a rejected delete reports false and keeps the reason', () async {
    when(() => mockApi.delete(any(), token: any(named: 'token')))
        .thenThrow(const ApiException(503, 'account deletion is disabled'));

    expect(await notifier.deleteAccount('jwt'), isFalse);
    expect(notifier.isDeleting, isFalse);
    expect(notifier.error, 'account deletion is disabled');
  });

  // Deleting an account is not idempotent from the user's side: a second call
  // would run against a token whose account is already gone.
  test('a second call while one is in flight is ignored', () async {
    final gate = Completer<void>();
    when(() => mockApi.delete(any(), token: any(named: 'token')))
        .thenAnswer((_) => gate.future);

    final first = notifier.deleteAccount('jwt');
    expect(notifier.isDeleting, isTrue);

    expect(await notifier.deleteAccount('jwt'), isFalse);

    gate.complete();
    expect(await first, isTrue);
    verify(() => mockApi.delete('/users/me', token: 'jwt')).called(1);
  });

  test('a retry after a failure clears the previous error', () async {
    when(() => mockApi.delete(any(), token: any(named: 'token')))
        .thenThrow(const ApiException(0, 'Network error. Try again.'));
    await notifier.deleteAccount('jwt');
    expect(notifier.error, isNotNull);

    when(() => mockApi.delete(any(), token: any(named: 'token')))
        .thenAnswer((_) async {});
    expect(await notifier.deleteAccount('jwt'), isTrue);
    expect(notifier.error, isNull);
  });
}
