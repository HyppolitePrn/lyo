package streaming

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"

	"github.com/hyppoliteprn/lyo/internal/auth"
	"github.com/hyppoliteprn/lyo/pkg/middleware"
)

func testAuthSvc() *auth.Service {
	return auth.NewService("test-jwt-secret-at-least-32-chars!", time.Minute, time.Hour)
}

func mustToken(t *testing.T, svc *auth.Service, userID string, role auth.Role) string {
	t.Helper()
	pair, err := svc.Issue(userID, role)
	if err != nil {
		t.Fatal(err)
	}
	return pair.AccessToken
}

// wsFixture wires a chi router with both WebSocket endpoints, mirroring main.go.
type wsFixture struct {
	srv     *httptest.Server
	svc     *Service
	authSvc *auth.Service
	repo    *fakeStreamRepo
	// metric reads back the current value of one instrument by name.
	metric func(name string) int64
}

func newWSFixture(t *testing.T, repo *fakeStreamRepo) *wsFixture {
	t.Helper()
	authSvc := testAuthSvc()
	metrics, sum := collectingMetrics(t)
	svc := NewService(repo, 8, testLogger(), metrics)

	r := chi.NewRouter()
	r.Use(middleware.Authenticate(authSvc))
	r.Get("/streams/{id}/ingest", NewIngestHandler(svc, authSvc, testLogger(), metrics).ServeHTTP)
	r.Get("/streams/{id}/listen", NewListenHandler(svc, authSvc, testLogger(), metrics).ServeHTTP)

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	return &wsFixture{srv: srv, svc: svc, authSvc: authSvc, repo: repo, metric: sum}
}

func (f *wsFixture) wsURL(path string) string {
	return "ws" + strings.TrimPrefix(f.srv.URL, "http") + path
}

// dial opens a WebSocket, returning the HTTP status when the upgrade is refused.
func (f *wsFixture) dial(t *testing.T, ctx context.Context, path, subprotocol, bearer string) (*websocket.Conn, int) {
	t.Helper()
	opts := &websocket.DialOptions{Subprotocols: []string{subprotocol}}
	if bearer != "" {
		opts.HTTPHeader = http.Header{"Authorization": []string{"Bearer " + bearer}}
	}
	conn, resp, err := websocket.Dial(ctx, f.wsURL(path), opts)
	if err != nil {
		status := 0
		if resp != nil {
			status = resp.StatusCode
			_ = resp.Body.Close()
		}
		return nil, status
	}
	return conn, http.StatusSwitchingProtocols
}

// liveStream registers a live stream plus its hub, as StartStream would.
func (f *wsFixture) liveStream(t *testing.T, id, broadcasterID string) {
	t.Helper()
	f.repo.got = &Stream{ID: id, BroadcasterID: broadcasterID, Status: "live", StartedAt: time.Now()}
	f.repo.created = f.repo.got
	if _, err := f.svc.StartStream(context.Background(), broadcasterID, "t", ""); err != nil {
		t.Fatal(err)
	}
}

// ── Ingest ────────────────────────────────────────────────────────────────────

func TestIngest_RejectsAnonymous(t *testing.T) {
	f := newWSFixture(t, &fakeStreamRepo{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, status := f.dial(t, ctx, "/streams/s1/ingest", "audio-ingest", ""); status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", status)
	}
}

func TestIngest_RejectsInvalidQueryToken(t *testing.T) {
	f := newWSFixture(t, &fakeStreamRepo{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, status := f.dial(t, ctx, "/streams/s1/ingest?token=not.a.jwt", "audio-ingest", ""); status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", status)
	}
}

func TestIngest_RejectsPlainUser(t *testing.T) {
	f := newWSFixture(t, &fakeStreamRepo{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	token := mustToken(t, f.authSvc, "user-1", auth.RoleUser)
	if _, status := f.dial(t, ctx, "/streams/s1/ingest", "audio-ingest", token); status != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", status)
	}
}

func TestIngest_UnknownStream(t *testing.T) {
	f := newWSFixture(t, &fakeStreamRepo{getErr: ErrNotFound})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	token := mustToken(t, f.authSvc, "bc-1", auth.RoleBroadcaster)
	if _, status := f.dial(t, ctx, "/streams/s1/ingest", "audio-ingest", token); status != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", status)
	}
}

func TestIngest_RepoErrorIs500(t *testing.T) {
	f := newWSFixture(t, &fakeStreamRepo{getErr: context.DeadlineExceeded})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	token := mustToken(t, f.authSvc, "bc-1", auth.RoleBroadcaster)
	if _, status := f.dial(t, ctx, "/streams/s1/ingest", "audio-ingest", token); status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", status)
	}
}

// Only the stream's own broadcaster may ingest into it.
func TestIngest_RejectsForeignBroadcaster(t *testing.T) {
	f := newWSFixture(t, &fakeStreamRepo{})
	f.liveStream(t, "s1", "bc-1")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	token := mustToken(t, f.authSvc, "bc-2", auth.RoleBroadcaster)
	if _, status := f.dial(t, ctx, "/streams/s1/ingest", "audio-ingest", token); status != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", status)
	}
}

