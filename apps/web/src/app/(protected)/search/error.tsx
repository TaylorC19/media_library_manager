"use client";

interface SearchErrorPageProps {
  error: Error & { digest?: string };
  reset: () => void;
}

export default function SearchErrorPage({ error, reset }: SearchErrorPageProps) {
  return (
    <div className="rounded-3xl border border-red-500/30 bg-red-500/10 p-6">
      <p className="text-sm font-medium uppercase tracking-[0.35em] text-red-200">
        Search error
      </p>
      <h1 className="mt-3 text-2xl font-semibold text-white">
        The search page could not load
      </h1>
      <p className="mt-2 text-sm text-red-100">
        {error.message || "Something went wrong while loading provider results."}
      </p>
      <button
        className="mt-6 rounded-2xl bg-white px-4 py-3 text-sm font-semibold text-slate-950 transition hover:bg-slate-200"
        onClick={() => reset()}
        type="button"
      >
        Try again
      </button>
    </div>
  );
}
