import type {
  NormalizedMediaRecordInput,
  NormalizedSearchResult
} from "@media-library/types";
import {
  createCreatorLine,
  extractYear,
  normalizeOptionalText,
  normalizeStringArray,
  normalizeText,
  toPositiveInteger
} from "../utils/normalization.js";

const TMDB_IMAGE_BASE_URL = "https://image.tmdb.org/t/p";

export interface TmdbSearchResultItem {
  id: number;
  title?: string;
  name?: string;
  overview?: string;
  poster_path?: string | null;
  release_date?: string;
  first_air_date?: string;
}

export interface TmdbPerson {
  name?: string;
  job?: string;
}

export interface TmdbCastMember {
  name?: string;
}

export interface TmdbDetailsResponse {
  id: number;
  title?: string;
  name?: string;
  overview?: string;
  poster_path?: string | null;
  release_date?: string;
  first_air_date?: string;
  runtime?: number | null;
  number_of_seasons?: number | null;
  number_of_episodes?: number | null;
  genres?: Array<{ name?: string }>;
  created_by?: Array<{ name?: string }>;
  credits?: {
    crew?: TmdbPerson[];
    cast?: TmdbCastMember[];
  };
  vote_average?: number | null;
}

function buildTmdbImageUrl(path?: string | null, size = "w342"): string | null {
  if (!path) {
    return null;
  }

  return `${TMDB_IMAGE_BASE_URL}/${size}${path}`;
}

export function mapTmdbSearchResult(
  item: TmdbSearchResultItem,
  mediaType: "movie" | "tv"
): NormalizedSearchResult {
  const title = normalizeText(item.title ?? item.name ?? "");
  const releaseDate = item.release_date ?? item.first_air_date ?? null;

  return {
    provider: "tmdb",
    providerId: String(item.id),
    mediaType,
    title,
    year: extractYear(releaseDate),
    imageUrl: buildTmdbImageUrl(item.poster_path),
    summary: normalizeOptionalText(item.overview),
    creatorLine: null,
    confidence: null
  };
}

export function mapTmdbDetails(
  details: TmdbDetailsResponse,
  mediaType: "movie" | "tv"
): NormalizedMediaRecordInput {
  const title = normalizeText(details.title ?? details.name ?? "");
  const releaseDate = details.release_date ?? details.first_air_date ?? null;
  const genres = normalizeStringArray(details.genres?.map((genre) => genre.name));
  const externalRatings =
    typeof details.vote_average === "number"
      ? { tmdb: details.vote_average }
      : undefined;

  if (mediaType === "movie") {
    const directors = normalizeStringArray(
      details.credits?.crew
        ?.filter((person) => person.job === "Director")
        .map((person) => person.name)
    );
    const cast = normalizeStringArray(
      details.credits?.cast?.slice(0, 8).map((person) => person.name)
    );

    return {
      mediaType: "movie",
      title,
      sortTitle: title,
      releaseDate,
      year: extractYear(releaseDate),
      imageUrl: buildTmdbImageUrl(details.poster_path, "w500"),
      summary: normalizeOptionalText(details.overview),
      providerRefs: {
        tmdb: {
          id: String(details.id),
          mediaKind: "movie"
        }
      },
      externalRatings,
      barcodeCandidates: [],
      details: {
        runtimeMinutes: toPositiveInteger(details.runtime),
        directors,
        cast,
        genres
      }
    };
  }

  const creators = normalizeStringArray(
    details.created_by?.map((creator) => creator.name)
  );

  return {
    mediaType: "tv",
    title,
    sortTitle: title,
    releaseDate,
    year: extractYear(releaseDate),
    imageUrl: buildTmdbImageUrl(details.poster_path, "w500"),
    summary: normalizeOptionalText(details.overview),
    providerRefs: {
      tmdb: {
        id: String(details.id),
        mediaKind: "tv"
      }
    },
    externalRatings,
    barcodeCandidates: [],
    details: {
      seasons: toPositiveInteger(details.number_of_seasons),
      episodes: toPositiveInteger(details.number_of_episodes),
      genres,
      creators
    }
  };
}

export function buildTmdbSubtitle(
  item: TmdbSearchResultItem,
  mediaType: "movie" | "tv"
): string | null {
  const releaseDate = item.release_date ?? item.first_air_date ?? null;
  const parts = [mediaType === "movie" ? "Movie" : "TV", extractYear(releaseDate)]
    .filter((value) => value !== null)
    .map((value) => String(value));

  return parts.length > 0 ? createCreatorLine(parts) : null;
}
