import { describe, expect, it } from "vitest";

import {
  AIRPATH_MAP_THEME,
  actualTrackLinePaint,
  plannedRouteLinePaint,
  routeEndpointLabelPaint,
  routeLinePaintProperties,
} from "@/features/mapbox/lib/map-theme";

describe("map theme", () => {
  it("defines route and endpoint paint tokens", () => {
    expect(AIRPATH_MAP_THEME.routeLine).toEqual(expect.any(String));
    expect(AIRPATH_MAP_THEME.endpointLabel).toEqual(expect.any(String));
    expect(AIRPATH_MAP_THEME.endpointLabelHalo).toEqual(expect.any(String));
    expect(AIRPATH_MAP_THEME.routeLineEmissiveStrength).toBeGreaterThanOrEqual(1);
  });

  it("builds route line paint from reusable theme-token properties", () => {
    const sharedLinePaint = routeLinePaintProperties();

    expect(plannedRouteLinePaint()).toMatchObject({
      ...sharedLinePaint,
      "line-dasharray": expect.any(Array),
      "line-opacity": expect.any(Number),
      "line-width": expect.any(Number),
    });
    expect(actualTrackLinePaint()).toMatchObject({
      ...sharedLinePaint,
      "line-opacity": expect.any(Number),
      "line-width": expect.any(Number),
    });
  });

  it("keeps route line paint properties wired to shared tokens", () => {
    expect(routeLinePaintProperties()).toEqual({
      "line-color": AIRPATH_MAP_THEME.routeLine,
      "line-emissive-strength": AIRPATH_MAP_THEME.routeLineEmissiveStrength,
    });
  });

  it("builds endpoint label paint from reusable theme tokens", () => {
    expect(routeEndpointLabelPaint()).toMatchObject({
      "text-color": AIRPATH_MAP_THEME.endpointLabel,
      "text-halo-color": AIRPATH_MAP_THEME.endpointLabelHalo,
      "text-halo-width": expect.any(Number),
    });
  });
});
