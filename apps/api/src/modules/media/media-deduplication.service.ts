import { Injectable } from "@nestjs/common";
import type {
  MediaRecord,
  NormalizedMediaRecordInput,
  ProviderName,
  ProviderRefs
} from "@media-library/types";
import { MediaService } from "./media.service";
import {
  getPrimaryCreator,
  getProviderLookupOptions,
  getProviderRefKey,
  normalizeText,
  supportsBarcodeDedupe
} from "./media-record.utils";

@Injectable()
export class MediaDeduplicationService {
  constructor(private readonly mediaService: MediaService) {}

  findByExactProviderRef(params: {
    provider: ProviderName;
    providerId: string;
    mediaType: MediaRecord["mediaType"];
  }): Promise<MediaRecord | null> {
    const providerRefKey = getProviderRefKey(params.provider);

    return this.mediaService.findByProviderRef(
      providerRefKey,
      params.providerId,
      getProviderLookupOptions(providerRefKey, params.mediaType)
    );
  }

  async findByAnyProviderRef(
    providerRefs: ProviderRefs
  ): Promise<MediaRecord | null> {
    const refEntries = Object.entries(providerRefs) as Array<
      [keyof ProviderRefs, ProviderRefs[keyof ProviderRefs]]
    >;

    for (const [providerKey, value] of refEntries) {
      if (!value?.id) {
        continue;
      }

      const record = await this.mediaService.findByProviderRef(
        providerKey,
        value.id,
        providerKey === "tmdb" && "mediaKind" in value && value.mediaKind
          ? { tmdbMediaKind: value.mediaKind }
          : undefined
      );

      if (record) {
        return record;
      }
    }

    return null;
  }

  async findByExactBarcodeOrIsbn(
    normalizedRecord: NormalizedMediaRecordInput
  ): Promise<MediaRecord | null> {
    if (!supportsBarcodeDedupe(normalizedRecord.mediaType)) {
      return null;
    }

    const barcodeCandidates = normalizedRecord.barcodeCandidates ?? [];
    if (barcodeCandidates.length === 0) {
      return null;
    }

    const records = await this.mediaService.findByBarcodeCandidates({
      barcodes: barcodeCandidates,
      mediaType: normalizedRecord.mediaType
    });

    return getSingleUniqueRecord(records);
  }

  async findByHeuristic(
    normalizedRecord: NormalizedMediaRecordInput
  ): Promise<MediaRecord | null> {
    const primaryCreator = getPrimaryCreator(normalizedRecord);
    const year = normalizedRecord.year ?? null;

    if (!primaryCreator || year === null) {
      return null;
    }

    const candidates = await this.mediaService.findByLooseTitleYear({
      mediaType: normalizedRecord.mediaType,
      title: normalizeText(normalizedRecord.title),
      year
    });

    const matchingCandidates = candidates.filter((candidate) => {
      if (normalizeText(candidate.title) !== normalizeText(normalizedRecord.title)) {
        return false;
      }

      if (candidate.year !== year) {
        return false;
      }

      return getPrimaryCreator(candidate) === primaryCreator;
    });

    return getSingleUniqueRecord(matchingCandidates);
  }
}

function getSingleUniqueRecord(records: MediaRecord[]): MediaRecord | null {
  const uniqueRecords = Array.from(
    new Map(records.map((record) => [record.id, record])).values()
  );

  return uniqueRecords.length === 1 ? uniqueRecords[0] ?? null : null;
}
