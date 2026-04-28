package awsintegration

import (
	"context"
	"encoding/json"
	"errors"

	"airpath/services/internal/application"
	"airpath/services/internal/domain"
	"airpath/services/internal/flightaware"
)

type DynamoDBTables struct {
	Flights         string
	FlightLookup    string
	FlightPositions string
	UsageBudget     string
}

type DynamoDBRepository struct {
	client     DynamoDBClient
	tables     DynamoDBTables
	usageScope application.UsageBudgetScope
}

func NewDynamoDBRepository(client DynamoDBClient, tables DynamoDBTables) *DynamoDBRepository {
	return &DynamoDBRepository{client: client, tables: tables}
}

func NewScopedDynamoDBRepository(client DynamoDBClient, tables DynamoDBTables, usageScope application.UsageBudgetScope) *DynamoDBRepository {
	return &DynamoDBRepository{client: client, tables: tables, usageScope: usageScope}
}

func (r *DynamoDBRepository) PutFlight(ctx context.Context, flight domain.Flight) error {
	return r.client.PutItem(ctx, r.tables.Flights, map[string]any{
		"flightId": flight.FlightID,
		"body":     mustJSON(flight),
	})
}

func (r *DynamoDBRepository) PutFlightLookup(ctx context.Context, lookupKey string, flightID domain.FlightID) error {
	return r.client.PutItem(ctx, r.tables.FlightLookup, map[string]any{
		"lookupKey": lookupKey + "#" + string(flightID),
		"flightId":  flightID,
	})
}

func (r *DynamoDBRepository) SearchByIdent(ctx context.Context, ident string) ([]domain.Flight, application.CacheMetadata, error) {
	lookupRows, err := r.client.QueryByPrefix(ctx, r.tables.FlightLookup, "lookupKey", "ident#"+ident+"#")
	if err != nil {
		return nil, application.CacheMetadata{}, err
	}
	flights := make([]domain.Flight, 0, len(lookupRows))
	for _, row := range lookupRows {
		flight, _, err := r.GetFlight(ctx, domain.FlightID(stringValue(row["flightId"])))
		if err != nil {
			return nil, application.CacheMetadata{}, err
		}
		flights = append(flights, flight)
	}
	return flights, cacheFor(len(flights) > 0), nil
}

func (r *DynamoDBRepository) GetFlight(ctx context.Context, flightID domain.FlightID) (domain.Flight, application.CacheMetadata, error) {
	item, ok, err := r.client.GetItem(ctx, r.tables.Flights, "flightId", string(flightID))
	if err != nil {
		return domain.Flight{}, application.CacheMetadata{}, err
	}
	if !ok {
		return domain.Flight{}, application.CacheMetadata{Freshness: application.CacheFreshnessMiss, Source: application.CacheSourceCache}, application.ErrNotFound
	}
	var flight domain.Flight
	if err := json.Unmarshal([]byte(stringValue(item["body"])), &flight); err != nil {
		return domain.Flight{}, application.CacheMetadata{}, err
	}
	return flight, cacheFor(true), nil
}

func (r *DynamoDBRepository) ListPollableFlights(ctx context.Context, now string, limit int) ([]domain.Flight, error) {
	items, err := r.client.QueryByPrefix(ctx, r.tables.Flights, "flightId", "")
	if err != nil {
		return nil, err
	}
	flights := make([]domain.Flight, 0, len(items))
	for _, item := range items {
		var flight domain.Flight
		if err := json.Unmarshal([]byte(stringValue(item["body"])), &flight); err != nil {
			return nil, err
		}
		if !hasDuePoll(flight, now) {
			continue
		}
		flights = append(flights, flight)
		if limit > 0 && len(flights) >= limit {
			break
		}
	}
	return flights, nil
}

func (r *DynamoDBRepository) UpdatePollSchedule(ctx context.Context, flight domain.Flight) error {
	return r.PutFlight(ctx, flight)
}

