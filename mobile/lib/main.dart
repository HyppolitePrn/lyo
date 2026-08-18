import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import 'core/deep_links/deep_link_listener.dart';
import 'core/features/feature_flags_provider.dart';
import 'core/router/app_router.dart';
import 'core/theme/lyo_theme.dart';
import 'features/auth/providers/auth_notifier.dart';
import 'features/broadcaster/providers/broadcaster_notifier.dart';
import 'features/home/providers/home_notifier.dart';
import 'features/player/providers/player_notifier.dart';

Future<void> main() async {
  runApp(
    MultiProvider(
      providers: [
        Provider<FeatureFlags>(create: (_) => const FeatureFlags()),
        ChangeNotifierProvider(create: (_) => AuthNotifier()),
        ChangeNotifierProvider(create: (_) => HomeNotifier()),
        ChangeNotifierProvider(create: (_) => PlayerNotifier()),
        ChangeNotifierProvider(create: (_) => BroadcasterNotifier()),
      ],
      child: const LyoApp(),
    ),
  );
  await DeepLinkListener(appRouter).init();
}

class LyoApp extends StatelessWidget {
  const LyoApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp.router(
      title: 'Lyo',
      theme: lyoTheme(Brightness.dark),
      darkTheme: lyoTheme(Brightness.dark),
      themeMode: ThemeMode.dark,
      routerConfig: appRouter,
      debugShowCheckedModeBanner: false,
    );
  }
}
