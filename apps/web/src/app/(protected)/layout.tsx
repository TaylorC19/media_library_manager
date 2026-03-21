import type { ReactNode } from "react";
import Link from "next/link";
import { LogoutButton } from "../../components/auth/logout-button";
import { requireAuth } from "../../lib/auth";

type ProtectedLayoutProps = Readonly<{
  children: ReactNode;
}>;

export default async function ProtectedLayout({
  children
}: ProtectedLayoutProps) {
  const user = await requireAuth();

  return (
    <div className="min-h-screen bg-slate-950 text-white">
      <header className="border-b border-slate-800 bg-slate-950/90">
        <div className="mx-auto flex max-w-5xl items-center justify-between gap-6 px-6 py-5">
          <div className="space-y-4">
            <div className="space-y-1">
              <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
                Media Library Manager
              </p>
              <div>
                <p className="text-lg font-semibold text-white">
                  {user.displayName ?? user.username}
                </p>
                <p className="text-sm text-slate-400">@{user.username}</p>
              </div>
            </div>

            <nav className="flex flex-wrap gap-2 text-sm text-slate-300">
              <Link
                className="rounded-full border border-slate-700 px-3 py-1 transition hover:border-sky-400 hover:text-white"
                href="/"
              >
                Dashboard
              </Link>
              <Link
                className="rounded-full border border-slate-700 px-3 py-1 transition hover:border-sky-400 hover:text-white"
                href="/catalog"
              >
                Catalog
              </Link>
              <Link
                className="rounded-full border border-slate-700 px-3 py-1 transition hover:border-sky-400 hover:text-white"
                href="/wishlist"
              >
                Wishlist
              </Link>
              <Link
                className="rounded-full border border-slate-700 px-3 py-1 transition hover:border-sky-400 hover:text-white"
                href="/search"
              >
                Search
              </Link>
            </nav>
          </div>

          <LogoutButton />
        </div>
      </header>

      <main className="mx-auto max-w-5xl px-6 py-12">{children}</main>
    </div>
  );
}
