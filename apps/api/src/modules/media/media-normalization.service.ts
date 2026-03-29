import { Injectable } from "@nestjs/common";
import type {
  CreateManualMediaRecordRequest,
  MediaRecord,
  NormalizedMediaRecordInput
} from "@media-library/types";
import type {
  CreateMediaRecordInput,
  UpdateMediaRecordInput
} from "./repositories/media-record.repository";
import {
  normalizeOptionalText,
  normalizeStringList,
  normalizeText
} from "./media-record.utils";

@Injectable()
export class MediaNormalizationService {
  toCreateInputFromManual(
    input: CreateManualMediaRecordRequest
  ): CreateMediaRecordInput {
    const normalizedTitle = normalizeText(input.title);

    return {
      source: "manual",
      mediaType: input.mediaType,
      title: normalizedTitle,
      sortTitle: normalizedTitle,
      releaseDate: normalizeOptionalText(input.releaseDate),
      year: input.year ?? null,
      imageUrl: normalizeOptionalText(input.imageUrl),
      summary: normalizeOptionalText(input.summary),
      providerRefs: {},
      externalRatings: undefined,
      barcodeCandidates: normalizeStringList(input.barcodeCandidates),
      details: this.normalizeManualDetails(input),
      lastSyncedAt: null
    };
  }

  toCreateInputFromProvider(
    input: NormalizedMediaRecordInput
  ): CreateMediaRecordInput {
    const normalizedTitle = normalizeText(input.title);

    return {
      source: "provider",
      mediaType: input.mediaType,
      title: normalizedTitle,
      sortTitle: normalizeOptionalText(input.sortTitle) ?? normalizedTitle,
      releaseDate: normalizeOptionalText(input.releaseDate),
      year: input.year ?? null,
      imageUrl: normalizeOptionalText(input.imageUrl),
      summary: normalizeOptionalText(input.summary),
      providerRefs: input.providerRefs,
      externalRatings: input.externalRatings,
      barcodeCandidates: normalizeStringList(input.barcodeCandidates),
      details: this.normalizeProviderDetails(input),
      lastSyncedAt: new Date().toISOString()
    };
  }

  mergeProviderData(
    existing: MediaRecord,
    input: NormalizedMediaRecordInput
  ): UpdateMediaRecordInput {
    const providerInput = this.toCreateInputFromProvider(input);

    return {
      source: "provider",
      mediaType: existing.mediaType,
      title: providerInput.title,
      sortTitle: coalesceString(providerInput.sortTitle, existing.sortTitle),
      releaseDate: coalesceString(providerInput.releaseDate, existing.releaseDate),
      year: coalesceNumber(providerInput.year, existing.year),
      imageUrl: coalesceString(providerInput.imageUrl, existing.imageUrl),
      summary: coalesceString(providerInput.summary, existing.summary),
      providerRefs: {
        ...existing.providerRefs,
        ...providerInput.providerRefs
      },
      externalRatings: mergeRatings(existing.externalRatings, providerInput.externalRatings),
      barcodeCandidates: mergeStringLists(
        existing.barcodeCandidates,
        providerInput.barcodeCandidates
      ),
      details: mergeDetails(existing, providerInput),
      lastSyncedAt: providerInput.lastSyncedAt
    };
  }

  private normalizeManualDetails(
    input: CreateManualMediaRecordRequest
  ): CreateMediaRecordInput["details"] {
    switch (input.mediaType) {
      case "movie":
        return {
          cast: normalizeStringList(input.details?.cast),
          directors: normalizeStringList(input.details?.directors),
          genres: normalizeStringList(input.details?.genres),
          runtimeMinutes: input.details?.runtimeMinutes ?? null
        };
      case "tv":
        return {
          creators: normalizeStringList(input.details?.creators),
          episodes: input.details?.episodes ?? null,
          genres: normalizeStringList(input.details?.genres),
          seasons: input.details?.seasons ?? null
        };
      case "album":
        return {
          artists: normalizeStringList(input.details.artists),
          catalogNumber: normalizeOptionalText(input.details.catalogNumber),
          label: normalizeOptionalText(input.details.label),
          releaseCountry: normalizeOptionalText(input.details.releaseCountry),
          trackCount: input.details.trackCount ?? null
        };
      case "book":
        return {
          authors: normalizeStringList(input.details.authors),
          isbn10: normalizeOptionalText(input.details.isbn10),
          isbn13: normalizeOptionalText(input.details.isbn13),
          pageCount: input.details.pageCount ?? null,
          publisher: normalizeOptionalText(input.details.publisher)
        };
      case "game":
        return {
          developers: normalizeStringList(input.details?.developers),
          genres: normalizeStringList(input.details?.genres),
          platforms: normalizeStringList(input.details?.platforms),
          publishers: normalizeStringList(input.details?.publishers)
        };
    }
  }

