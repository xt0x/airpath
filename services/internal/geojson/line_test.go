package geojson

import "testing"

func TestLineFeatureSplitsAcrossAntimeridian(t *testing.T) {
	feature, ok := LineFeature("flightaware_route", "planned_route", []Point{
		{Longitude: 170, Latitude: 35},
		{Longitude: -170, Latitude: 40},
	})
	if !ok {
		t.Fatal("LineFeature() ok = false, want true")
	}

	geometry := feature["geometry"].(map[string]any)
	if geometry["type"] != "MultiLineString" {
		t.Fatalf("geometry.type = %v, want MultiLineString", geometry["type"])
	}
	coordinates := geometry["coordinates"].([][][]float64)
	if len(coordinates) != 2 {
		t.Fatalf("line segments = %d, want 2", len(coordinates))
	}
}
