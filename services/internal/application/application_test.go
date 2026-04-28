package application

import (
	"context"
	"errors"
	"testing"

	"airpath/services/internal/domain"
	"airpath/services/internal/flightaware"
)

func TestSearchFlightsUsesCacheFirstAndDoesNotEnqueueOnHit(t *testing.T) {
	app, deps := newTestApp()
	deps.flights.searchResults["ANA110"] = []domain.Flight{testFlight("iflg_1", "ANA110")}

	response, err := app.SearchFlights(context.Background(), SearchFlightsInput{Ident: "ANA110", CheckedAt: "2026-04-29T00:00:00Z"})
	if err != nil {
		t.Fatalf("SearchFlights() error = %v", err)
	}

	if len(response.Items) != 1 {
		t.Fatalf("item count = %d, want 1", len(response.Items))
	}
	if response.Cache.Source != CacheSourceCache || response.Cache.Freshness != CacheFreshnessFresh {
		t.Fatalf("cache = %#v", response.Cache)
	}
	if len(deps.queue.tasks) != 0 {
		t.Fatalf("enqueued tasks = %#v, want none", deps.queue.tasks)
	}
}

func TestSearchFlightsEnqueuesSummaryFetchOnCacheMissWhenAllowed(t *testing.T) {
	app, deps := newTestApp()

	response, err := app.SearchFlights(context.Background(), SearchFlightsInput{Ident: "ANA110", CheckedAt: "2026-04-29T00:00:00Z"})
	if err != nil {
		t.Fatalf("SearchFlights() error = %v", err)
	}

	if len(response.Items) != 0 {
		t.Fatalf("item count = %d, want 0", len(response.Items))
	}
	if response.Cache.Freshness != CacheFreshnessMiss {
		t.Fatalf("freshness = %q, want miss", response.Cache.Freshness)
	}
	if len(deps.queue.tasks) != 1 {
		t.Fatalf("enqueued tasks = %#v, want one summary task", deps.queue.tasks)
	}
	if deps.queue.tasks[0].TaskType != FetchTaskSummary || deps.queue.tasks[0].Reason != FetchReasonSearchResultSeed {
		t.Fatalf("task = %#v", deps.queue.tasks[0])
	}
}

func TestSearchFlightsReturnsBudgetErrorOnCacheMissWhenFetchDisabled(t *testing.T) {
	app, deps := newTestApp()
	deps.budget.fetchingEnabled = false

	_, err := app.SearchFlights(context.Background(), SearchFlightsInput{Ident: "ANA110", CheckedAt: "2026-04-29T00:00:00Z"})
	if !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("SearchFlights(disabled) error = %v, want ErrBudgetExceeded", err)
	}
	if len(deps.queue.tasks) != 0 {
		t.Fatalf("enqueued tasks = %#v, want none", deps.queue.tasks)
	}
}

func TestGetFlightDetailReturnsCachedFlightWithRouteTrackAndCurrentPositionFreshness(t *testing.T) {
	app, deps := newTestApp()
	flight := testFlight("iflg_1", "ANA110")
	deps.flights.byID[flight.FlightID] = flight
	deps.mapData.routes[flight.FlightID] = MapLayer{Source: MapSourceFlightAwareRoute, Available: true, GeoJSON: map[string]any{"type": "Feature"}}
	deps.mapData.tracks[flight.FlightID] = MapLayer{Source: MapSourceFlightAwareTrack, Available: true, GeoJSON: map[string]any{"type": "Feature"}}
	deps.positions.latest[flight.FlightID] = testPosition(flight.FlightID)

	response, err := app.GetFlightDetail(context.Background(), FlightDetailInput{FlightID: flight.FlightID, CheckedAt: "2026-04-29T00:00:00Z"})
	if err != nil {
		t.Fatalf("GetFlightDetail() error = %v", err)
	}

	if response.Flight.FlightID != flight.FlightID {
		t.Fatalf("flight ID = %q, want %q", response.Flight.FlightID, flight.FlightID)
	}
	if !response.Route.Available || !response.Track.Available || response.Current == nil {
		t.Fatalf("detail did not include cached map/current state: %#v", response)
	}
	if response.Cache.Source != CacheSourceCache {
		t.Fatalf("cache source = %q, want cache", response.Cache.Source)
	}
}

