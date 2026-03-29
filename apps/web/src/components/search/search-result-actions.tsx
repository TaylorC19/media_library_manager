"use client";

import type {
  ImportMediaRecordResponse,
  LibraryBucket,
  NormalizedSearchResult
} from "@media-library/types";
import { useTranslations } from "next-intl";
import { useState } from "react";
import {
  getLocalizedApiErrorMessageFromResponse
} from "../../i18n/errors";
import { useRouter } from "../../i18n/navigation";
import { browserApiFetch } from "../../lib/api-client";

interface SearchResultActionsProps {
  result: NormalizedSearchResult;
}

export function SearchResultActions({ result }: SearchResultActionsProps) {
  const router = useRouter();
  const tErrors = useTranslations("errors");
  const tSearch = useTranslations("search.resultActions");
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
          mode: "search_result",
          result,
          entry: {
            bucket
          }
        })
      });

      if (!response.ok) {
        setErrorMessage(
          await getLocalizedApiErrorMessageFromResponse(response, tErrors)
        );
        return;
      }

      const payload = (await response.json()) as ImportMediaRecordResponse;
      const entryId = payload.libraryEntry?.entry.id;

      if (!entryId) {
        setErrorMessage(tErrors("importMissingEntry"));
        return;
      }

      router.push(`/library/${entryId}`);
      router.refresh();
    } catch {
      setErrorMessage(tErrors("apiUnavailable"));
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
          {activeBucket === "catalog"
            ? tSearch("adding")
            : tSearch("addToCatalog")}
        </button>
        <button
          className="rounded-2xl border border-slate-700 px-4 py-3 text-sm font-semibold text-slate-100 transition hover:border-sky-400 hover:text-white disabled:cursor-not-allowed disabled:opacity-70"
          disabled={activeBucket !== null}
          onClick={() => handleAdd("wishlist")}
          type="button"
        >
          {activeBucket === "wishlist"
            ? tSearch("adding")
            : tSearch("addToWishlist")}
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

