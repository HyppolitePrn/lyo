package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hyppoliteprn/lyo/internal/api"
	"github.com/hyppoliteprn/lyo/internal/auth"
	"github.com/hyppoliteprn/lyo/internal/features"
	"github.com/hyppoliteprn/lyo/internal/playlist"
	"github.com/hyppoliteprn/lyo/internal/streaming"
	"github.com/hyppoliteprn/lyo/internal/track"
	"github.com/hyppoliteprn/lyo/internal/user"
	"github.com/hyppoliteprn/lyo/pkg/middleware"
)

// ── Configurable service doubles ──────────────────────────────────────────────

// fakeUserSvc records favorite mutations and serves a fixed user.
type fakeUserSvc struct {
	user   *user.User
	getErr error
	err    error // returned by every favorite mutation

	added   []string
	removed []string
}

func (f *fakeUserSvc) Register(context.Context, string, string, string) (auth.TokenPair, error) {
	return auth.TokenPair{}, errors.New("not used")
}
func (f *fakeUserSvc) Login(context.Context, string, string) (auth.TokenPair, error) {
	return auth.TokenPair{}, errors.New("not used")
}
func (f *fakeUserSvc) GetByID(context.Context, string) (*user.User, error) {
	return f.user, f.getErr
}
func (f *fakeUserSvc) AddFavoriteTrack(_ context.Context, _, id string) error {
	f.added = append(f.added, "track:"+id)
	return f.err
}
func (f *fakeUserSvc) RemoveFavoriteTrack(_ context.Context, _, id string) error {
	f.removed = append(f.removed, "track:"+id)
	return f.err
}
func (f *fakeUserSvc) AddFavoriteStream(_ context.Context, _, id string) error {
	f.added = append(f.added, "stream:"+id)
	return f.err
}
func (f *fakeUserSvc) RemoveFavoriteStream(_ context.Context, _, id string) error {
	f.removed = append(f.removed, "stream:"+id)
	return f.err
}
func (f *fakeUserSvc) AddFavoritePlaylist(_ context.Context, _, id string) error {
	f.added = append(f.added, "playlist:"+id)
	return f.err
}
func (f *fakeUserSvc) RemoveFavoritePlaylist(_ context.Context, _, id string) error {
	f.removed = append(f.removed, "playlist:"+id)
	return f.err
}

type fakeTrackSvc struct {
	tracks     map[string]*track.Track
	list       []track.Track
	listErr    error
	createErr  error
	deleteErr  error
	getErr     error
	presign    [3]string
	presignErr error

	created     *track.Track
	deletedWith [2]string
	listedWith  struct {
		broadcaster string
		page, limit int
	}
}

func (f *fakeTrackSvc) ListTracks(_ context.Context, broadcasterID string, page, limit int) ([]track.Track, error) {
	f.listedWith.broadcaster, f.listedWith.page, f.listedWith.limit = broadcasterID, page, limit
	return f.list, f.listErr
}
func (f *fakeTrackSvc) CreateTrack(_ context.Context, broadcasterID, title, artist, audioURL string, duration int) (*track.Track, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	f.created = &track.Track{
		ID: uuid.NewString(), BroadcasterID: broadcasterID, Title: title,
		Artist: artist, AudioURL: audioURL, DurationSeconds: duration, CreatedAt: time.Now(),
	}
	return f.created, nil
}
func (f *fakeTrackSvc) GetTrack(_ context.Context, id string) (*track.Track, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	t, ok := f.tracks[id]
	if !ok {
		return nil, track.ErrNotFound
	}
	return t, nil
}
func (f *fakeTrackSvc) DeleteTrack(_ context.Context, id, requesterID string) error {
	f.deletedWith = [2]string{id, requesterID}
	return f.deleteErr
}
func (f *fakeTrackSvc) PresignUpload(context.Context, string, string) (string, string, string, error) {
	return f.presign[0], f.presign[1], f.presign[2], f.presignErr
}

type fakeStreamSvc struct {
	streams   map[string]*streaming.Stream
	list      []streaming.Stream
	listErr   error
	startErr  error
	endErr    error
	getErr    error
	started   *streaming.Stream
	endedWith [2]string
}

func (f *fakeStreamSvc) StartStream(_ context.Context, broadcasterID, title, description string) (*streaming.Stream, error) {
	if f.startErr != nil {
		return nil, f.startErr
	}
	f.started = &streaming.Stream{
		ID: uuid.NewString(), BroadcasterID: broadcasterID, Title: title,
		Description: description, Status: "live", StartedAt: time.Now(),
	}
	return f.started, nil
}
func (f *fakeStreamSvc) EndStream(_ context.Context, id, broadcasterID string) (*streaming.Stream, error) {
	f.endedWith = [2]string{id, broadcasterID}
	if f.endErr != nil {
		return nil, f.endErr
	}
	return &streaming.Stream{ID: id, BroadcasterID: broadcasterID, Status: "ended", StartedAt: time.Now()}, nil
}
func (f *fakeStreamSvc) GetStream(_ context.Context, id string) (*streaming.Stream, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	s, ok := f.streams[id]
	if !ok {
		return nil, streaming.ErrNotFound
	}
	return s, nil
}
func (f *fakeStreamSvc) ListLiveStreams(context.Context) ([]streaming.Stream, error) {
	return f.list, f.listErr
}

type fakePlaylistSvc struct {
	playlists map[string]*playlist.Playlist
	list      []playlist.Playlist
	listErr   error
	createErr error
	updateErr error
	deleteErr error
	mutateErr error
	getErr    error

	created     *playlist.Playlist
	updatedWith [2]string
	deletedWith [2]string
	addedWith   [3]string
	removedWith [3]string
}

func (f *fakePlaylistSvc) ListByOwner(context.Context, string) ([]playlist.Playlist, error) {
	return f.list, f.listErr
}
func (f *fakePlaylistSvc) Create(_ context.Context, ownerID, title, description string, isPublic bool) (*playlist.Playlist, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	now := time.Now()
	f.created = &playlist.Playlist{
		ID: uuid.NewString(), OwnerID: ownerID, Title: title, Description: description,
		TrackIDs: []string{}, IsPublic: isPublic, CreatedAt: now, UpdatedAt: now,
	}
	return f.created, nil
}
func (f *fakePlaylistSvc) Get(_ context.Context, id string) (*playlist.Playlist, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	p, ok := f.playlists[id]
	if !ok {
		return nil, playlist.ErrNotFound
	}
	return p, nil
}
func (f *fakePlaylistSvc) Update(_ context.Context, id, requesterID string, title, _ *string, _ *bool) (*playlist.Playlist, error) {
	f.updatedWith = [2]string{id, requesterID}
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	p := *f.playlists[id]
	if title != nil {
		p.Title = *title
	}
	return &p, nil
}
func (f *fakePlaylistSvc) Delete(_ context.Context, id, requesterID string) error {
	f.deletedWith = [2]string{id, requesterID}
	return f.deleteErr
}
func (f *fakePlaylistSvc) AddTrack(_ context.Context, id, requesterID, trackID string) (*playlist.Playlist, error) {
	f.addedWith = [3]string{id, requesterID, trackID}
	if f.mutateErr != nil {
		return nil, f.mutateErr
	}
	p := *f.playlists[id]
	p.TrackIDs = append(append([]string{}, p.TrackIDs...), trackID)
	return &p, nil
}
func (f *fakePlaylistSvc) RemoveTrack(_ context.Context, id, requesterID, trackID string) (*playlist.Playlist, error) {
	f.removedWith = [3]string{id, requesterID, trackID}
	if f.mutateErr != nil {
		return nil, f.mutateErr
	}
	p := *f.playlists[id]
	p.TrackIDs = []string{}
	return &p, nil
}

// flagSvc reports every flag as enabled except those listed in off.
type flagSvc struct{ off map[string]bool }

func (f flagSvc) IsEnabled(_ context.Context, name string) bool { return !f.off[name] }

func (f flagSvc) All(_ context.Context) ([]features.Flag, error) {
	return nil, errors.New("not implemented")
}

func (f flagSvc) Toggle(_ context.Context, _ string, _ bool) (*features.Flag, error) {
	return nil, errors.New("not implemented")
}

// fakeDeleteUserUC records the account it was asked to erase.
type fakeDeleteUserUC struct {
	err       error
	deletedID string
}

func (f *fakeDeleteUserUC) Execute(_ context.Context, id string) error {
	f.deletedID = id
	return f.err
}

// ── Fixture ───────────────────────────────────────────────────────────────────