func TestGetFlightMapDataBuildsRouteTrackAndCurrentLayersFromCachedData(t *testing.T) {
	app, deps := newTestApp()
	flight := testFlight("iflg_1", "ANA110")
	deps.flights.byID[flight.FlightID] = flight
	deps.mapData.routes[flight.FlightID] = MapLayer{Source: MapSourceAirportGreatCircleFallback, Available: true, GeoJSON: map[string]any{"type": "Feature", "kind": "planned"}}
	deps.mapData.tracks[flight.FlightID] = MapLayer{Source: MapSourceFlightAwareTrack, Available: true, GeoJSON: map[string]any{"type": "Feature", "kind": "track"}}
	deps.positions.latest[flight.FlightID] = testPosition(flight.FlightID)

	response, err := app.GetFlightMapData(context.Background(), FlightMapDataInput{FlightID: flight.FlightID, CheckedAt: "2026-04-29T00:00:00Z"})
	if err != nil {
		t.Fatalf("GetFlightMapData() error = %v", err)
	}

	if response.Planned.Source != MapSourceAirportGreatCircleFallback || !response.Actual.Available || !response.Current.Available {
		t.Fatalf("map response = %#v", response)
	}
	if response.FAFlightID == nil || *response.FAFlightID != "fa_1" {
		t.Fatalf("faFlightId = %v, want fa_1", response.FAFlightID)
	}
}

func TestRequestFlightRefreshCreatesDedupedTasksOnlyWhenBudgetAllows(t *testing.T) {
	app, deps := newTestApp()
	flight := testFlight("iflg_1", "ANA110")
	deps.flights.byID[flight.FlightID] = flight

	response, err := app.RequestFlightRefresh(context.Background(), FlightRefreshInput{
		FlightID:      flight.FlightID,
		TaskTypes:     []FetchTaskType{FetchTaskRoute, FetchTaskRoute, FetchTaskPosition},
		ClientReason:  FetchReasonUserManualRefresh,
		RequestedAt:   "2026-04-29T00:00:00Z",
		IdempotencyID: "req-1",
	})
	if err != nil {
		t.Fatalf("RequestFlightRefresh() error = %v", err)
	}

	if len(response.AcceptedTasks) != 2 {
		t.Fatalf("accepted task count = %d, want 2", len(response.AcceptedTasks))
	}
	if len(deps.queue.tasks) != 2 {
		t.Fatalf("enqueued task count = %d, want 2", len(deps.queue.tasks))
	}
	if response.AcceptedTasks[0].IdempotencyKey != "refresh:iflg_1:route:2026-04-29T00:00:00Z" {
		t.Fatalf("route idempotency key = %q, want window-based key", response.AcceptedTasks[0].IdempotencyKey)
	}

	deps.budget.fetchingEnabled = false
	stale, err := app.RequestFlightRefresh(context.Background(), FlightRefreshInput{
		FlightID:     flight.FlightID,
		TaskTypes:    []FetchTaskType{FetchTaskTrack},
		ClientReason: FetchReasonUserManualRefresh,
		RequestedAt:  "2026-04-29T00:01:00Z",
	})
	if err != nil {
		t.Fatalf("RequestFlightRefresh(disabled with cache) error = %v", err)
	}
	if len(stale.AcceptedTasks) != 0 || stale.Cache.Freshness != CacheFreshnessStale || !stale.Cache.Stale {
		t.Fatalf("disabled refresh response = %#v, want stale cached response without tasks", stale)
	}
}

