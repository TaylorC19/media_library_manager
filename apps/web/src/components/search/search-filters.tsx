import Link from "next/link";
import { mediaTypeOptions } from "../../lib/library-options";

interface SearchFiltersProps {
  currentMediaType: string;
  currentQuery: string;
}

export function SearchFilters({
  currentMediaType,
  currentQuery
}: SearchFiltersProps) {
  return (
    <section className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
            Search
          </p>
          <h1 className="mt-2 text-3xl font-semibold text-white sm:text-4xl">
            Search across providers
          </h1>
          <p className="mt-2 max-w-2xl text-sm text-slate-300">
            Find a title, then import it into your catalog or wishlist without exposing
            raw provider data to the UI.
          </p>
        </div>

        <Link
          className="inline-flex rounded-2xl border border-slate-700 px-4 py-2 text-sm font-medium text-slate-200 transition hover:border-sky-400 hover:text-white"
          href="/search"
        >
          Clear search
        </Link>
      </div>

      <form className="mt-6 grid gap-4 md:grid-cols-[1fr_220px_auto]" method="GET">
        <label className="block space-y-2">
          <span className="text-sm font-medium text-slate-200">Query</span>
          <input
            className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
            defaultValue={currentQuery}
            name="q"
            placeholder="Title, artist, author, director..."
          />
        </label>

        <label className="block space-y-2">
          <span className="text-sm font-medium text-slate-200">Media type</span>
          <select
            className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
            defaultValue={currentMediaType}
            name="mediaType"
          >
            {mediaTypeOptions.map((mediaType) => (
              <option key={mediaType} value={mediaType}>
                {mediaType === "tv"
                  ? "TV"
                  : mediaType.charAt(0).toUpperCase() + mediaType.slice(1)}
              </option>
            ))}
          </select>
        </label>

        <button
          className="rounded-2xl bg-sky-400 px-4 py-3 text-sm font-semibold text-slate-950 transition hover:bg-sky-300"
          type="submit"
        >
          Search
        </button>
      </form>
    </section>
  );
}
