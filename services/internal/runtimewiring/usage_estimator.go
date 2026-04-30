package runtimewiring

import (
	"strconv"
	"strings"

	"airpath/services/internal/flightaware"
)

const defaultFlightAwareEstimatedCostUSDPerCall = 0.01

func FlightAwareUsageEstimatorFromEnv(lookup EnvLookup) flightaware.StaticUsageEstimator {
	costUSD := defaultFlightAwareEstimatedCostUSDPerCall
	if lookup != nil {
		if parsed, err := strconv.ParseFloat(strings.TrimSpace(lookup(FlightAwareEstimatedCostUSDPerCallEnv)), 64); err == nil && parsed > 0 {
			costUSD = parsed
		}
	}

	endpoints := []flightaware.Endpoint{
		flightaware.EndpointSearch,
		flightaware.EndpointSummary,
		flightaware.EndpointRoute,
		flightaware.EndpointPosition,
		flightaware.EndpointTrack,
		flightaware.EndpointSchedule,
		flightaware.EndpointUsage,
	}
	resultSets := make(map[flightaware.Endpoint]int, len(endpoints))
	costs := make(map[flightaware.Endpoint]float64, len(endpoints))
	for _, endpoint := range endpoints {
		resultSets[endpoint] = 1
		costs[endpoint] = costUSD
	}

	return flightaware.StaticUsageEstimator{
		ResultSetsByEndpoint: resultSets,
		CostUSDByEndpoint:    costs,
	}
}
