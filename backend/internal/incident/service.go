package incident

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// Notifier delivers an incident to a human. It is satisfied by pkg/mailer.Sender.
type Notifier interface {
	Send(ctx context.Context, to, subject, body string) error
}

// notifyTimeout bounds the background fan-out of incident emails. It is
// deliberately generous: one slow SMTP server must not silence the others.
const notifyTimeout = 30 * time.Second

// Service applies incoming alerts to the incident feed and tells admins.
type Service struct {
	repo   Repository
	mail   Notifier
	logger *slog.Logger
}

func NewService(repo Repository, mail Notifier, logger *slog.Logger) *Service {
	return &Service{repo: repo, mail: mail, logger: logger}
}

// Ingest applies one batch of alert notifications. Firing alerts open (or
// refresh) an incident; resolved alerts close it.
//
// A failure on one alert does not abandon the rest of the batch: Grafana sends
// them together and retrying the whole batch would re-notify for the alerts
// that already succeeded.
func (s *Service) Ingest(ctx context.Context, alerts []Alert) error {
	var firstErr error
	for _, alert := range alerts {
		if err := s.ingestOne(ctx, alert); err != nil {
			s.logger.ErrorContext(ctx, "cannot apply alert",
				slog.String("fingerprint", alert.Fingerprint),
				slog.Any("err", err))
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (s *Service) ingestOne(ctx context.Context, alert Alert) error {
	if alert.Fingerprint == "" {
		return fmt.Errorf("alert %q has no fingerprint", alert.Title())
	}

	if alert.Resolved() {
		if err := s.repo.ResolveByFingerprint(ctx, alert.Fingerprint, alert.ResolvedAt()); err != nil {
			return err
		}
		s.logger.InfoContext(ctx, "incident resolved",
			slog.String("fingerprint", alert.Fingerprint),
			slog.String("title", alert.Title()))
		return nil
	}

	in, created, err := s.repo.Open(ctx, alert.ToIncident())
	if err != nil {
		return err
	}
	s.logger.WarnContext(ctx, "incident firing",
		slog.String("incident_id", in.ID),
		slog.String("title", in.Title),
		slog.String("severity", string(in.Severity)),
		slog.String("family", string(in.Family)),
		slog.Bool("new", created))

	// Only a genuinely new incident notifies. A flapping alert re-firing every
	// evaluation would otherwise mail every admin every minute, which trains
	// them to ignore the mailbox — the failure mode that makes alerting useless.
	if created {
		s.notifyAdmins(ctx, *in)
	}
	return nil
}

// notifyAdmins mails every admin, in the background. The webhook must answer
// Grafana promptly whatever SMTP is doing, and an incident that is recorded but
// not mailed is far better than one that is neither.
func (s *Service) notifyAdmins(ctx context.Context, in Incident) {
	bgCtx := context.WithoutCancel(ctx)
	go func() {
		ctx, cancel := context.WithTimeout(bgCtx, notifyTimeout)
		defer cancel()

		emails, err := s.repo.AdminEmails(ctx)
		if err != nil {
			s.logger.ErrorContext(ctx, "cannot list admins to notify", slog.Any("err", err))
			return
		}
		if len(emails) == 0 {
			s.logger.WarnContext(ctx, "incident has no admin to notify",
				slog.String("incident_id", in.ID))
			return
		}

		subject, body := notificationText(in)
		for _, to := range emails {
			if err := s.mail.Send(ctx, to, subject, body); err != nil {
				s.logger.ErrorContext(ctx, "cannot notify admin of incident",
					slog.String("incident_id", in.ID), slog.Any("err", err))
			}
		}
	}()
}

func notificationText(in Incident) (subject, body string) {
	subject = fmt.Sprintf("[Lyo %s] %s", strings.ToUpper(string(in.Severity)), in.Title)

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", in.Title)
	if in.Summary != "" {
		fmt.Fprintf(&b, "%s\n\n", in.Summary)
	}
	fmt.Fprintf(&b, "Severity: %s\n", in.Severity)
	// The family is spelled out because the two demand different reactions:
	// a technical incident is a bug to fix, an experience incident is usually
	// a capacity decision.
	switch in.Family {
	case FamilyExperience:
		fmt.Fprintf(&b, "Family:   experience — nothing is broken, users are suffering anyway\n")
	case FamilyTechnical:
		fmt.Fprintf(&b, "Family:   technical — the platform itself is failing\n")
	default:
		fmt.Fprintf(&b, "Family:   %s\n", in.Family)
	}
	fmt.Fprintf(&b, "Started:  %s\n", in.StartedAt.UTC().Format(time.RFC3339))
	if in.DashboardURL != "" {
		fmt.Fprintf(&b, "\nDashboard: %s\n", in.DashboardURL)
	}
	fmt.Fprintf(&b, "\nAcknowledge it in the Lyo admin supervision screen.\n")
	return subject, b.String()
}

// List returns the most recent incidents, optionally filtered by status.
func (s *Service) List(ctx context.Context, status Status, limit int) ([]Incident, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return s.repo.List(ctx, status, limit)
}

func (s *Service) Get(ctx context.Context, id string) (*Incident, error) {
	return s.repo.Get(ctx, id)
}

// Acknowledge records that an admin has taken ownership of a firing incident.
func (s *Service) Acknowledge(ctx context.Context, id, adminID string) (*Incident, error) {
	return s.repo.Acknowledge(ctx, id, adminID, time.Now())
}

// Resolve closes an incident by hand, for the case where the condition cleared
// without Grafana saying so — a rule that was edited, or an alert that fired
// while the collector itself was down.
func (s *Service) Resolve(ctx context.Context, id string) (*Incident, error) {
	return s.repo.Resolve(ctx, id, time.Now())
}

// Counts summarises the feed over the last 24 hours.
func (s *Service) Counts(ctx context.Context) (Counts, error) {
	return s.repo.CountsSince(ctx, time.Now().Add(-24*time.Hour))
}
