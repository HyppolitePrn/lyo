import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import 'package:mobile/core/features/feature_flags_provider.dart';
import 'package:mobile/core/router/app_router.dart';
import 'package:mobile/core/theme/lyo_theme.dart';
import 'package:mobile/features/auth/providers/auth_notifier.dart';
import 'package:mobile/features/auth/services/token_store.dart';

void main() {
  Future<GoRouter> pumpApp(WidgetTester tester) async {
    // Generous surface: the test font renderer makes every glyph a square em,
    // so a phone-width viewport overflows rows that fit fine on a device.
    tester.view.physicalSize = const Size(1000, 2000);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);

    final router = createAppRouter();
    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<FeatureFlags>(create: (_) => FeatureFlags()),
          ChangeNotifierProvider(
            create: (_) => AuthNotifier(tokenStore: InMemoryTokenStore()),
          ),
        ],
        child: MaterialApp.router(
            theme: lyoTheme(Brightness.dark),
            routerConfig: router,
          ),
      ),
    );
    await tester.pumpAndSettle();
    return router;
  }

  // The top of the stack, not `currentConfiguration.uri` — the latter stays on
  // the base location for imperative pushes.
  String location(GoRouter router) =>
      router.routerDelegate.currentConfiguration.matches.last.matchedLocation;

  testWidgets('back arrow pops when login was pushed', (tester) async {
    final router = await pumpApp(tester);
    router.push('/login');
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Back'));
    await tester.pumpAndSettle();

    expect(location(router), '/splash');
  });

  testWidgets('back arrow falls back to splash when the stack was replaced',
      (tester) async {
    final router = await pumpApp(tester);
    // What a sign-out or a guest tapping "sign in" used to do: replace the
    // stack, leaving nothing to pop.
    router.go('/login');
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Back'));
    await tester.pumpAndSettle();

    // Splash is reachable again, so "Get Started" (create an account) and
    // "Browse without account" are no longer out of reach.
    expect(location(router), '/splash');
    expect(find.text('Get Started'), findsOneWidget);
    expect(find.text('Browse without account'), findsOneWidget);
  });

  testWidgets('sign-up link from login reaches the register screen',
      (tester) async {
    final router = await pumpApp(tester);
    router.go('/login');
    await tester.pumpAndSettle();

    await tester.tap(find.text('Sign up'));
    await tester.pumpAndSettle();
    expect(location(router), '/register');
    expect(find.text('Create account'), findsOneWidget);
  });

  testWidgets('register back arrow falls back to splash', (tester) async {
    final router = await pumpApp(tester);
    router.go('/register');
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Back'));
    await tester.pumpAndSettle();

    expect(location(router), '/splash');
  });
}
