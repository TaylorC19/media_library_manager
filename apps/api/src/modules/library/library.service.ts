import { Injectable } from "@nestjs/common";
import type { LibraryEntry } from "@media-library/types";
import {
  LibraryEntryRepository,
  type CreateLibraryEntryInput,
  type LibraryEntryFilters
} from "./repositories/library-entry.repository";

@Injectable()
export class LibraryService {
  constructor(
    private readonly libraryEntryRepository: LibraryEntryRepository
  ) {}

  createEntry(input: CreateLibraryEntryInput): Promise<LibraryEntry> {
    return this.libraryEntryRepository.create(input);
  }

  findDuplicateEntry(input: {
    userId: string;
    mediaRecordId: string;
    bucket: CreateLibraryEntryInput["bucket"];
    format?: CreateLibraryEntryInput["format"];
  }): Promise<LibraryEntry | null> {
    return this.libraryEntryRepository.findDuplicate(input);
  }

  getEntryForUser(id: string, userId: string): Promise<LibraryEntry | null> {
    return this.libraryEntryRepository.findByIdForUser(id, userId);
  }

  listEntriesForUser(
    userId: string,
    filters?: LibraryEntryFilters
  ): Promise<LibraryEntry[]> {
    return this.libraryEntryRepository.listByUser(userId, filters);
  }
}
