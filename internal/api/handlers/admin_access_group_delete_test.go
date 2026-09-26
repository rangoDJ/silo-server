package handlers

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Silo-Server/silo-server/internal/access"
)

// memberMovingTestStore is a guarded group store whose delete moves members,
// recording what the handler asks it to do.
type memberMovingTestStore struct {
	*accessGroupHandlerTestStore
	moved        []int
	gotCallback  bool
	deleteErr    error
	plainDeletes int
}

func (s *memberMovingTestStore) ListPage(context.Context, *access.GroupPageKey, int) ([]access.Group, bool, error) {
	return nil, false, nil
}

func (s *memberMovingTestStore) UpdateConditional(context.Context, int64, access.UpdateGroupInput, access.GroupPrecondition) (*access.Group, error) {
	return nil, errors.New("not used")
}

func (s *memberMovingTestStore) DeleteConditional(context.Context, int64, access.GroupPrecondition) error {
	s.plainDeletes++
	return nil
}

// DeleteMovingMembers records that the handler supplied the in-transaction
// revocation callback; running it needs a real transaction (see the access
// package's DB test and auth.RevokeSignInsInTransaction).
func (s *memberMovingTestStore) DeleteMovingMembers(_ context.Context, _ int64, _ access.GroupPrecondition, onMoved func(context.Context, pgx.Tx, []int) error) ([]int, error) {
	s.gotCallback = onMoved != nil
	if s.deleteErr != nil {
		return nil, s.deleteErr
	}
	return s.moved, nil
}

func TestDeleteAdminAccessGroupMovesMembersThroughTheStore(t *testing.T) {
	store := &memberMovingTestStore{accessGroupHandlerTestStore: newAccessGroupHandlerTestStore()}
	handler := NewAccessGroupHandler(store)
	var notified []int
	handler.OnUserSessionsRevoked = func(_ context.Context, userID int) { notified = append(notified, userID) }

	if err := handler.DeleteAdminAccessGroup(t.Context(), 5, access.GroupPrecondition{Any: true}); err != nil {
		t.Fatalf("DeleteAdminAccessGroup() error: %v", err)
	}
	if store.plainDeletes != 0 {
		t.Fatal("used the plain delete instead of moving members")
	}
	if len(notified) != 0 {
		t.Fatalf("notified %v for a group without members", notified)
	}
}

func TestDeleteAdminAccessGroupRevokesAndNotifiesMovedMembers(t *testing.T) {
	store := &memberMovingTestStore{
		accessGroupHandlerTestStore: newAccessGroupHandlerTestStore(),
		moved:                       []int{7, 9},
	}
	handler := NewAccessGroupHandler(store)
	var notified []int
	handler.OnUserSessionsRevoked = func(_ context.Context, userID int) { notified = append(notified, userID) }

	if err := handler.DeleteAdminAccessGroup(t.Context(), 5, access.GroupPrecondition{Any: true}); err != nil {
		t.Fatalf("DeleteAdminAccessGroup() error: %v", err)
	}
	if !store.gotCallback {
		t.Fatal("the handler supplied no in-transaction revocation callback")
	}
	if !slices.Equal(notified, []int{7, 9}) {
		t.Fatalf("notified %v, want [7 9]", notified)
	}
}

func TestDeleteAdminAccessGroupKeepsStoreErrors(t *testing.T) {
	store := &memberMovingTestStore{
		accessGroupHandlerTestStore: newAccessGroupHandlerTestStore(),
		deleteErr:                   access.ErrDefaultGroupRequired,
	}
	handler := NewAccessGroupHandler(store)
	notified := false
	handler.OnUserSessionsRevoked = func(context.Context, int) { notified = true }

	err := handler.DeleteAdminAccessGroup(t.Context(), 1, access.GroupPrecondition{Any: true})
	if !errors.Is(err, access.ErrDefaultGroupRequired) {
		t.Fatalf("error = %v, want ErrDefaultGroupRequired", err)
	}
	if notified {
		t.Fatal("notified sessions after a failed delete")
	}
}
