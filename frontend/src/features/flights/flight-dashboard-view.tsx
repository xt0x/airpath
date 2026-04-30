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
    <main className="flight-dashboard">
      <header className="flight-dashboard__topbar">
        <div>
          <p className="flight-dashboard__eyebrow">Airpath</p>
          <h1 className="flight-dashboard__title">Flight route workspace</h1>
          <p className="flight-dashboard__demo-notice">
            Personal non-commercial demo - low-frequency use
          </p>
        </div>
        <UsageStatusBanner usage={props.usage} />
      </header>

      {props.error !== null ? (
        <div className="flight-dashboard__error-banner">{props.error}</div>
      ) : null}

      <section className="flight-dashboard__workspace">
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
        <section className="flight-dashboard__main-stack">
          <FlightMap mapData={props.mapData} />
          <div className="flight-dashboard__detail-grid">
            <SummaryPanel detail={props.detail} />
            <div className="flight-dashboard__panel">
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

function SearchPanel(props: SearchPanelProps) {
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    props.onSearch();
  }

  return (
    <aside className="flight-dashboard__search-pane">
      <form className="flight-dashboard__search-form" onSubmit={submit}>
        <label htmlFor="ident">Flight</label>
        <div className="flight-dashboard__search-row">
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
      <div className="flight-dashboard__result-list">
        {props.results.map((flight) => (
          <button
            key={flight.flightId}
            type="button"
            className="flight-dashboard__result-row"
            aria-current={props.selectedFlightId === flight.flightId}
            onClick={() => props.onSelectFlight(flight.flightId)}
          >
            <span>
              <strong>{flight.ident}</strong>
              <small>{flight.status}</small>
            </span>
            <span className="flight-dashboard__route-pair">
              {flight.origin} <span aria-hidden="true">→</span> {flight.destination}
            </span>
          </button>
        ))}
      </div>
      {props.error !== null ? (
        <p className="flight-dashboard__inline-error">{props.error}</p>
      ) : null}
    </aside>
  );
}

function SummaryPanel({ detail }: { detail: FlightDetailResponse | null }) {
  if (detail === null) {
    return <section className="flight-dashboard__panel flight-dashboard__empty-panel" />;
  }
  const { flight } = detail;
  return (
    <section className="flight-dashboard__panel flight-dashboard__summary-panel">
      <div className="flight-dashboard__summary-heading">
        <div>
          <p className="flight-dashboard__eyebrow">{flight.identIata ?? flight.ident}</p>
          <h2 className="flight-dashboard__heading">{flight.ident}</h2>
        </div>
        <span className="flight-dashboard__status-pill">{flight.status}</span>
      </div>
      <dl className="flight-dashboard__metric-grid">
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

function RefreshControls({
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
      className="flight-dashboard__refresh-button"
      disabled={disabled || guardActive || isRefreshing}
      onClick={onRefresh}
    >
      {isRefreshing ? "Refreshing" : "Refresh"}
    </button>
  );
}

function StaleDataNotice({ cache }: { cache: CacheMetadata | null }) {
  if (cache === null || !cache.stale) {
    return <p className="flight-dashboard__cache-note">Cache {cache?.freshness ?? "miss"}</p>;
  }
  return (
    <p className="flight-dashboard__cache-note stale">
      Cached data is {cache.freshness}. Checked {formatTime(cache.checkedAt)}.
    </p>
  );
}

function UsageStatusBanner({ usage }: { usage: UsageStatusResponse | null }) {
  if (usage === null) {
    return <div className="flight-dashboard__usage-banner">Usage unavailable</div>;
  }
  const remaining = Math.max(
    0,
    usage.budget.softStopThreshold - usage.budget.estimatedMonthToDateCost,
  );
  const guardLabel = usage.fetchingEnabled ? "Fetch enabled" : "Fetch stopped";
  return (
    <div
      className={
        usage.fetchingEnabled
          ? "flight-dashboard__usage-banner"
          : "flight-dashboard__usage-banner stopped"
      }
    >
      <strong>{remaining.toFixed(2)} USD remaining</strong>
      <span>{guardLabel}</span>
    </div>
  );
}
