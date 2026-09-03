import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import 'package:mobile/core/api/api_client.dart';
import 'package:mobile/features/admin/models/feature_flag_model.dart';
import 'package:mobile/features/admin/providers/admin_feature_notifier.dart';
import 'package:mobile/features/admin/services/admin_feature_service.dart';

class MockAdminFeatureService extends Mock implements AdminFeatureService {}

void main() {
  late MockAdminFeatureService mockSvc;
  late AdminFeatureNotifier notifier;

  setUp(() {
    mockSvc = MockAdminFeatureService();
    notifier = AdminFeatureNotifier(adminFeatureService: mockSvc);
  });

  final flag = FeatureFlag(
    id: 'f-1',
    name: 'chat_websocket',
    enabled: false,
    description: 'Live chat between listeners',
    updatedAt: DateTime(2026),
  );

  test('load success sets ready status and flags', () async {
    when(() => mockSvc.listFlags(token: any(named: 'token')))
        .thenAnswer((_) async => [flag]);

    await notifier.load('tok');

    expect(notifier.status, AdminFeatureStatus.ready);
    expect(notifier.flags, [flag]);
  });

  test('load failure sets error status and message', () async {
    when(() => mockSvc.listFlags(token: any(named: 'token')))
        .thenThrow(const ApiException(403, 'forbidden'));

    await notifier.load('tok');

    expect(notifier.status, AdminFeatureStatus.error);
    expect(notifier.error, 'forbidden');
  });

  test('toggle replaces the flag with the updated one', () async {
    when(() => mockSvc.listFlags(token: any(named: 'token')))
        .thenAnswer((_) async => [flag]);
    await notifier.load('tok');

    final updated = flag.copyWith(enabled: true);
    when(() => mockSvc.toggleFlag(
          name: 'chat_websocket',
          enabled: true,
          token: 'tok',
        )).thenAnswer((_) async => updated);

    final ok = await notifier.toggle(
        name: 'chat_websocket', enabled: true, token: 'tok');

    expect(ok, isTrue);
    expect(notifier.flags.single.enabled, isTrue);
  });

  test('toggle failure sets error message and returns false', () async {
    when(() => mockSvc.listFlags(token: any(named: 'token')))
        .thenAnswer((_) async => [flag]);
    await notifier.load('tok');

    when(() => mockSvc.toggleFlag(
          name: any(named: 'name'),
          enabled: any(named: 'enabled'),
          token: any(named: 'token'),
        )).thenThrow(const ApiException(404, 'not found'));

    final ok = await notifier.toggle(
        name: 'chat_websocket', enabled: true, token: 'tok');

    expect(ok, isFalse);
    expect(notifier.error, 'not found');
  });
}