func (r *DynamoDBRepository) AcquireFetchLease(ctx context.Context, flightID domain.FlightID, owner string, leaseUntil string, now string) (domain.Flight, bool, error) {
	flight, _, err := r.GetFlight(ctx, flightID)
	if err != nil {
		return domain.Flight{}, false, err
	}
	if flight.FetchLeaseUntil != nil && *flight.FetchLeaseUntil > now && (flight.FetchOwner == nil || *flight.FetchOwner != owner) {
		return flight, false, nil
	}
	flight.FetchOwner = &owner
	flight.FetchLeaseUntil = &leaseUntil
	flight.UpdatedAt = now
	if err := r.PutFlight(ctx, flight); err != nil {
		return domain.Flight{}, false, err
	}
	return flight, true, nil
}

func (r *DynamoDBRepository) UpdateFetchedFlight(ctx context.Context, flight domain.Flight) error {
	return r.PutFlight(ctx, flight)
}

func (r *DynamoDBRepository) ReleaseFetchLease(ctx context.Context, flightID domain.FlightID, owner string, now string) error {
	flight, _, err := r.GetFlight(ctx, flightID)
	if err != nil {
		return err
	}
	if flight.FetchOwner == nil || *flight.FetchOwner != owner {
		return nil
	}
	flight.FetchOwner = nil
	flight.FetchLeaseUntil = nil
	flight.UpdatedAt = now
	return r.PutFlight(ctx, flight)
}

func (r *DynamoDBRepository) PutPosition(ctx context.Context, position domain.FlightPosition) error {
	return r.client.PutItem(ctx, r.tables.FlightPositions, map[string]any{
		"flightId":  position.FlightID,
		"timestamp": position.Timestamp,
		"body":      mustJSON(position),
	})
}

func (r *DynamoDBRepository) AppendPosition(ctx context.Context, position domain.FlightPosition) error {
	return r.PutPosition(ctx, position)
}

func (r *DynamoDBRepository) PutFlightAwarePosition(ctx context.Context, flightID domain.FlightID, response flightaware.PositionResponse) error {
	if response.Latitude == nil || response.Longitude == nil || response.Timestamp == "" {
		return application.ErrValidation
	}
	return r.PutPosition(ctx, domain.FlightPosition{
		FlightID:  flightID,
		Latitude:  *response.Latitude,
		Longitude: *response.Longitude,
		Timestamp: response.Timestamp,
		Source:    domain.PositionSourceFlightAwarePosition,
	})
}

func (r *DynamoDBRepository) GetLatestPosition(ctx context.Context, flightID domain.FlightID) (*domain.FlightPosition, application.CacheMetadata, error) {
	items, err := r.client.QueryByPrefix(ctx, r.tables.FlightPositions, "flightId", string(flightID))
	if err != nil {
		return nil, application.CacheMetadata{}, err
	}
	if len(items) == 0 {
		return nil, application.CacheMetadata{Freshness: application.CacheFreshnessMiss, Source: application.CacheSourceCache}, nil
	}
	var position domain.FlightPosition
	if err := json.Unmarshal([]byte(stringValue(items[len(items)-1]["body"])), &position); err != nil {
		return nil, application.CacheMetadata{}, err
	}
	return &position, cacheFor(true), nil
}

func (r *DynamoDBRepository) PutUsageStatus(ctx context.Context, scope string, status application.UsageStatus) error {
	status = application.NormalizeUsageStatus(status)
	return r.client.PutItem(ctx, r.tables.UsageBudget, map[string]any{
		"budgetScope": scope,
		"body":        mustJSON(status),
	})
}

func (r *DynamoDBRepository) PutMonthlyUsageStatus(ctx context.Context, scope application.UsageBudgetScope, status application.UsageStatus) error {
	status.Budget.Environment = scope.Environment
	status.Budget.Month = scope.Month
	return r.PutUsageStatus(ctx, scope.Key(), status)
}

func (r *DynamoDBRepository) GetUsageStatus(ctx context.Context) (application.UsageStatus, error) {
	if r.usageScope.Key() != "" {
		return r.GetMonthlyUsageStatus(ctx, r.usageScope)
	}
	items, err := r.client.QueryByPrefix(ctx, r.tables.UsageBudget, "budgetScope", "")
	if err != nil {
		return application.UsageStatus{}, err
	}
	if len(items) == 0 {
		return application.UsageStatus{}, application.ErrNotFound
	}
	var status application.UsageStatus
	if err := json.Unmarshal([]byte(stringValue(items[len(items)-1]["body"])), &status); err != nil {
		return application.UsageStatus{}, err
	}
	return application.NormalizeUsageStatus(status), nil
}

