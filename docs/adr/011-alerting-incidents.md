# ADR 011 — Alerts Become Incidents Owned by the Platform

**Status:** Accepted
**Date:** 2026-09-03

## Context

With [ADR 009](009-observability-otlp.md) finally emitting all three signals, Grafana can evaluate rules
and fire. That closes the detection question and opens the response one, and they are not the same
question.

Grafana's own notification is transient in three ways that matter. It lives in a UI nobody is watching at
three in the morning. It does not remember, across its own restart, that a human already looked at an
alert — so the second person on call cannot tell whether the first is already working on it. And it is not
reachable from a phone, which is where an administrator actually is when something fires.

There is also a question of *what* to alert on. A 500 response and a dropped audio chunk are both bad, but
only the first means code failed; the second means the back-pressure policy of
[ADR 003](003-streaming-hub-pattern.md) worked exactly as designed and a listener still heard a gap.
Alerting that cannot express that difference sends both to the same person with the same words, and they
fix the wrong thing.

The options were: rely on Grafana notifications alone (nothing to build, nothing durable); add
Alertmanager (a fourth piece of infrastructure, still nothing an admin can acknowledge from the app); or
have the platform own the incident.

## Decision

**Grafana detects; Lyo owns the incident.** Every alert is POSTed to `/internal/alerts`, and the backend
turns it into a durable, acknowledgeable row.

Four consequences of that decision are worth stating explicitly, because each was a choice with an
alternative:

**Alert rules are code.** They live in `docker/grafana/provisioning/alerting/rules.yml` with UI edits
disabled. A rule someone tuned by hand in the UI at 3am and never wrote down is a rule that vanishes with
the container.

**Every rule declares a family.** `family=technical` means the platform failed; `family=experience` means
nothing is broken and users are suffering anyway. The label survives into the incident, the notification
and the mobile screen, because it decides what the reader does next: fix a bug, or make a capacity
decision. `lyo_chunks_dropped_total` is the alert that only exists because of this distinction — it would
never appear in an error rate.

**Incidents deduplicate on Grafana's fingerprint**, through a partial unique index over unresolved rows.
A flapping alert updates one incident instead of queueing dozens, and only a *newly inserted* incident
mails the administrators. Renotifying on every evaluation is precisely how a mailbox becomes noise that
people filter away, at which point the alerting is worse than none because it is believed to work.

**The webhook is authenticated by a shared secret, and an unset secret disables the endpoint.** Grafana's
webhook contact point cannot mint a JWT, so the platform's own JWT scheme does not apply here. Treating an
empty secret as "accept anything" would leave an unauthenticated incident-injection route open on every
deployment that forgot to configure it; failing closed is the only safe default.

## Consequences

- An administrator sees platform state and open incidents from the mobile app, and acknowledgement is
  recorded against their user — so a second admin can tell the incident is already owned.
- `GET /admin/supervision` runs the *same* PromQL expressions as the corresponding Grafana panels, so the
  two can never disagree about what "the error rate" means. It degrades rather than fails when Prometheus
  is unreachable: the incident counts come from our own database and are the part most worth having.
- One more moving part: if the backend is down, incidents are not recorded. This is accepted, because the
  `Backend unreachable` rule is exactly the one whose *absence* of an incident is itself the signal, and
  Grafana's own alert list remains as a fallback.
- Nothing here replaces Grafana for investigation. The mobile screen answers "is anything wrong?"; the
  dashboards answer "why?".
