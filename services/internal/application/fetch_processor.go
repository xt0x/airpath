package application

import (
	"context"
	"errors"
	"math"
	"sort"
	"time"

	"airpath/services/internal/domain"
)

type FetchFlightStore interface {
	GetFlight(context.Context, domain.FlightID) (domain.Flight, CacheMetadata, error)
	AcquireFetchLease(context.Context, domain.FlightID, string, string, string) (domain.Flight, bool, error)
	UpdateFetchedFlight(context.Context, domain.Flight, string, string, string) (bool, error)
	ReleaseFetchLease(context.Context, domain.FlightID, string, string, string) error
	PutFlight(context.Context, domain.Flight) error
	PutFlightLookup(context.Context, string, domain.FlightID) error
}

type SummaryFlightWriter interface {
	PutFlightSummary(context.Context, *domain.Flight, domain.Flight) (bool, error)
}

type PositionHistoryStore interface {
	AppendPosition(context.Context, domain.FlightPosition) error
}

type LeaseGuardedPositionHistoryStore interface {
	AppendPositionIfLeaseHeld(context.Context, domain.FlightPosition, string, string) (bool, error)
}

type FetchedPositionWriter interface {
	UpdateFetchedFlightWithPosition(context.Context, domain.Flight, domain.FlightPosition, string, string, string) (bool, error)
}

type FetchedTrackWriter interface {
	UpdateFetchedFlightWithTrackPositions(context.Context, domain.Flight, []domain.FlightPosition, string, string, string) (bool, error)
}

type FlightArtifactStore interface {
	StoreRouteArtifact(context.Context, domain.FlightID, ExternalRouteResponse) (string, error)
	StoreTrackArtifact(context.Context, domain.FlightID, ExternalTrackResponse) (string, error)
}

type FetchFlightAwareClient interface {
	SearchFlights(context.Context, string) (ExternalSearchFlightsResponse, error)
	GetFlightRoute(context.Context, string) (ExternalRouteResponse, error)
	GetFlightPosition(context.Context, string) (ExternalPositionResponse, error)
	GetFlightTrack(context.Context, string) (ExternalTrackResponse, error)
}

type FetchLeaseChecker interface {
	FetchLeaseHeld(context.Context, domain.FlightID, string, string) (bool, error)
}

type FetchProcessorConfig struct {
	Flights       FetchFlightStore
	Positions     PositionHistoryStore
	Artifacts     FlightArtifactStore
	FlightAware   FetchFlightAwareClient
	UsageGuard    UsageGuard
	FetchPolicy   FetchPolicy
	Diagnostics   FetchTaskDiagnosticStore
	Schedule      PollSchedulePolicy
	LeaseDuration time.Duration
}

type FetchProcessor struct {
	flights       FetchFlightStore
	positions     PositionHistoryStore
	artifacts     FlightArtifactStore
	flightAware   FetchFlightAwareClient
	usageGuard    UsageGuard
	fetchPolicy   FetchPolicy
	diagnostics   FetchTaskDiagnosticStore
	schedule      PollSchedulePolicy
	leaseDuration time.Duration
}

type ProcessFetchInput struct {
	Now      time.Time
	WorkerID string
}

type ProcessFetchResult struct {
	ExternalFetchAttempted bool
	Skipped                bool
	SkipReason             string
	UpdatedSummaryCount    int
	UpdatedPositionCount   int
	UpdatedRoute           bool
	UpdatedTrack           bool
}

func NewFetchProcessor(config FetchProcessorConfig) *FetchProcessor {
	mustProvide("FetchProcessorConfig.Flights", config.Flights)
	mustProvide("FetchProcessorConfig.Positions", config.Positions)
	mustProvide("FetchProcessorConfig.Artifacts", config.Artifacts)
	mustProvide("FetchProcessorConfig.FlightAware", config.FlightAware)
	leaseDuration := config.LeaseDuration
	if leaseDuration == 0 {
		leaseDuration = defaultFetchLeaseDuration
	}
	return &FetchProcessor{
		flights:       config.Flights,
		positions:     config.Positions,
		artifacts:     config.Artifacts,
		flightAware:   config.FlightAware,
		usageGuard:    config.UsageGuard,
		fetchPolicy:   config.FetchPolicy,
		diagnostics:   config.Diagnostics,
		schedule:      config.Schedule.withDefaults(),
		leaseDuration: leaseDuration,
	}
}

