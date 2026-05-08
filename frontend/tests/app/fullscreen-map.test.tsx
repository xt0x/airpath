import { renderToStaticMarkup } from "react-dom/server";
import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

import Home from "@/app/(app)/page";
import { TooltipProvider } from "@/components/ui/tooltip";
import { FullscreenMap } from "@/features/mapbox/components/fullscreen-map";

describe("fullscreen Mapbox UI", () => {
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

  it("uses a shared non-white map surface background during sidebar resize", () => {
    const css = readFileSync("src/app/globals.css", "utf8");

    expect(css).toMatch(/:root\s*\{[^}]*--map-surface-background:\s*var\(--background\);/s);
    expect(css).toMatch(/\.dark\s*\{[^}]*--background:\s*oklch\(0\.145 0 0\);/s);
    expect(css).toMatch(/\.dark\s*\{[^}]*color-scheme:\s*dark;/s);
    expect(css).not.toContain("--map-surface-background: #d7dde4");
    expect(css).toMatch(/html,\s*body\s*\{[^}]*background:\s*var\(--map-surface-background\);/s);
    expect(css).toMatch(/body\s*\{[^}]*background:\s*var\(--map-surface-background\);/s);
    expect(css).toMatch(
      /\[data-slot="sidebar-wrapper"\]\s*\{[^}]*background:\s*var\(--map-surface-background\);/s,
    );
    expect(css).toMatch(
      /\.map-workspace__inset\s*\{[^}]*background:\s*var\(--map-surface-background\);/s,
    );
    expect(css).toMatch(/\.mapbox-screen\s*\{[^}]*background:\s*var\(--map-surface-background\);/s);
    expect(css).toMatch(
      /\.mapbox-screen__canvas,[^{]*\.mapboxgl-map,[^{]*\.mapboxgl-canvas-container,[^{]*\.mapboxgl-canvas\s*\{[^}]*background:\s*var\(--map-surface-background\);/s,
    );
  });

  it("does not reapply the generic white app background during reload", () => {
    const css = readFileSync("src/app/globals.css", "utf8");
    const layout = readFileSync("src/app/layout.tsx", "utf8");

    expect(css).not.toContain("@apply bg-background text-foreground");
    expect(css).toMatch(
      /@layer base[\s\S]*body\s*\{[\s\S]*background:\s*var\(--map-surface-background\);[\s\S]*color:\s*var\(--foreground\);/,
    );
    expect(layout).toContain('className={cn("dark font-sans", geist.variable)}');
    expect(layout).toContain('const SHADCN_DARK_BACKGROUND = "oklch(0.145 0 0)"');
    expect(layout).toContain("style={{ backgroundColor: SHADCN_DARK_BACKGROUND }}");
    expect(layout).not.toContain("rgb(30 30 30)");
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
});
