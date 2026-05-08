import { describe, expect, it } from "vitest";

import {
  DEFAULT_MAPBOX_STYLE_URL,
  mapboxAccessToken,
  mapboxStyleURL,
} from "@/features/mapbox/config";

describe("mapbox config", () => {
  it("uses the configured public style URL when present", () => {
    expect(
      mapboxStyleURL({
        NEXT_PUBLIC_MAPBOX_STYLE_URL: "mapbox://styles/x----x----x/cmopdqw6o000301sq099jg2zf",
      }),
    ).toBe("mapbox://styles/x----x----x/cmopdqw6o000301sq099jg2zf");
  });

  it("falls back to the default Mapbox streets style", () => {
    expect(mapboxStyleURL({})).toBe(DEFAULT_MAPBOX_STYLE_URL);
  });

  it("trims whitespace from the configured style URL", () => {
    expect(
      mapboxStyleURL({
        NEXT_PUBLIC_MAPBOX_STYLE_URL: "  mapbox://styles/example/style-id  ",
      }),
    ).toBe("mapbox://styles/example/style-id");
  });

  it("reads the public Mapbox access token as a server-side snapshot value", () => {
    expect(mapboxAccessToken({ NEXT_PUBLIC_MAPBOX_ACCESS_TOKEN: "pk.test-token" })).toBe(
      "pk.test-token",
    );
  });

  it("falls back to an empty access token when unset", () => {
    expect(mapboxAccessToken({})).toBe("");
  });
});
