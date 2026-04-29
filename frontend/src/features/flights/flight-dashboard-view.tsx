import type { FormEvent } from "react";

import { airportLabel, formatTime } from "./formatters";
import { FlightMap } from "./flight-map";
import type {
  CacheMetadata,
  FlightDetailResponse,
  FlightMapDataResponse,
  FlightSummaryItem,
  UsageStatusResponse,
} from "./types";

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
          <p className="demo-notice">Personal non-commercial demo - low-frequency use</p>
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
          <dt>Cache</dt>
          <dd>{detail.cache.freshness}</dd>
        </div>
      </dl>
    </section>
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
