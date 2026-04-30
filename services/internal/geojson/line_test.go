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

func TestLineFeatureKeepsPositiveAntimeridianEndpoint(t *testing.T) {
	tests := []struct {
		name   string
		points []Point
		want   [][][]float64
	}{
		{
			name: "positive antimeridian origin",
			points: []Point{
				{Longitude: 180, Latitude: 35},
				{Longitude: 170, Latitude: 40},
			},
			want: [][][]float64{{{180, 35}, {170, 40}}},
		},
		{
			name: "positive antimeridian destination",
			points: []Point{
				{Longitude: 170, Latitude: 35},
				{Longitude: 180, Latitude: 40},
			},
			want: [][][]float64{{{170, 35}, {180, 40}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feature, ok := LineFeature("flightaware_route", "planned_route", tt.points)
			if !ok {
				t.Fatal("LineFeature() ok = false, want true")
			}

			geometry := feature["geometry"].(map[string]any)
			coordinates := geometry["coordinates"].([][][]float64)
			if !reflect.DeepEqual(coordinates, tt.want) {
				t.Fatalf("coordinates = %#v, want %#v", coordinates, tt.want)
			}
		})
	}
}

func TestLineFeatureSkipsDegenerateAntimeridianEndpointSegments(t *testing.T) {
	tests := []struct {
		name   string
		points []Point
		want   [][][]float64
	}{
		{
			name: "positive antimeridian origin wraps west",
			points: []Point{
				{Longitude: 180, Latitude: 35},
				{Longitude: -170, Latitude: 40},
			},
			want: [][][]float64{{{-180, 35}, {-170, 40}}},
		},
		{
			name: "negative antimeridian origin wraps east",
			points: []Point{
				{Longitude: -180, Latitude: 35},
				{Longitude: 170, Latitude: 40},
			},
			want: [][][]float64{{{180, 35}, {170, 40}}},
		},
		{
			name: "negative antimeridian destination from east",
			points: []Point{
				{Longitude: 170, Latitude: 35},
				{Longitude: -180, Latitude: 40},
			},
			want: [][][]float64{{{170, 35}, {180, 40}}},
		},
		{
			name: "positive antimeridian destination from west",
			points: []Point{
				{Longitude: -170, Latitude: 35},
				{Longitude: 180, Latitude: 40},
			},
			want: [][][]float64{{{-170, 35}, {-180, 40}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feature, ok := LineFeature("flightaware_route", "planned_route", tt.points)
			if !ok {
				t.Fatal("LineFeature() ok = false, want true")
			}

			geometry := feature["geometry"].(map[string]any)
			coordinates := geometry["coordinates"].([][][]float64)
			if !reflect.DeepEqual(coordinates, tt.want) {
				t.Fatalf("coordinates = %#v, want %#v", coordinates, tt.want)
			}
		})
	}
}

func TestLineFeatureKeepsAntimeridianBoundaryLine(t *testing.T) {
	tests := []struct {
		name   string
		points []Point
		want   [][][]float64
	}{
		{
			name: "positive to negative antimeridian",
			points: []Point{
				{Longitude: 180, Latitude: 35},
				{Longitude: -180, Latitude: 40},
			},
			want: [][][]float64{{{180, 35}, {180, 40}}},
		},
		{
			name: "negative to positive antimeridian",
			points: []Point{
				{Longitude: -180, Latitude: 35},
				{Longitude: 180, Latitude: 40},
			},
			want: [][][]float64{{{-180, 35}, {-180, 40}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feature, ok := LineFeature("flightaware_route", "planned_route", tt.points)
			if !ok {
				t.Fatal("LineFeature() ok = false, want true")
			}

			geometry := feature["geometry"].(map[string]any)
			coordinates := geometry["coordinates"].([][][]float64)
			if !reflect.DeepEqual(coordinates, tt.want) {
				t.Fatalf("coordinates = %#v, want %#v", coordinates, tt.want)
			}
		})
	}
}

func TestLineFeatureKeepsSegmentsOnCorrectSideAfterAntimeridianBoundaryLine(t *testing.T) {
	feature, ok := LineFeature("flightaware_route", "planned_route", []Point{
		{Longitude: 170, Latitude: 35},
		{Longitude: -180, Latitude: 40},
		{Longitude: 180, Latitude: 45},
		{Longitude: 170, Latitude: 50},
	})
	if !ok {
		t.Fatal("LineFeature() ok = false, want true")
	}

	geometry := feature["geometry"].(map[string]any)
	coordinates := geometry["coordinates"].([][][]float64)
	want := [][][]float64{
		{{170, 35}, {180, 40}},
		{{-180, 40}, {-180, 45}},
		{{180, 45}, {170, 50}},
	}
	if !reflect.DeepEqual(coordinates, want) {
		t.Fatalf("coordinates = %#v, want %#v", coordinates, want)
	}
}

func TestLineFeatureReturnsUnavailableWhenAntimeridianBoundaryLineHasNoLength(t *testing.T) {
	feature, ok := LineFeature("flightaware_route", "planned_route", []Point{
		{Longitude: 180, Latitude: 35},
		{Longitude: -180, Latitude: 35},
	})
	if ok {
		t.Fatalf("LineFeature() ok = true with feature %#v, want false", feature)
	}
	if feature != nil {
		t.Fatalf("LineFeature() feature = %#v, want nil", feature)
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
