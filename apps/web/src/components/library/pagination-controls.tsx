import Link from "next/link";

interface PaginationControlsProps {
  basePath: string;
  currentPage: number;
  hasNextPage: boolean;
  totalPages: number;
  query: Record<string, string | undefined>;
}

export function PaginationControls({
  basePath,
  currentPage,
  hasNextPage,
  totalPages,
  query
}: PaginationControlsProps) {
  if (totalPages <= 1) {
    return null;
  }

  return (
    <div className="flex items-center justify-between rounded-3xl border border-slate-800 bg-slate-950/70 p-4 text-sm text-slate-300">
      <Link
        aria-disabled={currentPage <= 1}
        className={`rounded-2xl border px-4 py-2 transition ${
          currentPage <= 1
            ? "pointer-events-none border-slate-800 text-slate-600"
            : "border-slate-700 hover:border-sky-400 hover:text-white"
        }`}
        href={buildHref(basePath, query, currentPage - 1)}
      >
        Previous
      </Link>

      <span>
        Page {currentPage} of {totalPages}
      </span>

      <Link
        aria-disabled={!hasNextPage}
        className={`rounded-2xl border px-4 py-2 transition ${
          !hasNextPage
            ? "pointer-events-none border-slate-800 text-slate-600"
            : "border-slate-700 hover:border-sky-400 hover:text-white"
        }`}
        href={buildHref(basePath, query, currentPage + 1)}
      >
        Next
      </Link>
    </div>
  );
}

function buildHref(
  basePath: string,
  query: Record<string, string | undefined>,
  page: number
): string {
  const searchParams = new URLSearchParams();

  for (const [key, value] of Object.entries(query)) {
    if (value) {
      searchParams.set(key, value);
    }
  }

  if (page > 1) {
    searchParams.set("page", String(page));
  }

  const serialized = searchParams.toString();
  return serialized.length > 0 ? `${basePath}?${serialized}` : basePath;
}
