import type { MediaType } from "@media-library/types";
import { getFormatter, getTranslations } from "next-intl/server";
import { Link } from "../../../i18n/navigation";
import { formatDateTimeLabel, getBucketLabel, getMediaTypeLabel } from "../../../i18n/ui";
import { type AppLocale } from "../../../i18n/routing";
import { requireAuth } from "../../../lib/auth";
import { getLibraryEntries } from "../../../lib/library-api";

interface HomePageProps {
  params: Promise<{
    locale: AppLocale;
  }>;
}

export default async function HomePage({ params }: HomePageProps) {
  const { locale } = await params;
  const user = await requireAuth(locale);
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
  const mediaTypeCountEntries = Object.entries(mediaTypeCounts) as Array<
    [MediaType, number]
  >;
  const format = await getFormatter();
  const tBucket = await getTranslations("enums.bucket");
  const tDashboard = await getTranslations("dashboard");
  const tMediaType = await getTranslations("enums.mediaType");

  return (
    <div className="space-y-8">
      <section className="space-y-4">
        <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
          {tDashboard("sectionLabel")}
        </p>
        <div className="space-y-4">
          <h1 className="text-4xl font-semibold tracking-tight text-white sm:text-6xl">
            {tDashboard("welcome", {
              name: user.displayName ?? user.username
            })}
          </h1>
          <p className="max-w-2xl text-lg text-slate-300">
            {tDashboard("description")}
          </p>
        </div>
      </section>

      <section className="grid gap-4 md:grid-cols-3">
        <DashboardLink
          description={tDashboard("cards.catalogDescription")}
          href="/catalog"
          label={tDashboard("quickNav")}
          title={tDashboard("cards.catalogTitle")}
        />
        <DashboardLink
          description={tDashboard("cards.wishlistDescription")}
          href="/wishlist"
          label={tDashboard("quickNav")}
          title={tDashboard("cards.wishlistTitle")}
        />
        <div className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6">
          <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
            {tDashboard("totalEntries")}
          </p>
          <h2 className="mt-3 text-4xl font-semibold text-white">
            {allEntries.pagination.total}
          </h2>
          <p className="mt-2 text-sm text-slate-300">
            {tDashboard("totalEntriesDescription")}
          </p>
        </div>
      </section>

      <section className="grid gap-4 lg:grid-cols-[1.2fr_1fr]">
        <article className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6">
          <h2 className="text-2xl font-semibold text-white">
            {tDashboard("countsByMediaType")}
          </h2>
          <div className="mt-5 grid gap-3 sm:grid-cols-2">
            {mediaTypeCountEntries.length > 0 ? (
              mediaTypeCountEntries.map(([mediaType, count]) => (
                <div
                  key={mediaType}
                  className="rounded-2xl border border-slate-800 bg-slate-900 px-4 py-4"
                >
                  <p className="text-sm uppercase tracking-[0.25em] text-slate-500">
                    {getMediaTypeLabel(tMediaType, mediaType)}
                  </p>
                  <p className="mt-2 text-2xl font-semibold text-white">{count}</p>
                </div>
              ))
            ) : (
              <p className="text-sm text-slate-300">{tDashboard("emptyCounts")}</p>
            )}
          </div>
        </article>

        <article className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6">
          <h2 className="text-2xl font-semibold text-white">
            {tDashboard("recentAdditions")}
          </h2>
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
                        {getBucketLabel(tBucket, entry.bucket)} ·{" "}
                        {getMediaTypeLabel(tMediaType, media.mediaType)}
                      </p>
                    </div>
                    <p className="text-xs text-slate-400">
                      {formatDateTimeLabel(format.dateTime, entry.createdAt)}
                    </p>
                  </div>
                </Link>
              ))
            ) : (
              <p className="text-sm text-slate-300">{tDashboard("emptyRecent")}</p>
            )}
          </div>
        </article>
      </section>
    </div>
  );
}

interface DashboardLinkProps {
  description: string;
  href: "/catalog" | "/wishlist";
  label: string;
  title: string;
}

function DashboardLink({ description, href, label, title }: DashboardLinkProps) {
  return (
    <Link
      className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6 transition hover:border-sky-400/50"
      href={href}
    >
      <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
        {label}
      </p>
      <h2 className="mt-3 text-2xl font-semibold text-white">{title}</h2>
      <p className="mt-2 text-sm text-slate-300">{description}</p>
    </Link>
  );
}
