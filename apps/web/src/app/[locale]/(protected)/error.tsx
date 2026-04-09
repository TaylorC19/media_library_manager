"use client";

import { useTranslations } from "next-intl";
import { Link } from "../../../i18n/navigation";

interface ProtectedRouteErrorProps {
  reset: () => void;
}

export default function ProtectedRouteError({ reset }: ProtectedRouteErrorProps) {
  const tCommon = useTranslations("common");

  return (
    <section className="rounded-3xl border border-red-500/30 bg-red-500/10 p-6">
      <p className="text-sm font-medium uppercase tracking-[0.35em] text-red-200">
        {tCommon("routeState.errorLabel")}
      </p>
      <h1 className="mt-3 text-2xl font-semibold text-white">
        {tCommon("routeState.errorTitle")}
      </h1>
      <p className="mt-2 max-w-2xl text-sm text-red-100">
        {tCommon("routeState.errorDescription")}
      </p>
      <div className="mt-6 flex flex-col gap-3 sm:flex-row">
        <button
          className="rounded-2xl bg-white px-4 py-3 text-sm font-semibold text-slate-950 transition hover:bg-slate-200"
          onClick={() => reset()}
          type="button"
        >
          {tCommon("actions.tryAgain")}
        </button>
        <Link
          className="rounded-2xl border border-red-200/40 px-4 py-3 text-center text-sm font-semibold text-red-50 transition hover:border-red-100 hover:text-white"
          href="/"
        >
          {tCommon("actions.backToDashboard")}
        </Link>
      </div>
    </section>
  );
}
