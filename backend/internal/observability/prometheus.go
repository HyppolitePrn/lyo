package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// promQueryTimeout bounds one instant query. The supervision endpoint runs
// several of them and must still answer inside the handler's own budget.
const promQueryTimeout = 3 * time.Second

// PrometheusClient runs instant PromQL queries.
//
// The mobile admin screen needs a handful of current values, not a query
// language, so this is deliberately a reader of single numbers rather than a
// general Prometheus client. Grafana remains the place to actually explore the
// data; this exists so an admin sees the state of the platform without one.
type PrometheusClient struct {
	baseURL string
	http    *http.Client
}

// NewPrometheusClient returns nil when baseURL is empty, which is the
// supported way to run without Prometheus: callers treat a nil client as
// "no live metrics available" rather than as an error.
func NewPrometheusClient(baseURL string) *PrometheusClient {
	if baseURL == "" {
		return nil
	}
	return &PrometheusClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: promQueryTimeout},
	}
}

// promResponse is the subset of Prometheus' instant-query envelope we read.
type promResponse struct {
	Status string `json:"status"`
	Data   struct {
		Result []struct {
			// [ <unix seconds float>, "<value as string>" ]
			Value []json.RawMessage `json:"value"`
		} `json:"result"`
	} `json:"data"`
	Error string `json:"error"`
}

// ScalarQuery runs an instant query and returns the first sample's value.
// An empty result is (0, false) rather than an error: "no series yet" is the
// normal state of a platform with no traffic, not a failure.
func (c *PrometheusClient) ScalarQuery(ctx context.Context, query string) (float64, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, promQueryTimeout)
	defer cancel()

	endpoint := c.baseURL + "/api/v1/query?query=" + url.QueryEscape(query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, false, fmt.Errorf("prometheus request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, false, fmt.Errorf("prometheus query: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return 0, false, fmt.Errorf("prometheus query: status %d", resp.StatusCode)
	}

	var body promResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, false, fmt.Errorf("prometheus decode: %w", err)
	}
	if body.Status != "success" {
		return 0, false, fmt.Errorf("prometheus query failed: %s", body.Error)
	}
	if len(body.Data.Result) == 0 || len(body.Data.Result[0].Value) < 2 {
		return 0, false, nil
	}

	// Prometheus encodes sample values as strings so that NaN and ±Inf survive
	// JSON; both decode to a float here and are reported as absent.
	var raw string
	if err := json.Unmarshal(body.Data.Result[0].Value[1], &raw); err != nil {
		return 0, false, fmt.Errorf("prometheus value: %w", err)
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false, fmt.Errorf("prometheus value %q: %w", raw, err)
	}
	// histogram_quantile over an empty histogram yields NaN, and a ratio over
	// zero traffic can yield ±Inf. Both are "nothing measured", and neither
	// can be encoded as JSON — reporting them as absent is both truthful and
	// the only thing that keeps the response serialisable.
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false, nil
	}
	return v, true, nil
}
