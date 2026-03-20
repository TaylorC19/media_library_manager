import type {
  NormalizedMediaRecordInput,
  NormalizedSearchResult
} from "@media-library/types";
import { normalizeBarcodes } from "../utils/barcode.js";
import {
  createCreatorLine,
  normalizeOptionalText,
  normalizeStringArray,
  normalizeText,
  toPositiveInteger
} from "../utils/normalization.js";

export interface DiscogsSearchResultItem {
  id: number;
  title?: string;
  year?: number;
  country?: string;
  format?: string[];
  cover_image?: string;
  genre?: string[];
  label?: string[];
  barcode?: string[];
}

export interface DiscogsArtist {
  name?: string;
}

export interface DiscogsLabel {
  name?: string;
  catno?: string;
}

export interface DiscogsImage {
  uri?: string;
}

export interface DiscogsTrack {
  title?: string;
}

export interface DiscogsReleaseDetails {
  id: number;
  title: string;
  year?: number;
  country?: string;
  notes?: string;
  artists?: DiscogsArtist[];
  labels?: DiscogsLabel[];
  images?: DiscogsImage[];
  tracklist?: DiscogsTrack[];
  genres?: string[];
  barcode?: string[];
}

function splitDiscogsTitle(value?: string): {
  artists: string[];
  title: string | null;
} {
  const normalized = normalizeOptionalText(value);
  if (!normalized) {
    return {
      artists: [],
      title: null
    };
  }

  const parts = normalized.split(" - ");
  if (parts.length < 2) {
    return {
      artists: [],
      title: normalized
    };
  }

  const [artistSegment = "", ...titleParts] = parts;

  return {
    artists: normalizeStringArray(artistSegment.split(", ")),
    title: normalizeOptionalText(titleParts.join(" - "))
  };
}

export function mapDiscogsSearchResult(
  item: DiscogsSearchResultItem
): NormalizedSearchResult | null {
  const parsed = splitDiscogsTitle(item.title);

  if (!parsed.title) {
    return null;
  }

  return {
    provider: "discogs",
    providerId: String(item.id),
    mediaType: "album",
    title: parsed.title,
    subtitle: normalizeOptionalText(item.country),
    year: typeof item.year === "number" ? item.year : null,
    imageUrl: normalizeOptionalText(item.cover_image),
    summary: null,
    creatorLine: createCreatorLine(parsed.artists),
    barcodeCandidates: normalizeBarcodes(item.barcode),
    confidence: null
  };
}

export function mapDiscogsDetails(
  release: DiscogsReleaseDetails
): NormalizedMediaRecordInput {
  const artists = normalizeStringArray(release.artists?.map((artist) => artist.name));
  const title = normalizeText(release.title);

  return {
    mediaType: "album",
    title,
    sortTitle: title,
    releaseDate:
      typeof release.year === "number" ? String(release.year) : null,
    year: typeof release.year === "number" ? release.year : null,
    imageUrl: normalizeOptionalText(release.images?.[0]?.uri),
    summary: normalizeOptionalText(release.notes),
    providerRefs: {
      discogs: {
        id: String(release.id)
      }
    },
    barcodeCandidates: normalizeBarcodes(release.barcode),
    details: {
      artists,
      label: normalizeOptionalText(release.labels?.[0]?.name),
      trackCount: toPositiveInteger(release.tracklist?.length),
      releaseCountry: normalizeOptionalText(release.country),
      catalogNumber: normalizeOptionalText(release.labels?.[0]?.catno)
    }
  };
}
