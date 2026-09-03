/// Models for the admin supervision screen: the incident feed and the
/// platform's current readings.
library;

/// How serious an incident is, as labelled by the Grafana alert rule.
enum IncidentSeverity { critical, warning, info }

/// Which of the two questions an incident answers.
///
/// [technical] means the platform itself failed. [experience] means nothing is
/// broken and listeners are suffering anyway — dropped audio, cut-off
/// sessions. The distinction is kept all the way to the UI because the two
/// demand different reactions: a bug to fix versus a capacity decision.
enum IncidentFamily { technical, experience }

/// Where the incident is in its lifecycle.
enum IncidentStatus { firing, acknowledged, resolved }

T _parseEnum<T>(List<T> values, String? raw, T fallback) {
  if (raw == null) {
    return fallback;
  }
  for (final v in values) {
    if (v.toString().split('.').last == raw) {
      return v;
    }
  }
  return fallback;
}

class Incident {
  const Incident({
    required this.id,
    required this.title,
    required this.severity,
    required this.family,
    required this.status,
    required this.startedAt,
    this.summary,
    this.ruleUid,
    this.dashboardUrl,
    this.acknowledgedAt,
    this.resolvedAt,
  });

  final String id;
  final String title;
  final String? summary;
  final String? ruleUid;
  final String? dashboardUrl;
  final IncidentSeverity severity;
  final IncidentFamily family;
  final IncidentStatus status;
  final DateTime startedAt;
  final DateTime? acknowledgedAt;
  final DateTime? resolvedAt;

  static DateTime? _date(dynamic v) =>
      v is String ? DateTime.tryParse(v)?.toLocal() : null;

  factory Incident.fromJson(Map<String, dynamic> json) {
    return Incident(
      id: json['id'] as String,
      title: json['title'] as String? ?? 'Unnamed alert',
      summary: json['summary'] as String?,
      ruleUid: json['rule_uid'] as String?,
      dashboardUrl: json['dashboard_url'] as String?,
      severity: _parseEnum(
        IncidentSeverity.values,
        json['severity'] as String?,
        IncidentSeverity.warning,
      ),
      family: _parseEnum(
        IncidentFamily.values,
        json['family'] as String?,
        IncidentFamily.technical,
      ),
      status: _parseEnum(
        IncidentStatus.values,
        json['status'] as String?,
        IncidentStatus.firing,
      ),
      startedAt: _date(json['started_at']) ?? DateTime.now(),
      acknowledgedAt: _date(json['acknowledged_at']),
      resolvedAt: _date(json['resolved_at']),
    );
  }
}

class IncidentCounts {
  const IncidentCounts({
    this.firing = 0,
    this.acknowledged = 0,
    this.criticalFiring = 0,
    this.resolved24h = 0,
  });

  final int firing;
  final int acknowledged;
  final int criticalFiring;
  final int resolved24h;

  factory IncidentCounts.fromJson(Map<String, dynamic> json) => IncidentCounts(
    firing: json['firing'] as int? ?? 0,
    acknowledged: json['acknowledged'] as int? ?? 0,
    criticalFiring: json['critical_firing'] as int? ?? 0,
    resolved24h: json['resolved_24h'] as int? ?? 0,
  );
}

/// Current platform readings.
///
/// Every field is nullable on purpose: the backend omits a metric that has no
/// series yet rather than sending zero, so the screen can show "—" for
/// "nothing measured" instead of claiming a confident zero.
class SupervisionMetrics {
  const SupervisionMetrics({
    this.backendUp,
    this.liveStreams,
    this.activeListeners,
    this.requestsPerSecond,
    this.errorRatePercent,
    this.latencyP95Seconds,
    this.chunksDroppedPerSecond,
  });

  final bool? backendUp;
  final double? liveStreams;
  final double? activeListeners;
  final double? requestsPerSecond;
  final double? errorRatePercent;
  final double? latencyP95Seconds;
  final double? chunksDroppedPerSecond;

  static double? _num(dynamic v) => v is num ? v.toDouble() : null;

  factory SupervisionMetrics.fromJson(Map<String, dynamic> json) =>
      SupervisionMetrics(
        backendUp: json['backend_up'] as bool?,
        liveStreams: _num(json['live_streams']),
        activeListeners: _num(json['active_listeners']),
        requestsPerSecond: _num(json['requests_per_second']),
        errorRatePercent: _num(json['error_rate_percent']),
        latencyP95Seconds: _num(json['latency_p95_seconds']),
        chunksDroppedPerSecond: _num(json['chunks_dropped_per_second']),
      );
}

class SupervisionSummary {
  const SupervisionSummary({
    required this.incidents,
    required this.metricsAvailable,
    required this.generatedAt,
    this.metrics,
  });

  final IncidentCounts incidents;

  /// False when Prometheus is unreachable or unconfigured. The incident counts
  /// remain valid in that case — they come from the platform's own database.
  final bool metricsAvailable;
  final SupervisionMetrics? metrics;
  final DateTime generatedAt;

  factory SupervisionSummary.fromJson(Map<String, dynamic> json) {
    final metrics = json['metrics'];
    return SupervisionSummary(
      incidents: IncidentCounts.fromJson(
        (json['incidents'] as Map<String, dynamic>?) ?? const {},
      ),
      metricsAvailable: json['metrics_available'] as bool? ?? false,
      metrics: metrics is Map<String, dynamic>
          ? SupervisionMetrics.fromJson(metrics)
          : null,
      generatedAt:
          DateTime.tryParse(json['generated_at'] as String? ?? '')?.toLocal() ??
          DateTime.now(),
    );
  }
}
