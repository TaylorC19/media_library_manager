export const libraryBuckets = ["catalog", "wishlist"] as const;
export type LibraryBucket = (typeof libraryBuckets)[number];

export const mediaTypes = ["movie", "tv", "album", "book", "game"] as const;
export type MediaType = (typeof mediaTypes)[number];

export const physicalFormats = [
  "blu_ray",
  "dvd",
  "vhs",
  "cd",
  "vinyl",
  "cassette",
  "hardcover",
  "paperback",
  "switch",
  "ps5",
  "xbox",
  "digital",
  "other"
] as const;

export type PhysicalFormat = (typeof physicalFormats)[number];

export interface LibraryEntry {
  id: string;
  userId: string;
  mediaRecordId: string;
  bucket: LibraryBucket;
  mediaType: MediaType;
  format?: PhysicalFormat | null;
  barcode?: string | null;
  purchaseDate?: string | null;
  notes?: string | null;
  tags: string[];
  createdAt: string;
  updatedAt: string;
}
