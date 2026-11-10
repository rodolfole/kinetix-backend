package agecategories

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

// AgeCategoryResponse is exposed as a DTO. These tests pin the JSON shape
// (field names, omitempty for optional age range). The List endpoint itself
// requires a real store/DB, so it is covered in integration tests instead.

func TestAgeCategoryResponse_JSON_WithAgeRange(t *testing.T) {
	id := uuid.New()
	min, max := 18, 39
	resp := AgeCategoryResponse{
		ID:           id,
		Name:         "Adults 18-39",
		MinAge:       &min,
		MaxAge:       &max,
		DisplayOrder: 2,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"id":"` + id.String() + `","name":"Adults 18-39","min_age":18,"max_age":39,"display_order":2}`
	if string(data) != want {
		t.Errorf("unexpected JSON:\n got: %s\nwant: %s", data, want)
	}
}

func TestAgeCategoryResponse_JSON_NoAgeRange(t *testing.T) {
	id := uuid.New()
	resp := AgeCategoryResponse{
		ID:           id,
		Name:         "Open",
		DisplayOrder: 1,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// min_age / max_age are *int with omitempty → must be absent from JSON.
	got := string(data)
	if contains(got, `"min_age"`) || contains(got, `"max_age"`) {
		t.Errorf("expected min_age and max_age to be omitted, got %s", got)
	}
	if !contains(got, `"name":"Open"`) {
		t.Errorf("missing name field in %s", got)
	}
}

func TestGenericResponse_JSONWrapsData(t *testing.T) {
	wrapper := GenericResponse{
		Message: "Age categories retrieved successfully",
		Data:    []string{"a", "b"},
	}
	data, err := json.Marshal(wrapper)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !contains(string(data), `"message":"Age categories retrieved successfully"`) {
		t.Errorf("missing message field: %s", data)
	}
	if !contains(string(data), `"data":["a","b"]`) {
		t.Errorf("missing data array: %s", data)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
