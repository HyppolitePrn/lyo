import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';
import 'package:provider/provider.dart';

import 'package:mobile/core/api/api_client.dart';
import 'package:mobile/core/features/feature_flags_provider.dart';
import 'package:mobile/core/router/app_router.dart';
import 'package:mobile/core/features/feature_flags_service.dart';
import 'package:mobile/core/theme/lyo_theme.dart';
import 'package:mobile/features/auth/providers/auth_notifier.dart';
import 'package:mobile/features/auth/services/token_store.dart';
import 'package:mobile/features/user/services/user_service.dart';
import 'package:mobile/features/profile/providers/account_notifier.dart';

class MockApiClient extends Mock implements ApiClient {}

/// Serves a fixed flag map without touching the network.
class _StubFlagsService implements FeatureFlagsService {
  const _StubFlagsService(this.flags);
  final Map<String, bool> flags;

  @override
  Future<Map<String, bool>> fetch() async => flags;
}

String fakeJwt({String role = 'user'}) {
  String seg(Map<String, dynamic> m) =>
      base64Url.encode(utf8.encode(jsonEncode(m))).replaceAll('=', '');
  final exp = DateTime.now().toUtc().add(const Duration(minutes: 10));
  return '${seg({'alg': 'HS256'})}.'
      '${seg({'role': role, 'exp': exp.millisecondsSinceEpoch ~/ 1000})}.sig';
}

void main() {
  late MockApiClient mockApi;

  setUp(() => mockApi = MockApiClient());

  Future<AuthNotifier> pumpProfile(
    WidgetTester tester, {
    required bool accountDeletion,
  }) async {
    tester.view.physicalSize = const Size(1000, 2000);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);

    final flags = FeatureFlags(
      service: _StubFlagsService({'account_deletion': accountDeletion}),
    );
    await flags.load();

    // A restored, still-valid session — the signed-out body has no profile
    // actions at all, so the screen has to be genuinely authenticated for any
    // of this to mean anything.
    final store = InMemoryTokenStore();
    await store.write(access: fakeJwt(), refresh: 'refresh-xyz');
    when(() => mockApi.get(any(), token: any(named: 'token')))
        .thenThrow(const ApiException(500, 'profile unavailable'));

    final auth = AuthNotifier(apiClient: mockApi, tokenStore: store);
    expect(await auth.restoreSession(), isTrue);

    // The real router, not a bare `home:` — deleting an account ends by
    // navigating to /splash, and that call needs a GoRouter above it.
    final router = createAppRouter(initialLocation: '/profile');
    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<FeatureFlags>.value(value: flags),
          ChangeNotifierProvider<AuthNotifier>.value(value: auth),
          ChangeNotifierProvider<AccountNotifier>(
            create: (_) => AccountNotifier(userService: UserService(mockApi)),
          ),
        ],
        child: MaterialApp.router(
          theme: lyoTheme(Brightness.dark),
          routerConfig: router,
        ),
      ),
    );
    await tester.pumpAndSettle();
    return auth;
  }

  testWidgets('the delete button is offered to a signed-in user',
      (tester) async {
    final auth = await pumpProfile(tester, accountDeletion: true);
    expect(find.text('Supprimer mon compte'), findsOneWidget);
    auth.dispose();
  });

  // An admin who turns the flag off must actually take the action away, not
  // leave a button that only fails once tapped.
  testWidgets('the delete button is hidden while the flag is off',
      (tester) async {
    final auth = await pumpProfile(tester, accountDeletion: false);
    expect(find.text('Se déconnecter'), findsOneWidget);
    expect(find.text('Supprimer mon compte'), findsNothing);
    auth.dispose();
  });

  // Deleting an account is irreversible, so a single stray tap must never be
  // enough to do it.
  testWidgets('tapping delete asks for confirmation before calling the API',
      (tester) async {
    final auth = await pumpProfile(tester, accountDeletion: true);

    await tester.tap(find.text('Supprimer mon compte'));
    await tester.pumpAndSettle();

    expect(find.text('Supprimer le compte ?'), findsOneWidget);
    verifyNever(() => mockApi.delete(any(), token: any(named: 'token')));

    await tester.tap(find.text('Annuler'));
    await tester.pumpAndSettle();

    verifyNever(() => mockApi.delete(any(), token: any(named: 'token')));
    expect(auth.isAuthenticated, isTrue);
    auth.dispose();
  });

  testWidgets('confirming deletes the account and ends the session',
      (tester) async {
    when(() => mockApi.delete(any(), token: any(named: 'token')))
        .thenAnswer((_) async {});

    final auth = await pumpProfile(tester, accountDeletion: true);

    await tester.tap(find.text('Supprimer mon compte'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Supprimer'));
    await tester.pumpAndSettle();

    verify(() => mockApi.delete('/users/me', token: any(named: 'token')))
        .called(1);
    // The token now names an account that no longer exists; keeping it would
    // leave the app in a session it cannot use.
    expect(auth.isAuthenticated, isFalse);
    auth.dispose();
  });

  testWidgets('a failed delete keeps the session and shows the reason',
      (tester) async {
    when(() => mockApi.delete(any(), token: any(named: 'token')))
        .thenThrow(const ApiException(503, 'account deletion is disabled'));

    final auth = await pumpProfile(tester, accountDeletion: true);

    await tester.tap(find.text('Supprimer mon compte'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Supprimer'));
    await tester.pumpAndSettle();

    expect(find.text('account deletion is disabled'), findsOneWidget);
    expect(auth.isAuthenticated, isTrue);
    auth.dispose();
  });
}
