"use client";

import type {
  BarcodeLookupCandidate,
  LibraryBucket
} from "@media-library/types";
import { useTranslations } from "next-intl";
import { Link } from "../../i18n/navigation";
import { getBucketLabel, getMediaTypeLabel, getProviderLabel } from "../../i18n/ui";

interface SelectedScanCandidatePanelProps {
  activeBucket: LibraryBucket | null;
  candidate: BarcodeLookupCandidate | null;
  errorMessage: string | null;
  onAdd: (bucket: LibraryBucket) => void;
}

export function SelectedScanCandidatePanel({
  activeBucket,
  candidate,
  errorMessage,
  onAdd
}: SelectedScanCandidatePanelProps) {
  const tBucket = useTranslations("enums.bucket");
  const tMediaType = useTranslations("enums.mediaType");
  const tProvider = useTranslations("enums.provider");
  const tScan = useTranslations("scan");

  if (!candidate) {
    return (
      <section className="rounded-3xl border border-dashed border-slate-800 bg-slate-950/40 p-6">
        <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
          {tScan("selectedLabel")}
        </p>
        <h2 className="mt-2 text-2xl font-semibold text-white">
          {tScan("selectedEmptyTitle")}
        </h2>
        <p className="mt-2 text-sm text-slate-300">
          {tScan("selectedEmptyDescription")}
        </p>
      </section>
    );
  }

  const existingBuckets =
    candidate.source === "local" ? new Set(candidate.linkedLibraryEntries.map((entry) => entry.bucket)) : null;
  const existingEntryId =
    candidate.source === "local" ? candidate.linkedLibraryEntries[0]?.entryId ?? null : null;

  return (
    <section className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="space-y-3">
          <div className="flex flex-wrap items-center gap-2">
            <span className="rounded-full border border-slate-700 px-3 py-1 text-xs uppercase tracking-[0.25em] text-slate-300">
              {getMediaTypeLabel(tMediaType, candidate.mediaType)}
            </span>
            {candidate.source === "provider" ? (
              <span className="rounded-full border border-sky-400/40 bg-sky-400/10 px-3 py-1 text-xs font-medium uppercase tracking-[0.25em] text-sky-200">
                {getProviderLabel(tProvider, candidate.provider)}
              </span>
            ) : (
              <span className="rounded-full border border-emerald-400/40 bg-emerald-400/10 px-3 py-1 text-xs font-medium uppercase tracking-[0.25em] text-emerald-200">
                {tScan("localMatchBadge")}
              </span>
            )}
          </div>

          <div>
            <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
              {tScan("selectedLabel")}
            </p>
            <h2 className="mt-2 text-2xl font-semibold text-white">{candidate.title}</h2>
            <p className="mt-1 text-sm text-slate-300">
              {buildSelectedSummary(candidate, tScan("metadataFallback"))}
            </p>
          </div>

          {candidate.source === "local" ? (
            <LocalCandidateSummary
              existingBuckets={existingBuckets}
            />
          ) : (
            <p className="text-sm text-slate-300">
              {tScan("providerSelectionDescription")}
            </p>
          )}
        </div>

        {candidate.imageUrl ? (
          <div className="flex h-28 w-24 shrink-0 items-center justify-center overflow-hidden rounded-2xl border border-slate-800 bg-slate-900">
            <img
              alt={tScan("coverAlt", { title: candidate.title })}
              className="h-full w-full object-cover"
              src={candidate.imageUrl}
            />
          </div>
        ) : null}
      </div>

      <div className="mt-6 flex flex-col gap-3 sm:flex-row">
        <button
          className="rounded-2xl bg-sky-400 px-4 py-3 text-sm font-semibold text-slate-950 transition hover:bg-sky-300 disabled:cursor-not-allowed disabled:opacity-70"
          disabled={activeBucket !== null || hasBucket(candidate, "catalog")}
          onClick={() => onAdd("catalog")}
          type="button"
        >
          {activeBucket === "catalog"
            ? tScan("actions.adding")
            : hasBucket(candidate, "catalog")
              ? tScan("actions.alreadyInCatalog")
              : tScan("actions.addToCatalog")}
        </button>

        <button
          className="rounded-2xl border border-slate-700 px-4 py-3 text-sm font-semibold text-slate-100 transition hover:border-sky-400 hover:text-white disabled:cursor-not-allowed disabled:opacity-70"
          disabled={activeBucket !== null || hasBucket(candidate, "wishlist")}
          onClick={() => onAdd("wishlist")}
          type="button"
        >
          {activeBucket === "wishlist"
            ? tScan("actions.adding")
            : hasBucket(candidate, "wishlist")
              ? tScan("actions.alreadyInWishlist")
              : tScan("actions.addToWishlist")}
        </button>

        {existingEntryId ? (
          <Link
            className="rounded-2xl border border-emerald-500/40 bg-emerald-500/10 px-4 py-3 text-center text-sm font-semibold text-emerald-100 transition hover:border-emerald-400 hover:text-white"
            href={`/library/${existingEntryId}`}
          >
            {tScan("actions.viewExistingEntry")}
          </Link>
        ) : null}
      </div>

      {candidate.source === "local" && existingBuckets && existingBuckets.size > 0 ? (
        <p className="mt-3 text-sm text-slate-400">
          {Array.from(existingBuckets)
            .map((bucket) => getBucketLabel(tBucket, bucket))
            .join(", ")}
        </p>
      ) : null}

      {errorMessage ? (
        <p className="mt-4 rounded-2xl border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-200">
          {errorMessage}
        </p>
      ) : null}
    </section>
  );
}

function LocalCandidateSummary({
  existingBuckets
}: {
  existingBuckets: Set<LibraryBucket> | null;
}) {
  const tBucket = useTranslations("enums.bucket");
  const tScan = useTranslations("scan");

  if (!existingBuckets || existingBuckets.size === 0) {
    return <p className="text-sm text-slate-300">{tScan("localRecordDescription")}</p>;
  }

  return (
    <div className="space-y-2">
      <p className="text-sm text-slate-300">{tScan("existingBucketsDescription")}</p>
      <div className="flex flex-wrap gap-2">
        {Array.from(existingBuckets).map((bucket) => (
          <span
            key={bucket}
            className="rounded-full border border-slate-700 px-3 py-1 text-xs uppercase tracking-[0.25em] text-slate-300"
          >
            {getBucketLabel(tBucket, bucket)}
          </span>
        ))}
      </div>
    </div>
  );
}

function buildSelectedSummary(
  candidate: BarcodeLookupCandidate,
  metadataFallback: string
): string {
  const parts = [candidate.year ? String(candidate.year) : null, candidate.creatorLine ?? null].filter(
    (value): value is string => Boolean(value && value.trim().length > 0)
  );

  return parts.length > 0 ? parts.join(" · ") : metadataFallback;
}

function hasBucket(candidate: BarcodeLookupCandidate, bucket: LibraryBucket): boolean {
  return candidate.source === "local"
    ? candidate.linkedLibraryEntries.some((entry) => entry.bucket === bucket)
    : false;
}
