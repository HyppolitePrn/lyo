import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../../core/features/feature_flags_provider.dart';
import '../../../core/theme/lyo_tokens.dart';
import '../../auth/providers/auth_notifier.dart';
import '../models/supervision_model.dart';
import '../providers/supervision_notifier.dart';

/// Admin-only view of platform health: current readings, incident counts, and
/// the feed of alerts that fired, each acknowledgeable in place.
///
/// Grafana remains the place to actually explore the data. This exists so an
/// admin knows whether anything is wrong without opening it.
class SupervisionScreen extends StatefulWidget {
  const SupervisionScreen({super.key});

  @override
  State<SupervisionScreen> createState() => _SupervisionScreenState();
}

class _SupervisionScreenState extends State<SupervisionScreen> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _load());
  }

  Future<void> _load() async {
    final token = context.read<AuthNotifier>().accessToken;
    if (token != null) {
      await context.read<SupervisionNotifier>().load(token);
    }
  }

  Future<void> _act(Incident incident, String action) async {
    final notifier = context.read<SupervisionNotifier>();
    final token = context.read<AuthNotifier>().accessToken;
    if (token == null) {
      return;
    }
    final ok = action == 'acknowledge'
        ? await notifier.acknowledge(incident.id, token)
        : await notifier.resolve(incident.id, token);

    if (!mounted) {
      return;
    }
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(
          ok
              ? 'Incident ${action}d.'
              : notifier.error ?? 'Could not update the incident.',
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    final auth = context.watch<AuthNotifier>();
    final flags = context.watch<FeatureFlags>();
    final notifier = context.watch<SupervisionNotifier>();

    return Scaffold(
      appBar: AppBar(title: const Text('Supervision')),
      body: !auth.isAdmin || !flags.isEnabled('admin_supervision')
          ? const _Unavailable()
          : RefreshIndicator(
              onRefresh: _load,
              child: _Body(notifier: notifier, dark: dark, onAct: _act),
            ),
    );
  }
}

class _Unavailable extends StatelessWidget {
  const _Unavailable();

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    return Center(
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: lyoPadH),
        child: Text(
          'Supervision is available to administrators only.',
          textAlign: TextAlign.center,
          style: TextStyle(
            color: dark ? lyoSubDark : lyoSubLight,
            fontSize: lyoBody1,
          ),
        ),
      ),
    );
  }
}

class _Body extends StatelessWidget {
  const _Body({
    required this.notifier,
    required this.dark,
    required this.onAct,
  });

  final SupervisionNotifier notifier;
  final bool dark;
  final Future<void> Function(Incident, String) onAct;

  @override
  Widget build(BuildContext context) {
    if (notifier.status == SupervisionStatus.loading &&
        notifier.summary == null) {
      return const Center(child: CircularProgressIndicator());
    }

    final textSub = dark ? lyoSubDark : lyoSubLight;
    final summary = notifier.summary;

    return ListView(
      // AlwaysScrollable keeps pull-to-refresh working when the feed is empty,
      // which is the state an admin most wants to be able to re-check.
      physics: const AlwaysScrollableScrollPhysics(),
      padding: const EdgeInsets.fromLTRB(
        lyoPadHMain,
        lyoGapL,
        lyoPadHMain,
        lyoGapXXL,
      ),
      children: [
        if (notifier.status == SupervisionStatus.error)
          _ErrorBanner(message: notifier.error ?? 'Something went wrong.'),
        if (summary != null) ...[
          _MetricsGrid(summary: summary, dark: dark),
          const SizedBox(height: lyoGapXL),
          _CountsRow(counts: summary.incidents, dark: dark),
          const SizedBox(height: lyoGapXL),
        ],
        Text(
          'Incidents',
          style: TextStyle(
            fontSize: lyoH2,
            fontWeight: FontWeight.w700,
            color: dark ? lyoTextDark : lyoTextLight,
          ),
        ),
        const SizedBox(height: lyoGapM),
        if (notifier.incidents.isEmpty)
          Padding(
            padding: const EdgeInsets.symmetric(vertical: lyoGapXL),
            child: Text(
              'Nothing has fired. Pull to refresh.',
              style: TextStyle(color: textSub, fontSize: lyoBody2),
            ),
          )
        else
          for (final incident in notifier.incidents) ...[
            _IncidentCard(
              incident: incident,
              dark: dark,
              busy: notifier.pendingIncidentId == incident.id,
              onAct: onAct,
            ),
            const SizedBox(height: lyoGapM),
          ],
      ],
    );
  }
}

