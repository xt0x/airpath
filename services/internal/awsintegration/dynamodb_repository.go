package awsintegration

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

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
	client     DynamoDBClient
	tables     DynamoDBTables
	usageScope application.UsageBudgetScope
	now        func() time.Time
}

const (
	pollDueIndexName          = "poll-due-index"
	pollShardActive           = "active"
	pollQueryPageLimit        = 25
	fetchTaskIdempotencyTTL   = 30 * time.Minute
	fetchTaskIdempotencyScope = "fetchTaskIdempotency"
)

func NewDynamoDBRepository(client DynamoDBClient, tables DynamoDBTables) *DynamoDBRepository {
	return &DynamoDBRepository{client: client, tables: tables, now: time.Now}
}

func NewScopedDynamoDBRepository(client DynamoDBClient, tables DynamoDBTables, usageScope application.UsageBudgetScope) *DynamoDBRepository {
	return &DynamoDBRepository{client: client, tables: tables, usageScope: usageScope, now: time.Now}
}

func (r *DynamoDBRepository) PutFlight(ctx context.Context, flight domain.Flight) error {
	return r.client.PutItem(ctx, r.tables.Flights, flightItem(flight))
}

func (r *DynamoDBRepository) PutFlightSummary(ctx context.Context, observed *domain.Flight, flight domain.Flight) (bool, error) {
	item := flightItem(flight)
	if observed == nil {
		return r.client.PutItemIfAbsent(ctx, r.tables.Flights, item, "flightId")
	}
	var err error
	item, err = r.flightItemPreservingCommittedPositionWriteToken(ctx, flight)
	if err != nil {
		return false, err
	}
	expected, absent := flightWriteConditions(*observed, *observed)
	return r.client.PutItemIfCurrentAttributesMatch(ctx, r.tables.Flights, item, expected, absent)
}

func (r *DynamoDBRepository) putFlightIfLeaseAvailable(ctx context.Context, flight domain.Flight, owner string, now string) (bool, error) {
	return r.client.PutItemIfLeaseAvailable(ctx, r.tables.Flights, flightItem(flight), owner, now)
}

func (r *DynamoDBRepository) putFlightIfLeaseOwner(ctx context.Context, flight domain.Flight, owner string, leaseUntil string) (bool, error) {
	return r.putFlightIfLeaseOwnerAt(ctx, flight, owner, leaseUntil, "")
}

func (r *DynamoDBRepository) putFlightIfLeaseOwnerAt(ctx context.Context, flight domain.Flight, owner string, leaseUntil string, now string) (bool, error) {
	item, err := r.flightItemPreservingCommittedPositionWriteToken(ctx, flight)
	if err != nil {
		return false, err
	}
	return r.client.PutItemIfLeaseOwner(ctx, r.tables.Flights, item, owner, leaseUntil, now)
}

func flightItem(flight domain.Flight) map[string]any {
	bodyFlight := flight
	// Lease attributes live outside the serialized body so workers can acquire or
	// release leases without rewriting cached flight facts from a stale snapshot.
	bodyFlight.FetchLeaseUntil = nil
	bodyFlight.FetchOwner = nil
	item := map[string]any{
		"flightId": flight.FlightID,
		"body":     mustJSON(bodyFlight),
	}
	if flight.TTL != nil {
		item["ttl"] = *flight.TTL
	}
	if flight.FetchLeaseUntil != nil {
		item["fetchLeaseUntil"] = *flight.FetchLeaseUntil
	}
	if flight.FetchOwner != nil {
		item["fetchOwner"] = *flight.FetchOwner
	}
	if nextPollAt := nextPollAt(flight); nextPollAt != nil {
		item["pollShard"] = pollShardActive
		item["nextPollAt"] = *nextPollAt
	}
	return item
}

func (r *DynamoDBRepository) PutFlightLookup(ctx context.Context, lookupKey string, flightID domain.FlightID) error {
	lookupType := lookupTypeFromKey(lookupKey)
	return r.client.PutItem(ctx, r.tables.FlightLookup, map[string]any{
		"lookupType": lookupType,
		"lookupKey":  lookupKey + "#" + string(flightID),
		"flightId":   flightID,
	})
}

func (r *DynamoDBRepository) ReserveFetchTaskIdempotency(ctx context.Context, idempotencyKey string) (bool, error) {
	if idempotencyKey == "" {
		return false, application.ErrValidation
	}
	now := time.Now().UTC()
	lookupKey := fetchTaskIdempotencyScope + "#" + idempotencyKey
	return r.client.PutItemIfAbsentOrExpired(ctx, r.tables.FlightLookup, map[string]any{
		"lookupType": fetchTaskIdempotencyScope,
		"lookupKey":  lookupKey,
		"flightId":   idempotencyKey,
		"ttl":        now.Add(fetchTaskIdempotencyTTL).Unix(),
	}, "lookupType", "ttl", now.Unix())
}

