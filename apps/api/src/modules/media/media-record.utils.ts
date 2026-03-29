import type {
  CreateManualMediaRecordRequest,
  ImportMediaRecordRequest,
  MediaRecord,
  MediaType,
  NormalizedMediaRecordInput,
  ProviderName,
  ProviderRefs
} from "@media-library/types";

export interface ProviderLookupOptions {
  tmdbMediaKind?: "movie" | "tv";
}

export interface ProviderIdentity {
  provider: ProviderName;
  providerId: string;
  mediaType: MediaType;
}

const refreshProviderPriority: Array<keyof ProviderRefs> = [
  "tmdb",
  "musicBrainz",
  "discogs",
  "openLibrary",
  "rawg"
];

export function normalizeText(value: string): string {
  return value.trim().replace(/\s+/g, " ");
}

export function normalizeOptionalText(value?: string | null): string | null {
  if (typeof value !== "string") {
    return null;
  }

  const trimmed = value.trim();
  return trimmed.length > 0 ? trimmed : null;
}

export function normalizeStringList(values?: Array<string | null | undefined>): string[] {
  if (!values) {
    return [];
  }

  return Array.from(
    new Set(
      values
        .map((value) => normalizeOptionalText(value))
        .filter((value): value is string => value !== null)
    )
  );
}

export function getProviderRefKey(provider: ProviderName): keyof ProviderRefs {
  switch (provider) {
    case "tmdb":
      return "tmdb";
    case "musicbrainz":
      return "musicBrainz";
    case "discogs":
      return "discogs";
    case "openlibrary":
      return "openLibrary";
    case "rawg":
      return "rawg";
  }
}

export function getProviderLookupOptions(
  provider: keyof ProviderRefs,
  mediaType: MediaType
): ProviderLookupOptions | undefined {
  if (provider !== "tmdb") {
    return undefined;
  }

  return mediaType === "movie" || mediaType === "tv"
    ? { tmdbMediaKind: mediaType }
    : undefined;
}

export function resolveImportIdentity(
  request: ImportMediaRecordRequest
): ProviderIdentity {
  if (request.mode === "provider_ref") {
    return {
      provider: request.provider,
      providerId: request.providerId,
      mediaType: request.mediaType
    };
  }

  return {
    provider: request.result.provider,
    providerId: request.result.providerId,
    mediaType: request.result.mediaType
  };
}

export function selectRefreshProvider(record: MediaRecord): ProviderIdentity | null {
  for (const providerKey of refreshProviderPriority) {
    const value = record.providerRefs[providerKey];

    if (!value?.id) {
      continue;
    }

    switch (providerKey) {
      case "tmdb": {
        const tmdbValue = value as ProviderRefs["tmdb"] | undefined;

        if (!tmdbValue?.id) {
          continue;
        }

        return {
          provider: "tmdb",
          providerId: tmdbValue.id,
          mediaType: tmdbValue.mediaKind ?? record.mediaType
        };
      }
      case "musicBrainz":
        return {
          provider: "musicbrainz",
          providerId: value.id,
          mediaType: "album"
        };
      case "discogs":
        return {
          provider: "discogs",
          providerId: value.id,
          mediaType: "album"
        };
      case "openLibrary":
        return {
          provider: "openlibrary",
          providerId: value.id,
          mediaType: "book"
        };
      case "rawg":
        return {
          provider: "rawg",
          providerId: value.id,
          mediaType: "game"
        };
    }
  }

  return null;
}

export function getPrimaryCreator(
  record:
    | MediaRecord
    | NormalizedMediaRecordInput
    | CreateManualMediaRecordRequest
): string | null {
  switch (record.mediaType) {
    case "movie":
      return normalizeOptionalText(record.details?.directors?.[0]);
    case "tv":
      return normalizeOptionalText(record.details?.creators?.[0]);
    case "album":
      return normalizeOptionalText(record.details.artists?.[0]);
    case "book":
      return normalizeOptionalText(record.details.authors?.[0]);
    case "game":
      return normalizeOptionalText(record.details?.developers?.[0]);
  }
}

export function supportsBarcodeDedupe(mediaType: MediaType): boolean {
  return mediaType === "album" || mediaType === "book";
}
