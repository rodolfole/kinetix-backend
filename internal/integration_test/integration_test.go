package integration_test

import (
	"net/http"
	"testing"
	"time"

	"kinetix-api/internal/testhelpers"
)

// These tests assume the docker-compose stack is running (api on :8085, postgres on :5432)
// and that the seed binary has populated test data:
//   organizers: contact@kinetix.com / seed1234
//   participants: juan.perez@example.com / seed1234 (and others)
//   events: carrera-kinetix-2026, 10k-valencia-marathon, neon-dash-10k, summit-peak-trail

func TestHealthEndpoint(t *testing.T) {
	var resp map[string]any
	status, _ := testhelpers.JSONRequest(t, http.MethodGet, "/health", nil, nil, &resp)
	if status != http.StatusOK {
		t.Fatalf("health: want 200, got %d", status)
	}
	if resp["status"] != "ok" {
		t.Errorf("health.status: want ok, got %v", resp["status"])
	}
}

func TestWelcomeEndpoint(t *testing.T) {
	var resp map[string]any
	status, _ := testhelpers.JSONRequest(t, http.MethodGet, "/", nil, nil, &resp)
	if status != http.StatusOK {
		t.Fatalf("/: want 200, got %d", status)
	}
	if msg, ok := resp["message"].(string); !ok || msg == "" {
		t.Errorf("/: expected non-empty message, got %v", resp["message"])
	}
}

func TestListEvents_Public(t *testing.T) {
	var resp struct {
		Message string `json:"message"`
		Data    []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Slug string `json:"slug"`
		} `json:"data"`
		Total int `json:"total"`
	}
	status, _ := testhelpers.JSONRequest(t, http.MethodGet, "/api/v1/events", nil, nil, &resp)
	if status != http.StatusOK {
		t.Fatalf("list events: want 200, got %d", status)
	}
	if resp.Total < 1 {
		t.Fatalf("expected at least one event, got %d", resp.Total)
	}
	if len(resp.Data) != resp.Total {
		t.Errorf("len(data)=%d != total=%d", len(resp.Data), resp.Total)
	}
}

func TestListEvents_SlugLookup(t *testing.T) {
	var resp struct {
		Data struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Slug string `json:"slug"`
		} `json:"data"`
	}
	status, _ := testhelpers.JSONRequest(t, http.MethodGet, "/api/v1/events/slug/carrera-kinetix-2026", nil, nil, &resp)
	if status != http.StatusOK {
		t.Fatalf("get by slug: want 200, got %d", status)
	}
	if resp.Data.Slug != "carrera-kinetix-2026" {
		t.Errorf("want slug carrera-kinetix-2026, got %q", resp.Data.Slug)
	}
}

func TestListEvents_SlugNotFound(t *testing.T) {
	var resp map[string]any
	status, _ := testhelpers.JSONRequest(t, http.MethodGet, "/api/v1/events/slug/does-not-exist", nil, nil, &resp)
	if status != http.StatusNotFound {
		t.Errorf("expected 404 for missing slug, got %d", status)
	}
}

func TestLogin_Organizer(t *testing.T) {
	token := testhelpers.Login(t, "contact@kinetix.com", "seed1234")
	if token == "" {
		t.Fatal("expected token for organizer login")
	}
}

func TestLogin_Participant(t *testing.T) {
	token := testhelpers.Login(t, "juan.perez@example.com", "seed1234")
	if token == "" {
		t.Fatal("expected token for participant login")
	}
}

func TestLogin_BadCredentials(t *testing.T) {
	body := map[string]string{"email": "contact@kinetix.com", "password": "wrong-pass"}
	var resp map[string]any
	status, _ := testhelpers.JSONRequest(t, http.MethodPost, "/api/v1/auth/login", body, nil, &resp)
	if status != http.StatusUnauthorized {
		t.Errorf("bad creds: want 401, got %d (body=%v)", status, resp)
	}
}

func TestLogin_InvalidPayload(t *testing.T) {
	body := map[string]string{"email": "no-at", "password": ""}
	var resp map[string]any
	status, _ := testhelpers.JSONRequest(t, http.MethodPost, "/api/v1/auth/login", body, nil, &resp)
	if status != http.StatusBadRequest {
		t.Errorf("invalid payload: want 400, got %d", status)
	}
}

