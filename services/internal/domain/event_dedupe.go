package domain

type FlightEventDedupeKeyInput struct {
	Source                 FlightEventSource
	AlertID                *string
	EventType              FlightEventType
	FAFlightID             *FAFlightID
	ProvisionalFlightLegID *ProvisionalFlightLegID
	EventTimestamp         ISODateTimeString
}

func GenerateFlightEventDedupeKey(input FlightEventDedupeKeyInput) string {
	return hashFlightIDParts(
		string(input.Source),
		eventDedupeSourceID(input),
		string(input.EventType),
		eventDedupeFlightKey(input),
		string(input.EventTimestamp),
	)
}

func eventDedupeSourceID(input FlightEventDedupeKeyInput) string {
	if input.Source == FlightEventSourceAlert {
		if input.AlertID == nil {
			return ""
		}

		return *input.AlertID
	}

	return "polling"
}

func eventDedupeFlightKey(input FlightEventDedupeKeyInput) string {
	if input.FAFlightID != nil {
		return string(*input.FAFlightID)
	}
	if input.ProvisionalFlightLegID != nil {
		return string(*input.ProvisionalFlightLegID)
	}

	return ""
}
