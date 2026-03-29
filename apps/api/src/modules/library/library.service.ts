import {
  ConflictException,
  Injectable,
  NotFoundException
} from "@nestjs/common";
import type {
  CreateLibraryEntryRequest,
  DeleteLibraryEntryResponse,
  LibraryEntry,
  LibraryEntryListItem,
  LibraryEntryResponse,
  ListLibraryEntriesQuery,
  ListLibraryEntriesResponse,
  MediaRecord,
  PaginationMeta,
  UpdateLibraryEntryRequest
} from "@media-library/types";
import { MediaService } from "../media/media.service";
import {
  LibraryEntryRepository,
  type CreateLibraryEntryInput,
  type LibraryEntryFilters,
  type UpdateLibraryEntryInput
} from "./repositories/library-entry.repository";

@Injectable()
export class LibraryService {
  constructor(
    private readonly libraryEntryRepository: LibraryEntryRepository,
    private readonly mediaService: MediaService
  ) {}

  async createEntryForUser(
    userId: string,
    input: CreateLibraryEntryRequest
  ): Promise<LibraryEntryResponse> {
    const result = await this.createOrReuseEntryForUser(userId, input);

    if (result.wasExistingLibraryEntry) {
      throw new ConflictException(
        "That item is already saved in this bucket with the same format."
      );
    }

    return result.libraryEntry;
  }

