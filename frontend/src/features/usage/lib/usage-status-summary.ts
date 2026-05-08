import { formatDisplayDateTime } from "@/lib/display-date-time";

import type { UsageStatusResponse } from "@/features/flights/types";

export interface UsageStatusSummary {
  budgetSummaryLabel: string;
  budgetPercent: number;
  stopReasons: string[];
  rateLimitResetLabel: string | null;
  lastCheckedLabel: string;
  monthLabel: string;
}

export interface MonthlyCostChartDatum {
  month: string;
  monthLabel: string;
  estimatedMonthToDateCost: number;
  estimatedMonthToDateCostLabel: string;
}

export function buildUsageStatusSummary(status: UsageStatusResponse): UsageStatusSummary {
  return {
    budgetSummaryLabel: `${formatUsageCurrency(status.budget.estimatedMonthToDateCost)} / ${formatUsageCurrency(status.budget.softStopThreshold)}`,
    budgetPercent: budgetPercent(
      status.budget.estimatedMonthToDateCost,
      status.budget.softStopThreshold,
    ),
    stopReasons: stopReasons(status),
    rateLimitResetLabel: status.rateLimit.limited
      ? formatDisplayDateTime(status.rateLimit.resetAt)
      : null,
    lastCheckedLabel: formatDisplayDateTime(status.cache.checkedAt),
    monthLabel: formatUsageMonthLabel(status.budget.month),
  };
}

export function buildMonthlyCostChartData(status: UsageStatusResponse): MonthlyCostChartDatum[] {
  return [
    {
      month: status.budget.month,
      monthLabel: formatUsageMonthLabel(status.budget.month),
      estimatedMonthToDateCost: status.budget.estimatedMonthToDateCost,
      estimatedMonthToDateCostLabel: formatUsageCurrency(status.budget.estimatedMonthToDateCost),
    },
  ];
}

export function formatUsageCurrency(value: number): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value);
}

function formatUsageMonthLabel(month: string): string {
  const parsed = new Date(`${month}-01T00:00:00Z`);
  if (Number.isNaN(parsed.getTime())) {
    return month;
  }
  return new Intl.DateTimeFormat("en", {
    month: "long",
    year: "numeric",
    timeZone: "UTC",
  }).format(parsed);
}

function budgetPercent(cost: number, threshold: number): number {
  if (threshold <= 0) {
    return cost > 0 ? 100 : 0;
  }
  return Math.max(0, Math.min(100, Math.round((cost / threshold) * 100)));
}

function stopReasons(status: UsageStatusResponse): string[] {
  const reasons: string[] = [];
  if (status.budget.stopped) {
    reasons.push("Budget stop");
  }
  if (status.rateLimit.limited) {
    reasons.push("Rate limit");
  }
  return reasons;
}
