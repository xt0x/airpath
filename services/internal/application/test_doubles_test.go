package application

import (
	"context"
	"errors"

	"airpath/services/internal/domain"
)

type memoryFlightStore struct {
	searchResults map[string][]domain.Flight
	byID          map[domain.FlightID]domain.Flight
}

func newMemoryFlightStore() *memoryFlightStore {
	return &memoryFlightStore{
		searchResults: map[string][]domain.Flight{},
		byID:          map[domain.FlightID]domain.Flight{},
	}
}

func (s *memoryFlightStore) SearchByIdent(_ context.Context, ident string) ([]domain.Flight, CacheMetadata, error) {
	flights := s.searchResults[ident]
	if len(flights) == 0 {
		return nil, CacheMetadata{Freshness: CacheFreshnessMiss, Source: CacheSourceCache, Stale: false}, nil
	}
	return flights, CacheMetadata{Freshness: CacheFreshnessFresh, Source: CacheSourceCache, Stale: false, FetchedAt: ptr("2026-04-29T00:00:00Z")}, nil
}

func (s *memoryFlightStore) GetFlight(_ context.Context, flightID domain.FlightID) (domain.Flight, CacheMetadata, error) {
	flight, ok := s.byID[flightID]
	if !ok {
		return domain.Flight{}, CacheMetadata{}, ErrNotFound
	}
	return flight, CacheMetadata{Freshness: CacheFreshnessFresh, Source: CacheSourceCache, Stale: false}, nil
}

type memoryMapDataStore struct {
	routes map[domain.FlightID]MapLayer
	tracks map[domain.FlightID]MapLayer
}

func newMemoryMapDataStore() *memoryMapDataStore {
	return &memoryMapDataStore{
		routes: map[domain.FlightID]MapLayer{},
		tracks: map[domain.FlightID]MapLayer{},
	}
}

func (s *memoryMapDataStore) GetPlannedRoute(_ context.Context, flight domain.Flight) (MapLayer, error) {
	if layer, ok := s.routes[flight.FlightID]; ok {
		return layer, nil
	}
	return MapLayer{Source: MapSourceAirportGreatCircleFallback, Available: false, UnavailableReason: ptr("route unavailable")}, nil
}

func (s *memoryMapDataStore) GetActualTrack(_ context.Context, flight domain.Flight) (MapLayer, error) {
	if layer, ok := s.tracks[flight.FlightID]; ok {
		return layer, nil
	}
	return MapLayer{Source: MapSourceFlightAwareTrack, Available: false, UnavailableReason: ptr("track unavailable")}, nil
}

type memoryPositionStore struct {
	latest  map[domain.FlightID]domain.FlightPosition
	history map[domain.FlightID][]domain.FlightPosition
}

func newMemoryPositionStore() *memoryPositionStore {
	return &memoryPositionStore{
		latest:  map[domain.FlightID]domain.FlightPosition{},
		history: map[domain.FlightID][]domain.FlightPosition{},
	}
}

func (s *memoryPositionStore) GetLatestPosition(_ context.Context, flightID domain.FlightID) (*domain.FlightPosition, CacheMetadata, error) {
	position, ok := s.latest[flightID]
	if !ok {
		return nil, CacheMetadata{Freshness: CacheFreshnessMiss, Source: CacheSourceCache}, nil
	}
	return &position, CacheMetadata{Freshness: CacheFreshnessFresh, Source: CacheSourceCache}, nil
}

func (s *memoryPositionStore) ListPositions(_ context.Context, flightID domain.FlightID, since *string, limit int) ([]domain.FlightPosition, CacheMetadata, error) {
	positions := s.history[flightID]
	filtered := make([]domain.FlightPosition, 0, len(positions))
	for _, position := range positions {
		if since != nil && position.Timestamp < *since {
			continue
		}
		filtered = append(filtered, position)
		if limit > 0 && len(filtered) >= limit {
			break
		}
	}
	return filtered, CacheMetadata{Freshness: CacheFreshnessFresh, Source: CacheSourceCache}, nil
}

type memoryFetchTaskQueue struct {
	tasks      []FetchTask
	seen       map[string]struct{}
	enqueueErr error
}

func (q *memoryFetchTaskQueue) EnqueueFetchTask(_ context.Context, task FetchTask) (bool, error) {
	if q.enqueueErr != nil {
		return false, q.enqueueErr
	}
	if q.seen == nil {
		q.seen = map[string]struct{}{}
	}
	if _, ok := q.seen[task.IdempotencyKey]; ok {
		return false, nil
	}
	q.seen[task.IdempotencyKey] = struct{}{}
	q.tasks = append(q.tasks, task)
	return true, nil
}

func errQueueUnavailable() error {
	return errors.New("queue unavailable")
}

type memoryUsageGuard struct {
	fetchingEnabled bool
	status          UsageStatus
}

func (g *memoryUsageGuard) FetchingAllowed(context.Context) (bool, error) {
	return g.fetchingEnabled, nil
}

func (g *memoryUsageGuard) GetUsageStatus(context.Context) (UsageStatus, error) {
	status := g.status
	status.FetchingEnabled = g.fetchingEnabled
	return status, nil
}
