"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { browserApiFetch } from "../../lib/api-client";

interface DeleteEntryButtonProps {
  entryId: string;
}

export function DeleteEntryButton({ entryId }: DeleteEntryButtonProps) {
  const router = useRouter();
  const [isDeleting, setIsDeleting] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  async function handleDelete() {
    const shouldDelete = window.confirm(
      "Delete this library entry? The media record will stay available for future entries."
    );

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
        setErrorMessage("Unable to delete this entry right now.");
        return;
      }

      router.push("/");
      router.refresh();
    } catch {
      setErrorMessage("Unable to reach the API right now.");
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
        {isDeleting ? "Deleting..." : "Delete entry"}
      </button>

      {errorMessage ? (
        <p className="rounded-2xl border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-200">
          {errorMessage}
        </p>
      ) : null}
    </div>
  );
}
