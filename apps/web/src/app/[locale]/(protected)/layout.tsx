import type { ReactNode } from "react";
import { getTranslations } from "next-intl/server";
import { LogoutButton } from "../../../components/auth/logout-button";
import { LocaleSwitcher } from "../../../components/locale/locale-switcher";
import { Link } from "../../../i18n/navigation";
import { type AppLocale } from "../../../i18n/routing";
import { requireAuth } from "../../../lib/auth";

interface ProtectedLayoutProps {
  children: ReactNode;
  params: Promise<{
    locale: AppLocale;
  }>;
}

export default async function ProtectedLayout({
  children,
  params
}: ProtectedLayoutProps) {
  const { locale } = await params;
  const user = await requireAuth(locale);
  const tCommon = await getTranslations("common");
  const tNav = await getTranslations("nav");

  return (
    <div className="min-h-screen bg-slate-950 text-white">
      <header className="border-b border-slate-800 bg-slate-950/90">
        <div className="mx-auto flex max-w-6xl flex-col gap-6 px-6 py-5 lg:flex-row lg:items-center lg:justify-between">
          <div className="space-y-4">
            <div className="space-y-1">
              <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
                {tCommon("appName")}
              </p>
              <div>
                <p className="text-lg font-semibold text-white">
                  {user.displayName ?? user.username}
                </p>
                <p className="text-sm text-slate-400">@{user.username}</p>
              </div>
            </div>

            <nav className="flex flex-wrap gap-2 text-sm text-slate-300">
              <NavLink href="/">{tNav("dashboard")}</NavLink>
              <NavLink href="/catalog">{tNav("catalog")}</NavLink>
              <NavLink href="/wishlist">{tNav("wishlist")}</NavLink>
              <NavLink href="/search">{tNav("search")}</NavLink>
              <NavLink href="/scan">{tNav("scan")}</NavLink>
              <NavLink href="/settings">{tNav("settings")}</NavLink>
            </nav>
          </div>

          <div className="flex flex-col items-start gap-3 lg:items-end">
            <LocaleSwitcher />
            <LogoutButton />
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-6 py-12">{children}</main>
    </div>
  );
}

interface NavLinkProps {
  children: ReactNode;
  href: "/" | "/catalog" | "/wishlist" | "/search" | "/scan" | "/settings";
}

function NavLink({ children, href }: NavLinkProps) {
  return (
    <Link
      className="rounded-full border border-slate-700 px-3 py-1 transition hover:border-sky-400 hover:text-white"
      href={href}
    >
      {children}
    </Link>
  );
}
