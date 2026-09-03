package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/hyppoliteprn/lyo/internal/auth"
	"github.com/hyppoliteprn/lyo/internal/incident"
)

// ── Doubles ───────────────────────────────────────────────────────────────────

type fakeIncidentSvc struct {
	list      []incident.Incident
	listErr   error
	counts    incident.Counts
	countsErr error

	updated  *incident.Incident
	updenErr error

	acknowledgedBy string
	resolvedID     string
	listedStatus   incident.Status
	listedLimit    int
}

func (f *fakeIncidentSvc) List(_ context.Context, status incident.Status, limit int) ([]incident.Incident, error) {
	f.listedStatus, f.listedLimit = status, limit
	return f.list, f.listErr
}

func (f *fakeIncidentSvc) Acknowledge(_ context.Context, id, adminID string) (*incident.Incident, error) {
	f.acknowledgedBy = adminID
	return f.updated, f.updenErr
}

func (f *fakeIncidentSvc) Resolve(_ context.Context, id string) (*incident.Incident, error) {
	f.resolvedID = id
	return f.updated, f.updenErr
}

func (f *fakeIncidentSvc) Counts(context.Context) (incident.Counts, error) {
	return f.counts, f.countsErr
}

type fakeMetrics struct {
	values map[string]float64
	err    error
	calls  int
}

func (f *fakeMetrics) ScalarQuery(_ context.Context, query string) (float64, bool, error) {
	f.calls++
	if f.err != nil {
		return 0, false, f.err
	}
	v, ok := f.values[query]
	return v, ok, nil
}

const sampleIncidentID = "11111111-1111-1111-1111-111111111111"

func sampleIncident() incident.Incident {
	return incident.Incident{
		ID:          sampleIncidentID,
		Fingerprint: "fp-1",
		RuleUID:     "lyo-chunks-dropped",
		Title:       "The hub is dropping audio chunks",
		Severity:    incident.SeverityCritical,
		Family:      incident.FamilyExperience,
		Status:      incident.StatusFiring,
		StartedAt:   time.Now().Add(-time.Hour),
	}
}

// ── Access control ────────────────────────────────────────────────────────────

// TestSupervision_ClosedBelowAdmin: incidents describe how the platform fails,
// which is not something a listener or a broadcaster gets to read.
func TestSupervision_ClosedBelowAdmin(t *testing.T) {
	routes := []struct {
		method, path, body string
	}{
		{http.MethodGet, "/admin/supervision", ""},
		{http.MethodGet, "/admin/incidents", ""},
		{http.MethodPatch, "/admin/incidents/" + sampleIncidentID, `{"action":"acknowledge"}`},
	}
	roles := map[string]auth.Role{
		"anonymous":   "",
		"user":        auth.RoleUser,
		"broadcaster": auth.RoleBroadcaster,
	}

	for name, role := range roles {
		for _, route := range routes {
			t.Run(name+" "+route.method+" "+route.path, func(t *testing.T) {
				f := newFixture(t)
				token := ""
				if role != "" {
					token = f.token(t, "user-1", role)
				}
				rec := f.do(t, route.method, route.path, token, route.body)
				if rec.Code != http.StatusForbidden {
					t.Fatalf("status = %d, want 403", rec.Code)
				}
			})
		}
	}
}

