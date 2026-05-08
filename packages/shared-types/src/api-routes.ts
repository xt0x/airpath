export const apiRouteTemplates = {
  airportDepartures: "/v1/airports/{airportCode}/departures",
  airportArrivals: "/v1/airports/{airportCode}/arrivals",
  searchFlights: "/v1/flights/search",
  flightDetail: "/v1/flights/{flightId}",
  flightMapData: "/v1/flights/{flightId}/map-data",
  flightPositions: "/v1/flights/{flightId}/positions",
  flightRefresh: "/v1/flights/{flightId}/refresh",
  usageStatus: "/v1/usage/status",
} as const;

export interface AirportBoardRouteQuery {
  date?: string;
}

export interface FlightPositionsRouteQuery {
  since?: string;
  limit?: number;
}

export const apiRouteBuilders = {
  airportDepartures(airportCode: string, query: AirportBoardRouteQuery = {}): string {
    return withQuery(replaceAirportCode(apiRouteTemplates.airportDepartures, airportCode), query);
  },
  airportArrivals(airportCode: string, query: AirportBoardRouteQuery = {}): string {
    return withQuery(replaceAirportCode(apiRouteTemplates.airportArrivals, airportCode), query);
  },
  searchFlights(ident: string): string {
    return withQuery(apiRouteTemplates.searchFlights, { ident });
  },
  flightDetail(flightId: string): string {
    return replaceFlightID(apiRouteTemplates.flightDetail, flightId);
  },
  flightMapData(flightId: string): string {
    return replaceFlightID(apiRouteTemplates.flightMapData, flightId);
  },
  flightPositions(flightId: string, query: FlightPositionsRouteQuery = {}): string {
    return withQuery(replaceFlightID(apiRouteTemplates.flightPositions, flightId), query);
  },
  flightRefresh(flightId: string): string {
    return replaceFlightID(apiRouteTemplates.flightRefresh, flightId);
  },
  usageStatus(): string {
    return apiRouteTemplates.usageStatus;
  },
} as const;

function replaceAirportCode(template: string, airportCode: string): string {
  return template.replace("{airportCode}", encodeURIComponent(airportCode.trim().toUpperCase()));
}

function replaceFlightID(template: string, flightId: string): string {
  return template.replace("{flightId}", encodeURIComponent(flightId));
}

function withQuery(path: string, query: object): string {
  const params = new URLSearchParams();
  for (const [name, value] of Object.entries(query)) {
    if (typeof value === "string" || typeof value === "number") {
      params.set(name, String(value));
    }
  }
  const body = params.toString();
  return body === "" ? path : `${path}?${body}`;
}
