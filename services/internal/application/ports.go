package application

import (
	"context"

	"airpath/services/internal/domain"
)

type FlightStore interface {
	SearchByIdent(context.Context, string) ([]domain.Flight, CacheMetadata, error)
	GetFlight(context.Context, domain.FlightID) (domain.Flight, CacheMetadata, error)
}

type MapDataStore interface {
	GetPlannedRoute(context.Context, domain.Flight) (MapLayer, error)
	GetActualTrack(context.Context, domain.Flight) (MapLayer, error)
}

type PositionStore interface {
	GetLatestPosition(context.Context, domain.FlightID) (*domain.FlightPosition, CacheMetadata, error)
}

type FetchTaskQueue interface {
	EnqueueFetchTask(context.Context, FetchTask) (bool, error)
}

type UsageGuard interface {
	FetchingAllowed(context.Context) (bool, error)
	GetUsageStatus(context.Context) (UsageStatus, error)
}

type FetchPolicy interface {
	FetchTaskAllowed(context.Context, FetchTaskType, FetchReason) (bool, error)
}

type Config struct {
	Flights     FlightStore
	MapData     MapDataStore
	Positions   PositionStore
	FetchTasks  FetchTaskQueue
	UsageGuard  UsageGuard
	FetchPolicy FetchPolicy
}

type Application struct {
	flights     FlightStore
	mapData     MapDataStore
	positions   PositionStore
	fetchTasks  FetchTaskQueue
	usageGuard  UsageGuard
	fetchPolicy FetchPolicy
}

func New(config Config) *Application {
	return &Application{
		flights:     config.Flights,
		mapData:     config.MapData,
		positions:   config.Positions,
		fetchTasks:  config.FetchTasks,
		usageGuard:  config.UsageGuard,
		fetchPolicy: config.FetchPolicy,
	}
}
