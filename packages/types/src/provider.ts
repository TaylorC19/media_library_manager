import type { MediaType } from "./library";
import type { ExternalRatings, ProviderRefs } from "./media";

export const providerNames = [
  "tmdb",
  "musicbrainz",
  "discogs",
  "openlibrary",
  "rawg"
] as const;

export type ProviderName = (typeof providerNames)[number];

export interface NormalizedSearchResult {
  provider: ProviderName;
  providerId: string;
  mediaType: MediaType;
  title: string;
  subtitle?: string | null;
  year?: number | null;
  imageUrl?: string | null;
  summary?: string | null;
  creatorLine?: string | null;
  barcodeCandidates?: string[];
  confidence?: number | null;
}

export interface NormalizedMediaRecordInput {
  mediaType: MediaType;
  title: string;
  sortTitle?: string | null;
  releaseDate?: string | null;
  year?: number | null;
  imageUrl?: string | null;
  summary?: string | null;
  providerRefs: ProviderRefs;
  externalRatings?: ExternalRatings;
  barcodeCandidates?: string[];
  details: Record<string, unknown>;
}
