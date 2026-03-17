import type { ReactNode } from "react";
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

          <LogoutButton />
        </div>
      </header>

      <main className="mx-auto max-w-5xl px-6 py-12">{children}</main>
    </div>
  );
}
