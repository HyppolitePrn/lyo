import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';

import 'package:mobile/core/theme/lyo_theme.dart';
import 'package:mobile/features/player/services/volume_controller.dart';
import 'package:mobile/features/player/widgets/volume_control.dart';

void main() {
  Future<VolumeController> pumpControl(WidgetTester tester) async {
    final volume = VolumeController();
    await tester.pumpWidget(
      ChangeNotifierProvider.value(
        value: volume,
        child: MaterialApp(
          theme: lyoTheme(Brightness.dark),
          home: const Scaffold(body: VolumeControl()),
        ),
      ),
    );
    return volume;
  }

  testWidgets('dragging the slider changes the level', (tester) async {
    final volume = await pumpControl(tester);

    // Drag to the far left: the slider spans the full width, so this bottoms
    // the level out whatever the test viewport is.
    await tester.drag(find.byType(Slider), const Offset(-500, 0));
    await tester.pump();

    expect(volume.level, 0.0);
  });

  testWidgets('the speaker button mutes and unmutes', (tester) async {
    final volume = await pumpControl(tester);

    await tester.tap(find.byType(IconButton));
    await tester.pump();
    expect(volume.isMuted, isTrue);
    expect(find.byIcon(Icons.volume_off), findsOneWidget);

    await tester.tap(find.byType(IconButton));
    await tester.pump();
    expect(volume.isMuted, isFalse);
    expect(find.byIcon(Icons.volume_up), findsOneWidget);
  });
}
