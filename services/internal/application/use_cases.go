package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"airpath/services/internal/domain"
	"airpath/services/internal/flightaware"
)

const FetchTaskDedupeWindow = 5 * time.Minute

func (a *Application) SearchFlights(ctx context.Context, input SearchFlightsInput) (FlightSearchResponse, error) {
	flights, cache, err := a.flights.SearchByIdent(ctx, input.Ident)
	if err != nil {
		return FlightSearchResponse{}, err
	}
	cache.CheckedAt = input.CheckedAt
	if len(flights) > 0 {
		return FlightSearchResponse{Items: summarizeFlights(flights), Cache: cache}, nil
	}

	allowed, err := a.usageGuard.FetchingAllowed(ctx)
	if err != nil {
		return FlightSearchResponse{}, err
	}
	if !allowed {
		return FlightSearchResponse{}, ErrBudgetExceeded
	}

	_, err = a.fetchTasks.EnqueueFetchTask(ctx, FetchTask{
		SchemaVersion:  1,
		TaskID:         stableTaskID("search", domain.FlightID(input.Ident), FetchTaskSummary, input.CheckedAt),
		TaskType:       FetchTaskSummary,
		FlightID:       domain.FlightID(input.Ident),
		RequestedAt:    input.CheckedAt,
		Reason:         FetchReasonSearchResultSeed,
		IdempotencyKey: stableTaskID("search", domain.FlightID(input.Ident), FetchTaskSummary, "seed"),
	})
	if err != nil {
		return FlightSearchResponse{}, err
	}

	return FlightSearchResponse{Items: []FlightSummaryItem{}, Cache: cache}, nil
}

func (a *Application) GetFlightDetail(ctx context.Context, input FlightDetailInput) (FlightDetailResponse, error) {
	flight, cache, err := a.flights.GetFlight(ctx, input.FlightID)
	if err != nil {
		return FlightDetailResponse{}, err
	}
	cache.CheckedAt = input.CheckedAt

	route, err := a.mapData.GetPlannedRoute(ctx, flight)
	if err != nil {
		return FlightDetailResponse{}, err
	}
	track, err := a.mapData.GetActualTrack(ctx, flight)
	if err != nil {
		return FlightDetailResponse{}, err
	}
	currentPosition, _, err := a.positions.GetLatestPosition(ctx, input.FlightID)
	if err != nil {
		return FlightDetailResponse{}, err
	}

	return FlightDetailResponse{
		Flight:  flight,
		Route:   route,
		Track:   track,
		Current: mapPosition(currentPosition),
		Cache:   cache,
	}, nil
}

func (a *Application) GetFlightMapData(ctx context.Context, input FlightMapDataInput) (FlightMapDataResponse, error) {
	flight, cache, err := a.flights.GetFlight(ctx, input.FlightID)
	if err != nil {
		return FlightMapDataResponse{}, err
	}
	cache.CheckedAt = input.CheckedAt

	planned, err := a.mapData.GetPlannedRoute(ctx, flight)
	if err != nil {
		return FlightMapDataResponse{}, err
	}
	actual, err := a.mapData.GetActualTrack(ctx, flight)
	if err != nil {
		return FlightMapDataResponse{}, err
	}
	currentPosition, _, err := a.positions.GetLatestPosition(ctx, input.FlightID)
	if err != nil {
		return FlightMapDataResponse{}, err
	}

	return FlightMapDataResponse{
		FlightID:   flight.FlightID,
		FAFlightID: flight.FAFlightID,
		Planned:    planned,
		Actual:     actual,
		Current:    currentLayer(currentPosition),
		Cache:      cache,
	}, nil
}

func (a *Application) RequestFlightRefresh(ctx context.Context, input FlightRefreshInput) (FlightRefreshResponse, error) {
	flight, cache, err := a.flights.GetFlight(ctx, input.FlightID)
	if err != nil {
		return FlightRefreshResponse{}, err
	}
	cache.CheckedAt = input.RequestedAt

	allowed, err := a.usageGuard.FetchingAllowed(ctx)
	if err != nil {
		return FlightRefreshResponse{}, err
	}
	if !allowed {
		return FlightRefreshResponse{FlightID: input.FlightID, AcceptedTasks: []FetchTask{}, Cache: staleCache(cache, input.RequestedAt)}, nil
	}

	accepted := make([]FetchTask, 0, len(input.TaskTypes))
	for _, taskType := range dedupeTaskTypes(input.TaskTypes) {
		taskAllowed, err := a.fetchTaskAllowed(ctx, taskType, input.ClientReason)
		if err != nil {
			return FlightRefreshResponse{}, err
		}
		if !taskAllowed {
			continue
		}
		task := FetchTask{
			SchemaVersion:  1,
			TaskID:         stableTaskID("refresh", input.FlightID, taskType, input.RequestedAt),
			TaskType:       taskType,
			FlightID:       input.FlightID,
			FAFlightID:     flight.FAFlightID,
			RequestedAt:    input.RequestedAt,
			Reason:         input.ClientReason,
			IdempotencyKey: fetchTaskWindowKey(input.FlightID, taskType, input.RequestedAt),
		}
		enqueued, err := a.fetchTasks.EnqueueFetchTask(ctx, task)
		if err != nil {
			return FlightRefreshResponse{}, err
		}
		if enqueued {
			accepted = append(accepted, task)
		}
	}

	return FlightRefreshResponse{FlightID: input.FlightID, AcceptedTasks: accepted, Cache: cache}, nil
}

