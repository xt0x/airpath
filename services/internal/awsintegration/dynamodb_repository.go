package awsintegration

import (
	"context"
	"encoding/json"

	"airpath/services/internal/application"
	"airpath/services/internal/domain"
)

type DynamoDBTables struct {
	Flights         string
	FlightLookup    string
	FlightPositions string
	UsageBudget     string
}

type DynamoDBRepository struct {
	client DynamoDBClient
	tables DynamoDBTables
}

func NewDynamoDBRepository(client DynamoDBClient, tables DynamoDBTables) *DynamoDBRepository {
	return &DynamoDBRepository{client: client, tables: tables}
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

func (r *DynamoDBRepository) PutPosition(ctx context.Context, position domain.FlightPosition) error {
	return r.client.PutItem(ctx, r.tables.FlightPositions, map[string]any{
		"flightId":  position.FlightID,
		"timestamp": position.Timestamp,
		"body":      mustJSON(position),
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
	return r.client.PutItem(ctx, r.tables.UsageBudget, map[string]any{
		"budgetScope": scope,
		"body":        mustJSON(status),
	})
}

func (r *DynamoDBRepository) GetUsageStatus(ctx context.Context) (application.UsageStatus, error) {
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
	return status, nil
}

func (r *DynamoDBRepository) FetchingAllowed(ctx context.Context) (bool, error) {
	status, err := r.GetUsageStatus(ctx)
	if err != nil {
		if err == application.ErrNotFound {
			return true, nil
		}
		return false, err
	}
	return status.FetchingEnabled && !status.Budget.Stopped, nil
}

func cacheFor(hit bool) application.CacheMetadata {
	if !hit {
		return application.CacheMetadata{Freshness: application.CacheFreshnessMiss, Source: application.CacheSourceCache}
	}
	return application.CacheMetadata{Freshness: application.CacheFreshnessFresh, Source: application.CacheSourceCache}
}

func mustJSON(value any) string {
	body, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(body)
}
