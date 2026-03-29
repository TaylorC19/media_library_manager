import {
  Body,
  Controller,
  Get,
  NotFoundException,
  Param,
  Post
} from "@nestjs/common";
import type {
  AuthUser,
  CreateLibraryEntryRequest,
  GetMediaRecordResponse,
  ImportMediaRecordResponse,
  CreateManualMediaRecordRequest,
  ImportMediaRecordRequest,
  MediaRecord,
  ManualMediaRecordResponse,
  RefreshMediaRecordResponse
} from "@media-library/types";
import { CurrentUser } from "../auth/decorators/current-user.decorator";
import { LibraryService } from "../library/library.service";
import { CreateManualMediaRecordDto } from "./dto/create-manual-media-record.dto";
import { ImportMediaDtoPipe } from "./dto/import-media.dto";
import { MediaImportService } from "./media-import.service";
import { MediaService } from "./media.service";

@Controller("media")
export class MediaController {
  constructor(
    private readonly mediaService: MediaService,
    private readonly mediaImportService: MediaImportService,
    private readonly libraryService: LibraryService
  ) {}

  @Post("manual")
  async createManualMediaRecord(
    @Body() body: CreateManualMediaRecordDto
  ): Promise<ManualMediaRecordResponse> {
    const mediaRecord = await this.mediaService.createManualRecord(
      body as unknown as CreateManualMediaRecordRequest
    );

    return { mediaRecord };
  }

  @Post("import")
  async importMediaRecord(
    @CurrentUser() user: AuthUser,
    @Body(ImportMediaDtoPipe) body: ImportMediaRecordRequest
  ): Promise<ImportMediaRecordResponse> {
    const importResult = await this.mediaImportService.importMediaRecord(body);

    if (!body.entry) {
      return {
        mediaRecord: importResult.mediaRecord,
        wasExistingMediaRecord: importResult.wasExistingMediaRecord
      };
    }

    const libraryResult = await this.libraryService.createOrReuseEntryForUser(
      user.id,
      buildCreateLibraryEntryPayload(body.entry, importResult.mediaRecord)
    );

    return {
      mediaRecord: importResult.mediaRecord,
      libraryEntry: libraryResult.libraryEntry,
      wasExistingMediaRecord: importResult.wasExistingMediaRecord,
      wasExistingLibraryEntry: libraryResult.wasExistingLibraryEntry
    };
  }

  @Get(":mediaRecordId")
  async getMediaRecord(
    @Param("mediaRecordId") mediaRecordId: string
  ): Promise<GetMediaRecordResponse> {
    const mediaRecord = await this.mediaService.getRecord(mediaRecordId);

    if (!mediaRecord) {
      throw new NotFoundException("Media record not found.");
    }

    return { mediaRecord };
  }

  @Post("refresh/:mediaRecordId")
  refreshMediaRecord(
    @Param("mediaRecordId") mediaRecordId: string
  ): Promise<RefreshMediaRecordResponse> {
    return this.mediaImportService.refreshMediaRecord(mediaRecordId);
  }
}

function buildCreateLibraryEntryPayload(
  entry: NonNullable<ImportMediaRecordRequest["entry"]>,
  mediaRecord: MediaRecord
): CreateLibraryEntryRequest {
  return {
    mediaRecordId: mediaRecord.id,
    bucket: entry.bucket,
    format: entry.format ?? null,
    barcode:
      entry.barcode ??
      (mediaRecord.barcodeCandidates?.length === 1
        ? mediaRecord.barcodeCandidates[0] ?? null
        : null),
    purchaseDate: entry.purchaseDate ?? null,
    notes: entry.notes ?? null,
    tags: entry.tags ?? []
  };
}
