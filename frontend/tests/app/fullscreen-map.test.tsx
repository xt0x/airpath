// @vitest-environment happy-dom

import * as React from "react";
import { createRoot, type Root } from "react-dom/client";
import { renderToStaticMarkup } from "react-dom/server";
import { readFileSync } from "node:fs";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import Home from "@/app/(app)/page";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { FlightMapDataResponse } from "@/features/flights/types";
import { FullscreenMap } from "@/features/mapbox/components/fullscreen-map";

const mapboxMock = vi.hoisted(() => {
  const instances: FakeMapboxMap[] = [];

  class FakeLngLatBounds {
    readonly first: [number, number];
    readonly extended: [number, number][];

    constructor(first: [number, number]) {
      this.first = first;
      this.extended = [first];
    }

    extend(coordinate: [number, number]) {
      this.extended.push(coordinate);
    }
  }

  class FakeNavigationControl {}

  class FakeMarker {
    addTo() {
      return this;
    }

    remove() {}

    setLngLat() {
      return this;
    }

    setRotation() {
      return this;
    }
  }

  class FakeMapboxMap {
    readonly fitBoundsCalls: Array<{ bounds: FakeLngLatBounds; options: Record<string, unknown> }> =
      [];
    readonly flyToCalls: Array<Record<string, unknown>> = [];

    constructor() {
      instances.push(this);
    }

    addControl() {}

    fitBounds(bounds: FakeLngLatBounds, options: Record<string, unknown>) {
      this.fitBoundsCalls.push({ bounds, options });
    }

    flyTo(options: Record<string, unknown>) {
      this.flyToCalls.push(options);
    }

    getZoom() {
      return 3.2;
    }

    isStyleLoaded() {
      return false;
    }

    once() {}

    on() {}

    remove() {}

    resize() {}
  }

  return {
    instances,
    mapboxgl: {
      accessToken: "",
      LngLatBounds: FakeLngLatBounds,
      Map: FakeMapboxMap,
      Marker: FakeMarker,
      NavigationControl: FakeNavigationControl,
    },
  };
});

vi.mock("mapbox-gl", () => ({
  default: mapboxMock.mapboxgl,
}));

let mountedRoot: Root | null = null;
let mountedContainer: HTMLDivElement | null = null;

describe("fullscreen Mapbox UI", () => {
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
    mapboxMock.instances.length = 0;
  });

  it("renders the root route as the fullscreen Mapbox surface", () => {
    const html = renderToStaticMarkup(
      <TooltipProvider>
        <Home />
      </TooltipProvider>,
    );

    expect(html).toContain('data-testid="fullscreen-map"');
    expect(html).toContain('data-map-provider="mapbox"');
  });

  it("keeps the map container sized to the full viewport", () => {
    const html = renderToStaticMarkup(
      <FullscreenMap
        accessToken="pk.test"
        styleURL="mapbox://styles/example/style-id"
        mapData={null}
        currentPosition={null}
      />,
    );

    expect(html).toContain('class="mapbox-screen"');
    expect(html).toContain('class="mapbox-screen__canvas"');
  });

  it("keeps a shared map surface background token in global CSS", () => {
    const css = readFileSync("src/app/globals.css", "utf8");

    expect(css).toContain("--map-surface-background");
    expect(css).not.toContain("--map-surface-background: #d7dde4");
  });

  it("does not reapply the generic white app background during reload", () => {
    const css = readFileSync("src/app/globals.css", "utf8");

    expect(css).not.toContain("@apply bg-background text-foreground");
  });

  it("exposes the configured Mapbox style URL on the map surface", () => {
    const html = renderToStaticMarkup(
      <FullscreenMap
        accessToken="pk.test"
        styleURL="mapbox://styles/x----x----x/cmopdqw6o000301sq099jg2zf"
        mapData={null}
        currentPosition={null}
      />,
    );

    expect(html).toContain(
      'data-map-style-url="mapbox://styles/x----x----x/cmopdqw6o000301sq099jg2zf"',
    );
  });

  it("renders the missing-token notice in the initial markup", () => {
    const html = renderToStaticMarkup(
      <FullscreenMap
        accessToken=""
        styleURL="mapbox://styles/example/style-id"
        mapData={null}
        currentPosition={null}
      />,
    );

    expect(html).toContain("Set NEXT_PUBLIC_MAPBOX_ACCESS_TOKEN to show the map.");
    expect(html).toContain('role="status"');
  });

  it("keeps a route focus request pending until route bounds are available", async () => {
    const { rerender } = await renderClientMap({
      accessToken: "pk.test",
      styleURL: "mapbox://styles/example/style-id",
      mapData: null,
      currentPosition: null,
      aircraftFocusRequest: 1,
    });
    const map = mapboxMock.instances[0];
    if (map === undefined) {
      throw new Error("Mapbox map was not constructed");
    }

    expect(map.fitBoundsCalls).toHaveLength(0);

    await rerender({
      accessToken: "pk.test",
      styleURL: "mapbox://styles/example/style-id",
      mapData: mapDataWithRoute(),
      currentPosition: null,
      aircraftFocusRequest: 1,
    });

    expect(map.fitBoundsCalls).toHaveLength(1);
    expect(map.fitBoundsCalls[0]?.bounds).toBeDefined();
  });
});

async function renderClientMap(props: React.ComponentProps<typeof FullscreenMap>) {
  mountedContainer = document.createElement("div");
  document.body.appendChild(mountedContainer);
  mountedRoot = createRoot(mountedContainer);

  await rerenderClientMap(props);

  return {
    rerender: rerenderClientMap,
  };
}

async function rerenderClientMap(props: React.ComponentProps<typeof FullscreenMap>) {
  await React.act(async () => {
    mountedRoot?.render(<FullscreenMap {...props} />);
    await Promise.resolve();
  });
  await React.act(async () => {
    await Promise.resolve();
  });
}

function mapDataWithRoute(): FlightMapDataResponse {
  return {
    flightId: "iflg_1",
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
        properties: {},
      },
    },
    actual: { source: "flightaware_track", available: false, geojson: null },
    current: { source: "flightaware_position", available: false, geojson: null },
    cache: {
      freshness: "fresh",
      source: "cache",
      stale: false,
      checkedAt: "2026-04-29T00:00:00Z",
    },
  };
}
