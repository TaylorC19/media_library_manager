import { Injectable } from "@nestjs/common";
import type { MediaRecord, ProviderRefs } from "@media-library/types";
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
}