func (r *DynamoDBRepository) GetMonthlyUsageStatus(ctx context.Context, scope application.UsageBudgetScope) (application.UsageStatus, error) {
	item, ok, err := r.client.GetItem(ctx, r.tables.UsageBudget, "budgetScope", scope.Key())
	if err != nil {
		return application.UsageStatus{}, err
	}
	if !ok {
		return application.UsageStatus{}, application.ErrNotFound
	}
	var status application.UsageStatus
	if err := json.Unmarshal([]byte(stringValue(item["body"])), &status); err != nil {
		return application.UsageStatus{}, err
	}
	status.Budget.Environment = scope.Environment
	status.Budget.Month = scope.Month
	return application.NormalizeUsageStatus(status), nil
}

func (r *DynamoDBRepository) FetchingAllowed(ctx context.Context) (bool, error) {
	status, err := r.GetUsageStatus(ctx)
	if err != nil {
		if err == application.ErrNotFound {
			return true, nil
		}
		return false, err
	}
	status = application.NormalizeUsageStatus(status)
	return status.FetchingEnabled && !status.Budget.Stopped, nil
}

func (r *DynamoDBRepository) RecordFlightAwareCall(ctx context.Context, record flightaware.UsageCallRecord) error {
	if record.Phase != flightaware.UsageRecordPhaseBefore {
		return nil
	}
	scope := r.usageScope
	if scope.Key() == "" {
		return application.ErrValidation
	}
	status, err := r.GetMonthlyUsageStatus(ctx, scope)
	if err != nil {
		if !errors.Is(err, application.ErrNotFound) {
			return err
		}
		status = application.UsageStatus{
			Budget: application.UsageBudgetStatus{
				Environment:       scope.Environment,
				Month:             scope.Month,
				Currency:          "USD",
				SoftStopThreshold: application.DefaultSoftStopThresholdUSD,
			},
			FetchingEnabled: true,
		}
	}
	status.Budget.EstimatedMonthToDateCost += record.EstimatedCostUSD
	return r.PutMonthlyUsageStatus(ctx, scope, status)
}

func (r *DynamoDBRepository) ReconcileAccountUsage(ctx context.Context, scope application.UsageBudgetScope, response flightaware.UsageResponse) error {
	if scope.Key() == "" {
		return application.ErrValidation
	}
	status, err := r.GetMonthlyUsageStatus(ctx, scope)
	if err != nil {
		if !errors.Is(err, application.ErrNotFound) {
			return err
		}
		status = application.UsageStatus{
			Budget: application.UsageBudgetStatus{
				Environment:       scope.Environment,
				Month:             scope.Month,
				Currency:          response.Currency,
				SoftStopThreshold: application.DefaultSoftStopThresholdUSD,
			},
			FetchingEnabled: true,
		}
	}
	if response.Currency != "" {
		status.Budget.Currency = response.Currency
	}
	if response.MonthToDate.EstimatedCostUSD > status.Budget.EstimatedMonthToDateCost {
		status.Budget.EstimatedMonthToDateCost = response.MonthToDate.EstimatedCostUSD
	}
	return r.PutMonthlyUsageStatus(ctx, scope, status)
}

func cacheFor(hit bool) application.CacheMetadata {
	if !hit {
		return application.CacheMetadata{Freshness: application.CacheFreshnessMiss, Source: application.CacheSourceCache}
	}
	return application.CacheMetadata{Freshness: application.CacheFreshnessFresh, Source: application.CacheSourceCache}
}

func hasDuePoll(flight domain.Flight, now string) bool {
	return pollTimeDue(flight.NextPositionPollAt, now) ||
		pollTimeDue(flight.NextRoutePollAt, now) ||
		pollTimeDue(flight.NextTrackPollAt, now)
}

func pollTimeDue(value *domain.ISODateTimeString, now string) bool {
	return value != nil && *value <= now
}

func mustJSON(value any) string {
	body, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(body)
}
