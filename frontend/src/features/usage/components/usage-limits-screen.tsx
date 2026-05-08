"use client";

import * as React from "react";

import { AirpathApiClient } from "@/features/flights/api/api-client";
import type { UsageStatusResponse } from "@/features/flights/types";
import {
  UsageLimitsView,
  type UsageLimitsViewState,
} from "@/features/usage/components/usage-limits-view";

export interface UsageStatusClient {
  getUsageStatus(): Promise<UsageStatusResponse>;
}

export function UsageLimitsScreen({ client }: { client?: UsageStatusClient }) {
  const usageClient = React.useMemo(() => client ?? new AirpathApiClient(), [client]);
  const [state, setState] = React.useState<UsageLimitsViewState>({ kind: "loading" });

  React.useEffect(() => {
    let cancelled = false;

    usageClient
      .getUsageStatus()
      .then((status) => {
        if (!cancelled) {
          setState({ kind: "ready", status });
        }
      })
      .catch((error: unknown) => {
        if (!cancelled) {
          setState({
            kind: "error",
            message: error instanceof Error ? error.message : "Unknown usage status error",
          });
        }
      });

    return () => {
      cancelled = true;
    };
  }, [usageClient]);

  return <UsageLimitsView state={state} />;
}
