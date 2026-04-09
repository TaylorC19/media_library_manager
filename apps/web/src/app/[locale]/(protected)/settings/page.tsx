import { getTranslations } from "next-intl/server";
import { Link } from "../../../../i18n/navigation";

export default async function SettingsPage() {
  const tSettings = await getTranslations("settings");
  const tCommon = await getTranslations("common");
  const tNav = await getTranslations("nav");

  return (
    <div className="space-y-8">
      <section className="rounded-3xl border border-slate-800 bg-slate-950/70 p-8">
        <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
          {tSettings("eyebrow")}
        </p>
        <h1 className="mt-3 text-3xl font-semibold text-white">{tSettings("title")}</h1>
        <p className="mt-3 max-w-3xl text-sm text-slate-300">
          {tSettings("description")}
        </p>
      </section>

      <section className="grid gap-4 lg:grid-cols-2">
        <article className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6">
          <h2 className="text-2xl font-semibold text-white">
            {tSettings("currentStateTitle")}
          </h2>
          <p className="mt-3 text-sm text-slate-300">
            {tSettings("currentStateDescription")}
          </p>
        </article>

        <article className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6">
          <h2 className="text-2xl font-semibold text-white">{tSettings("roadmapTitle")}</h2>
          <p className="mt-3 text-sm text-slate-300">
            {tSettings("roadmapDescription")}
          </p>
        </article>
      </section>

      <section className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6">
        <h2 className="text-2xl font-semibold text-white">{tSettings("navigationTitle")}</h2>
        <p className="mt-3 max-w-3xl text-sm text-slate-300">
          {tSettings("navigationDescription")}
        </p>
        <div className="mt-6 flex flex-wrap gap-3">
          <Link
            className="rounded-2xl border border-slate-700 px-4 py-3 text-sm font-semibold text-slate-100 transition hover:border-sky-400 hover:text-white"
            href="/search"
          >
            {tCommon("actions.search")}
          </Link>
          <Link
            className="rounded-2xl border border-slate-700 px-4 py-3 text-sm font-semibold text-slate-100 transition hover:border-sky-400 hover:text-white"
            href="/scan"
          >
            {tNav("scan")}
          </Link>
          <Link
            className="rounded-2xl border border-slate-700 px-4 py-3 text-sm font-semibold text-slate-100 transition hover:border-sky-400 hover:text-white"
            href="/"
          >
            {tCommon("actions.backToDashboard")}
          </Link>
        </div>
      </section>
    </div>
  );
}