type fixture struct {
	handler   http.Handler
	authSvc   *auth.Service
	users     *fakeUserSvc
	tracks    *fakeTrackSvc
	streams   *fakeStreamSvc
	playlist  *fakePlaylistSvc
	flags     flagSvc
	incidents *fakeIncidentSvc
	metrics   *fakeMetrics
	delUser   *fakeDeleteUserUC
}

func newFixture(t *testing.T, disabledFlags ...string) *fixture {
	t.Helper()
	off := make(map[string]bool, len(disabledFlags))
	for _, f := range disabledFlags {
		off[f] = true
	}

	f := &fixture{
		authSvc:   newTestAuthSvc(),
		users:     &fakeUserSvc{},
		tracks:    &fakeTrackSvc{tracks: map[string]*track.Track{}},
		streams:   &fakeStreamSvc{streams: map[string]*streaming.Stream{}},
		playlist:  &fakePlaylistSvc{playlists: map[string]*playlist.Playlist{}},
		flags:     flagSvc{off: off},
		incidents: &fakeIncidentSvc{},
		metrics:   &fakeMetrics{},
		delUser:   &fakeDeleteUserUC{},
	}

	r := chi.NewRouter()
	r.Use(middleware.Authenticate(f.authSvc))
	strict := api.NewStrictHandlerWithOptions(
		api.NewHandlers(f.users, f.authSvc, f.streams, f.flags, nil, f.tracks, f.playlist, nil, nil, f.delUser,
			f.incidents, f.metrics, slog.New(slog.NewTextHandler(io.Discard, nil))),
		nil,
		api.StrictHTTPServerOptions{
			ResponseErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
				var he *api.HTTPError
				if errors.As(err, &he) {
					http.Error(w, he.Msg, he.Code)
					return
				}
				http.Error(w, "internal error", http.StatusInternalServerError)
			},
			RequestErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
				http.Error(w, err.Error(), http.StatusBadRequest)
			},
		},
	)
	api.HandlerFromMux(strict, r)
	f.handler = r
	return f
}

// token issues an access token for the given role. An empty role means anonymous.
func (f *fixture) token(t *testing.T, userID string, role auth.Role) string {
	t.Helper()
	if role == "" {
		return ""
	}
	pair, err := f.authSvc.Issue(userID, role)
	if err != nil {
		t.Fatal(err)
	}
	return pair.AccessToken
}

func (f *fixture) do(t *testing.T, method, path, bearer, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequestWithContext(context.Background(), method, path, r)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, req)
	return w
}

func decodeInto(t *testing.T, w *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.NewDecoder(w.Body).Decode(v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func newTrack(broadcasterID string) *track.Track {
	return &track.Track{
		ID: uuid.NewString(), BroadcasterID: broadcasterID, Title: "Nightcall",
		Artist: "Kavinsky", AudioURL: "https://cdn/x.mp3", DurationSeconds: 267, CreatedAt: time.Now(),
	}
}

func newStream(broadcasterID string) *streaming.Stream {
	return &streaming.Stream{
		ID: uuid.NewString(), BroadcasterID: broadcasterID, Title: "My show",
		Description: "desc", Status: "live", StartedAt: time.Now(),
	}
}

func newPlaylist(ownerID string, isPublic bool, trackIDs ...string) *playlist.Playlist {
	now := time.Now()
	if trackIDs == nil {
		trackIDs = []string{}
	}
	return &playlist.Playlist{
		ID: uuid.NewString(), OwnerID: ownerID, Title: "Road trip", Description: "for the car",
		TrackIDs: trackIDs, IsPublic: isPublic, CreatedAt: now, UpdatedAt: now,
	}
}

// ── Health ────────────────────────────────────────────────────────────────────

func TestGetHealth(t *testing.T) {
	f := newFixture(t)

	w := f.do(t, http.MethodGet, "/health", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}
	var body struct {
		Status string `json:"status"`
	}
	decodeInto(t, w, &body)
	if body.Status != "ok" {
		t.Fatalf("status = %q", body.Status)
	}
}

// ── Tracks ────────────────────────────────────────────────────────────────────

func TestListTracks_DefaultPagination(t *testing.T) {
	f := newFixture(t)
	bcID := uuid.NewString()
	f.tracks.list = []track.Track{*newTrack(bcID), *newTrack(bcID)}

	w := f.do(t, http.MethodGet, "/tracks", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}
	if f.tracks.listedWith.page != 1 || f.tracks.listedWith.limit != 20 {
		t.Errorf("pagination = %+v, want page 1 limit 20", f.tracks.listedWith)
	}

	var body struct {
		Items []api.Track `json:"items"`
	}
	decodeInto(t, w, &body)
	if len(body.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(body.Items))
	}
}

func TestListTracks_PassesQueryParams(t *testing.T) {
	f := newFixture(t)
	bcID := uuid.NewString()

	w := f.do(t, http.MethodGet, "/tracks?broadcaster_id="+bcID+"&page=3&limit=5", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}
	if f.tracks.listedWith.broadcaster != bcID || f.tracks.listedWith.page != 3 || f.tracks.listedWith.limit != 5 {
		t.Fatalf("filters = %+v", f.tracks.listedWith)
	}
}

// A service timeout must surface as 503, never 500.
func TestListTracks_TimeoutIs503(t *testing.T) {
	f := newFixture(t)
	f.tracks.listErr = context.DeadlineExceeded

	if w := f.do(t, http.MethodGet, "/tracks", "", ""); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503: %s", w.Code, w.Body)
	}
}

func TestListTracks_ServiceErrorIs500(t *testing.T) {
	f := newFixture(t)
	f.tracks.listErr = errors.New("db down")

	if w := f.do(t, http.MethodGet, "/tracks", "", ""); w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500: %s", w.Code, w.Body)
	}
}

// The API contract types IDs as UUIDs; a non-UUID from the service can't be marshalled.
func TestListTracks_NonUUIDFromServiceIs500(t *testing.T) {
	f := newFixture(t)
	f.tracks.list = []track.Track{{ID: "not-a-uuid", BroadcasterID: uuid.NewString()}}

	if w := f.do(t, http.MethodGet, "/tracks", "", ""); w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500: %s", w.Code, w.Body)
	}
}

func TestGetTrack(t *testing.T) {
	f := newFixture(t)
	tr := newTrack(uuid.NewString())
	f.tracks.tracks[tr.ID] = tr

	w := f.do(t, http.MethodGet, "/tracks/"+tr.ID, "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}
	var got api.Track
	decodeInto(t, w, &got)
	if got.Title != "Nightcall" || got.Artist == nil || *got.Artist != "Kavinsky" {
		t.Fatalf("unexpected track: %+v", got)
	}
}

func TestGetTrack_NotFound(t *testing.T) {
	f := newFixture(t)

	if w := f.do(t, http.MethodGet, "/tracks/"+uuid.NewString(), "", ""); w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404: %s", w.Code, w.Body)
	}
}

// The generated binding rejects a path parameter that isn't a UUID.
func TestGetTrack_MalformedIDIs400(t *testing.T) {
	f := newFixture(t)

	if w := f.do(t, http.MethodGet, "/tracks/not-a-uuid", "", ""); w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", w.Code, w.Body)
	}
}

func TestCreateTrack_RequiresBroadcaster(t *testing.T) {
	for _, role := range []auth.Role{"", auth.RoleUser} {
		f := newFixture(t)
		w := f.do(t, http.MethodPost, "/tracks", f.token(t, "u-1", role),
			`{"title":"Nightcall","audio_url":"https://cdn/x.mp3"}`)

		if w.Code != http.StatusForbidden {
			t.Fatalf("role %q: status = %d, want 403: %s", role, w.Code, w.Body)
		}
	}
}

func TestCreateTrack_Success(t *testing.T) {
	f := newFixture(t)
	bcID := uuid.NewString()

	w := f.do(t, http.MethodPost, "/tracks", f.token(t, bcID, auth.RoleBroadcaster),
		`{"title":"Nightcall","artist":"Kavinsky","audio_url":"https://cdn/x.mp3","duration_seconds":267}`)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", w.Code, w.Body)
	}
	if f.tracks.created.BroadcasterID != bcID {
		t.Errorf("broadcaster = %q, want the caller's own id", f.tracks.created.BroadcasterID)
	}
	if f.tracks.created.DurationSeconds != 267 {
		t.Errorf("duration = %d", f.tracks.created.DurationSeconds)
	}
}

