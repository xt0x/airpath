"use client";

import { useMemo, useState, type FormEvent } from "react";

import { AirpathApiClient } from "./api-client";
import { buildDeckGlLayers } from "./map-layers";
import { sampleDetail, sampleMapData, sampleSearch, sampleUsage } from "./sample-data";
import type {
  ApiError,
  CacheMetadata,
  FlightDetailResponse,
  FlightMapDataResponse,
  FlightSummaryItem,
  UsageStatusResponse,
} from "./types";

interface FlightDashboardProps {
  apiClient?: AirpathApiClient;
}

export function FlightDashboard({ apiClient }: FlightDashboardProps) {
  const client = useMemo(() => apiClient ?? new AirpathApiClient(), [apiClient]);
  const [query, setQuery] = useState("ANA110");
  const [searchResults, setSearchResults] = useState<FlightSummaryItem[]>(sampleSearch.items);
  const [selectedFlightId, setSelectedFlightId] = useState<string | null>(
    sampleSearch.items[0]?.flightId ?? null,
  );
  const [detail, setDetail] = useState<FlightDetailResponse | null>(sampleDetail);
  const [mapData, setMapData] = useState<FlightMapDataResponse | null>(sampleMapData);
  const [usage, setUsage] = useState<UsageStatusResponse | null>(sampleUsage);
  const [isSearching, setIsSearching] = useState(false);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function runSearch() {
    setIsSearching(true);
    setError(null);
    try {
      const response = await client.searchFlights(query.trim());
      setSearchResults(response.items);
      const firstFlightId = response.items[0]?.flightId ?? null;
      setSelectedFlightId(firstFlightId);
      if (firstFlightId !== null) {
        await loadFlight(firstFlightId);
      }
    } catch (caught) {
      setError(errorMessage(caught));
    } finally {
      setIsSearching(false);
    }
  }

  async function loadFlight(flightId: string) {
    setError(null);
    setSelectedFlightId(flightId);
    try {
      const [detailResponse, mapResponse, usageResponse] = await Promise.all([
        client.getFlightDetail(flightId),
        client.getFlightMapData(flightId),
        client.getUsageStatus(),
      ]);
      setDetail(detailResponse);
      setMapData(mapResponse);
      setUsage(usageResponse);
    } catch (caught) {
      setError(errorMessage(caught));
    }
  }

  async function refreshFlight() {
    if (selectedFlightId === null) {
      return;
    }
    setIsRefreshing(true);
    setError(null);
    try {
      await client.requestFlightRefresh(selectedFlightId, ["position", "route", "track"]);
      const [detailResponse, mapResponse, usageResponse] = await Promise.all([
        client.getFlightDetail(selectedFlightId),
        client.getFlightMapData(selectedFlightId),
        client.getUsageStatus(),
      ]);
      setDetail(detailResponse);
      setMapData(mapResponse);
      setUsage(usageResponse);
    } catch (caught) {
      setError(errorMessage(caught));
    } finally {
      setIsRefreshing(false);
    }
  }

  return (
    <FlightDashboardView
      detail={detail}
      error={error}
      isRefreshing={isRefreshing}
      isSearching={isSearching}
      mapData={mapData}
      query={query}
      searchResults={searchResults}
      selectedFlightId={selectedFlightId}
      usage={usage}
      onQueryChange={setQuery}
      onRefresh={refreshFlight}
      onSearch={runSearch}
      onSelectFlight={loadFlight}
    />
  );
}

interface FlightDashboardViewProps {
  detail: FlightDetailResponse | null;
  error: string | null;
  isRefreshing: boolean;
  isSearching: boolean;
  mapData: FlightMapDataResponse | null;
  query: string;
  searchResults: FlightSummaryItem[];
  selectedFlightId: string | null;
  usage: UsageStatusResponse | null;
  onQueryChange: (value: string) => void;
  onRefresh: () => void;
  onSearch: () => void;
  onSelectFlight: (flightId: string) => void;
}

