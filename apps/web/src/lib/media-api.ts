import type { MediaRecord, PhysicalFormat } from "@media-library/types";

export function getMediaTypeLabel(value: MediaRecord["mediaType"]): string {
  return value === "tv" ? "TV" : capitalizeLabel(value);
}

export function getPhysicalFormatLabel(value?: PhysicalFormat | null): string {
  if (!value) {
    return "Unspecified";
  }

  return value
    .split("_")
    .map((part) => part.toUpperCase() === "PS5" ? "PS5" : capitalizeLabel(part))
    .join(" ");
}

export function getMediaCreatorLine(media: MediaRecord): string | null {
  switch (media.mediaType) {
    case "album":
      return media.details.artists.join(", ") || null;
    case "book":
      return media.details.authors.join(", ") || null;
    case "movie":
      return media.details.directors?.join(", ") || null;
    case "tv":
      return media.details.creators?.join(", ") || null;
    case "game":
      return media.details.developers?.join(", ") || null;
  }
}

export function formatDateLabel(value?: string | null): string {
  if (!value) {
    return "Not set";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat("en-US", {
    dateStyle: "medium"
  }).format(date);
}

export function formatDateTimeLabel(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat("en-US", {
    dateStyle: "medium",
    timeStyle: "short"
  }).format(date);
}

function capitalizeLabel(value: string): string {
  return value.charAt(0).toUpperCase() + value.slice(1);
}
