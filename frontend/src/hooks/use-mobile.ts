import * as React from "react";

export const MOBILE_BREAKPOINT = 640;
const MOBILE_MEDIA_QUERY = `(max-width: ${MOBILE_BREAKPOINT - 1}px)`;

type MobileViewportListener = () => void;
type Unsubscribe = () => void;

type MobileViewportStore = {
  getSnapshot: () => boolean;
  subscribe: (onStoreChange: MobileViewportListener) => Unsubscribe;
};

type MobileViewportTarget = {
  matchMedia: (query: string) => MobileMediaQueryList;
};

type MobileMediaQueryList = {
  matches: boolean;
  addEventListener?: (event: "change", listener: MobileViewportListener) => void;
  removeEventListener?: (event: "change", listener: MobileViewportListener) => void;
  addListener?: (listener: MobileViewportListener) => void;
  removeListener?: (listener: MobileViewportListener) => void;
};

const browserMobileViewportStore = createMobileViewportStore(getBrowserViewport);

export function isMobileViewport(width: number): boolean {
  return Number.isFinite(width) && width >= 0 && width < MOBILE_BREAKPOINT;
}

export function useIsMobile(): boolean {
  return React.useSyncExternalStore(
    browserMobileViewportStore.subscribe,
    browserMobileViewportStore.getSnapshot,
    getServerMobileSnapshot,
  );
}

function getServerMobileSnapshot(): boolean {
  return false;
}

function createMobileViewportStore(
  resolveViewport: () => MobileViewportTarget | undefined,
): MobileViewportStore {
  return {
    getSnapshot: () => readMobileViewport(resolveViewport()),
    subscribe: (onStoreChange) => subscribeToMobileViewport(resolveViewport(), onStoreChange),
  };
}

function readMobileViewport(viewport: MobileViewportTarget | undefined): boolean {
  const mediaQueryList = getMobileMediaQueryList(viewport);
  return mediaQueryList?.matches ?? false;
}

function subscribeToMobileViewport(
  viewport: MobileViewportTarget | undefined,
  onStoreChange: MobileViewportListener,
): Unsubscribe {
  const mediaQueryList = getMobileMediaQueryList(viewport);

  if (!mediaQueryList) {
    return noop;
  }

  return subscribeToMediaQueryList(mediaQueryList, onStoreChange);
}

function subscribeToMediaQueryList(
  mediaQueryList: MobileMediaQueryList,
  listener: MobileViewportListener,
): Unsubscribe {
  if (supportsEventTargetListeners(mediaQueryList)) {
    mediaQueryList.addEventListener("change", listener);
    return () => mediaQueryList.removeEventListener("change", listener);
  }

  if (supportsLegacyListeners(mediaQueryList)) {
    mediaQueryList.addListener(listener);
    return () => mediaQueryList.removeListener(listener);
  }

  return noop;
}

function supportsEventTargetListeners(
  mediaQueryList: MobileMediaQueryList,
): mediaQueryList is MobileMediaQueryList &
  Required<Pick<MobileMediaQueryList, "addEventListener" | "removeEventListener">> {
  return (
    typeof mediaQueryList.addEventListener === "function" &&
    typeof mediaQueryList.removeEventListener === "function"
  );
}

function supportsLegacyListeners(
  mediaQueryList: MobileMediaQueryList,
): mediaQueryList is MobileMediaQueryList &
  Required<Pick<MobileMediaQueryList, "addListener" | "removeListener">> {
  return (
    typeof mediaQueryList.addListener === "function" &&
    typeof mediaQueryList.removeListener === "function"
  );
}

function getMobileMediaQueryList(
  viewport: MobileViewportTarget | undefined,
): MobileMediaQueryList | undefined {
  return viewport?.matchMedia(MOBILE_MEDIA_QUERY);
}

function getBrowserViewport(): MobileViewportTarget | undefined {
  if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
    return undefined;
  }

  return window;
}

function noop() {}
