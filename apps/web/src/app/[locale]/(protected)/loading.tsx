import { getTranslations } from "next-intl/server";

export default async function ProtectedRouteLoading() {
  const tCommon = await getTranslations("common");

  return (
    <div className="space-y-8" aria-busy="true" aria-live="polite">
      <section className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6">
        <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
          {tCommon("routeState.loadingLabel")}
        </p>
        <h1 className="mt-3 text-3xl font-semibold text-white">
          {tCommon("routeState.loadingTitle")}
        </h1>
        <p className="mt-3 max-w-2xl text-sm text-slate-300">
          {tCommon("routeState.loadingDescription")}
        </p>
      </section>

      <section className="grid gap-4 md:grid-cols-3">
        <div className="h-40 rounded-3xl border border-slate-800 bg-slate-950/70" />
        <div className="h-40 rounded-3xl border border-slate-800 bg-slate-950/70" />
        <div className="h-40 rounded-3xl border border-slate-800 bg-slate-950/70" />
      </section>

      <section className="grid gap-4">
        {Array.from({ length: 3 }, (_, index) => (
          <div
            key={index}
            className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6"
          >
            <div className="h-5 w-28 rounded bg-slate-800" />
            <div className="mt-4 h-8 w-1/3 rounded bg-slate-900" />
            <div className="mt-6 h-20 rounded-2xl bg-slate-900" />
          </div>
        ))}
      </section>
    </div>
  );
}
