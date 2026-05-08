const displayDateTimeOptions = {
  year: "numeric",
  month: "short",
  day: "numeric",
  hour: "numeric",
  minute: "2-digit",
  timeZone: "UTC",
  timeZoneName: "short",
} satisfies Intl.DateTimeFormatOptions;

const displayDateTimeFormatter = new Intl.DateTimeFormat("en-US", displayDateTimeOptions);

export function formatDisplayDateTime(
  value: string | null | undefined,
  fallback = "Unavailable",
): string {
  const timestamp = value?.trim();
  if (!timestamp) {
    return fallback;
  }

  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) {
    return fallback;
  }

  return displayDateTimeFormatter.format(date);
}