class _ErrorBanner extends StatelessWidget {
  const _ErrorBanner({required this.message});
  final String message;

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.only(bottom: lyoGapL),
      padding: const EdgeInsets.all(lyoGapM),
      decoration: BoxDecoration(
        color: lyoError.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(lyoRadiusCard),
        border: Border.all(color: lyoError),
      ),
      child: Text(
        message,
        style: const TextStyle(color: lyoError, fontSize: lyoBody2),
      ),
    );
  }
}

// ── Metrics ───────────────────────────────────────────────────────────────────

class _MetricsGrid extends StatelessWidget {
  const _MetricsGrid({required this.summary, required this.dark});

  final SupervisionSummary summary;
  final bool dark;

  /// A null reading is shown as an em dash, never as zero: "no series yet" and
  /// "measured zero" mean different things to whoever is on call.
  static String _fmt(double? v, {int decimals = 0, String suffix = ''}) =>
      v == null ? '—' : '${v.toStringAsFixed(decimals)}$suffix';

  @override
  Widget build(BuildContext context) {
    final textSub = dark ? lyoSubDark : lyoSubLight;
    if (!summary.metricsAvailable) {
      return Container(
        padding: const EdgeInsets.all(lyoGapL),
        decoration: BoxDecoration(
          color: dark ? lyoSurfaceDark : lyoSurfaceLight,
          borderRadius: BorderRadius.circular(lyoRadiusCard),
          border: Border.all(color: dark ? lyoBorderDark : lyoBorderLight),
        ),
        child: Text(
          'Live metrics unavailable — Prometheus is unreachable. '
          'Incident counts below are still accurate.',
          style: TextStyle(color: textSub, fontSize: lyoBody2),
        ),
      );
    }

    final m = summary.metrics ?? const SupervisionMetrics();
    final tiles = <Widget>[
      _MetricTile(
        label: 'Backend',
        value: m.backendUp == null ? '—' : (m.backendUp! ? 'Up' : 'Down'),
        dark: dark,
        alert: m.backendUp == false,
      ),
      _MetricTile(
        label: 'Live streams',
        value: _fmt(m.liveStreams),
        dark: dark,
      ),
      _MetricTile(
        label: 'Listeners',
        value: _fmt(m.activeListeners),
        dark: dark,
      ),
      _MetricTile(
        label: 'Requests/s',
        value: _fmt(m.requestsPerSecond, decimals: 2),
        dark: dark,
      ),
      _MetricTile(
        label: 'Errors',
        value: _fmt(m.errorRatePercent, decimals: 2, suffix: '%'),
        dark: dark,
        alert: (m.errorRatePercent ?? 0) > 5,
      ),
      _MetricTile(
        label: 'Latency p95',
        value: m.latencyP95Seconds == null
            ? '—'
            : '${(m.latencyP95Seconds! * 1000).toStringAsFixed(0)} ms',
        dark: dark,
        alert: (m.latencyP95Seconds ?? 0) > 1,
      ),
      _MetricTile(
        // The keystone experience metric: above zero, someone is hearing gaps.
        label: 'Audio dropped/s',
        value: _fmt(m.chunksDroppedPerSecond, decimals: 2),
        dark: dark,
        alert: (m.chunksDroppedPerSecond ?? 0) > 0,
      ),
    ];

    return GridView.count(
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      crossAxisCount: 2,
      mainAxisSpacing: lyoGapM,
      crossAxisSpacing: lyoGapM,
      childAspectRatio: 2.1,
      children: tiles,
    );
  }
}

class _MetricTile extends StatelessWidget {
  const _MetricTile({
    required this.label,
    required this.value,
    required this.dark,
    this.alert = false,
  });

  final String label;
  final String value;
  final bool dark;
  final bool alert;