export function FlightDashboardView(props: FlightDashboardViewProps) {
  return (
    <main className="app-shell">
      <header className="topbar">
        <div>
          <p className="eyebrow">Airpath</p>
          <h1>Flight route workspace</h1>
        </div>
        <UsageStatusBanner usage={props.usage} />
      </header>

      {props.error !== null ? <div className="error-banner">{props.error}</div> : null}

      <section className="workspace">
        <SearchPanel
          error={props.error}
          isSearching={props.isSearching}
          query={props.query}
          results={props.searchResults}
          selectedFlightId={props.selectedFlightId}
          onQueryChange={props.onQueryChange}
          onSearch={props.onSearch}
          onSelectFlight={props.onSelectFlight}
        />
        <section className="main-stack">
          <FlightMap mapData={props.mapData} />
          <div className="detail-grid">
            <SummaryPanel detail={props.detail} />
            <div className="panel">
              <StaleDataNotice cache={props.detail?.cache ?? null} />
              <RefreshControls
                disabled={props.selectedFlightId === null}
                isRefreshing={props.isRefreshing}
                usage={props.usage}
                onRefresh={props.onRefresh}
              />
            </div>
          </div>
        </section>
      </section>
    </main>
  );
}

interface SearchPanelProps {
  error: string | null;
  isSearching: boolean;
  query: string;
  results: FlightSummaryItem[];
  selectedFlightId: string | null;
  onQueryChange: (value: string) => void;
  onSearch: () => void;
  onSelectFlight: (flightId: string) => void;
}

export function SearchPanel(props: SearchPanelProps) {
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    props.onSearch();
  }

  return (
    <aside className="search-pane">
      <form className="search-form" onSubmit={submit}>
        <label htmlFor="ident">Flight</label>
        <div className="search-row">
          <input
            id="ident"
            name="ident"
            value={props.query}
            autoComplete="off"
            onChange={(event) => props.onQueryChange(event.target.value)}
          />
          <button type="submit" disabled={props.isSearching || props.query.trim().length < 2}>
            {props.isSearching ? "Searching" : "Search"}
          </button>
        </div>
      </form>
      <div className="result-list">
        {props.results.map((flight) => (
          <button
            key={flight.flightId}
            type="button"
            className="result-row"
            aria-current={props.selectedFlightId === flight.flightId}
            onClick={() => props.onSelectFlight(flight.flightId)}
          >
            <span>
              <strong>{flight.ident}</strong>
              <small>{flight.status}</small>
            </span>
            <span className="route-pair">
              {flight.origin} <span aria-hidden="true">→</span> {flight.destination}
            </span>
          </button>
        ))}
      </div>
      {props.error !== null ? <p className="inline-error">{props.error}</p> : null}
    </aside>
  );
}

export function SummaryPanel({ detail }: { detail: FlightDetailResponse | null }) {
  if (detail === null) {
    return <section className="panel empty-panel" />;
  }
  const { flight } = detail;
  return (
    <section className="panel summary-panel">
      <div className="summary-heading">
        <div>
          <p className="eyebrow">{flight.identIata ?? flight.ident}</p>
          <h2>{flight.ident}</h2>
        </div>
        <span className="status-pill">{flight.status}</span>
      </div>
      <dl className="metric-grid">
        <div>
          <dt>Origin</dt>
          <dd>{airportLabel(flight.origin)}</dd>
        </div>
        <div>
          <dt>Destination</dt>
          <dd>{airportLabel(flight.destination)}</dd>
        </div>
        <div>
          <dt>Aircraft</dt>
          <dd>{flight.aircraftType ?? "Not acquired"}</dd>
        </div>
        <div>
          <dt>Scheduled out</dt>
          <dd>{formatTime(flight.times.scheduledOut)}</dd>
        </div>
        <div>
          <dt>Progress</dt>
          <dd>{flight.progressPercent === null ? "Not acquired" : `${flight.progressPercent}%`}</dd>
        </div>
        <div>
          <dt>Freshness</dt>
          <dd>{detail.cache.freshness}</dd>
        </div>
      </dl>
    </section>
  );
}

export function FlightMap({ mapData }: { mapData: FlightMapDataResponse | null }) {
  const deckLayers = buildDeckGlLayers(mapData);
  const plannedPath = pathForLayer(mapData?.planned.geojson ?? null);
  const actualPath = pathForLayer(mapData?.actual.geojson ?? null);
  const currentPoint = pointForLayer(mapData?.current.geojson ?? null);

  return (
    <section
      className="map-surface"
      data-deck-layer-count={deckLayers.length}
      data-renderer="mapbox-deckgl-compatible"
    >
      <svg aria-label="Flight map" role="img" viewBox="0 0 100 56" preserveAspectRatio="none">
        <rect width="100" height="56" rx="2" />
        {plannedPath !== "" ? <polyline data-layer="planned" points={plannedPath} /> : null}
        {actualPath !== "" ? <polyline data-layer="actual" points={actualPath} /> : null}
        {currentPoint !== null ? (
          <circle data-layer="current" cx={currentPoint.x} cy={currentPoint.y} r="1.8" />
        ) : null}
      </svg>
      <div className="layer-strip">
        <LayerState label="Planned" layer={mapData?.planned ?? null} />
        <LayerState label="Actual" layer={mapData?.actual ?? null} />
        <LayerState label="Current" layer={mapData?.current ?? null} />
      </div>
    </section>
  );
}