  async createOrReuseEntryForUser(
    userId: string,
    input: CreateLibraryEntryRequest
  ): Promise<{
    libraryEntry: LibraryEntryResponse;
    wasExistingLibraryEntry: boolean;
  }> {
    const mediaRecord = await this.mediaService.getRecord(input.mediaRecordId);

    if (!mediaRecord) {
      throw new NotFoundException("Media record not found.");
    }

    const duplicateEntry = await this.findDuplicateEntry({
      bucket: input.bucket,
      format: input.format,
      mediaRecordId: input.mediaRecordId,
      userId
    });

    if (duplicateEntry) {
      return {
        libraryEntry: {
          entry: duplicateEntry,
          media: mediaRecord
        },
        wasExistingLibraryEntry: true
      };
    }

    const entry = await this.libraryEntryRepository.create({
      ...normalizeCreateEntryInput(input),
      mediaType: mediaRecord.mediaType,
      userId
    });

    return {
      libraryEntry: {
        entry,
        media: mediaRecord
      },
      wasExistingLibraryEntry: false
    };
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

  async getEntryDetailForUser(
    id: string,
    userId: string
  ): Promise<LibraryEntryResponse> {
    const entry = await this.getEntryForUser(id, userId);

    if (!entry) {
      throw new NotFoundException("Library entry not found.");
    }

    const mediaRecord = await this.mediaService.getRecord(entry.mediaRecordId);

    if (!mediaRecord) {
      throw new NotFoundException("Media record not found.");
    }

    return {
      entry,
      media: mediaRecord
    };
  }

  async listEntriesForUser(
    userId: string,
    filters: ListLibraryEntriesQuery = {}
  ): Promise<ListLibraryEntriesResponse> {
    const entries = await this.libraryEntryRepository.listByUser(
      userId,
      toRepositoryFilters(filters)
    );

    const itemResults = await this.buildListItems(entries);
    const filteredItems = filterListItems(itemResults, filters.search);
    const pagination = getPaginationMeta(
      filteredItems.length,
      filters.page,
      filters.pageSize
    );
    const startIndex = (pagination.page - 1) * pagination.pageSize;
    const pagedItems = filteredItems.slice(
      startIndex,
      startIndex + pagination.pageSize
    );

    return {
      items: pagedItems,
      pagination
    };
  }

  async updateEntryForUser(
    id: string,
    userId: string,
    input: UpdateLibraryEntryRequest
  ): Promise<LibraryEntryResponse> {
    const existingEntry = await this.getEntryForUser(id, userId);

    if (!existingEntry) {
      throw new NotFoundException("Library entry not found.");
    }

    const nextBucket = input.bucket ?? existingEntry.bucket;
    const nextFormat =
      input.format === undefined ? existingEntry.format : input.format;
    const duplicateEntry = await this.findDuplicateEntry({
      bucket: nextBucket,
      format: nextFormat,
      mediaRecordId: existingEntry.mediaRecordId,
      userId
    });

    if (duplicateEntry && duplicateEntry.id !== existingEntry.id) {
      throw new ConflictException(
        "That item is already saved in this bucket with the same format."
      );
    }

    const updatedEntry = await this.libraryEntryRepository.updateForUser(
      id,
      userId,
      normalizeUpdateEntryInput(input)
    );

    if (!updatedEntry) {
      throw new NotFoundException("Library entry not found.");
    }

    const mediaRecord = await this.mediaService.getRecord(updatedEntry.mediaRecordId);

    if (!mediaRecord) {
      throw new NotFoundException("Media record not found.");
    }

    return {
      entry: updatedEntry,
      media: mediaRecord
    };
  }

  async deleteEntryForUser(
    id: string,
    userId: string
  ): Promise<DeleteLibraryEntryResponse> {
    const deleted = await this.libraryEntryRepository.deleteForUser(id, userId);

    if (!deleted) {
      throw new NotFoundException("Library entry not found.");
    }

    return { success: true };
  }

  private async buildListItems(
    entries: LibraryEntry[]
  ): Promise<LibraryEntryListItem[]> {
    const mediaRecords = await this.mediaService.getRecords(
      Array.from(new Set(entries.map((entry) => entry.mediaRecordId)))
    );
    const mediaById = new Map(mediaRecords.map((record) => [record.id, record]));

    return entries.flatMap((entry) => {
      const mediaRecord = mediaById.get(entry.mediaRecordId);

      return mediaRecord
        ? [
            {
              entry,
              media: mediaRecord
            }
          ]
        : [];
    });
  }
}

function toRepositoryFilters(
  filters: ListLibraryEntriesQuery
): LibraryEntryFilters {
  return {
    bucket: filters.bucket,
    mediaType: filters.mediaType,
    tag: normalizeOptionalText(filters.tag) ?? undefined
  };
}

function normalizeCreateEntryInput(
  input: CreateLibraryEntryRequest
): Omit<CreateLibraryEntryInput, "userId" | "mediaType"> {
  return {
    ...input,
    barcode: normalizeOptionalText(input.barcode),
    notes: normalizeOptionalText(input.notes),
    purchaseDate: normalizeOptionalText(input.purchaseDate),
    tags: normalizeStringList(input.tags)
  };
}

function normalizeUpdateEntryInput(
  input: UpdateLibraryEntryRequest
): UpdateLibraryEntryInput {
  return {
    barcode: normalizeOptionalText(input.barcode),
    bucket: input.bucket,
    format: input.format,
    notes: normalizeOptionalText(input.notes),
    purchaseDate: normalizeOptionalText(input.purchaseDate),
    tags: input.tags ? normalizeStringList(input.tags) : undefined
  };
}

function normalizeOptionalText(value?: string | null): string | null {
  if (typeof value !== "string") {
    return null;
  }

  const trimmed = value.trim();
  return trimmed.length > 0 ? trimmed : null;
}

function normalizeStringList(values?: string[]): string[] {
  if (!values) {
    return [];
  }

  return Array.from(
    new Set(
      values
        .map((value) => value.trim())
        .filter((value) => value.length > 0)
    )
  );
}

function filterListItems(
  items: LibraryEntryListItem[],
  search?: string
): LibraryEntryListItem[] {
  const normalizedSearch = search?.trim().toLowerCase();

  if (!normalizedSearch) {
    return items;
  }

  return items.filter(({ entry, media }) => {
    const searchableParts = [
      media.title,
      media.summary ?? "",
      entry.notes ?? "",
      entry.barcode ?? "",
      entry.tags.join(" "),
      getMediaCreatorLine(media)
    ];

    return searchableParts.some((part) =>
      part.toLowerCase().includes(normalizedSearch)
    );
  });
}

function getMediaCreatorLine(media: MediaRecord): string {
  switch (media.mediaType) {
    case "album":
      return media.details.artists.join(" ");
    case "book":
      return media.details.authors.join(" ");
    case "movie":
      return media.details.directors?.join(" ") ?? "";
    case "tv":
      return media.details.creators?.join(" ") ?? "";
    case "game":
      return media.details.developers?.join(" ") ?? "";
  }
}

function getPaginationMeta(
  total: number,
  page?: number,
  pageSize?: number
): PaginationMeta {
  const safePageSize = Math.max(1, pageSize ?? 12);
  const totalPages = total === 0 ? 1 : Math.ceil(total / safePageSize);
  const safePage = Math.min(Math.max(1, page ?? 1), totalPages);

  return {
    hasNextPage: safePage < totalPages,
    page: safePage,
    pageSize: safePageSize,
    total,
    totalPages
  };
}
