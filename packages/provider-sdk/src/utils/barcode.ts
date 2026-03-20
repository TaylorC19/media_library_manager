function normalizeSingleBarcode(value: string): string {
  return value
    .trim()
    .toUpperCase()
    .replace(/[\s-]+/g, "")
    .replace(/[^0-9X]/g, "");
}

export function normalizeBarcode(value?: string | null): string | null {
  if (typeof value !== "string") {
    return null;
  }

  const normalized = normalizeSingleBarcode(value);
  return normalized.length > 0 ? normalized : null;
}

export function normalizeBarcodes(
  values?: Array<string | null | undefined>
): string[] {
  if (!values) {
    return [];
  }

  const normalized = values
    .map((value) => normalizeBarcode(value))
    .filter((value): value is string => value !== null);

  return Array.from(new Set(normalized));
}

export function isIsbnLikeBarcode(value?: string | null): boolean {
  const normalized = normalizeBarcode(value);
  return normalized !== null && (normalized.length === 10 || normalized.length === 13);
}