func TestRequestFlightRefreshDedupesSameFlightKindAndWindowAcrossRequests(t *testing.T) {
	app, deps := newTestApp()
	flight := testFlight("iflg_1", "ANA110")
	deps.flights.byID[flight.FlightID] = flight

	first, err := app.RequestFlightRefresh(context.Background(), FlightRefreshInput{
		FlightID:      flight.FlightID,
		TaskTypes:     []FetchTaskType{FetchTaskRoute},
		ClientReason:  FetchReasonUserManualRefresh,
		RequestedAt:   "2026-04-29T00:00:30Z",
		IdempotencyID: "req-1",
	})
	if err != nil {
		t.Fatalf("first RequestFlightRefresh() error = %v", err)
	}
	second, err := app.RequestFlightRefresh(context.Background(), FlightRefreshInput{
		FlightID:      flight.FlightID,
		TaskTypes:     []FetchTaskType{FetchTaskRoute},
		ClientReason:  FetchReasonUserManualRefresh,
		RequestedAt:   "2026-04-29T00:04:59Z",
		IdempotencyID: "req-2",
	})
	if err != nil {
		t.Fatalf("second RequestFlightRefresh() error = %v", err)
	}
	third, err := app.RequestFlightRefresh(context.Background(), FlightRefreshInput{
		FlightID:      flight.FlightID,
		TaskTypes:     []FetchTaskType{FetchTaskRoute},
		ClientReason:  FetchReasonUserManualRefresh,
		RequestedAt:   "2026-04-29T00:05:00Z",
		IdempotencyID: "req-3",
	})
	if err != nil {
		t.Fatalf("third RequestFlightRefresh() error = %v", err)
	}

	if len(first.AcceptedTasks) != 1 || len(second.AcceptedTasks) != 0 || len(third.AcceptedTasks) != 1 {
		t.Fatalf("accepted counts = %d, %d, %d; want 1, 0, 1", len(first.AcceptedTasks), len(second.AcceptedTasks), len(third.AcceptedTasks))
	}
	if len(deps.queue.tasks) != 2 {
		t.Fatalf("enqueued task count = %d, want 2", len(deps.queue.tasks))
	}
}

func TestRequestFlightRefreshHonorsLowPriorityKillSwitch(t *testing.T) {
	policy := NewRuntimeFetchPolicy(RuntimeFetchConfig{
		RouteFetchEnabled:      false,
		TrackFetchEnabled:      false,
		BackgroundFetchEnabled: false,
	})
	app, deps := newTestAppWithPolicy(policy)
	flight := testFlight("iflg_1", "ANA110")
	deps.flights.byID[flight.FlightID] = flight

	response, err := app.RequestFlightRefresh(context.Background(), FlightRefreshInput{
		FlightID:      flight.FlightID,
		TaskTypes:     []FetchTaskType{FetchTaskRoute, FetchTaskTrack, FetchTaskPosition},
		ClientReason:  FetchReasonUserManualRefresh,
		RequestedAt:   "2026-04-29T00:00:00Z",
		IdempotencyID: "req-1",
	})
	if err != nil {
		t.Fatalf("RequestFlightRefresh() error = %v", err)
	}

	if len(response.AcceptedTasks) != 1 || response.AcceptedTasks[0].TaskType != FetchTaskPosition {
		t.Fatalf("accepted tasks = %#v, want only position", response.AcceptedTasks)
	}

	background, err := app.RequestFlightRefresh(context.Background(), FlightRefreshInput{
		FlightID:      flight.FlightID,
		TaskTypes:     []FetchTaskType{FetchTaskPosition},
		ClientReason:  FetchReasonLowFrequencyPoll,
		RequestedAt:   "2026-04-29T00:05:00Z",
		IdempotencyID: "req-2",
	})
	if err != nil {
		t.Fatalf("background RequestFlightRefresh() error = %v", err)
	}
	if len(background.AcceptedTasks) != 0 {
		t.Fatalf("background accepted tasks = %#v, want none", background.AcceptedTasks)
	}

	policy.Update(RuntimeFetchConfig{RouteFetchEnabled: true, TrackFetchEnabled: true, BackgroundFetchEnabled: true})
	resumed, err := app.RequestFlightRefresh(context.Background(), FlightRefreshInput{
		FlightID:      flight.FlightID,
		TaskTypes:     []FetchTaskType{FetchTaskRoute},
		ClientReason:  FetchReasonUserManualRefresh,
		RequestedAt:   "2026-04-29T00:10:00Z",
		IdempotencyID: "req-3",
	})
	if err != nil {
		t.Fatalf("resumed RequestFlightRefresh() error = %v", err)
	}
	if len(resumed.AcceptedTasks) != 1 || resumed.AcceptedTasks[0].TaskType != FetchTaskRoute {
		t.Fatalf("resumed accepted tasks = %#v, want route", resumed.AcceptedTasks)
	}
}

