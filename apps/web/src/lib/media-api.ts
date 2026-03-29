import type {
  ExternalRatings,
  MediaRecord,
  ProviderName
} from "@media-library/types";

export type MediaDetailFieldKey =
  | "authors"
  | "artists"
  | "cast"
  | "catalogNumber"
  | "creators"
  | "developers"
  | "directors"
  | "episodes"
  | "genres"
  | "isbn10"
  | "isbn13"
  | "label"
  | "pageCount"
  | "platforms"
  | "publisher"
  | "publishers"
  | "releaseCountry"
  | "runtimeMinutes"
  | "seasons"
  | "trackCount";

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

export function getMediaDetailFields(
  media: MediaRecord
): Array<{ key: MediaDetailFieldKey; value: string }> {
  switch (media.mediaType) {
    case "movie":
      return compactFields([
        { key: "runtimeMinutes", value: formatNumber(media.details.runtimeMinutes) },
        { key: "directors", value: formatList(media.details.directors) },
        { key: "cast", value: formatList(media.details.cast) },
        { key: "genres", value: formatList(media.details.genres) }
      ]);
    case "tv":
      return compactFields([
        { key: "seasons", value: formatNumber(media.details.seasons) },
        { key: "episodes", value: formatNumber(media.details.episodes) },
        { key: "creators", value: formatList(media.details.creators) },
        { key: "genres", value: formatList(media.details.genres) }
      ]);
    case "album":
      return compactFields([
        { key: "artists", value: formatList(media.details.artists) },
        { key: "label", value: media.details.label ?? null },
        { key: "trackCount", value: formatNumber(media.details.trackCount) },
        { key: "releaseCountry", value: media.details.releaseCountry ?? null },
        { key: "catalogNumber", value: media.details.catalogNumber ?? null }
      ]);
    case "book":
      return compactFields([
        { key: "authors", value: formatList(media.details.authors) },
        { key: "isbn10", value: media.details.isbn10 ?? null },
        { key: "isbn13", value: media.details.isbn13 ?? null },
        { key: "publisher", value: media.details.publisher ?? null },
        { key: "pageCount", value: formatNumber(media.details.pageCount) }
      ]);
    case "game":
      return compactFields([
        { key: "platforms", value: formatList(media.details.platforms) },
        { key: "developers", value: formatList(media.details.developers) },
        { key: "publishers", value: formatList(media.details.publishers) },
        { key: "genres", value: formatList(media.details.genres) }
      ]);
  }
}

export function getProviderRefEntries(
  media: MediaRecord
): Array<{ provider: ProviderName; id: string }> {
  const entries: Array<{ provider: ProviderName; id: string }> = [];

  if (media.providerRefs.tmdb?.id) {
    entries.push({ provider: "tmdb", id: media.providerRefs.tmdb.id });
  }

  if (media.providerRefs.musicBrainz?.id) {
    entries.push({ provider: "musicbrainz", id: media.providerRefs.musicBrainz.id });
  }

  if (media.providerRefs.discogs?.id) {
    entries.push({ provider: "discogs", id: media.providerRefs.discogs.id });
  }

  if (media.providerRefs.openLibrary?.id) {
    entries.push({ provider: "openlibrary", id: media.providerRefs.openLibrary.id });
  }

  if (media.providerRefs.rawg?.id) {
    entries.push({ provider: "rawg", id: media.providerRefs.rawg.id });
  }

  return entries;
}

export function getExternalRatingEntries(
  media: MediaRecord
): Array<{ key: keyof ExternalRatings; value: string }> {
  const ratings = media.externalRatings;

  if (!ratings) {
    return [];
  }

  return compactFields([
    { key: "imdb", value: formatDecimal(ratings.imdb) },
    { key: "rottenTomatoes", value: formatDecimal(ratings.rottenTomatoes) },
    { key: "tmdb", value: formatDecimal(ratings.tmdb) },
    { key: "metacritic", value: formatDecimal(ratings.metacritic) }
  ]);
}

function formatList(values?: string[] | null): string | null {
  return values && values.length > 0 ? values.join(", ") : null;
}

function formatNumber(value?: number | null): string | null {
  return typeof value === "number" ? String(value) : null;
}

function formatDecimal(value?: number | null): string | null {
  return typeof value === "number" ? String(value) : null;
}

function compactFields<T extends string>(
  fields: Array<{ key: T; value: string | null }>
): Array<{ key: T; value: string }> {
  return fields.filter((field): field is { key: T; value: string } => field.value !== null);
}
