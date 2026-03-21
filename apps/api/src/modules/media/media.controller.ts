import { Body, Controller, Post } from "@nestjs/common";
import type {
  AuthUser,
  CreateLibraryEntryRequest,
  ImportMediaRecordResponse,
  CreateManualMediaRecordRequest,
  ManualMediaRecordResponse
} from "@media-library/types";
import { CurrentUser } from "../auth/decorators/current-user.decorator";
import { LibraryService } from "../library/library.service";
import { CreateManualMediaRecordDto } from "./dto/create-manual-media-record.dto";
import { ImportMediaDto } from "./dto/import-media.dto";
import { MediaService } from "./media.service";

@Controller("media")
export class MediaController {
  constructor(
    private readonly mediaService: MediaService,
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
    @Body() body: ImportMediaDto
  ): Promise<ImportMediaRecordResponse> {
    const mediaRecord = await this.mediaService.importFromProvider(body);

    if (!body.bucket) {
      return { mediaRecord };
    }

    const libraryEntry = await this.libraryService.createEntryForUser(
      user.id,
      buildCreateLibraryEntryPayload(body, mediaRecord.id)
    );

    return {
      mediaRecord,
      libraryEntry
    };
  }
}

function buildCreateLibraryEntryPayload(
  body: ImportMediaDto,
  mediaRecordId: string
): CreateLibraryEntryRequest {
  return {
    mediaRecordId,
    bucket: body.bucket!,
    format: body.format ?? null,
    barcode: body.barcode ?? null,
    purchaseDate: body.purchaseDate ?? null,
    notes: body.notes ?? null,
    tags: body.tags ?? []
  };
}
