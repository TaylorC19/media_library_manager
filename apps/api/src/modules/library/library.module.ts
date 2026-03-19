import { Module } from "@nestjs/common";
import { MongooseModule } from "@nestjs/mongoose";
import { LibraryEntryRepository } from "./repositories/library-entry.repository";
import {
  LibraryEntryDocumentModel,
  LibraryEntrySchema
} from "./schemas/library-entry.schema";
import { LibraryService } from "./library.service";

@Module({
  imports: [
    MongooseModule.forFeature([
      {
        name: LibraryEntryDocumentModel.name,
        schema: LibraryEntrySchema
      }
    ])
  ],
  providers: [LibraryEntryRepository, LibraryService],
  exports: [LibraryEntryRepository, LibraryService]
})
export class LibraryModule {}