// artist and duration_seconds are optional and default to zero values.
func TestCreateTrack_OptionalFieldsOmitted(t *testing.T) {
	f := newFixture(t)

	w := f.do(t, http.MethodPost, "/tracks", f.token(t, uuid.NewString(), auth.RoleBroadcaster),
		`{"title":"Untitled","audio_url":"https://cdn/x.mp3"}`)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", w.Code, w.Body)
	}
	if f.tracks.created.Artist != "" || f.tracks.created.DurationSeconds != 0 {
		t.Fatalf("unexpected defaults: %+v", f.tracks.created)
	}
	var got api.Track
	decodeInto(t, w, &got)
	if got.Artist != nil {
		t.Errorf("artist should be omitted when empty, got %q", *got.Artist)
	}
}

func TestCreateTrack_TimeoutIs503(t *testing.T) {
	f := newFixture(t)
	f.tracks.createErr = context.DeadlineExceeded

	w := f.do(t, http.MethodPost, "/tracks", f.token(t, uuid.NewString(), auth.RoleBroadcaster),
		`{"title":"t","audio_url":"u"}`)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503: %s", w.Code, w.Body)
	}
}

func TestCreateTrack_MissingRequiredFieldIs400(t *testing.T) {
	f := newFixture(t)

	w := f.do(t, http.MethodPost, "/tracks", f.token(t, uuid.NewString(), auth.RoleBroadcaster), `not json`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", w.Code, w.Body)
	}
}

func TestDeleteTrack_BroadcasterIsScopedToOwnTracks(t *testing.T) {
	f := newFixture(t)
	bcID := uuid.NewString()
	id := uuid.NewString()

	w := f.do(t, http.MethodDelete, "/tracks/"+id, f.token(t, bcID, auth.RoleBroadcaster), "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204: %s", w.Code, w.Body)
	}
	if f.tracks.deletedWith != [2]string{id, bcID} {
		t.Fatalf("DeleteTrack called with %v, want ownership scoping", f.tracks.deletedWith)
	}
}

// An admin passes an empty requester so the ownership predicate is dropped.
func TestDeleteTrack_AdminIsUnscoped(t *testing.T) {
	f := newFixture(t)
	id := uuid.NewString()

	w := f.do(t, http.MethodDelete, "/tracks/"+id, f.token(t, uuid.NewString(), auth.RoleAdmin), "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204: %s", w.Code, w.Body)
	}
	if f.tracks.deletedWith != [2]string{id, ""} {
		t.Fatalf("DeleteTrack called with %v, want an unscoped delete", f.tracks.deletedWith)
	}
}

func TestDeleteTrack_Forbidden(t *testing.T) {
	f := newFixture(t)

	w := f.do(t, http.MethodDelete, "/tracks/"+uuid.NewString(), f.token(t, "u-1", auth.RoleUser), "")
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403: %s", w.Code, w.Body)
	}
}

func TestDeleteTrack_NotFoundAndTimeout(t *testing.T) {
	tests := map[error]int{
		track.ErrNotFound:        http.StatusNotFound,
		context.DeadlineExceeded: http.StatusServiceUnavailable,
		errors.New("db down"):    http.StatusInternalServerError,
	}
	for err, want := range tests {
		f := newFixture(t)
		f.tracks.deleteErr = err

		w := f.do(t, http.MethodDelete, "/tracks/"+uuid.NewString(), f.token(t, uuid.NewString(), auth.RoleBroadcaster), "")
		if w.Code != want {
			t.Errorf("err %v: status = %d, want %d", err, w.Code, want)
		}
	}
}

// ── Track upload URL ──────────────────────────────────────────────────────────

func TestCreateTrackUploadURL_Success(t *testing.T) {
	f := newFixture(t)
	f.tracks.presign = [3]string{"https://minio/put", "tracks/bc-1/x.mp3", "https://minio/lyo/tracks/bc-1/x.mp3"}

	w := f.do(t, http.MethodPost, "/tracks/upload-url", f.token(t, uuid.NewString(), auth.RoleBroadcaster),
		`{"filename":"nightcall.mp3"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}

	var body struct {
		UploadURL string `json:"upload_url"`
		Key       string `json:"key"`
		AudioURL  string `json:"audio_url"`
	}
	decodeInto(t, w, &body)
	if body.UploadURL != "https://minio/put" || body.Key != "tracks/bc-1/x.mp3" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestCreateTrackUploadURL_RequiresBroadcaster(t *testing.T) {
	f := newFixture(t)

	w := f.do(t, http.MethodPost, "/tracks/upload-url", f.token(t, "u-1", auth.RoleUser), `{"filename":"x.mp3"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403: %s", w.Code, w.Body)
	}
}

func TestCreateTrackUploadURL_GatedByFeatureFlag(t *testing.T) {
	f := newFixture(t, "track_uploads")

	w := f.do(t, http.MethodPost, "/tracks/upload-url", f.token(t, uuid.NewString(), auth.RoleBroadcaster),
		`{"filename":"x.mp3"}`)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503: %s", w.Code, w.Body)
	}
}

func TestCreateTrackUploadURL_Errors(t *testing.T) {
	for err, want := range map[error]int{
		context.DeadlineExceeded: http.StatusServiceUnavailable,
		errors.New("s3 down"):    http.StatusInternalServerError,
	} {
		f := newFixture(t)
		f.tracks.presignErr = err

		w := f.do(t, http.MethodPost, "/tracks/upload-url", f.token(t, uuid.NewString(), auth.RoleBroadcaster),
			`{"filename":"x.mp3"}`)
		if w.Code != want {
			t.Errorf("err %v: status = %d, want %d", err, w.Code, want)
		}
	}
}

// ── Streams ───────────────────────────────────────────────────────────────────

func TestListStreams(t *testing.T) {
	f := newFixture(t)
	f.streams.list = []streaming.Stream{*newStream(uuid.NewString())}

	w := f.do(t, http.MethodGet, "/streams", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}
	var body struct {
		Items []api.Stream `json:"items"`
	}
	decodeInto(t, w, &body)
	if len(body.Items) != 1 || body.Items[0].Status != api.StreamStatus("live") {
		t.Fatalf("unexpected items: %+v", body.Items)
	}
}

func TestListStreams_Errors(t *testing.T) {
	for err, want := range map[error]int{
		context.DeadlineExceeded: http.StatusServiceUnavailable,
		errors.New("db down"):    http.StatusInternalServerError,
	} {
		f := newFixture(t)
		f.streams.listErr = err

		if w := f.do(t, http.MethodGet, "/streams", "", ""); w.Code != want {
			t.Errorf("err %v: status = %d, want %d", err, w.Code, want)
		}
	}
}

func TestCreateStream_Success(t *testing.T) {
	f := newFixture(t)
	bcID := uuid.NewString()

	w := f.do(t, http.MethodPost, "/streams", f.token(t, bcID, auth.RoleBroadcaster),
		`{"title":"My show","description":"desc"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", w.Code, w.Body)
	}
	if f.streams.started.BroadcasterID != bcID || f.streams.started.Description != "desc" {
		t.Fatalf("unexpected stream: %+v", f.streams.started)
	}
}

func TestCreateStream_RequiresBroadcaster(t *testing.T) {
	f := newFixture(t)

	w := f.do(t, http.MethodPost, "/streams", f.token(t, "u-1", auth.RoleUser), `{"title":"My show"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403: %s", w.Code, w.Body)
	}
}

func TestCreateStream_GatedByFeatureFlag(t *testing.T) {
	f := newFixture(t, "live_streaming")

	w := f.do(t, http.MethodPost, "/streams", f.token(t, uuid.NewString(), auth.RoleBroadcaster), `{"title":"My show"}`)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503: %s", w.Code, w.Body)
	}
}

