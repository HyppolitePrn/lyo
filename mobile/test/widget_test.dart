import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';

import 'package:mobile/core/features/feature_flags_provider.dart';
import 'package:mobile/core/router/app_router.dart';
import 'package:mobile/features/auth/providers/auth_notifier.dart';
import 'package:mobile/features/auth/services/token_store.dart';
import 'package:mobile/features/broadcaster/providers/broadcaster_notifier.dart';
import 'package:mobile/features/home/providers/home_notifier.dart';
import 'package:mobile/features/player/providers/player_notifier.dart';
import 'package:mobile/main.dart';

void main() {
  testWidgets('LyoApp renders without crashing', (WidgetTester tester) async {
    tester.view.physicalSize = const Size(390, 844); // iPhone 14-ish
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<FeatureFlags>(create: (_) => FeatureFlags()),
          ChangeNotifierProvider(
            create: (_) => AuthNotifier(tokenStore: InMemoryTokenStore()),
          ),
          ChangeNotifierProvider(create: (_) => HomeNotifier()),
          ChangeNotifierProvider(create: (_) => PlayerNotifier()),
          ChangeNotifierProvider(create: (_) => BroadcasterNotifier()),
        ],
        child: LyoApp(router: createAppRouter()),
      ),
    );
    expect(find.text('Listen live.\nHear everything.'), findsOneWidget);
    expect(find.text('Get Started'), findsOneWidget);
  });
}
