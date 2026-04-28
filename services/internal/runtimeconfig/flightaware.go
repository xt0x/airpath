package runtimeconfig

import (
	"errors"
	"strings"
)

const (
	AirpathEnvironmentEnv             = "AIRPATH_ENVIRONMENT"
	AirpathPersonalDemoNoticeEnv      = "AIRPATH_PERSONAL_DEMO_NOTICE"
	FlightAwareFetchEnabledEnv        = "FLIGHTAWARE_FETCH_ENABLED"
	FlightAwareRealCallsEnabledEnv    = "FLIGHTAWARE_REAL_CALLS_ENABLED"
	FlightAwareFetchDisabledReasonEnv = "FLIGHTAWARE_FETCH_DISABLED_REASON"

	DefaultPersonalDemoNotice = "Personal non-commercial low-frequency demo; real FlightAware calls are opt-in."
)

type EnvLookup func(string) string

type FlightAwareRuntimeConfig struct {
	Environment        string
	PersonalDemoNotice string
	FetchEnabled       bool
	RealCallsEnabled   bool
	DisabledReason     string
}

func LoadFlightAwareRuntimeConfig(lookup EnvLookup) (FlightAwareRuntimeConfig, error) {
	if lookup == nil {
		return FlightAwareRuntimeConfig{}, errors.New("environment lookup is required")
	}

	notice := strings.TrimSpace(lookup(AirpathPersonalDemoNoticeEnv))
	if notice == "" {
		notice = DefaultPersonalDemoNotice
	}

	return FlightAwareRuntimeConfig{
		Environment:        strings.TrimSpace(lookup(AirpathEnvironmentEnv)),
		PersonalDemoNotice: notice,
		FetchEnabled:       BoolEnv(lookup, FlightAwareFetchEnabledEnv),
		RealCallsEnabled:   BoolEnv(lookup, FlightAwareRealCallsEnabledEnv),
		DisabledReason:     strings.TrimSpace(lookup(FlightAwareFetchDisabledReasonEnv)),
	}, nil
}

func (c FlightAwareRuntimeConfig) ExternalCallsAllowed() bool {
	return c.FetchEnabled && c.RealCallsEnabled
}

func BoolEnv(lookup EnvLookup, name string) bool {
	if lookup == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(lookup(name)), "true")
}
