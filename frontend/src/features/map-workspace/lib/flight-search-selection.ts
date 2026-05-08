import type { FlightSearchResponse, FlightSummaryItem } from "@/features/flights/types";

export function firstOrderedSearchResultFlightId(response: FlightSearchResponse): string | null {
  return response.items[0]?.flightId ?? null;
}

export function sortFlightSearchResponseByScheduledOut(
  response: FlightSearchResponse,
): FlightSearchResponse {
  return {
    ...response,
    items: sortFlightSearchItemsByScheduledOut(response.items),
  };
}

export function sortFlightSearchItemsByScheduledOut(
  items: readonly FlightSummaryItem[],
): FlightSummaryItem[] {
  return [...items].sort((left, right) => compareScheduledOut(left, right));
}

function compareScheduledOut(left: FlightSummaryItem, right: FlightSummaryItem): number {
  const leftTime = scheduledOutTime(left);
  const rightTime = scheduledOutTime(right);
  if (leftTime === null && rightTime === null) {
    return 0;
  }
  if (leftTime === null) {
    return 1;
  }
  if (rightTime === null) {
    return -1;
  }
  return leftTime - rightTime;
}

function scheduledOutTime(item: FlightSummaryItem): number | null {
  if (item.scheduledOut === null) {
    return null;
  }
  const timestamp = Date.parse(item.scheduledOut);
  return Number.isNaN(timestamp) ? null : timestamp;
}
