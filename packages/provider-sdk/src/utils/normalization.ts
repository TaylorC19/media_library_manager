export function normalizeText(value: string): string {
  return value.trim().replace(/\s+/g, " ");
}

export function normalizeOptionalText(value?: string | null): string | null {
  if (typeof value !== "string") {
    return null;
  }

  const normalized = normalizeText(value);
  return normalized.length > 0 ? normalized : null;
}

export function normalizeStringArray(
  values?: Array<string | null | undefined>
): string[] {
  if (!values) {
    return [];
  }

  const normalizedValues = values
    .map((value) => normalizeOptionalText(value))
    .filter((value): value is string => value !== null);

  return Array.from(new Set(normalizedValues));
}

export function extractYear(value?: string | null): number | null {
  if (!value) {
    return null;
  }

  const match = value.match(/\b(\d{4})\b/);
  if (!match) {
    return null;
  }

  const matchedYear = match[1];
  if (!matchedYear) {
    return null;
  }

  const year = Number.parseInt(matchedYear, 10);
  return Number.isFinite(year) ? year : null;
}

export function toPositiveInteger(
  value?: number | string | null
): number | null {
  if (typeof value === "number" && Number.isFinite(value) && value > 0) {
    return Math.trunc(value);
  }

  if (typeof value === "string" && value.trim().length > 0) {
    const parsed = Number.parseInt(value, 10);
    if (Number.isFinite(parsed) && parsed > 0) {
      return parsed;
    }
  }

  return null;
}

export function createCreatorLine(
  values?: Array<string | null | undefined>
): string | null {
  const creators = normalizeStringArray(values);
  return creators.length > 0 ? creators.join(", ") : null;
}
