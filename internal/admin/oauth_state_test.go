package admin

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestOAuthStateRoundTrip(t *testing.T) {
	id := uuid.New()
	state := SignOAuthState("secret", id)
	got, err := VerifyOAuthState("secret", state)
	if err != nil || got != id {
		t.Fatalf("got %v err=%v, want %s", got, err, id)
	}
}

func TestOAuthStateRejectsTampering(t *testing.T) {
	id := uuid.New()
	state := SignOAuthState("secret", id)

	if _, err := VerifyOAuthState("other-secret", state); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("different secret must fail: %v", err)
	}
	if _, err := VerifyOAuthState("secret", "no-sig"); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("unsigned state must fail: %v", err)
	}
	state2 := SignOAuthState("secret", uuid.New())
	if _, err := VerifyOAuthState("secret", "x"+state2[len(state2)-40:]); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("mangled state must fail: %v", err)
	}
	// raw UUID state from before the fix must be rejected
	if _, err := VerifyOAuthState("secret", id.String()); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("raw organizer UUID must be rejected: %v", err)
	}
}
