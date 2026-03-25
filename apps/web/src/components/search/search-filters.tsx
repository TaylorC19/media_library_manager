import { useTranslations } from "next-intl";
import { Link } from "../../i18n/navigation";
import { mediaTypeOptions } from "../../lib/library-options";

interface SearchFiltersProps {
  currentMediaType: string;
  currentQuery: string;
}

export function SearchFilters({
  currentMediaType,
  currentQuery
}: SearchFiltersProps) {
  const tCommon = useTranslations("common");
  const tMediaType = useTranslations("enums.mediaType");
  const tSearch = useTranslations("search");

  return (
    <section className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
            {tCommon("actions.search")}
          </p>
          <h1 className="mt-2 text-3xl font-semibold text-white sm:text-4xl">
            {tSearch("heading")}
          </h1>
          <p className="mt-2 max-w-2xl text-sm text-slate-300">
            {tSearch("description")}
          </p>
        </div>

        <Link
          className="inline-flex rounded-2xl border border-slate-700 px-4 py-2 text-sm font-medium text-slate-200 transition hover:border-sky-400 hover:text-white"
          href="/search"
        >
          {tSearch("clearSearch")}
        </Link>
      </div>

      <form className="mt-6 grid gap-4 md:grid-cols-[1fr_220px_auto]" method="GET">
        <label className="block space-y-2">
          <span className="text-sm font-medium text-slate-200">
            {tSearch("queryLabel")}
          </span>
          <input
            className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
            defaultValue={currentQuery}
            name="q"
            placeholder={tSearch("queryPlaceholder")}
          />
        </label>

        <label className="block space-y-2">
          <span className="text-sm font-medium text-slate-200">
            {tSearch("mediaTypeLabel")}
          </span>
          <select
            className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
            defaultValue={currentMediaType}
            name="mediaType"
          >
            {mediaTypeOptions.map((mediaType) => (
              <option key={mediaType} value={mediaType}>
                {tMediaType(mediaType)}
              </option>
            ))}
          </select>
        </label>

        <button
          className="rounded-2xl bg-sky-400 px-4 py-3 text-sm font-semibold text-slate-950 transition hover:bg-sky-300"
          type="submit"
        >
          {tCommon("actions.search")}
        </button>
      </form>
    </section>
  );
}
