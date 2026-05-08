const PUBLIC_STATUS_SEPARATOR = " / ";
const UNKNOWN_PUBLIC_STATUS = "Unknown";
const ASCII_EDGE_HORIZONTAL_SPACING = /^[ \t]+|[ \t]+$/g;
const ASCII_SLASH_WITH_OPTIONAL_HORIZONTAL_SPACING = /[ \t]*\/[ \t]*/;
const PRINTABLE_ASCII_STATUS_TEXT = /^[\x20-\x7E]*$/;

export function formatPublicFlightStatus(status: null): null;
export function formatPublicFlightStatus(status: undefined): undefined;
export function formatPublicFlightStatus(status: string): string;
export function formatPublicFlightStatus(
  status: string | null | undefined,
): string | null | undefined;
export function formatPublicFlightStatus(
  status: string | null | undefined,
): string | null | undefined {
  if (status == null) {
    return status;
  }

  const publicStatusParts = splitPublicStatusParts(trimPublicStatus(status));
  if (publicStatusParts.length === 0) {
    return UNKNOWN_PUBLIC_STATUS;
  }

  return publicStatusParts.map(formatPublicStatusPart).join(PUBLIC_STATUS_SEPARATOR);
}

function splitPublicStatusParts(status: string): string[] {
  return status
    .split(ASCII_SLASH_WITH_OPTIONAL_HORIZONTAL_SPACING)
    .filter((part) => part.length > 0);
}

function formatPublicStatusPart(status: string): string {
  return isPublicStatusText(status) ? status : UNKNOWN_PUBLIC_STATUS;
}

function trimPublicStatus(status: string): string {
  return status.replace(ASCII_EDGE_HORIZONTAL_SPACING, "");
}

function isPublicStatusText(status: string): boolean {
  return PRINTABLE_ASCII_STATUS_TEXT.test(status);
}
