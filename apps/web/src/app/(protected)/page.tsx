import { requireAuth } from "../../lib/auth";

const mediaTypes = ["movie", "tv", "album", "book", "game"] as const;

export default async function HomePage() {
  const user = await requireAuth();

  return (
    <div className="space-y-10">
      <section className="space-y-6">
        <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
          Private dashboard
        </p>
        <div className="space-y-4">
          <h1 className="text-4xl font-semibold tracking-tight text-white sm:text-6xl">
            Welcome, {user.displayName ?? user.username}.
          </h1>
          <p className="max-w-2xl text-lg text-slate-300">
            Your session is active and the app shell is now protected by
            default. Catalog, wishlist, search, and scanning routes can build on
            this authenticated layout.
          </p>
        </div>
      </section>

      <section className="grid gap-4 sm:grid-cols-2">
        <article className="rounded-2xl border border-slate-800 bg-slate-950/70 p-6">
          <h2 className="text-lg font-semibold text-white">Planned buckets</h2>
          <p className="mt-3 text-sm text-slate-300">
            v1 keeps the collection model intentionally focused with exactly two
            buckets: catalog and wishlist.
          </p>
        </article>
        <article className="rounded-2xl border border-slate-800 bg-slate-950/70 p-6">
          <h2 className="text-lg font-semibold text-white">Supported media</h2>
          <ul className="mt-3 flex flex-wrap gap-2 text-sm text-slate-200">
            {mediaTypes.map((mediaType) => (
              <li
                key={mediaType}
                className="rounded-full border border-slate-700 px-3 py-1"
              >
                {mediaType}
              </li>
            ))}
          </ul>
        </article>
      </section>
    </div>
  );
}
