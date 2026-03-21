import type { SearchResponse } from "@media-library/types";
import { getMediaTypeLabel } from "../../lib/media-api";
import { SearchResultActions } from "./search-result-actions";

interface SearchResultsProps {
  response: SearchResponse | null;
}

export function SearchResults({ response }: SearchResultsProps) {
  if (!response) {
    return (
      <section className="rounded-3xl border border-dashed border-slate-800 bg-slate-950/40 p-8 text-center">
        <h2 className="text-2xl font-semibold text-white">Start with a query</h2>
        <p className="mt-3 text-sm text-slate-300">
          Search one media type at a time to get normalized results from the matching
          providers.
        </p>
      </section>
    );
  }

  return (
    <section className="space-y-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
            Results
          </p>
          <h2 className="mt-2 text-2xl font-semibold text-white">
            {response.results.length} match{response.results.length === 1 ? "" : "es"} for
            {" "}
            &ldquo;{response.query}&rdquo;
          </h2>
          <p className="mt-1 text-sm text-slate-400">
            {getMediaTypeLabel(response.mediaType)}
          </p>
        </div>
      </div>

      {response.failures.length > 0 ? (
        <div className="rounded-3xl border border-amber-500/30 bg-amber-500/10 p-4 text-sm text-amber-100">
          Some providers could not be reached:
          {" "}
          {response.failures
            .map((failure) => `${getProviderLabel(failure.provider)} (${failure.message})`)
            .join(", ")}
        </div>
      ) : null}

      {response.results.length === 0 ? (
        <div className="rounded-3xl border border-slate-800 bg-slate-950/60 p-8 text-center">
          <h3 className="text-xl font-semibold text-white">No matches found</h3>
          <p className="mt-2 text-sm text-slate-300">
            Try a broader title, a different spelling, or another media type.
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
                      alt={`${result.title} cover`}
                      className="h-full w-full object-cover"
                      src={result.imageUrl}
                    />
                  ) : (
                    <span>No image</span>
                  )}
                </div>

                <div className="min-w-0 flex-1 space-y-4">
                  <div className="space-y-2">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="rounded-full border border-sky-400/40 bg-sky-400/10 px-3 py-1 text-xs font-medium uppercase tracking-[0.25em] text-sky-200">
                        {getProviderLabel(result.provider)}
                      </span>
                      <span className="rounded-full border border-slate-700 px-3 py-1 text-xs uppercase tracking-[0.25em] text-slate-300">
                        {getMediaTypeLabel(result.mediaType)}
                      </span>
                    </div>

                    <div>
                      <h3 className="text-2xl font-semibold text-white">{result.title}</h3>
                      <p className="mt-1 text-sm text-slate-300">
                        {buildMetaLine(result.year, result.creatorLine, result.subtitle)}
                      </p>
                    </div>
                  </div>

                  {result.summary ? (
                    <p className="text-sm leading-6 text-slate-300">{result.summary}</p>
                  ) : null}

                  <SearchResultActions
                    mediaType={result.mediaType}
                    provider={result.provider}
                    providerId={result.providerId}
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
  year?: number | null,
  creatorLine?: string | null,
  subtitle?: string | null
): string {
  const parts = [year ? String(year) : null, creatorLine ?? null, subtitle ?? null].filter(
    (value): value is string => Boolean(value && value.trim().length > 0)
  );

  return parts.length > 0 ? parts.join(" · ") : "Metadata varies by provider";
}

function getProviderLabel(provider: SearchResponse["failures"][number]["provider"]): string {
  switch (provider) {
    case "tmdb":
      return "TMDB";
    case "musicbrainz":
      return "MusicBrainz";
    case "discogs":
      return "Discogs";
    case "openlibrary":
      return "Open Library";
    case "rawg":
      return "RAWG";
  }
}
