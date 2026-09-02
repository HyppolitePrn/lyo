package storage_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/hyppoliteprn/lyo/internal/storage"
	"github.com/hyppoliteprn/lyo/pkg/config"
)

func newStorage(t *testing.T, cfg config.S3Config) storage.Storage {
	t.Helper()
	s, err := storage.New(cfg)
	if err != nil {
		t.Fatalf("new storage: %v", err)
	}
	return s
}

func minioCfg(endpoint, publicEndpoint string) config.S3Config {
	return config.S3Config{
		BucketName:      "lyo-tracks",
		Region:          "us-east-1",
		AccessKeyID:     "key",
		SecretAccessKey: "secret",
		Endpoint:        endpoint,
		PublicEndpoint:  publicEndpoint,
	}
}

// With no endpoint configured, URLs use real AWS virtual-hosted-style addressing.
func TestPublicURL_AWSFallback(t *testing.T) {
	s := newStorage(t, config.S3Config{BucketName: "lyo-tracks", Region: "eu-west-3"})

	got := s.PublicURL("tracks/bc-1/abc.mp3")
	want := "https://lyo-tracks.s3.eu-west-3.amazonaws.com/tracks/bc-1/abc.mp3"
	if got != want {
		t.Fatalf("PublicURL = %q, want %q", got, want)
	}
}

// A self-hosted backend uses path-style addressing under the public endpoint.
func TestPublicURL_SelfHostedEndpoint(t *testing.T) {
	s := newStorage(t, minioCfg("http://minio:9000", "http://192.168.1.10:9000"))

	got := s.PublicURL("tracks/bc-1/abc.mp3")
	want := "http://192.168.1.10:9000/lyo-tracks/tracks/bc-1/abc.mp3"
	if got != want {
		t.Fatalf("PublicURL = %q, want %q", got, want)
	}
}

// A trailing slash on the public endpoint must not produce a doubled separator.
func TestPublicURL_TrimsTrailingSlash(t *testing.T) {
	s := newStorage(t, minioCfg("http://minio:9000", "http://minio.example.com/"))

	if got := s.PublicURL("k"); got != "http://minio.example.com/lyo-tracks/k" {
		t.Fatalf("PublicURL = %q", got)
	}
}

// An empty PublicEndpoint falls back to Endpoint.
func TestPublicURL_FallsBackToInternalEndpoint(t *testing.T) {
	s := newStorage(t, minioCfg("http://minio:9000", ""))

	if got := s.PublicURL("k"); got != "http://minio:9000/lyo-tracks/k" {
		t.Fatalf("PublicURL = %q", got)
	}
}

func TestKeyFromURL_RoundTripsPublicURL(t *testing.T) {
	for _, cfg := range []config.S3Config{
		{BucketName: "lyo-tracks", Region: "eu-west-3"},
		minioCfg("http://minio:9000", "http://192.168.1.10:9000"),
	} {
		s := newStorage(t, cfg)
		const key = "tracks/bc-1/abc.mp3"

		got, ok := s.KeyFromURL(s.PublicURL(key))
		if !ok {
			t.Fatalf("endpoint %q: expected the key to be recognised", cfg.Endpoint)
		}
		if got != key {
			t.Fatalf("endpoint %q: key = %q, want %q", cfg.Endpoint, got, key)
		}
	}
}

// A URL that isn't ours (e.g. a track whose audio_url points elsewhere) must be
// reported as foreign so the caller skips the S3 delete.
func TestKeyFromURL_RejectsForeignURL(t *testing.T) {
	s := newStorage(t, minioCfg("http://minio:9000", "http://192.168.1.10:9000"))

	for _, u := range []string{
		"https://cdn.example.com/tracks/abc.mp3",
		"http://192.168.1.10:9000/other-bucket/tracks/abc.mp3",
		"",
	} {
		if key, ok := s.KeyFromURL(u); ok {
			t.Errorf("KeyFromURL(%q) = %q, true; want false", u, key)
		}
	}
}

func TestPresignUpload_SignsAgainstThePublicEndpoint(t *testing.T) {
	s := newStorage(t, minioCfg("http://minio:9000", "http://192.168.1.10:9000"))

	raw, err := s.PresignUpload(context.Background(), "tracks/bc-1/abc.mp3")
	if err != nil {
		t.Fatalf("presign: %v", err)
	}

	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse presigned url: %v", err)
	}
	if u.Host != "192.168.1.10:9000" {
		t.Errorf("host = %q, want the public endpoint (signature is bound to it)", u.Host)
	}
	if u.Path != "/lyo-tracks/tracks/bc-1/abc.mp3" {
		t.Errorf("path = %q, want path-style addressing", u.Path)
	}
	q := u.Query()
	for _, param := range []string{"X-Amz-Signature", "X-Amz-Expires", "X-Amz-Credential"} {
		if q.Get(param) == "" {
			t.Errorf("presigned URL is missing %s: %s", param, raw)
		}
	}
	if q.Get("X-Amz-Expires") != "900" {
		t.Errorf("X-Amz-Expires = %q, want 900 (15 minutes)", q.Get("X-Amz-Expires"))
	}
}

func TestDelete(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	s := newStorage(t, minioCfg(srv.URL, srv.URL))

	if err := s.Delete(context.Background(), "tracks/bc-1/abc.mp3"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if gotPath != "/lyo-tracks/tracks/bc-1/abc.mp3" {
		t.Errorf("path = %q", gotPath)
	}
}

func TestDelete_PropagatesBackendError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`<Error><Code>AccessDenied</Code></Error>`))
	}))
	defer srv.Close()

	s := newStorage(t, minioCfg(srv.URL, srv.URL))

	err := s.Delete(context.Background(), "tracks/bc-1/abc.mp3")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "AccessDenied") && !strings.Contains(err.Error(), "403") {
		t.Errorf("unexpected error: %v", err)
	}
}