// One live stream per broadcaster: a second attempt is a conflict.
func TestCreateStream_AlreadyLiveIs409(t *testing.T) {
	f := newFixture(t)
	f.streams.startErr = streaming.ErrAlreadyLive

	w := f.do(t, http.MethodPost, "/streams", f.token(t, uuid.NewString(), auth.RoleBroadcaster), `{"title":"My show"}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409: %s", w.Code, w.Body)
	}
}

func TestCreateStream_Errors(t *testing.T) {
	for err, want := range map[error]int{
		context.DeadlineExceeded: http.StatusServiceUnavailable,
		errors.New("db down"):    http.StatusInternalServerError,
	} {
		f := newFixture(t)
		f.streams.startErr = err

		w := f.do(t, http.MethodPost, "/streams", f.token(t, uuid.NewString(), auth.RoleBroadcaster), `{"title":"t"}`)
		if w.Code != want {
			t.Errorf("err %v: status = %d, want %d", err, w.Code, want)
		}
	}
}

func TestGetStream(t *testing.T) {
	f := newFixture(t)
	s := newStream(uuid.NewString())
	f.streams.streams[s.ID] = s

	w := f.do(t, http.MethodGet, "/streams/"+s.ID, "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}
	var got api.Stream
	decodeInto(t, w, &got)
	if got.Title != "My show" || got.Description == nil || *got.Description != "desc" {
		t.Fatalf("unexpected stream: %+v", got)
	}
	if got.EndedAt != nil {
		t.Errorf("ended_at should be nil for a live stream")
	}
}

func TestGetStream_NotFound(t *testing.T) {
	f := newFixture(t)

	if w := f.do(t, http.MethodGet, "/streams/"+uuid.NewString(), "", ""); w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404: %s", w.Code, w.Body)
	}
}

func TestDeleteStream_BroadcasterIsScopedToOwnStreams(t *testing.T) {
	f := newFixture(t)
	bcID := uuid.NewString()
	id := uuid.NewString()

	w := f.do(t, http.MethodDelete, "/streams/"+id, f.token(t, bcID, auth.RoleBroadcaster), "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204: %s", w.Code, w.Body)
	}
	if f.streams.endedWith != [2]string{id, bcID} {
		t.Fatalf("EndStream called with %v", f.streams.endedWith)
	}
}

func TestDeleteStream_AdminIsUnscoped(t *testing.T) {
	f := newFixture(t)
	id := uuid.NewString()

	w := f.do(t, http.MethodDelete, "/streams/"+id, f.token(t, uuid.NewString(), auth.RoleAdmin), "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204: %s", w.Code, w.Body)
	}
	if f.streams.endedWith != [2]string{id, ""} {
		t.Fatalf("EndStream called with %v", f.streams.endedWith)
	}
}

func TestDeleteStream_Errors(t *testing.T) {
	f := newFixture(t)
	w := f.do(t, http.MethodDelete, "/streams/"+uuid.NewString(), f.token(t, "u-1", auth.RoleUser), "")
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403: %s", w.Code, w.Body)
	}

	for err, want := range map[error]int{
		streaming.ErrNotFound:    http.StatusNotFound,
		context.DeadlineExceeded: http.StatusServiceUnavailable,
		errors.New("db down"):    http.StatusInternalServerError,
	} {
		f := newFixture(t)
		f.streams.endErr = err

		w := f.do(t, http.MethodDelete, "/streams/"+uuid.NewString(), f.token(t, uuid.NewString(), auth.RoleBroadcaster), "")
		if w.Code != want {
			t.Errorf("err %v: status = %d, want %d", err, w.Code, want)
		}
	}
}

// ── Favorites ─────────────────────────────────────────────────────────────────

func TestFavorites_RequireAuthentication(t *testing.T) {
	id := uuid.NewString()
	routes := []struct{ method, path string }{
		{http.MethodGet, "/users/me/favorites"},
		{http.MethodPost, "/users/me/favorites/tracks/" + id},
		{http.MethodDelete, "/users/me/favorites/tracks/" + id},
		{http.MethodPost, "/users/me/favorites/streams/" + id},
		{http.MethodDelete, "/users/me/favorites/streams/" + id},
		{http.MethodPost, "/users/me/favorites/playlists/" + id},
		{http.MethodDelete, "/users/me/favorites/playlists/" + id},
	}

	for _, r := range routes {
		f := newFixture(t)
		if w := f.do(t, r.method, r.path, "", ""); w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: status = %d, want 401", r.method, r.path, w.Code)
		}
	}
}

func TestFavorites_GatedByFeatureFlag(t *testing.T) {
	id := uuid.NewString()
	routes := []struct{ method, path string }{
		{http.MethodGet, "/users/me/favorites"},
		{http.MethodPost, "/users/me/favorites/tracks/" + id},
		{http.MethodDelete, "/users/me/favorites/tracks/" + id},
		{http.MethodPost, "/users/me/favorites/streams/" + id},
		{http.MethodDelete, "/users/me/favorites/streams/" + id},
		{http.MethodPost, "/users/me/favorites/playlists/" + id},
		{http.MethodDelete, "/users/me/favorites/playlists/" + id},
	}

	for _, r := range routes {
		f := newFixture(t, "favorites")
		token := f.token(t, uuid.NewString(), auth.RoleUser)
		if w := f.do(t, r.method, r.path, token, ""); w.Code != http.StatusServiceUnavailable {
			t.Errorf("%s %s: status = %d, want 503", r.method, r.path, w.Code)
		}
	}
}

func TestFavoriteTrack_Success(t *testing.T) {
	f := newFixture(t)
	tr := newTrack(uuid.NewString())
	f.tracks.tracks[tr.ID] = tr

	w := f.do(t, http.MethodPost, "/users/me/favorites/tracks/"+tr.ID, f.token(t, uuid.NewString(), auth.RoleUser), "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204: %s", w.Code, w.Body)
	}
	if len(f.users.added) != 1 || f.users.added[0] != "track:"+tr.ID {
		t.Fatalf("added = %v", f.users.added)
	}
}

// Favoriting something that doesn't exist must 404 rather than storing a dangling id.
func TestFavoriteTrack_UnknownTrackIs404(t *testing.T) {
	f := newFixture(t)

	w := f.do(t, http.MethodPost, "/users/me/favorites/tracks/"+uuid.NewString(), f.token(t, "u-1", auth.RoleUser), "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404: %s", w.Code, w.Body)
	}
	if len(f.users.added) != 0 {
		t.Fatalf("nothing should have been favorited, got %v", f.users.added)
	}
}

func TestFavoriteTrack_ServiceErrorIs500(t *testing.T) {
	f := newFixture(t)
	tr := newTrack(uuid.NewString())
	f.tracks.tracks[tr.ID] = tr
	f.users.err = errors.New("db down")

	w := f.do(t, http.MethodPost, "/users/me/favorites/tracks/"+tr.ID, f.token(t, "u-1", auth.RoleUser), "")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500: %s", w.Code, w.Body)
	}
}

func TestUnfavoriteTrack_Success(t *testing.T) {
	f := newFixture(t)
	id := uuid.NewString()

	w := f.do(t, http.MethodDelete, "/users/me/favorites/tracks/"+id, f.token(t, "u-1", auth.RoleUser), "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204: %s", w.Code, w.Body)
	}
	if len(f.users.removed) != 1 || f.users.removed[0] != "track:"+id {
		t.Fatalf("removed = %v", f.users.removed)
	}
}

func TestFavoriteStream_Success(t *testing.T) {
	f := newFixture(t)
	s := newStream(uuid.NewString())
	f.streams.streams[s.ID] = s

	w := f.do(t, http.MethodPost, "/users/me/favorites/streams/"+s.ID, f.token(t, "u-1", auth.RoleUser), "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204: %s", w.Code, w.Body)
	}
	if len(f.users.added) != 1 || f.users.added[0] != "stream:"+s.ID {
		t.Fatalf("added = %v", f.users.added)
	}
}

func TestFavoriteStream_UnknownStreamIs404(t *testing.T) {
	f := newFixture(t)

	w := f.do(t, http.MethodPost, "/users/me/favorites/streams/"+uuid.NewString(), f.token(t, "u-1", auth.RoleUser), "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404: %s", w.Code, w.Body)
	}
}

func TestUnfavoriteStream_Success(t *testing.T) {
	f := newFixture(t)
	id := uuid.NewString()

	w := f.do(t, http.MethodDelete, "/users/me/favorites/streams/"+id, f.token(t, "u-1", auth.RoleUser), "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204: %s", w.Code, w.Body)
	}
	if len(f.users.removed) != 1 || f.users.removed[0] != "stream:"+id {
		t.Fatalf("removed = %v", f.users.removed)
	}
}

func TestFavoritePlaylist_Success(t *testing.T) {
	f := newFixture(t)
	p := newPlaylist(uuid.NewString(), true)
	f.playlist.playlists[p.ID] = p

	w := f.do(t, http.MethodPost, "/users/me/favorites/playlists/"+p.ID, f.token(t, "u-1", auth.RoleUser), "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204: %s", w.Code, w.Body)
	}
	if len(f.users.added) != 1 || f.users.added[0] != "playlist:"+p.ID {
		t.Fatalf("added = %v", f.users.added)
	}
}

func TestFavoritePlaylist_UnknownPlaylistIs404(t *testing.T) {
	f := newFixture(t)

	w := f.do(t, http.MethodPost, "/users/me/favorites/playlists/"+uuid.NewString(), f.token(t, "u-1", auth.RoleUser), "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404: %s", w.Code, w.Body)
	}
}

func TestUnfavoritePlaylist_Success(t *testing.T) {
	f := newFixture(t)
	id := uuid.NewString()

	w := f.do(t, http.MethodDelete, "/users/me/favorites/playlists/"+id, f.token(t, "u-1", auth.RoleUser), "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204: %s", w.Code, w.Body)
	}
	if len(f.users.removed) != 1 || f.users.removed[0] != "playlist:"+id {
		t.Fatalf("removed = %v", f.users.removed)
	}
}

func TestUnfavorite_PropagatesErrors(t *testing.T) {
	for err, want := range map[error]int{
		context.DeadlineExceeded: http.StatusServiceUnavailable,
		errors.New("db down"):    http.StatusInternalServerError,
	} {
		for _, kind := range []string{"tracks", "streams", "playlists"} {
			f := newFixture(t)
			f.users.err = err

			w := f.do(t, http.MethodDelete, "/users/me/favorites/"+kind+"/"+uuid.NewString(),
				f.token(t, "u-1", auth.RoleUser), "")
			if w.Code != want {
				t.Errorf("%s / %v: status = %d, want %d", kind, err, w.Code, want)
			}
		}
	}
}

// ── ListFavorites ─────────────────────────────────────────────────────────────

func favoriteUser(ids ...uuid.UUID) *user.User {
	pg := make([]pgtype.UUID, len(ids))
	for i, id := range ids {
		pg[i] = pgtype.UUID{Bytes: id, Valid: true}
	}
	return &user.User{
		ID:                  pgtype.UUID{Bytes: uuid.New(), Valid: true},
		Username:            "alice",
		Email:               "alice@example.com",
		Role:                auth.RoleUser,
		FavoriteTrackIDs:    pg,
		FavoriteStreamIDs:   pg,
		FavoritePlaylistIDs: pg,
	}
}

func TestListFavorites_HydratesEachKind(t *testing.T) {
	f := newFixture(t)

	trackID, streamID, playlistID := uuid.New(), uuid.New(), uuid.New()
	tr := newTrack(uuid.NewString())
	tr.ID = trackID.String()
	f.tracks.tracks[tr.ID] = tr
	s := newStream(uuid.NewString())
	s.ID = streamID.String()
	f.streams.streams[s.ID] = s
	p := newPlaylist(uuid.NewString(), true)
	p.ID = playlistID.String()
	f.playlist.playlists[p.ID] = p

	u := favoriteUser()
	u.FavoriteTrackIDs = []pgtype.UUID{{Bytes: trackID, Valid: true}}
	u.FavoriteStreamIDs = []pgtype.UUID{{Bytes: streamID, Valid: true}}
	u.FavoritePlaylistIDs = []pgtype.UUID{{Bytes: playlistID, Valid: true}}
	f.users.user = u

	w := f.do(t, http.MethodGet, "/users/me/favorites", f.token(t, "u-1", auth.RoleUser), "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}

	var body struct {
		Tracks    []api.Track    `json:"tracks"`
		Streams   []api.Stream   `json:"streams"`
		Playlists []api.Playlist `json:"playlists"`
	}
	decodeInto(t, w, &body)
	if len(body.Tracks) != 1 || len(body.Streams) != 1 || len(body.Playlists) != 1 {
		t.Fatalf("unexpected favorites: %d tracks, %d streams, %d playlists",
			len(body.Tracks), len(body.Streams), len(body.Playlists))
	}
}

// A favorite whose target was deleted must be skipped, not fail the whole request.
func TestListFavorites_SkipsDeletedTargets(t *testing.T) {
	f := newFixture(t)
	f.users.user = favoriteUser(uuid.New(), uuid.New())

	w := f.do(t, http.MethodGet, "/users/me/favorites", f.token(t, "u-1", auth.RoleUser), "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}

	var body struct {
		Tracks    []api.Track    `json:"tracks"`
		Streams   []api.Stream   `json:"streams"`
		Playlists []api.Playlist `json:"playlists"`
	}
	decodeInto(t, w, &body)
	if len(body.Tracks) != 0 || len(body.Streams) != 0 || len(body.Playlists) != 0 {
		t.Fatalf("expected every dangling favorite to be skipped, got %+v", body)
	}
}

func TestListFavorites_Errors(t *testing.T) {
	for err, want := range map[error]int{
		context.DeadlineExceeded: http.StatusServiceUnavailable,
		errors.New("db down"):    http.StatusInternalServerError,
	} {
		f := newFixture(t)
		f.users.getErr = err

		w := f.do(t, http.MethodGet, "/users/me/favorites", f.token(t, "u-1", auth.RoleUser), "")
		if w.Code != want {
			t.Errorf("err %v: status = %d, want %d", err, w.Code, want)
		}
	}
}

// ── Playlists ─────────────────────────────────────────────────────────────────

func TestPlaylists_GatedByFeatureFlag(t *testing.T) {
	id := uuid.NewString()
	routes := []struct{ method, path, body string }{
		{http.MethodGet, "/playlists", ""},
		{http.MethodPost, "/playlists", `{"title":"t"}`},
		{http.MethodGet, "/playlists/" + id, ""},
		{http.MethodPatch, "/playlists/" + id, `{"title":"t"}`},
		{http.MethodDelete, "/playlists/" + id, ""},
		{http.MethodPost, "/playlists/" + id + "/tracks", `{"track_id":"` + id + `"}`},
		{http.MethodDelete, "/playlists/" + id + "/tracks/" + id, ""},
	}

	for _, r := range routes {
		f := newFixture(t, "playlists")
		token := f.token(t, uuid.NewString(), auth.RoleUser)
		if w := f.do(t, r.method, r.path, token, r.body); w.Code != http.StatusServiceUnavailable {
			t.Errorf("%s %s: status = %d, want 503", r.method, r.path, w.Code)
		}
	}
}

func TestPlaylists_RequireAuthentication(t *testing.T) {
	id := uuid.NewString()
	routes := []struct{ method, path, body string }{
		{http.MethodGet, "/playlists", ""},
		{http.MethodPost, "/playlists", `{"title":"t"}`},
		{http.MethodPatch, "/playlists/" + id, `{"title":"t"}`},
		{http.MethodDelete, "/playlists/" + id, ""},
		{http.MethodPost, "/playlists/" + id + "/tracks", `{"track_id":"` + id + `"}`},
		{http.MethodDelete, "/playlists/" + id + "/tracks/" + id, ""},
	}

	for _, r := range routes {
		f := newFixture(t)
		if w := f.do(t, r.method, r.path, "", r.body); w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: status = %d, want 401", r.method, r.path, w.Code)
		}
	}
}

func TestListPlaylists(t *testing.T) {
	f := newFixture(t)
	ownerID := uuid.NewString()
	f.playlist.list = []playlist.Playlist{*newPlaylist(ownerID, false), *newPlaylist(ownerID, true)}

	w := f.do(t, http.MethodGet, "/playlists", f.token(t, ownerID, auth.RoleUser), "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}
	var body struct {
		Items []api.Playlist `json:"items"`
	}
	decodeInto(t, w, &body)
	if len(body.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(body.Items))
	}
}

func TestListPlaylists_Errors(t *testing.T) {
	for err, want := range map[error]int{
		context.DeadlineExceeded: http.StatusServiceUnavailable,
		errors.New("db down"):    http.StatusInternalServerError,
	} {
		f := newFixture(t)
		f.playlist.listErr = err

		w := f.do(t, http.MethodGet, "/playlists", f.token(t, "u-1", auth.RoleUser), "")
		if w.Code != want {
			t.Errorf("err %v: status = %d, want %d", err, w.Code, want)
		}
	}
}

func TestCreatePlaylist_Success(t *testing.T) {
	f := newFixture(t)
	ownerID := uuid.NewString()

	w := f.do(t, http.MethodPost, "/playlists", f.token(t, ownerID, auth.RoleUser),
		`{"title":"Road trip","description":"for the car","is_public":true}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", w.Code, w.Body)
	}
	if f.playlist.created.OwnerID != ownerID || !f.playlist.created.IsPublic {
		t.Fatalf("unexpected playlist: %+v", f.playlist.created)
	}
}

