package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/hyppoliteprn/lyo/internal/auth"
	"github.com/hyppoliteprn/lyo/internal/incident"
	"github.com/hyppoliteprn/lyo/pkg/middleware"
)

// supervisionFlag gates the whole admin supervision surface.
const supervisionFlag = "admin_supervision"

// Messages a client can act on: one means "log in again", the other means
// "this account will never be allowed in". A single "forbidden" for both is
// what makes an expired session look like a permissions bug.
const (
	sessionExpiredMsg = "session expired, sign in again"
	notAdminMsg       = "administrator access required"
)

// IncidentService is the subset of incident.Service consumed by the handlers.
type IncidentService interface {
	List(ctx context.Context, status incident.Status, limit int) ([]incident.Incident, error)
	Acknowledge(ctx context.Context, id, adminID string) (*incident.Incident, error)
	Resolve(ctx context.Context, id string) (*incident.Incident, error)
	Counts(ctx context.Context) (incident.Counts, error)
}

// MetricsQuerier reads current values out of Prometheus. It is optional: a nil
// implementation means the deployment runs without Prometheus, and the
// supervision endpoint then reports incidents only.
type MetricsQuerier interface {
	ScalarQuery(ctx context.Context, query string) (float64, bool, error)
}

// adminGate is the single gate for every supervision endpoint. Incidents
// expose the shape of the platform's failures, so nothing here is readable
// below the admin role.
type adminGate int

const (
	// adminOK — the caller is an authenticated admin.
	adminOK adminGate = iota
	// adminExpired — a token was presented and rejected. Almost always an
	// expired access token, which is a session to renew, not a permission the
	// caller lacks. Answering 403 here tells an admin they are not an admin.
	adminExpired
	// adminForbidden — anonymous, or authenticated below the admin role.
	adminForbidden
)

func checkAdmin(ctx context.Context) adminGate {
	claims, ok := middleware.ClaimsFromContext(ctx)
	if ok && claims.Role.AtLeast(auth.RoleAdmin) {
		return adminOK
	}
	if !ok && middleware.TokenRejected(ctx) {
		return adminExpired
	}
	return adminForbidden
}

func incidentToAPI(in *incident.Incident) (Incident, error) {
	id, err := uuid.Parse(in.ID)
	if err != nil {
		return Incident{}, fmt.Errorf("invalid incident ID %q: %w", in.ID, err)
	}
	out := Incident{
		Id:        openapi_types.UUID(id),
		Title:     in.Title,
		Severity:  IncidentSeverity(in.Severity),
		Family:    IncidentFamily(in.Family),
		Status:    IncidentStatus(in.Status),
		StartedAt: in.StartedAt,
	}
	if in.Summary != "" {
		out.Summary = &in.Summary
	}
	if in.RuleUID != "" {
		out.RuleUid = &in.RuleUID
	}
	if in.DashboardURL != "" {
		out.DashboardUrl = &in.DashboardURL
	}
	out.AcknowledgedAt = in.AcknowledgedAt
	out.ResolvedAt = in.ResolvedAt
	if in.AcknowledgedBy != nil {
		if by, err := uuid.Parse(*in.AcknowledgedBy); err == nil {
			ackBy := openapi_types.UUID(by)
			out.AcknowledgedBy = &ackBy
		}
	}
	return out, nil
}

