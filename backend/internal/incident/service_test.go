package incident_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hyppoliteprn/lyo/internal/incident"
)

// errTest stands in for any persistence failure.
var errTest = errors.New("db down")

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// fakeRepo records what the service asked of persistence.
type fakeRepo struct {
	mu sync.Mutex

	opened      []incident.Incident
	openResult  incident.Incident
	openCreated bool
	openErr     error

	resolvedFingerprints []string
	resolveErr           error

	admins    []string
	adminsErr error
}

func (f *fakeRepo) Open(_ context.Context, in incident.Incident) (*incident.Incident, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.opened = append(f.opened, in)
	if f.openErr != nil {
		return nil, false, f.openErr
	}
	out := f.openResult
	if out.ID == "" {
		out = in
		out.ID = "incident-1"
		out.Status = incident.StatusFiring
	}
	return &out, f.openCreated, nil
}

func (f *fakeRepo) ResolveByFingerprint(_ context.Context, fingerprint string, _ time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.resolvedFingerprints = append(f.resolvedFingerprints, fingerprint)
	return f.resolveErr
}

func (f *fakeRepo) AdminEmails(_ context.Context) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.admins, f.adminsErr
}

func (f *fakeRepo) List(context.Context, incident.Status, int) ([]incident.Incident, error) {
	return nil, nil
}
func (f *fakeRepo) Get(context.Context, string) (*incident.Incident, error) { return nil, nil }
func (f *fakeRepo) Acknowledge(context.Context, string, string, time.Time) (*incident.Incident, error) {
	return nil, nil
}
func (f *fakeRepo) Resolve(context.Context, string, time.Time) (*incident.Incident, error) {
	return nil, nil
}
func (f *fakeRepo) CountsSince(context.Context, time.Time) (incident.Counts, error) {
	return incident.Counts{}, nil
}

// fakeMailer captures notifications. Sends happen on a background goroutine,
// so waitFor lets a test block until they land.
type fakeMailer struct {
	mu   sync.Mutex
	sent []struct{ To, Subject, Body string }
	err  error
}

func (m *fakeMailer) Send(_ context.Context, to, subject, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, struct{ To, Subject, Body string }{to, subject, body})
	return m.err
}

func (m *fakeMailer) waitFor(t *testing.T, want int) []struct{ To, Subject, Body string } {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		m.mu.Lock()
		got := len(m.sent)
		snapshot := append([]struct{ To, Subject, Body string }(nil), m.sent...)
		m.mu.Unlock()
		if got >= want {
			return snapshot
		}
		time.Sleep(5 * time.Millisecond)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	t.Fatalf("sent %d notifications, want %d", len(m.sent), want)
	return nil
}

func firingAlert(fingerprint string) incident.Alert {
	return incident.Alert{
		Status:      "firing",
		Fingerprint: fingerprint,
		Labels: map[string]string{
			"alertname":          "Listeners are losing audio",
			"severity":           "critical",
			"family":             "experience",
			"__alert_rule_uid__": "lyo-chunks-dropped",
		},
		Annotations: map[string]string{
			"summary":     "The hub is dropping audio chunks",
			"description": "Back-pressure is engaging",
		},
		StartsAt:     time.Now().Add(-time.Minute),
		DashboardURL: "http://grafana/d/lyo-streaming",
	}
}

func TestIngest_NewIncidentNotifiesEveryAdmin(t *testing.T) {
	repo := &fakeRepo{openCreated: true, admins: []string{"a@lyo.test", "b@lyo.test"}}
	mail := &fakeMailer{}
	svc := incident.NewService(repo, mail, testLogger())

	if err := svc.Ingest(t.Context(), []incident.Alert{firingAlert("fp-1")}); err != nil {
		t.Fatalf("Ingest: %v", err)
	}

	sent := mail.waitFor(t, 2)
	if sent[0].To != "a@lyo.test" || sent[1].To != "b@lyo.test" {
		t.Errorf("recipients = %q, %q; want both admins", sent[0].To, sent[1].To)
	}
	if !strings.Contains(sent[0].Subject, "CRITICAL") {
		t.Errorf("subject %q should carry the severity", sent[0].Subject)
	}
	// The family is the actionable part: an experience incident is a capacity
	// decision, not a bug hunt.
	if !strings.Contains(sent[0].Body, "experience") {
		t.Errorf("body should name the family, got %q", sent[0].Body)
	}
}

