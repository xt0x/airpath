package runtimewiring

import "airpath/services/internal/runtimeconfig"

const (
	RuntimeBackendEnv        = "AIRPATH_RUNTIME_BACKEND"
	AWSLambdaFunctionNameEnv = "AWS_LAMBDA_FUNCTION_NAME"

	FlightsTableNameEnv         = "FLIGHTS_TABLE_NAME"
	FlightLookupTableNameEnv    = "FLIGHT_LOOKUP_TABLE_NAME"
	FlightPositionsTableNameEnv = "FLIGHT_POSITIONS_TABLE_NAME"
	UsageBudgetTableNameEnv     = "USAGE_BUDGET_TABLE_NAME"

	GeoJSONBucketNameEnv                  = "GEOJSON_BUCKET_NAME"
	FetchTaskQueueURLEnv                  = "FETCH_TASK_QUEUE_URL"
	FetchTaskDiagnosticQueueURLEnv        = "FETCH_TASK_DIAGNOSTIC_QUEUE_URL"
	DispatcherAssumeActiveViewerEnv       = "DISPATCHER_ASSUME_ACTIVE_VIEWER"
	AirpathEnvironmentEnv                 = runtimeconfig.AirpathEnvironmentEnv
	FlightAwareFetchEnabledEnv            = runtimeconfig.FlightAwareFetchEnabledEnv
	FlightAwareRealCallsEnabledEnv        = runtimeconfig.FlightAwareRealCallsEnabledEnv
	FlightAwareEstimatedCostUSDPerCallEnv = "FLIGHTAWARE_ESTIMATED_COST_USD_PER_CALL"
)
