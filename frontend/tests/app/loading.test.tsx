import { readFileSync } from "node:fs";
import { join } from "node:path";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import Loading from "@/app/(app)/loading";

describe("route loading UI", () => {
  it("renders as a generic application loading state for every route in the app group", () => {
    const html = renderToStaticMarkup(<Loading />);

    expect(html).toContain('role="status"');
    expect(html).toContain('aria-label="Loading application"');
    expect(html).not.toContain("map workspace");
  });

  it("keeps route loading styles independent from map-only surface tokens", () => {
    const css = readFileSync(
      join(import.meta.dirname, "../../src/app/(app)/loading.module.css"),
      "utf8",
    );

    expect(css).not.toContain("--map-surface-background");
  });
});
