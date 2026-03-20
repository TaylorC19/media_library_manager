import type {
  NormalizedMediaRecordInput,
  NormalizedSearchResult
} from "@media-library/types";
import {
  extractYear,
  normalizeOptionalText,
  normalizeStringArray,
  normalizeText
} from "../utils/normalization.js";

export interface RawgNamedEntity {
  name?: string;
}

export interface RawgSearchResultItem {
  id: number;
  name: string;
  released?: string;
  background_image?: string | null;
  rating?: number | null;
}

export interface RawgDetailsResponse {
  id: number;
  name: string;
  description_raw?: string;
  released?: string;
  background_image?: string | null;
  rating?: number | null;
  metacritic?: number | null;
  platforms?: Array<{ platform?: RawgNamedEntity }>;
  developers?: RawgNamedEntity[];
  publishers?: RawgNamedEntity[];
  genres?: RawgNamedEntity[];
}

export function mapRawgSearchResult(
  item: RawgSearchResultItem
): NormalizedSearchResult {
  return {
    provider: "rawg",
    providerId: String(item.id),
    mediaType: "game",
    title: normalizeText(item.name),
    subtitle: null,
    year: extractYear(item.released),
    imageUrl: item.background_image ?? null,
    summary: null,
    creatorLine: null,
    confidence: typeof item.rating === "number" ? item.rating : null
  };
}

export function mapRawgDetails(
  game: RawgDetailsResponse
): NormalizedMediaRecordInput {
  const title = normalizeText(game.name);

  return {
    mediaType: "game",
    title,
    sortTitle: title,
    releaseDate: normalizeOptionalText(game.released),
    year: extractYear(game.released),
    imageUrl: game.background_image ?? null,
    summary: normalizeOptionalText(game.description_raw),
    providerRefs: {
      rawg: {
        id: String(game.id)
      }
    },
    externalRatings: {
      metacritic:
        typeof game.metacritic === "number" ? game.metacritic : null
    },
    barcodeCandidates: [],
    details: {
      platforms: normalizeStringArray(
        game.platforms?.map((platform) => platform.platform?.name)
      ),
      developers: normalizeStringArray(
        game.developers?.map((developer) => developer.name)
      ),
      publishers: normalizeStringArray(
        game.publishers?.map((publisher) => publisher.name)
      ),
      genres: normalizeStringArray(game.genres?.map((genre) => genre.name))
    }
  };
}
