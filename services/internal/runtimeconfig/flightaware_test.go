package runtimeconfig

import "testing"

func TestLoadFlightAwareRuntimeConfigDefaultsToDisabledPersonalDemo(t *testing.T) {
	config, err := LoadFlightAwareRuntimeConfig(mapLookup(map[string]string{}))
	if err != nil {
		t.Fatalf("LoadFlightAwareRuntimeConfig() error = %v", err)
	}

	if config.Environment != DefaultEnvironment {
		t.Fatalf("Environment = %q, want %q", config.Environment, DefaultEnvironment)
	}
	if config.FetchEnabled {
		t.Fatal("FetchEnabled = true, want false by default")
	}
	if config.RealCallsEnabled {
		t.Fatal("RealCallsEnabled = true, want false by default")
	}
	if config.ExternalCallsAllowed() {
		t.Fatal("ExternalCallsAllowed() = true, want false by default")
	}
	if config.PersonalDemoNotice != DefaultPersonalDemoNotice {
		t.Fatalf("PersonalDemoNotice = %q, want default notice", config.PersonalDemoNotice)
	}

	blankEnvironment, err := LoadFlightAwareRuntimeConfig(mapLookup(map[string]string{
		AirpathEnvironmentEnv: "   ",
	}))
	if err != nil {
		t.Fatalf("LoadFlightAwareRuntimeConfig(blank environment) error = %v", err)
	}
	if blankEnvironment.Environment != DefaultEnvironment {
		t.Fatalf("blank Environment = %q, want %q", blankEnvironment.Environment, DefaultEnvironment)
	}
}

func TestLoadFlightAwareRuntimeConfigRequiresBothFetchAndRealCallFlags(t *testing.T) {
	fetchOnly, err := LoadFlightAwareRuntimeConfig(mapLookup(map[string]string{
		FlightAwareFetchEnabledEnv: "true",
	}))
	if err != nil {
		t.Fatalf("LoadFlightAwareRuntimeConfig(fetchOnly) error = %v", err)
	}
	if fetchOnly.ExternalCallsAllowed() {
		t.Fatal("ExternalCallsAllowed() = true with fetch flag only, want false")
	}

	realOnly, err := LoadFlightAwareRuntimeConfig(mapLookup(map[string]string{
		FlightAwareRealCallsEnabledEnv: "true",
	}))
	if err != nil {
		t.Fatalf("LoadFlightAwareRuntimeConfig(realOnly) error = %v", err)
	}
	if realOnly.ExternalCallsAllowed() {
		t.Fatal("ExternalCallsAllowed() = true with real-call flag only, want false")
	}

	both, err := LoadFlightAwareRuntimeConfig(mapLookup(map[string]string{
		AirpathEnvironmentEnv:             " dev ",
		FlightAwareFetchEnabledEnv:        "TRUE",
		FlightAwareRealCallsEnabledEnv:    "true",
		FlightAwareFetchDisabledReasonEnv: " ",
		AirpathPersonalDemoNoticeEnv:      "personal non-commercial low-frequency dev",
	}))
	if err != nil {
		t.Fatalf("LoadFlightAwareRuntimeConfig(both) error = %v", err)
	}
	if !both.ExternalCallsAllowed() {
		t.Fatal("ExternalCallsAllowed() = false with both flags true, want true")
	}
	if both.Environment != "dev" {
		t.Fatalf("Environment = %q, want dev", both.Environment)
	}
	if both.PersonalDemoNotice != "personal non-commercial low-frequency dev" {
		t.Fatalf("PersonalDemoNotice = %q", both.PersonalDemoNotice)
	}
}

func TestLoadFlightAwareRuntimeConfigRequiresLookup(t *testing.T) {
	_, err := LoadFlightAwareRuntimeConfig(nil)
	if err == nil {
		t.Fatal("LoadFlightAwareRuntimeConfig(nil) error = nil, want error")
	}
}

func TestBoolEnvTreatsOnlyExplicitTrueAsTrue(t *testing.T) {
	lookup := mapLookup(map[string]string{
		"LOWER": "true",
		"UPPER": "TRUE",
		"FALSE": "false",
		"SPACE": " true ",
	})

	if !BoolEnv(lookup, "LOWER") {
		t.Fatal("BoolEnv(LOWER) = false, want true")
	}
	if !BoolEnv(lookup, "UPPER") {
		t.Fatal("BoolEnv(UPPER) = false, want true")
	}
	if !BoolEnv(lookup, "SPACE") {
		t.Fatal("BoolEnv(SPACE) = false, want true")
	}
	if BoolEnv(lookup, "FALSE") {
		t.Fatal("BoolEnv(FALSE) = true, want false")
	}
	if BoolEnv(lookup, "MISSING") {
		t.Fatal("BoolEnv(MISSING) = true, want false")
	}
}

func mapLookup(values map[string]string) EnvLookup {
	return func(name string) string {
		return values[name]
	}
}