func (a *Application) fetchTaskAllowed(ctx context.Context, taskType FetchTaskType, reason FetchReason) (bool, error) {
	if a.fetchPolicy == nil {
		return true, nil
	}
	return a.fetchPolicy.FetchTaskAllowed(ctx, taskType, reason)
}

func (a *Application) GetUsageStatus(ctx context.Context, input UsageStatusInput) (UsageStatus, error) {
	status, err := a.usageGuard.GetUsageStatus(ctx)
	if err != nil {
		return UsageStatus{}, err
	}
	status.Cache = CacheMetadata{
		Freshness: CacheFreshnessFresh,
		Source:    CacheSourceLocalAccounting,
		Stale:     false,
		CheckedAt: input.CheckedAt,
	}
	return status, nil
}

func MapApplicationError(err error, requestID string) APIError {
	switch {
	case errors.Is(err, ErrBudgetExceeded):
		return apiError(ApiErrorFlightAwareBudgetExceeded, "FlightAware budget has been exceeded", false, requestID)
	case errors.Is(err, flightaware.ErrFlightAwareRateLimited):
		return apiError(ApiErrorFlightAwareRateLimited, "FlightAware rate limit is active", true, requestID)
	case errors.Is(err, ErrStaleCacheUnavailable):
		apiErr := apiError(ApiErrorStaleCacheUnavailable, "No fresh or stale cache is available", false, requestID)
		apiErr.StaleCacheAvailable = false
		return apiErr
	case errors.Is(err, flightaware.ErrFlightAwareFetchDisabled):
		return apiError(ApiErrorFlightAwareFetchDisabled, "FlightAware fetch is disabled", false, requestID)
	case errors.Is(err, ErrNotFound):
		return apiError(ApiErrorStaleCacheUnavailable, "Requested cached resource was not found", false, requestID)
	case errors.Is(err, ErrValidation):
		return apiError(ApiErrorUpstreamFailure, "Request validation failed", false, requestID)
	default:
		return apiError(ApiErrorUpstreamFailure, "Upstream request failed", true, requestID)
	}
}

func summarizeFlights(flights []domain.Flight) []FlightSummaryItem {
	items := make([]FlightSummaryItem, 0, len(flights))
	for _, flight := range flights {
		items = append(items, FlightSummaryItem{
			FlightID:               flight.FlightID,
			FlightIDType:           flight.FlightIDType,
			ProvisionalFlightLegID: flight.ProvisionalFlightLegID,
			FAFlightID:             flight.FAFlightID,
			Ident:                  flight.Ident,
			IdentIATA:              flight.IdentIATA,
			Origin:                 flight.Origin.Code,
			Destination:            flight.Destination.Code,
			ScheduledOut:           flight.Times.ScheduledOut,
			LegIndex:               flight.LegIndex,
			Status:                 flight.Status,
		})
	}
	return items
}

func mapPosition(position *domain.FlightPosition) *Position {
	if position == nil {
		return nil
	}
	return &Position{
		Latitude:             position.Latitude,
		Longitude:            position.Longitude,
		AltitudeHundredsFeet: position.AltitudeHundredsFeet,
		AltitudeFeet:         position.AltitudeFeet,
		GroundspeedKnots:     position.GroundspeedKnots,
		HeadingDegrees:       position.HeadingDegrees,
		Timestamp:            position.Timestamp,
		Source:               position.Source,
	}
}

func currentLayer(position *domain.FlightPosition) MapLayer {
	mapped := mapPosition(position)
	if mapped == nil {
		return MapLayer{Source: MapSourceFlightAwarePosition, Available: false, UnavailableReason: ptr("current position unavailable")}
	}
	return MapLayer{
		Source:    MapSourceFlightAwarePosition,
		Available: true,
		GeoJSON: map[string]any{
			"type": "Feature",
			"geometry": map[string]any{
				"type":        "Point",
				"coordinates": []float64{mapped.Longitude, mapped.Latitude},
			},
			"properties": map[string]any{
				"kind":      "current_position",
				"timestamp": mapped.Timestamp,
			},
		},
	}
}

func dedupeTaskTypes(taskTypes []FetchTaskType) []FetchTaskType {
	seen := map[FetchTaskType]struct{}{}
	deduped := make([]FetchTaskType, 0, len(taskTypes))
	for _, taskType := range taskTypes {
		if _, ok := seen[taskType]; ok {
			continue
		}
		seen[taskType] = struct{}{}
		deduped = append(deduped, taskType)
	}
	return deduped
}

func stableTaskID(prefix string, flightID domain.FlightID, taskType FetchTaskType, suffix any) string {
	return fmt.Sprintf("%s:%s:%s:%v", prefix, flightID, taskType, suffix)
}

func fetchTaskWindowKey(flightID domain.FlightID, taskType FetchTaskType, requestedAt string) string {
	windowStart := requestedAt
	if parsed, err := time.Parse(time.RFC3339, requestedAt); err == nil {
		windowStart = parsed.UTC().Truncate(FetchTaskDedupeWindow).Format("2006-01-02T15:04:05Z")
	}
	return stableTaskID("refresh", flightID, taskType, windowStart)
}

func staleCache(cache CacheMetadata, checkedAt string) CacheMetadata {
	cache.Freshness = CacheFreshnessStale
	cache.Stale = true
	cache.CheckedAt = checkedAt
	return cache
}

func apiError(code ApiErrorCode, message string, retryable bool, requestID string) APIError {
	return APIError{
		Code:      code,
		Message:   message,
		Retryable: retryable,
		RequestID: requestID,
	}
}
