import { forwardRef, Module } from "@nestjs/common";
import { MongooseModule } from "@nestjs/mongoose";
import { MediaModule } from "../media/media.module";
import { LibraryController } from "./library.controller";
import { LibraryEntryRepository } from "./repositories/library-entry.repository";
import {
  LibraryEntryDocumentModel,
  LibraryEntrySchema
} from "./schemas/library-entry.schema";
import { LibraryService } from "./library.service";

@Module({
  imports: [
    forwardRef(() => MediaModule),
    MongooseModule.forFeature([
      {
        name: LibraryEntryDocumentModel.name,
        schema: LibraryEntrySchema
      }
    ])
  ],
  controllers: [LibraryController],
  providers: [LibraryEntryRepository, LibraryService],
  exports: [LibraryEntryRepository, LibraryService]
})
export class LibraryModule {}
