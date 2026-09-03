package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/hyppoliteprn/lyo/internal/api"
)

func decodeError(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var body errorResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode %q: %v", w.Body.String(), err)
	}
	return body.Error
}

func TestWriteJSONError(t *testing.T) {
	w := httptest.NewRecorder()

	writeJSONError(w, http.StatusTeapot, errors.New("no coffee"))

	if w.Code != http.StatusTeapot {
		t.Errorf("status = %d, want %d", w.Code, http.StatusTeapot)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content type = %q", ct)
	}
	if got := decodeError(t, w); got != "no coffee" {
		t.Errorf("error = %q", got)
	}
}

// A handler's *api.HTTPError carries the status the client should see.
func TestHandleResponseError_UsesHTTPErrorStatus(t *testing.T) {
	w := httptest.NewRecorder()

	handleResponseError(w, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil),
		&api.HTTPError{Code: http.StatusServiceUnavailable, Msg: "request timeout"})

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
	if got := decodeError(t, w); got != "request timeout" {
		t.Fatalf("error = %q", got)
	}
}

// A wrapped HTTPError must still be unwrapped to its status.
func TestHandleResponseError_UnwrapsWrappedHTTPError(t *testing.T) {
	w := httptest.NewRecorder()
	wrapped := errors.Join(errors.New("context"), &api.HTTPError{Code: http.StatusConflict, Msg: "already live"})

	handleResponseError(w, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil), wrapped)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
}

// Anything else is an unimplemented handler, not a client error.
func TestHandleResponseError_FallsBackToNotImplemented(t *testing.T) {
	w := httptest.NewRecorder()

	handleResponseError(w, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil),
		errors.New("not implemented"))

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", w.Code)
	}
}

func TestHandleRequestError(t *testing.T) {
	w := httptest.NewRecorder()

	handleRequestError(w, httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil),
		errors.New("invalid format for parameter id"))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if got := decodeError(t, w); !strings.Contains(got, "invalid format") {
		t.Fatalf("error = %q", got)
	}
}

// runMigrations must report a clear error rather than panic when it cannot
// reach the database.
func TestRunMigrations_UnreachableDatabase(t *testing.T) {
	db, err := sql.Open("pgx", "postgres://lyo:lyo@127.0.0.1:1/lyo?connect_timeout=1&sslmode=disable")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := runMigrations(db); err == nil {
		t.Fatal("expected an error when the database is unreachable")
	} else if !strings.Contains(err.Error(), "migrate driver") {
		t.Fatalf("unexpected error: %v", err)
	}
}