// description and is_public are optional; a new playlist is private by default.
func TestCreatePlaylist_DefaultsToPrivate(t *testing.T) {
	f := newFixture(t)

	w := f.do(t, http.MethodPost, "/playlists", f.token(t, uuid.NewString(), auth.RoleUser), `{"title":"Road trip"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", w.Code, w.Body)
	}
	if f.playlist.created.IsPublic || f.playlist.created.Description != "" {
		t.Fatalf("unexpected defaults: %+v", f.playlist.created)
	}
}

func TestCreatePlaylist_Errors(t *testing.T) {
	for err, want := range map[error]int{
		context.DeadlineExceeded: http.StatusServiceUnavailable,
		errors.New("db down"):    http.StatusInternalServerError,
	} {
		f := newFixture(t)
		f.playlist.createErr = err

		w := f.do(t, http.MethodPost, "/playlists", f.token(t, "u-1", auth.RoleUser), `{"title":"t"}`)
		if w.Code != want {
			t.Errorf("err %v: status = %d, want %d", err, w.Code, want)
		}
	}
}

func TestGetPlaylist_PublicIsVisibleToAnyone(t *testing.T) {
	f := newFixture(t)
	trackID := uuid.NewString()
	tr := newTrack(uuid.NewString())
	tr.ID = trackID
	f.tracks.tracks[trackID] = tr

	p := newPlaylist(uuid.NewString(), true, trackID)
	f.playlist.playlists[p.ID] = p

	w := f.do(t, http.MethodGet, "/playlists/"+p.ID, "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}

	var got api.Playlist
	decodeInto(t, w, &got)
	if got.Tracks == nil || len(*got.Tracks) != 1 {
		t.Fatalf("expected the playlist's tracks to be hydrated, got %+v", got.Tracks)
	}
}

// A private playlist must 404 (not 403) for anyone but its owner, so its
// existence isn't leaked.
func TestGetPlaylist_PrivateIsHiddenFromOthers(t *testing.T) {
	ownerID := uuid.NewString()

	t.Run("anonymous", func(t *testing.T) {
		f := newFixture(t)
		p := newPlaylist(ownerID, false)
		f.playlist.playlists[p.ID] = p

		if w := f.do(t, http.MethodGet, "/playlists/"+p.ID, "", ""); w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})

	t.Run("another user", func(t *testing.T) {
		f := newFixture(t)
		p := newPlaylist(ownerID, false)
		f.playlist.playlists[p.ID] = p

		w := f.do(t, http.MethodGet, "/playlists/"+p.ID, f.token(t, uuid.NewString(), auth.RoleUser), "")
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})

	t.Run("owner", func(t *testing.T) {
		f := newFixture(t)
		p := newPlaylist(ownerID, false)
		f.playlist.playlists[p.ID] = p

		w := f.do(t, http.MethodGet, "/playlists/"+p.ID, f.token(t, ownerID, auth.RoleUser), "")
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", w.Code, w.Body)
		}
	})

	t.Run("admin", func(t *testing.T) {
		f := newFixture(t)
		p := newPlaylist(ownerID, false)
		f.playlist.playlists[p.ID] = p

		w := f.do(t, http.MethodGet, "/playlists/"+p.ID, f.token(t, uuid.NewString(), auth.RoleAdmin), "")
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", w.Code, w.Body)
		}
	})
}

func TestGetPlaylist_NotFound(t *testing.T) {
	f := newFixture(t)

	if w := f.do(t, http.MethodGet, "/playlists/"+uuid.NewString(), "", ""); w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestUpdatePlaylist_OwnerCanEdit(t *testing.T) {
	f := newFixture(t)
	ownerID := uuid.NewString()
	p := newPlaylist(ownerID, false)
	f.playlist.playlists[p.ID] = p

	w := f.do(t, http.MethodPatch, "/playlists/"+p.ID, f.token(t, ownerID, auth.RoleUser), `{"title":"New title"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}
	if f.playlist.updatedWith != [2]string{p.ID, ownerID} {
		t.Fatalf("Update called with %v", f.playlist.updatedWith)
	}

	var got api.Playlist
	decodeInto(t, w, &got)
	if got.Title != "New title" {
		t.Fatalf("title = %q", got.Title)
	}
}

func TestUpdatePlaylist_AdminIsUnscoped(t *testing.T) {
	f := newFixture(t)
	p := newPlaylist(uuid.NewString(), false)
	f.playlist.playlists[p.ID] = p

	w := f.do(t, http.MethodPatch, "/playlists/"+p.ID, f.token(t, uuid.NewString(), auth.RoleAdmin), `{"title":"New title"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}
	if f.playlist.updatedWith != [2]string{p.ID, ""} {
		t.Fatalf("Update called with %v, want an unscoped update", f.playlist.updatedWith)
	}
}

func TestUpdatePlaylist_NonOwnerIsForbidden(t *testing.T) {
	f := newFixture(t)
	p := newPlaylist(uuid.NewString(), false)
	f.playlist.playlists[p.ID] = p

	w := f.do(t, http.MethodPatch, "/playlists/"+p.ID, f.token(t, uuid.NewString(), auth.RoleUser), `{"title":"Mine now"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403: %s", w.Code, w.Body)
	}
}

func TestUpdatePlaylist_NotFound(t *testing.T) {
	f := newFixture(t)

	w := f.do(t, http.MethodPatch, "/playlists/"+uuid.NewString(), f.token(t, "u-1", auth.RoleUser), `{"title":"t"}`)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

// The row can disappear between the ownership check and the update.
func TestUpdatePlaylist_RacesDeletionTo404(t *testing.T) {
	f := newFixture(t)
	ownerID := uuid.NewString()
	p := newPlaylist(ownerID, false)
	f.playlist.playlists[p.ID] = p
	f.playlist.updateErr = playlist.ErrNotFound

	w := f.do(t, http.MethodPatch, "/playlists/"+p.ID, f.token(t, ownerID, auth.RoleUser), `{"title":"t"}`)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404: %s", w.Code, w.Body)
	}
}

func TestDeletePlaylist_OwnerCanDelete(t *testing.T) {
	f := newFixture(t)
	ownerID := uuid.NewString()
	p := newPlaylist(ownerID, false)
	f.playlist.playlists[p.ID] = p

	w := f.do(t, http.MethodDelete, "/playlists/"+p.ID, f.token(t, ownerID, auth.RoleUser), "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204: %s", w.Code, w.Body)
	}
	if f.playlist.deletedWith != [2]string{p.ID, ownerID} {
		t.Fatalf("Delete called with %v", f.playlist.deletedWith)
	}
}

func TestDeletePlaylist_AdminIsUnscoped(t *testing.T) {
	f := newFixture(t)
	p := newPlaylist(uuid.NewString(), false)
	f.playlist.playlists[p.ID] = p

	w := f.do(t, http.MethodDelete, "/playlists/"+p.ID, f.token(t, uuid.NewString(), auth.RoleAdmin), "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204: %s", w.Code, w.Body)
	}
	if f.playlist.deletedWith != [2]string{p.ID, ""} {
		t.Fatalf("Delete called with %v", f.playlist.deletedWith)
	}
}

