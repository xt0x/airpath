import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import {
  FlightDashboardView,
  FlightMap,
  RefreshControls,
  SearchPanel,
  StaleDataNotice,
  SummaryPanel,
  UsageStatusBanner,
} from "./flight-dashboard";
import { sampleDetail, sampleMapData, sampleSearch, sampleUsage } from "./sample-data";

describe("flight dashboard UI", () => {
  it("renders search input and selectable flight results", () => {
    const html = renderToStaticMarkup(
      <SearchPanel
        error={null}
        isSearching={false}
        query="ANA110"
        results={sampleSearch.items}
        selectedFlightId="iflg_1"
        onQueryChange={() => undefined}
        onSearch={() => undefined}
        onSelectFlight={() => undefined}
      />,
    );

    expect(html).toContain('name="ident"');
    expect(html).toContain("ANA110");
    expect(html).toContain("RJTT");
    expect(html).toContain("KJFK");
    expect(html).toContain('aria-current="true"');
  });

  it("renders summary, freshness, and stale cache state", () => {
    const summary = renderToStaticMarkup(<SummaryPanel detail={sampleDetail} />);
    const stale = renderToStaticMarkup(
      <StaleDataNotice cache={{ ...sampleDetail.cache, freshness: "stale", stale: true }} />,
    );

    expect(summary).toContain("En Route");
    expect(summary).toContain("Tokyo Haneda");
    expect(summary).toContain("B789");
    expect(summary).toContain("fresh");
    expect(stale).toContain("Cached data");
    expect(stale).toContain("stale");
  });

  it("renders map route, track, and current-position layers", () => {
    const html = renderToStaticMarkup(<FlightMap mapData={sampleMapData} />);

    expect(html).toContain('data-renderer="mapbox-deckgl-compatible"');
    expect(html).toContain('data-deck-layer-count="3"');
    expect(html).toContain('data-layer="planned"');
    expect(html).toContain('data-layer="actual"');
    expect(html).toContain('data-layer="current"');
    expect(html).toContain("<svg");
  });

  it("disables manual refresh when budget or rate guards are active", () => {
    const stopped = renderToStaticMarkup(
      <RefreshControls
        disabled={false}
        isRefreshing={false}
        usage={{ ...sampleUsage, fetchingEnabled: false }}
        onRefresh={() => undefined}
      />,
    );
    const active = renderToStaticMarkup(
      <RefreshControls
        disabled={false}
        isRefreshing={false}
        usage={sampleUsage}
        onRefresh={() => undefined}
      />,
    );

    expect(stopped).toContain("disabled");
    expect(active).not.toContain("disabled");
  });

  it("renders usage state and the complete dashboard shell", () => {
    const usage = renderToStaticMarkup(<UsageStatusBanner usage={sampleUsage} />);
    const dashboard = renderToStaticMarkup(
      <FlightDashboardView
        detail={sampleDetail}
        error={null}
        isRefreshing={false}
        isSearching={false}
        mapData={sampleMapData}
        query="ANA110"
        searchResults={sampleSearch.items}
        selectedFlightId="iflg_1"
        usage={sampleUsage}
        onQueryChange={() => undefined}
        onRefresh={() => undefined}
        onSearch={() => undefined}
        onSelectFlight={() => undefined}
      />,
    );

    expect(usage).toContain("3.00 USD remaining");
    expect(dashboard).toContain("Airpath");
    expect(dashboard).toContain("ANA110");
    expect(dashboard).toContain("3.00 USD remaining");
  });

  it("renders the personal non-commercial low-frequency demo notice", () => {
    const dashboard = renderToStaticMarkup(
      <FlightDashboardView
        detail={sampleDetail}
        error={null}
        isRefreshing={false}
        isSearching={false}
        mapData={sampleMapData}
        query="ANA110"
        searchResults={sampleSearch.items}
        selectedFlightId="iflg_1"
        usage={sampleUsage}
        onQueryChange={() => undefined}
        onRefresh={() => undefined}
        onSearch={() => undefined}
        onSelectFlight={() => undefined}
      />,
    );

    expect(dashboard).toContain("Personal non-commercial demo");
    expect(dashboard).toContain("low-frequency");
  });
});
