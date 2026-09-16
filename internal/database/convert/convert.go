package convert

import (
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// ============ pgtype.Text conversions ============

// ToPgText converts string to pgtype.Text
func ToPgText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

// ToPgTextPtr converts *string to pgtype.Text
func ToPgTextPtr(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// PgTextToString converts pgtype.Text to string
func PgTextToString(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

// PgTextToStringPtr converts pgtype.Text to *string
func PgTextToStringPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

// ============ pgtype.Numeric conversions ============

// ToPgNumeric converts float64 to pgtype.Numeric
func ToPgNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	if err := n.Scan(strconv.FormatFloat(f, 'f', -1, 64)); err != nil {
		return pgtype.Numeric{}
	}
	return n
}

// ToPgNumericPtr converts *float64 to pgtype.Numeric
func ToPgNumericPtr(f *float64) pgtype.Numeric {
	if f == nil {
		return pgtype.Numeric{}
	}
	return ToPgNumeric(*f)
}

// PgNumericToFloat64 converts pgtype.Numeric to float64
func PgNumericToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f8, err := n.Float64Value()
	if err != nil {
		return 0
	}
	return f8.Float64
}

// PgNumericToFloat64Ptr converts pgtype.Numeric to *float64
func PgNumericToFloat64Ptr(n pgtype.Numeric) *float64 {
	if !n.Valid {
		return nil
	}
	f8, err := n.Float64Value()
	if err != nil {
		return nil
	}
	return &f8.Float64
}

// ============ pgtype.Timestamptz conversions ============

// ToPgTime converts time.Time to pgtype.Timestamptz
func ToPgTime(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// ToPgTimePtr converts *time.Time to pgtype.Timestamptz
func ToPgTimePtr(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

// PgTimeToTime converts pgtype.Timestamptz to time.Time
func PgTimeToTime(pt pgtype.Timestamptz) time.Time {
	if !pt.Valid {
		return time.Time{}
	}
	return pt.Time
}

// PgTimeToTimePtr converts pgtype.Timestamptz to *time.Time
func PgTimeToTimePtr(pt pgtype.Timestamptz) *time.Time {
	if !pt.Valid {
		return nil
	}
	t := pt.Time
	return &t
}

// ============ JSONB ([]byte) conversions ============

// ToPgJSONB converts a JSON string to []byte (JSONB)
func ToPgJSONB(jsonStr string) []byte {
	if jsonStr == "" || jsonStr == "null" {
		return nil
	}
	return []byte(jsonStr)
}

// ToPgJSONBFromBytes converts []byte to []byte (passthrough for JSONB)
func ToPgJSONBFromBytes(b []byte) []byte {
	if len(b) == 0 {
		return nil
	}
	return b
}

// PgJSONBToBytes converts []byte (JSONB) to []byte (passthrough)
func PgJSONBToBytes(j []byte) []byte {
	if len(j) == 0 {
		return nil
	}
	return j
}

// PgJSONBToString converts []byte (JSONB) to string
func PgJSONBToString(j []byte) string {
	if len(j) == 0 {
		return "[]"
	}
	return string(j)
}

// ============ pgtype.Int4 conversions ============

// PgInt4ToIntPtr converts pgtype.Int4 to *int
func PgInt4ToIntPtr(i pgtype.Int4) *int {
	if !i.Valid {
		return nil
	}
	v := int(i.Int32)
	return &v
}

// ToPgInt4 converts int to pgtype.Int4
func ToPgInt4(i int) pgtype.Int4 {
	return pgtype.Int4{Int32: int32(i), Valid: true}
}

// ToPgInt4Ptr converts *int to pgtype.Int4
func ToPgInt4Ptr(i *int) pgtype.Int4 {
	if i == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(*i), Valid: true}
}

// ============ Generic pointer helpers ============

// ToStringPtr converts string to *string
func ToStringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ToTimePtr converts time.Time to *time.Time
func ToTimePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// ============ UUID conversions ============

// UUIDToPtr converts uuid.UUID to *uuid.UUID
func UUIDToPtr(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}

// UUIDToNil converts *uuid.UUID to uuid.UUID (for SQL)
func UUIDToNil(id *uuid.UUID) uuid.UUID {
	if id == nil {
		return uuid.Nil
	}
	return *id
}

// PtrToUUID converts *uuid.UUID to uuid.UUID
func PtrToUUID(id *uuid.UUID) uuid.UUID {
	if id == nil {
		return uuid.Nil
	}
	return *id
}

// ============ UUID conversions for PostgreSQL ============

// PgUUIDToPtr converts pgtype.UUID to *uuid.UUID
func PgUUIDToPtr(id pgtype.UUID) *uuid.UUID {
	if !id.Valid {
		return nil
	}
	// pgtype.UUID stores bytes in a scanf-friendly format - copy to uuid.UUID
	var u uuid.UUID
	copy(u[:], id.Bytes[:])
	return &u
}

// PtrToPgUUID converts *uuid.UUID to pgtype.UUID (for SQL)
func PtrToPgUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

// ToPgUUID converts uuid.UUID to pgtype.UUID (for SQL)
func ToPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}
