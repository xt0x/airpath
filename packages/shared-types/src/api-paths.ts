import { refreshTaskTypes } from "./api-types.js";
import { apiRouteTemplates } from "./api-routes.js";
import { flightIdParameter, successJsonResponse, typedErrorResponse } from "./api-schemas.js";

export const apiPaths = {
  [apiRouteTemplates.searchFlights]: {
    get: {
      operationId: "searchFlights",
      summary: "Search flights by ident with cache-first semantics",
      parameters: [
        {
          name: "ident",
          in: "query",
          required: true,
          schema: {
            type: "string",
            minLength: 2,
          },
        },
      ],
      responses: {
        "200": {
          ...successJsonResponse,
          content: {
            "application/json": {
              schema: { $ref: "#/components/schemas/FlightSearchResponse" },
            },
          },
        },
        "400": typedErrorResponse,
        "429": typedErrorResponse,
        "502": typedErrorResponse,
        "503": typedErrorResponse,
      },
    },
  },
  [apiRouteTemplates.flightDetail]: {
    get: {
      operationId: "getFlightDetail",
      summary: "Get a normalized flight summary",
      parameters: [flightIdParameter],
      responses: {
        "200": {
          ...successJsonResponse,
          content: {
            "application/json": {
              schema: { $ref: "#/components/schemas/FlightDetailResponse" },
            },
          },
        },
        "429": typedErrorResponse,
        "502": typedErrorResponse,
        "503": typedErrorResponse,
      },
    },
  },
  [apiRouteTemplates.flightMapData]: {
    get: {
      operationId: "getFlightMapData",
      summary: "Get planned route, actual track, and current-position map data",
      parameters: [flightIdParameter],
      responses: {
        "200": {
          ...successJsonResponse,
          content: {
            "application/json": {
              schema: { $ref: "#/components/schemas/FlightMapDataResponse" },
            },
          },
        },
        "400": typedErrorResponse,
        "429": typedErrorResponse,
        "502": typedErrorResponse,
        "503": typedErrorResponse,
      },
    },
  },
  [apiRouteTemplates.flightPositions]: {
    get: {
      operationId: "getFlightPositions",
      summary: "Get cached position history",
      parameters: [
        flightIdParameter,
        {
          name: "since",
          in: "query",
          required: false,
          schema: {
            type: "string",
            format: "date-time",
          },
        },
        {
          name: "limit",
          in: "query",
          required: false,
          schema: {
            type: "integer",
            minimum: 1,
            maximum: 500,
            default: 200,
          },
        },
      ],
      responses: {
        "200": {
          ...successJsonResponse,
          content: {
            "application/json": {
              schema: { $ref: "#/components/schemas/FlightPositionsResponse" },
            },
          },
        },
        "400": typedErrorResponse,
        "429": typedErrorResponse,
        "502": typedErrorResponse,
        "503": typedErrorResponse,
      },
    },
  },
  [apiRouteTemplates.flightRefresh]: {
    post: {
      operationId: "requestFlightRefresh",
      summary: "Request bounded FlightAware fetch tasks for one flight",
      parameters: [flightIdParameter],
      requestBody: {
        required: true,
        content: {
          "application/json": {
            schema: {
              type: "object",
              additionalProperties: false,
              required: ["taskTypes", "clientReason"],
              properties: {
                taskTypes: {
                  type: "array",
                  minItems: 1,
                  uniqueItems: true,
                  items: {
                    type: "string",
                    enum: refreshTaskTypes,
                  },
                },
                clientReason: {
                  type: "string",
                  enum: ["user_opened_flight_detail", "user_manual_refresh"],
                },
              },
            },
          },
        },
      },
      responses: {
        "202": {
          ...successJsonResponse,
          content: {
            "application/json": {
              schema: { $ref: "#/components/schemas/FlightRefreshResponse" },
            },
          },
        },
        "400": typedErrorResponse,
        "429": typedErrorResponse,
        "502": typedErrorResponse,
        "503": typedErrorResponse,
      },
    },
  },
  [apiRouteTemplates.usageStatus]: {
    get: {
      operationId: "getUsageStatus",
      summary: "Get local and reconciled FlightAware usage guard state",
      responses: {
        "200": {
          ...successJsonResponse,
          content: {
            "application/json": {
              schema: { $ref: "#/components/schemas/UsageStatusResponse" },
            },
          },
        },
        "429": typedErrorResponse,
        "502": typedErrorResponse,
        "503": typedErrorResponse,
      },
    },
  },
} as const;
