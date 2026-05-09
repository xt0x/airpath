import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { existsSync, readFileSync } from "node:fs";

import Home from "@/app/(app)/page";
import { TooltipProvider } from "@/components/ui/tooltip";
import { MapWorkspace } from "@/features/map-workspace/components/map-workspace";
import { UsageWorkspace } from "@/features/usage/components/usage-workspace";

describe("sidebar layout", () => {
  it("wraps the map in a shadcn-style sidebar provider", () => {
    const html = renderToStaticMarkup(withTooltipProvider(<Home />));

    expect(html).toContain('data-slot="sidebar-wrapper"');
    expect(html).toContain('data-slot="sidebar"');
    expect(html).toContain('data-slot="sidebar-inset"');
    expect(html).toContain('data-slot="sidebar-trigger"');
    expect(html).toContain('data-testid="fullscreen-map"');
    expect(html.indexOf('data-slot="sidebar"')).toBeLessThan(
      html.indexOf('data-slot="sidebar-inset"'),
    );
  });

  it("renders the documented sidebar composition with header, content, and rail", () => {
    const html = renderToStaticMarkup(
      withTooltipProvider(
        <MapWorkspace accessToken="pk.test" styleURL="mapbox://styles/example/style-id" />,
      ),
    );

    expect(html).toContain('data-slot="sidebar-header"');
    expect(html).toContain('data-slot="sidebar-content"');
    expect(html).not.toContain('data-slot="sidebar-footer"');
    expect(html).toContain('data-slot="sidebar-group"');
    expect(html).toContain('data-slot="sidebar-group-label"');
    expect(html).toContain('data-slot="sidebar-menu"');
    expect(html).toContain('data-slot="sidebar-menu-item"');
    expect(html).toContain('data-slot="sidebar-menu-button"');
    expect(html).toContain('data-slot="sidebar-rail"');
    expect(html).toContain("Airpath Ops");
    expect(html).toContain("Workspace");
    expect(html).toContain("Data");
    expect(html).toContain("System");
    expect(html).toContain("Live Map");
    expect(html).toContain("Flight Search");
    expect(html).toContain("Tracked Flights");
    expect(html).toContain("Usage &amp; Limits");
    expect(html).toContain("Settings");
    expect(html).not.toContain("Map Layers");
    expect(html).not.toContain("Fetch Status");
    expect(html).not.toContain("Projects");
    expect(html).not.toContain("Documentation");
    expect(html).not.toContain("Saved Views");
    expect(html).not.toContain("Airpath</span>");
    expect(html).not.toContain("ops@airpath.local");
    expect(html).not.toContain("Upgrade to Pro");
    expect(html).not.toContain("Account");
    expect(html).not.toContain("Billing");
    expect(html).not.toContain("Notifications");
    expect(html).not.toContain("Log out");
    expect(html).not.toContain('href="/flight-search"');
    expect(html).not.toContain('href="/?flightSearch=1"');
    expect(html).toContain("Flight Search");
    expect(html).not.toContain('data-active="true"');
  });

  it("links Flight Search back to the home search state outside the Live Map workspace", () => {
    const html = renderToStaticMarkup(withTooltipProvider(<UsageWorkspace />));

    expect(html).toContain('href="/?flightSearch=1"');
    expect(html).not.toContain('href="/flight-search"');
  });

  it("renders Flight Search as a Live Map action when the workspace owns the panel", () => {
    const html = renderToStaticMarkup(
      withTooltipProvider(
        <MapWorkspace accessToken="pk.test" styleURL="mapbox://styles/example/style-id" />,
      ),
    );

    expect(html).toContain("Flight Search");
    expect(html).toContain('data-flight-search-trigger="true"');
    expect(html).not.toContain('href="/?flightSearch=1"');
  });

  it("renders sidebar action and route items through the shared menu", () => {
    const html = renderToStaticMarkup(
      withTooltipProvider(
        <MapWorkspace accessToken="pk.test" styleURL="mapbox://styles/example/style-id" />,
      ),
    );

    expect(html).toContain("Live Map");
    expect(html).toContain('href="/"');
    expect(html).toContain("Flight Search");
    expect(html).toContain('data-flight-search-trigger="true"');
    expect(html).toContain("Tracked Flights");
    expect(html).toContain('data-tracked-flights-trigger="true"');
    expect(html).toContain("Usage &amp; Limits");
    expect(html).toContain('href="/usage"');
  });

  it("marks the Usage route active in the shared sidebar", () => {
    const html = renderToStaticMarkup(withTooltipProvider(<UsageWorkspace />));

    expect(html).toContain('href="/usage"');
    expect(html).toContain('data-active="true"');
  });

  it("keeps app sidebar source under components as documented", () => {
    const mapWorkspace = readFileSync(
      "src/features/map-workspace/components/map-workspace.tsx",
      "utf8",
    );

    expect(existsSync("src/components/app-sidebar.tsx")).toBe(true);
    expect(mapWorkspace).toContain("@/components/app-sidebar");
    expect(existsSync("src/features/map-workspace/components/app-sidebar.tsx")).toBe(false);
  });

  it("renders Airpath Ops identity without a Teams hover card", () => {
    const html = renderToStaticMarkup(
      withTooltipProvider(
        <MapWorkspace accessToken="pk.test" styleURL="mapbox://styles/example/style-id" />,
      ),
    );

    expect(html).toContain("Airpath Ops");
    expect(html).not.toContain("Teams");
    expect(html).not.toContain("Flight Data");
    expect(html).not.toContain("Dispatch Desk");
    expect(html).not.toContain("Add team");
  });

  it("uses the official shadcn sidebar composition", () => {
    const html = renderToStaticMarkup(
      withTooltipProvider(
        <MapWorkspace accessToken="pk.test" styleURL="mapbox://styles/example/style-id" />,
      ),
    );
    const css = readFileSync("src/app/globals.css", "utf8");

    expect(html).toContain("map-workspace__inset");
    expect(html).toContain("map-workspace__toolbar");
    expect(css).not.toContain(".map-workspace__sidebar-shell");
  });

  it("keeps map-specific overlay behavior on the provider instead of extending Sidebar props", () => {
    const mapHtml = renderToStaticMarkup(
      withTooltipProvider(
        <MapWorkspace accessToken="pk.test" styleURL="mapbox://styles/example/style-id" />,
      ),
    );
    const usageHtml = renderToStaticMarkup(withTooltipProvider(<UsageWorkspace />));
    const css = readFileSync("src/app/globals.css", "utf8");

    expect(mapHtml).toContain("map-sidebar-layout");
    expect(usageHtml).not.toContain("map-sidebar-layout");
    expect(css).toContain('.map-sidebar-layout [data-slot="sidebar-gap"]');
    expect(css).not.toContain(".map-sidebar__identity-button");
    expect(css).not.toContain(".map-sidebar__action-button");
  });

  it("keeps the Usage sidebar provider as the reserved layout wrapper", () => {
    const usageHtml = renderToStaticMarkup(withTooltipProvider(<UsageWorkspace />));
    const usageCss = readFileSync(
      "src/features/usage/components/usage-workspace.module.css",
      "utf8",
    );

    expect(usageHtml).toContain("usage-sidebar-layout");
    expect(usageHtml).toContain("usage-workspace__inset");
    expect(usageCss).not.toContain("display: contents");
  });
});

function withTooltipProvider(children: React.ReactNode) {
  return <TooltipProvider>{children}</TooltipProvider>;
}
