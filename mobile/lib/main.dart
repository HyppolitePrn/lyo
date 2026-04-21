import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'core/router/app_router.dart';
import 'core/theme/lyo_theme.dart';

void main() {
  runApp(const ProviderScope(child: LyoApp()));
}

class LyoApp extends ConsumerWidget {
  const LyoApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final router = ref.watch(routerProvider);
    return MaterialApp.router(
      title: 'Lyo',
      theme: lyoTheme(Brightness.dark),
      darkTheme: lyoTheme(Brightness.dark),
      themeMode: ThemeMode.dark,
      routerConfig: router,
      debugShowCheckedModeBanner: false,
    );
  }
}
