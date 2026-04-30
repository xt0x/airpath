package application

import (
	"context"
	"sync"
)

type RuntimeFetchConfig struct {
	ExternalFetchEnabled   bool
	RouteFetchEnabled      bool
	TrackFetchEnabled      bool
	BackgroundFetchEnabled bool
}

type RuntimeFetchPolicy struct {
	mu     sync.RWMutex
	config RuntimeFetchConfig
}

func NewRuntimeFetchPolicy(config RuntimeFetchConfig) *RuntimeFetchPolicy {
	return &RuntimeFetchPolicy{config: config}
}

func (p *RuntimeFetchPolicy) Update(config RuntimeFetchConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.config = config
}

func (p *RuntimeFetchPolicy) FetchTaskAllowed(_ context.Context, taskType FetchTaskType, reason FetchReason) (bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.config.ExternalFetchEnabled {
		return false, nil
	}
	if reason == FetchReasonLowFrequencyPoll && !p.config.BackgroundFetchEnabled {
		return false, nil
	}
	switch taskType {
	case FetchTaskRoute:
		return p.config.RouteFetchEnabled, nil
	case FetchTaskTrack, FetchTaskFinalTrack:
		return p.config.TrackFetchEnabled, nil
	default:
		return true, nil
	}
}
