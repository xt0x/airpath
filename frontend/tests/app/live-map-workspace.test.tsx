// @vitest-environment happy-dom

import * as React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { createRoot, type Root } from "react-dom/client";
import { readFileSync } from "node:fs";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import { TooltipProvider } from "@/components/ui/tooltip";
import type { AirpathApiClient } from "@/features/flights/api/api-client";
import type {
  AirportBoardResponse,
  FlightDetailResponse,
  FlightMapDataResponse,
  FlightRefreshResponse,
  FlightSearchResponse,
  UsageStatusResponse,
} from "@/features/flights/types";
import { MapWorkspace } from "@/features/map-workspace/components/map-workspace";

const GLOBAL_CSS_PATH = "src/app/globals.css";
const LIVE_MAP_CSS_PATH = "src/features/map-workspace/components/map-workspace.module.css";

let mountedRoot: Root | null = null;
let mountedContainer: HTMLDivElement | null = null;

describe("Live Map workspace", () => {
  beforeAll(() => {
    (globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;
  });

  afterEach(async () => {
    if (mountedRoot !== null) {
      await React.act(async () => {
        mountedRoot?.unmount();
      });
      mountedContainer?.remove();
      mountedRoot = null;
      mountedContainer = null;
    }
    vi.useRealTimers();
    window.history.replaceState({}, "", "/");
  });

  it("keeps the search panel hidden in the default Live Map state", () => {
    const html = renderToStaticMarkup(
      <TooltipProvider>
        <MapWorkspace
          accessToken="pk.test"
          styleURL="mapbox://styles/example/style-id"
          initialAirportBoardResponse={airportBoard()}
          initialDetail={flightDetail()}
          initialMapData={mapData()}
        />
      </TooltipProvider>,
    );

    expect(html).not.toContain('data-testid="live-map-panel"');
    expect(html).not.toContain("Airport board");
    expect(html).toContain("Flight Search");
    expect(html).not.toContain('data-active="true"');
  });

  it("renders the airport board, selected flight details, and refresh controls when Flight Search is open", () => {
    const html = renderToStaticMarkup(
      <TooltipProvider>
        <MapWorkspace
          accessToken="pk.test"
          styleURL="mapbox://styles/example/style-id"
          initialSearchPanelFocused
          initialAirportBoardResponse={airportBoard()}
          initialDetail={flightDetail()}
          initialMapData={mapData()}
        />
      </TooltipProvider>,
    );

    expect(html).toContain('data-testid="live-map-panel"');
    expect(html).toContain("Airport board");
    expect(html).toContain("Departures");
    expect(html).toContain("Arrivals");
    expect(html).toContain("Tokyo Haneda");
    expect(html).toContain("HND / RJTT");
    expect(html).toContain("Board date");
    expect(html).toContain("ANA110");
    expect(html).toContain("NH110");
    expect(html).toContain("RJTT");
    expect(html).toContain("KJFK");
    expect(html).toContain("B789");
    expect(html).toContain("JA890A");
    expect(html).toContain("37,000 ft");
    expect(html).toContain("488 kt");
    expect(html).toContain("275 deg");
    expect(html).toContain("Flight data");
    expect(html).toContain("Updated");
    expect(html).toContain("2026");
    expect(html).not.toContain("Updated 2026-04-29T00:05:00Z");
    expect(html).not.toContain("Timestamp</dt><dd>2026-04-29T00:05:00Z");
    expect(html).not.toContain("<small>2026-05-04T01:00:00Z</small>");
    expect(html).not.toContain("Cached data");
    expect(html).not.toContain("Checked");
    expect(html).not.toContain("Stale at");
    expect(html).not.toContain("Expires");
    expect(html).toContain("Refresh");
  });

  it("keeps map layer diagnostics out of the search panel", () => {
    const html = renderToStaticMarkup(
      <TooltipProvider>
        <MapWorkspace
          accessToken="pk.test"
          styleURL="mapbox://styles/example/style-id"
          initialDetail={flightDetail()}
          initialMapData={mapData()}
        />
      </TooltipProvider>,
    );

    expect(html).not.toContain("planned available");
    expect(html).not.toContain("actual available");
    expect(html).not.toContain("current available");
  });

  it("keeps flight-number search as a secondary fallback instead of the primary flow", () => {
    const html = renderToStaticMarkup(
      <TooltipProvider>
        <MapWorkspace
          accessToken="pk.test"
          styleURL="mapbox://styles/example/style-id"
          initialSearchPanelFocused
        />
      </TooltipProvider>,
    );

    expect(html).toContain("Airport board");
    expect(html).toContain("Flight number");
    expect(html.indexOf("Airport board")).toBeLessThan(html.indexOf("Flight number"));
  });

  it("shows non-ANA/JAL airline ident examples beside flight-number search", () => {
    const html = renderToStaticMarkup(
      <TooltipProvider>
        <MapWorkspace
          accessToken="pk.test"
          styleURL="mapbox://styles/example/style-id"
          initialSearchPanelFocused
        />
      </TooltipProvider>,
    );

    expect(html).toContain("Examples");
    expect(html).toContain("UAL");
    expect(html).toContain("DAL");
    expect(html).toContain("AAL");
    expect(html).toContain("CPA");
    expect(html).toContain("SIA");
    expect(html).toContain('placeholder="ANA110"');
    expect(html).not.toContain("Examples: ANA");
    expect(html).not.toContain("Examples: JAL");
  });

  it("does not duplicate the selected board direction beside the Airport board heading", async () => {
    const container = await renderInteractiveWorkspace(createMockApiClient());
    const heading = headingByText(container, "Airport board");

    expect(heading.parentElement?.textContent?.trim()).toBe("Airport board");
    expect(container.textContent).toContain("Departures");
    expect(container.textContent).toContain("Arrivals");
  });

  it("can render the search panel aligned to the Live Map menu item without a page destination", () => {
    const html = renderToStaticMarkup(
      <TooltipProvider>
        <MapWorkspace
          accessToken="pk.test"
          styleURL="mapbox://styles/example/style-id"
          initialSearchPanelFocused
        />
      </TooltipProvider>,
    );

    expect(html).toContain('data-testid="live-map-panel"');
    expect(html).toContain("live-map-panel--centered");
    expect(html).toContain("<button");
    expect(html).toContain("Flight Search");
    expect(html).not.toContain('href="/flight-search"');
    expect(html).toContain('data-active="true"');
  });

  it("opens the centered search panel from the Live Map sidebar instead of linking to another route", async () => {
    const container = await renderInteractiveWorkspace(createMockApiClient(), {
      initialSearchPanelFocused: false,
    });

    expect(container.textContent).not.toContain("Airport board");
    expect(container.querySelector('a[href="/flight-search"]')).toBeNull();

    await clickElement(buttonByText(container, "Flight Search"));

    const searchPanel = container.querySelector('[data-testid="live-map-panel"]');
    expect(container.textContent).toContain("Airport board");
    expect(searchPanel).not.toBeNull();
    expect(searchPanel?.classList.contains("live-map-panel--centered")).toBe(true);
  });

  it("opens Flight Search from the home URL query when another page links back to Live Map", async () => {
    window.history.pushState({}, "", "/?flightSearch=1");

    const container = await renderInteractiveWorkspace(createMockApiClient(), {
      initialSearchPanelFocused: false,
    });

    const searchPanel = container.querySelector('[data-testid="live-map-panel"]');
    expect(container.textContent).toContain("Airport board");
    expect(searchPanel).not.toBeNull();
    expect(searchPanel?.classList.contains("live-map-panel--centered")).toBe(true);
  });

  it("clears the previous selected flight when airport board criteria changes", async () => {
    const container = await renderInteractiveWorkspace(createMockApiClient(), {
      initialAirportBoardResponse: airportBoard(),
      initialDetail: flightDetail(),
      initialMapData: mapData(),
    });

    expect(container.textContent).toContain("Show on map");
    expect(container.textContent).toContain("B789");

    await clickElement(buttonByText(container, "Arrivals"));

    expect(container.textContent).not.toContain("Show on map");
    expect(container.textContent).not.toContain("B789");
    expect(container.textContent).toContain("No selected flight");
  });

  it("focuses the map on the current tracked aircraft from the Tracked Flights sidebar action", async () => {
    const container = await renderInteractiveWorkspace(createMockApiClient(), {
      initialDetail: flightDetail(),
      initialMapData: mapData(),
    });

    expect(container.textContent).toContain("Airport board");

    const trackedFlightsButton = buttonByText(container, "Tracked Flights");
    await clickElement(trackedFlightsButton);

    expect(container.textContent).not.toContain("Airport board");
    expect(trackedFlightsButton.getAttribute("data-active")).toBe("true");
  });

  it("loads the earliest scheduled flight-number search result onto the map", async () => {
    const apiClient = createMockApiClient({
      searchResponse: flightSearchResponse([
        { flightId: "iflg_later", ident: "ANA110", scheduledOut: "2026-05-04T10:00:00Z" },
        { flightId: "iflg_earlier", ident: "ANA110", scheduledOut: "2026-05-04T08:00:00Z" },
      ]),
      detailResponse: flightDetail({ flightId: "iflg_earlier" }),
      mapDataResponse: mapData({ flightId: "iflg_earlier" }),
    });
    const container = await renderInteractiveWorkspace(apiClient, {
      initialDetail: flightDetail(),
      initialMapData: mapData(),
    });

    await clickElement(buttonByText(container, "Search"));

    expect(apiClient.searchFlights).toHaveBeenCalledWith("ANA110");
    expect(apiClient.requestFlightRefresh).toHaveBeenCalledWith("iflg_earlier", [
      "position",
      "route",
      "final_track",
    ]);
    expect(apiClient.getFlightDetail).toHaveBeenCalledWith("iflg_earlier");
    expect(apiClient.getFlightMapData).toHaveBeenCalledWith("iflg_earlier");
    expect(container.textContent).toContain("ANA110");
  });

  it("lets a visible board result explicitly open the selected aircraft on the map", async () => {
    const apiClient = createMockApiClient({
      detailResponse: flightDetail({ flightId: "iflg_1" }),
      mapDataResponse: mapData({ flightId: "iflg_1" }),
    });
    const container = await renderInteractiveWorkspace(apiClient, {
      initialAirportBoardResponse: airportBoard(),
      initialDetail: null,
      initialMapData: null,
    });

    expect(container.textContent).toContain("Show on map");

    await clickElement(buttonByLabel(container, "Show ANA110 on map"));

    expect(apiClient.requestFlightRefresh).toHaveBeenCalledWith("iflg_1", [
      "position",
      "route",
      "final_track",
    ]);
    expect(apiClient.getFlightDetail).toHaveBeenCalledWith("iflg_1");
    expect(apiClient.getFlightMapData).toHaveBeenCalledWith("iflg_1");
    expect(container.textContent).toContain("B789");
  });

  it("renders English search result and selected flight statuses", () => {
    const board = airportBoard();
    const firstBoardItem = board.items[0];
    if (firstBoardItem === undefined) {
      throw new Error("airport board fixture must include a flight");
    }

    const html = renderToStaticMarkup(
      <TooltipProvider>
        <MapWorkspace
          accessToken="pk.test"
          styleURL="mapbox://styles/example/style-id"
          initialSearchPanelFocused
          initialAirportBoardResponse={{
            ...board,
            items: [
              {
                ...firstBoardItem,
                scheduledOut: null,
                status: "Taxiing / Gate Departure",
              },
            ],
          }}
          initialDetail={{
            ...flightDetail(),
            flight: {
              ...flightDetail().flight,
              status: "Arrived / Gate Arrival",
            },
          }}
          initialMapData={mapData()}
        />
      </TooltipProvider>,
    );

    expect(html).toContain("Taxiing / Gate Departure");
    expect(html).toContain("Arrived / Gate Arrival");
    expect(html).not.toContain("Unknown");
  });

  it("closes the centered search panel when a pointer press starts outside the search surface", async () => {
    vi.useFakeTimers();
    const container = await renderInteractiveWorkspace(createMockApiClient());

    const openedPanel = searchPanel(container);
    expect(openedPanel).not.toBeNull();
    expect(openedPanel?.classList.contains("live-map-panel--centered")).toBe(true);

    await pointerDown(document.body);

    const closingPanel = searchPanel(container);
    expect(closingPanel).not.toBeNull();
    expect(closingPanel?.classList.contains("live-map-panel--closing")).toBe(true);

    await advanceTimersByTime(420);

    expect(searchPanel(container)).toBeNull();
  });

  it("animates the Flight Search panel while opening and closing from the sidebar action", async () => {
    vi.useFakeTimers();
    const container = await renderInteractiveWorkspace(createMockApiClient(), {
      initialSearchPanelFocused: false,
    });

    expect(searchPanel(container)).toBeNull();

    await clickElement(buttonByText(container, "Flight Search"));

    const openedPanel = searchPanel(container);
    expect(openedPanel).not.toBeNull();
    expect(openedPanel?.classList.contains("live-map-panel--centered")).toBe(true);

    await clickElement(buttonByText(container, "Flight Search"));

    const closingPanel = searchPanel(container);
    expect(closingPanel).not.toBeNull();
    expect(closingPanel?.classList.contains("live-map-panel--closing")).toBe(true);

    await advanceTimersByTime(420);

    expect(searchPanel(container)).toBeNull();
  });

  it("smoothly releases the blurred map background while the search panel closes", async () => {
    vi.useFakeTimers();
    const globalCss = readFileSync(GLOBAL_CSS_PATH, "utf8");
    const liveMapCss = cssModuleGlobalView(readFileSync(LIVE_MAP_CSS_PATH, "utf8"));
    const container = await renderInteractiveWorkspace(createMockApiClient());
    const inset = sidebarInset(container);

    expect(inset.classList.contains("map-workspace__inset--search-focus")).toBe(true);

    await clickElement(buttonByText(container, "Flight Search"));

    expect(searchPanel(container)?.classList.contains("live-map-panel--closing")).toBe(true);
    expect(inset.classList.contains("map-workspace__inset--search-focus")).toBe(false);

    await advanceTimersByTime(420);

    expect(searchPanel(container)).toBeNull();
    expect(globalCss).toMatch(
      /\.mapbox-screen\s*\{[^}]*filter:\s*blur\(0\) saturate\(1\);[^}]*transform:\s*scale\(1\);[^}]*transition:[^}]*filter 420ms[^}]*transform 420ms/s,
    );
    expect(liveMapCss).toMatch(/\.live-map-panel\s*\{[^}]*will-change:\s*opacity, transform;/s);
  });

  it("blurs only the map background behind the centered search screen", () => {
    const html = renderToStaticMarkup(
      <TooltipProvider>
        <MapWorkspace
          accessToken="pk.test"
          styleURL="mapbox://styles/example/style-id"
          initialSearchPanelFocused
        />
      </TooltipProvider>,
    );
    const css = cssModuleGlobalView(readFileSync(LIVE_MAP_CSS_PATH, "utf8"));

    expect(html).toContain("map-workspace__inset--search-focus");
    expect(css).toMatch(
      /\.map-workspace__inset--search-focus\s+\.mapbox-screen\s*\{[^}]*filter:\s*blur\(10px\)[^;]*;[^}]*transform:\s*scale\(1\.02\);/s,
    );
    expect(css).not.toMatch(
      /\.map-workspace__inset--search-focus\s+\.live-map-panel\s*\{[^}]*filter:\s*blur/s,
    );
    expect(css).not.toMatch(
      /\.map-workspace__inset--search-focus\s+\.map-workspace__sidebar-trigger\s*\{[^}]*filter:\s*blur/s,
    );
  });

  it("exposes airport, board-date, and flight-number controls without a native date input", async () => {
    const container = await renderInteractiveWorkspace(createMockApiClient());

    expect(airportCombobox(container).textContent).toContain("Tokyo Haneda");
    expect(buttonByLabel(container, "Board date").textContent).toContain("2026");
    expect(inputByLabel(container, "Flight number").getAttribute("placeholder")).toBe("ANA110");
    expect(buttonByText(container, "Load")).not.toBeNull();
    expect(buttonByText(container, "Search")).not.toBeNull();
    expect(container.querySelector('input[type="date"]')).toBeNull();
  });

  it("does not restyle shadcn airport controls with the old form button selector", () => {
    const css = cssModuleGlobalView(readFileSync(LIVE_MAP_CSS_PATH, "utf8"));

    expect(css).not.toContain(".live-map-panel__search button,");
    expect(css).not.toContain(".live-map-panel__segmented button");
    expect(css).not.toContain(".live-map-panel__search input");
    expect(css).toContain(".live-map-panel__airport-trigger");
    expect(css).toContain(".live-map-panel__date-trigger");
    expect(css).toContain(".live-map-panel__submit");
    expect(css).toContain(".live-map-panel__direction-button");
    expect(css).toContain(".live-map-panel__flight-input");
    expect(css).toContain(".live-map-panel__results button:disabled");
  });

  it("uses the same action column width for Load and Flight number Search controls", () => {
    const css = cssModuleGlobalView(readFileSync(LIVE_MAP_CSS_PATH, "utf8"));

    expect(css).toMatch(/\.live-map-panel\s*\{[^}]*--live-map-panel-action-width:\s*5rem;/s);
    expect(css).toMatch(
      /\.live-map-panel__board-controls\s*\{[^}]*grid-template-columns:\s*minmax\(0, 1fr\) minmax\(7\.5rem, 0\.72fr\) var\(--live-map-panel-action-width\);/s,
    );
    expect(css).toMatch(
      /\.live-map-panel__search--secondary > div:not\(\.live-map-panel__heading, \.live-map-panel__segmented\)\s*\{[^}]*grid-template-columns:\s*minmax\(0, 1fr\) var\(--live-map-panel-action-width\);/s,
    );
    expect(css).toMatch(/\.live-map-panel__submit\s*\{[^}]*width:\s*100%;/s);
  });

  it("uses shadcn default dark tokens for the search panel colors", () => {
    const globalCss = readFileSync(GLOBAL_CSS_PATH, "utf8");
    const css = cssModuleGlobalView(readFileSync(LIVE_MAP_CSS_PATH, "utf8"));

    expect(globalCss).toContain("--map-surface-background: var(--background)");
    expect(css).toContain("background: var(--card)");
    expect(css).toContain("color: var(--card-foreground)");
    expect(css).toContain("border: 1px solid var(--border)");
    expect(css).toContain("background: var(--input)");
    expect(css).toContain("background: var(--primary)");
    expect(css).not.toContain("rgb(18 18 18 / 88%)");
    expect(css).not.toContain("rgb(245 245 245 / 94%)");
    expect(css).not.toContain("rgb(14 165 233)");
  });

  it("uses 4px radii for the search panel box, buttons, inputs, and popover layers", () => {
    const css = cssModuleGlobalView(readFileSync(LIVE_MAP_CSS_PATH, "utf8"));

    expect(css).toMatch(/\.live-map-panel\s*\{[^}]*border-radius:\s*4px;/s);
    expect(css).toMatch(
      /\.live-map-panel__refresh,[^{]+\.live-map-panel__results button\s*\{[^}]*border-radius:\s*4px;/s,
    );
    expect(css).toMatch(/\.live-map-panel__direction-button\s*\{[^}]*border-radius:\s*4px;/s);
    expect(css).toMatch(
      /\.live-map-panel__airport-trigger,[^{]+\.live-map-panel__date-trigger,[^{]+\.live-map-panel__submit,[^{]+\.live-map-panel__flight-input\s*\{[^}]*border-radius:\s*4px;/s,
    );
    expect(css).toMatch(
      /\.live-map-panel__airport-popover,[^{]+\.live-map-panel__calendar-popover\s*\{[^}]*border-radius:\s*4px;/s,
    );
    expect(css).toMatch(/\.live-map-panel__alert\s*\{[^}]*border-radius:\s*4px;/s);
    expect(css).not.toMatch(
      /\.live-map-panel(?:__[\w-]+)?[^{}]*\{[^}]*border-radius:\s*(?:6px|8px);/s,
    );
  });

  it("keeps Live Map panel CSS out of the global stylesheet", () => {
    const globalCss = readFileSync(GLOBAL_CSS_PATH, "utf8");
    const liveMapCss = cssModuleGlobalView(readFileSync(LIVE_MAP_CSS_PATH, "utf8"));

    expect(globalCss).not.toContain(".live-map-panel");
    expect(liveMapCss).toContain(".live-map-panel");
  });

  it("surfaces stale cache state without inventing map layer diagnostics in the search panel", () => {
    const unavailableMapData: FlightMapDataResponse = {
      ...mapData(),
      planned: {
        source: "flightaware_route",
        available: false,
        geojson: null,
        unavailableReason: "route coordinates unavailable",
      },
      actual: {
        source: "flightaware_track",
        available: false,
        geojson: null,
        unavailableReason: "track coordinates unavailable",
      },
      current: {
        source: "flightaware_position",
        available: false,
        geojson: null,
        unavailableReason: "current position unavailable",
      },
      cache: { ...validCache(), freshness: "stale", stale: true },
    };

    const html = renderToStaticMarkup(
      <TooltipProvider>
        <MapWorkspace
          accessToken="pk.test"
          styleURL="mapbox://styles/example/style-id"
          initialSearchPanelFocused
          initialDetail={{ ...flightDetail(), current: null }}
          initialMapData={unavailableMapData}
        />
      </TooltipProvider>,
    );

    expect(html).toContain("Flight data may be outdated");
    expect(html).not.toContain("Cached data is stale");
    expect(html).not.toContain("route coordinates unavailable");
    expect(html).not.toContain("track coordinates unavailable");
    expect(html).not.toContain("current position unavailable");
    expect(html).toContain("Current position unavailable");
  });
});

function validCache() {
  return {
    freshness: "fresh" as const,
    source: "cache" as const,
    stale: false,
    checkedAt: "2026-04-29T00:06:00Z",
    fetchedAt: "2026-04-29T00:05:00Z",
    staleAt: null,
    expiresAt: null,
  };
}

function createMockApiClient({
  searchResponse = flightSearchResponse([]),
  detailResponse = flightDetail(),
  mapDataResponse = mapData(),
}: {
  searchResponse?: FlightSearchResponse;
  detailResponse?: FlightDetailResponse;
  mapDataResponse?: FlightMapDataResponse;
} = {}) {
  const refreshResponse: FlightRefreshResponse = {
    flightId: detailResponse.flight.flightId,
    acceptedTasks: [],
    cache: validCache(),
  };
  return {
    searchFlights: vi.fn(() => Promise.resolve(searchResponse)),
    getAirportBoard: vi.fn(() => Promise.resolve(airportBoard())),
    getFlightDetail: vi.fn(() => Promise.resolve(detailResponse)),
    getFlightMapData: vi.fn(() => Promise.resolve(mapDataResponse)),
    getFlightPositions: vi.fn(),
    requestFlightRefresh: vi.fn(() => Promise.resolve(refreshResponse)),
    getUsageStatus: vi.fn(() => Promise.resolve(usageStatus())),
  } as unknown as AirpathApiClient & {
    searchFlights: ReturnType<typeof vi.fn>;
    getFlightDetail: ReturnType<typeof vi.fn>;
    getFlightMapData: ReturnType<typeof vi.fn>;
    requestFlightRefresh: ReturnType<typeof vi.fn>;
  };
}

async function renderInteractiveWorkspace(
  apiClient: AirpathApiClient,
  props: Partial<React.ComponentProps<typeof MapWorkspace>> = {},
): Promise<HTMLDivElement> {
  mountedContainer = document.createElement("div");
  document.body.appendChild(mountedContainer);
  mountedRoot = createRoot(mountedContainer);

  await React.act(async () => {
    mountedRoot?.render(
      <TooltipProvider>
        <MapWorkspace
          accessToken=""
          styleURL="mapbox://styles/example/style-id"
          initialSearchPanelFocused
          apiClient={apiClient}
          {...props}
        />
      </TooltipProvider>,
    );
  });

  return mountedContainer;
}

async function clickElement(element: HTMLElement) {
  await React.act(async () => {
    element.click();
    await Promise.resolve();
  });
}

async function pointerDown(element: HTMLElement) {
  await React.act(async () => {
    element.dispatchEvent(new Event("pointerdown", { bubbles: true }));
    await Promise.resolve();
  });
}

async function advanceTimersByTime(milliseconds: number) {
  await React.act(async () => {
    vi.advanceTimersByTime(milliseconds);
    await Promise.resolve();
  });
}

function searchPanel(container: HTMLElement): HTMLElement | null {
  return container.querySelector('[data-testid="live-map-panel"]');
}

function sidebarInset(container: HTMLElement): HTMLElement {
  const inset = container.querySelector(".map-workspace__inset");
  if (!(inset instanceof HTMLElement)) {
    throw new Error("map workspace inset not found");
  }
  return inset;
}

function cssModuleGlobalView(css: string): string {
  let output = "";
  for (let index = 0; index < css.length; index += 1) {
    if (!css.startsWith(":global(", index)) {
      output += css[index];
      continue;
    }

    index += ":global(".length;
    let depth = 1;
    while (index < css.length && depth > 0) {
      const char = css[index];
      if (char === "(") {
        depth += 1;
        output += char;
      } else if (char === ")") {
        depth -= 1;
        if (depth > 0) {
          output += char;
        }
      } else {
        output += char;
      }
      index += 1;
    }
    index -= 1;
  }
  return output;
}

function headingByText(container: HTMLElement, text: string): HTMLHeadingElement {
  const heading = Array.from(container.querySelectorAll("h1, h2, h3")).find(
    (candidate) => candidate.textContent?.trim() === text,
  );
  if (!(heading instanceof HTMLHeadingElement)) {
    throw new Error(`heading not found: ${text}`);
  }
  return heading;
}

function airportCombobox(container: HTMLElement): HTMLElement {
  const combobox = container.querySelector('[role="combobox"]');
  if (!(combobox instanceof HTMLElement)) {
    throw new Error("airport combobox not found");
  }
  return combobox;
}

function buttonByText(container: HTMLElement, text: string): HTMLButtonElement {
  const button = Array.from(container.querySelectorAll("button")).find(
    (candidate) => candidate.textContent?.trim() === text,
  );
  if (!(button instanceof HTMLButtonElement)) {
    throw new Error(`button not found: ${text}`);
  }
  return button;
}

function buttonByLabel(container: HTMLElement, label: string): HTMLButtonElement {
  const button = container.querySelector(`button[aria-label="${label}"]`);
  if (!(button instanceof HTMLButtonElement)) {
    throw new Error(`button not found: ${label}`);
  }
  return button;
}

function inputByLabel(container: HTMLElement, label: string): HTMLInputElement {
  const input = container.querySelector(`input[aria-label="${label}"]`);
  if (!(input instanceof HTMLInputElement)) {
    throw new Error(`input not found: ${label}`);
  }
  return input;
}

function airportBoard(): AirportBoardResponse {
  return {
    airportCode: "RJTT",
    direction: "departures",
    date: "2026-05-04",
    items: [
      {
        flightId: "iflg_1",
        flightIdType: "internal",
        provisionalFlightLegId: null,
        faFlightId: "fa_1",
        ident: "ANA110",
        identIata: "NH110",
        origin: "RJTT",
        destination: "KJFK",
        scheduledOut: "2026-05-04T01:00:00Z",
        legIndex: 0,
        status: "Scheduled",
      },
    ],
    cache: validCache(),
  };
}

function flightSearchResponse(
  items: Array<
    Pick<FlightSearchResponse["items"][number], "flightId" | "ident"> &
      Partial<Pick<FlightSearchResponse["items"][number], "scheduledOut">>
  >,
): FlightSearchResponse {
  return {
    items: items.map((item) => ({
      flightId: item.flightId,
      flightIdType: "internal",
      provisionalFlightLegId: null,
      faFlightId: "fa_1",
      ident: item.ident,
      identIata: null,
      origin: "RJTT",
      destination: "KJFK",
      scheduledOut: "scheduledOut" in item ? (item.scheduledOut ?? null) : "2026-05-04T01:00:00Z",
      legIndex: 0,
      status: "Scheduled",
    })),
    cache: validCache(),
  };
}

function usageStatus(): UsageStatusResponse {
  return {
    budget: {
      environment: "dev",
      month: "2026-04",
      currency: "USD",
      estimatedMonthToDateCost: 2.5,
      softStopThreshold: 4,
      stopped: false,
      dailyUsage: [],
    },
    rateLimit: {
      limited: false,
      resetAt: null,
    },
    fetchingEnabled: true,
    cache: validCache(),
  };
}

function flightDetail({ flightId = "iflg_1" }: { flightId?: string } = {}): FlightDetailResponse {
  return {
    flight: {
      flightId,
      flightIdType: "internal",
      provisionalFlightLegId: null,
      faFlightId: "fa_1",
      ident: "ANA110",
      identIata: "NH110",
      aircraftType: "B789",
      registration: "JA890A",
      origin: { code: "RJTT", name: "Tokyo Haneda", timezone: "Asia/Tokyo" },
      destination: { code: "KJFK", name: "John F. Kennedy", timezone: "America/New_York" },
      legIndex: 0,
      status: "En Route",
      progressPercent: 62,
      times: {
        scheduledOut: "2026-04-29T00:00:00Z",
        estimatedOut: null,
        actualOut: null,
        scheduledOff: null,
        estimatedOff: null,
        actualOff: null,
        scheduledOn: null,
        estimatedOn: null,
        actualOn: null,
        scheduledIn: null,
        estimatedIn: null,
        actualIn: null,
      },
    },
    route: mapData().planned,
    track: mapData().actual,
    current: {
      latitude: 45.1,
      longitude: 160.4,
      altitudeHundredsFeet: 370,
      altitudeFeet: 37000,
      altitudeChange: "level",
      groundspeedKnots: 488,
      headingDegrees: 275,
      timestamp: "2026-04-29T00:05:00Z",
      updateType: "estimated",
      source: "flightaware_position",
    },
    cache: validCache(),
  };
}

function mapData({ flightId = "iflg_1" }: { flightId?: string } = {}): FlightMapDataResponse {
  return {
    flightId,
    faFlightId: "fa_1",
    planned: {
      source: "flightaware_route",
      available: true,
      geojson: {
        type: "Feature",
        geometry: {
          type: "LineString",
          coordinates: [
            [139.7, 35.6],
            [-73.8, 40.6],
          ],
        },
        properties: { kind: "planned_route" },
      },
    },
    actual: {
      source: "flightaware_track",
      available: true,
      geojson: {
        type: "Feature",
        geometry: {
          type: "LineString",
          coordinates: [
            [139.7, 35.6],
            [160.4, 45.1],
          ],
        },
        properties: { kind: "actual_track" },
      },
    },
    current: {
      source: "flightaware_position",
      available: true,
      geojson: {
        type: "Feature",
        geometry: { type: "Point", coordinates: [160.4, 45.1] },
        properties: { kind: "current_position", timestamp: "2026-04-29T00:05:00Z" },
      },
    },
    cache: validCache(),
  };
}
