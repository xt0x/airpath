"use client";

import { AlertTriangle } from "lucide-react";
import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";
import { Progress } from "@/components/ui/progress";

import type { UsageStatusResponse } from "@/features/flights/types";
import {
  buildMonthlyCostChartData,
  buildUsageStatusSummary,
  formatUsageCurrency,
} from "@/features/usage/lib/usage-status-summary";

export type UsageLimitsViewState =
  | { kind: "loading" }
  | { kind: "error"; message: string }
  | { kind: "ready"; status: UsageStatusResponse };

const usageCostChartConfig = {
  estimatedMonthToDateCost: {
    label: "Usage cost",
    color: "var(--chart-2)",
  },
} satisfies ChartConfig;

export function UsageLimitsView({ state }: { state: UsageLimitsViewState }) {
  const showAvailable =
    state.kind === "ready" &&
    state.status.fetchingEnabled &&
    !state.status.budget.stopped &&
    !state.status.rateLimit.limited;

  return (
    <main className="usage-limits" aria-busy={state.kind === "loading" ? "true" : undefined}>
      <div className="usage-limits__header">
        <h1>Usage &amp; Limits</h1>
        {showAvailable ? <AvailableStatus /> : null}
      </div>
      {state.kind === "error" ? <UsageStatusError message={state.message} /> : null}
      {state.kind === "ready" ? <UsageStatusOverview status={state.status} /> : null}
    </main>
  );
}

function AvailableStatus() {
  return (
    <span className="usage-limits__header-status" aria-label="Overall fetch status is available">
      <span
        className="usage-limits__header-status-dot usage-limits__status-dot--available"
        aria-hidden="true"
      />
      Available
    </span>
  );
}

function UsageStatusError({ message }: { message: string }) {
  return (
    <Alert variant="destructive" className="usage-limits__stop-alert">
      <AlertTriangle aria-hidden="true" />
      <AlertTitle>Usage status unavailable</AlertTitle>
      <AlertDescription>{message}</AlertDescription>
    </Alert>
  );
}

function UsageStatusOverview({ status }: { status: UsageStatusResponse }) {
  const summary = buildUsageStatusSummary(status);
  const monthlyCostData = buildMonthlyCostChartData(status);

  return (
    <div className="usage-limits__content">
      <Card className="usage-limits__panel" aria-label="API cost">
        <CardHeader>
          <CardTitle>API cost</CardTitle>
          <CardDescription>{summary.budgetSummaryLabel}</CardDescription>
          <CardAction>
            <Badge variant="outline">{summary.budgetPercent}% used</Badge>
          </CardAction>
        </CardHeader>
        <CardContent className="usage-limits__budget-content">
          <Progress value={summary.budgetPercent} className="usage-limits__budget-progress" />
        </CardContent>
      </Card>

      {summary.stopReasons.length > 0 ? (
        <Alert variant="destructive" className="usage-limits__stop-alert">
          <AlertTriangle aria-hidden="true" />
          <AlertTitle>Stop reason</AlertTitle>
          <AlertDescription>
            <span>{summary.stopReasons.join(" / ")}</span>
            {summary.rateLimitResetLabel ? (
              <span>Rate limit reset: {summary.rateLimitResetLabel}</span>
            ) : null}
          </AlertDescription>
        </Alert>
      ) : null}

      <Card
        className="usage-limits__panel"
        aria-label={`API usage cost by month for ${summary.monthLabel}`}
      >
        <CardHeader>
          <CardTitle>API usage cost</CardTitle>
          <CardDescription>{summary.monthLabel}</CardDescription>
        </CardHeader>
        <CardContent>
          <ChartContainer config={usageCostChartConfig} className="usage-limits__chart">
            <BarChart accessibilityLayer data={monthlyCostData}>
              <CartesianGrid vertical={false} />
              <XAxis dataKey="monthLabel" tickLine={false} axisLine={false} />
              <YAxis hide />
              <ChartTooltip
                content={
                  <ChartTooltipContent
                    labelFormatter={(_, payload) => {
                      const monthLabel = payload[0]?.payload?.monthLabel;
                      return typeof monthLabel === "string" ? monthLabel : "";
                    }}
                    formatter={(value) => formatUsageCurrency(Number(value))}
                  />
                }
              />
              <Bar
                dataKey="estimatedMonthToDateCost"
                fill="var(--color-estimatedMonthToDateCost)"
                radius={4}
              />
            </BarChart>
          </ChartContainer>
          <div className="usage-limits__chart-meta">
            <span>Last checked</span>
            <strong>{summary.lastCheckedLabel}</strong>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
