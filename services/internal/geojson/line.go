package geojson

import "math"

type Point struct {
	Longitude float64
	Latitude  float64
}

func LineFeature(source string, kind string, points []Point) (map[string]any, bool) {
	if len(points) < 2 {
		return nil, false
	}
	coordinates := make([][]float64, 0, len(points))
	for _, point := range points {
		if !validPoint(point) {
			continue
		}
		coordinates = append(coordinates, []float64{roundCoordinate(point.Longitude), roundCoordinate(point.Latitude)})
	}
	if len(coordinates) < 2 {
		return nil, false
	}
	segments := splitLineStringForAntimeridian(coordinates)
	if len(segments) == 0 {
		return nil, false
	}
	return map[string]any{
		"type": "Feature",
		"geometry": map[string]any{
			"type":        "MultiLineString",
			"coordinates": segments,
		},
		"properties": map[string]any{
			"kind":   kind,
			"source": source,
		},
	}, true
}

func splitLineStringForAntimeridian(coordinates [][]float64) [][][]float64 {
	if len(coordinates) == 0 {
		return [][][]float64{}
	}
	segments := [][][]float64{{coordinates[0]}}
	previous := coordinates[0]
	for index := 1; index < len(coordinates); index++ {
		current := effectiveCurrentPoint(previous, coordinates[index])
		delta := current[0] - previous[0]
		if math.Abs(delta) <= 180 {
			segments = appendToLastSegment(segments, current)
			previous = current
			continue
		}

		adjustedCurrentLongitude := current[0]
		crossingLongitude := 180.0
		wrappedLongitude := -180.0
		// Interpolate the antimeridian crossing in an unwrapped longitude space,
		// then start the next segment from the opposite boundary.
		if delta > 0 {
			adjustedCurrentLongitude = current[0] - 360
			crossingLongitude = -180
			wrappedLongitude = 180
		} else {
			adjustedCurrentLongitude = current[0] + 360
		}
		ratio := (crossingLongitude - previous[0]) / (adjustedCurrentLongitude - previous[0])
		latitude := previous[1] + (current[1]-previous[1])*ratio
		segments = appendToLastSegment(segments, []float64{crossingLongitude, roundCoordinate(latitude)})
		segments = append(segments, [][]float64{{wrappedLongitude, roundCoordinate(latitude)}})
		segments = appendToLastSegment(segments, current)
		previous = current
	}
	return nonDegenerateSegments(segments)
}

func effectiveCurrentPoint(previous []float64, current []float64) []float64 {
	if sameAntimeridianBoundary(previous[0], current[0]) {
		return []float64{previous[0], current[1]}
	}
	return current
}

func appendToLastSegment(segments [][][]float64, point []float64) [][][]float64 {
	lastIndex := len(segments) - 1
	segment := segments[lastIndex]
	if len(segment) == 0 || !sameCoordinate(segment[len(segment)-1], point) {
		segments[lastIndex] = append(segment, point)
	}
	return segments
}

func nonDegenerateSegments(segments [][][]float64) [][][]float64 {
	filtered := make([][][]float64, 0, len(segments))
	for _, segment := range segments {
		if lineStringHasLength(segment) {
			filtered = append(filtered, segment)
		}
	}
	return filtered
}

func lineStringHasLength(segment [][]float64) bool {
	if len(segment) < 2 {
		return false
	}
	first := segment[0]
	for _, point := range segment[1:] {
		if !sameCoordinate(first, point) {
			return true
		}
	}
	return false
}

func sameCoordinate(left []float64, right []float64) bool {
	return len(left) >= 2 && len(right) >= 2 && left[0] == right[0] && left[1] == right[1]
}

func sameAntimeridianBoundary(leftLongitude float64, rightLongitude float64) bool {
	return (leftLongitude == 180 && rightLongitude == -180) ||
		(leftLongitude == -180 && rightLongitude == 180)
}

func validPoint(point Point) bool {
	return !math.IsNaN(point.Longitude) && !math.IsInf(point.Longitude, 0) &&
		!math.IsNaN(point.Latitude) && !math.IsInf(point.Latitude, 0) &&
		point.Longitude >= -180 && point.Longitude <= 180 &&
		point.Latitude >= -90 && point.Latitude <= 90
}

func roundCoordinate(value float64) float64 {
	return math.Round(value*1_000_000) / 1_000_000
}
