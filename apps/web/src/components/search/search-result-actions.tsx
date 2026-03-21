"use client";

import type {
  ImportMediaRecordResponse,
  LibraryBucket,
  MediaType,
  ProviderName
} from "@media-library/types";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { browserApiFetch } from "../../lib/api-client";

interface SearchResultActionsProps {
  mediaType: MediaType;
  provider: ProviderName;
  providerId: string;
}

export function SearchResultActions({
  mediaType,
  provider,
  providerId
}: SearchResultActionsProps) {
  const router = useRouter();
  const [activeBucket, setActiveBucket] = useState<LibraryBucket | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  async function handleAdd(bucket: LibraryBucket) {
    setActiveBucket(bucket);
    setErrorMessage(null);

    try {
      const response = await browserApiFetch("/media/import", {
        method: "POST",
        headers: {
          "content-type": "application/json"
        },
        body: JSON.stringify({
          provider,
          providerId,
          mediaType,
          bucket
        })
      });

      if (!response.ok) {
        setErrorMessage(await readErrorMessage(response));
        return;
      }

      const payload = (await response.json()) as ImportMediaRecordResponse;
      const entryId = payload.libraryEntry?.entry.id;

      if (!entryId) {
        setErrorMessage("The item was imported, but no library entry was created.");
        return;
      }

      router.push(`/library/${entryId}`);
      router.refresh();
    } catch {
      setErrorMessage("Unable to reach the API right now.");
    } finally {
      setActiveBucket(null);
    }
  }

  return (
    <div className="space-y-3">
      <div className="flex flex-col gap-3 sm:flex-row">
        <button
          className="rounded-2xl bg-sky-400 px-4 py-3 text-sm font-semibold text-slate-950 transition hover:bg-sky-300 disabled:cursor-not-allowed disabled:opacity-70"
          disabled={activeBucket !== null}
          onClick={() => handleAdd("catalog")}
          type="button"
        >
          {activeBucket === "catalog" ? "Adding..." : "Add to catalog"}
        </button>
        <button
          className="rounded-2xl border border-slate-700 px-4 py-3 text-sm font-semibold text-slate-100 transition hover:border-sky-400 hover:text-white disabled:cursor-not-allowed disabled:opacity-70"
          disabled={activeBucket !== null}
          onClick={() => handleAdd("wishlist")}
          type="button"
        >
          {activeBucket === "wishlist" ? "Adding..." : "Add to wishlist"}
        </button>
      </div>

      {errorMessage ? (
        <p className="rounded-2xl border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-200">
          {errorMessage}
        </p>
      ) : null}
    </div>
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