func (r *DynamoDBRepository) ReleaseFetchTaskIdempotency(ctx context.Context, idempotencyKey string) error {
	if idempotencyKey == "" {
		return application.ErrValidation
	}
	return r.client.DeleteItem(ctx, r.tables.FlightLookup, map[string]any{
		"lookupType": fetchTaskIdempotencyScope,
		"lookupKey":  fetchTaskIdempotencyScope + "#" + idempotencyKey,
	})
}

func (r *DynamoDBRepository) SearchByIdent(ctx context.Context, ident string) ([]domain.Flight, application.CacheMetadata, error) {
	lookupRows, err := r.client.QueryStringPrefix(ctx, r.tables.FlightLookup, "", "lookupType", "ident", "lookupKey", "ident#"+ident+"#", 0)
	if err != nil {
		return nil, application.CacheMetadata{}, err
	}
	flights := make([]domain.Flight, 0, len(lookupRows))
	for _, row := range lookupRows {
		flight, _, err := r.GetFlight(ctx, domain.FlightID(stringValue(row["flightId"])))
		if err != nil {
			if errors.Is(err, application.ErrNotFound) {
				continue
			}
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
	if itemTTLExpired(item, r.now()) {
		return domain.Flight{}, application.CacheMetadata{Freshness: application.CacheFreshnessMiss, Source: application.CacheSourceCache}, application.ErrNotFound
	}
	flight, err := flightFromItem(item)
	if err != nil {
		return domain.Flight{}, application.CacheMetadata{}, err
	}
	return flight, cacheFor(true), nil
}

func itemTTLExpired(item map[string]any, now time.Time) bool {
	if _, ok := item["ttl"]; !ok {
		return false
	}
	return int64(numberValue(item["ttl"])) <= now.UTC().Unix()
}

func itemTTLExpiredAtISO(item map[string]any, now string) (bool, error) {
	parsed, err := time.Parse(time.RFC3339, now)
	if err != nil {
		return false, err
	}
	return itemTTLExpired(item, parsed), nil
}

func flightFromItem(item map[string]any) (domain.Flight, error) {
	var flight domain.Flight
	if err := json.Unmarshal([]byte(stringValue(item["body"])), &flight); err != nil {
		return domain.Flight{}, err
	}
	flight.FetchOwner = nil
	flight.FetchLeaseUntil = nil
	if owner := stringValue(item["fetchOwner"]); owner != "" {
		flight.FetchOwner = &owner
	}
	if leaseUntil := stringValue(item["fetchLeaseUntil"]); leaseUntil != "" {
		value := domain.ISODateTimeString(leaseUntil)
		flight.FetchLeaseUntil = &value
	}
	return flight, nil
}

func (r *DynamoDBRepository) ListPollableFlights(ctx context.Context, now string, limit int) ([]domain.Flight, error) {
	flights := []domain.Flight{}
	pageLimit := pollQueryPageLimit
	if limit > 0 && limit > pageLimit {
		pageLimit = limit
	}
	var startKey map[string]any
	for {
		// Query bounded GSI pages and filter expired or no-longer-due rows locally;
		// DynamoDB TTL deletion is asynchronous and the poll index can lag.
		items, nextKey, err := r.client.QueryRangeUntilPage(ctx, r.tables.Flights, pollDueIndexName, "pollShard", pollShardActive, "nextPollAt", now, pageLimit, startKey)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			expired, err := itemTTLExpiredAtISO(item, now)
			if err != nil {
				return nil, err
			}
			if expired {
				continue
			}
			var flight domain.Flight
			if err := json.Unmarshal([]byte(stringValue(item["body"])), &flight); err != nil {
				return nil, err
			}
			if !hasDuePoll(flight, now) {
				continue
			}
			flights = append(flights, flight)
			if limit > 0 && len(flights) >= limit {
				return flights, nil
			}
		}
		if len(nextKey) == 0 {
			return flights, nil
		}
		startKey = nextKey
	}
}

func (r *DynamoDBRepository) UpdatePollSchedule(ctx context.Context, flight domain.Flight) error {
	current, _, err := r.GetFlight(ctx, flight.FlightID)
	if err != nil {
		return err
	}
	updated := applyPollSchedule(current, flight)
	expected, absent := flightWriteConditions(current, flight)
	item, err := r.flightItemPreservingCommittedPositionWriteToken(ctx, updated)
	if err != nil {
		return err
	}
	ok, err := r.client.PutItemIfCurrentAttributesMatch(
		ctx,
		r.tables.Flights,
		item,
		expected,
		absent,
	)
	if err != nil {
		return err
	}
	_ = ok
	return nil
}

func (r *DynamoDBRepository) flightItemPreservingCommittedPositionWriteToken(ctx context.Context, flight domain.Flight) (map[string]any, error) {
	item := flightItem(flight)
	tokens, err := r.committedPositionWriteTokens(ctx, flight.FlightID)
	if err != nil {
		return nil, err
	}
	applyCommittedPositionWriteTokens(item, tokens)
	return item, nil
}

func (r *DynamoDBRepository) AcquireFetchLease(ctx context.Context, flightID domain.FlightID, owner string, leaseUntil string, now string) (domain.Flight, bool, error) {
	item, acquired, err := r.client.AcquireLeaseIfAvailable(ctx, r.tables.Flights, string(flightID), owner, leaseUntil, now)
	if err != nil {
		return domain.Flight{}, false, err
	}
	if acquired {
		flight, err := flightFromItem(item)
		if err != nil {
			return domain.Flight{}, false, err
		}
		return flight, true, nil
	}
	flight, _, err := r.GetFlight(ctx, flightID)
	if err != nil {
		return domain.Flight{}, false, err
	}
	return flight, false, nil
}

func (r *DynamoDBRepository) UpdateFetchedFlight(ctx context.Context, flight domain.Flight, owner string, leaseUntil string, now string) (bool, error) {
	flight.UpdatedAt = now
	return r.putFlightIfLeaseOwnerAt(ctx, flight, owner, leaseUntil, now)
}

func (r *DynamoDBRepository) FetchLeaseHeld(ctx context.Context, flightID domain.FlightID, owner string, leaseUntil string) (bool, error) {
	flight, _, err := r.GetFlight(ctx, flightID)
	if err != nil {
		return false, err
	}
	return flight.FetchOwner != nil && *flight.FetchOwner == owner &&
		flight.FetchLeaseUntil != nil && *flight.FetchLeaseUntil == leaseUntil &&
		string(*flight.FetchLeaseUntil) > r.nowISO(), nil
}

func (r *DynamoDBRepository) ReleaseFetchLease(ctx context.Context, flightID domain.FlightID, owner string, leaseUntil string, now string) error {
	_, err := r.client.RemoveLeaseIfOwner(ctx, r.tables.Flights, string(flightID), owner, leaseUntil)
	return err
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

func lookupTypeFromKey(lookupKey string) string {
	before, _, ok := strings.Cut(lookupKey, "#")
	if !ok {
		return lookupKey
	}
	return before
}

func nextPollAt(flight domain.Flight) *domain.ISODateTimeString {
	var next *domain.ISODateTimeString
	for _, candidate := range []*domain.ISODateTimeString{
		flight.NextPositionPollAt,
		flight.NextRoutePollAt,
		flight.NextTrackPollAt,
	} {
		if candidate == nil {
			continue
		}
		if next == nil || *candidate < *next {
			value := *candidate
			next = &value
		}
	}
	return next
}

func (r *DynamoDBRepository) nowISO() string {
	now := time.Now
	if r.now != nil {
		now = r.now
	}
	return now().UTC().Format(time.RFC3339)
}

func applyPollSchedule(current domain.Flight, scheduled domain.Flight) domain.Flight {
	current.PollState = scheduled.PollState
	current.NextSummaryPollAt = scheduled.NextSummaryPollAt
	current.NextPositionPollAt = scheduled.NextPositionPollAt
	current.NextTrackPollAt = scheduled.NextTrackPollAt
	current.NextRoutePollAt = scheduled.NextRoutePollAt
	current.IdleSince = scheduled.IdleSince
	return current
}

func flightWriteConditions(observedBody domain.Flight, observedLease domain.Flight) (map[string]any, []string) {
	bodyItem := flightItem(observedBody)
	leaseItem := flightItem(observedLease)
	expected := map[string]any{"body": bodyItem["body"]}
	absent := []string{}
	for _, name := range []string{"fetchOwner", "fetchLeaseUntil"} {
		if value, ok := leaseItem[name]; ok {
			expected[name] = value
			continue
		}
		absent = append(absent, name)
	}
	return expected, absent
}

func mustJSON(value any) string {
	body, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(body)
}
