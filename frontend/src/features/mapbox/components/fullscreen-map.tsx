"use client";

import { Plane } from "lucide-react";
import type { ReactNode } from "react";
import { useCallback, useEffect, useRef, useState } from "react";
import { createRoot } from "react-dom/client";
import type { Root } from "react-dom/client";
import type { Map as MapboxMap, Marker as MapboxMarker } from "mapbox-gl";

import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { formatPublicFlightStatus } from "@/features/flights/lib/public-flight-status";
import type { FlightDetail, FlightMapDataResponse, Position } from "@/features/flights/types";
import {
  formatAircraftAltitude,
  formatAircraftSpeed,
  formatAircraftTimestamp,
} from "@/features/mapbox/lib/aircraft-hover-card-formatters";
import { boundsFromMapData } from "@/features/mapbox/lib/map-camera";
import {
  applyNativeRouteLayerPaintProperties,
  ensureNativeRouteLayers,
  updateNativeRouteLayers,
} from "@/features/mapbox/lib/map-route-layers";
import { AIRPATH_MAP_THEME } from "@/features/mapbox/lib/map-theme";

import styles from "@/features/mapbox/components/fullscreen-map.module.css";

const DEFAULT_CENTER: [number, number] = [139.767, 35.681];
const DEFAULT_ZOOM = 3.2;
const AIRCRAFT_FOCUS_ZOOM = 7;
const MAP_RESIZE_SETTLE_MS = 240;
const ROUTE_FOCUS_PADDING = 80;

export interface FullscreenMapProps {
  accessToken: string;
  styleURL: string;
  mapData?: FlightMapDataResponse | null;
  currentPosition?: Position | null;
  flight?: FlightDetail | null;
  aircraftFocusRequest?: number;
}

export function FullscreenMap({
  accessToken,
  styleURL,
  mapData = null,
  currentPosition = null,
  flight = null,
  aircraftFocusRequest = 0,
}: FullscreenMapProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const mapRef = useRef<MapboxMap | null>(null);
  const mapboxGLRef = useRef<(typeof import("mapbox-gl"))["default"] | null>(null);
  const aircraftMarkerRef = useRef<MapboxMarker | null>(null);
  const aircraftMarkerRootRef = useRef<Root | null>(null);
  const mapDataRef = useRef<FlightMapDataResponse | null>(mapData);
  const flightRef = useRef<FlightDetail | null>(flight);
  const resizeTimeoutRef = useRef<number | null>(null);
  const handledAircraftFocusRequestRef = useRef(0);
  const [mapReady, setMapReady] = useState(false);
  const missingToken = accessToken.length === 0;

  const removeAircraftMarker = useCallback(() => {
    aircraftMarkerRef.current?.remove();
    aircraftMarkerRef.current = null;
    const markerRoot = aircraftMarkerRootRef.current;
    aircraftMarkerRootRef.current = null;
    if (markerRoot !== null) {
      scheduleAircraftMarkerRootUnmount(markerRoot);
    }
  }, []);

  useEffect(() => {
    if (!containerRef.current || mapRef.current) {
      return;
    }

    if (missingToken) {
      return;
    }

    let cancelled = false;

    void import("mapbox-gl").then(({ default: mapboxgl }) => {
      if (cancelled || !containerRef.current) {
        return;
      }

      mapboxgl.accessToken = accessToken;
      mapboxGLRef.current = mapboxgl;
      mapRef.current = new mapboxgl.Map({
        attributionControl: true,
        center: DEFAULT_CENTER,
        container: containerRef.current,
        style: styleURL,
        zoom: DEFAULT_ZOOM,
      });
      mapRef.current.addControl(
        new mapboxgl.NavigationControl({ visualizePitch: true }),
        "top-right",
      );
      const refreshNativeRouteLayers = () => {
        if (cancelled || !mapRef.current?.isStyleLoaded()) {
          return;
        }
        ensureNativeRouteLayers(mapRef.current);
        updateNativeRouteLayers(mapRef.current, mapDataRef.current, flightRef.current);
      };
      mapRef.current.once("load", refreshNativeRouteLayers);
      mapRef.current.on("styledata", refreshNativeRouteLayers);
      mapRef.current.on("idle", refreshNativeRouteLayers);
      mapRef.current.resize();
      setMapReady(true);
    });

    return () => {
      cancelled = true;
      setMapReady(false);
      if (resizeTimeoutRef.current !== null) {
        window.clearTimeout(resizeTimeoutRef.current);
        resizeTimeoutRef.current = null;
      }
      removeAircraftMarker();
      mapRef.current?.remove();
      mapRef.current = null;
      mapboxGLRef.current = null;
    };
  }, [accessToken, missingToken, removeAircraftMarker, styleURL]);

  useEffect(() => {
    mapDataRef.current = mapData;
    flightRef.current = flight;
    if (mapRef.current?.isStyleLoaded()) {
      ensureNativeRouteLayers(mapRef.current);
      updateNativeRouteLayers(mapRef.current, mapData, flight);
    }
  }, [flight, mapData]);

  useEffect(() => {
    if (mapRef.current?.isStyleLoaded()) {
      applyNativeRouteLayerPaintProperties(mapRef.current);
    }
  });

  useEffect(() => {
    if (
      aircraftFocusRequest === 0 ||
      handledAircraftFocusRequestRef.current === aircraftFocusRequest ||
      !mapReady ||
      !mapRef.current
    ) {
      return;
    }

    if (currentPosition !== null) {
      mapRef.current.flyTo({
        center: [currentPosition.longitude, currentPosition.latitude],
        essential: true,
        zoom: Math.max(mapRef.current.getZoom(), AIRCRAFT_FOCUS_ZOOM),
      });
      handledAircraftFocusRequestRef.current = aircraftFocusRequest;
      return;
    }

    const bounds = boundsFromMapData(mapData, mapboxGLRef.current);
    if (bounds === null) {
      return;
    }
    mapRef.current.fitBounds(bounds, {
      essential: true,
      maxZoom: AIRCRAFT_FOCUS_ZOOM,
      padding: ROUTE_FOCUS_PADDING,
    });
    handledAircraftFocusRequestRef.current = aircraftFocusRequest;
  }, [aircraftFocusRequest, currentPosition, mapData, mapReady]);

  useEffect(() => {
    if (!mapReady || !mapRef.current || !mapboxGLRef.current || missingToken) {
      return;
    }

    if (currentPosition === null) {
      removeAircraftMarker();
      return;
    }

    const markerCoordinates: [number, number] = [
      currentPosition.longitude,
      currentPosition.latitude,
    ];
    let marker = aircraftMarkerRef.current;
    if (marker === null) {
      const element = document.createElement("div");
      element.className = "mapbox-screen__aircraft-marker-host";
      marker = new mapboxGLRef.current.Marker({ element, rotationAlignment: "map" });
      marker.setLngLat(markerCoordinates);
      marker.addTo(mapRef.current);
      aircraftMarkerRef.current = marker;
      aircraftMarkerRootRef.current = createRoot(element);
    }

    aircraftMarkerRootRef.current?.render(
      <AircraftHoverCard flight={flight} position={currentPosition}>
        <div
          aria-label="Current aircraft position"
          className="mapbox-screen__aircraft-marker"
          data-has-heading={currentPosition.headingDegrees === null ? "false" : "true"}
        >
          <Plane aria-hidden="true" className="mapbox-screen__aircraft-marker-icon" />
        </div>
      </AircraftHoverCard>,
    );
    marker.setRotation(currentPosition.headingDegrees ?? 0);
    marker.setLngLat(markerCoordinates);
  }, [currentPosition, flight, mapReady, missingToken, removeAircraftMarker]);

  useEffect(() => {
    if (!containerRef.current || typeof ResizeObserver === "undefined") {
      return;
    }

    const resizeMap = () => {
      mapRef.current?.resize();
    };

    const scheduleResize = () => {
      if (resizeTimeoutRef.current !== null) {
        window.clearTimeout(resizeTimeoutRef.current);
      }

      resizeTimeoutRef.current = window.setTimeout(() => {
        resizeTimeoutRef.current = null;
        resizeMap();
      }, MAP_RESIZE_SETTLE_MS);
    };

    const observer = new ResizeObserver(() => {
      scheduleResize();
    });
    observer.observe(containerRef.current);

    return () => {
      observer.disconnect();
      if (resizeTimeoutRef.current !== null) {
        window.clearTimeout(resizeTimeoutRef.current);
        resizeTimeoutRef.current = null;
      }
    };
  }, []);

  return (
    <div
      className={`mapbox-screen ${styles.styleScope}`}
      data-map-provider="mapbox"
      data-route-line-color={AIRPATH_MAP_THEME.routeLine}
      data-map-style-url={styleURL}
      data-testid="fullscreen-map"
    >
      <div ref={containerRef} className="mapbox-screen__canvas" />
      {missingToken ? (
        <div className="mapbox-screen__notice" role="status">
          Set NEXT_PUBLIC_MAPBOX_ACCESS_TOKEN to show the map.
        </div>
      ) : null}
    </div>
  );
}