func TestRefreshToken_Organizer(t *testing.T) {
	body := map[string]string{"email": "contact@kinetix.com", "password": "seed1234"}
	var login struct {
		RefreshToken string `json:"refresh_token"`
	}
	status, _ := testhelpers.JSONRequest(t, http.MethodPost, "/api/v1/auth/login", body, nil, &login)
	if status != http.StatusOK {
		t.Fatalf("login: want 200, got %d", status)
	}
	if login.RefreshToken == "" {
		t.Fatal("expected refresh_token in login response")
	}

	var refreshed struct {
		Message string `json:"message"`
		Data    struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int64  `json:"expires_in"`
		} `json:"data"`
	}
	status, _ = testhelpers.JSONRequest(t, http.MethodPost, "/api/v1/auth/refresh",
		map[string]string{"refresh_token": login.RefreshToken}, nil, &refreshed)
	if status != http.StatusOK {
		t.Fatalf("refresh: want 200, got %d", status)
	}
	if refreshed.Data.AccessToken == "" {
		t.Error("expected access_token in refresh response")
	}
	if refreshed.Data.ExpiresIn <= 0 {
		t.Errorf("expected positive expires_in, got %d", refreshed.Data.ExpiresIn)
	}
}

func TestProtectedRoute_RequiresAuth(t *testing.T) {
	var resp map[string]any
	status, _ := testhelpers.JSONRequest(t, http.MethodGet, "/api/v1/organizers", nil, nil, &resp)
	if status != http.StatusUnauthorized {
		t.Errorf("protected route without token: want 401, got %d", status)
	}
}

func TestProtectedRoute_WithOrganizerToken(t *testing.T) {
	token := testhelpers.Login(t, "contact@kinetix.com", "seed1234")
	var resp struct {
		Data []struct {
			ID          string `json:"id"`
			BusinessName string `json:"business_name"`
		} `json:"data"`
	}
	status := testhelpers.GetJSON(t, "/api/v1/organizers", token, &resp)
	if status != http.StatusOK {
		t.Fatalf("list organizers: want 200, got %d", status)
	}
	if len(resp.Data) == 0 {
		t.Error("expected at least one organizer for seeded user")
	}
}

