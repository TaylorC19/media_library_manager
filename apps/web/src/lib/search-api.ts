import type { SearchQuery, SearchResponse } from "@media-library/types";
import { serverApiFetch } from "./server-api-client";

export async function getSearchResults(
  query: SearchQuery
): Promise<SearchResponse> {
  const response = await serverApiFetch(`/search${toQueryString(query)}`);

  if (!response.ok) {
    throw new Error(await getErrorMessage(response, "Failed to search providers."));
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

async function getErrorMessage(
  response: Response,
  fallback: string
): Promise<string> {
  const payload = (await response.json().catch(() => null)) as
    | { message?: string | string[] }
    | null;

  if (typeof payload?.message === "string") {
    return payload.message;
  }

  if (Array.isArray(payload?.message)) {
    return payload.message.join(", ");
  }

  return fallback;
}
