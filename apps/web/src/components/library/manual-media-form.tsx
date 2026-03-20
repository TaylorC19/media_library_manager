"use client";

import type { MediaType } from "@media-library/types";

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
  const contributorLabel = getContributorLabel(mediaType);
  const contributorPlaceholder = getContributorPlaceholder(mediaType);

  return (
    <div className="space-y-5">
      <div className="grid gap-4 sm:grid-cols-2">
        <label className="block space-y-2">
          <span className="text-sm font-medium text-slate-200">Media type</span>
          <select
            className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
            onChange={(event) => onMediaTypeChange(event.target.value as MediaType)}
            value={mediaType}
          >
            <option value="movie">Movie</option>
            <option value="tv">TV</option>
            <option value="album">Album</option>
            <option value="book">Book</option>
            <option value="game">Game</option>
          </select>
        </label>

        <label className="block space-y-2">
          <span className="text-sm font-medium text-slate-200">Year</span>
          <input
            className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
            inputMode="numeric"
            onChange={(event) => onChange({ ...draft, year: event.target.value })}
            placeholder="Optional"
            value={draft.year}
          />
        </label>
      </div>

      <label className="block space-y-2">
        <span className="text-sm font-medium text-slate-200">Title</span>
        <input
          className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
          onChange={(event) => onChange({ ...draft, title: event.target.value })}
          placeholder="Required"
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
          Barcode candidates
        </span>
        <input
          className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
          onChange={(event) =>
            onChange({ ...draft, barcodeCandidates: event.target.value })
          }
          placeholder="Comma-separated UPC/EAN/ISBN values"
          value={draft.barcodeCandidates}
        />
      </label>

      <label className="block space-y-2">
        <span className="text-sm font-medium text-slate-200">Summary</span>
        <textarea
          className="min-h-28 w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
          onChange={(event) => onChange({ ...draft, summary: event.target.value })}
          placeholder="Optional metadata snapshot"
          value={draft.summary}
        />
      </label>
    </div>
  );
}

function getContributorLabel(mediaType: MediaType): string {
  switch (mediaType) {
    case "album":
      return "Artists";
    case "book":
      return "Authors";
    case "movie":
      return "Directors";
    case "tv":
      return "Creators";
    case "game":
      return "Developers";
  }
}

function getContributorPlaceholder(mediaType: MediaType): string {
  switch (mediaType) {
    case "album":
      return "Comma-separated artists";
    case "book":
      return "Comma-separated authors";
    case "movie":
      return "Comma-separated directors";
    case "tv":
      return "Comma-separated creators";
    case "game":
      return "Comma-separated developers";
  }
}
