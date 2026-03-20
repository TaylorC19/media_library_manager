import { Injectable } from "@nestjs/common";
import type {
  CreateManualMediaRecordRequest,
  MediaRecord,
  ProviderRefs
} from "@media-library/types";
import {
  MediaRecordRepository,
  type CreateMediaRecordInput
} from "./repositories/media-record.repository";

@Injectable()
export class MediaService {
  constructor(
    private readonly mediaRecordRepository: MediaRecordRepository
  ) {}

  createRecord(input: CreateMediaRecordInput): Promise<MediaRecord> {
    return this.mediaRecordRepository.create(input);
  }

  getRecord(id: string): Promise<MediaRecord | null> {
    return this.mediaRecordRepository.findById(id);
  }

  getRecords(ids: string[]): Promise<MediaRecord[]> {
    return this.mediaRecordRepository.findManyByIds(ids);
  }

  findByBarcodeCandidate(barcode: string): Promise<MediaRecord[]> {
    return this.mediaRecordRepository.findByBarcodeCandidate(barcode);
  }

  findByProviderRef(
    provider: keyof ProviderRefs,
    providerId: string,
    options?: {
      tmdbMediaKind?: "movie" | "tv";
    }
  ): Promise<MediaRecord | null> {
    return this.mediaRecordRepository.findByProviderRef(
      provider,
      providerId,
      options
    );
  }

  findByTitleYear(params: {
    mediaType: MediaRecord["mediaType"];
    title: string;
    year?: number | null;
  }): Promise<MediaRecord[]> {
    return this.mediaRecordRepository.findByTitleYear(params);
  }

  findByLooseTitleYear(params: {
    mediaType: MediaRecord["mediaType"];
    title: string;
    year?: number | null;
  }): Promise<MediaRecord[]> {
    return this.mediaRecordRepository.findByLooseTitleYear(params);
  }

  async createManualRecord(
    input: CreateManualMediaRecordRequest
  ): Promise<MediaRecord> {
    const normalizedTitle = normalizeText(input.title);
    const existingRecords = await this.findByLooseTitleYear({
      mediaType: input.mediaType,
      title: normalizedTitle,
      year: input.year ?? undefined
    });

    const matchedRecord = existingRecords.find(
      (record) => normalizeText(record.title) === normalizedTitle
    );

    if (matchedRecord) {
      return matchedRecord;
    }

    return this.createRecord({
      mediaType: input.mediaType,
      title: normalizedTitle,
      sortTitle: normalizedTitle,
      releaseDate: normalizeOptionalText(input.releaseDate),
      year: input.year ?? null,
      imageUrl: normalizeOptionalText(input.imageUrl),
      summary: normalizeOptionalText(input.summary),
      providerRefs: {},
      barcodeCandidates: normalizeStringList(input.barcodeCandidates),
      details: normalizeDetails(input),
      lastSyncedAt: null
    });
  }
}

function normalizeText(value: string): string {
  return value.trim().replace(/\s+/g, " ");
}

function normalizeOptionalText(value?: string | null): string | null {
  if (typeof value !== "string") {
    return null;
  }

  const trimmed = value.trim();
  return trimmed.length > 0 ? trimmed : null;
}

function normalizeStringList(values?: string[]): string[] {
  if (!values) {
    return [];
  }

  return Array.from(
    new Set(
      values
        .map((value) => value.trim())
        .filter((value) => value.length > 0)
    )
  );
}

function normalizeDetails(
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
