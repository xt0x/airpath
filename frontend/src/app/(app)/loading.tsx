import { Progress } from "@/components/ui/progress";

export default function Loading() {
  return (
    <main className="map-reload-progress" role="status" aria-label="Loading map workspace">
      <Progress value={66} className="map-reload-progress__bar w-[60%]" />
    </main>
  );
}
