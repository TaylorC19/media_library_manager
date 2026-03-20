import type { LibraryBucket, MediaType } from "@media-library/types";
import { CreateLibraryEntryForm } from "./create-library-entry-form";
import { LibraryFilters } from "./library-filters";
import { LibraryList } from "./library-list";
import { PaginationControls } from "./pagination-controls";
import { getLibraryEntries } from "../../lib/library-api";
import { mediaTypeOptions } from "../../lib/library-options";

interface LibraryBucketPageProps {
  basePath: string;
  bucket: LibraryBucket;
  searchParams: Record<string, string | string[] | undefined>;
}

export async function LibraryBucketPage({
  basePath,
  bucket,
  searchParams
}: LibraryBucketPageProps) {
  const currentFilters = {
    mediaType: readMediaType(searchParams.mediaType),
    search: readString(searchParams.search),
    tag: readString(searchParams.tag)
  };
  const currentPage = Number.parseInt(readString(searchParams.page) ?? "1", 10);
  const bucketLabel = bucket === "catalog" ? "Catalog" : "Wishlist";
  const result = await getLibraryEntries({
    ...currentFilters,
    bucket,
    page: Number.isNaN(currentPage) ? 1 : currentPage,
    pageSize: 12
  });

  return (
    <div className="space-y-8">
      <LibraryFilters
        basePath={basePath}
        bucketLabel={bucketLabel}
        currentFilters={currentFilters}
      />

      <CreateLibraryEntryForm bucket={bucket} />

      <LibraryList bucketLabel={bucketLabel} items={result.items} />

      <PaginationControls
        basePath={basePath}
        currentPage={result.pagination.page}
        hasNextPage={result.pagination.hasNextPage}
        query={{
          mediaType: currentFilters.mediaType,
          search: currentFilters.search,
          tag: currentFilters.tag
        }}
        totalPages={result.pagination.totalPages}
      />
    </div>
  );
}

function readString(value: string | string[] | undefined): string | undefined {
  if (typeof value === "string") {
    return value;
  }

  if (Array.isArray(value)) {
    return value[0];
  }

  return undefined;
}

function readMediaType(value: string | string[] | undefined): MediaType | undefined {
  const candidate = readString(value);
  return candidate && mediaTypeOptions.includes(candidate as MediaType)
    ? (candidate as MediaType)
    : undefined;
}