// TestIngest_RefiringDoesNotRenotify is the difference between alerting that
// gets read and a mailbox admins learn to ignore.
func TestIngest_RefiringDoesNotRenotify(t *testing.T) {
	repo := &fakeRepo{openCreated: false, admins: []string{"a@lyo.test"}}
	mail := &fakeMailer{}
	svc := incident.NewService(repo, mail, testLogger())

	if err := svc.Ingest(t.Context(), []incident.Alert{firingAlert("fp-1")}); err != nil {
		t.Fatalf("Ingest: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	mail.mu.Lock()
	defer mail.mu.Unlock()
	if len(mail.sent) != 0 {
		t.Errorf("sent %d notifications for an already-open incident, want 0", len(mail.sent))
	}
}

func TestIngest_ResolvedAlertClosesTheIncident(t *testing.T) {
	repo := &fakeRepo{}
	svc := incident.NewService(repo, &fakeMailer{}, testLogger())

	alert := firingAlert("fp-1")
	alert.Status = "resolved"
	alert.EndsAt = time.Now()

	if err := svc.Ingest(t.Context(), []incident.Alert{alert}); err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if len(repo.resolvedFingerprints) != 1 || repo.resolvedFingerprints[0] != "fp-1" {
		t.Errorf("resolved = %v, want [fp-1]", repo.resolvedFingerprints)
	}
	if len(repo.opened) != 0 {
		t.Errorf("a resolved alert must not open an incident, opened %d", len(repo.opened))
	}
}

// TestIngest_OneBadAlertDoesNotAbandonTheBatch: Grafana batches alerts and
// retries the whole request, so giving up on the first failure would both lose
// the healthy alerts and re-notify for them on the retry.
func TestIngest_OneBadAlertDoesNotAbandonTheBatch(t *testing.T) {
	repo := &fakeRepo{openCreated: true}
	svc := incident.NewService(repo, &fakeMailer{}, testLogger())

	noFingerprint := firingAlert("")
	err := svc.Ingest(t.Context(), []incident.Alert{noFingerprint, firingAlert("fp-2")})
	if err == nil {
		t.Fatal("expected the batch to report the failed alert")
	}
	if len(repo.opened) != 1 || repo.opened[0].Fingerprint != "fp-2" {
		t.Errorf("opened = %+v, want only fp-2", repo.opened)
	}
}

func TestIngest_RepositoryFailurePropagates(t *testing.T) {
	sentinel := errTest
	repo := &fakeRepo{openErr: sentinel}
	svc := incident.NewService(repo, &fakeMailer{}, testLogger())

	if err := svc.Ingest(t.Context(), []incident.Alert{firingAlert("fp-1")}); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
}

func TestAlert_MappingFallsBackRatherThanRejecting(t *testing.T) {
	// A rule that forgets its labels must still produce an incident: a missing
	// incident is worse than a mislabelled one.
	bare := incident.Alert{Status: "firing", Fingerprint: "fp-x"}
	got := bare.ToIncident()

	if got.Severity != incident.SeverityWarning {
		t.Errorf("severity = %q, want the warning fallback", got.Severity)
	}
	if got.Family != incident.FamilyTechnical {
		t.Errorf("family = %q, want the technical fallback", got.Family)
	}
	if got.Title == "" {
		t.Error("title must never be empty")
	}
	if got.StartedAt.IsZero() {
		t.Error("started_at must never be the zero time")
	}
}

func TestAlert_TitlePrefersSummaryOverRuleName(t *testing.T) {
	a := incident.Alert{
		Labels:      map[string]string{"alertname": "lyo-chunks-dropped"},
		Annotations: map[string]string{"summary": "The hub is dropping audio chunks"},
	}
	if got := a.Title(); got != "The hub is dropping audio chunks" {
		t.Errorf("Title() = %q, want the summary annotation", got)
	}

	a.Annotations = nil
	if got := a.Title(); got != "lyo-chunks-dropped" {
		t.Errorf("Title() = %q, want the rule name fallback", got)
	}
}