func (p *FetchProcessor) Process(ctx context.Context, task FetchTask, input ProcessFetchInput) (ProcessFetchResult, error) {
	if result, stopped, err := p.skipIfFetchingStopped(ctx); err != nil {
		_ = p.recordDiagnostic(ctx, task, input.Now, err)
		return ProcessFetchResult{}, err
	} else if stopped {
		return result, nil
	}

	if !IsValidFetchReason(task.Reason) {
		err := ErrValidation
		_ = p.recordDiagnostic(ctx, task, input.Now, err)
		return ProcessFetchResult{}, err
	}

	if !IsValidFetchTaskType(task.TaskType) {
		err := ErrValidation
		_ = p.recordDiagnostic(ctx, task, input.Now, err)
		return ProcessFetchResult{}, err
	}

	if !fetchTaskReasonAllowedForType(task.TaskType, task.Reason) {
		err := ErrValidation
		_ = p.recordDiagnostic(ctx, task, input.Now, err)
		return ProcessFetchResult{}, err
	}

	if result, blocked, err := p.skipIfFetchPolicyBlocks(ctx, task); err != nil {
		_ = p.recordDiagnostic(ctx, task, input.Now, err)
		return ProcessFetchResult{}, err
	} else if blocked {
		return result, nil
	}

	if task.TaskType == FetchTaskSummary {
		result, err := p.processSummary(ctx, task, input.Now)
		if err != nil {
			_ = p.recordDiagnostic(ctx, task, input.Now, err)
		}
		return result, err
	}

	workerID := input.WorkerID
	if workerID == "" {
		workerID = "fetcher"
	}
	nowString := formatISO(input.Now)
	leaseUntil := formatISO(input.Now.Add(p.leaseDuration))
	// Non-summary fetches publish multiple cache artifacts, so the worker must hold
	// the flight lease before calling the upstream API and again when committing.
	flight, leased, err := p.flights.AcquireFetchLease(ctx, task.FlightID, workerID, leaseUntil, nowString)
	if err != nil {
		return ProcessFetchResult{}, err
	}
	if !leased {
		return ProcessFetchResult{Skipped: true, SkipReason: "lease_held"}, nil
	}
	defer func() {
		_ = p.flights.ReleaseFetchLease(ctx, task.FlightID, workerID, leaseUntil, nowString)
	}()

	if flight.FAFlightID == nil || *flight.FAFlightID == "" {
		err := ErrValidation
		_ = p.recordDiagnostic(ctx, task, input.Now, err)
		return ProcessFetchResult{}, err
	}

	result := ProcessFetchResult{ExternalFetchAttempted: true}
	switch task.TaskType {
	case FetchTaskPosition:
		result, err = p.processPosition(ctx, flight, input.Now, workerID, leaseUntil, result)
	case FetchTaskRoute:
		result, err = p.processRoute(ctx, flight, input.Now, workerID, leaseUntil, result)
	case FetchTaskTrack, FetchTaskFinalTrack:
		result, err = p.processTrack(ctx, flight, input.Now, workerID, leaseUntil, result)
	}
	if err != nil {
		_ = p.recordDiagnostic(ctx, task, input.Now, err)
	}
	return result, err
}

func fetchTaskReasonAllowedForType(taskType FetchTaskType, reason FetchReason) bool {
	switch taskType {
	case FetchTaskSummary:
		return reason == FetchReasonSearchResultSeed
	case FetchTaskPosition, FetchTaskRoute, FetchTaskTrack:
		switch reason {
		case FetchReasonUserOpenedDetail, FetchReasonUserManualRefresh, FetchReasonLowFrequencyPoll:
			return true
		default:
			return false
		}
	case FetchTaskFinalTrack:
		switch reason {
		case FetchReasonUserOpenedDetail, FetchReasonUserManualRefresh, FetchReasonArrivalFinalization:
			return true
		default:
			return false
		}
	default:
		return false
	}
}

func (p *FetchProcessor) skipIfFetchingStopped(ctx context.Context) (ProcessFetchResult, bool, error) {
	if p.usageGuard == nil {
		return ProcessFetchResult{}, false, nil
	}
	allowed, err := p.usageGuard.FetchingAllowed(ctx)
	if err != nil {
		return ProcessFetchResult{}, false, err
	}
	if allowed {
		return ProcessFetchResult{}, false, nil
	}
	return ProcessFetchResult{Skipped: true, SkipReason: "budget_stopped"}, true, nil
}