interface AircraftHoverCardProps {
  children: ReactNode;
  flight: FlightDetail | null;
  position: Position;
}

function AircraftHoverCard({ children, flight, position }: AircraftHoverCardProps) {
  return (
    <HoverCard closeDelay={80} openDelay={160}>
      <HoverCardTrigger asChild>{children}</HoverCardTrigger>
      <HoverCardContent align="end" className="mapbox-screen__aircraft-hover-card" side="top">
        <div className="mapbox-screen__aircraft-card-header">
          <div className="mapbox-screen__aircraft-card-title">
            {flight?.identIata ?? flight?.ident ?? "Current aircraft"}
          </div>
          <div className="mapbox-screen__aircraft-card-route">
            {flight === null
              ? "Position summary"
              : `${flight.origin.code} -> ${flight.destination.code}`}
          </div>
        </div>
        <div className="mapbox-screen__aircraft-card-grid">
          <AircraftFact
            label="Status"
            value={formatPublicFlightStatus(flight?.status) ?? "Tracking"}
          />
          <AircraftFact label="Aircraft" value={flight?.aircraftType} />
          <AircraftFact label="Registration" value={flight?.registration} />
          <AircraftFact label="Altitude" value={formatAircraftAltitude(position.altitudeFeet)} />
          <AircraftFact label="Speed" value={formatAircraftSpeed(position.groundspeedKnots)} />
          <AircraftFact label="Updated" value={formatAircraftTimestamp(position.timestamp)} />
        </div>
      </HoverCardContent>
    </HoverCard>
  );
}

function scheduleAircraftMarkerRootUnmount(root: Root) {
  window.setTimeout(() => {
    root.unmount();
  }, 0);
}

function AircraftFact({ label, value }: { label: string; value: string | null | undefined }) {
  return (
    <div className="mapbox-screen__aircraft-card-fact">
      <span>{label}</span>
      <strong>
        {value === null || value === undefined || value === "" ? "Unavailable" : value}
      </strong>
    </div>
  );
}
