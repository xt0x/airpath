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
