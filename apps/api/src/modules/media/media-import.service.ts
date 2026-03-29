import {
  BadGatewayException,
  HttpException,
  Injectable,
  InternalServerErrorException,
  Logger,
  NotFoundException
} from "@nestjs/common";
import type {
  ImportMediaRecordRequest,
  MediaRecord,
  NormalizedMediaRecordInput,
  ProviderName,
  RefreshMediaRecordResponse
} from "@media-library/types";
import { ProvidersService } from "../providers/providers.service";
import { MediaDeduplicationService } from "./media-deduplication.service";
import { MediaNormalizationService } from "./media-normalization.service";
import {
  resolveImportIdentity,
  selectRefreshProvider
} from "./media-record.utils";
import { MediaService } from "./media.service";

export interface ImportMediaResult {
  mediaRecord: MediaRecord;
  wasExistingMediaRecord: boolean;
}

@Injectable()
export class MediaImportService {
  private readonly logger = new Logger(MediaImportService.name);

  constructor(
    private readonly mediaService: MediaService,
    private readonly providersService: ProvidersService,
    private readonly mediaNormalizationService: MediaNormalizationService,
    private readonly mediaDeduplicationService: MediaDeduplicationService
  ) {}

  async importMediaRecord(
    request: ImportMediaRecordRequest
  ): Promise<ImportMediaResult> {
    try {
      const identity = resolveImportIdentity(request);
      const exactProviderMatch =
        await this.mediaDeduplicationService.findByExactProviderRef(identity);

      if (exactProviderMatch) {
        return {
          mediaRecord: exactProviderMatch,
          wasExistingMediaRecord: true
        };
      }

      const normalizedRecord = await this.fetchProviderDetails(identity);
      const matchedRecord = await this.findReusableRecord(normalizedRecord);

      if (!matchedRecord) {
        const mediaRecord = await this.mediaService.createRecord(
          this.mediaNormalizationService.toCreateInputFromProvider(normalizedRecord)
        );

        return {
          mediaRecord,
          wasExistingMediaRecord: false
        };
      }

      const updatedRecord = await this.mediaService.updateRecord(
        matchedRecord.id,
        this.mediaNormalizationService.mergeProviderData(
          matchedRecord,
          normalizedRecord
        )
      );

      if (!updatedRecord) {
        throw new InternalServerErrorException(
          "Failed to update the matched media record."
        );
      }

      return {
        mediaRecord: updatedRecord,
        wasExistingMediaRecord: true
      };
    } catch (error) {
      this.logger.error(
        "Failed to import media record",
        error instanceof Error ? error.stack : undefined
      );

      if (error instanceof HttpException) {
        throw error;
      }

      throw new InternalServerErrorException(
        "Failed to normalize provider media record."
      );
    }
  }

  async refreshMediaRecord(mediaRecordId: string): Promise<RefreshMediaRecordResponse> {
    const mediaRecord = await this.mediaService.getRecord(mediaRecordId);

    if (!mediaRecord) {
      throw new NotFoundException("Media record not found.");
    }

    const refreshProvider = selectRefreshProvider(mediaRecord);
    if (!refreshProvider) {
      return {
        mediaRecord,
        wasRefreshed: false,
        unavailableReason: "provider_ref_unavailable"
      };
    }

    const detailsResponse = await this.providersService.getDetails(refreshProvider);

    if (detailsResponse.failure) {
      throw new BadGatewayException(detailsResponse.failure.message);
    }

    if (!detailsResponse.record) {
      return {
        mediaRecord,
        wasRefreshed: false,
        refreshedFromProvider: refreshProvider.provider,
        unavailableReason: "provider_record_unavailable"
      };
    }

    try {
      const updatedRecord = await this.mediaService.updateRecord(
        mediaRecord.id,
        this.mediaNormalizationService.mergeProviderData(
          mediaRecord,
          detailsResponse.record
        )
      );

      if (!updatedRecord) {
        throw new InternalServerErrorException("Failed to refresh media record.");
      }

      return {
        mediaRecord: updatedRecord,
        wasRefreshed: true,
        refreshedFromProvider: refreshProvider.provider
      };
    } catch (error) {
      this.logger.error(
        `Failed to refresh media record ${mediaRecord.id}`,
        error instanceof Error ? error.stack : undefined
      );

      if (error instanceof HttpException) {
        throw error;
      }

      throw new InternalServerErrorException(
        "Failed to normalize provider media record."
      );
    }
  }

  private async fetchProviderDetails(params: {
    provider: ProviderName;
    providerId: string;
    mediaType: MediaRecord["mediaType"];
  }): Promise<NormalizedMediaRecordInput> {
    const detailsResponse = await this.providersService.getDetails(params);

    if (detailsResponse.failure) {
      throw new BadGatewayException(detailsResponse.failure.message);
    }

    if (!detailsResponse.record) {
      throw new NotFoundException("Provider record not found.");
    }

    return detailsResponse.record;
  }

  private async findReusableRecord(
    normalizedRecord: NormalizedMediaRecordInput
  ): Promise<MediaRecord | null> {
    const providerRefMatch =
      await this.mediaDeduplicationService.findByAnyProviderRef(
        normalizedRecord.providerRefs
      );

    if (providerRefMatch) {
      return providerRefMatch;
    }

    const barcodeMatch =
      await this.mediaDeduplicationService.findByExactBarcodeOrIsbn(
        normalizedRecord
      );

    if (barcodeMatch) {
      return barcodeMatch;
    }

    return this.mediaDeduplicationService.findByHeuristic(normalizedRecord);
  }
}
