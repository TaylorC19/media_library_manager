import { Injectable } from "@nestjs/common";
import type {
  CreateManualMediaRecordRequest,
  MediaRecord,
  ProviderRefs
} from "@media-library/types";
import { MediaNormalizationService } from "./media-normalization.service";
import {
  MediaRecordRepository,
  type CreateMediaRecordInput,
  type UpdateMediaRecordInput
} from "./repositories/media-record.repository";
import { normalizeText } from "./media-record.utils";

@Injectable()
export class MediaService {
  constructor(
    private readonly mediaRecordRepository: MediaRecordRepository,
    private readonly mediaNormalizationService: MediaNormalizationService
  ) {}

  createRecord(input: CreateMediaRecordInput): Promise<MediaRecord> {
    return this.mediaRecordRepository.create(input);
  }

  updateRecord(id: string, input: UpdateMediaRecordInput): Promise<MediaRecord | null> {
    return this.mediaRecordRepository.update(id, input);
  }

  getRecord(id: string): Promise<MediaRecord | null> {
    return this.mediaRecordRepository.findById(id);
  }

  getRecords(ids: string[]): Promise<MediaRecord[]> {
    return this.mediaRecordRepository.findManyByIds(ids);
  }

  findByBarcodeCandidate(
    barcode: string,
    mediaType?: MediaRecord["mediaType"]
  ): Promise<MediaRecord[]> {
    return this.mediaRecordRepository.findByBarcodeCandidate(barcode, mediaType);
  }

  findByBarcodeCandidates(params: {
    barcodes: string[];
    mediaType: MediaRecord["mediaType"];
  }): Promise<MediaRecord[]> {
    return this.mediaRecordRepository.findByAnyBarcodeCandidates(params);
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

    return this.createRecord(
      this.mediaNormalizationService.toCreateInputFromManual(input)
    );
  }
}
