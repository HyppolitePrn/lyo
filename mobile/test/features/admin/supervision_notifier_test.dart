import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import 'package:mobile/core/api/api_client.dart';
import 'package:mobile/features/admin/models/supervision_model.dart';
import 'package:mobile/features/admin/providers/supervision_notifier.dart';
import 'package:mobile/features/admin/services/supervision_service.dart';

class MockSupervisionService extends Mock implements SupervisionService {}

Incident incident({
  String id = 'i-1',
  IncidentStatus status = IncidentStatus.firing,
}) => Incident(
  id: id,
  title: 'The hub is dropping audio chunks',
  severity: IncidentSeverity.critical,
  family: IncidentFamily.experience,
  status: status,
  startedAt: DateTime(2026, 9, 3, 12),
);

SupervisionSummary summary({int firing = 1}) => SupervisionSummary(
  incidents: IncidentCounts(firing: firing, criticalFiring: firing),
  metricsAvailable: true,
  metrics: const SupervisionMetrics(backendUp: true, liveStreams: 2),
  generatedAt: DateTime(2026, 9, 3, 12),
);

void main() {
  late MockSupervisionService svc;
  late SupervisionNotifier notifier;

  setUp(() {
    svc = MockSupervisionService();
    notifier = SupervisionNotifier(service: svc);
  });

  test('load fetches the summary and the feed together', () async {
    when(() => svc.getSummary(any())).thenAnswer((_) async => summary());
    when(
      () => svc.listIncidents(token: any(named: 'token')),
    ).thenAnswer((_) async => [incident()]);

    await notifier.load('tok');

    expect(notifier.status, SupervisionStatus.ready);
    expect(notifier.incidents, hasLength(1));
    expect(notifier.summary?.incidents.firing, 1);
    expect(notifier.summary?.metrics?.liveStreams, 2);
  });

  test('load surfaces the API message on failure', () async {
    when(
      () => svc.getSummary(any()),
    ).thenThrow(const ApiException(403, 'forbidden'));
    when(
      () => svc.listIncidents(token: any(named: 'token')),
    ).thenAnswer((_) async => []);

    await notifier.load('tok');

    expect(notifier.status, SupervisionStatus.error);
    expect(notifier.error, 'forbidden');
  });

  // Acknowledging changes both the row and the header count; refreshing only
  // one would leave the screen contradicting itself.
  test('acknowledge replaces the incident and refreshes the counts', () async {
    when(() => svc.getSummary(any())).thenAnswer((_) async => summary());
    when(
      () => svc.listIncidents(token: any(named: 'token')),
    ).thenAnswer((_) async => [incident()]);
    await notifier.load('tok');

    when(
      () => svc.updateIncident(
        id: any(named: 'id'),
        action: any(named: 'action'),
        token: any(named: 'token'),
      ),
    ).thenAnswer((_) async => incident(status: IncidentStatus.acknowledged));
    when(
      () => svc.getSummary(any()),
    ).thenAnswer((_) async => summary(firing: 0));

    final ok = await notifier.acknowledge('i-1', 'tok');

    expect(ok, isTrue);
    expect(notifier.incidents.single.status, IncidentStatus.acknowledged);
    expect(notifier.summary?.incidents.firing, 0);
    expect(notifier.pendingIncidentId, isNull);
  });

  test('a failed update keeps the incident and reports the error', () async {
    when(() => svc.getSummary(any())).thenAnswer((_) async => summary());
    when(
      () => svc.listIncidents(token: any(named: 'token')),
    ).thenAnswer((_) async => [incident()]);
    await notifier.load('tok');

    when(
      () => svc.updateIncident(
        id: any(named: 'id'),
        action: any(named: 'action'),
        token: any(named: 'token'),
      ),
    ).thenThrow(const ApiException(404, 'no incident to update'));

    final ok = await notifier.resolve('i-1', 'tok');

    expect(ok, isFalse);
    expect(notifier.error, 'no incident to update');
    expect(notifier.incidents.single.status, IncidentStatus.firing);
    // The row must become interactive again, or a failed tap would lock it.
    expect(notifier.pendingIncidentId, isNull);
  });

  group('parsing', () {
    test('maps the backend payload', () {
      final parsed = SupervisionSummary.fromJson(const {
        'incidents': {
          'firing': 2,
          'acknowledged': 1,
          'critical_firing': 1,
          'resolved_24h': 5,
        },
        'metrics_available': true,
        'metrics': {'backend_up': true, 'live_streams': 3},
        'generated_at': '2026-09-03T12:00:00Z',
      });

      expect(parsed.incidents.firing, 2);
      expect(parsed.incidents.resolved24h, 5);
      expect(parsed.metrics?.backendUp, isTrue);
      expect(parsed.metrics?.liveStreams, 3);
      // Omitted by the backend because it has no series yet — it must stay
      // null so the UI shows "—" rather than a confident zero.
      expect(parsed.metrics?.errorRatePercent, isNull);
    });

    test('survives a payload with no metrics at all', () {
      final parsed = SupervisionSummary.fromJson(const {
        'incidents': {'firing': 1},
        'metrics_available': false,
        'generated_at': '2026-09-03T12:00:00Z',
      });

      expect(parsed.metricsAvailable, isFalse);
      expect(parsed.metrics, isNull);
      expect(parsed.incidents.firing, 1);
    });

    test('falls back rather than throwing on unknown enum values', () {
      final parsed = Incident.fromJson(const {
        'id': 'i-9',
        'title': 'Something new',
        'severity': 'catastrophic',
        'family': 'unknown',
        'status': 'weird',
        'started_at': '2026-09-03T12:00:00Z',
      });

      expect(parsed.severity, IncidentSeverity.warning);
      expect(parsed.family, IncidentFamily.technical);
      expect(parsed.status, IncidentStatus.firing);
    });
  });
}
