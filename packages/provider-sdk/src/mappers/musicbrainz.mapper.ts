import type {
  NormalizedMediaRecordInput,
  NormalizedSearchResult
} from "@media-library/types";
import { normalizeBarcodes } from "../utils/barcode.js";
import {
  createCreatorLine,
  extractYear,
  normalizeOptionalText,
  normalizeStringArray,
  normalizeText,
  toPositiveInteger
} from "../utils/normalization.js";

export interface MusicBrainzArtistCredit {
  name?: string;
  artist?: {
    name?: string;
  };
}

export interface MusicBrainzLabelInfo {
  "catalog-number"?: string;
  label?: {
    name?: string;
  };
}

export interface MusicBrainzTrack {
  title?: string;
}

export interface MusicBrainzMedium {
  tracks?: MusicBrainzTrack[];
  "track-count"?: number;
}

export interface MusicBrainzRelease {
  id: string;
  title: string;
  date?: string;
  barcode?: string;
  country?: string;
  disambiguation?: string;
  "artist-credit"?: MusicBrainzArtistCredit[];
  "label-info"?: MusicBrainzLabelInfo[];
  media?: MusicBrainzMedium[];
}

function getArtistNames(artistCredit?: MusicBrainzArtistCredit[]): string[] {
  return normalizeStringArray(
    artistCredit?.map((credit) => credit.artist?.name ?? credit.name)
  );
}

function getTrackCount(media?: MusicBrainzMedium[]): number | null {
  if (!media || media.length === 0) {
    return null;
  }

  const count = media.reduce((total, medium) => {
    if (typeof medium["track-count"] === "number") {
      return total + medium["track-count"];
    }

    return total + (medium.tracks?.length ?? 0);
  }, 0);

  return toPositiveInteger(count);
}

export function mapMusicBrainzSearchResult(
  release: MusicBrainzRelease
): NormalizedSearchResult {
  const artists = getArtistNames(release["artist-credit"]);

  return {
    provider: "musicbrainz",
    providerId: release.id,
    mediaType: "album",
    title: normalizeText(release.title),
    subtitle: normalizeOptionalText(release.disambiguation),
    year: extractYear(release.date),
    imageUrl: null,
    summary: null,
    creatorLine: createCreatorLine(artists),
    barcodeCandidates: normalizeBarcodes([release.barcode]),
    confidence: null
  };
}

export function mapMusicBrainzDetails(
  release: MusicBrainzRelease
): NormalizedMediaRecordInput {
  const artists = getArtistNames(release["artist-credit"]);
  const labelInfo = release["label-info"] ?? [];

  return {
    mediaType: "album",
    title: normalizeText(release.title),
    sortTitle: normalizeText(release.title),
    releaseDate: normalizeOptionalText(release.date),
    year: extractYear(release.date),
    imageUrl: null,
    summary: normalizeOptionalText(release.disambiguation),
    providerRefs: {
      musicBrainz: {
        id: release.id
      }
    },
    barcodeCandidates: normalizeBarcodes([release.barcode]),
    details: {
      artists,
      label: normalizeOptionalText(labelInfo[0]?.label?.name),
      trackCount: getTrackCount(release.media),
      releaseCountry: normalizeOptionalText(release.country),
      catalogNumber: normalizeOptionalText(labelInfo[0]?.["catalog-number"])
    }
  };
}