func (p *FetchProcessor) skipIfFetchPolicyBlocks(ctx context.Context, task FetchTask) (ProcessFetchResult, bool, error) {
	if p.fetchPolicy == nil {
		return ProcessFetchResult{}, false, nil
	}
	allowed, err := p.fetchPolicy.FetchTaskAllowed(ctx, task.TaskType, task.Reason)
	if err != nil {
		return ProcessFetchResult{}, false, err
	}
	if allowed {
		return ProcessFetchResult{}, false, nil
	}
	return ProcessFetchResult{Skipped: true, SkipReason: "fetch_disabled"}, true, nil
}

func (p *FetchProcessor) processSummary(ctx context.Context, task FetchTask, now time.Time) (ProcessFetchResult, error) {
	ident := string(task.FlightID)
	if ident == "" {
		return ProcessFetchResult{}, ErrValidation
	}
	response, err := p.flightAware.SearchFlights(ctx, ident)
	if err != nil {
		return ProcessFetchResult{ExternalFetchAttempted: true}, err
	}
	result := ProcessFetchResult{ExternalFetchAttempted: true}
	for index, summary := range response.Flights {
		flight, ok := flightFromExternalSummary(summary, now, index)
		if !ok {
			_ = p.recordDiagnostic(ctx, task, now, ErrValidation)
			continue
		}
		var observed *domain.Flight
		existing, _, err := p.flights.GetFlight(ctx, flight.FlightID)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return result, err
		}
		if err == nil {
			observedFlight := existing
			observed = &observedFlight
			flight = mergeFetchedSummary(existing, flight)
		}
		stored, err := p.putSummaryFlight(ctx, observed, flight)
		if err != nil {
			return result, err
		}
		if !stored {
			continue
		}
		if err := p.flights.PutFlightLookup(ctx, "ident#"+ident, flight.FlightID); err != nil {
			return result, err
		}
		if summary.Ident != "" && summary.Ident != ident {
			if err := p.flights.PutFlightLookup(ctx, "ident#"+summary.Ident, flight.FlightID); err != nil {
				return result, err
			}
		}
		result.UpdatedSummaryCount++
	}
	return result, nil
}

func (p *FetchProcessor) putSummaryFlight(ctx context.Context, observed *domain.Flight, flight domain.Flight) (bool, error) {
	if writer, ok := p.flights.(SummaryFlightWriter); ok {
		return writer.PutFlightSummary(ctx, observed, flight)
	}
	if err := p.flights.PutFlight(ctx, flight); err != nil {
		return false, err
	}
	return true, nil
}

func mergeFetchedSummary(existing domain.Flight, summary domain.Flight) domain.Flight {
	// Summary search refreshes public flight facts only. Operational fields are
	// preserved so a newer lease, artifact pointer, or poll schedule is not rolled
	// back by an older search result.
	summary.PlannedRouteS3Key = existing.PlannedRouteS3Key
	summary.ActualTrackS3Key = existing.ActualTrackS3Key
	summary.LatestPositionTimestamp = existing.LatestPositionTimestamp
	summary.LatestPositionSource = existing.LatestPositionSource
	summary.PollState = existing.PollState
	summary.FetchLeaseUntil = existing.FetchLeaseUntil
	summary.FetchOwner = existing.FetchOwner
	summary.NextSummaryPollAt = existing.NextSummaryPollAt
	summary.NextPositionPollAt = existing.NextPositionPollAt
	summary.NextTrackPollAt = existing.NextTrackPollAt
	summary.NextRoutePollAt = existing.NextRoutePollAt
	summary.IdleSince = existing.IdleSince
	summary.TTL = existing.TTL
	return summary
}

func (p *FetchProcessor) processPosition(ctx context.Context, flight domain.Flight, now time.Time, owner string, leaseUntil string, result ProcessFetchResult) (ProcessFetchResult, error) {
	response, err := p.flightAware.GetFlightPosition(ctx, string(*flight.FAFlightID))
	if err != nil {
		return result, err
	}
	position, ok := positionFromFlightAware(flight, response)
	if !ok {
		return result, ErrValidation
	}
	flight.LatestPositionTimestamp = &position.Timestamp
	flight.LatestPositionSource = &position.Source
	flight = p.schedule.Compute(flight, now, PollActivitySignal{ActiveViewer: true})
	updated, err := p.updateFetchedFlightWithPosition(ctx, flight, position, owner, leaseUntil, formatISO(now))
	if err != nil {
		return result, err
	}
	if !updated {
		result.Skipped = true
		result.SkipReason = "lease_lost"
		return result, nil
	}
	result.UpdatedPositionCount = 1
	return result, nil
}

