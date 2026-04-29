package geojson

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

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

func TestLineFeatureMatchesSharedAntimeridianFixture(t *testing.T) {
	fixture := loadAntimeridianFixture(t)
	points := make([]Point, 0, len(fixture.Input))
	for _, point := range fixture.Input {
		points = append(points, Point{Longitude: point.Longitude, Latitude: point.Latitude})
	}

	feature, ok := LineFeature("flightaware_route", "planned_route", points)
	if !ok {
		t.Fatal("LineFeature() ok = false, want true")
	}
	geometry := feature["geometry"].(map[string]any)
	coordinates := geometry["coordinates"].([][][]float64)
	if !reflect.DeepEqual(coordinates, fixture.ExpectedCoordinates) {
		t.Fatalf("coordinates = %#v, want %#v", coordinates, fixture.ExpectedCoordinates)
	}
}

type antimeridianFixture struct {
	Input []struct {
		Longitude float64 `json:"longitude"`
		Latitude  float64 `json:"latitude"`
	} `json:"input"`
	ExpectedCoordinates [][][]float64 `json:"expectedCoordinates"`
}

func loadAntimeridianFixture(t *testing.T) antimeridianFixture {
	t.Helper()
	path := filepath.Join("..", "..", "..", "packages", "shared-types", "fixtures", "geojson", "antimeridian-line.json")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read shared fixture: %v", err)
	}
	var fixture antimeridianFixture
	if err := json.Unmarshal(body, &fixture); err != nil {
		t.Fatalf("unmarshal shared fixture: %v", err)
	}
	return fixture
}
