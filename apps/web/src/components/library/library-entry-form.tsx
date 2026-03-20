"use client";

import type {
  LibraryBucket,
  LibraryEntry,
  PhysicalFormat,
  UpdateLibraryEntryRequest
} from "@media-library/types";
import { useRouter } from "next/navigation";
import type { FormEvent } from "react";
import { useState } from "react";
import { browserApiFetch } from "../../lib/api-client";
import {
  libraryBucketOptions,
  physicalFormatOptions
} from "../../lib/library-options";

interface LibraryEntryFormProps {
  entry: LibraryEntry;
}

export function LibraryEntryForm({ entry }: LibraryEntryFormProps) {
  const router = useRouter();
  const [bucket, setBucket] = useState<LibraryBucket>(entry.bucket);
  const [format, setFormat] = useState<PhysicalFormat | "">(entry.format ?? "");
  const [barcode, setBarcode] = useState(entry.barcode ?? "");
  const [purchaseDate, setPurchaseDate] = useState(entry.purchaseDate ?? "");
  const [notes, setNotes] = useState(entry.notes ?? "");
  const [tags, setTags] = useState(entry.tags.join(", "));
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setErrorMessage(null);
    setSuccessMessage(null);
    setIsSubmitting(true);

    try {
      const response = await browserApiFetch(`/library/${entry.id}`, {
        body: JSON.stringify({
          barcode: barcode || null,
          bucket,
          format: format || null,
          notes: notes || null,
          purchaseDate: purchaseDate || null,
          tags: splitCommaSeparated(tags)
        } satisfies UpdateLibraryEntryRequest),
        headers: {
          "content-type": "application/json"
        },
        method: "PATCH"
      });

      if (!response.ok) {
        setErrorMessage(await readErrorMessage(response));
        return;
      }

      setSuccessMessage("Saved.");
      router.refresh();
    } catch {
      setErrorMessage("Unable to reach the API right now.");
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <form className="space-y-5" onSubmit={handleSubmit}>
      <div className="grid gap-4 sm:grid-cols-2">
        <label className="block space-y-2">
          <span className="text-sm font-medium text-slate-200">Bucket</span>
          <select
            className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
            onChange={(event) => setBucket(event.target.value as LibraryBucket)}
            value={bucket}
          >
            {libraryBucketOptions.map((value) => (
              <option key={value} value={value}>
                {value}
              </option>
            ))}
          </select>
        </label>

        <label className="block space-y-2">
          <span className="text-sm font-medium text-slate-200">Format</span>
          <select
            className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
            onChange={(event) =>
              setFormat(event.target.value as PhysicalFormat | "")
            }
            value={format}
          >
            <option value="">Unspecified</option>
            {physicalFormatOptions.map((value) => (
              <option key={value} value={value}>
                {value.replaceAll("_", " ")}
              </option>
            ))}
          </select>
        </label>
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        <label className="block space-y-2">
          <span className="text-sm font-medium text-slate-200">Barcode</span>
          <input
            className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
            onChange={(event) => setBarcode(event.target.value)}
            value={barcode}
          />
        </label>

        <label className="block space-y-2">
          <span className="text-sm font-medium text-slate-200">Purchase date</span>
          <input
            className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
            onChange={(event) => setPurchaseDate(event.target.value)}
            type="date"
            value={purchaseDate}
          />
        </label>
      </div>

      <label className="block space-y-2">
        <span className="text-sm font-medium text-slate-200">Tags</span>
        <input
          className="w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
          onChange={(event) => setTags(event.target.value)}
          placeholder="Comma-separated"
          value={tags}
        />
      </label>

      <label className="block space-y-2">
        <span className="text-sm font-medium text-slate-200">Notes</span>
        <textarea
          className="min-h-28 w-full rounded-2xl border border-slate-700 bg-slate-900 px-4 py-3 text-sm text-white outline-none transition focus:border-sky-400"
          onChange={(event) => setNotes(event.target.value)}
          value={notes}
        />
      </label>

      {errorMessage ? (
        <p className="rounded-2xl border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-200">
          {errorMessage}
        </p>
      ) : null}

      {successMessage ? (
        <p className="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-200">
          {successMessage}
        </p>
      ) : null}

      <button
        className="w-full rounded-2xl bg-sky-400 px-4 py-3 text-sm font-semibold text-slate-950 transition hover:bg-sky-300 disabled:cursor-not-allowed disabled:opacity-70"
        disabled={isSubmitting}
        type="submit"
      >
        {isSubmitting ? "Saving..." : "Save changes"}
      </button>
    </form>
  );
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
