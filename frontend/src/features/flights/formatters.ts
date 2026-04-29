import type { ApiError } from "./types";

export function errorMessage(caught: unknown): string {
  if (typeof caught === "object" && caught !== null && "error" in caught) {
    const apiError = caught as { error?: ApiError };
    if (apiError.error !== undefined) {
      return apiError.error.message;
    }
  }
  return caught instanceof Error ? caught.message : "Request failed";
}

export function airportLabel(airport: { code: string; name: string | null }) {
  return airport.name === null ? airport.code : `${airport.code} - ${airport.name}`;
}

export function formatTime(value: string | null) {
  if (value === null) {
    return "Not announced";
  }
  return value.replace("T", " ").replace("Z", " UTC");
}
