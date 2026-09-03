package incident

import (
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// Alert is one alert instance from a Grafana webhook notification. Only the
// fields the incident feed needs are decoded; Grafana sends many more.
type Alert struct {
	Status      string            `json:"status"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    time.Time         `json:"startsAt"`
	EndsAt      time.Time         `json:"endsAt"`
	// Fingerprint identifies the alert instance (rule + label set) and is
	// stable across re-fires, which is what makes deduplication possible.
	Fingerprint  string `json:"fingerprint"`
	DashboardURL string `json:"dashboardURL"`
	PanelURL     string `json:"panelURL"`
	GeneratorURL string `json:"generatorURL"`
}

// webhookPayload is the envelope Grafana POSTs.
type webhookPayload struct {
	Alerts []Alert `json:"alerts"`
}

// Resolved reports whether this notification closes the alert.
func (a Alert) Resolved() bool { return a.Status == "resolved" }

// ResolvedAt is when the condition cleared. Grafana zeroes EndsAt on firing
// alerts and can send a resolved alert with no end time, so fall back to now
// rather than writing a zero timestamp into the feed.
func (a Alert) ResolvedAt() time.Time {
	if a.EndsAt.IsZero() {
		return time.Now()
	}
	return a.EndsAt
}

// Title prefers the rule's own summary annotation and falls back to the rule
// name, so an incident is never titled with an empty string.
func (a Alert) Title() string {
	if s := strings.TrimSpace(a.Annotations["summary"]); s != "" {
		return s
	}
	if n := strings.TrimSpace(a.Labels["alertname"]); n != "" {
		return n
	}
	return "Unnamed alert"
}

// ToIncident maps the alert onto the platform's own model.
func (a Alert) ToIncident() Incident {
	started := a.StartsAt
	if started.IsZero() {
		started = time.Now()
	}
	return Incident{
		Fingerprint:  a.Fingerprint,
		RuleUID:      a.Labels["__alert_rule_uid__"],
		Title:        a.Title(),
		Summary:      strings.TrimSpace(a.Annotations["description"]),
		Severity:     parseSeverity(a.Labels["severity"]),
		Family:       parseFamily(a.Labels["family"]),
		DashboardURL: firstNonEmpty(a.PanelURL, a.DashboardURL, a.GeneratorURL),
		StartedAt:    started,
	}
}

// parseSeverity and parseFamily fall back rather than reject: an alert rule
// that forgets a label must still produce an incident, because a missing
// incident is worse than a mislabelled one.
func parseSeverity(s string) Severity {
	switch Severity(strings.ToLower(s)) {
	case SeverityCritical:
		return SeverityCritical
	case SeverityInfo:
		return SeverityInfo
	case SeverityWarning:
		return SeverityWarning
	default:
		return SeverityWarning
	}
}

func parseFamily(s string) Family {
	if Family(strings.ToLower(s)) == FamilyExperience {
		return FamilyExperience
	}
	return FamilyTechnical
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// WebhookHandler receives Grafana alert notifications.
//
// It sits outside the OpenAPI contract for the same reason the WebSocket
// endpoints do: the request body is Grafana's schema, not Lyo's, and it is
// called by infrastructure rather than by clients. Authentication is a shared
// secret because Grafana's webhook contact point cannot mint a JWT.
type WebhookHandler struct {
	svc    *Service
	secret string
	logger *slog.Logger
}

func NewWebhookHandler(svc *Service, secret string, logger *slog.Logger) *WebhookHandler {
	return &WebhookHandler{svc: svc, secret: secret, logger: logger}
}

// maxWebhookBytes caps the payload. Grafana batches alerts, but a body this
// large is a bug or an attack, not a notification.
const maxWebhookBytes = 1 << 20

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// An unset secret disables the endpoint entirely rather than leaving an
	// unauthenticated incident-injection route open by default.
	if h.secret == "" {
		h.logger.WarnContext(ctx, "alert webhook called but ALERT_WEBHOOK_SECRET is unset")
		http.Error(w, "alert webhook disabled", http.StatusServiceUnavailable)
		return
	}
	if !h.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var payload webhookPayload
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxWebhookBytes)).Decode(&payload); err != nil {
		h.logger.WarnContext(ctx, "malformed alert webhook payload", slog.Any("err", err))
		http.Error(w, "malformed payload", http.StatusBadRequest)
		return
	}

	if err := h.svc.Ingest(ctx, payload.Alerts); err != nil {
		// 500 makes Grafana retry, which is what we want: a lost alert is a
		// blind spot, and Open is idempotent per fingerprint.
		http.Error(w, "cannot record incidents", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// authorized accepts the secret as a Bearer token, which is what Grafana's
// webhook contact point can send as an Authorization header.
func (h *WebhookHandler) authorized(r *http.Request) bool {
	token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	// Constant time: a timing oracle on this comparison would leak the secret.
	return subtle.ConstantTimeCompare([]byte(token), []byte(h.secret)) == 1
}
