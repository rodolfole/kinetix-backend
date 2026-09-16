package routeparser

import (
	"encoding/xml"
	"fmt"
	"io"
	"math"
)

// GPX structs for XML parsing
type gpx struct {
	XMLName xml.Name `xml:"gpx"`
	Tracks  []trk    `xml:"trk"`
}

type trk struct {
	Name     string   `xml:"name"`
	Segments []trkseg `xml:"trkseg"`
}

type trkseg struct {
	Points []trkpt `xml:"trkpt"`
}

type trkpt struct {
	Lat float64 `xml:"lat,attr"`
	Lon float64 `xml:"lon,attr"`
	Ele float64 `xml:"ele"`
}

// Point represents a single waypoint with elevation
type Point struct {
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	Elevation float64 `json:"elevation"`
	Distance  float64 `json:"distance"` // cumulative distance in km
}

// ParseResult contains all parsed route data
type ParseResult struct {
	Name             string           `json:"name"`
	Points           []Point          `json:"points"`
	TotalDistanceKm  float64          `json:"total_distance_km"`
	TotalElevationGain float64        `json:"total_elevation_gain"`
	MinElevation     float64          `json:"min_elevation"`
	MaxElevation     float64          `json:"max_elevation"`
}

// GeoJSONLineString generates a GeoJSON LineString from parsed points
func (pr *ParseResult) GeoJSONLineString() []byte {
	if len(pr.Points) == 0 {
		return []byte(`{"type":"LineString","coordinates":[]}`)
	}

	coords := "["
	for i, p := range pr.Points {
		if i > 0 {
			coords += ","
		}
		coords += fmt.Sprintf("[%.7f,%.7f]", p.Lon, p.Lat)
	}
	coords += "]"

	geojson := fmt.Sprintf(`{"type":"LineString","coordinates":%s}`, coords)
	return []byte(geojson)
}

// ElevationProfile generates elevation profile as array of [distance, elevation] pairs
func (pr *ParseResult) ElevationProfile() []byte {
	if len(pr.Points) == 0 {
		return []byte(`[]`)
	}

	profile := "["
	for i, p := range pr.Points {
		if i > 0 {
			profile += ","
		}
		profile += fmt.Sprintf("[%.3f,%.1f]", p.Distance, p.Elevation)
	}
	profile += "]"

	return []byte(profile)
}

// ParseGPX parses a GPX file from reader and returns route data
func ParseGPX(r io.Reader) (*ParseResult, error) {
	var gpxData gpx
	if err := xml.NewDecoder(r).Decode(&gpxData); err != nil {
		return nil, fmt.Errorf("failed to parse GPX: %w", err)
	}

	if len(gpxData.Tracks) == 0 {
		return nil, fmt.Errorf("GPX file contains no tracks")
	}

	track := gpxData.Tracks[0]
	result := &ParseResult{
		Name: track.Name,
	}

	// Collect all points from all segments
	var allPoints []trkpt
	for _, seg := range track.Segments {
		allPoints = append(allPoints, seg.Points...)
	}

	if len(allPoints) == 0 {
		return nil, fmt.Errorf("GPX track contains no points")
	}

	// Convert to Points with cumulative distance
	var prevLat, prevLon float64
	result.MinElevation = math.MaxFloat64

	for i, pt := range allPoints {
		p := Point{
			Lat:       pt.Lat,
			Lon:       pt.Lon,
			Elevation: pt.Ele,
		}

		if i == 0 {
			p.Distance = 0
		} else {
			// Haversine distance in km
			d := haversine(prevLat, prevLon, pt.Lat, pt.Lon)
			p.Distance = result.Points[i-1].Distance + d
		}

		// Elevation gain
		if i > 0 && pt.Ele > allPoints[i-1].Ele {
			result.TotalElevationGain += pt.Ele - allPoints[i-1].Ele
		}

		// Min/Max elevation
		if pt.Ele < result.MinElevation {
			result.MinElevation = pt.Ele
		}
		if pt.Ele > result.MaxElevation {
			result.MaxElevation = pt.Ele
		}

		result.Points = append(result.Points, p)
		prevLat, prevLon = pt.Lat, pt.Lon
	}

	if len(result.Points) > 0 {
		result.TotalDistanceKm = result.Points[len(result.Points)-1].Distance
	}

	return result, nil
}

// haversine calculates distance in km between two lat/lon points
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0 // Earth radius in km

	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}