func (h *Handlers) ListIncidents(ctx context.Context, req ListIncidentsRequestObject) (ListIncidentsResponseObject, error) {
	switch checkAdmin(ctx) {
	case adminExpired:
		return ListIncidents401JSONResponse{
			UnauthorizedJSONResponse: UnauthorizedJSONResponse{Code: 401, Message: sessionExpiredMsg},
		}, nil
	case adminForbidden:
		return ListIncidents403JSONResponse{
			ForbiddenJSONResponse: ForbiddenJSONResponse{Code: 403, Message: notAdminMsg},
		}, nil
	case adminOK:
	}
	if !h.featureSvc.IsEnabled(ctx, supervisionFlag) {
		return ListIncidents503JSONResponse{
			ServiceUnavailableJSONResponse: ServiceUnavailableJSONResponse{Code: 503, Message: "feature disabled"},
		}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var status incident.Status
	if req.Params.Status != nil {
		status = incident.Status(*req.Params.Status)
	}
	limit := 50
	if req.Params.Limit != nil {
		limit = *req.Params.Limit
	}

	found, err := h.incidentSvc.List(ctx, status, limit)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "list incidents timeout", "route", "GET /admin/incidents")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	items := make([]Incident, 0, len(found))
	for i := range found {
		item, err := incidentToAPI(&found[i])
		if err != nil {
			return nil, fmt.Errorf("marshal incident: %w", err)
		}
		items = append(items, item)
	}
	return ListIncidents200JSONResponse{Items: items}, nil
}

