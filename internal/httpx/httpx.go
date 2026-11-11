// Package httpx is the shared HTTP envelope layer (PLAN-16 T12): one place
// for decode, parameter parsing, validation and error mapping per
// docs/API_CONTRACTS.md.
package httpx

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"kinetix-api/internal/json"
)

// Sentinel errors handlers map once (PATTERNS.md). Services wrap their
// domain errors with these via errors.Is.
var (
	ErrNotFound       = errors.New("not found")
	ErrConflict       = errors.New("conflict")
	ErrForbidden      = errors.New("forbidden")
	ErrUnathenticated = errors.New("unauthenticated")
	ErrInvalidInput   = errors.New("invalid input")
	ErrUpstream       = errors.New("upstream error")
)

type codeAndStatus struct {
	code   string
	status int
}

func mapError(err error) codeAndStatus {
	switch {
	case errors.Is(err, ErrNotFound):
		return codeAndStatus{"NOT_FOUND", http.StatusNotFound}
	case errors.Is(err, ErrConflict):
		return codeAndStatus{"CONFLICT", http.StatusConflict}
	case errors.Is(err, ErrForbidden):
		return codeAndStatus{"FORBIDDEN", http.StatusForbidden}
	case errors.Is(err, ErrUnathenticated):
		return codeAndStatus{"UNAUTHORIZED", http.StatusUnauthorized}
	case errors.Is(err, ErrInvalidInput):
		return codeAndStatus{"VALIDATION_FAILED", http.StatusBadRequest}
	case errors.Is(err, ErrUpstream):
		return codeAndStatus{"UPSTREAM_ERROR", http.StatusBadGateway}
	default:
		return codeAndStatus{"INTERNAL", http.StatusInternalServerError}
	}
}

// WriteError maps a service error to the documented envelope. Unknown
// errors become 500 INTERNAL and are logged with the request id; their
// message never reaches the client.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	m := mapError(err)
	if m.code == "INTERNAL" {
		slog.ErrorContext(r.Context(), "handler error", slog.Any("err", err), slog.String("request_id", chiGetRequestID(r)))
		json.WriteError(w, m.status, "internal error")
		return
	}
	WriteErrorCode(w, m.status, m.code, err.Error())
}

// WriteErrorCode writes a specific error envelope with a machine code.
func WriteErrorCode(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.Write(w, status, map[string]any{"error": message, "code": code})
}

// Read decodes the JSON body; on failure it writes the 400 envelope and
// returns false so the handler just returns.
func Read(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.Read(r, dst); err != nil {
		WriteErrorCode(w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid request body")
		return false
	}
	return true
}

// UUIDParam parses a path parameter; on failure it writes the 400 INVALID_ID
// envelope and returns false.
func UUIDParam(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		WriteErrorCode(w, http.StatusBadRequest, "INVALID_ID", "invalid "+strings.ToLower(name))
		return uuid.Nil, false
	}
	return id, true
}

// WriteValidation renders validator.ValidationErrors as field messages.
func WriteValidation(w http.ResponseWriter, err error) {
	fields := map[string]string{}
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		for _, e := range ve {
			fields[strings.ToLower(e.Field())] = e.Tag()
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.Write(w, http.StatusBadRequest, map[string]any{
		"error":  "validation failed",
		"code":   "VALIDATION_FAILED",
		"fields": fields,
	})
}

// WriteData writes a single-resource envelope.
func WriteData(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.Write(w, status, map[string]any{"data": payload})
}

// WriteList writes a list envelope with an explicit total.
func WriteList(w http.ResponseWriter, items any, total int64) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.Write(w, http.StatusOK, map[string]any{"data": items, "total": total})
}

// WriteNoContent answers a delete.
func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// chiGetRequestID reads chi's RequestID middleware value (context key) via
// a decoupled hook so this package does not build on chi internals.
var chiGetRequestID = func(r *http.Request) string {
	return ""
}

// SetRequestIDResolver wires the request-id reader (called from cmd/api).
func SetRequestIDResolver(fn func(r *http.Request) string) {
	chiGetRequestID = fn
}
