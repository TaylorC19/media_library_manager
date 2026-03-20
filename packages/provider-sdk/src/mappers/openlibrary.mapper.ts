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

const OPEN_LIBRARY_COVER_BASE_URL = "https://covers.openlibrary.org/b";

export interface OpenLibrarySearchDoc {
  key?: string;
  title?: string;
  subtitle?: string;
  author_name?: string[];
  first_publish_year?: number;
  isbn?: string[];
  cover_i?: number;
  edition_key?: string[];
}

export interface OpenLibraryBookAuthor {
  name?: string;
}

export interface OpenLibraryBookIdentifierMap {
  isbn_10?: string[];
  isbn_13?: string[];
}

export interface OpenLibraryBookData {
  title?: string;
  subtitle?: string;
  publish_date?: string;
  number_of_pages?: number;
  by_statement?: string;
  authors?: OpenLibraryBookAuthor[];
  publishers?: Array<{ name?: string }>;
  identifiers?: OpenLibraryBookIdentifierMap;
  cover?: {
    medium?: string;
    large?: string;
  };
  notes?: string | { value?: string };
}

function buildOpenLibraryCoverUrl(coverId?: number): string | null {
  if (!coverId) {
    return null;
  }

  return `${OPEN_LIBRARY_COVER_BASE_URL}/id/${coverId}-M.jpg`;
}

function getOpenLibrarySummary(
  notes?: string | { value?: string }
): string | null {
  if (typeof notes === "string") {
    return normalizeOptionalText(notes);
  }

  return normalizeOptionalText(notes?.value);
}

export function mapOpenLibrarySearchResult(
  doc: OpenLibrarySearchDoc
): NormalizedSearchResult | null {
  const providerId = doc.edition_key?.[0];
  const title = normalizeOptionalText(doc.title);

  if (!providerId || !title) {
    return null;
  }

  return {
    provider: "openlibrary",
    providerId,
    mediaType: "book",
    title,
    subtitle: normalizeOptionalText(doc.subtitle),
    year:
      typeof doc.first_publish_year === "number"
        ? doc.first_publish_year
        : null,
    imageUrl: buildOpenLibraryCoverUrl(doc.cover_i),
    summary: null,
    creatorLine: createCreatorLine(doc.author_name),
    barcodeCandidates: normalizeBarcodes(doc.isbn),
    confidence: null
  };
}

export function mapOpenLibraryDetails(
  providerId: string,
  book: OpenLibraryBookData
): NormalizedMediaRecordInput {
  const title = normalizeText(book.title ?? providerId);
  const authors = normalizeStringArray(book.authors?.map((author) => author.name));
  const identifiers = book.identifiers ?? {};

  return {
    mediaType: "book",
    title,
    sortTitle: title,
    releaseDate: normalizeOptionalText(book.publish_date),
    year: extractYear(book.publish_date),
    imageUrl: book.cover?.large ?? book.cover?.medium ?? null,
    summary: getOpenLibrarySummary(book.notes),
    providerRefs: {
      openLibrary: {
        id: providerId
      }
    },
    barcodeCandidates: normalizeBarcodes([
      ...(identifiers.isbn_10 ?? []),
      ...(identifiers.isbn_13 ?? [])
    ]),
    details: {
      authors,
      isbn10: normalizeOptionalText(identifiers.isbn_10?.[0]),
      isbn13: normalizeOptionalText(identifiers.isbn_13?.[0]),
      publisher: normalizeOptionalText(book.publishers?.[0]?.name),
      pageCount: toPositiveInteger(book.number_of_pages)
    }
  };
}