func (p *FetchProcessor) processRoute(ctx context.Context, flight domain.Flight, now time.Time, owner string, leaseUntil string, result ProcessFetchResult) (ProcessFetchResult, error) {
	response, err := p.flightAware.GetFlightRoute(ctx, string(*flight.FAFlightID))
	if err != nil {
		return result, err
	}
	response = sanitizedRouteResponse(response)
	if !routeHasEnoughCoordinates(response) {
		result.Skipped = true
		result.SkipReason = "route_unavailable"
		return result, nil
	}
	key, err := p.artifacts.StoreRouteArtifact(ctx, flight.FlightID, response)
	if err != nil {
		return result, err
	}
	flight.PlannedRouteS3Key = &key
	flight = p.schedule.Compute(flight, now, PollActivitySignal{ActiveViewer: true})
	updated, err := p.flights.UpdateFetchedFlight(ctx, flight, owner, leaseUntil, formatISO(now))
	if err != nil {
		return result, err
	}
	if !updated {
		result.Skipped = true
		result.SkipReason = "lease_lost"
		return result, nil
	}
	result.UpdatedRoute = true
	return result, nil
}

func (p *FetchProcessor) processTrack(ctx context.Context, flight domain.Flight, now time.Time, owner string, leaseUntil string, result ProcessFetchResult) (ProcessFetchResult, error) {
	response, err := p.flightAware.GetFlightTrack(ctx, string(*flight.FAFlightID))
	if err != nil {
		return result, err
	}
	positions := positionsFromFlightAwareTrack(flight, response)
	if len(positions) < 2 {
		result.Skipped = true
		result.SkipReason = "track_unavailable"
		return result, nil
	}
	key, err := p.artifacts.StoreTrackArtifact(ctx, flight.FlightID, trackResponseFromPositions(positions))
	if err != nil {
		return result, err
	}
	if len(positions) > 0 {
		lastPosition := positions[len(positions)-1]
		flight.LatestPositionTimestamp = &lastPosition.Timestamp
		flight.LatestPositionSource = &lastPosition.Source
	}
	flight.ActualTrackS3Key = &key
	flight = p.schedule.Compute(flight, now, PollActivitySignal{ActiveViewer: true})
	updated, err := p.updateFetchedFlightWithTrackPositions(ctx, flight, positions, owner, leaseUntil, formatISO(now))
	if err != nil {
		return result, err
	}
	if !updated {
		result.UpdatedPositionCount = 0
		result.Skipped = true
		result.SkipReason = "lease_lost"
		return result, nil
	}
	result.UpdatedPositionCount = len(positions)
	result.UpdatedTrack = true
	return result, nil
}

func (p *FetchProcessor) fetchLeaseHeld(ctx context.Context, flightID domain.FlightID, owner string, leaseUntil string) (bool, error) {
	checker, ok := p.flights.(FetchLeaseChecker)
	if !ok {
		return true, nil
	}
	return checker.FetchLeaseHeld(ctx, flightID, owner, leaseUntil)
}

func (p *FetchProcessor) appendPositionIfLeaseHeld(ctx context.Context, position domain.FlightPosition, owner string, leaseUntil string) (bool, error) {
	if guarded, ok := p.positions.(LeaseGuardedPositionHistoryStore); ok {
		return guarded.AppendPositionIfLeaseHeld(ctx, position, owner, leaseUntil)
	}
	held, err := p.fetchLeaseHeld(ctx, position.FlightID, owner, leaseUntil)
	if err != nil {
		return false, err
	}
	if !held {
		return false, nil
	}
	if err := p.positions.AppendPosition(ctx, position); err != nil {
		return false, err
	}
	return true, nil
}

func (p *FetchProcessor) updateFetchedFlightWithTrackPositions(ctx context.Context, flight domain.Flight, positions []domain.FlightPosition, owner string, leaseUntil string, now string) (bool, error) {
	if writer, ok := p.flights.(FetchedTrackWriter); ok {
		return writer.UpdateFetchedFlightWithTrackPositions(ctx, flight, positions, owner, leaseUntil, now)
	}
	return false, ErrAtomicTrackWriterRequired
}

