import { forwardRef, Module } from "@nestjs/common";
import { MongooseModule } from "@nestjs/mongoose";
import { LibraryModule } from "../library/library.module";
import { ProvidersModule } from "../providers/providers.module";
import { ImportMediaDtoPipe } from "./dto/import-media.dto";
import { MediaController } from "./media.controller";
import { MediaDeduplicationService } from "./media-deduplication.service";
import { MediaImportService } from "./media-import.service";
import { MediaNormalizationService } from "./media-normalization.service";
import { MediaRecordRepository } from "./repositories/media-record.repository";
import { ScanLogRepository } from "./repositories/scan-log.repository";
import { MediaService } from "./media.service";
import {
  MediaRecordDocumentModel,
  MediaRecordSchema
} from "./schemas/media-record.schema";
import {
  ProviderCacheDocumentModel,
  ProviderCacheSchema
} from "./schemas/provider-cache.schema";
import {
  ScanLogDocumentModel,
  ScanLogSchema
} from "./schemas/scan-log.schema";

@Module({
  imports: [
    forwardRef(() => LibraryModule),
    ProvidersModule,
    MongooseModule.forFeature([
      {
        name: MediaRecordDocumentModel.name,
        schema: MediaRecordSchema
      },
      {
        name: ProviderCacheDocumentModel.name,
        schema: ProviderCacheSchema
      },
      {
        name: ScanLogDocumentModel.name,
        schema: ScanLogSchema
      }
    ])
  ],
  controllers: [MediaController],
  providers: [
    ImportMediaDtoPipe,
    MediaRecordRepository,
    ScanLogRepository,
    MediaNormalizationService,
    MediaService,
    MediaDeduplicationService,
    MediaImportService
  ],
  exports: [MediaRecordRepository, MediaService, ScanLogRepository]
})
export class MediaModule {}
