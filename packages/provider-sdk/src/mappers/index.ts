import type {
  NormalizedMediaRecordInput,
  NormalizedSearchResult
} from "@media-library/types";
import { normalizeBarcodes } from "../utils/barcode.js";
import {
  normalizeOptionalText,
  normalizeStringArray,
  normalizeText
} from "../utils/normalization.js";

export * from "./discogs.mapper.js";
export * from "./musicbrainz.mapper.js";
export * from "./openlibrary.mapper.js";
export * from "./rawg.mapper.js";
export * from "./tmdb.mapper.js";

export function normalizeSearchResults(
  results: NormalizedSearchResult[]
): NormalizedSearchResult[] {
  return results.map((result) => ({
    ...result,
    title: normalizeText(result.title),
    subtitle: normalizeOptionalText(result.subtitle),
    imageUrl: normalizeOptionalText(result.imageUrl),
    summary: normalizeOptionalText(result.summary),
    creatorLine: normalizeOptionalText(result.creatorLine),
    barcodeCandidates: normalizeBarcodes(result.barcodeCandidates)
  }));
}

export function normalizeMediaRecord(
  record: NormalizedMediaRecordInput
): NormalizedMediaRecordInput {
  switch (record.mediaType) {
    case "movie":
      return {
        ...record,
        title: normalizeText(record.title),
        sortTitle: normalizeOptionalText(record.sortTitle),
        releaseDate: normalizeOptionalText(record.releaseDate),
        imageUrl: normalizeOptionalText(record.imageUrl),
        summary: normalizeOptionalText(record.summary),
        barcodeCandidates: normalizeBarcodes(record.barcodeCandidates),
        details: {
          runtimeMinutes: record.details.runtimeMinutes ?? null,
          directors: normalizeStringArray(record.details.directors),
          cast: normalizeStringArray(record.details.cast),
          genres: normalizeStringArray(record.details.genres)
        }
      };
    case "tv":
      return {
        ...record,
        title: normalizeText(record.title),
        sortTitle: normalizeOptionalText(record.sortTitle),
        releaseDate: normalizeOptionalText(record.releaseDate),
        imageUrl: normalizeOptionalText(record.imageUrl),
        summary: normalizeOptionalText(record.summary),
        barcodeCandidates: normalizeBarcodes(record.barcodeCandidates),
        details: {
          seasons: record.details.seasons ?? null,
          episodes: record.details.episodes ?? null,
          genres: normalizeStringArray(record.details.genres),
          creators: normalizeStringArray(record.details.creators)
        }
      };
    case "album":
      return {
        ...record,
        title: normalizeText(record.title),
        sortTitle: normalizeOptionalText(record.sortTitle),
        releaseDate: normalizeOptionalText(record.releaseDate),
        imageUrl: normalizeOptionalText(record.imageUrl),
        summary: normalizeOptionalText(record.summary),
        barcodeCandidates: normalizeBarcodes(record.barcodeCandidates),
        details: {
          artists: normalizeStringArray(record.details.artists),
          label: normalizeOptionalText(record.details.label),
          trackCount: record.details.trackCount ?? null,
          releaseCountry: normalizeOptionalText(record.details.releaseCountry),
          catalogNumber: normalizeOptionalText(record.details.catalogNumber)
        }
      };
    case "book":
      return {
        ...record,
        title: normalizeText(record.title),
        sortTitle: normalizeOptionalText(record.sortTitle),
        releaseDate: normalizeOptionalText(record.releaseDate),
        imageUrl: normalizeOptionalText(record.imageUrl),
        summary: normalizeOptionalText(record.summary),
        barcodeCandidates: normalizeBarcodes(record.barcodeCandidates),
        details: {
          authors: normalizeStringArray(record.details.authors),
          isbn10: normalizeOptionalText(record.details.isbn10),
          isbn13: normalizeOptionalText(record.details.isbn13),
          publisher: normalizeOptionalText(record.details.publisher),
          pageCount: record.details.pageCount ?? null
        }
      };
    case "game":
      return {
        ...record,
        title: normalizeText(record.title),
        sortTitle: normalizeOptionalText(record.sortTitle),
        releaseDate: normalizeOptionalText(record.releaseDate),
        imageUrl: normalizeOptionalText(record.imageUrl),
        summary: normalizeOptionalText(record.summary),
        barcodeCandidates: normalizeBarcodes(record.barcodeCandidates),
        details: {
          platforms: normalizeStringArray(record.details.platforms),
          developers: normalizeStringArray(record.details.developers),
          publishers: normalizeStringArray(record.details.publishers),
          genres: normalizeStringArray(record.details.genres)
        }
      };
  }
}
