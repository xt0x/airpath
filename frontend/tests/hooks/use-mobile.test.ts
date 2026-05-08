import * as React from "react";
import { createRoot, type Root } from "react-dom/client";
import { renderToStaticMarkup } from "react-dom/server";
import { afterEach, beforeAll, describe, expect, it } from "vitest";

import { MOBILE_BREAKPOINT, isMobileViewport, useIsMobile } from "@/hooks/use-mobile";
import {
  createLegacyMediaQueryList,
  createModernMediaQueryList,
  installFakeDomGlobals,
} from "../helpers/fake-react-dom";

describe("mobile viewport hook", () => {
  let restoreFakeDomGlobals = () => {};

  beforeAll(() => {
    (globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;
  });

  afterEach(() => {
    restoreFakeDomGlobals();
    restoreFakeDomGlobals = () => {};
  });

  it("treats widths below the shared 640px breakpoint as mobile", () => {
    expect(isMobileViewport(0)).toBe(true);
    expect(isMobileViewport(MOBILE_BREAKPOINT - 1)).toBe(true);
    expect(isMobileViewport(MOBILE_BREAKPOINT)).toBe(false);
    expect(isMobileViewport(1024)).toBe(false);
  });

  it("rejects invalid viewport widths instead of treating them as mobile", () => {
    expect(isMobileViewport(-1)).toBe(false);
    expect(isMobileViewport(Number.NaN)).toBe(false);
    expect(isMobileViewport(Number.POSITIVE_INFINITY)).toBe(false);
  });

  it("renders the hook as desktop during server rendering", () => {
    function MobileProbe() {
      return React.createElement("span", null, useIsMobile() ? "mobile" : "desktop");
    }

    expect(renderToStaticMarkup(React.createElement(MobileProbe))).toBe("<span>desktop</span>");
  });

  it("reads the browser media query snapshot and updates after change events", async () => {
    restoreFakeDomGlobals = installFakeDomGlobals();
    const mediaQuery = createModernMediaQueryList(true);
    const matchMediaCalls: string[] = [];

    window.matchMedia = (query) => {
      matchMediaCalls.push(query);
      return mediaQuery as unknown as MediaQueryList;
    };

    const renders: boolean[] = [];
    const root = await renderMobileProbe(renders);

    expect(renders.at(-1)).toBe(true);
    expect(matchMediaCalls).toContain(`(max-width: ${MOBILE_BREAKPOINT - 1}px)`);

    await React.act(async () => {
      mediaQuery.setMatches(false);
    });

    expect(renders.at(-1)).toBe(false);

    await React.act(async () => {
      root.unmount();
    });

    expect(mediaQuery.listenerCount()).toBe(0);
  });

  it("falls back to legacy media query listeners when EventTarget listeners are unavailable", async () => {
    restoreFakeDomGlobals = installFakeDomGlobals();
    const mediaQuery = createLegacyMediaQueryList(false);

    window.matchMedia = () => mediaQuery as unknown as MediaQueryList;

    const renders: boolean[] = [];
    const root = await renderMobileProbe(renders);

    expect(renders.at(-1)).toBe(false);

    await React.act(async () => {
      mediaQuery.setMatches(true);
    });

    expect(renders.at(-1)).toBe(true);

    await React.act(async () => {
      root.unmount();
    });

    expect(mediaQuery.listenerCount()).toBe(0);
  });

  it("renders as desktop when the browser does not support matchMedia", async () => {
    restoreFakeDomGlobals = installFakeDomGlobals();

    const renders: boolean[] = [];
    const root = await renderMobileProbe(renders);

    expect(renders.at(-1)).toBe(false);

    await React.act(async () => {
      root.unmount();
    });
  });

  async function renderMobileProbe(renders: boolean[]): Promise<Root> {
    const container = document.createElement("div");
    const root = createRoot(container);

    function MobileProbe() {
      renders.push(useIsMobile());
      return React.createElement("span");
    }

    await React.act(async () => {
      root.render(React.createElement(MobileProbe));
    });

    return root;
  }
});