func TestIngest_RejectsEndedStream(t *testing.T) {
	repo := &fakeStreamRepo{got: &Stream{ID: "s1", BroadcasterID: "bc-1", Status: "ended"}}
	f := newWSFixture(t, repo)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	token := mustToken(t, f.authSvc, "bc-1", auth.RoleBroadcaster)
	if _, status := f.dial(t, ctx, "/streams/s1/ingest", "audio-ingest", token); status != http.StatusGone {
		t.Fatalf("status = %d, want 410", status)
	}
}

// The DB row can outlive its hub if EndStream raced the connection.
func TestIngest_MissingHubIs503(t *testing.T) {
	repo := &fakeStreamRepo{got: &Stream{ID: "s1", BroadcasterID: "bc-1", Status: "live"}}
	f := newWSFixture(t, repo)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	token := mustToken(t, f.authSvc, "bc-1", auth.RoleBroadcaster)
	if _, status := f.dial(t, ctx, "/streams/s1/ingest", "audio-ingest", token); status != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", status)
	}
}

// Clients that cannot set headers (browsers, the Flutter WS client) authenticate
// through ?token= instead.
func TestIngest_AcceptsQueryParamToken(t *testing.T) {
	f := newWSFixture(t, &fakeStreamRepo{})
	f.liveStream(t, "s1", "bc-1")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	token := mustToken(t, f.authSvc, "bc-1", auth.RoleBroadcaster)
	conn, status := f.dial(t, ctx, "/streams/s1/ingest?token="+token, "audio-ingest", "")
	if status != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d, want 101", status)
	}
	_ = conn.CloseNow()
}

// ── Ingest → Hub → Listen, end to end ─────────────────────────────────────────

func TestIngestToListen_ForwardsBinaryChunksAndIgnoresText(t *testing.T) {
	f := newWSFixture(t, &fakeStreamRepo{})
	f.liveStream(t, "s1", "bc-1")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	listener, status := f.dial(t, ctx, "/streams/s1/listen", "audio-stream", "")
	if status != http.StatusSwitchingProtocols {
		t.Fatalf("listen dial status = %d, want 101", status)
	}
	defer func() { _ = listener.CloseNow() }()

	token := mustToken(t, f.authSvc, "bc-1", auth.RoleBroadcaster)
	broadcaster, status := f.dial(t, ctx, "/streams/s1/ingest", "audio-ingest", token)
	if status != http.StatusSwitchingProtocols {
		t.Fatalf("ingest dial status = %d, want 101", status)
	}
	defer func() { _ = broadcaster.CloseNow() }()

	// Wait for the listener to be registered on the hub before broadcasting.
	hub := f.svc.Hub("s1")
	waitFor(t, func() bool { return hub.ListenerCount() == 1 })

	// Text frames are control chatter and must not reach listeners.
	if err := broadcaster.Write(ctx, websocket.MessageText, []byte("ignore me")); err != nil {
		t.Fatalf("write text: %v", err)
	}
	if err := broadcaster.Write(ctx, websocket.MessageBinary, []byte("frame-1")); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	typ, data, err := listener.Read(ctx)
	if err != nil {
		t.Fatalf("listener read: %v", err)
	}
	if typ != websocket.MessageBinary {
		t.Errorf("message type = %v, want binary", typ)
	}
	if string(data) != "frame-1" {
		t.Errorf("payload = %q, want %q", data, "frame-1")
	}
}

// EndStream closes the hub, which must tear down both live connections.
func TestEndStream_DisconnectsListener(t *testing.T) {
	f := newWSFixture(t, &fakeStreamRepo{})
	f.liveStream(t, "s1", "bc-1")
	f.repo.ended = &Stream{ID: "s1", BroadcasterID: "bc-1", Status: "ended"}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	listener, status := f.dial(t, ctx, "/streams/s1/listen", "audio-stream", "")
	if status != http.StatusSwitchingProtocols {
		t.Fatalf("dial status = %d, want 101", status)
	}
	defer func() { _ = listener.CloseNow() }()

	hub := f.svc.Hub("s1")
	waitFor(t, func() bool { return hub.ListenerCount() == 1 })

	if _, err := f.svc.EndStream(context.Background(), "s1", "bc-1"); err != nil {
		t.Fatalf("end stream: %v", err)
	}

	if _, _, err := listener.Read(ctx); err == nil {
		t.Fatal("expected the listener connection to be closed when the stream ends")
	}
}

// ── Listen ────────────────────────────────────────────────────────────────────