func TestSupervision_DisabledFlagReturns503(t *testing.T) {
	f := newFixture(t, "admin_supervision")
	rec := f.do(t, http.MethodGet, "/admin/incidents", f.token(t, "admin-1", auth.RoleAdmin), "")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

// ── Incident feed ─────────────────────────────────────────────────────────────

func TestListIncidents_ReturnsTheFeed(t *testing.T) {
	f := newFixture(t)
	f.incidents.list = []incident.Incident{sampleIncident()}

	rec := f.do(t, http.MethodGet, "/admin/incidents?status=firing&limit=10",
		f.token(t, "admin-1", auth.RoleAdmin), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Items []struct {
			Id       string `json:"id"`
			Title    string `json:"title"`
			Severity string `json:"severity"`
			Family   string `json:"family"`
			Status   string `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(body.Items))
	}
	if body.Items[0].Family != "experience" {
		t.Errorf("family = %q, want it preserved through the API", body.Items[0].Family)
	}
	if f.incidents.listedStatus != incident.StatusFiring || f.incidents.listedLimit != 10 {
		t.Errorf("filters not forwarded: status=%q limit=%d",
			f.incidents.listedStatus, f.incidents.listedLimit)
	}
}

func TestUpdateIncident_AcknowledgeRecordsTheAdmin(t *testing.T) {
	f := newFixture(t)
	ack := sampleIncident()
	ack.Status = incident.StatusAcknowledged
	f.incidents.updated = &ack

	rec := f.do(t, http.MethodPatch, "/admin/incidents/"+sampleIncidentID,
		f.token(t, "admin-7", auth.RoleAdmin), `{"action":"acknowledge"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	// Who acknowledged is the whole point of acknowledgement.
	if f.incidents.acknowledgedBy != "admin-7" {
		t.Errorf("acknowledged by %q, want the caller's user ID", f.incidents.acknowledgedBy)
	}
}

func TestUpdateIncident_ResolveAndUnknownAction(t *testing.T) {
	f := newFixture(t)
	resolved := sampleIncident()
	resolved.Status = incident.StatusResolved
	f.incidents.updated = &resolved
	token := f.token(t, "admin-1", auth.RoleAdmin)

	if rec := f.do(t, http.MethodPatch, "/admin/incidents/"+sampleIncidentID,
		token, `{"action":"resolve"}`); rec.Code != http.StatusOK {
		t.Fatalf("resolve status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if f.incidents.resolvedID != sampleIncidentID {
		t.Errorf("resolved %q, want %q", f.incidents.resolvedID, sampleIncidentID)
	}

	// The enum is enforced by the generated validation layer, so an unknown
	// action must never reach the service.
	if rec := f.do(t, http.MethodPatch, "/admin/incidents/"+sampleIncidentID,
		token, `{"action":"delete"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("unknown action status = %d, want 400", rec.Code)
	}
}

func TestUpdateIncident_NotFound(t *testing.T) {
	f := newFixture(t)
	f.incidents.updenErr = incident.ErrNotFound

	rec := f.do(t, http.MethodPatch, "/admin/incidents/"+sampleIncidentID,
		f.token(t, "admin-1", auth.RoleAdmin), `{"action":"acknowledge"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

// ── Summary ───────────────────────────────────────────────────────────────────

func TestSupervisionSummary_CombinesIncidentsAndMetrics(t *testing.T) {
	f := newFixture(t)
	f.incidents.counts = incident.Counts{Firing: 2, Acknowledged: 1, CriticalFiring: 1, Resolved24h: 5}
	f.metrics.values = map[string]float64{
		`up{job="lyo-backend"}`:                        1,
		`sum(lyo_streams_live{job="lyo-backend"})`:     3,
		`sum(lyo_listeners_active{job="lyo-backend"})`: 42,
	}

	rec := f.do(t, http.MethodGet, "/admin/supervision", f.token(t, "admin-1", auth.RoleAdmin), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Incidents struct {
			Firing         int `json:"firing"`
			CriticalFiring int `json:"critical_firing"`
		} `json:"incidents"`
		MetricsAvailable bool `json:"metrics_available"`
		Metrics          struct {
			BackendUp       *bool    `json:"backend_up"`
			LiveStreams     *float64 `json:"live_streams"`
			ActiveListeners *float64 `json:"active_listeners"`
			// Never queried in this test's map, so it must be absent rather
			// than reported as a confident zero.
			RequestsPerSecond *float64 `json:"requests_per_second"`
		} `json:"metrics"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if body.Incidents.Firing != 2 || body.Incidents.CriticalFiring != 1 {
		t.Errorf("incident counts = %+v", body.Incidents)
	}
	if !body.MetricsAvailable {
		t.Error("metrics_available should be true when Prometheus answered")
	}
	if body.Metrics.BackendUp == nil || !*body.Metrics.BackendUp {
		t.Error("backend_up should be true for up == 1")
	}
	if body.Metrics.LiveStreams == nil || *body.Metrics.LiveStreams != 3 {
		t.Errorf("live_streams = %v, want 3", body.Metrics.LiveStreams)
	}
	if body.Metrics.RequestsPerSecond != nil {
		t.Errorf("a metric with no series must be absent, got %v", *body.Metrics.RequestsPerSecond)
	}
}

// TestSupervisionSummary_SurvivesPrometheusBeingDown: the incident counts come
// from our own database and are the part an admin most needs. Losing
// Prometheus must degrade the response, not fail it.
func TestSupervisionSummary_SurvivesPrometheusBeingDown(t *testing.T) {
	f := newFixture(t)
	f.incidents.counts = incident.Counts{Firing: 1}
	f.metrics.err = errors.New("connection refused")

	rec := f.do(t, http.MethodGet, "/admin/supervision", f.token(t, "admin-1", auth.RoleAdmin), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Incidents        struct{ Firing int } `json:"incidents"`
		MetricsAvailable bool                 `json:"metrics_available"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Incidents.Firing != 1 {
		t.Errorf("firing = %d, want the incident counts to survive", body.Incidents.Firing)
	}
	if body.MetricsAvailable {
		t.Error("metrics_available must be false when every query failed")
	}
}
