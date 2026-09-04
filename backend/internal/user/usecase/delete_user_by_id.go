package usecase

import (
	"context"
	"fmt"

	"github.com/hyppoliteprn/lyo/internal/user"
)

// TrackPurger erases a broadcaster's uploaded audio. Satisfied by
// *track.Service; kept as an interface here so the user package never learns
// about S3.
type TrackPurger interface {
	PurgeByBroadcaster(ctx context.Context, broadcasterID string) error
}

// DeleteUserByIDUsecase permanently erases an account and everything it owns.
type DeleteUserByIDUsecase struct {
	repo   user.Repository
	tracks TrackPurger
}

// NewDeleteUserByIDUsecase creates a DeleteUserByIDUsecase. tracks may be nil
// when no storage backend is configured, in which case there are no audio
// objects to erase in the first place.
func NewDeleteUserByIDUsecase(repo user.Repository, tracks TrackPurger) *DeleteUserByIDUsecase {
	return &DeleteUserByIDUsecase{repo: repo, tracks: tracks}
}

// Execute deletes the user identified by id.
//
// Audio objects go first, on purpose. Every table referencing users cascades,
// so deleting the row would take the track rows — and with them the only
// record of which S3 objects belonged to this account — leaving undeletable
// personal data behind. Doing it in this order can at worst leave the account
// intact after a storage failure, which the caller can retry; the reverse
// order fails in a way nobody can repair.
func (uc *DeleteUserByIDUsecase) Execute(ctx context.Context, id string) error {
	if uc.tracks != nil {
		if err := uc.tracks.PurgeByBroadcaster(ctx, id); err != nil {
			return fmt.Errorf("purge tracks: %w", err)
		}
	}
	return uc.repo.Delete(ctx, id)
}