// Listening is public: no token at all is fine.
func TestListen_AllowsAnonymous(t *testing.T) {
	f := newWSFixture(t, &fakeStreamRepo{})
	f.liveStream(t, "s1", "bc-1")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, status := f.dial(t, ctx, "/streams/s1/listen", "audio-stream", "")
	if status != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d, want 101", status)
	}
	_ = conn.CloseNow()
}

// A token that is present but invalid is a client error, not an anonymous listen.
func TestListen_RejectsInvalidQueryToken(t *testing.T) {
	f := newWSFixture(t, &fakeStreamRepo{})
	f.liveStream(t, "s1", "bc-1")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, status := f.dial(t, ctx, "/streams/s1/listen?token=not.a.jwt", "audio-stream", ""); status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", status)
	}
}

func TestListen_AcceptsQueryParamToken(t *testing.T) {
	f := newWSFixture(t, &fakeStreamRepo{})
	f.liveStream(t, "s1", "bc-1")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	token := mustToken(t, f.authSvc, "user-1", auth.RoleUser)
	conn, status := f.dial(t, ctx, "/streams/s1/listen?token="+token, "audio-stream", "")
	if status != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d, want 101", status)
	}
	_ = conn.CloseNow()
}

func TestListen_UnknownStream(t *testing.T) {
	f := newWSFixture(t, &fakeStreamRepo{getErr: ErrNotFound})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, status := f.dial(t, ctx, "/streams/s1/listen", "audio-stream", ""); status != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", status)
	}
}

func TestListen_RepoErrorIs500(t *testing.T) {
	f := newWSFixture(t, &fakeStreamRepo{getErr: context.DeadlineExceeded})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, status := f.dial(t, ctx, "/streams/s1/listen", "audio-stream", ""); status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", status)
	}
}

func TestListen_RejectsEndedStream(t *testing.T) {
	repo := &fakeStreamRepo{got: &Stream{ID: "s1", BroadcasterID: "bc-1", Status: "ended"}}
	f := newWSFixture(t, repo)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, status := f.dial(t, ctx, "/streams/s1/listen", "audio-stream", ""); status != http.StatusGone {
		t.Fatalf("status = %d, want 410", status)
	}
}

func TestListen_MissingHubIs503(t *testing.T) {
	repo := &fakeStreamRepo{got: &Stream{ID: "s1", BroadcasterID: "bc-1", Status: "live"}}
	f := newWSFixture(t, repo)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, status := f.dial(t, ctx, "/streams/s1/listen", "audio-stream", ""); status != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", status)
	}
}

// Disconnecting must release the listener slot on the hub. The handler only
// notices the drop when a write fails, so a chunk is broadcast to provoke it.
func TestListen_UnsubscribesOnDisconnect(t *testing.T) {
	f := newWSFixture(t, &fakeStreamRepo{})
	f.liveStream(t, "s1", "bc-1")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, status := f.dial(t, ctx, "/streams/s1/listen", "audio-stream", "")
	if status != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d, want 101", status)
	}

	hub := f.svc.Hub("s1")
	waitFor(t, func() bool { return hub.ListenerCount() == 1 })

	_ = conn.CloseNow()

	waitFor(t, func() bool {
		hub.Broadcast(ctx, Chunk("x"))
		return hub.ListenerCount() == 0
	})
}

// waitFor polls cond until it holds or the test times out.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition was not met within 5s")
}

// waitForMetric polls until an instrument reaches want, running tick between
// attempts. Disconnects are recorded by the handler goroutine after the client
// has already gone, so the value cannot be read synchronously.
func (f *wsFixture) waitForMetric(t *testing.T, name string, want int64, tick func()) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	var got int64
	for time.Now().Before(deadline) {
		if got = f.metric(name); got == want {
			return
		}
		if tick != nil {
			tick()
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("%s = %d, want %d", name, got, want)
}

// TestListen_ListenerGaugeReturnsToZero is the leak check: a listener count
// that only ever goes up would make every capacity reading on the supervision
// dashboard wrong, and the drift is invisible without asserting it.
func TestListen_ListenerGaugeReturnsToZero(t *testing.T) {
	f := newWSFixture(t, &fakeStreamRepo{})
	f.liveStream(t, "s1", "bc-1")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, status := f.dial(t, ctx, "/streams/s1/listen", "audio-stream", "")
	if status != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d, want 101", status)
	}
	f.waitForMetric(t, "lyo.listeners.active", 1, nil)

	// CloseNow, not Close: the listen loop only writes, so it never reads the
	// close frame a graceful handshake would wait for.
	_ = conn.CloseNow()

	// The loop is parked on its channel and only learns the client is gone
	// when a write fails, so keep the stream flowing until it notices.
	broadcast := func() { f.svc.Hub("s1").Broadcast(ctx, Chunk("audio")) }
	f.waitForMetric(t, "lyo.listeners.active", 0, broadcast)
	f.waitForMetric(t, "lyo.listener.disconnect", 1, nil)
}
