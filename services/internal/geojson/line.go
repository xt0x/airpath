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
		coordinates = append(coordinates, []float64{roundCoordinate(normalizeLongitude(point.Longitude)), roundCoordinate(point.Latitude)})
	}
	if len(coordinates) < 2 {
		return nil, false
	}
	return map[string]any{
		"type": "Feature",
		"geometry": map[string]any{
			"type":        "MultiLineString",
			"coordinates": splitLineStringForAntimeridian(coordinates),
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
	for index := 1; index < len(coordinates); index++ {
		previous := coordinates[index-1]
		current := coordinates[index]
		delta := current[0] - previous[0]
		if math.Abs(delta) <= 180 {
			segments[len(segments)-1] = append(segments[len(segments)-1], current)
			continue
		}

		crossingLongitude := 180.0
		wrappedLongitude := -180.0
		if delta > 0 {
			crossingLongitude = -180
			wrappedLongitude = 180
		}
		ratio := (crossingLongitude - previous[0]) / delta
		latitude := previous[1] + (current[1]-previous[1])*ratio
		segments[len(segments)-1] = append(segments[len(segments)-1], []float64{crossingLongitude, roundCoordinate(latitude)})
		segments = append(segments, [][]float64{{wrappedLongitude, roundCoordinate(latitude)}, current})
	}
	return segments
}

func validPoint(point Point) bool {
	return !math.IsNaN(point.Longitude) && !math.IsInf(point.Longitude, 0) &&
		!math.IsNaN(point.Latitude) && !math.IsInf(point.Latitude, 0) &&
		point.Longitude >= -180 && point.Longitude <= 180 &&
		point.Latitude >= -90 && point.Latitude <= 90
}

func normalizeLongitude(longitude float64) float64 {
	normalized := math.Mod(longitude+180, 360)
	if normalized < 0 {
		normalized += 360
	}
	return normalized - 180
}

func roundCoordinate(value float64) float64 {
	return math.Round(value*1_000_000) / 1_000_000
}