func TestGetUsageStatusReturnsBudgetRateLimitAndCacheState(t *testing.T) {
	app, deps := newTestApp()
	deps.budget.status = UsageStatus{
		Budget: UsageBudgetStatus{
			Currency:                 "USD",
			EstimatedMonthToDateCost: 3.25,
			SoftStopThreshold:        4,
			Stopped:                  false,
		},
		RateLimit:       RateLimitStatus{Limited: true, ResetAt: ptr("2026-04-29T00:10:00Z")},
		FetchingEnabled: true,
	}

	response, err := app.GetUsageStatus(context.Background(), UsageStatusInput{CheckedAt: "2026-04-29T00:00:00Z"})
	if err != nil {
		t.Fatalf("GetUsageStatus() error = %v", err)
	}

	if response.Budget.EstimatedMonthToDateCost != 3.25 || !response.RateLimit.Limited || !response.FetchingEnabled {
		t.Fatalf("usage status = %#v", response)
	}
	if response.Cache.Source != CacheSourceLocalAccounting {
		t.Fatalf("cache source = %q, want local accounting", response.Cache.Source)
	}
}

func TestMapApplicationErrorProducesTypedAPIError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code ApiErrorCode
	}{
		{name: "budget", err: ErrBudgetExceeded, code: ApiErrorFlightAwareBudgetExceeded},
		{name: "rate", err: flightaware.NewRateLimitedError(flightaware.EndpointRoute, 429, "limited"), code: ApiErrorFlightAwareRateLimited},
		{name: "stale cache", err: ErrStaleCacheUnavailable, code: ApiErrorStaleCacheUnavailable},
		{name: "fetch disabled", err: flightaware.ErrFlightAwareFetchDisabled, code: ApiErrorFlightAwareFetchDisabled},
		{name: "validation", err: ErrValidation, code: ApiErrorUpstreamFailure},
		{name: "not found", err: ErrNotFound, code: ApiErrorStaleCacheUnavailable},
		{name: "upstream", err: errors.New("upstream failed"), code: ApiErrorUpstreamFailure},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := MapApplicationError(testCase.err, "req-123")
			if got.Code != testCase.code {
				t.Fatalf("code = %q, want %q", got.Code, testCase.code)
			}
			if got.RequestID != "req-123" {
				t.Fatalf("request ID = %q, want req-123", got.RequestID)
			}
		})
	}
}

type testDeps struct {
	flights   *memoryFlightStore
	mapData   *memoryMapDataStore
	positions *memoryPositionStore
	queue     *memoryFetchTaskQueue
	budget    *memoryUsageGuard
}

func newTestApp() (*Application, testDeps) {
	return newTestAppWithPolicy(nil)
}

func newTestAppWithPolicy(policy FetchPolicy) (*Application, testDeps) {
	deps := testDeps{
		flights:   newMemoryFlightStore(),
		mapData:   newMemoryMapDataStore(),
		positions: newMemoryPositionStore(),
		queue:     &memoryFetchTaskQueue{},
		budget: &memoryUsageGuard{
			fetchingEnabled: true,
			status: UsageStatus{
				Budget:          UsageBudgetStatus{Currency: "USD", SoftStopThreshold: 4},
				FetchingEnabled: true,
			},
		},
	}
	return New(Config{
		Flights:     deps.flights,
		MapData:     deps.mapData,
		Positions:   deps.positions,
		FetchTasks:  deps.queue,
		UsageGuard:  deps.budget,
		FetchPolicy: policy,
	}), deps
}

func testFlight(id domain.FlightID, ident string) domain.Flight {
	faFlightID := domain.FAFlightID("fa_1")
	provisionalID := domain.ProvisionalFlightLegID("sched_1")
	return domain.Flight{
		FlightID:               id,
		FlightIDType:           domain.FlightIDTypeInternal,
		InternalFlightLegID:    ptrDomainID(id),
		ProvisionalFlightLegID: &provisionalID,
		FAFlightID:             &faFlightID,
		Ident:                  ident,
		Origin:                 domain.Airport{Code: "RJTT"},
		Destination:            domain.Airport{Code: "KJFK"},
		Status:                 "En Route",
		Times:                  domain.FlightTimes{},
		UpdatedAt:              "2026-04-29T00:00:00Z",
	}
}

func testPosition(flightID domain.FlightID) domain.FlightPosition {
	return domain.FlightPosition{
		FlightID:         flightID,
		Latitude:         45.123,
		Longitude:        160.456,
		Timestamp:        "2026-04-29T00:00:00Z",
		Source:           domain.PositionSourceFlightAwarePosition,
		AltitudeFeet:     ptrInt(37000),
		GroundspeedKnots: ptrInt(488),
		HeadingDegrees:   ptrInt(275),
	}
}

func ptrDomainID(id domain.FlightID) *domain.InternalFlightLegID {
	value := domain.InternalFlightLegID(id)
	return &value
}

func ptrInt(value int) *int {
	return &value
}