function LayerState({
  label,
  layer,
}: {
  label: string;
  layer: FlightMapDataResponse["planned"] | null;
}) {
  return (
    <span className={layer?.available ? "layer-state available" : "layer-state"}>{label}</span>
  );
}

export function RefreshControls({
  disabled,
  isRefreshing,
  usage,
  onRefresh,
}: {
  disabled: boolean;
  isRefreshing: boolean;
  usage: UsageStatusResponse | null;
  onRefresh: () => void;
}) {
  const guardActive =
    usage === null
      ? false
      : !usage.fetchingEnabled || usage.budget.stopped || usage.rateLimit.limited;
  return (
    <button
      type="button"
      className="refresh-button"
      disabled={disabled || guardActive || isRefreshing}
      onClick={onRefresh}
    >
      {isRefreshing ? "Refreshing" : "Refresh"}
    </button>
  );
}

export function StaleDataNotice({ cache }: { cache: CacheMetadata | null }) {
  if (cache === null || !cache.stale) {
    return <p className="cache-note">Cache {cache?.freshness ?? "miss"}</p>;
  }
  return (
    <p className="cache-note stale">
      Cached data is {cache.freshness}. Checked {formatTime(cache.checkedAt)}.
    </p>
  );
}

export function UsageStatusBanner({ usage }: { usage: UsageStatusResponse | null }) {
  if (usage === null) {
    return <div className="usage-banner">Usage unavailable</div>;
  }
  const remaining = Math.max(
    0,
    usage.budget.softStopThreshold - usage.budget.estimatedMonthToDateCost,
  );
  const guardLabel = usage.fetchingEnabled ? "Fetch enabled" : "Fetch stopped";
  return (
    <div className={usage.fetchingEnabled ? "usage-banner" : "usage-banner stopped"}>
      <strong>{remaining.toFixed(2)} USD remaining</strong>
      <span>{guardLabel}</span>
    </div>
  );
}

function errorMessage(caught: unknown): string {
  if (typeof caught === "object" && caught !== null && "error" in caught) {
    const apiError = caught as { error?: ApiError };
    if (apiError.error !== undefined) {
      return apiError.error.message;
    }
  }
  return caught instanceof Error ? caught.message : "Request failed";
}

function airportLabel(airport: { code: string; name: string | null }) {
  return airport.name === null ? airport.code : `${airport.code} - ${airport.name}`;
}

function formatTime(value: string | null) {
  if (value === null) {
    return "Not announced";
  }
  return value.replace("T", " ").replace("Z", " UTC");
}

function pathForLayer(feature: { geometry: { coordinates: unknown } } | null) {
  const coordinates = firstLineCoordinates(feature?.geometry.coordinates);
  return coordinates
    .map(([longitude, latitude]) => `${scaleLongitude(longitude)},${scaleLatitude(latitude)}`)
    .join(" ");
}

function pointForLayer(feature: { geometry: { coordinates: unknown } } | null) {
  const coordinates = feature?.geometry.coordinates;
  if (!Array.isArray(coordinates) || coordinates.length < 2) {
    return null;
  }
  const longitude = Number(coordinates[0]);
  const latitude = Number(coordinates[1]);
  if (!Number.isFinite(longitude) || !Number.isFinite(latitude)) {
    return null;
  }
  return { x: scaleLongitude(longitude), y: scaleLatitude(latitude) };
}

function firstLineCoordinates(value: unknown): Array<[number, number]> {
  if (!Array.isArray(value)) {
    return [];
  }
  const line = Array.isArray(value[0]?.[0]) ? value[0] : value;
  if (!Array.isArray(line)) {
    return [];
  }
  return line.reduce<Array<[number, number]>>((coordinates, point: unknown) => {
    if (!Array.isArray(point) || point.length < 2) {
      return coordinates;
    }
    const longitude = Number(point[0]);
    const latitude = Number(point[1]);
    if (Number.isFinite(longitude) && Number.isFinite(latitude)) {
      coordinates.push([longitude, latitude]);
    }
    return coordinates;
  }, []);
}

function scaleLongitude(longitude: number) {
  return ((longitude + 180) / 360) * 100;
}

function scaleLatitude(latitude: number) {
  return ((90 - latitude) / 180) * 56;
}
