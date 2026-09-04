import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';

import 'package:mobile/core/features/feature_flags_provider.dart';
import 'package:mobile/core/theme/lyo_theme.dart';
import 'package:mobile/features/auth/providers/auth_notifier.dart';
import 'package:mobile/features/auth/services/token_store.dart';
import 'package:mobile/features/home/providers/home_notifier.dart';
import 'package:mobile/features/home/screens/home_screen.dart';
import 'package:mobile/features/player/providers/player_notifier.dart';
import 'package:mobile/features/player/providers/recorded_player_notifier.dart';
import 'package:mobile/features/player/services/lyo_audio_handler.dart';

/// A HomeNotifier that never touches the network — the tab machinery under
/// test is the same either way.
class _OfflineHomeNotifier extends HomeNotifier {
  @override
  void refreshLiveStreams(String token) {
    liveStreams = Future<List<Never>>.value(const []);
    notifyListeners();
  }
}

void main() {
  Future<void> pumpHome(WidgetTester tester) async {
    // Generous surface: the test font renderer makes every glyph a square em,
    // so a phone-width viewport overflows rows that fit fine on a device.
    tester.view.physicalSize = const Size(1000, 2000);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<FeatureFlags>(create: (_) => FeatureFlags()),
          ChangeNotifierProvider(
            create: (_) => AuthNotifier(tokenStore: InMemoryTokenStore()),
          ),
          ChangeNotifierProvider<HomeNotifier>(
            create: (_) => _OfflineHomeNotifier(),
          ),
          ChangeNotifierProvider(create: (_) => PlayerNotifier()),
          ChangeNotifierProvider(
            create: (_) =>
                RecordedPlayerNotifier(audioHandler: LyoAudioHandler()),
          ),
        ],
        child: MaterialApp(
          theme: lyoTheme(Brightness.dark),
          home: const HomeScreen(),
        ),
      ),
    );
    await tester.pump();
  }

  Future<void> tapTab(WidgetTester tester, String label) async {
    await tester.tap(find.text(label));
    await tester.pump();
  }

  // Regression: _HomeBody used to call HomeNotifier.refreshLiveStreams — and
  // therefore notifyListeners() — synchronously from initState. On the way
  // back to the Home tab that ran inside the build pass rebuilding HomeScreen
  // itself, which left HomeScreen's element permanently marked dirty and never
  // scheduled: the bottom bar went dead and the app was stuck on Home.
  testWidgets('bottom navigation still works after leaving and re-entering '
      'the Home tab', (tester) async {
    await pumpHome(tester);
    expect(find.text('Live Now'), findsOneWidget);

    await tapTab(tester, 'Browse');
    expect(find.text('Live Now'), findsNothing);

    await tapTab(tester, 'Home');
    expect(find.text('Live Now'), findsOneWidget);

    // The tap that used to do nothing at all.
    await tapTab(tester, 'Search');
    expect(find.text('Search — coming soon'), findsOneWidget);
    expect(find.text('Live Now'), findsNothing);
  });
}
