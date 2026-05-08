export const AIRPATH_MAP_THEME = {
  routeLine: "#facc15",
  routeLineEmissiveStrength: 1,
  endpointLabel: "#111827",
  endpointLabelHalo: "#ffffff",
} as const;

export function plannedRouteLinePaint() {
  return {
    ...routeLinePaintProperties(),
    "line-dasharray": [2, 2],
    "line-opacity": 0.96,
    "line-width": 3,
  };
}

export function actualTrackLinePaint() {
  return {
    ...routeLinePaintProperties(),
    "line-opacity": 0.98,
    "line-width": 4,
  };
}

export function routeLinePaintProperties() {
  return {
    "line-color": AIRPATH_MAP_THEME.routeLine,
    "line-emissive-strength": AIRPATH_MAP_THEME.routeLineEmissiveStrength,
  };
}

export function routeEndpointLabelPaint() {
  return {
    "text-color": AIRPATH_MAP_THEME.endpointLabel,
    "text-halo-color": AIRPATH_MAP_THEME.endpointLabelHalo,
    "text-halo-width": 1.5,
  };
}
