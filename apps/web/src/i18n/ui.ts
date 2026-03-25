import type {
  LibraryBucket,
  MediaRecord,
  PhysicalFormat,
  ProviderName
} from "@media-library/types";

type Translate = {
  bivarianceHack(key: string, values?: unknown): string;
}["bivarianceHack"];
type FormatDateTime = {
  bivarianceHack(
    value: number | Date,
    formatOrOptions?: string | Intl.DateTimeFormatOptions,
    options?: Intl.DateTimeFormatOptions
  ): string;
}["bivarianceHack"];

export function formatDateLabel(
  formatDateTime: FormatDateTime,
  tCommon: Translate,
  value?: string | null
): string {
  if (!value) {
    return tCommon("notSet");
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return formatDateTime(date, "short");
}

export function formatDateTimeLabel(
  formatDateTime: FormatDateTime,
  value: string
): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return formatDateTime(date, "long");
}

export function getBucketLabel(tBucket: Translate, value: LibraryBucket): string {
  return tBucket(value);
}

export function getMediaTypeLabel(
  tMediaType: Translate,
  value: MediaRecord["mediaType"]
): string {
  return tMediaType(value);
}

export function getPhysicalFormatLabel(
  tPhysicalFormat: Translate,
  tCommon: Translate,
  value?: PhysicalFormat | null
): string {
  if (!value) {
    return tCommon("unspecified");
  }

  return tPhysicalFormat(value);
}

export function getProviderLabel(
  tProvider: Translate,
  provider: ProviderName
): string {
  return tProvider(provider);
}
