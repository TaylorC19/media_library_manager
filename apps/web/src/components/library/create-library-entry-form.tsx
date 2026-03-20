"use client";

import type {
  CreateLibraryEntryRequest,
  CreateManualMediaRecordRequest,
  LibraryBucket,
  LibraryEntryResponse,
  ManualMediaRecordResponse,
  MediaType,
  PhysicalFormat
} from "@media-library/types";
import { useRouter } from "next/navigation";
import type { FormEvent } from "react";
import { useMemo, useState } from "react";
import { browserApiFetch } from "../../lib/api-client";
import { physicalFormatOptions } from "../../lib/library-options";
import {
  ManualMediaForm,
  type ManualMediaDraft
} from "./manual-media-form";

interface CreateLibraryEntryFormProps {
  bucket: LibraryBucket;
}

interface EntryDraft {
  barcode: string;
  format: PhysicalFormat | "";
  notes: string;
  purchaseDate: string;
  tags: string;
}

const emptyMediaDraft: ManualMediaDraft = {
  barcodeCandidates: "",
  contributors: "",
  summary: "",
  title: "",
  year: ""
};

const emptyEntryDraft: EntryDraft = {
  barcode: "",
  format: "",
  notes: "",
  purchaseDate: "",
  tags: ""
};

export function CreateLibraryEntryForm({
  bucket
}: CreateLibraryEntryFormProps) {
  const router = useRouter();
  const [mediaType, setMediaType] = useState<MediaType>("movie");
  const [mediaDraft, setMediaDraft] = useState<ManualMediaDraft>(emptyMediaDraft);
  const [entryDraft, setEntryDraft] = useState<EntryDraft>(emptyEntryDraft);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const bucketTitle = useMemo(
    () => (bucket === "catalog" ? "Add to catalog" : "Add to wishlist"),
    [bucket]
  );

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setErrorMessage(null);
    setIsSubmitting(true);

    try {
      const mediaResponse = await browserApiFetch("/media/manual", {
        body: JSON.stringify(buildManualMediaPayload(mediaType, mediaDraft)),
        headers: {
          "content-type": "application/json"
        },
        method: "POST"
      });

      if (!mediaResponse.ok) {
        setErrorMessage(await readErrorMessage(mediaResponse));
        return;
      }

      const mediaPayload = (await mediaResponse.json()) as ManualMediaRecordResponse;
      const libraryResponse = await browserApiFetch("/library", {
        body: JSON.stringify(
          buildCreateLibraryPayload(bucket, mediaPayload.mediaRecord.id, entryDraft)
        ),
        headers: {
          "content-type": "application/json"
        },
        method: "POST"
      });

      if (!libraryResponse.ok) {
        setErrorMessage(await readErrorMessage(libraryResponse));
        return;
      }

      const libraryPayload = (await libraryResponse.json()) as LibraryEntryResponse;
      setMediaDraft(emptyMediaDraft);
      setEntryDraft(emptyEntryDraft);
      router.push(`/library/${libraryPayload.entry.id}`);
      router.refresh();
    } catch {
      setErrorMessage("Unable to reach the API right now.");
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <section className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6">
      <div className="space-y-3">
        <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
          Manual add
        </p>
        <h2 className="text-2xl font-semibold text-white">{bucketTitle}</h2>
        <p className="max-w-2xl text-sm text-slate-300">
          Create a minimal media record, then link it into your private library.
        </p>
      </div>

      <form className="mt-6 space-y-8" onSubmit={handleSubmit}>
        <ManualMediaForm
          draft={mediaDraft}
          mediaType={mediaType}
          onChange={setMediaDraft}
          onMediaTypeChange={setMediaType}
        />

        <div className="space-y-5 border-t border-slate-800 pt-6">
          <div>
            <h3 className="text-lg font-semibold text-white">My copy</h3>
            <p className="mt-1 text-sm text-slate-300">
              Optional fields stored on your personal library entry.
            </p>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block space-y-2">
              <span className="text-sm font-medium text-slate-200">Format</span>
              <select
                className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
                onChange={(event) =>
                  setEntryDraft({
                    ...entryDraft,
                    format: event.target.value as PhysicalFormat | ""
                  })
                }
                value={entryDraft.format}
              >
                <option value="">Unspecified</option>
                {physicalFormatOptions.map((format) => (
                  <option key={format} value={format}>
                    {format.replaceAll("_", " ")}
                  </option>
                ))}
              </select>
            </label>

            <label className="block space-y-2">
              <span className="text-sm font-medium text-slate-200">
                Purchase date
              </span>
              <input
                className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
                onChange={(event) =>
                  setEntryDraft({ ...entryDraft, purchaseDate: event.target.value })
                }
                type="date"
                value={entryDraft.purchaseDate}
              />
            </label>
          </div>

          <label className="block space-y-2">
            <span className="text-sm font-medium text-slate-200">Barcode</span>
            <input
              className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
              onChange={(event) =>
                setEntryDraft({ ...entryDraft, barcode: event.target.value })
              }
              placeholder="Optional"
              value={entryDraft.barcode}
            />
          </label>

          <label className="block space-y-2">
            <span className="text-sm font-medium text-slate-200">Tags</span>
            <input
              className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
              onChange={(event) =>
                setEntryDraft({ ...entryDraft, tags: event.target.value })
              }
              placeholder="Comma-separated"
              value={entryDraft.tags}
            />
          </label>

          <label className="block space-y-2">
            <span className="text-sm font-medium text-slate-200">Notes</span>
            <textarea
              className="min-h-28 w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
              onChange={(event) =>
                setEntryDraft({ ...entryDraft, notes: event.target.value })
              }
              placeholder="Edition details, condition, where you found it..."
              value={entryDraft.notes}
            />
          </label>
        </div>

        {errorMessage ? (
          <p className="rounded-2xl border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-200">
            {errorMessage}
          </p>
        ) : null}

        <button
          className="w-full rounded-2xl bg-sky-400 px-4 py-3 text-sm font-semibold text-slate-950 transition hover:bg-sky-300 disabled:cursor-not-allowed disabled:opacity-70"
          disabled={isSubmitting}
          type="submit"
        >
          {isSubmitting ? "Saving..." : bucketTitle}
        </button>
      </form>
    </section>
  );
}

