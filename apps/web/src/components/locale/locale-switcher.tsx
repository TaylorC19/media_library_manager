"use client";

import { useLocale, useTranslations } from "next-intl";
import { useSearchParams } from "next/navigation";
import { useTransition } from "react";
import { usePathname, useRouter } from "../../i18n/navigation";
import { routing, type AppLocale } from "../../i18n/routing";

export function LocaleSwitcher() {
  const locale = useLocale() as AppLocale;
  const pathname = usePathname();
  const router = useRouter();
  const searchParams = useSearchParams();
  const t = useTranslations("nav");
  const tLocale = useTranslations("nav.locale");
  const [isPending, startTransition] = useTransition();

  function handleLocaleChange(nextLocale: AppLocale) {
    const query = Object.fromEntries(searchParams.entries());

    startTransition(() => {
      router.replace(
        {
          pathname,
          query
        },
        { locale: nextLocale }
      );
    });
  }

  return (
    <label className="flex items-center gap-3 text-sm text-slate-300">
      <span>{t("languageLabel")}</span>
      <select
        className="rounded-full border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-white outline-none transition focus:border-sky-400"
        disabled={isPending}
        onChange={(event) => handleLocaleChange(event.target.value as AppLocale)}
        value={locale}
      >
        {routing.locales.map((item) => (
          <option key={item} value={item}>
            {tLocale(item)}
          </option>
        ))}
      </select>
    </label>
  );
}
