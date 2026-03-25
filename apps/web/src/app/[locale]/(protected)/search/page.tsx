import type { MediaType } from "@media-library/types";
import { SearchFilters } from "../../../../components/search/search-filters";
import { SearchResults } from "../../../../components/search/search-results";
import { mediaTypeOptions } from "../../../../lib/library-options";
import { getSearchResults } from "../../../../lib/search-api";

interface SearchPageProps {
  searchParams?: Promise<Record<string, string | string[] | undefined>>;
}

const defaultMediaType: MediaType = "movie";

export default async function SearchPage({ searchParams }: SearchPageProps) {
  const resolvedSearchParams = (await searchParams) ?? {};
  const query = getSingleParam(resolvedSearchParams.q)?.trim() ?? "";
  const mediaType = toMediaType(getSingleParam(resolvedSearchParams.mediaType));

  const response =
    query.length > 0
      ? await getSearchResults({
          q: query,
          mediaType
        })
      : null;

  return (
    <div className="space-y-8">
      <SearchFilters currentMediaType={mediaType} currentQuery={query} />
      <SearchResults response={response} />
    </div>
  );
}

function getSingleParam(value: string | string[] | undefined): string | undefined {
  return Array.isArray(value) ? value[0] : value;
}

function toMediaType(value: string | undefined): MediaType {
  return mediaTypeOptions.includes(value as MediaType)
    ? (value as MediaType)
    : defaultMediaType;
}
