package incident_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hyppoliteprn/lyo/internal/incident"
)

const testSecret = "webhook-secret"

func newWebhook(t *testing.T, repo *fakeRepo, secret string) http.Handler {
	t.Helper()
	svc := incident.NewService(repo, &fakeMailer{}, testLogger())
	return incident.NewWebhookHandler(svc, secret, testLogger())
}

func post(t *testing.T, h http.Handler, body, bearer string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/internal/alerts", strings.NewReader(body))
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

const grafanaPayload = `{
  "status": "firing",
  "alerts": [{
    "status": "firing",
    "labels": {"alertname": "Backend unreachable", "severity": "critical", "family": "technical",
               "__alert_rule_uid__": "lyo-backend-down"},
    "annotations": {"summary": "The Lyo backend is not exposing metrics",
                    "description": "Prometheus cannot scrape the backend"},
    "startsAt": "2026-09-03T10:00:00Z",
    "fingerprint": "abc123",
    "dashboardURL": "http://grafana/d/lyo-performance"
  }]
}`

func TestWebhook_AcceptsGrafanaPayload(t *testing.T) {
	repo := &fakeRepo{openCreated: true}
	rec := post(t, newWebhook(t, repo, testSecret), grafanaPayload, testSecret)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204: %s", rec.Code, rec.Body.String())
	}
	if len(repo.opened) != 1 {
		t.Fatalf("opened %d incidents, want 1", len(repo.opened))
	}
	got := repo.opened[0]
	if got.Fingerprint != "abc123" {
		t.Errorf("fingerprint = %q", got.Fingerprint)
	}
	if got.RuleUID != "lyo-backend-down" {
		t.Errorf("rule_uid = %q, want the rule UID label", got.RuleUID)
	}
	if got.Severity != incident.SeverityCritical || got.Family != incident.FamilyTechnical {
		t.Errorf("severity/family = %q/%q", got.Severity, got.Family)
	}
	if got.Title != "The Lyo backend is not exposing metrics" {
		t.Errorf("title = %q", got.Title)
	}
}

func TestWebhook_RejectsWrongOrMissingSecret(t *testing.T) {
	for name, bearer := range map[string]string{
		"no token":    "",
		"wrong token": "not-the-secret",
		// A prefix of the real secret must not pass: the comparison is over
		// the whole value, in constant time.
		"prefix": testSecret[:5],
	} {
		t.Run(name, func(t *testing.T) {
			repo := &fakeRepo{}
			rec := post(t, newWebhook(t, repo, testSecret), grafanaPayload, bearer)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", rec.Code)
			}
			if len(repo.opened) != 0 {
				t.Error("an unauthorized call must not touch the incident feed")
			}
		})
	}
}

// TestWebhook_UnsetSecretDisablesTheEndpoint: the alternative — treating an
// empty secret as "accept anything" — would leave an unauthenticated
// incident-injection route open on every deployment that forgot to set it.
func TestWebhook_UnsetSecretDisablesTheEndpoint(t *testing.T) {
	repo := &fakeRepo{}
	rec := post(t, newWebhook(t, repo, ""), grafanaPayload, "")

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	if len(repo.opened) != 0 {
		t.Error("the disabled endpoint must not record anything")
	}
}

func TestWebhook_RejectsMalformedBody(t *testing.T) {
	rec := post(t, newWebhook(t, &fakeRepo{}, testSecret), "{not json", testSecret)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// TestWebhook_FailureAsksGrafanaToRetry: answering 2xx on a failed write would
// make Grafana drop the notification, and the incident would never exist.
func TestWebhook_FailureAsksGrafanaToRetry(t *testing.T) {
	repo := &fakeRepo{openErr: errTest}
	rec := post(t, newWebhook(t, repo, testSecret), grafanaPayload, testSecret)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}
