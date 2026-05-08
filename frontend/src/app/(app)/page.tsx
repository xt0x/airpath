import { MapWorkspace } from "@/features/map-workspace/components/map-workspace";
import { mapboxAccessToken, mapboxStyleURL } from "@/features/mapbox/config";

export default function Home() {
  return <MapWorkspace accessToken={mapboxAccessToken()} styleURL={mapboxStyleURL()} />;
}
