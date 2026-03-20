import type { LibraryBucket, MediaType, PhysicalFormat } from "@media-library/types";

export const libraryBucketOptions: LibraryBucket[] = ["catalog", "wishlist"];

export const mediaTypeOptions: MediaType[] = [
  "movie",
  "tv",
  "album",
  "book",
  "game"
];

export const physicalFormatOptions: PhysicalFormat[] = [
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
];
