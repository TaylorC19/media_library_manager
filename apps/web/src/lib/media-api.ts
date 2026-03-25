import type { MediaRecord } from "@media-library/types";

export function getMediaCreatorLine(media: MediaRecord): string | null {
  switch (media.mediaType) {
    case "album":
      return media.details.artists.join(", ") || null;
    case "book":
      return media.details.authors.join(", ") || null;
    case "movie":
      return media.details.directors?.join(", ") || null;
    case "tv":
      return media.details.creators?.join(", ") || null;
    case "game":
      return media.details.developers?.join(", ") || null;
  }
}
