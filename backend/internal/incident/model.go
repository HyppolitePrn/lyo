// Package incident turns Grafana alert notifications into a durable,
// admin-visible incident feed.
//
// Grafana already alerts. What it does not do is survive its own restart with
// the knowledge that a human looked at an alert, and it is not something a
// mobile admin can be expected to log into. An incident is therefore the
// platform's own record of an alert that fired: deduplicated, acknowledgeable,
// and readable through the API.
package incident

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned when an incident lookup yields no rows.
var ErrNotFound = errors.New("incident not found")

// Severity mirrors the severity label carried by the Grafana alert rules.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityWarning  Severity = "warning"
	SeverityInfo     Severity = "info"
)

// Family separates the two questions of ADR 009. Technical incidents mean the
// code failed; experience incidents mean the system behaved exactly as
// designed and users suffered anyway (dropped audio, cut-off listeners).
// Conflating them leads to fixing the wrong thing.
type Family string

const (
	FamilyTechnical  Family = "technical"
	FamilyExperience Family = "experience"
)

// Status is the incident's lifecycle position.
type Status string

const (
	// StatusFiring — the alert is active and nobody has claimed it.
	StatusFiring Status = "firing"
	// StatusAcknowledged — an admin has seen it and is on it. The alert may
	// still be firing; this only records that a human is aware.
	StatusAcknowledged Status = "acknowledged"
	// StatusResolved — the condition cleared, or an admin closed it by hand.
	StatusResolved Status = "resolved"
)

// Incident is one alert instance, deduplicated by fingerprint.
type Incident struct {
	ID             string
	Fingerprint    string
	RuleUID        string
	Title          string
	Summary        string
	Severity       Severity
	Family         Family
	Status         Status
	DashboardURL   string
	StartedAt      time.Time
	AcknowledgedAt *time.Time
	AcknowledgedBy *string
	ResolvedAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Counts summarises the feed for the mobile supervision screen, which needs a
// headline number long before it needs the list itself.
type Counts struct {
	Firing         int
	Acknowledged   int
	CriticalFiring int
	Resolved24h    int
}

// Repository is the persistence contract for incidents.
type Repository interface {
	// Open records a newly firing alert. If an unresolved incident already
	// exists for the fingerprint it is updated in place and returned with
	// created=false, so a flapping alert does not flood the feed.
	Open(ctx context.Context, in Incident) (result *Incident, created bool, err error)
	// ResolveByFingerprint closes the open incident for a fingerprint. It is a
	// no-op — not an error — when nothing is open, because Grafana sends
	// resolved notifications for alerts this instance never saw fire.
	ResolveByFingerprint(ctx context.Context, fingerprint string, at time.Time) error

	List(ctx context.Context, status Status, limit int) ([]Incident, error)
	Get(ctx context.Context, id string) (*Incident, error)
	Acknowledge(ctx context.Context, id, adminID string, at time.Time) (*Incident, error)
	Resolve(ctx context.Context, id string, at time.Time) (*Incident, error)
	CountsSince(ctx context.Context, since time.Time) (Counts, error)

	// AdminEmails lists the addresses to notify. Incidents are an admin-only
	// concern, so this is the only audience the notifier ever sees.
	AdminEmails(ctx context.Context) ([]string, error)
}
