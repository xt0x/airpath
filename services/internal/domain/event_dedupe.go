package domain

type FlightEventDedupeKeyInput struct {
	EventType              FlightEventType
	FAFlightID             *FAFlightID
	ProvisionalFlightLegID *ProvisionalFlightLegID
	EventTimestamp         ISODateTimeString
}

func GenerateFlightEventDedupeKey(input FlightEventDedupeKeyInput) string {
	return hashFlightIDParts(
		string(FlightEventSourcePolling),
		string(input.EventType),
		eventDedupeFlightKey(input),
		string(input.EventTimestamp),
	)
}

func eventDedupeFlightKey(input FlightEventDedupeKeyInput) string {
	if input.FAFlightID != nil {
		return string(*input.FAFlightID)
	}
	if input.ProvisionalFlightLegID != nil {
		return string(*input.ProvisionalFlightLegID)
	}

	panic("faFlightId or provisionalFlightLegId is required for event dedupe keys")
}
