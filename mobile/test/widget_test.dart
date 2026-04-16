import 'package:flutter_test/flutter_test.dart';

import 'package:mobile/main.dart';

void main() {
  testWidgets('LyoApp renders without crashing', (WidgetTester tester) async {
    await tester.pumpWidget(const LyoApp());
    expect(find.text('Lyo — StreamPulse'), findsOneWidget);
  });
}
