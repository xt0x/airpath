package domain

import (
	"fmt"
	"hash/fnv"
	"strconv"
)

type ProvisionalFlightLegIDInput struct {
	Ident           string
	OriginCode      AirportCode
	DestinationCode AirportCode
	ScheduledOut    ISODateTimeString
}

type InternalFlightLegIDInput struct {
	FAFlightID      FAFlightID
	OriginCode      AirportCode
	DestinationCode AirportCode
	ScheduledOut    ISODateTimeString
	LegIndex        int
}

func GenerateProvisionalFlightLegID(input ProvisionalFlightLegIDInput) ProvisionalFlightLegID {
	return ProvisionalFlightLegID("sched_" + hashFlightIDParts(
		input.Ident,
		string(input.OriginCode),
		string(input.DestinationCode),
		string(input.ScheduledOut),
	))
}

func GenerateInternalFlightLegID(input InternalFlightLegIDInput) InternalFlightLegID {
	return InternalFlightLegID("iflg_" + hashFlightIDParts(
		string(input.FAFlightID),
		string(input.OriginCode),
		string(input.DestinationCode),
		string(input.ScheduledOut),
		strconv.Itoa(input.LegIndex),
	))
}

func hashFlightIDParts(parts ...string) string {
	hasher := fnv.New64a()
	for index, part := range parts {
		if index > 0 {
			_, _ = hasher.Write([]byte{0x1f})
		}
		_, _ = hasher.Write([]byte(part))
	}

	return fmt.Sprintf("%016x", hasher.Sum64())[:12]
}
