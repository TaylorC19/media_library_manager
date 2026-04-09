import type { SearchQuery, SearchResponse } from "@media-library/types";
import { getApiErrorMessage } from "./api-error";
import { serverApiFetch } from "./server-api-client";

export async function getSearchResults(
  query: SearchQuery
): Promise<SearchResponse> {
  const response = await serverApiFetch(`/search${toQueryString(query)}`);

  if (!response.ok) {
    throw new Error(await getApiErrorMessage(response, "Failed to search providers."));
  }

  return (await response.json()) as SearchResponse;
}

function toQueryString(query: SearchQuery): string {
  const searchParams = new URLSearchParams();

  searchParams.set("q", query.q);
  searchParams.set("mediaType", query.mediaType);

  if (query.limit !== undefined) {
    searchParams.set("limit", String(query.limit));
  }

  return `?${searchParams.toString()}`;
}
