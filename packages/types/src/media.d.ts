import type { MediaType } from "./library";
export interface ProviderRefs {
    tmdb?: {
        id: string;
        mediaKind?: "movie" | "tv";
    };
    musicBrainz?: {
        id: string;
    };
    discogs?: {
        id: string;
    };
    openLibrary?: {
        id: string;
    };
    rawg?: {
        id: string;
    };
}
export interface ExternalRatings {
    imdb?: number | null;
    rottenTomatoes?: number | null;
    tmdb?: number | null;
    metacritic?: number | null;
}
export interface MediaRecordBase {
    id: string;
    mediaType: MediaType;
    title: string;
    sortTitle?: string | null;
    releaseDate?: string | null;
    year?: number | null;
    imageUrl?: string | null;
    summary?: string | null;
    providerRefs: ProviderRefs;
    externalRatings?: ExternalRatings;
    barcodeCandidates?: string[];
    createdAt: string;
    updatedAt: string;
    lastSyncedAt?: string | null;
}
export interface MovieMediaRecord extends MediaRecordBase {
    mediaType: "movie";
    details: {
        runtimeMinutes?: number | null;
        directors?: string[];
        cast?: string[];
        genres?: string[];
    };
}
export interface TvMediaRecord extends MediaRecordBase {
    mediaType: "tv";
    details: {
        seasons?: number | null;
        episodes?: number | null;
        genres?: string[];
        creators?: string[];
    };
}
export interface AlbumMediaRecord extends MediaRecordBase {
    mediaType: "album";
    details: {
        artists: string[];
        label?: string | null;
        trackCount?: number | null;
        releaseCountry?: string | null;
        catalogNumber?: string | null;
    };
}
export interface BookMediaRecord extends MediaRecordBase {
    mediaType: "book";
    details: {
        authors: string[];
        isbn10?: string | null;
        isbn13?: string | null;
        publisher?: string | null;
        pageCount?: number | null;
    };
}
export interface GameMediaRecord extends MediaRecordBase {
    mediaType: "game";
    details: {
        platforms?: string[];
        developers?: string[];
        publishers?: string[];
        genres?: string[];
    };
}
export type MediaRecord = MovieMediaRecord | TvMediaRecord | AlbumMediaRecord | BookMediaRecord | GameMediaRecord;
