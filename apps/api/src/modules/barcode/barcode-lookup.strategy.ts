import { isIsbnLikeBarcode } from "@media-library/provider-sdk";
import type { MediaType, ProviderName } from "@media-library/types";

export interface BarcodeLookupHint {
  inferredMediaType: MediaType | null;
  isIsbnLike: boolean;
  preferredMediaType: MediaType | null;
}

export interface BarcodeLookupStage {
  mediaType: MediaType;
  providers: ProviderName[];
  rankTier: 2 | 3;
}

const WEAK_BARCODE_COVERAGE_MEDIA_TYPES = new Set<MediaType>(["movie", "tv", "game"]);

export function inferBarcodeLookupHint(
  barcode: string,
  preferredMediaType?: MediaType
): BarcodeLookupHint {
  const isIsbn = isIsbnLikeBarcode(barcode);

  return {
    inferredMediaType: preferredMediaType ?? (isIsbn ? "book" : null),
    isIsbnLike: isIsbn,
    preferredMediaType: preferredMediaType ?? null
  };
}

export function getBarcodeLookupStages(
  hint: BarcodeLookupHint
): BarcodeLookupStage[] {
  const preferredMediaType = hint.preferredMediaType;

  if (preferredMediaType === "book") {
    return [{ mediaType: "book", providers: ["openlibrary"], rankTier: 2 }];
  }

  if (preferredMediaType === "album") {
    return [
      { mediaType: "album", providers: ["discogs"], rankTier: 2 },
      { mediaType: "album", providers: ["musicbrainz"], rankTier: 2 }
    ];
  }

  if (preferredMediaType && hasWeakBarcodeCoverage(preferredMediaType)) {
    return [];
  }

  if (hint.isIsbnLike) {
    return [
      { mediaType: "book", providers: ["openlibrary"], rankTier: 2 },
      { mediaType: "album", providers: ["discogs"], rankTier: 3 },
      { mediaType: "album", providers: ["musicbrainz"], rankTier: 3 }
    ];
  }

  return [
    { mediaType: "album", providers: ["discogs"], rankTier: 2 },
    { mediaType: "album", providers: ["musicbrainz"], rankTier: 2 }
  ];
}

export function getLocalLookupMediaType(
  preferredMediaType?: MediaType
): MediaType | undefined {
  return preferredMediaType;
}

export function hasWeakBarcodeCoverage(mediaType?: MediaType | null): boolean {
  return mediaType ? WEAK_BARCODE_COVERAGE_MEDIA_TYPES.has(mediaType) : false;
}
