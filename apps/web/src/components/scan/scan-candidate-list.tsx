"use client";

import type {
  BarcodeLookupCandidate,
  BarcodeLookupLocalCandidate
} from "@media-library/types";
import { useTranslations } from "next-intl";
import { getMediaTypeLabel, getProviderLabel } from "../../i18n/ui";

interface ScanCandidateListProps {
  candidates: BarcodeLookupCandidate[];
  selectedCandidateKey: string | null;
  onSelect: (candidateKey: string) => void;
}

export function ScanCandidateList({
  candidates,
  selectedCandidateKey,
  onSelect
}: ScanCandidateListProps) {
  const tBucket = useTranslations("enums.bucket");
  const tCommon = useTranslations("common");
  const tMediaType = useTranslations("enums.mediaType");
  const tProvider = useTranslations("enums.provider");
  const tScan = useTranslations("scan");

  return (
    <section className="space-y-4">
      <div>
        <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
          {tScan("resultsLabel")}
        </p>
        <h2 className="mt-2 text-2xl font-semibold text-white">
          {tScan("resultsTitle", { count: candidates.length })}
        </h2>
        <p className="mt-1 text-sm text-slate-300">{tScan("resultsDescription")}</p>
      </div>

      <div className="grid gap-4">
        {candidates.map((candidate) => {
          const candidateKey = getCandidateKey(candidate);
          const isSelected = selectedCandidateKey === candidateKey;
          const isLocal = candidate.source === "local";
          const localCandidate = isLocal ? candidate : null;

          return (
            <button
              key={candidateKey}
              className={`w-full rounded-3xl border p-5 text-left transition ${
                isSelected
                  ? "border-sky-400 bg-sky-400/10"
                  : "border-slate-800 bg-slate-950/70 hover:border-sky-500/50"
              }`}
              onClick={() => onSelect(candidateKey)}
              type="button"
            >
              <div className="flex flex-col gap-5 sm:flex-row">
                <div className="flex h-36 w-full shrink-0 items-center justify-center overflow-hidden rounded-2xl border border-slate-800 bg-slate-900 text-sm text-slate-500 sm:w-28">
                  {candidate.imageUrl ? (
                    <img
                      alt={tScan("coverAlt", { title: candidate.title })}
                      className="h-full w-full object-cover"
                      src={candidate.imageUrl}
                    />
                  ) : (
                    <span>{tCommon("states.noImage")}</span>
                  )}
                </div>

                <div className="min-w-0 flex-1 space-y-4">
                  <div className="space-y-2">
                    <div className="flex flex-wrap items-center gap-2">
                      {candidate.source === "provider" ? (
                        <span className="rounded-full border border-sky-400/40 bg-sky-400/10 px-3 py-1 text-xs font-medium uppercase tracking-[0.25em] text-sky-200">
                          {getProviderLabel(tProvider, candidate.provider)}
                        </span>
                      ) : (
                        <span className="rounded-full border border-emerald-400/40 bg-emerald-400/10 px-3 py-1 text-xs font-medium uppercase tracking-[0.25em] text-emerald-200">
                          {tScan("localMatchBadge")}
                        </span>
                      )}

                      <span className="rounded-full border border-slate-700 px-3 py-1 text-xs uppercase tracking-[0.25em] text-slate-300">
                        {getMediaTypeLabel(tMediaType, candidate.mediaType)}
                      </span>

                      {isSelected ? (
                        <span className="rounded-full border border-sky-400/40 bg-sky-400/10 px-3 py-1 text-xs uppercase tracking-[0.25em] text-sky-100">
                          {tScan("actions.selected")}
                        </span>
                      ) : null}
                    </div>

                    <div>
                      <h3 className="text-xl font-semibold text-white">{candidate.title}</h3>
                      <p className="mt-1 text-sm text-slate-300">
                        {buildMetaLine(
                          tScan("metadataFallback"),
                          candidate.year,
                          candidate.creatorLine
                        )}
                      </p>
                    </div>
                  </div>

                  {localCandidate ? (
                    <div className="space-y-2">
                      <p className="text-sm text-slate-300">
                        {localCandidate.hasLinkedLibraryEntry
                          ? tScan("existingBucketsDescription")
                          : tScan("localRecordDescription")}
                      </p>
                      <div className="flex flex-wrap gap-2">
                        {localCandidate.linkedLibraryEntries.length > 0 ? (
                          localCandidate.linkedLibraryEntries.map((entry) => (
                            <span
                              key={entry.entryId}
                              className="rounded-full border border-slate-700 px-3 py-1 text-xs uppercase tracking-[0.25em] text-slate-300"
                            >
                              {tBucket(entry.bucket)}
                            </span>
                          ))
                        ) : (
                          <span className="rounded-full border border-slate-700 px-3 py-1 text-xs uppercase tracking-[0.25em] text-slate-400">
                            {tScan("notInLibraryYet")}
                          </span>
                        )}
                      </div>
                    </div>
                  ) : null}

                  <div className="text-sm font-medium text-sky-200">
                    {isSelected ? tScan("selectionActive") : tScan("actions.selectCandidate")}
                  </div>
                </div>
              </div>
            </button>
          );
        })}
      </div>
    </section>
  );
}

function buildMetaLine(
  metadataFallback: string,
  year?: number | null,
  creatorLine?: string | null
): string {
  const parts = [year ? String(year) : null, creatorLine ?? null].filter(
    (value): value is string => Boolean(value && value.trim().length > 0)
  );

  return parts.length > 0 ? parts.join(" · ") : metadataFallback;
}

export function getCandidateKey(candidate: BarcodeLookupCandidate): string {
  return candidate.source === "local"
    ? `local:${candidate.mediaRecordId}`
    : `provider:${candidate.provider}:${candidate.providerId}`;
}

export function getExistingLocalEntryId(
  candidate: BarcodeLookupLocalCandidate
): string | null {
  return candidate.linkedLibraryEntries[0]?.entryId ?? null;
}
