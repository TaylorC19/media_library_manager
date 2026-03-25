import { useTranslations } from "next-intl";
import { Link } from "../../i18n/navigation";
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
  const tCommon = useTranslations("common");
  const tFilters = useTranslations("library.filters");
  const tMediaType = useTranslations("enums.mediaType");

  return (
    <section className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
            {bucketLabel}
          </p>
          <h2 className="mt-2 text-2xl font-semibold text-white">
            {tFilters("heading")}
          </h2>
        </div>

        <Link
          className="inline-flex rounded-2xl border border-slate-700 px-4 py-2 text-sm font-medium text-slate-200 transition hover:border-sky-400 hover:text-white"
          href={basePath}
        >
          {tCommon("actions.clearFilters")}
        </Link>
      </div>

      <form className="mt-6 grid gap-4 md:grid-cols-[1fr_1fr_1fr_auto]" method="GET">
        <label className="block space-y-2">
          <span className="text-sm font-medium text-slate-200">
            {tFilters("searchLabel")}
          </span>
          <input
            className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
            defaultValue={currentFilters.search ?? ""}
            name="search"
            placeholder={tFilters("searchPlaceholder")}
          />
        </label>

        <label className="block space-y-2">
          <span className="text-sm font-medium text-slate-200">
            {tFilters("mediaTypeLabel")}
          </span>
          <select
            className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
            defaultValue={currentFilters.mediaType ?? ""}
            name="mediaType"
          >
            <option value="">{tFilters("allMedia")}</option>
            {mediaTypeOptions.map((mediaType) => (
              <option key={mediaType} value={mediaType}>
                {tMediaType(mediaType)}
              </option>
            ))}
          </select>
        </label>

        <label className="block space-y-2">
          <span className="text-sm font-medium text-slate-200">
            {tFilters("tagLabel")}
          </span>
          <input
            className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
            defaultValue={currentFilters.tag ?? ""}
            name="tag"
            placeholder={tFilters("tagPlaceholder")}
          />
        </label>

        <button
          className="rounded-2xl bg-sky-400 px-4 py-3 text-sm font-semibold text-slate-950 transition hover:bg-sky-300"
          type="submit"
        >
          {tCommon("actions.apply")}
        </button>
      </form>
    </section>
  );
}
