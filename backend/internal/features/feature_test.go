package features_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hyppoliteprn/lyo/internal/features"
)

type stubRepo struct {
	all     []features.Flag
	allErr  error
	enabled map[string]bool
	isErr   error
	toggled *features.Flag
	togErr  error
}

func (s *stubRepo) All(_ context.Context) ([]features.Flag, error) { return s.all, s.allErr }
func (s *stubRepo) IsEnabled(_ context.Context, name string) (bool, error) {
	if s.isErr != nil {
		return false, s.isErr
	}
	return s.enabled[name], nil
}
func (s *stubRepo) Toggle(_ context.Context, _ string, _ bool) (*features.Flag, error) {
	return s.toggled, s.togErr
}

func TestAll_DelegatesToRepo(t *testing.T) {
	want := []features.Flag{{Name: "playlists", Enabled: true, UpdatedAt: time.Now()}}
	svc := features.NewService(&stubRepo{all: want})

	got, err := svc.All(context.Background())
	if err != nil {
		t.Fatalf("all: %v", err)
	}
	if len(got) != 1 || got[0].Name != "playlists" {
		t.Fatalf("unexpected flags: %+v", got)
	}
}

func TestAll_PropagatesRepoError(t *testing.T) {
	sentinel := errors.New("db down")
	if _, err := features.NewService(&stubRepo{allErr: sentinel}).All(context.Background()); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
}

func TestIsEnabled(t *testing.T) {
	svc := features.NewService(&stubRepo{enabled: map[string]bool{"playlists": true, "transcoding": false}})

	if !svc.IsEnabled(context.Background(), "playlists") {
		t.Error("playlists should be enabled")
	}
	if svc.IsEnabled(context.Background(), "transcoding") {
		t.Error("transcoding should be disabled")
	}
	if svc.IsEnabled(context.Background(), "never_seeded") {
		t.Error("an unknown flag must not be enabled")
	}
}

// A repository failure must fail closed: a feature is off if we can't prove it's on.
func TestIsEnabled_FailsClosedOnRepoError(t *testing.T) {
	svc := features.NewService(&stubRepo{isErr: errors.New("db down"), enabled: map[string]bool{"playlists": true}})

	if svc.IsEnabled(context.Background(), "playlists") {
		t.Fatal("expected IsEnabled to return false when the repository errors")
	}
}

func TestToggle_DelegatesToRepo(t *testing.T) {
	want := &features.Flag{Name: "transcoding", Enabled: true}
	svc := features.NewService(&stubRepo{toggled: want})

	got, err := svc.Toggle(context.Background(), "transcoding", true)
	if err != nil {
		t.Fatalf("toggle: %v", err)
	}
	if got.Name != "transcoding" || !got.Enabled {
		t.Fatalf("unexpected flag: %+v", got)
	}
}

func TestToggle_PropagatesRepoError(t *testing.T) {
	sentinel := errors.New("no such flag")
	if _, err := features.NewService(&stubRepo{togErr: sentinel}).Toggle(context.Background(), "nope", true); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
}

// Every flag documented in CLAUDE.md must be seeded, with a unique name and a description.
func TestSeeds_AreWellFormed(t *testing.T) {
	seen := make(map[string]bool, len(features.Seeds))
	for _, f := range features.Seeds {
		if f.Name == "" {
			t.Errorf("seed with empty name: %+v", f)
		}
		if f.Description == "" {
			t.Errorf("seed %q has no description", f.Name)
		}
		if seen[f.Name] {
			t.Errorf("duplicate seed %q", f.Name)
		}
		seen[f.Name] = true
	}

	for _, required := range []string{"live_streaming", "track_uploads", "playlists", "favorites"} {
		if !seen[required] {
			t.Errorf("missing seed for %q", required)
		}
	}
}
