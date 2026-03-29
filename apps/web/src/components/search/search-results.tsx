import type { SearchResponse } from "@media-library/types";
import { useTranslations } from "next-intl";
import { getMediaTypeLabel, getProviderLabel } from "../../i18n/ui";
import { SearchResultActions } from "./search-result-actions";

interface SearchResultsProps {
  response: SearchResponse | null;
}

export function SearchResults({ response }: SearchResultsProps) {
  const tCommon = useTranslations("common");
  const tMediaType = useTranslations("enums.mediaType");
  const tProvider = useTranslations("enums.provider");
  const tSearch = useTranslations("search");

  if (!response) {
    return (
      <section className="rounded-3xl border border-dashed border-slate-800 bg-slate-950/40 p-8 text-center">
        <h2 className="text-2xl font-semibold text-white">
          {tSearch("emptyQueryTitle")}
        </h2>
        <p className="mt-3 text-sm text-slate-300">
          {tSearch("emptyQueryDescription")}
        </p>
      </section>
    );
  }

  return (
    <section className="space-y-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
            {tSearch("resultsLabel")}
          </p>
          <h2 className="mt-2 text-2xl font-semibold text-white">
            {tSearch("resultsTitle", {
              count: response.results.length,
              query: response.query
            })}
          </h2>
          <p className="mt-1 text-sm text-slate-400">
            {getMediaTypeLabel(tMediaType, response.mediaType)}
          </p>
        </div>
      </div>

      {response.failures.length > 0 ? (
        <div className="rounded-3xl border border-amber-500/30 bg-amber-500/10 p-4 text-sm text-amber-100">
          {tSearch("failurePrefix")}
          {" "}
          {response.failures
            .map(
              (failure) =>
                `${getProviderLabel(tProvider, failure.provider)} (${failure.message})`
            )
            .join(", ")}
        </div>
      ) : null}

      {response.results.length === 0 ? (
        <div className="rounded-3xl border border-slate-800 bg-slate-950/60 p-8 text-center">
          <h3 className="text-xl font-semibold text-white">
            {tSearch("noResultsTitle")}
          </h3>
          <p className="mt-2 text-sm text-slate-300">
            {tSearch("noResultsDescription")}
          </p>
        </div>
      ) : (
        <div className="grid gap-4">
          {response.results.map((result) => (
            <article
              key={`${result.provider}:${result.providerId}`}
              className="rounded-3xl border border-slate-800 bg-slate-950/70 p-5"
            >
              <div className="flex flex-col gap-5 sm:flex-row">
                <div className="flex h-40 w-full shrink-0 items-center justify-center overflow-hidden rounded-2xl border border-slate-800 bg-slate-900 text-sm text-slate-500 sm:w-28">
                  {result.imageUrl ? (
                    <img
                      alt={tSearch("coverAlt", { title: result.title })}
                      className="h-full w-full object-cover"
                      src={result.imageUrl}
                    />
                  ) : (
                    <span>{tCommon("states.noImage")}</span>
                  )}
                </div>

                <div className="min-w-0 flex-1 space-y-4">
                  <div className="space-y-2">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="rounded-full border border-sky-400/40 bg-sky-400/10 px-3 py-1 text-xs font-medium uppercase tracking-[0.25em] text-sky-200">
                        {getProviderLabel(tProvider, result.provider)}
                      </span>
                      <span className="rounded-full border border-slate-700 px-3 py-1 text-xs uppercase tracking-[0.25em] text-slate-300">
                        {getMediaTypeLabel(tMediaType, result.mediaType)}
                      </span>
                    </div>

                    <div>
                      <h3 className="text-2xl font-semibold text-white">{result.title}</h3>
                      <p className="mt-1 text-sm text-slate-300">
                        {buildMetaLine(
                          tSearch("metadataFallback"),
                          result.year,
                          result.creatorLine,
                          result.subtitle
                        )}
                      </p>
                    </div>
                  </div>

                  {result.summary ? (
                    <p className="text-sm leading-6 text-slate-300">{result.summary}</p>
                  ) : null}

                  <SearchResultActions
                    result={result}
                  />
                </div>
              </div>
            </article>
          ))}
        </div>
      )}
    </section>
  );
}

function buildMetaLine(
  metadataFallback: string,
  year?: number | null,
  creatorLine?: string | null,
  subtitle?: string | null
): string {
  const parts = [year ? String(year) : null, creatorLine ?? null, subtitle ?? null].filter(
    (value): value is string => Boolean(value && value.trim().length > 0)
  );

  return parts.length > 0 ? parts.join(" · ") : metadataFallback;
}
