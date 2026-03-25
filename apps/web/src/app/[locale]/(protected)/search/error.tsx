"use client";

import { useTranslations } from "next-intl";

interface SearchErrorPageProps {
  error: Error & { digest?: string };
  reset: () => void;
}

export default function SearchErrorPage({ error, reset }: SearchErrorPageProps) {
  const tCommon = useTranslations("common");
  const tErrors = useTranslations("errors");
  const tSearch = useTranslations("search");
  const description = error.message ? tErrors("searchLoadFailed") : tSearch("errorDescription");

  return (
    <div className="rounded-3xl border border-red-500/30 bg-red-500/10 p-6">
      <p className="text-sm font-medium uppercase tracking-[0.35em] text-red-200">
        {tSearch("errorLabel")}
      </p>
      <h1 className="mt-3 text-2xl font-semibold text-white">
        {tSearch("errorTitle")}
      </h1>
      <p className="mt-2 text-sm text-red-100">{description}</p>
      <button
        className="mt-6 rounded-2xl bg-white px-4 py-3 text-sm font-semibold text-slate-950 transition hover:bg-slate-200"
        onClick={() => reset()}
        type="button"
      >
        {tCommon("actions.tryAgain")}
      </button>
    </div>
  );
}
