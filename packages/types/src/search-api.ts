import type { MediaType } from "./library.js";
import type { NormalizedSearchResult, ProviderName } from "./provider.js";

export interface SearchQuery {
  q: string;
  mediaType: MediaType;
  limit?: number;
}

export interface SearchProviderFailure {
  provider: ProviderName;
  message: string;
}

export interface SearchResponse {
  query: string;
  mediaType: MediaType;
  results: NormalizedSearchResult[];
  failures: SearchProviderFailure[];
}
