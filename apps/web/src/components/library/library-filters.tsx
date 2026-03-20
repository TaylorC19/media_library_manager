import Link from "next/link";
import { mediaTypeOptions } from "../../lib/library-options";

interface LibraryFiltersProps {
  basePath: string;
  bucketLabel: string;
  currentFilters: {
    mediaType?: string;
    search?: string;
    tag?: string;
  };
}

export function LibraryFilters({
  basePath,
  bucketLabel,
  currentFilters
}: LibraryFiltersProps) {
  return (
    <section className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
            {bucketLabel}
          </p>
          <h2 className="mt-2 text-2xl font-semibold text-white">
            Filter your collection
          </h2>
        </div>

        <Link
          className="inline-flex rounded-2xl border border-slate-700 px-4 py-2 text-sm font-medium text-slate-200 transition hover:border-sky-400 hover:text-white"
          href={basePath}
        >
          Clear filters
        </Link>
      </div>

      <form className="mt-6 grid gap-4 md:grid-cols-[1fr_1fr_1fr_auto]" method="GET">
        <label className="block space-y-2">
          <span className="text-sm font-medium text-slate-200">Search</span>
          <input
            className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
            defaultValue={currentFilters.search ?? ""}
            name="search"
            placeholder="Title, creator, notes, barcode..."
          />
        </label>

        <label className="block space-y-2">
          <span className="text-sm font-medium text-slate-200">Media type</span>
          <select
            className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
            defaultValue={currentFilters.mediaType ?? ""}
            name="mediaType"
          >
            <option value="">All media</option>
            {mediaTypeOptions.map((mediaType) => (
              <option key={mediaType} value={mediaType}>
                {mediaType === "tv"
                  ? "TV"
                  : mediaType.charAt(0).toUpperCase() + mediaType.slice(1)}
              </option>
            ))}
          </select>
        </label>

        <label className="block space-y-2">
          <span className="text-sm font-medium text-slate-200">Tag</span>
          <input
            className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
            defaultValue={currentFilters.tag ?? ""}
            name="tag"
            placeholder="vinyl, gift, signed..."
          />
        </label>

        <button
          className="rounded-2xl bg-sky-400 px-4 py-3 text-sm font-semibold text-slate-950 transition hover:bg-sky-300"
          type="submit"
        >
          Apply
        </button>
      </form>
    </section>
  );
}
