"use client";

import { CalendarIcon, Check, ChevronsUpDown, MapPinned, RefreshCw, Search } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { FormEvent } from "react";

import { AppSidebar } from "@/components/app-sidebar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { Input } from "@/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Spinner } from "@/components/ui/spinner";
import { formatPublicFlightStatus } from "@/features/flights/lib/public-flight-status";
import { formatDisplayDateTime } from "@/lib/display-date-time";
import { cn } from "@/lib/utils";

import styles from "@/features/map-workspace/components/map-workspace.module.css";
import { SidebarInset, SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar";
import { AirpathApiClient, AirpathApiError } from "@/features/flights/api/api-client";
import type {
  AirportBoardDirection,
  AirportBoardResponse,
  CacheMetadata,
  FlightDetailResponse,
  FlightMapDataResponse,
  FlightRefreshResponse,
  FlightSearchResponse,
  FlightSummaryItem,
  Position,
  RefreshTaskType,
  UsageStatusResponse,
} from "@/features/flights/types";
import { FullscreenMap } from "@/features/mapbox/components/fullscreen-map";
import {
  AIRPORT_OPTIONS,
  airportLabel,
  airportOptionByCode,
  formatDateOnly,
  parseDateOnly,
} from "@/features/map-workspace/lib/airport-board-controls";
import {
  firstOrderedSearchResultFlightId,
  sortFlightSearchResponseByScheduledOut,
} from "@/features/map-workspace/lib/flight-search-selection";

export interface MapWorkspaceProps {
  accessToken: string;
  styleURL: string;
  initialSearchPanelFocused?: boolean;
  apiClient?: AirpathApiClient;
  initialAirportBoardResponse?: AirportBoardResponse | null;
  initialSearchResponse?: FlightSearchResponse | null;
  initialDetail?: FlightDetailResponse | null;
  initialMapData?: FlightMapDataResponse | null;
  initialUsageStatus?: UsageStatusResponse | null;
}

const REFRESH_TASKS = [
  "position",
  "route",
  "final_track",
] as const satisfies readonly RefreshTaskType[];
const TRACK_HYDRATION_RETRY_DELAYS_MS = [600, 1200, 2200] as const;
const SEARCH_PANEL_ANIMATION_MS = 420;

type SearchPanelPlacement = "hidden" | "centered" | "closing";
type SidebarAction = "flight-search" | "tracked-flights";
interface LoadFlightOnMapOptions {
  closePanelAfterLoad: boolean;
  focusMapAfterLoad: boolean;
  showCandidatePending: boolean;
}

export function MapWorkspace({
  accessToken,
  styleURL,
  initialSearchPanelFocused = false,
  apiClient,
  initialAirportBoardResponse = null,
  initialSearchResponse = null,
  initialDetail = null,
  initialMapData = null,
  initialUsageStatus = null,
}: MapWorkspaceProps) {
  const client = useMemo(() => apiClient ?? new AirpathApiClient(), [apiClient]);
  const initialOrderedSearchResponse = useMemo(
    () =>
      initialSearchResponse === null
        ? null
        : sortFlightSearchResponseByScheduledOut(initialSearchResponse),
    [initialSearchResponse],
  );
  const searchPanelRef = useRef<HTMLElement | null>(null);
  const searchPanelCloseTimerRef = useRef<number | null>(null);
  const flightLoadRequestRef = useRef(0);
  const refreshRequestRef = useRef(0);
  const searchRequestRef = useRef(0);
  const airportBoardRequestRef = useRef(0);
  const [airportCode, setAirportCode] = useState(
    initialAirportBoardResponse?.airportCode ?? initialDetail?.flight.origin.code ?? "RJTT",
  );
  const [boardDate, setBoardDate] = useState(
    initialAirportBoardResponse?.date ?? formatDateOnly(new Date()),
  );
  const [boardDirection, setBoardDirection] = useState<AirportBoardDirection>(
    initialAirportBoardResponse?.direction ?? "departures",
  );
  const [airportPopoverOpen, setAirportPopoverOpen] = useState(false);
  const [datePopoverOpen, setDatePopoverOpen] = useState(false);
  const [airportBoard, setAirportBoard] = useState<AirportBoardResponse | null>(
    initialAirportBoardResponse,
  );
  const [ident, setIdent] = useState(initialDetail?.flight.ident ?? "");
  const [searchResponse, setSearchResponse] = useState<FlightSearchResponse | null>(
    initialOrderedSearchResponse,
  );
  const [selectedFlightId, setSelectedFlightId] = useState<string | null>(
    initialDetail?.flight.flightId ??
      initialMapData?.flightId ??
      initialOrderedSearchResponse?.items[0]?.flightId ??
      null,
  );
  const selectedFlightIdRef = useRef(selectedFlightId);
  selectedFlightIdRef.current = selectedFlightId;
  const [detail, setDetail] = useState<FlightDetailResponse | null>(initialDetail);
  const [mapData, setMapData] = useState<FlightMapDataResponse | null>(initialMapData);
  const [usageStatus, setUsageStatus] = useState<UsageStatusResponse | null>(initialUsageStatus);
  const [pendingMapRevealFlightId, setPendingMapRevealFlightId] = useState<string | null>(null);
  const [flightDataRequest, setFlightDataRequest] = useState(0);
  const [failedFlightPayloadRequest, setFailedFlightPayloadRequest] = useState<{
    flightId: string;
    request: number;
  } | null>(null);
  const [searchPanelPlacement, setSearchPanelPlacement] = useState<SearchPanelPlacement>(
    initialSearchPanelFocused ? "centered" : "hidden",
  );
  const [activeSidebarAction, setActiveSidebarAction] = useState<SidebarAction | undefined>(
    initialSearchPanelFocused ? "flight-search" : undefined,
  );
  const [aircraftFocusRequest, setAircraftFocusRequest] = useState(0);
  const [requestLoading, setRequestLoading] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const selectedFlightPayloadNeedsLoad =
    selectedFlightId !== null &&
    (detail?.flight.flightId !== selectedFlightId || mapData?.flightId !== selectedFlightId);
  const selectedFlightPayloadFailureIsCurrent =
    failedFlightPayloadRequest?.flightId === selectedFlightId &&
    failedFlightPayloadRequest.request === flightDataRequest;
  const selectedFlightPayloadLoading =
    selectedFlightPayloadNeedsLoad && !selectedFlightPayloadFailureIsCurrent;
  const loading = requestLoading || selectedFlightPayloadLoading;

  const clearSearchPanelCloseTimer = useCallback(() => {
    if (searchPanelCloseTimerRef.current === null) {
      return;
    }

    window.clearTimeout(searchPanelCloseTimerRef.current);
    searchPanelCloseTimerRef.current = null;
  }, []);

  const openSearchPanel = useCallback(() => {
    clearSearchPanelCloseTimer();
    setActiveSidebarAction("flight-search");
    setSearchPanelPlacement("centered");
  }, [clearSearchPanelCloseTimer]);

  const closeSearchPanel = useCallback(() => {
    clearSearchPanelCloseTimer();
    setSearchPanelPlacement("closing");
    searchPanelCloseTimerRef.current = window.setTimeout(() => {
      setSearchPanelPlacement("hidden");
      setActiveSidebarAction(undefined);
      searchPanelCloseTimerRef.current = null;
    }, SEARCH_PANEL_ANIMATION_MS);
  }, [clearSearchPanelCloseTimer]);

  const invalidatePendingFlightRequests = useCallback(() => {
    flightLoadRequestRef.current += 1;
    refreshRequestRef.current += 1;
    searchRequestRef.current += 1;
    airportBoardRequestRef.current += 1;
    setFlightDataRequest(flightLoadRequestRef.current);
    setPendingMapRevealFlightId(null);
    setRequestLoading(false);
    setRefreshing(false);
  }, []);

  const toggleSearchPanel = useCallback(() => {
    if (searchPanelPlacement === "centered") {
      closeSearchPanel();
      return;
    }

    openSearchPanel();
  }, [closeSearchPanel, openSearchPanel, searchPanelPlacement]);

  const handleTrackedFlightsSelect = useCallback(() => {
    clearSearchPanelCloseTimer();
    invalidatePendingFlightRequests();
    setSearchPanelPlacement("hidden");
    setActiveSidebarAction("tracked-flights");
    setAircraftFocusRequest((request) => request + 1);
  }, [clearSearchPanelCloseTimer, invalidatePendingFlightRequests]);

  useEffect(() => {
    return () => {
      clearSearchPanelCloseTimer();
    };
  }, [clearSearchPanelCloseTimer]);

  useEffect(() => {
    let cancelled = false;
    client
      .getUsageStatus()
      .then((response) => {
        if (!cancelled) {
          setUsageStatus(response);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setUsageStatus(null);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [client]);

  useEffect(() => {
    function openSearchPanelFromURL() {
      const params = new URLSearchParams(window.location.search);
      if (params.get("flightSearch") === "1") {
        openSearchPanel();
      }
      if (params.get("trackedFlights") === "1") {
        handleTrackedFlightsSelect();
      }
    }

    openSearchPanelFromURL();
  }, [handleTrackedFlightsSelect, openSearchPanel]);

  useEffect(() => {
    if (
      selectedFlightId === null ||
      !selectedFlightPayloadNeedsLoad ||
      selectedFlightPayloadFailureIsCurrent
    ) {
      return;
    }

    let cancelled = false;
    Promise.all([
      client.getFlightDetail(selectedFlightId),
      client.getFlightMapData(selectedFlightId),
    ])
      .then(([nextDetail, nextMapData]) => {
        if (cancelled) {
          return;
        }
        setDetail(nextDetail);
        setMapData(nextMapData);
        setFailedFlightPayloadRequest(null);
      })
      .catch((error: unknown) => {
        if (!cancelled) {
          setPendingMapRevealFlightId((pendingFlightId) =>
            pendingFlightId === selectedFlightId ? null : pendingFlightId,
          );
          setFailedFlightPayloadRequest({
            flightId: selectedFlightId,
            request: flightDataRequest,
          });
          setErrorMessage(errorLabel(error));
        }
      });

    return () => {
      cancelled = true;
    };
  }, [
    client,
    flightDataRequest,
    selectedFlightId,
    selectedFlightPayloadFailureIsCurrent,
    selectedFlightPayloadNeedsLoad,
  ]);

  useEffect(() => {
    if (searchPanelPlacement !== "centered") {
      return;
    }

    function handleOutsidePointerDown(event: PointerEvent) {
      const target = event.target;
      if (!(target instanceof Node)) {
        return;
      }
      if (searchPanelRef.current?.contains(target)) {
        return;
      }
      if (target instanceof Element && target.closest('[data-slot="popover-content"]')) {
        return;
      }
      if (target instanceof Element && target.closest('[data-flight-search-trigger="true"]')) {
        return;
      }

      closeSearchPanel();
    }

    document.addEventListener("pointerdown", handleOutsidePointerDown);
    return () => {
      document.removeEventListener("pointerdown", handleOutsidePointerDown);
    };
  }, [closeSearchPanel, searchPanelPlacement]);

  function clearAirportBoardSelection() {
    invalidatePendingFlightRequests();
    setAirportBoard(null);
    setSelectedFlightId(null);
    setDetail(null);
    setMapData(null);
    setErrorMessage(null);
  }

  function nextFlightLoadRequest() {
    flightLoadRequestRef.current += 1;
    refreshRequestRef.current += 1;
    searchRequestRef.current += 1;
    setFlightDataRequest(flightLoadRequestRef.current);
    return flightLoadRequestRef.current;
  }

  function nextSearchRequest() {
    searchRequestRef.current += 1;
    flightLoadRequestRef.current += 1;
    refreshRequestRef.current += 1;
    setFlightDataRequest(flightLoadRequestRef.current);
    return searchRequestRef.current;
  }

  function nextAirportBoardRequest() {
    airportBoardRequestRef.current += 1;
    flightLoadRequestRef.current += 1;
    refreshRequestRef.current += 1;
    setFlightDataRequest(flightLoadRequestRef.current);
    return airportBoardRequestRef.current;
  }

  function nextRefreshRequest() {
    refreshRequestRef.current += 1;
    return refreshRequestRef.current;
  }

  function flightLoadRequestIsCurrent(request: number) {
    return flightLoadRequestRef.current === request;
  }

  function refreshRequestIsCurrent(request: number, flightId: string) {
    return refreshRequestRef.current === request && selectedFlightIdRef.current === flightId;
  }

  function searchRequestIsCurrent(request: number) {
    return searchRequestRef.current === request;
  }

  function airportBoardRequestIsCurrent(request: number) {
    return airportBoardRequestRef.current === request;
  }

  function selectBoardDirection(nextDirection: AirportBoardDirection) {
    if (nextDirection === boardDirection) {
      return;
    }
    setBoardDirection(nextDirection);
    clearAirportBoardSelection();
  }

  function selectAirportCode(nextAirportCode: string) {
    if (nextAirportCode === airportCode) {
      return;
    }
    setAirportCode(nextAirportCode);
    clearAirportBoardSelection();
  }

  function selectBoardDate(nextBoardDate: string) {
    if (nextBoardDate === boardDate) {
      return;
    }
    setBoardDate(nextBoardDate);
    clearAirportBoardSelection();
  }

  async function loadFlightOnMap(flightId: string, options: LoadFlightOnMapOptions) {
    const request = nextFlightLoadRequest();
    if (options.showCandidatePending) {
      setPendingMapRevealFlightId(flightId);
    }
    setRequestLoading(true);
    setErrorMessage(null);
    try {
      const refreshResponse = await client.requestFlightRefresh(flightId, [...REFRESH_TASKS]);
      const payload = await loadFlightPayloadAfterRefresh(flightId, refreshResponse, () =>
        flightLoadRequestIsCurrent(request),
      );
      if (payload === null || !flightLoadRequestIsCurrent(request)) {
        return;
      }
      const { detail: nextDetail, mapData: nextMapData } = payload;
      setSelectedFlightId(flightId);
      setDetail(nextDetail);
      setMapData(nextMapData);
      setFailedFlightPayloadRequest(null);
      if (options.showCandidatePending) {
        setPendingMapRevealFlightId(null);
      }
      if (options.closePanelAfterLoad) {
        closeSearchPanel();
        setActiveSidebarAction(undefined);
      }
      if (options.focusMapAfterLoad) {
        setAircraftFocusRequest((request) => request + 1);
      }
    } catch (error) {
      if (!flightLoadRequestIsCurrent(request)) {
        return;
      }
      if (options.showCandidatePending) {
        setPendingMapRevealFlightId(null);
      }
      setErrorMessage(errorLabel(error));
    } finally {
      if (flightLoadRequestIsCurrent(request)) {
        setRequestLoading(false);
      }
    }
  }

  function handleFlightCandidateShowOnMap(flightId: string) {
    void loadFlightOnMap(flightId, {
      closePanelAfterLoad: true,
      focusMapAfterLoad: true,
      showCandidatePending: true,
    });
  }

  async function loadFlightPayloadAfterRefresh(
    flightId: string,
    refreshResponse: FlightRefreshResponse,
    shouldContinue: () => boolean,
  ): Promise<{ detail: FlightDetailResponse; mapData: FlightMapDataResponse } | null> {
    if (!shouldContinue()) {
      return null;
    }
    let payload = await loadFlightPayload(flightId);
    if (!shouldContinue()) {
      return null;
    }
    if (!refreshAcceptedTrackTask(refreshResponse) || payload.mapData.actual.available) {
      return payload;
    }

    for (const delayMs of TRACK_HYDRATION_RETRY_DELAYS_MS) {
      if (!shouldContinue()) {
        return null;
      }
      await waitForTrackHydration(delayMs);
      if (!shouldContinue()) {
        return null;
      }
      payload = await loadFlightPayload(flightId);
      if (!shouldContinue()) {
        return null;
      }
      if (payload.mapData.actual.available) {
        return payload;
      }
    }
    return payload;
  }

  async function loadFlightPayload(
    flightId: string,
  ): Promise<{ detail: FlightDetailResponse; mapData: FlightMapDataResponse }> {
    const [detail, mapData] = await Promise.all([
      client.getFlightDetail(flightId),
      client.getFlightMapData(flightId),
    ]);
    return { detail, mapData };
  }

  async function handleSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const trimmed = ident.trim();
    if (trimmed.length < 2) {
      setErrorMessage("Enter at least two characters.");
      return;
    }

    const request = nextSearchRequest();
    setRequestLoading(true);
    setErrorMessage(null);
    try {
      const response = await client.searchFlights(trimmed);
      if (!searchRequestIsCurrent(request)) {
        return;
      }
      const orderedResponse = sortFlightSearchResponseByScheduledOut(response);
      setSearchResponse(orderedResponse);
      const firstResultFlightId = firstOrderedSearchResultFlightId(orderedResponse);
      if (firstResultFlightId === null) {
        invalidatePendingFlightRequests();
        setSelectedFlightId(null);
        setDetail(null);
        setMapData(null);
        return;
      }
      await loadFlightOnMap(firstResultFlightId, {
        closePanelAfterLoad: false,
        focusMapAfterLoad: false,
        showCandidatePending: false,
      });
    } catch (error) {
      if (!searchRequestIsCurrent(request)) {
        return;
      }
      setErrorMessage(errorLabel(error));
    } finally {
      if (searchRequestIsCurrent(request)) {
        setRequestLoading(false);
      }
    }
  }

  async function handleAirportBoardSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const trimmed = airportCode.trim().toUpperCase();
    if (trimmed.length < 3) {
      setErrorMessage("Enter a 3 or 4 character airport code.");
      return;
    }

    const request = nextAirportBoardRequest();
    setRequestLoading(true);
    setErrorMessage(null);
    try {
      const response = await client.getAirportBoard(trimmed, boardDirection, { date: boardDate });
      if (!airportBoardRequestIsCurrent(request)) {
        return;
      }
      setAirportBoard(response);
      if (response.items.length === 0) {
        setSelectedFlightId(null);
        setDetail(null);
        setMapData(null);
      }
    } catch (error) {
      if (!airportBoardRequestIsCurrent(request)) {
        return;
      }
      setAirportBoard(null);
      setErrorMessage(errorLabel(error));
    } finally {
      if (airportBoardRequestIsCurrent(request)) {
        setRequestLoading(false);
      }
    }
  }

  async function handleRefresh() {
    if (selectedFlightId === null || refreshDisabled) {
      return;
    }

    const flightId = selectedFlightId;
    const request = nextRefreshRequest();
    setRefreshing(true);
    setErrorMessage(null);
    try {
      const refreshResponse = await client.requestFlightRefresh(flightId, [...REFRESH_TASKS]);
      const payload = await loadFlightPayloadAfterRefresh(flightId, refreshResponse, () =>
        refreshRequestIsCurrent(request, flightId),
      );
      if (payload === null || !refreshRequestIsCurrent(request, flightId)) {
        return;
      }
      const { detail: nextDetail, mapData: nextMapData } = payload;
      setDetail(nextDetail);
      setMapData(nextMapData);
    } catch (error) {
      if (!refreshRequestIsCurrent(request, flightId)) {
        return;
      }
      setErrorMessage(errorLabel(error));
    } finally {
      if (refreshRequestIsCurrent(request, flightId)) {
        setRefreshing(false);
      }
    }
  }

  const current = detail?.current ?? null;
  const selectedAirport = airportOptionByCode(airportCode);
  const selectedBoardDate = parseDateOnly(boardDate);
  const cache =
    mapData?.cache ?? detail?.cache ?? airportBoard?.cache ?? searchResponse?.cache ?? null;
  const refreshDisabled =
    selectedFlightId === null ||
    refreshing ||
    loading ||
    usageStatus?.fetchingEnabled === false ||
    usageStatus?.budget.stopped === true ||
    usageStatus?.rateLimit.limited === true;
  const searchPanelVisible = searchPanelPlacement !== "hidden";
  const searchPanelFocused = searchPanelPlacement === "centered";
  const sidebarActivePath = searchPanelVisible ? "flight-search" : activeSidebarAction;

  return (
    <SidebarProvider className="app-shell--fixed map-sidebar-layout">
      <AppSidebar
        activePath={sidebarActivePath}
        onFlightSearchSelect={toggleSearchPanel}
        onTrackedFlightsSelect={handleTrackedFlightsSelect}
      />
      <SidebarInset
        className={cn(
          "map-workspace__inset",
          styles.styleScope,
          searchPanelFocused && "map-workspace__inset--search-focus",
        )}
      >
        <header className="app-shell__toolbar">
          <SidebarTrigger className="app-shell__sidebar-trigger -ml-1" />
        </header>
        <FullscreenMap
          accessToken={accessToken}
          styleURL={styleURL}
          mapData={mapData}
          currentPosition={current}
          flight={detail?.flight ?? null}
          aircraftFocusRequest={aircraftFocusRequest}
        />
        {searchPanelVisible ? (
          <section
            ref={searchPanelRef}
            className={cn(
              "live-map-panel live-map-panel--centered",
              searchPanelPlacement === "closing" && "live-map-panel--closing",
            )}
            data-testid="live-map-panel"
            aria-label="Flight search"
          >
            <form className="live-map-panel__search" onSubmit={handleAirportBoardSearch}>
              <div className="live-map-panel__heading">
                <h2>Airport board</h2>
              </div>
              <div className="live-map-panel__segmented" role="group" aria-label="Board direction">
                <Button
                  type="button"
                  variant={boardDirection === "departures" ? "secondary" : "outline"}
                  data-selected={boardDirection === "departures"}
                  className="live-map-panel__direction-button"
                  onClick={() => selectBoardDirection("departures")}
                >
                  Departures
                </Button>
                <Button
                  type="button"
                  variant={boardDirection === "arrivals" ? "secondary" : "outline"}
                  data-selected={boardDirection === "arrivals"}
                  className="live-map-panel__direction-button"
                  onClick={() => selectBoardDirection("arrivals")}
                >
                  Arrivals
                </Button>
              </div>
              <span className="live-map-panel__label">Airport</span>
              <div className="live-map-panel__board-controls">
                <Popover open={airportPopoverOpen} onOpenChange={setAirportPopoverOpen}>
                  <PopoverTrigger asChild>
                    <Button
                      type="button"
                      variant="outline"
                      role="combobox"
                      aria-expanded={airportPopoverOpen}
                      className="live-map-panel__airport-trigger"
                    >
                      <span>{airportLabel(selectedAirport, airportCode)}</span>
                      <ChevronsUpDown aria-hidden="true" />
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent className="live-map-panel__airport-popover" align="start">
                    <Command>
                      <CommandInput placeholder="Search airport..." />
                      <CommandList>
                        <CommandEmpty>No airport found.</CommandEmpty>
                        <CommandGroup heading="Airports">
                          {AIRPORT_OPTIONS.map((airport) => (
                            <CommandItem
                              key={airport.icao}
                              value={`${airport.name} ${airport.city} ${airport.iata} ${airport.icao}`}
                              onSelect={() => {
                                selectAirportCode(airport.icao);
                                setAirportPopoverOpen(false);
                              }}
                            >
                              <Check
                                aria-hidden="true"
                                className={
                                  airport.icao === airportCode ? "opacity-100" : "opacity-0"
                                }
                              />
                              <span className="live-map-panel__airport-option">
                                <strong>{airport.name}</strong>
                                <small>
                                  {airport.city} · {airport.iata} / {airport.icao}
                                </small>
                              </span>
                            </CommandItem>
                          ))}
                        </CommandGroup>
                      </CommandList>
                    </Command>
                  </PopoverContent>
                </Popover>
                <Popover open={datePopoverOpen} onOpenChange={setDatePopoverOpen}>
                  <PopoverTrigger asChild>
                    <Button
                      type="button"
                      variant="outline"
                      aria-label="Board date"
                      className="live-map-panel__date-trigger"
                    >
                      <CalendarIcon aria-hidden="true" />
                      <span>{boardDate}</span>
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent className="live-map-panel__calendar-popover" align="start">
                    <Calendar
                      mode="single"
                      selected={selectedBoardDate}
                      month={selectedBoardDate}
                      onSelect={(date) => {
                        if (date === undefined) {
                          return;
                        }
                        selectBoardDate(formatDateOnly(date));
                        setDatePopoverOpen(false);
                      }}
                    />
                  </PopoverContent>
                </Popover>
                <Button type="submit" disabled={loading} className="live-map-panel__submit">
                  <Search aria-hidden="true" />
                  <span>Load</span>
                </Button>
              </div>
            </form>

            {airportBoard !== null ? (
              <FlightCandidateList
                label={`${airportBoard.airportCode} ${airportBoard.direction}`}
                items={airportBoard.items}
                selectedFlightId={selectedFlightId}
                showingOnMapFlightId={pendingMapRevealFlightId}
                onShowOnMap={handleFlightCandidateShowOnMap}
              />
            ) : null}

            <form
              className="live-map-panel__search live-map-panel__search--secondary"
              onSubmit={handleSearch}
            >
              <span className="live-map-panel__label">Flight number</span>
              <div>
                <Input
                  aria-label="Flight number"
                  value={ident}
                  onChange={(event) => setIdent(event.target.value)}
                  placeholder="ANA110"
                  autoComplete="off"
                  className="live-map-panel__flight-input"
                />
                <Button type="submit" disabled={loading} className="live-map-panel__submit">
                  <Search aria-hidden="true" />
                  <span>Search</span>
                </Button>
              </div>
              <p className="live-map-panel__muted live-map-panel__flight-examples">
                Examples: UAL130, DAL276, AAL176, CPA509, SIA12
              </p>
            </form>

            {searchResponse !== null ? (
              <FlightCandidateList
                label="Flight number results"
                items={searchResponse.items}
                selectedFlightId={selectedFlightId}
                showingOnMapFlightId={pendingMapRevealFlightId}
                onShowOnMap={handleFlightCandidateShowOnMap}
              />
            ) : null}

            {errorMessage !== null ? (
              <p className="live-map-panel__alert" role="alert">
                {errorMessage}
              </p>
            ) : null}

            <FlightSummary detail={detail} />
            <PositionFacts position={current} />
            <FlightDataStatus cache={cache} />

            <Button
              variant="outline"
              className="live-map-panel__refresh"
              type="button"
              disabled={refreshDisabled}
              onClick={handleRefresh}
            >
              <RefreshCw aria-hidden="true" />
              <span>{refreshing ? "Refreshing" : "Refresh"}</span>
            </Button>
            {usageStatus?.fetchingEnabled === false ? (
              <p className="live-map-panel__muted">FlightAware fetching is paused.</p>
            ) : null}
          </section>
        ) : null}
      </SidebarInset>
    </SidebarProvider>
  );
}

function FlightCandidateList({
  label,
  items,
  selectedFlightId,
  showingOnMapFlightId,
  onShowOnMap,
}: {
  label: string;
  items: FlightSummaryItem[];
  selectedFlightId: string | null;
  showingOnMapFlightId: string | null;
  onShowOnMap: (flightId: string) => void;
}) {
  if (items.length === 0) {
    return (
      <div className="live-map-panel__results" aria-label={label}>
        <p className="live-map-panel__muted">No flights found.</p>
      </div>
    );
  }

  return (
    <div className="live-map-panel__results" aria-label={label}>
      {items.map((item) => {
        const isShowingOnMap = item.flightId === showingOnMapFlightId;
        const isMapRevealPending = showingOnMapFlightId !== null;
        return (
          <button
            key={item.flightId}
            type="button"
            data-selected={item.flightId === selectedFlightId}
            aria-label={`Show ${item.ident} on map`}
            aria-busy={isShowingOnMap ? "true" : undefined}
            disabled={isMapRevealPending}
            onClick={() => onShowOnMap(item.flightId)}
          >
            <span className="live-map-panel__candidate-main">
              <strong>{item.ident}</strong>
              <span>
                {item.origin} to {item.destination}
              </span>
              <small>{formatFlightCandidateTime(item)}</small>
            </span>
            <span className="live-map-panel__candidate-action">
              {isShowingOnMap ? (
                <>
                  <Spinner className="live-map-panel__candidate-spinner" />
                  <span>Showing</span>
                </>
              ) : (
                <>
                  <MapPinned aria-hidden="true" />
                  <span>Show on map</span>
                </>
              )}
            </span>
          </button>
        );
      })}
    </div>
  );
}

function FlightSummary({ detail }: { detail: FlightDetailResponse | null }) {
  if (detail === null) {
    return (
      <section className="live-map-panel__section">
        <h2>No selected flight</h2>
        <p className="live-map-panel__muted">Search for a flight ident to load cached map data.</p>
      </section>
    );
  }

  const flight = detail.flight;
  return (
    <section className="live-map-panel__section">
      <div className="live-map-panel__heading">
        <h2>{flight.ident}</h2>
        <Badge variant="outline" className="live-map-panel__heading-badge">
          {formatPublicFlightStatus(flight.status)}
        </Badge>
      </div>
      <dl className="live-map-panel__facts">
        <Fact label="IATA" value={flight.identIata} />
        <Fact label="Route" value={`${flight.origin.code} to ${flight.destination.code}`} />
        <Fact label="Aircraft" value={flight.aircraftType} />
        <Fact label="Registration" value={flight.registration} />
        <Fact label="Progress" value={formatPercent(flight.progressPercent)} />
      </dl>
    </section>
  );
}

function PositionFacts({ position }: { position: Position | null }) {
  if (position === null) {
    return (
      <section className="live-map-panel__section">
        <h2>Current position unavailable</h2>
        <p className="live-map-panel__muted">No current aircraft position is available.</p>
      </section>
    );
  }

  return (
    <section className="live-map-panel__section">
      <h2>Current position</h2>
      <dl className="live-map-panel__facts">
        <Fact label="Latitude" value={position.latitude.toFixed(4)} />
        <Fact label="Longitude" value={position.longitude.toFixed(4)} />
        <Fact label="Altitude" value={formatFeet(position.altitudeFeet)} />
        <Fact label="Speed" value={formatKnots(position.groundspeedKnots)} />
        <Fact label="Heading" value={formatDegrees(position.headingDegrees)} />
        <Fact label="Vertical" value={position.altitudeChange} />
        <Fact label="Update" value={position.updateType} />
        <Fact label="Source" value={position.source} />
        <Fact label="Timestamp" value={formatDisplayDateTime(position.timestamp)} />
      </dl>
    </section>
  );
}

function FlightDataStatus({ cache }: { cache: CacheMetadata | null }) {
  if (cache === null) {
    return null;
  }
  const label = cache.stale ? "Flight data may be outdated" : "Flight data";
  const badge = cache.stale ? "Outdated" : cache.freshness === "miss" ? "Unavailable" : "Updated";
  const updatedAt = formatDisplayDateTime(cache.fetchedAt ?? cache.checkedAt);
  return (
    <section className="live-map-panel__section">
      <div className="live-map-panel__heading">
        <h2>{label}</h2>
        <Badge variant="outline" className="live-map-panel__heading-badge">
          {badge}
        </Badge>
      </div>
      <p className="live-map-panel__muted">{`Updated ${updatedAt}`}</p>
    </section>
  );
}

function Fact({ label, value }: { label: string; value: string | number | null | undefined }) {
  return (
    <>
      <dt>{label}</dt>
      <dd>{value === null || value === undefined || value === "" ? "Unavailable" : value}</dd>
    </>
  );
}

function formatPercent(value: number | null): string | null {
  return value === null ? null : `${value}%`;
}

function formatFeet(value: number | null): string | null {
  return value === null ? null : `${value.toLocaleString("en-US")} ft`;
}

function formatKnots(value: number | null): string | null {
  return value === null ? null : `${value.toLocaleString("en-US")} kt`;
}

function formatDegrees(value: number | null): string | null {
  return value === null ? null : `${value} deg`;
}

function formatFlightCandidateTime(item: FlightSummaryItem): string {
  return item.scheduledOut === null
    ? formatPublicFlightStatus(item.status)
    : formatDisplayDateTime(item.scheduledOut);
}

function refreshAcceptedTrackTask(response: FlightRefreshResponse) {
  return response.acceptedTasks.some((task) => isTrackRefreshTaskType(task.taskType));
}

function isTrackRefreshTaskType(
  taskType: FlightRefreshResponse["acceptedTasks"][number]["taskType"],
) {
  return taskType === "track" || taskType === "final_track";
}

function waitForTrackHydration(delayMs: number) {
  return new Promise<void>((resolve) => {
    globalThis.setTimeout(resolve, delayMs);
  });
}

function errorLabel(error: unknown): string {
  if (error instanceof AirpathApiError) {
    return error.error.message;
  }
  return "Unable to load flight data.";
}
