import Link from "next/link";
import { requireAuth } from "../../lib/auth";
import { getLibraryEntries } from "../../lib/library-api";
import { formatDateTimeLabel, getMediaTypeLabel } from "../../lib/media-api";

export default async function HomePage() {
  const user = await requireAuth();
  const allEntries = await getLibraryEntries({
    page: 1,
    pageSize: 1000
  });
  const recentEntries = await getLibraryEntries({
    page: 1,
    pageSize: 5
  });
  const mediaTypeCounts = allEntries.items.reduce<Record<string, number>>(
    (counts, item) => {
      counts[item.media.mediaType] = (counts[item.media.mediaType] ?? 0) + 1;
      return counts;
    },
    {}
  );

  return (
    <div className="space-y-8">
      <section className="space-y-4">
        <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
          Dashboard
        </p>
        <div className="space-y-4">
          <h1 className="text-4xl font-semibold tracking-tight text-white sm:text-6xl">
            Welcome, {user.displayName ?? user.username}.
          </h1>
          <p className="max-w-2xl text-lg text-slate-300">
            Your private media library is ready. Jump into cataloging, browse recent
            additions, or start a wishlist for future pickups.
          </p>
        </div>
      </section>

      <section className="grid gap-4 md:grid-cols-3">
        <Link
          className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6 transition hover:border-sky-400/50"
          href="/catalog"
        >
          <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
            Quick nav
          </p>
          <h2 className="mt-3 text-2xl font-semibold text-white">Catalog</h2>
          <p className="mt-2 text-sm text-slate-300">
            Manage the items you already own.
          </p>
        </Link>
        <Link
          className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6 transition hover:border-sky-400/50"
          href="/wishlist"
        >
          <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
            Quick nav
          </p>
          <h2 className="mt-3 text-2xl font-semibold text-white">Wishlist</h2>
          <p className="mt-2 text-sm text-slate-300">
            Track future additions without mixing them into your owned collection.
          </p>
        </Link>
        <div className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6">
          <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
            Total entries
          </p>
          <h2 className="mt-3 text-4xl font-semibold text-white">
            {allEntries.pagination.total}
          </h2>
          <p className="mt-2 text-sm text-slate-300">
            Across catalog and wishlist.
          </p>
        </div>
      </section>

      <section className="grid gap-4 lg:grid-cols-[1.2fr_1fr]">
        <article className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6">
          <h2 className="text-2xl font-semibold text-white">Counts by media type</h2>
          <div className="mt-5 grid gap-3 sm:grid-cols-2">
            {Object.entries(mediaTypeCounts).length > 0 ? (
              Object.entries(mediaTypeCounts).map(([mediaType, count]) => (
                <div
                  key={mediaType}
                  className="rounded-2xl border border-slate-800 bg-slate-900 px-4 py-4"
                >
                  <p className="text-sm uppercase tracking-[0.25em] text-slate-500">
                    {getMediaTypeLabel(mediaType as "movie" | "tv" | "album" | "book" | "game")}
                  </p>
                  <p className="mt-2 text-2xl font-semibold text-white">{count}</p>
                </div>
              ))
            ) : (
              <p className="text-sm text-slate-300">
                Add your first entry to start seeing counts here.
              </p>
            )}
          </div>
        </article>

        <article className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6">
          <h2 className="text-2xl font-semibold text-white">Recent additions</h2>
          <div className="mt-5 space-y-3">
            {recentEntries.items.length > 0 ? (
              recentEntries.items.map(({ entry, media }) => (
                <Link
                  key={entry.id}
                  className="block rounded-2xl border border-slate-800 bg-slate-900 px-4 py-4 transition hover:border-sky-400/50"
                  href={`/library/${entry.id}`}
                >
                  <div className="flex items-start justify-between gap-4">
                    <div>
                      <p className="text-sm font-medium text-white">{media.title}</p>
                      <p className="mt-1 text-xs uppercase tracking-[0.25em] text-slate-500">
                        {entry.bucket} · {getMediaTypeLabel(media.mediaType)}
                      </p>
                    </div>
                    <p className="text-xs text-slate-400">
                      {formatDateTimeLabel(entry.createdAt)}
                    </p>
                  </div>
                </Link>
              ))
            ) : (
              <p className="text-sm text-slate-300">
                Your recent additions will show up here once you save something.
              </p>
            )}
          </div>
        </article>
      </section>
    </div>
  );
}
