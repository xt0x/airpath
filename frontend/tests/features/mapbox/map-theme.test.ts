import { describe, expect, it } from "vitest";

import {
  actualTrackLinePaint,
  plannedRouteLinePaint,
  routeEndpointLabelPaint,
} from "@/features/mapbox/lib/map-theme";

describe("map theme", () => {
  it("keeps planned and actual route lines visually distinct while sharing the aircraft yellow", () => {
    expect(plannedRouteLinePaint()).toMatchObject({
      "line-color": "#facc15",
      "line-dasharray": [2, 2],
      "line-emissive-strength": 1,
    });
    expect(actualTrackLinePaint()).toMatchObject({
      "line-color": "#facc15",
      "line-emissive-strength": 1,
    });
    expect(actualTrackLinePaint()).not.toHaveProperty("line-dasharray");
    expect(actualTrackLinePaint()["line-width"]).toBeGreaterThan(
      plannedRouteLinePaint()["line-width"],
    );
  });

  it("uses a readable native symbol label paint for route endpoints", () => {
    expect(routeEndpointLabelPaint()).toEqual({
      "text-color": "#111827",
      "text-halo-color": "#ffffff",
      "text-halo-width": 1.5,
    });
  });
});
