package routeparser

import (
	"strings"
	"testing"
)

// TestParseGPX_ValidFile verifies a minimal valid GPX file is parsed correctly.
func TestParseGPX_ValidFile(t *testing.T) {
	gpxXML := `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1">
  <trk>
    <name>Test Track</name>
    <trkseg>
      <trkpt lat="40.7128" lon="-74.0060"><ele>10</ele></trkpt>
      <trkpt lat="40.7589" lon="-73.9851"><ele>20</ele></trkpt>
    </trkseg>
  </trk>
</gpx>`

	res, err := ParseGPX(strings.NewReader(gpxXML))
	if err != nil {
		t.Fatalf("ParseGPX failed: %v", err)
	}
	if res.Name != "Test Track" {
		t.Errorf("expected name 'Test Track', got %q", res.Name)
	}
	if len(res.Points) != 2 {
		t.Errorf("expected 2 points, got %d", len(res.Points))
	}
	if res.Points[0].Distance != 0 {
		t.Errorf("first point distance should be 0, got %f", res.Points[0].Distance)
	}
	if res.Points[1].Distance <= 0 {
		t.Errorf("second point distance should be > 0, got %f", res.Points[1].Distance)
	}
	if res.MinElevation != 10 || res.MaxElevation != 20 {
		t.Errorf("min/max elevation wrong: min=%f max=%f", res.MinElevation, res.MaxElevation)
	}
	if res.TotalElevationGain != 10 {
		t.Errorf("elevation gain should be 10 (20-10), got %f", res.TotalElevationGain)
	}
}

// TestParseGPX_NoTracks ensures we reject files without any tracks.
func TestParseGPX_NoTracks(t *testing.T) {
	gpxXML := `<?xml version="1.0"?><gpx></gpx>`
	_, err := ParseGPX(strings.NewReader(gpxXML))
	if err == nil {
		t.Error("expected error for GPX with no tracks, got nil")
	}
}

// TestParseGPX_NoPoints ensures we reject tracks with no points.
func TestParseGPX_NoPoints(t *testing.T) {
	gpxXML := `<?xml version="1.0"?>
<gpx><trk><name>Empty</name><trkseg></trkseg></trk></gpx>`
	_, err := ParseGPX(strings.NewReader(gpxXML))
	if err == nil {
		t.Error("expected error for track with no points, got nil")
	}
}

// TestParseGPX_InvalidXML ensures we reject malformed XML.
func TestParseGPX_InvalidXML(t *testing.T) {
	_, err := ParseGPX(strings.NewReader("<not gpx>"))
	if err == nil {
		t.Error("expected error for invalid XML, got nil")
	}
}

// TestHaversine_KnownDistance verifies the formula against a known city pair.
// NYC to LA is roughly 3935 km.
func TestHaversine_KnownDistance(t *testing.T) {
	nyLat, nyLon := 40.7128, -74.0060
	laLat, laLon := 34.0522, -118.2437

	d := haversine(nyLat, nyLon, laLat, laLon)
	// Accept 1% tolerance
	if d < 3895 || d > 3975 {
		t.Errorf("NYC->LA expected ~3935km, got %f", d)
	}
}

// TestHaversine_SamePoint returns 0 for identical coords.
func TestHaversine_SamePoint(t *testing.T) {
	d := haversine(40.0, -74.0, 40.0, -74.0)
	if d != 0 {
		t.Errorf("haversine(same, same) should be 0, got %f", d)
	}
}

// TestGeoJSONLineString_Empty verifies empty points return valid empty LineString.
func TestGeoJSONLineString_Empty(t *testing.T) {
	res := &ParseResult{}
	out := string(res.GeoJSONLineString())
	expected := `{"type":"LineString","coordinates":[]}`
	if out != expected {
		t.Errorf("expected %q, got %q", expected, out)
	}
}

// TestGeoJSONLineString_OrderIsLonLat verifies coordinates are [lon, lat] per GeoJSON spec.
func TestGeoJSONLineString_OrderIsLonLat(t *testing.T) {
	res := &ParseResult{
		Points: []Point{
			{Lat: 10.0, Lon: 20.0},
			{Lat: 30.0, Lon: 40.0},
		},
	}
	out := string(res.GeoJSONLineString())
	// lon, lat order
	if !strings.Contains(out, "[20.0000000,10.0000000]") {
		t.Errorf("expected first coord [lon,lat], got: %s", out)
	}
	if !strings.Contains(out, "[40.0000000,30.0000000]") {
		t.Errorf("expected second coord [lon,lat], got: %s", out)
	}
}

// TestElevationProfile_Empty verifies empty points return valid empty array.
func TestElevationProfile_Empty(t *testing.T) {
	res := &ParseResult{}
	out := string(res.ElevationProfile())
	if out != "[]" {
		t.Errorf("expected empty array, got %s", out)
	}
}

// TestParseGPX_MultipleSegments verifies all segments are concatenated.
func TestParseGPX_MultipleSegments(t *testing.T) {
	gpxXML := `<?xml version="1.0"?>
<gpx>
  <trk>
    <name>Multi</name>
    <trkseg><trkpt lat="1" lon="1"><ele>0</ele></trkpt></trkseg>
    <trkseg><trkpt lat="2" lon="2"><ele>0</ele></trkpt></trkseg>
  </trk>
</gpx>`

	res, err := ParseGPX(strings.NewReader(gpxXML))
	if err != nil {
		t.Fatalf("ParseGPX failed: %v", err)
	}
	if len(res.Points) != 2 {
		t.Errorf("expected 2 points across segments, got %d", len(res.Points))
	}
}
