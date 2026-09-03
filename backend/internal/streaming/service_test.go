package streaming

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeStreamRepo struct {
	hasLive    bool
	hasLiveErr error
	created    *Stream
	createErr  error
	ended      *Stream
	endErr     error
	got        *Stream
	getErr     error
	live       []Stream
	listErr    error

	createdWith  [3]string
	endedWith    [2]string
	hasLiveCalls int
}

func (f *fakeStreamRepo) Create(_ context.Context, broadcasterID, title, description string) (*Stream, error) {
	f.createdWith = [3]string{broadcasterID, title, description}
	return f.created, f.createErr
}
func (f *fakeStreamRepo) Get(_ context.Context, _ string) (*Stream, error) { return f.got, f.getErr }
func (f *fakeStreamRepo) ListLive(_ context.Context) ([]Stream, error)     { return f.live, f.listErr }
func (f *fakeStreamRepo) End(_ context.Context, id, broadcasterID string) (*Stream, error) {
	f.endedWith = [2]string{id, broadcasterID}
	return f.ended, f.endErr
}
func (f *fakeStreamRepo) HasLive(_ context.Context, _ string) (bool, error) {
	f.hasLiveCalls++
	return f.hasLive, f.hasLiveErr
}

func newTestService(repo StreamRepository) *Service {
	return NewService(repo, 8, testLogger())
}

func TestStartStream_CreatesRecordAndHub(t *testing.T) {
	repo := &fakeStreamRepo{created: &Stream{ID: "stream-1", BroadcasterID: "bc-1", Status: "live", StartedAt: time.Now()}}
	svc := newTestService(repo)

	got, err := svc.StartStream(context.Background(), "bc-1", "My show", "desc")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if got.ID != "stream-1" {
		t.Errorf("stream id = %q", got.ID)
	}
	if repo.createdWith != [3]string{"bc-1", "My show", "desc"} {
		t.Errorf("repo.Create called with %v", repo.createdWith)
	}
	if svc.Hub("stream-1") == nil {
		t.Fatal("expected a hub to be registered for the new stream")
	}
}

// A broadcaster may only have one live stream at a time.
func TestStartStream_RejectsSecondLiveStream(t *testing.T) {
	repo := &fakeStreamRepo{hasLive: true}
	svc := newTestService(repo)

	_, err := svc.StartStream(context.Background(), "bc-1", "My show", "")
	if !errors.Is(err, ErrAlreadyLive) {
		t.Fatalf("err = %v, want %v", err, ErrAlreadyLive)
	}
	if repo.createdWith != [3]string{} {
		t.Error("repo.Create must not be called when a stream is already live")
	}
}

func TestStartStream_PropagatesRepoErrors(t *testing.T) {
	sentinel := errors.New("db down")

	t.Run("HasLive fails", func(t *testing.T) {
		svc := newTestService(&fakeStreamRepo{hasLiveErr: sentinel})
		if _, err := svc.StartStream(context.Background(), "bc-1", "t", ""); !errors.Is(err, sentinel) {
			t.Fatalf("err = %v, want %v", err, sentinel)
		}
	})

	t.Run("Create fails", func(t *testing.T) {
		svc := newTestService(&fakeStreamRepo{createErr: sentinel})
		if _, err := svc.StartStream(context.Background(), "bc-1", "t", ""); !errors.Is(err, sentinel) {
			t.Fatalf("err = %v, want %v", err, sentinel)
		}
		if svc.Hub("stream-1") != nil {
			t.Error("no hub should be registered when the DB insert fails")
		}
	})
}

func TestEndStream_ClosesAndRemovesHub(t *testing.T) {
	repo := &fakeStreamRepo{
		created: &Stream{ID: "stream-1", BroadcasterID: "bc-1"},
		ended:   &Stream{ID: "stream-1", BroadcasterID: "bc-1", Status: "ended"},
	}
	svc := newTestService(repo)

	if _, err := svc.StartStream(context.Background(), "bc-1", "t", ""); err != nil {
		t.Fatal(err)
	}
	hub := svc.Hub("stream-1")

	got, err := svc.EndStream(context.Background(), "stream-1", "bc-1")
	if err != nil {
		t.Fatalf("end: %v", err)
	}
	if got.Status != "ended" {
		t.Errorf("status = %q, want ended", got.Status)
	}
	if repo.endedWith != [2]string{"stream-1", "bc-1"} {
		t.Errorf("repo.End called with %v", repo.endedWith)
	}
	if svc.Hub("stream-1") != nil {
		t.Error("hub should be removed from the pool")
	}

	select {
	case <-hub.Done():
	case <-time.After(time.Second):
		t.Fatal("the hub was not closed, so ingest/listen loops would leak")
	}
}

// A failed End must leave the hub in place — the stream is still live.
func TestEndStream_KeepsHubWhenRepoFails(t *testing.T) {
	sentinel := errors.New("not your stream")
	repo := &fakeStreamRepo{created: &Stream{ID: "stream-1"}, endErr: sentinel}
	svc := newTestService(repo)

	if _, err := svc.StartStream(context.Background(), "bc-1", "t", ""); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.EndStream(context.Background(), "stream-1", "bc-1"); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
	if svc.Hub("stream-1") == nil {
		t.Fatal("hub must survive a failed EndStream")
	}
}

func TestEndStream_UnknownStreamIsNotAPanic(t *testing.T) {
	svc := newTestService(&fakeStreamRepo{ended: &Stream{ID: "stream-9"}})

	if _, err := svc.EndStream(context.Background(), "stream-9", ""); err != nil {
		t.Fatalf("end: %v", err)
	}
}

func TestGetStream_DelegatesToRepo(t *testing.T) {
	svc := newTestService(&fakeStreamRepo{got: &Stream{ID: "stream-1"}})

	got, err := svc.GetStream(context.Background(), "stream-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != "stream-1" {
		t.Errorf("id = %q", got.ID)
	}

	svc = newTestService(&fakeStreamRepo{getErr: ErrNotFound})
	if _, err := svc.GetStream(context.Background(), "nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrNotFound)
	}
}

func TestListLiveStreams_DelegatesToRepo(t *testing.T) {
	svc := newTestService(&fakeStreamRepo{live: []Stream{{ID: "a"}, {ID: "b"}}})

	got, err := svc.ListLiveStreams(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}

	sentinel := errors.New("db down")
	svc = newTestService(&fakeStreamRepo{listErr: sentinel})
	if _, err := svc.ListLiveStreams(context.Background()); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
}

func TestHub_NilForUnknownStream(t *testing.T) {
	if h := newTestService(&fakeStreamRepo{}).Hub("nope"); h != nil {
		t.Fatal("expected nil hub for an unknown stream")
	}
}
