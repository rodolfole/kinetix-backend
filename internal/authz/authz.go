// Package authz enforces resource ownership from JWT claims. It exists so
// slices share one rule without importing each other.
package authz

import (
	"errors"

	"github.com/google/uuid"

	"kinetix-api/internal/auth"
)

// ErrForbidden is returned when the claims do not allow the operation.
var ErrForbidden = errors.New("forbidden")

// CanManageEvent reports whether the caller may mutate resources of the
// event owned by eventOrganizerID. Admins pass; organizers must own it.
// nil claims (unauthenticated) are rejected.
func CanManageEvent(claims *auth.Claims, eventOrganizerID uuid.UUID) error {
	if claims == nil {
		return ErrForbidden
	}
	if claims.Role == "admin" {
		return nil
	}
	if claims.Role == "organizer" && claims.OrganizerID != nil && *claims.OrganizerID == eventOrganizerID {
		return nil
	}
	return ErrForbidden
}
