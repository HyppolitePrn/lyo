package streaming

import (
	"context"
	"log/slog"
	"sync"
)

// Service manages stream lifecycle and the pool of active Hubs.
type Service struct {
	repo    StreamRepository
	hubs    map[string]*Hub
	mu      sync.Mutex
	bufSize int
	logger  *slog.Logger
}

func NewService(repo StreamRepository, bufSize int, logger *slog.Logger) *Service {
	return &Service{
		repo:    repo,
		hubs:    make(map[string]*Hub),
		bufSize: bufSize,
		logger:  logger,
	}
}

// StartStream creates a DB record and a Hub for the new broadcast.
// A broadcaster can only have one live stream at a time.
func (s *Service) StartStream(ctx context.Context, broadcasterID, title, description string) (*Stream, error) {
	has, err := s.repo.HasLive(ctx, broadcasterID)
	if err != nil {
		return nil, err
	}
	if has {
		return nil, ErrAlreadyLive
	}

	stream, err := s.repo.Create(ctx, broadcasterID, title, description)
	if err != nil {
		return nil, err
	}

	hub := NewHub(s.bufSize, s.logger)
	s.mu.Lock()
	s.hubs[stream.ID] = hub
	s.mu.Unlock()

	s.logger.Info("stream started",
		slog.String("stream_id", stream.ID),
		slog.String("broadcaster_id", broadcasterID))
	return stream, nil
}

// EndStream soft-ends the stream and removes its Hub from the pool.
// If broadcasterID is non-empty, ownership is enforced at the DB level.
func (s *Service) EndStream(ctx context.Context, id, broadcasterID string) (*Stream, error) {
	stream, err := s.repo.End(ctx, id, broadcasterID)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	if hub, ok := s.hubs[id]; ok {
		hub.Close()
		delete(s.hubs, id)
	}
	s.mu.Unlock()

	s.logger.Info("stream ended", slog.String("stream_id", id))
	return stream, nil
}

func (s *Service) GetStream(ctx context.Context, id string) (*Stream, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) ListLiveStreams(ctx context.Context) ([]Stream, error) {
	return s.repo.ListLive(ctx)
}

// Hub returns the active Hub for a stream, or nil if the stream is not live.
func (s *Service) Hub(streamID string) *Hub {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hubs[streamID]
}
