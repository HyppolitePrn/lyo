package observability_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hyppoliteprn/lyo/internal/observability"
)

func promServer(t *testing.T, status int, body string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestPrometheusClient_ReadsTheFirstSample(t *testing.T) {
	url := promServer(t, http.StatusOK,
		`{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1788436012.172,"42.5"]}]}}`)

	v, found, err := observability.NewPrometheusClient(url).ScalarQuery(t.Context(), `up`)
	if err != nil {
		t.Fatalf("ScalarQuery: %v", err)
	}
	if !found || v != 42.5 {
		t.Errorf("value = %v (found=%v), want 42.5", v, found)
	}
}

// TestPrometheusClient_EmptyResultIsNotAnError: a platform with no traffic has
// no series, and reporting that as a failure would make the supervision screen
// cry wolf on a quiet Sunday.
func TestPrometheusClient_EmptyResultIsNotAnError(t *testing.T) {
	url := promServer(t, http.StatusOK, `{"status":"success","data":{"resultType":"vector","result":[]}}`)

	v, found, err := observability.NewPrometheusClient(url).ScalarQuery(t.Context(), `up`)
	if err != nil {
		t.Fatalf("ScalarQuery: %v", err)
	}
	if found || v != 0 {
		t.Errorf("got (%v, %v), want (0, false)", v, found)
	}
}

func TestPrometheusClient_ErrorsAreReported(t *testing.T) {
	t.Run("http error", func(t *testing.T) {
		url := promServer(t, http.StatusInternalServerError, `{}`)
		if _, _, err := observability.NewPrometheusClient(url).ScalarQuery(t.Context(), `up`); err == nil {
			t.Error("expected an error for a 500 response")
		}
	})

	t.Run("query error", func(t *testing.T) {
		url := promServer(t, http.StatusOK, `{"status":"error","error":"parse error"}`)
		if _, _, err := observability.NewPrometheusClient(url).ScalarQuery(t.Context(), `up{`); err == nil {
			t.Error("expected an error when Prometheus rejects the query")
		}
	})

	t.Run("unparsable value", func(t *testing.T) {
		url := promServer(t, http.StatusOK,
			`{"status":"success","data":{"result":[{"value":[1,"not-a-number"]}]}}`)
		if _, _, err := observability.NewPrometheusClient(url).ScalarQuery(t.Context(), `up`); err == nil {
			t.Error("expected an error for an unparsable sample value")
		}
	})
}

// TestNewPrometheusClient_EmptyURLIsNil is what lets a deployment run without
// Prometheus: callers treat nil as "no live metrics", not as an error.
func TestNewPrometheusClient_EmptyURLIsNil(t *testing.T) {
	if c := observability.NewPrometheusClient(""); c != nil {
		t.Error("expected a nil client for an empty URL")
	}
}

// TestPrometheusClient_NonFiniteValuesAreAbsent: histogram_quantile over an
// empty histogram returns NaN and a ratio over zero traffic can return +Inf.
// Neither is a number the API can encode, and reporting them as measured
// readings would break the whole supervision response.
func TestPrometheusClient_NonFiniteValuesAreAbsent(t *testing.T) {
	for _, raw := range []string{"NaN", "+Inf", "-Inf"} {
		t.Run(raw, func(t *testing.T) {
			url := promServer(t, http.StatusOK,
				`{"status":"success","data":{"result":[{"value":[1,"`+raw+`"]}]}}`)

			v, found, err := observability.NewPrometheusClient(url).ScalarQuery(t.Context(), `q`)
			if err != nil {
				t.Fatalf("ScalarQuery: %v", err)
			}
			if found || v != 0 {
				t.Errorf("got (%v, %v), want (0, false)", v, found)
			}
		})
	}
}
