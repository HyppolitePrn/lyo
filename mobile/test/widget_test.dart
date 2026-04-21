import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:mobile/main.dart';

void main() {
  testWidgets('LyoApp renders without crashing', (WidgetTester tester) async {
    tester.view.physicalSize = const Size(390, 844); // iPhone 14-ish
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);

    await tester.pumpWidget(const ProviderScope(child: LyoApp()));
    expect(find.text('Listen live.\nHear everything.'), findsOneWidget);
    expect(find.text('Get Started'), findsOneWidget);
  });
}
