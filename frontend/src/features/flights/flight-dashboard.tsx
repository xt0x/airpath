"use client";

import { useMemo, useState } from "react";

import { AirpathApiClient } from "./api-client";
import { errorMessage } from "./formatters";
import { FlightDashboardView } from "./flight-dashboard-view";
import type {
  FlightDetailResponse,
  FlightMapDataResponse,
  FlightSummaryItem,
  UsageStatusResponse,
} from "./types";

export {
  FlightDashboardView,
  RefreshControls,
  SearchPanel,
  StaleDataNotice,
  SummaryPanel,
  UsageStatusBanner,
} from "./flight-dashboard-view";
export { FlightMap } from "./flight-map";

interface FlightDashboardProps {
  apiClient?: AirpathApiClient;
}

interface FlightSnapshotClient {
  getFlightDetail(flightId: string): Promise<FlightDetailResponse>;
  getFlightMapData(flightId: string): Promise<FlightMapDataResponse>;
  getUsageStatus(): Promise<UsageStatusResponse>;
}

interface FlightSnapshot {
  detail: FlightDetailResponse;
  mapData: FlightMapDataResponse;
  usage: UsageStatusResponse;
}

export function FlightDashboard({ apiClient }: FlightDashboardProps) {
  const client = useMemo(() => apiClient ?? new AirpathApiClient(), [apiClient]);
  const [query, setQuery] = useState("");
  const [searchResults, setSearchResults] = useState<FlightSummaryItem[]>([]);
  const [selectedFlightId, setSelectedFlightId] = useState<string | null>(null);
  const [detail, setDetail] = useState<FlightDetailResponse | null>(null);
  const [mapData, setMapData] = useState<FlightMapDataResponse | null>(null);
  const [usage, setUsage] = useState<UsageStatusResponse | null>(null);
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
      applyFlightSnapshot(await loadFlightSnapshot(client, flightId));
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
      applyFlightSnapshot(await loadFlightSnapshot(client, selectedFlightId));
    } catch (caught) {
      setError(errorMessage(caught));
    } finally {
      setIsRefreshing(false);
    }
  }

  function applyFlightSnapshot(snapshot: FlightSnapshot) {
    setDetail(snapshot.detail);
    setMapData(snapshot.mapData);
    setUsage(snapshot.usage);
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

export async function loadFlightSnapshot(
  client: FlightSnapshotClient,
  flightId: string,
): Promise<FlightSnapshot> {
  const [detail, mapData, usage] = await Promise.all([
    client.getFlightDetail(flightId),
    client.getFlightMapData(flightId),
    client.getUsageStatus(),
  ]);
  return { detail, mapData, usage };
}