func (p *FetchProcessor) updateFetchedFlightWithPosition(ctx context.Context, flight domain.Flight, position domain.FlightPosition, owner string, leaseUntil string, now string) (bool, error) {
	if writer, ok := p.flights.(FetchedPositionWriter); ok {
		return writer.UpdateFetchedFlightWithPosition(ctx, flight, position, owner, leaseUntil, now)
	}
	return false, ErrAtomicPositionWriterRequired
}

func positionsFromFlightAwareTrack(flight domain.Flight, response ExternalTrackResponse) []domain.FlightPosition {
	positions := make([]domain.FlightPosition, 0, len(response.Positions))
	seen := map[string]int{}
	for _, point := range response.Positions {
		if !validTrackPoint(point) {
			continue
		}
		position := domain.FlightPosition{
			FlightID:             flight.FlightID,
			FAFlightID:           flight.FAFlightID,
			Latitude:             point.Latitude,
			Longitude:            point.Longitude,
			AltitudeHundredsFeet: point.AltitudeHundredsFeet,
			AltitudeChange:       point.AltitudeChange,
			GroundspeedKnots:     point.GroundspeedKnots,
			HeadingDegrees:       point.HeadingDegrees,
			Timestamp:            point.Timestamp,
			UpdateType:           point.UpdateType,
			Source:               domain.PositionSourceFlightAwareTrack,
		}
		metrics := domain.NormalizeFlightPositionMetrics(domain.FlightPositionMetricsInput{
			AltitudeHundredsFeet: point.AltitudeHundredsFeet,
			GroundspeedKnots:     point.GroundspeedKnots,
			HeadingDegrees:       point.HeadingDegrees,
		})
		position.AltitudeFeet = metrics.AltitudeFeet
		position.HeadingDegrees = metrics.HeadingDegrees
		// FlightAware can repeat a timestamp inside one track response. Keep the
		// latest accepted point for that timestamp so the persisted range key is
		// deterministic and the artifact matches history.
		if index, ok := seen[position.Timestamp]; ok {
			positions[index] = position
			continue
		}
		seen[position.Timestamp] = len(positions)
		positions = append(positions, position)
	}
	sort.SliceStable(positions, func(left, right int) bool {
		leftTime, leftErr := time.Parse(time.RFC3339, positions[left].Timestamp)
		rightTime, rightErr := time.Parse(time.RFC3339, positions[right].Timestamp)
		if leftErr == nil && rightErr == nil {
			return leftTime.Before(rightTime)
		}
		return positions[left].Timestamp < positions[right].Timestamp
	})
	return positions
}

func trackResponseFromPositions(positions []domain.FlightPosition) ExternalTrackResponse {
	response := ExternalTrackResponse{Positions: make([]ExternalTrackPoint, 0, len(positions))}
	for _, position := range positions {
		response.Positions = append(response.Positions, ExternalTrackPoint{
			Latitude:             position.Latitude,
			Longitude:            position.Longitude,
			AltitudeHundredsFeet: position.AltitudeHundredsFeet,
			AltitudeChange:       position.AltitudeChange,
			GroundspeedKnots:     position.GroundspeedKnots,
			HeadingDegrees:       position.HeadingDegrees,
			Timestamp:            position.Timestamp,
			UpdateType:           position.UpdateType,
		})
	}
	return response
}

func validTrackPoint(point ExternalTrackPoint) bool {
	if _, err := time.Parse(time.RFC3339, point.Timestamp); err != nil {
		return false
	}
	return validCoordinate(point.Latitude, point.Longitude)
}

func routeHasEnoughCoordinates(response ExternalRouteResponse) bool {
	return len(validRouteFixes(response.Fixes)) >= 2
}

func sanitizedRouteResponse(response ExternalRouteResponse) ExternalRouteResponse {
	response.Fixes = validRouteFixes(response.Fixes)
	return response
}

func validRouteFixes(fixes []ExternalRouteFix) []ExternalRouteFix {
	valid := make([]ExternalRouteFix, 0, len(fixes))
	for _, fix := range fixes {
		if fix.Latitude == nil || fix.Longitude == nil {
			continue
		}
		if !validCoordinate(*fix.Latitude, *fix.Longitude) {
			continue
		}
		valid = append(valid, fix)
	}
	return valid
}

func validCoordinate(latitude float64, longitude float64) bool {
	return !math.IsNaN(latitude) && !math.IsInf(latitude, 0) &&
		!math.IsNaN(longitude) && !math.IsInf(longitude, 0) &&
		latitude >= -90 && latitude <= 90 &&
		longitude >= -180 && longitude <= 180
}

