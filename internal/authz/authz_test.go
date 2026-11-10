package authz

import (
	"testing"

	"github.com/google/uuid"

	"kinetix-api/internal/auth"
)

func TestCanManageEvent(t *testing.T) {
	owner := uuid.New()
	other := uuid.New()

	tests := []struct {
		name    string
		claims  *auth.Claims
		ownerID uuid.UUID
		wantErr bool
	}{
		{"admin allowed", &auth.Claims{UserID: "a", Role: "admin"}, owner, false},
		{"owner allowed", &auth.Claims{UserID: "o", Role: "organizer", OrganizerID: &owner}, owner, false},
		{"other organizer forbidden", &auth.Claims{UserID: "o2", Role: "organizer", OrganizerID: &other}, owner, true},
		{"organizer without profile forbidden", &auth.Claims{UserID: "o3", Role: "organizer"}, owner, true},
		{"participant forbidden", &auth.Claims{UserID: "p", Role: "participant", ParticipantID: &other}, owner, true},
		{"no claims forbidden", nil, owner, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CanManageEvent(tt.claims, tt.ownerID); (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
