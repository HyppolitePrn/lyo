import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import 'package:mobile/core/api/api_client.dart';
import 'package:mobile/features/user/services/user_service.dart';

class MockApiClient extends Mock implements ApiClient {}

void main() {
  late MockApiClient mockApi;
  late UserService service;

  setUp(() {
    mockApi = MockApiClient();
    service = UserService(mockApi);
  });

  group('deleteAccount', () {
    test('sends an authenticated DELETE to /users/me', () async {
      when(() => mockApi.delete(any(), token: any(named: 'token')))
          .thenAnswer((_) async {});

      await service.deleteAccount('jwt-abc');

      // The account is identified by the token, never by a path segment: a
      // user-supplied id here would be a way to delete somebody else.
      verify(() => mockApi.delete('/users/me', token: 'jwt-abc')).called(1);
    });

    test('surfaces the server error instead of reporting success', () async {
      when(() => mockApi.delete(any(), token: any(named: 'token')))
          .thenThrow(const ApiException(503, 'account deletion is disabled'));

      expect(
        () => service.deleteAccount('jwt-abc'),
        throwsA(isA<ApiException>()),
      );
    });
  });
}
