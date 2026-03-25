"use client";

import { useTranslations } from "next-intl";
import { useState } from "react";
import { useRouter } from "../../i18n/navigation";
import { browserApiFetch } from "../../lib/api-client";

interface DeleteEntryButtonProps {
  entryId: string;
}

export function DeleteEntryButton({ entryId }: DeleteEntryButtonProps) {
  const router = useRouter();
  const tCommon = useTranslations("common");
  const tErrors = useTranslations("errors");
  const tLibrary = useTranslations("library");
  const [isDeleting, setIsDeleting] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  async function handleDelete() {
    const shouldDelete = window.confirm(tLibrary("confirmDelete"));

    if (!shouldDelete) {
      return;
    }

    setErrorMessage(null);
    setIsDeleting(true);

    try {
      const response = await browserApiFetch(`/library/${entryId}`, {
        method: "DELETE"
      });

      if (!response.ok) {
        setErrorMessage(tErrors("deleteFailed"));
        return;
      }

      router.push("/");
      router.refresh();
    } catch {
      setErrorMessage(tErrors("apiUnavailable"));
    } finally {
      setIsDeleting(false);
    }
  }

  return (
    <div className="space-y-3">
      <button
        className="w-full rounded-2xl border border-red-500/40 px-4 py-3 text-sm font-semibold text-red-200 transition hover:bg-red-500/10 disabled:cursor-not-allowed disabled:opacity-70"
        disabled={isDeleting}
        onClick={handleDelete}
        type="button"
      >
        {isDeleting ? tCommon("actions.deleting") : tCommon("actions.delete")}
      </button>

      {errorMessage ? (
        <p className="rounded-2xl border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-200">
          {errorMessage}
        </p>
      ) : null}
    </div>
  );
}