func flightFromExternalSummary(summary ExternalFlightSummary, now time.Time, index int) (domain.Flight, bool) {
	if summary.Ident == "" || summary.Origin == "" || summary.Destination == "" {
		return domain.Flight{}, false
	}
	scheduledOut := ""
	if summary.ScheduledOut != nil {
		scheduledOut = *summary.ScheduledOut
	}
	legIndex := index
	flight := domain.Flight{
		Ident:           summary.Ident,
		IdentIATA:       summary.IdentIATA,
		AircraftType:    summary.AircraftType,
		Registration:    summary.Registration,
		Origin:          domain.Airport{Code: domain.AirportCode(summary.Origin)},
		Destination:     domain.Airport{Code: domain.AirportCode(summary.Destination)},
		LegIndex:        &legIndex,
		Status:          summary.Status,
		ProgressPercent: summary.ProgressPercent,
		Times: domain.FlightTimes{
			ScheduledOut: summary.ScheduledOut,
			EstimatedOut: summary.EstimatedOut,
			ActualOut:    summary.ActualOut,
			ScheduledOff: summary.ScheduledOff,
			EstimatedOff: summary.EstimatedOff,
			ActualOff:    summary.ActualOff,
			ScheduledOn:  summary.ScheduledOn,
			EstimatedOn:  summary.EstimatedOn,
			ActualOn:     summary.ActualOn,
			ScheduledIn:  summary.ScheduledIn,
			EstimatedIn:  summary.EstimatedIn,
			ActualIn:     summary.ActualIn,
		},
		FiledEteSeconds: summary.FiledEteSeconds,
		UpdatedAt:       formatISO(now),
	}
	if summary.FAFlightID != nil && *summary.FAFlightID != "" {
		faFlightID := domain.FAFlightID(*summary.FAFlightID)
		internalID := domain.GenerateInternalFlightLegID(domain.InternalFlightLegIDInput{
			FAFlightID:      faFlightID,
			OriginCode:      domain.AirportCode(summary.Origin),
			DestinationCode: domain.AirportCode(summary.Destination),
			ScheduledOut:    scheduledOut,
			LegIndex:        0,
		})
		flight.FlightID = domain.FlightID(internalID)
		flight.FlightIDType = domain.FlightIDTypeInternal
		flight.InternalFlightLegID = &internalID
		flight.FAFlightID = &faFlightID
		return flight, true
	}
	provisionalID := domain.GenerateProvisionalFlightLegID(domain.ProvisionalFlightLegIDInput{
		Ident:           summary.Ident,
		OriginCode:      domain.AirportCode(summary.Origin),
		DestinationCode: domain.AirportCode(summary.Destination),
		ScheduledOut:    scheduledOut,
	})
	flight.FlightID = domain.FlightID(provisionalID)
	flight.FlightIDType = domain.FlightIDTypeProvisional
	flight.ProvisionalFlightLegID = &provisionalID
	return flight, true
}

func positionFromFlightAware(flight domain.Flight, response ExternalPositionResponse) (domain.FlightPosition, bool) {
	if response.Latitude == nil || response.Longitude == nil || response.Timestamp == "" {
		return domain.FlightPosition{}, false
	}
	point := ExternalTrackPoint{Latitude: *response.Latitude, Longitude: *response.Longitude, Timestamp: response.Timestamp}
	if !validTrackPoint(point) {
		return domain.FlightPosition{}, false
	}
	metrics := domain.NormalizeFlightPositionMetrics(domain.FlightPositionMetricsInput{
		AltitudeHundredsFeet: response.AltitudeHundredsFeet,
		GroundspeedKnots:     response.GroundspeedKnots,
		HeadingDegrees:       response.HeadingDegrees,
	})
	return domain.FlightPosition{
		FlightID:             flight.FlightID,
		FAFlightID:           flight.FAFlightID,
		Latitude:             *response.Latitude,
		Longitude:            *response.Longitude,
		AltitudeHundredsFeet: response.AltitudeHundredsFeet,
		AltitudeFeet:         metrics.AltitudeFeet,
		AltitudeChange:       response.AltitudeChange,
		GroundspeedKnots:     response.GroundspeedKnots,
		HeadingDegrees:       metrics.HeadingDegrees,
		Timestamp:            response.Timestamp,
		UpdateType:           response.UpdateType,
		Source:               domain.PositionSourceFlightAwarePosition,
	}, true
}
