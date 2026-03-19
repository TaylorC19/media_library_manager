import { Module } from "@nestjs/common";
import { MongooseModule } from "@nestjs/mongoose";
import { MediaRecordRepository } from "./repositories/media-record.repository";
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
  providers: [MediaRecordRepository, MediaService],
  exports: [MediaRecordRepository, MediaService]
})
export class MediaModule {}
