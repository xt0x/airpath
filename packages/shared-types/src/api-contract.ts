import { apiPaths } from "./api-paths.js";
import { apiSchemas } from "./api-schemas.js";
import { mvpExcludedContracts } from "./api-types.js";

export { apiPaths } from "./api-paths.js";
export { apiRouteBuilders, apiRouteTemplates } from "./api-routes.js";
export type { FlightPositionsRouteQuery } from "./api-routes.js";
export type { AirportBoardRouteQuery } from "./api-routes.js";
export { apiSchemas, fetchTaskSchema } from "./api-schemas.js";
export { validateApiSchema } from "./api-validation.js";
export type { ApiSchemaName, ApiSchemaValidationResult } from "./api-validation.js";
export {
  apiErrorCodes,
  fetchTaskTypes,
  mvpExcludedContracts,
  refreshTaskTypes,
} from "./api-types.js";
export type {
  ApiError,
  ApiErrorCode,
  AirportBoardDirection,
  AirportBoardResponse,
  Airport,
  CacheFreshness,
  CacheMetadata,
  CacheSource,
  FetchTask,
  FetchTaskType,
  FlightDetail,
  FlightDetailResponse,
  FlightMapDataResponse,
  FlightPositionsResponse,
  FlightRefreshResponse,
  FlightSearchResponse,
  FlightSummaryItem,
  FlightTimes,
  GeoJSONFeature,
  MapLayer,
  MapSource,
  MvpExcludedContract,
  Position,
  RefreshTaskType,
  UsageDailyStatus,
  UsageStatusResponse,
} from "./api-types.js";

export const apiContract = {
  openapi: "3.1.0",
  info: {
    title: "Airpath Free-Allowance MVP API",
    version: "0.4.0",
    description:
      "HTTP API contract for the FlightAware AeroAPI Personal free-allowance MVP. WebSocket delivery and FlightAware Alerts are outside this contract.",
  },
  paths: apiPaths,
  components: {
    schemas: apiSchemas,
  },
  "x-airpath-mvp": {
    excludedContracts: mvpExcludedContracts,
    fetchTaskQueue: {
      messageSchema: { $ref: "#/components/schemas/FetchTask" },
    },
  },
} as const;
