import type { FlightDetail, FlightMapDataResponse } from "@/features/flights/types";

import {
  availableActualTrackFeature,
  availableCurrentPositionFeature,
  flightAwarePlannedRouteFeature,
} from "@/features/mapbox/lib/map-layer-availability";
import { drawableRouteLineFeature } from "@/features/mapbox/lib/route-line-feature";
import {
  actualTrackFeatureForDisplay,
  routeEndpointFeatureCollection,
} from "@/features/mapbox/lib/route-source-data";

const EMPTY_ROUTE_FEATURE_COLLECTION: GeoJSON.FeatureCollection = {
  type: "FeatureCollection",
  features: [],
};

export interface RouteLayerSourceData {
  actualTrack: GeoJSON.FeatureCollection;
  plannedRoute: GeoJSON.FeatureCollection;
  routeEndpoints: GeoJSON.FeatureCollection;
}

export function routeLayerSourceData(
  mapData: FlightMapDataResponse | null,
  flight: FlightDetail | null,
): RouteLayerSourceData {
  return {
    actualTrack: routeFeatureCollection(
      actualTrackFeatureForDisplay(
        availableActualTrackFeature(mapData),
        availableCurrentPositionFeature(mapData),
      ),
    ),
    plannedRoute: routeLineFeatureCollection(flightAwarePlannedRouteFeature(mapData)),
    routeEndpoints: routeEndpointFeatureCollection(mapData, flight),
  };
}

export function emptyRouteFeatureCollection(): GeoJSON.FeatureCollection {
  return EMPTY_ROUTE_FEATURE_COLLECTION;
}

function routeLineFeatureCollection(
  feature: FlightMapDataResponse["planned"]["geojson"],
): GeoJSON.FeatureCollection {
  return routeFeatureCollection(drawableRouteLineFeature(feature));
}

function routeFeatureCollection(
  feature: FlightMapDataResponse["planned"]["geojson"],
): GeoJSON.FeatureCollection {
  if (feature === null) {
    return EMPTY_ROUTE_FEATURE_COLLECTION;
  }
  return {
    type: "FeatureCollection",
    features: [feature as GeoJSON.Feature],
  };
}
