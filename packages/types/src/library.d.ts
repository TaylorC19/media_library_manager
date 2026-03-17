export declare const libraryBuckets: readonly ["catalog", "wishlist"];
export type LibraryBucket = (typeof libraryBuckets)[number];
export declare const mediaTypes: readonly ["movie", "tv", "album", "book", "game"];
export type MediaType = (typeof mediaTypes)[number];
export declare const physicalFormats: readonly ["blu_ray", "dvd", "vhs", "cd", "vinyl", "cassette", "hardcover", "paperback", "switch", "ps5", "xbox", "digital", "other"];
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
