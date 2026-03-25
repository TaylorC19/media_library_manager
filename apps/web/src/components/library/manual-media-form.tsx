"use client";

import type { MediaType } from "@media-library/types";
import { useTranslations } from "next-intl";

export interface ManualMediaDraft {
  title: string;
  year: string;
  summary: string;
  contributors: string;
  barcodeCandidates: string;
}

interface ManualMediaFormProps {
  draft: ManualMediaDraft;
  mediaType: MediaType;
  onChange: (nextDraft: ManualMediaDraft) => void;
  onMediaTypeChange: (mediaType: MediaType) => void;
}

export function ManualMediaForm({
  draft,
  mediaType,
  onChange,
  onMediaTypeChange
}: ManualMediaFormProps) {
  const tCommon = useTranslations("common");
  const tManualMedia = useTranslations("library.manualMedia");
  const tMediaType = useTranslations("enums.mediaType");
  const tContributorLabel = useTranslations("library.manualMedia.contributorLabel");
  const tContributorPlaceholder = useTranslations(
    "library.manualMedia.contributorPlaceholder"
  );
  const contributorLabel = getContributorLabel(tContributorLabel, mediaType);
  const contributorPlaceholder = getContributorPlaceholder(
    tContributorPlaceholder,
    mediaType
  );

  return (
    <div className="space-y-5">
      <div className="grid gap-4 sm:grid-cols-2">
        <label className="block space-y-2">
          <span className="text-sm font-medium text-slate-200">
            {tManualMedia("mediaType")}
          </span>
          <select
            className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
            onChange={(event) => onMediaTypeChange(event.target.value as MediaType)}
            value={mediaType}
          >
            <option value="movie">{tMediaType("movie")}</option>
            <option value="tv">{tMediaType("tv")}</option>
            <option value="album">{tMediaType("album")}</option>
            <option value="book">{tMediaType("book")}</option>
            <option value="game">{tMediaType("game")}</option>
          </select>
        </label>

        <label className="block space-y-2">
          <span className="text-sm font-medium text-slate-200">
            {tManualMedia("year")}
          </span>
          <input
            className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
            inputMode="numeric"
            onChange={(event) => onChange({ ...draft, year: event.target.value })}
            placeholder={tCommon("optional")}
            value={draft.year}
          />
        </label>
      </div>

      <label className="block space-y-2">
        <span className="text-sm font-medium text-slate-200">
          {tManualMedia("title")}
        </span>
        <input
          className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
          onChange={(event) => onChange({ ...draft, title: event.target.value })}
          placeholder={tCommon("required")}
          required
          value={draft.title}
        />
      </label>

      <label className="block space-y-2">
        <span className="text-sm font-medium text-slate-200">{contributorLabel}</span>
        <input
          className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
          onChange={(event) =>
            onChange({ ...draft, contributors: event.target.value })
          }
          placeholder={contributorPlaceholder}
          value={draft.contributors}
        />
      </label>

      <label className="block space-y-2">
        <span className="text-sm font-medium text-slate-200">
          {tManualMedia("barcodeCandidates")}
        </span>
        <input
          className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
          onChange={(event) =>
            onChange({ ...draft, barcodeCandidates: event.target.value })
          }
          placeholder={tManualMedia("barcodeCandidatesPlaceholder")}
          value={draft.barcodeCandidates}
        />
      </label>

      <label className="block space-y-2">
        <span className="text-sm font-medium text-slate-200">
          {tManualMedia("summary")}
        </span>
        <textarea
          className="min-h-28 w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
          onChange={(event) => onChange({ ...draft, summary: event.target.value })}
          placeholder={tManualMedia("summaryPlaceholder")}
          value={draft.summary}
        />
      </label>
    </div>
  );
}

function getContributorLabel(
  tContributorLabel: (key: string) => string,
  mediaType: MediaType
): string {
  switch (mediaType) {
    case "album":
      return tContributorLabel("album");
    case "book":
      return tContributorLabel("book");
    case "movie":
      return tContributorLabel("movie");
    case "tv":
      return tContributorLabel("tv");
    case "game":
      return tContributorLabel("game");
  }
}

function getContributorPlaceholder(
  tContributorPlaceholder: (key: string) => string,
  mediaType: MediaType
): string {
  switch (mediaType) {
    case "album":
      return tContributorPlaceholder("album");
    case "book":
      return tContributorPlaceholder("book");
    case "movie":
      return tContributorPlaceholder("movie");
    case "tv":
      return tContributorPlaceholder("tv");
    case "game":
      return tContributorPlaceholder("game");
  }
}
