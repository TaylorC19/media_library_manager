import { getTranslations } from "next-intl/server";
import { Link } from "../../i18n/navigation";

export default async function LocaleNotFoundPage() {
  const tCommon = await getTranslations("common");

  return (
    <main className="flex min-h-screen items-center justify-center px-6 py-20">
      <div className="w-full max-w-xl rounded-3xl border border-slate-800 bg-slate-950/70 p-8 text-center">
        <h1 className="text-3xl font-semibold text-white">
          {tCommon("notFound.title")}
        </h1>
        <p className="mt-3 text-sm text-slate-300">
          {tCommon("notFound.description")}
        </p>
        <div className="mt-6">
          <Link
            className="inline-flex rounded-2xl bg-sky-400 px-4 py-3 text-sm font-semibold text-slate-950 transition hover:bg-sky-300"
            href="/"
          >
            {tCommon("actions.backToDashboard")}
          </Link>
        </div>
      </div>
    </main>
  );
}
