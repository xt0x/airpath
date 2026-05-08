export const DEFAULT_MAPBOX_STYLE_URL = "mapbox://styles/mapbox/streets-v12";

export type MapboxEnvironment = Partial<Record<string, string | undefined>>;

export function mapboxAccessToken(env: MapboxEnvironment = process.env): string {
  return env.NEXT_PUBLIC_MAPBOX_ACCESS_TOKEN?.trim() ?? "";
}

export function mapboxStyleURL(env: MapboxEnvironment = process.env): string {
  const configuredStyleURL = env.NEXT_PUBLIC_MAPBOX_STYLE_URL?.trim();

  return configuredStyleURL && configuredStyleURL.length > 0
    ? configuredStyleURL
    : DEFAULT_MAPBOX_STYLE_URL;
}