func TestEventDetail_OrganizerCanFetch(t *testing.T) {
	token := testhelpers.Login(t, "contact@kinetix.com", "seed1234")

	// Find an event id from the public list
	var list struct {
		Data []struct {
			ID   string `json:"id"`
			Slug string `json:"slug"`
		} `json:"data"`
	}
	status, _ := testhelpers.JSONRequest(t, http.MethodGet, "/api/v1/events", nil, nil, &list)
	if status != http.StatusOK {
		t.Fatalf("list events: %d", status)
	}
	if len(list.Data) == 0 {
		t.Fatal("expected at least one seeded event")
	}
	eventID := list.Data[0].ID

	var detail struct {
		Data struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	status = testhelpers.GetJSON(t, "/api/v1/events/"+eventID, token, &detail)
	if status != http.StatusOK {
		t.Fatalf("event detail: want 200, got %d", status)
	}
	if detail.Data.ID != eventID {
		t.Errorf("event detail id mismatch: want %q got %q", eventID, detail.Data.ID)
	}
}

func TestListDistanceTypes_Public(t *testing.T) {
	var resp struct {
		Data []struct {
			Name string  `json:"name"`
			Km   float64 `json:"km"`
		} `json:"data"`
	}
	status, _ := testhelpers.JSONRequest(t, http.MethodGet, "/api/v1/distances/types", nil, nil, &resp)
	if status != http.StatusOK {
		t.Fatalf("distance types: want 200, got %d", status)
	}
	if len(resp.Data) == 0 {
		t.Error("expected distance types from seed")
	}
}

func TestListAgeCategories_Public(t *testing.T) {
	var resp struct {
		Data []struct {
			Name      string `json:"name"`
			MinAge    int    `json:"min_age"`
			MaxAge    int    `json:"max_age"`
		} `json:"data"`
	}
	status, _ := testhelpers.JSONRequest(t, http.MethodGet, "/api/v1/age-categories", nil, nil, &resp)
	if status != http.StatusOK {
		t.Fatalf("age categories: want 200, got %d", status)
	}
}

// seedEventID is the slug for the seeded event that has distances, pricing and runner kits.
const seedEventSlug = "carrera-kinetix-2026"

// seedEventIDLookup fetches the UUID of the seeded event with distances.
func seedEventID(t *testing.T) string {
	t.Helper()
	var resp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	status, _ := testhelpers.JSONRequest(t, http.MethodGet, "/api/v1/events/slug/"+seedEventSlug, nil, nil, &resp)
	if status != http.StatusOK {
		t.Fatalf("fetch seeded event by slug: %d", status)
	}
	if resp.Data.ID == "" {
		t.Fatal("seeded event not found")
	}
	return resp.Data.ID
}

func TestEventDistances_Public(t *testing.T) {
	eventID := seedEventID(t)
	var distances struct {
		Data []struct {
			ID       string  `json:"id"`
			Km       float64 `json:"km"`
			Capacity int     `json:"capacity"`
		} `json:"data"`
	}
	status, _ := testhelpers.JSONRequest(t, http.MethodGet, "/api/v1/events/event/"+eventID+"/distances", nil, nil, &distances)
	if status != http.StatusOK {
		t.Fatalf("event distances: want 200, got %d", status)
	}
	if len(distances.Data) == 0 {
		t.Error("expected at least one distance for the seeded event")
	}
}

func TestPricingStages_Public(t *testing.T) {
	eventID := seedEventID(t)
	var distances struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	status, _ := testhelpers.JSONRequest(t, http.MethodGet, "/api/v1/events/event/"+eventID+"/distances", nil, nil, &distances)
	if status != http.StatusOK || len(distances.Data) == 0 {
		t.Fatal("no distances")
	}
	var stages struct {
		Data []struct {
			Name       string  `json:"name"`
			Price      float64 `json:"price"`
			IsEarlyBird bool   `json:"is_early_bird"`
		} `json:"data"`
	}
	status, _ = testhelpers.JSONRequest(t, http.MethodGet, "/api/v1/events/distance/"+distances.Data[0].ID+"/pricing", nil, nil, &stages)
	if status != http.StatusOK {
		t.Fatalf("pricing stages: want 200, got %d", status)
	}
	if len(stages.Data) == 0 {
		t.Error("expected pricing stages for seeded distance")
	}
}

func TestCurrentPricing_Public(t *testing.T) {
	eventID := seedEventID(t)
	var distances struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	status, _ := testhelpers.JSONRequest(t, http.MethodGet, "/api/v1/events/event/"+eventID+"/distances", nil, nil, &distances)
	if status != http.StatusOK || len(distances.Data) == 0 {
		t.Fatal("no distances")
	}
	status, _ = testhelpers.JSONRequest(t, http.MethodGet, "/api/v1/events/distance/"+distances.Data[0].ID+"/pricing/current", nil, nil, nil)
	// 200 if there's a current stage, 404 otherwise — both are valid for a given distance
	if status != http.StatusOK && status != http.StatusNotFound {
		t.Errorf("current pricing: want 200 or 404, got %d", status)
	}
}

func TestRunnerKits_Public(t *testing.T) {
	eventID := seedEventID(t)
	var kits struct {
		Data []struct {
			Name string `json:"name"`
			Icon string `json:"icon"`
		} `json:"data"`
	}
	status, _ := testhelpers.JSONRequest(t, http.MethodGet, "/api/v1/events/event/"+eventID+"/runner-kits", nil, nil, &kits)
	if status != http.StatusOK {
		t.Fatalf("runner kits: want 200, got %d", status)
	}
}

func TestOrganizers_ProtectedList(t *testing.T) {
	token := testhelpers.Login(t, "contact@kinetix.com", "seed1234")
	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	status := testhelpers.GetJSON(t, "/api/v1/organizers", token, &resp)
	if status != http.StatusOK {
		t.Fatalf("list organizers: want 200, got %d", status)
	}
}

func TestEvents_CreateRequiresAuth(t *testing.T) {
	body := map[string]any{"name": "X"}
	var resp map[string]any
	status, _ := testhelpers.JSONRequest(t, http.MethodPost, "/api/v1/events", body, nil, &resp)
	if status != http.StatusUnauthorized {
		t.Errorf("create event without auth: want 401, got %d", status)
	}
}

func TestEvents_CreateWithOrganizerToken(t *testing.T) {
	token := testhelpers.Login(t, "contact@kinetix.com", "seed1234")

	var list struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	status, _ := testhelpers.JSONRequest(t, http.MethodGet, "/api/v1/organizers", nil, testhelpers.AuthHeader(token), &list)
	if status != http.StatusOK || len(list.Data) == 0 {
		t.Fatalf("no organizers found: %d", status)
	}
	organizerID := list.Data[0].ID

	// Use unique slug to avoid collisions
	slug := "test-event-" + time.Now().Format("20060102150405")
	body := map[string]any{
		"organizer_id": organizerID,
		"name":         "Test Event",
		"slug":         slug,
		"description":  "Created by integration tests",
		"event_date":   time.Now().AddDate(0, 3, 0).Format(time.RFC3339),
		"deadline":     time.Now().AddDate(0, 2, 0).Format(time.RFC3339),
		"state":        "Test State",
		"municipality": "Test City",
		"address":      "123 Test Street",
		"itinerary": []map[string]string{
			{"key": "Salida", "time": "07:00", "icon": "flag"},
		},
	}
	var resp struct {
		Data struct {
			ID   string `json:"id"`
			Slug string `json:"slug"`
		} `json:"data"`
	}
	status = testhelpers.PostJSON(t, "/api/v1/events", token, body, &resp)
	if status != http.StatusCreated && status != http.StatusOK {
		t.Fatalf("create event: want 201/200, got %d", status)
	}
	if resp.Data.ID == "" {
		t.Error("expected created event id")
	}
	if resp.Data.Slug != slug {
		t.Errorf("created slug mismatch: want %q, got %q", slug, resp.Data.Slug)
	}
}

func TestEvents_CreateInvalidPayload(t *testing.T) {
	token := testhelpers.Login(t, "contact@kinetix.com", "seed1234")
	body := map[string]any{"name": ""} // missing required fields
	var resp map[string]any
	status := testhelpers.PostJSON(t, "/api/v1/events", token, body, &resp)
	if status != http.StatusBadRequest {
		t.Errorf("create event with invalid payload: want 400, got %d", status)
	}
}

func TestPublicOrganizerDetail(t *testing.T) {
	// First list organizers with auth
	token := testhelpers.Login(t, "contact@kinetix.com", "seed1234")
	var list struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	status, _ := testhelpers.JSONRequest(t, http.MethodGet, "/api/v1/organizers", nil, testhelpers.AuthHeader(token), &list)
	if status != http.StatusOK || len(list.Data) == 0 {
		t.Fatal("no organizers")
	}
	// Now fetch that organizer publicly
	var resp struct {
		Data struct {
			ID          string `json:"id"`
			BusinessName string `json:"business_name"`
		} `json:"data"`
	}
	status, _ = testhelpers.JSONRequest(t, http.MethodGet, "/api/v1/organizers/"+list.Data[0].ID, nil, nil, &resp)
	if status != http.StatusOK {
		t.Errorf("public organizer detail: want 200, got %d", status)
	}
}

func TestEventRoutes_Public(t *testing.T) {
	var list struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	status, _ := testhelpers.JSONRequest(t, http.MethodGet, "/api/v1/events", nil, nil, &list)
	if status != http.StatusOK || len(list.Data) == 0 {
		t.Fatal("no events")
	}
	var routes struct {
		Data []any `json:"data"`
	}
	status, _ = testhelpers.JSONRequest(t, http.MethodGet, "/api/v1/events/event/"+list.Data[0].ID+"/routes", nil, nil, &routes)
	if status != http.StatusOK && status != http.StatusNotFound {
		t.Errorf("event routes: want 200 or 404, got %d", status)
	}
}

func TestEventSponsors_Public(t *testing.T) {
	var list struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	status, _ := testhelpers.JSONRequest(t, http.MethodGet, "/api/v1/events", nil, nil, &list)
	if status != http.StatusOK || len(list.Data) == 0 {
		t.Fatal("no events")
	}
	var sponsors struct {
		Data []any `json:"data"`
	}
	status, _ = testhelpers.JSONRequest(t, http.MethodGet, "/api/v1/events/event/"+list.Data[0].ID+"/sponsors", nil, nil, &sponsors)
	if status != http.StatusOK && status != http.StatusNotFound {
		t.Errorf("event sponsors: want 200 or 404, got %d", status)
	}
}

func TestEventLocations_Public(t *testing.T) {
	var list struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	status, _ := testhelpers.JSONRequest(t, http.MethodGet, "/api/v1/events", nil, nil, &list)
	if status != http.StatusOK || len(list.Data) == 0 {
		t.Fatal("no events")
	}
	var locations struct {
		Data []any `json:"data"`
	}
	status, _ = testhelpers.JSONRequest(t, http.MethodGet, "/api/v1/events/event/"+list.Data[0].ID+"/locations", nil, nil, &locations)
	if status != http.StatusOK && status != http.StatusNotFound {
		t.Errorf("event locations: want 200 or 404, got %d", status)
	}
}

func TestEventFieldConfig_Public(t *testing.T) {
	// /events/{id}/field-config has both a public and protected registration in main.go;
	// the protected (later-registered) route wins, so without auth we expect 401.
	var list struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	status, _ := testhelpers.JSONRequest(t, http.MethodGet, "/api/v1/events", nil, nil, &list)
	if status != http.StatusOK || len(list.Data) == 0 {
		t.Fatal("no events")
	}
	status, _ = testhelpers.JSONRequest(t, http.MethodGet, "/api/v1/events/"+list.Data[0].ID+"/field-config", nil, nil, nil)
	if status != http.StatusUnauthorized {
		t.Errorf("event field config: want 401 (protected), got %d", status)
	}

	// With auth, it should return 200 or 404
	token := testhelpers.Login(t, "contact@kinetix.com", "seed1234")
	status, _ = testhelpers.JSONRequest(t, http.MethodGet, "/api/v1/events/"+list.Data[0].ID+"/field-config", nil, testhelpers.AuthHeader(token), nil)
	if status != http.StatusOK && status != http.StatusNotFound {
		t.Errorf("event field config with auth: want 200 or 404, got %d", status)
	}
}