  private normalizeProviderDetails(
    input: NormalizedMediaRecordInput
  ): CreateMediaRecordInput["details"] {
    switch (input.mediaType) {
      case "movie":
        return {
          cast: normalizeStringList(input.details.cast),
          directors: normalizeStringList(input.details.directors),
          genres: normalizeStringList(input.details.genres),
          runtimeMinutes: input.details.runtimeMinutes ?? null
        };
      case "tv":
        return {
          creators: normalizeStringList(input.details.creators),
          episodes: input.details.episodes ?? null,
          genres: normalizeStringList(input.details.genres),
          seasons: input.details.seasons ?? null
        };
      case "album":
        return {
          artists: normalizeStringList(input.details.artists),
          catalogNumber: normalizeOptionalText(input.details.catalogNumber),
          label: normalizeOptionalText(input.details.label),
          releaseCountry: normalizeOptionalText(input.details.releaseCountry),
          trackCount: input.details.trackCount ?? null
        };
      case "book":
        return {
          authors: normalizeStringList(input.details.authors),
          isbn10: normalizeOptionalText(input.details.isbn10),
          isbn13: normalizeOptionalText(input.details.isbn13),
          pageCount: input.details.pageCount ?? null,
          publisher: normalizeOptionalText(input.details.publisher)
        };
      case "game":
        return {
          developers: normalizeStringList(input.details.developers),
          genres: normalizeStringList(input.details.genres),
          platforms: normalizeStringList(input.details.platforms),
          publishers: normalizeStringList(input.details.publishers)
        };
    }
  }
}

function mergeDetails(
  existing: MediaRecord,
  input: CreateMediaRecordInput
): UpdateMediaRecordInput["details"] {
  switch (existing.mediaType) {
    case "movie": {
      const details = input.details as MediaRecord["details"] & {
        runtimeMinutes?: number | null;
        directors?: string[];
        cast?: string[];
        genres?: string[];
      };

      return {
        runtimeMinutes: coalesceNumber(
          details.runtimeMinutes,
          existing.details.runtimeMinutes
        ),
        directors: mergeStringLists(details.directors, existing.details.directors),
        cast: mergeStringLists(details.cast, existing.details.cast),
        genres: mergeStringLists(details.genres, existing.details.genres)
      };
    }
    case "tv": {
      const details = input.details as MediaRecord["details"] & {
        seasons?: number | null;
        episodes?: number | null;
        creators?: string[];
        genres?: string[];
      };

      return {
        seasons: coalesceNumber(details.seasons, existing.details.seasons),
        episodes: coalesceNumber(details.episodes, existing.details.episodes),
        genres: mergeStringLists(details.genres, existing.details.genres),
        creators: mergeStringLists(details.creators, existing.details.creators)
      };
    }
    case "album": {
      const details = input.details as MediaRecord["details"] & {
        artists: string[];
        label?: string | null;
        trackCount?: number | null;
        releaseCountry?: string | null;
        catalogNumber?: string | null;
      };

      return {
        artists: mergeStringLists(details.artists, existing.details.artists),
        label: coalesceString(details.label, existing.details.label),
        trackCount: coalesceNumber(details.trackCount, existing.details.trackCount),
        releaseCountry: coalesceString(
          details.releaseCountry,
          existing.details.releaseCountry
        ),
        catalogNumber: coalesceString(
          details.catalogNumber,
          existing.details.catalogNumber
        )
      };
    }
    case "book": {
      const details = input.details as MediaRecord["details"] & {
        authors: string[];
        isbn10?: string | null;
        isbn13?: string | null;
        publisher?: string | null;
        pageCount?: number | null;
      };

      return {
        authors: mergeStringLists(details.authors, existing.details.authors),
        isbn10: coalesceString(details.isbn10, existing.details.isbn10),
        isbn13: coalesceString(details.isbn13, existing.details.isbn13),
        publisher: coalesceString(details.publisher, existing.details.publisher),
        pageCount: coalesceNumber(details.pageCount, existing.details.pageCount)
      };
    }
    case "game": {
      const details = input.details as MediaRecord["details"] & {
        platforms?: string[];
        developers?: string[];
        publishers?: string[];
        genres?: string[];
      };

      return {
        platforms: mergeStringLists(details.platforms, existing.details.platforms),
        developers: mergeStringLists(details.developers, existing.details.developers),
        publishers: mergeStringLists(details.publishers, existing.details.publishers),
        genres: mergeStringLists(details.genres, existing.details.genres)
      };
    }
  }
}

function mergeRatings(
  existing?: MediaRecord["externalRatings"],
  incoming?: CreateMediaRecordInput["externalRatings"]
): CreateMediaRecordInput["externalRatings"] {
  if (!existing && !incoming) {
    return undefined;
  }

  return {
    imdb: coalesceNumber(incoming?.imdb, existing?.imdb),
    rottenTomatoes: coalesceNumber(
      incoming?.rottenTomatoes,
      existing?.rottenTomatoes
    ),
    tmdb: coalesceNumber(incoming?.tmdb, existing?.tmdb),
    metacritic: coalesceNumber(incoming?.metacritic, existing?.metacritic)
  };
}

function mergeStringLists(
  incoming?: string[] | null,
  existing?: string[] | null
): string[] {
  return Array.from(new Set([...(existing ?? []), ...(incoming ?? [])]));
}

function coalesceString(
  incoming?: string | null,
  existing?: string | null
): string | null {
  return incoming && incoming.trim().length > 0 ? incoming : existing ?? null;
}

function coalesceNumber(
  incoming?: number | null,
  existing?: number | null
): number | null {
  return typeof incoming === "number" ? incoming : existing ?? null;
}
