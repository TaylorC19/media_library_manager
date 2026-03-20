import type { LibraryBucket, LibraryEntry, MediaType, PhysicalFormat } from "./library.js";
import type { MediaRecord } from "./media.js";

export interface PaginationMeta {
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
  hasNextPage: boolean;
}

export interface LibraryEntryListItem {
  entry: LibraryEntry;
  media: MediaRecord;
}

export interface ListLibraryEntriesQuery {
  bucket?: LibraryBucket;
  mediaType?: MediaType;
  tag?: string;
  search?: string;
  page?: number;
  pageSize?: number;
}

export interface CreateLibraryEntryRequest {
  mediaRecordId: string;
  bucket: LibraryBucket;
  format?: PhysicalFormat | null;
  barcode?: string | null;
  purchaseDate?: string | null;
  notes?: string | null;
  tags?: string[];
}

export interface UpdateLibraryEntryRequest {
  bucket?: LibraryBucket;
  format?: PhysicalFormat | null;
  barcode?: string | null;
  purchaseDate?: string | null;
  notes?: string | null;
  tags?: string[];
}

export interface LibraryEntryResponse {
  entry: LibraryEntry;
  media: MediaRecord;
}

export interface ListLibraryEntriesResponse {
  items: LibraryEntryListItem[];
  pagination: PaginationMeta;
}

export interface DeleteLibraryEntryResponse {
  success: true;
}