func (h *Handlers) UpdateIncident(ctx context.Context, req UpdateIncidentRequestObject) (UpdateIncidentResponseObject, error) {
	switch checkAdmin(ctx) {
	case adminExpired:
		return UpdateIncident401JSONResponse{
			UnauthorizedJSONResponse: UnauthorizedJSONResponse{Code: 401, Message: sessionExpiredMsg},
		}, nil
	case adminForbidden:
		return UpdateIncident403JSONResponse{
			ForbiddenJSONResponse: ForbiddenJSONResponse{Code: 403, Message: notAdminMsg},
		}, nil
	case adminOK:
	}
	// Safe after adminOK: the gate only returns it when claims are present.
	claims, _ := middleware.ClaimsFromContext(ctx)
	if !h.featureSvc.IsEnabled(ctx, supervisionFlag) {
		return UpdateIncident503JSONResponse{
			ServiceUnavailableJSONResponse: ServiceUnavailableJSONResponse{Code: 503, Message: "feature disabled"},
		}, nil
	}
	if req.Body == nil {
		return UpdateIncident400JSONResponse{
			BadRequestJSONResponse: BadRequestJSONResponse{Code: 400, Message: "missing body"},
		}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var (
		updated *incident.Incident
		err     error
	)
	switch req.Body.Action {
	case Acknowledge:
		updated, err = h.incidentSvc.Acknowledge(ctx, req.Id.String(), claims.UserID)
	case Resolve:
		updated, err = h.incidentSvc.Resolve(ctx, req.Id.String())
	default:
		return UpdateIncident400JSONResponse{
			BadRequestJSONResponse: BadRequestJSONResponse{Code: 400, Message: "unknown action"},
		}, nil
	}

	if err != nil {
		// ErrNotFound also covers "the incident exists but is not in a state
		// this action applies to" — acknowledging something already resolved,
		// for instance — because the update is expressed as a conditional
		// UPDATE rather than a read-then-write race.
		if errors.Is(err, incident.ErrNotFound) {
			return UpdateIncident404JSONResponse{
				NotFoundJSONResponse: NotFoundJSONResponse{Code: 404, Message: "no incident to update"},
			}, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "update incident timeout", "route", "PATCH /admin/incidents/{id}")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	out, err := incidentToAPI(updated)
	if err != nil {
		return nil, fmt.Errorf("marshal incident: %w", err)
	}
	return UpdateIncident200JSONResponse(out), nil
}

// supervisionQueries are the readings the admin screen shows. They are the
// same expressions as the corresponding Grafana panels, so the two never
// disagree about what "the error rate" means.
var supervisionQueries = []struct {
	name   string
	query  string
	assign func(*SupervisionMetrics, float64)
}{
	{
		name:   "backend_up",
		query:  `up{job="lyo-backend"}`,
		assign: func(m *SupervisionMetrics, v float64) { up := v >= 1; m.BackendUp = &up },
	},
	{
		name:   "live_streams",
		query:  `sum(lyo_streams_live{job="lyo-backend"})`,
		assign: func(m *SupervisionMetrics, v float64) { m.LiveStreams = float32Ptr(v) },
	},
	{
		name:   "active_listeners",
		query:  `sum(lyo_listeners_active{job="lyo-backend"})`,
		assign: func(m *SupervisionMetrics, v float64) { m.ActiveListeners = float32Ptr(v) },
	},
	{
		name:   "requests_per_second",
		query:  `sum(rate(http_server_request_duration_seconds_count{job="lyo-backend"}[5m]))`,
		assign: func(m *SupervisionMetrics, v float64) { m.RequestsPerSecond = float32Ptr(v) },
	},
	{
		name: "error_rate_percent",
		query: `100 * sum(rate(http_server_request_duration_seconds_count{job="lyo-backend",http_response_status_code=~"5.."}[5m]))` +
			` / clamp_min(sum(rate(http_server_request_duration_seconds_count{job="lyo-backend"}[5m])), 0.001)`,
		assign: func(m *SupervisionMetrics, v float64) { m.ErrorRatePercent = float32Ptr(v) },
	},
	{
		name:   "latency_p95_seconds",
		query:  `histogram_quantile(0.95, sum by (le) (rate(http_server_request_duration_seconds_bucket{job="lyo-backend"}[5m])))`,
		assign: func(m *SupervisionMetrics, v float64) { m.LatencyP95Seconds = float32Ptr(v) },
	},
	{
		name:   "chunks_dropped_per_second",
		query:  `sum(rate(lyo_chunks_dropped_total{job="lyo-backend"}[5m]))`,
		assign: func(m *SupervisionMetrics, v float64) { m.ChunksDroppedPerSecond = float32Ptr(v) },
	},
}

func float32Ptr(v float64) *float32 {
	f := float32(v)
	return &f
}

func (h *Handlers) GetSupervisionSummary(ctx context.Context, _ GetSupervisionSummaryRequestObject) (GetSupervisionSummaryResponseObject, error) {
	switch checkAdmin(ctx) {
	case adminExpired:
		return GetSupervisionSummary401JSONResponse{
			UnauthorizedJSONResponse: UnauthorizedJSONResponse{Code: 401, Message: sessionExpiredMsg},
		}, nil
	case adminForbidden:
		return GetSupervisionSummary403JSONResponse{
			ForbiddenJSONResponse: ForbiddenJSONResponse{Code: 403, Message: notAdminMsg},
		}, nil
	case adminOK:
	}
	if !h.featureSvc.IsEnabled(ctx, supervisionFlag) {
		return GetSupervisionSummary503JSONResponse{
			ServiceUnavailableJSONResponse: ServiceUnavailableJSONResponse{Code: 503, Message: "feature disabled"},
		}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	counts, err := h.incidentSvc.Counts(ctx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			h.logger.WarnContext(ctx, "supervision summary timeout", "route", "GET /admin/supervision")
			return nil, &HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"}
		}
		return nil, err
	}

	resp := SupervisionSummary{
		GeneratedAt: time.Now(),
		Incidents: IncidentCounts{
			Firing:         counts.Firing,
			Acknowledged:   counts.Acknowledged,
			CriticalFiring: counts.CriticalFiring,
			Resolved24h:    counts.Resolved24h,
		},
	}

	// Prometheus being unreachable degrades this endpoint, it does not fail
	// it: the incident counts are the part an admin most needs, and they come
	// from our own database.
	if h.metricsSvc != nil {
		metrics := SupervisionMetrics{}
		available := false
		for _, q := range supervisionQueries {
			value, found, err := h.metricsSvc.ScalarQuery(ctx, q.query)
			if err != nil {
				h.logger.WarnContext(ctx, "supervision metric unavailable",
					"metric", q.name, "err", err)
				continue
			}
			available = true
			if found {
				q.assign(&metrics, value)
			}
		}
		if available {
			resp.Metrics = &metrics
			resp.MetricsAvailable = true
		}
	}

	return GetSupervisionSummary200JSONResponse(resp), nil
}
