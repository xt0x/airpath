package domain

type FlightPositionMetricsInput struct {
	AltitudeHundredsFeet *int
	GroundspeedKnots     *int
	HeadingDegrees       *int
}

type FlightPositionMetrics struct {
	AltitudeHundredsFeet *int
	AltitudeFeet         *int
	GroundspeedKnots     *int
	HeadingDegrees       *int
}

func NormalizeAltitudeFeet(altitudeHundredsFeet *int) *int {
	if altitudeHundredsFeet == nil {
		return nil
	}

	altitudeFeet := *altitudeHundredsFeet * 100
	return &altitudeFeet
}

func NormalizeGroundspeedKnots(groundspeedKnots *int) *int {
	return groundspeedKnots
}

func NormalizeHeadingDegrees(headingDegrees *int) *int {
	if headingDegrees == nil {
		return nil
	}

	normalized := *headingDegrees
	if normalized == 360 {
		normalized = 0
	}

	return &normalized
}

func NormalizeFlightPositionMetrics(input FlightPositionMetricsInput) FlightPositionMetrics {
	return FlightPositionMetrics{
		AltitudeHundredsFeet: input.AltitudeHundredsFeet,
		AltitudeFeet:         NormalizeAltitudeFeet(input.AltitudeHundredsFeet),
		GroundspeedKnots:     NormalizeGroundspeedKnots(input.GroundspeedKnots),
		HeadingDegrees:       NormalizeHeadingDegrees(input.HeadingDegrees),
	}
}