function buildManualMediaPayload(
  mediaType: MediaType,
  draft: ManualMediaDraft
): CreateManualMediaRecordRequest {
  const contributors = splitCommaSeparated(draft.contributors);
  const basePayload = {
    barcodeCandidates: splitCommaSeparated(draft.barcodeCandidates),
    summary: draft.summary || undefined,
    title: draft.title.trim(),
    year: draft.year ? Number.parseInt(draft.year, 10) : undefined
  };

  switch (mediaType) {
    case "album":
      return {
        ...basePayload,
        details: {
          artists: contributors
        },
        mediaType
      };
    case "book":
      return {
        ...basePayload,
        details: {
          authors: contributors
        },
        mediaType
      };
    case "movie":
      return {
        ...basePayload,
        details: contributors.length > 0 ? { directors: contributors } : undefined,
        mediaType
      };
    case "tv":
      return {
        ...basePayload,
        details: contributors.length > 0 ? { creators: contributors } : undefined,
        mediaType
      };
    case "game":
      return {
        ...basePayload,
        details: contributors.length > 0 ? { developers: contributors } : undefined,
        mediaType
      };
  }
}

function buildCreateLibraryPayload(
  bucket: LibraryBucket,
  mediaRecordId: string,
  draft: EntryDraft
): CreateLibraryEntryRequest {
  return {
    barcode: draft.barcode || null,
    bucket,
    format: draft.format || null,
    mediaRecordId,
    notes: draft.notes || null,
    purchaseDate: draft.purchaseDate || null,
    tags: splitCommaSeparated(draft.tags)
  };
}

function splitCommaSeparated(value: string): string[] {
  return Array.from(
    new Set(
      value
        .split(",")
        .map((part) => part.trim())
        .filter((part) => part.length > 0)
    )
  );
}

async function readErrorMessage(response: Response): Promise<string> {
  const payload = (await response.json().catch(() => null)) as
    | { message?: string | string[] }
    | null;

  if (typeof payload?.message === "string") {
    return payload.message;
  }

  if (Array.isArray(payload?.message)) {
    return payload.message.join(", ");
  }

  return "Something went wrong. Please try again.";
}
