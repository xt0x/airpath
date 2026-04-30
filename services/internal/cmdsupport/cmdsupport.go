package cmdsupport

import (
	"context"
	"time"

	"airpath/services/internal/runtimewiring"
)

type EnvLookup = runtimewiring.EnvLookup

func BackgroundContext() context.Context {
	return context.Background()
}

func NowUTC() time.Time {
	return time.Now().UTC()
}

func RuntimeBackend(lookup EnvLookup) runtimewiring.Backend {
	return runtimewiring.BackendFromEnv(lookup)
}