func TestDeletePlaylist_NonOwnerIsForbidden(t *testing.T) {
	f := newFixture(t)
	p := newPlaylist(uuid.NewString(), false)
	f.playlist.playlists[p.ID] = p

	w := f.do(t, http.MethodDelete, "/playlists/"+p.ID, f.token(t, uuid.NewString(), auth.RoleUser), "")
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestDeletePlaylist_NotFoundBeforeAndAfterTheOwnershipCheck(t *testing.T) {
	f := newFixture(t)
	if w := f.do(t, http.MethodDelete, "/playlists/"+uuid.NewString(), f.token(t, "u-1", auth.RoleUser), ""); w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}

	f = newFixture(t)
	ownerID := uuid.NewString()
	p := newPlaylist(ownerID, false)
	f.playlist.playlists[p.ID] = p
	f.playlist.deleteErr = playlist.ErrNotFound

	if w := f.do(t, http.MethodDelete, "/playlists/"+p.ID, f.token(t, ownerID, auth.RoleUser), ""); w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestAddTrackToPlaylist_Success(t *testing.T) {
	f := newFixture(t)
	ownerID := uuid.NewString()
	p := newPlaylist(ownerID, false)
	f.playlist.playlists[p.ID] = p
	tr := newTrack(uuid.NewString())
	f.tracks.tracks[tr.ID] = tr

	w := f.do(t, http.MethodPost, "/playlists/"+p.ID+"/tracks", f.token(t, ownerID, auth.RoleUser),
		`{"track_id":"`+tr.ID+`"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}
	if f.playlist.addedWith != [3]string{p.ID, ownerID, tr.ID} {
		t.Fatalf("AddTrack called with %v", f.playlist.addedWith)
	}

	var got api.Playlist
	decodeInto(t, w, &got)
	if len(got.TrackIds) != 1 {
		t.Fatalf("track ids = %v", got.TrackIds)
	}
}

func TestAddTrackToPlaylist_UnknownTrackIs404(t *testing.T) {
	f := newFixture(t)
	ownerID := uuid.NewString()
	p := newPlaylist(ownerID, false)
	f.playlist.playlists[p.ID] = p

	w := f.do(t, http.MethodPost, "/playlists/"+p.ID+"/tracks", f.token(t, ownerID, auth.RoleUser),
		`{"track_id":"`+uuid.NewString()+`"}`)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404: %s", w.Code, w.Body)
	}
}

func TestAddTrackToPlaylist_NonOwnerIsForbidden(t *testing.T) {
	f := newFixture(t)
	p := newPlaylist(uuid.NewString(), false)
	f.playlist.playlists[p.ID] = p

	w := f.do(t, http.MethodPost, "/playlists/"+p.ID+"/tracks", f.token(t, uuid.NewString(), auth.RoleUser),
		`{"track_id":"`+uuid.NewString()+`"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestAddTrackToPlaylist_AdminIsUnscoped(t *testing.T) {
	f := newFixture(t)
	p := newPlaylist(uuid.NewString(), false)
	f.playlist.playlists[p.ID] = p
	tr := newTrack(uuid.NewString())
	f.tracks.tracks[tr.ID] = tr

	w := f.do(t, http.MethodPost, "/playlists/"+p.ID+"/tracks", f.token(t, uuid.NewString(), auth.RoleAdmin),
		`{"track_id":"`+tr.ID+`"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}
	if f.playlist.addedWith[1] != "" {
		t.Fatalf("AddTrack requester = %q, want empty for an admin", f.playlist.addedWith[1])
	}
}

func TestAddTrackToPlaylist_UnknownPlaylistIs404(t *testing.T) {
	f := newFixture(t)

	w := f.do(t, http.MethodPost, "/playlists/"+uuid.NewString()+"/tracks", f.token(t, "u-1", auth.RoleUser),
		`{"track_id":"`+uuid.NewString()+`"}`)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestRemoveTrackFromPlaylist_Success(t *testing.T) {
	f := newFixture(t)
	ownerID := uuid.NewString()
	trackID := uuid.NewString()
	p := newPlaylist(ownerID, false, trackID)
	f.playlist.playlists[p.ID] = p

	w := f.do(t, http.MethodDelete, "/playlists/"+p.ID+"/tracks/"+trackID, f.token(t, ownerID, auth.RoleUser), "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}
	if f.playlist.removedWith != [3]string{p.ID, ownerID, trackID} {
		t.Fatalf("RemoveTrack called with %v", f.playlist.removedWith)
	}
}

func TestRemoveTrackFromPlaylist_AdminIsUnscoped(t *testing.T) {
	f := newFixture(t)
	trackID := uuid.NewString()
	p := newPlaylist(uuid.NewString(), false, trackID)
	f.playlist.playlists[p.ID] = p

	w := f.do(t, http.MethodDelete, "/playlists/"+p.ID+"/tracks/"+trackID, f.token(t, uuid.NewString(), auth.RoleAdmin), "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}
	if f.playlist.removedWith[1] != "" {
		t.Fatalf("RemoveTrack requester = %q, want empty for an admin", f.playlist.removedWith[1])
	}
}

func TestRemoveTrackFromPlaylist_NonOwnerIsForbidden(t *testing.T) {
	f := newFixture(t)
	trackID := uuid.NewString()
	p := newPlaylist(uuid.NewString(), false, trackID)
	f.playlist.playlists[p.ID] = p

	w := f.do(t, http.MethodDelete, "/playlists/"+p.ID+"/tracks/"+trackID, f.token(t, uuid.NewString(), auth.RoleUser), "")
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestRemoveTrackFromPlaylist_NotFound(t *testing.T) {
	f := newFixture(t)
	w := f.do(t, http.MethodDelete, "/playlists/"+uuid.NewString()+"/tracks/"+uuid.NewString(),
		f.token(t, "u-1", auth.RoleUser), "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}

	f = newFixture(t)
	ownerID := uuid.NewString()
	trackID := uuid.NewString()
	p := newPlaylist(ownerID, false, trackID)
	f.playlist.playlists[p.ID] = p
	f.playlist.mutateErr = playlist.ErrNotFound

	w = f.do(t, http.MethodDelete, "/playlists/"+p.ID+"/tracks/"+trackID, f.token(t, ownerID, auth.RoleUser), "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

// ── Admin feature flags (not yet implemented) ─────────────────────────────────

func TestFeatureFlagEndpoints_NotImplemented(t *testing.T) {
	f := newFixture(t)
	token := f.token(t, uuid.NewString(), auth.RoleAdmin)

	for _, r := range []struct{ method, path, body string }{
		{http.MethodGet, "/admin/features", ""},
		{http.MethodPatch, "/admin/features/playlists/toggle", `{"enabled":false}`},
	} {
		w := f.do(t, r.method, r.path, token, r.body)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("%s %s: status = %d, want 500 (handler returns errNotImplemented)", r.method, r.path, w.Code)
		}
	}
}

// ── Timeouts and unexpected failures on the remaining routes ──────────────────

// Every handler maps context.DeadlineExceeded to 503 and anything else to 500;
// this walks the routes whose error paths the happy-path tests don't reach.
func TestRemainingRoutes_TimeoutAnd500(t *testing.T) {
	id := uuid.NewString()

	cases := []struct {
		name   string
		inject func(*fixture, error)
		method string
		path   string
		body   string
		role   auth.Role
	}{
		{"get track", func(f *fixture, err error) { f.tracks.getErr = err },
			http.MethodGet, "/tracks/" + id, "", ""},
		{"get stream", func(f *fixture, err error) { f.streams.getErr = err },
			http.MethodGet, "/streams/" + id, "", ""},
		{"favorite track lookup", func(f *fixture, err error) { f.tracks.getErr = err },
			http.MethodPost, "/users/me/favorites/tracks/" + id, "", auth.RoleUser},
		{"favorite stream lookup", func(f *fixture, err error) { f.streams.getErr = err },
			http.MethodPost, "/users/me/favorites/streams/" + id, "", auth.RoleUser},
		{"favorite playlist lookup", func(f *fixture, err error) { f.playlist.getErr = err },
			http.MethodPost, "/users/me/favorites/playlists/" + id, "", auth.RoleUser},
		{"get playlist", func(f *fixture, err error) { f.playlist.getErr = err },
			http.MethodGet, "/playlists/" + id, "", ""},
		{"update playlist lookup", func(f *fixture, err error) { f.playlist.getErr = err },
			http.MethodPatch, "/playlists/" + id, `{"title":"t"}`, auth.RoleUser},
		{"delete playlist lookup", func(f *fixture, err error) { f.playlist.getErr = err },
			http.MethodDelete, "/playlists/" + id, "", auth.RoleUser},
		{"add track to playlist lookup", func(f *fixture, err error) { f.playlist.getErr = err },
			http.MethodPost, "/playlists/" + id + "/tracks", `{"track_id":"` + id + `"}`, auth.RoleUser},
		{"remove track from playlist lookup", func(f *fixture, err error) { f.playlist.getErr = err },
			http.MethodDelete, "/playlists/" + id + "/tracks/" + id, "", auth.RoleUser},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for err, want := range map[error]int{
				context.DeadlineExceeded: http.StatusServiceUnavailable,
				errors.New("db down"):    http.StatusInternalServerError,
			} {
				f := newFixture(t)
				tc.inject(f, err)

				w := f.do(t, tc.method, tc.path, f.token(t, uuid.NewString(), tc.role), tc.body)
				if w.Code != want {
					t.Errorf("%v: status = %d, want %d: %s", err, w.Code, want, w.Body)
				}
			}
		})
	}
}

// The same mapping applies after the ownership check, on the mutation itself.
func TestPlaylistMutations_TimeoutAnd500(t *testing.T) {
	ownerID := uuid.NewString()

	cases := []struct {
		name   string
		inject func(*fixture, error)
		method string
		path   func(string) string
		body   func(string) string
	}{
		{"update", func(f *fixture, err error) { f.playlist.updateErr = err },
			http.MethodPatch, func(id string) string { return "/playlists/" + id }, func(string) string { return `{"title":"t"}` }},
		{"delete", func(f *fixture, err error) { f.playlist.deleteErr = err },
			http.MethodDelete, func(id string) string { return "/playlists/" + id }, func(string) string { return "" }},
		{"remove track", func(f *fixture, err error) { f.playlist.mutateErr = err },
			http.MethodDelete, func(id string) string { return "/playlists/" + id + "/tracks/" + uuid.NewString() },
			func(string) string { return "" }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for err, want := range map[error]int{
				context.DeadlineExceeded: http.StatusServiceUnavailable,
				errors.New("db down"):    http.StatusInternalServerError,
			} {
				f := newFixture(t)
				p := newPlaylist(ownerID, false)
				f.playlist.playlists[p.ID] = p
				tc.inject(f, err)

				w := f.do(t, tc.method, tc.path(p.ID), f.token(t, ownerID, auth.RoleUser), tc.body(p.ID))
				if w.Code != want {
					t.Errorf("%v: status = %d, want %d: %s", err, w.Code, want, w.Body)
				}
			}
		})
	}
}

func TestAddTrackToPlaylist_TrackLookupAndMutationErrors(t *testing.T) {
	ownerID := uuid.NewString()

	t.Run("track lookup", func(t *testing.T) {
		for err, want := range map[error]int{
			context.DeadlineExceeded: http.StatusServiceUnavailable,
			errors.New("db down"):    http.StatusInternalServerError,
		} {
			f := newFixture(t)
			p := newPlaylist(ownerID, false)
			f.playlist.playlists[p.ID] = p
			f.tracks.getErr = err

			w := f.do(t, http.MethodPost, "/playlists/"+p.ID+"/tracks", f.token(t, ownerID, auth.RoleUser),
				`{"track_id":"`+uuid.NewString()+`"}`)
			if w.Code != want {
				t.Errorf("%v: status = %d, want %d", err, w.Code, want)
			}
		}
	})

	t.Run("mutation", func(t *testing.T) {
		for err, want := range map[error]int{
			context.DeadlineExceeded: http.StatusServiceUnavailable,
			errors.New("db down"):    http.StatusInternalServerError,
			playlist.ErrNotFound:     http.StatusNotFound,
		} {
			f := newFixture(t)
			p := newPlaylist(ownerID, false)
			f.playlist.playlists[p.ID] = p
			tr := newTrack(uuid.NewString())
			f.tracks.tracks[tr.ID] = tr
			f.playlist.mutateErr = err

			w := f.do(t, http.MethodPost, "/playlists/"+p.ID+"/tracks", f.token(t, ownerID, auth.RoleUser),
				`{"track_id":"`+tr.ID+`"}`)
			if w.Code != want {
				t.Errorf("%v: status = %d, want %d", err, w.Code, want)
			}
		}
	})
}

// A playlist whose stored track IDs are not UUIDs can't satisfy the contract.
func TestGetPlaylist_NonUUIDTrackIDIs500(t *testing.T) {
	f := newFixture(t)
	p := newPlaylist(uuid.NewString(), true, "not-a-uuid")
	f.playlist.playlists[p.ID] = p

	if w := f.do(t, http.MethodGet, "/playlists/"+p.ID, "", ""); w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500: %s", w.Code, w.Body)
	}
}

func TestListPlaylists_NonUUIDOwnerIs500(t *testing.T) {
	f := newFixture(t)
	f.playlist.list = []playlist.Playlist{{ID: uuid.NewString(), OwnerID: "not-a-uuid", TrackIDs: []string{}}}

	w := f.do(t, http.MethodGet, "/playlists", f.token(t, uuid.NewString(), auth.RoleUser), "")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500: %s", w.Code, w.Body)
	}
}

func TestListStreams_NonUUIDIs500(t *testing.T) {
	f := newFixture(t)
	f.streams.list = []streaming.Stream{{ID: "not-a-uuid", BroadcasterID: uuid.NewString()}}

	if w := f.do(t, http.MethodGet, "/streams", "", ""); w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500: %s", w.Code, w.Body)
	}
}

// A favorite whose target has a corrupt ID must fail loudly rather than be skipped.
func TestListFavorites_NonUUIDTargetIs500(t *testing.T) {
	f := newFixture(t)
	trackID := uuid.New()
	f.tracks.tracks[trackID.String()] = &track.Track{ID: trackID.String(), BroadcasterID: "not-a-uuid"}
	f.users.user = &user.User{
		ID:               pgtype.UUID{Bytes: uuid.New(), Valid: true},
		FavoriteTrackIDs: []pgtype.UUID{{Bytes: trackID, Valid: true}},
	}

	w := f.do(t, http.MethodGet, "/users/me/favorites", f.token(t, "u-1", auth.RoleUser), "")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500: %s", w.Code, w.Body)
	}
}

// The mobile client tells a disabled feature apart from the other causes of a
// 503 (timeouts, outages) by the message suffix, and re-syncs its flags when it
// sees one. Rewording a gate here without updating mobile/lib/core/api breaks
// that, so pin the shape.
func TestFeatureGates_MessageEndsWithDisabled(t *testing.T) {
	cases := []struct {
		flag, method, path, body string
		role                     auth.Role
	}{
		{"track_uploads", http.MethodPost, "/tracks/upload-url", `{"filename":"x.mp3"}`, auth.RoleBroadcaster},
		{"live_streaming", http.MethodPost, "/streams", `{"title":"x"}`, auth.RoleBroadcaster},
		{"favorites", http.MethodGet, "/users/me/favorites", "", auth.RoleUser},
		{"playlists", http.MethodGet, "/playlists", "", auth.RoleUser},
		{"account_deletion", http.MethodDelete, "/users/me", "", auth.RoleUser},
	}

	for _, tc := range cases {
		t.Run(tc.flag, func(t *testing.T) {
			f := newFixture(t, tc.flag)
			w := f.do(t, tc.method, tc.path, f.token(t, uuid.NewString(), tc.role), tc.body)

			if w.Code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, want 503: %s", w.Code, w.Body)
			}
			msg := strings.ToLower(strings.TrimSpace(w.Body.String()))
			if !strings.HasSuffix(msg, "disabled") {
				t.Fatalf("message = %q, want it to end with \"disabled\"", msg)
			}
		})
	}
}

// ── DELETE /users/me ──────────────────────────────────────────────────────────

func TestDeleteUserByID_ErasesTheCallersOwnAccount(t *testing.T) {
	f := newFixture(t)
	id := uuid.NewString()

	w := f.do(t, http.MethodDelete, "/users/me", f.token(t, id, auth.RoleUser), "")

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204: %s", w.Code, w.Body)
	}
	// The subject comes from the token, never from the request — this is what
	// keeps "delete my account" from becoming "delete an account".
	if f.delUser.deletedID != id {
		t.Fatalf("deleted %q, want the caller %q", f.delUser.deletedID, id)
	}
}

func TestDeleteUserByID_RequiresAuthentication(t *testing.T) {
	f := newFixture(t)

	w := f.do(t, http.MethodDelete, "/users/me", "", "")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401: %s", w.Code, w.Body)
	}
	if f.delUser.deletedID != "" {
		t.Fatalf("an anonymous request deleted %q", f.delUser.deletedID)
	}
}

// An admin deleting themselves is still only deleting themselves: the role
// changes nothing about which account the handler picks.
func TestDeleteUserByID_AdminStillOnlyDeletesItself(t *testing.T) {
	f := newFixture(t)
	id := uuid.NewString()

	w := f.do(t, http.MethodDelete, "/users/me", f.token(t, id, auth.RoleAdmin), "")

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204: %s", w.Code, w.Body)
	}
	if f.delUser.deletedID != id {
		t.Fatalf("deleted %q, want %q", f.delUser.deletedID, id)
	}
}

// Retrying a delete that already succeeded must not look like a failure.
func TestDeleteUserByID_IsIdempotent(t *testing.T) {
	f := newFixture(t)
	f.delUser.err = user.ErrNotFound

	w := f.do(t, http.MethodDelete, "/users/me", f.token(t, uuid.NewString(), auth.RoleUser), "")

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204: %s", w.Code, w.Body)
	}
}

func TestDeleteUserByID_TimeoutIs503(t *testing.T) {
	f := newFixture(t)
	f.delUser.err = context.DeadlineExceeded

	w := f.do(t, http.MethodDelete, "/users/me", f.token(t, uuid.NewString(), auth.RoleUser), "")

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503: %s", w.Code, w.Body)
	}
}