  @override
  Widget build(BuildContext context) {
    final textSub = dark ? lyoSubDark : lyoSubLight;
    return Semantics(
      label: '$label: $value',
      excludeSemantics: true,
      child: Container(
        padding: const EdgeInsets.all(lyoGapM),
        decoration: BoxDecoration(
          color: dark ? lyoSurfaceDark : lyoSurfaceLight,
          borderRadius: BorderRadius.circular(lyoRadiusCard),
          border: Border.all(
            color: alert ? lyoError : (dark ? lyoBorderDark : lyoBorderLight),
          ),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Text(
              label,
              style: TextStyle(color: textSub, fontSize: lyoSmall),
            ),
            const SizedBox(height: lyoGapXS),
            Text(
              value,
              style: TextStyle(
                color: alert ? lyoError : (dark ? lyoTextDark : lyoTextLight),
                fontSize: lyoH2,
                fontWeight: FontWeight.w700,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _CountsRow extends StatelessWidget {
  const _CountsRow({required this.counts, required this.dark});

  final IncidentCounts counts;
  final bool dark;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(
          child: _MetricTile(
            label: 'Firing',
            value: '${counts.firing}',
            dark: dark,
            alert: counts.criticalFiring > 0,
          ),
        ),
        const SizedBox(width: lyoGapM),
        Expanded(
          child: _MetricTile(
            label: 'Acknowledged',
            value: '${counts.acknowledged}',
            dark: dark,
          ),
        ),
        const SizedBox(width: lyoGapM),
        Expanded(
          child: _MetricTile(
            label: 'Resolved 24h',
            value: '${counts.resolved24h}',
            dark: dark,
          ),
        ),
      ],
    );
  }
}

// ── Incident card ─────────────────────────────────────────────────────────────

class _IncidentCard extends StatelessWidget {
  const _IncidentCard({
    required this.incident,
    required this.dark,
    required this.busy,
    required this.onAct,
  });

  final Incident incident;
  final bool dark;
  final bool busy;
  final Future<void> Function(Incident, String) onAct;

  Color get _severityColor => switch (incident.severity) {
    IncidentSeverity.critical => lyoError,
    IncidentSeverity.warning => lyoAccent,
    IncidentSeverity.info => lyoSubDark,
  };

  @override
  Widget build(BuildContext context) {
    final textSub = dark ? lyoSubDark : lyoSubLight;
    final textPrimary = dark ? lyoTextDark : lyoTextLight;
    final resolved = incident.status == IncidentStatus.resolved;

    return Container(
      padding: const EdgeInsets.all(lyoGapL),
      decoration: BoxDecoration(
        color: dark ? lyoSurfaceDark : lyoSurfaceLight,
        borderRadius: BorderRadius.circular(lyoRadiusCard),
        border: Border.all(
          color: resolved
              ? (dark ? lyoBorderDark : lyoBorderLight)
              : _severityColor,
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              _Chip(
                text: incident.severity.name,
                color: _severityColor,
              ),
              const SizedBox(width: lyoGapS),
              // The family is on the card because it decides what an admin
              // does next: fix a bug, or add capacity.
              _Chip(
                text: incident.family.name,
                color: incident.family == IncidentFamily.experience
                    ? lyoAccent
                    : textSub,
              ),
              const Spacer(),
              Text(
                incident.status.name,
                style: TextStyle(color: textSub, fontSize: lyoSmall),
              ),
            ],
          ),
          const SizedBox(height: lyoGapM),
          Text(
            incident.title,
            style: TextStyle(
              color: textPrimary,
              fontSize: lyoBody1,
              fontWeight: FontWeight.w700,
            ),
          ),
          if (incident.summary?.isNotEmpty ?? false) ...[
            const SizedBox(height: lyoGapXS),
            Text(
              incident.summary!,
              style: TextStyle(color: textSub, fontSize: lyoCaption),
            ),
          ],
          const SizedBox(height: lyoGapS),
          Text(
            'Started ${_ago(incident.startedAt)}',
            style: TextStyle(color: textSub, fontSize: lyoSmall),
          ),
          if (!resolved) ...[
            const SizedBox(height: lyoGapM),
            Row(
              children: [
                if (incident.status == IncidentStatus.firing)
                  TextButton(
                    onPressed: busy
                        ? null
                        : () => onAct(incident, 'acknowledge'),
                    child: const Text('Acknowledge'),
                  ),
                TextButton(
                  onPressed: busy ? null : () => onAct(incident, 'resolve'),
                  child: const Text('Resolve'),
                ),
              ],
            ),
          ],
        ],
      ),
    );
  }
}

class _Chip extends StatelessWidget {
  const _Chip({required this.text, required this.color});

  final String text;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: lyoGapS, vertical: 2),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.15),
        borderRadius: BorderRadius.circular(lyoRadiusChip),
      ),
      child: Text(
        text,
        style: TextStyle(
          color: color,
          fontSize: lyoTiny,
          fontWeight: FontWeight.w700,
        ),
      ),
    );
  }
}

String _ago(DateTime when) {
  final d = DateTime.now().difference(when);
  if (d.inMinutes < 1) {
    return 'just now';
  }
  if (d.inHours < 1) {
    return '${d.inMinutes} min ago';
  }
  if (d.inDays < 1) {
    return '${d.inHours} h ago';
  }
  return '${d.inDays} d ago';
}
