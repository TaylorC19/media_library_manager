import type {
  LibraryEntryResponse,
  ListLibraryEntriesQuery,
  ListLibraryEntriesResponse
} from "@media-library/types";
import { serverApiFetch } from "./server-api-client";

export async function getLibraryEntries(
  query: ListLibraryEntriesQuery = {}
): Promise<ListLibraryEntriesResponse> {
  const response = await serverApiFetch(`/library${toQueryString(query)}`);

  if (!response.ok) {
    throw new Error(await getErrorMessage(response, "Failed to load library entries."));
  }

  return (await response.json()) as ListLibraryEntriesResponse;
}

export async function getLibraryEntry(
  entryId: string
): Promise<LibraryEntryResponse | null> {
  const response = await serverApiFetch(`/library/${entryId}`);

  if (response.status === 404) {
    return null;
  }

  if (!response.ok) {
    throw new Error(await getErrorMessage(response, "Failed to load library entry."));
  }

  return (await response.json()) as LibraryEntryResponse;
}

function toQueryString(query: ListLibraryEntriesQuery): string {
  const searchParams = new URLSearchParams();

  for (const [key, value] of Object.entries(query)) {
    if (value === undefined || value === null || value === "") {
      continue;
    }

    searchParams.set(key, String(value));
  }

  const serialized = searchParams.toString();
  return serialized.length > 0 ? `?${serialized}` : "";
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
