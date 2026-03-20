import type { LibraryEntryListItem } from "@media-library/types";
import Link from "next/link";
import { formatDateTimeLabel, getMediaCreatorLine, getMediaTypeLabel, getPhysicalFormatLabel } from "../../lib/media-api";

interface LibraryListProps {
  bucketLabel: string;
  items: LibraryEntryListItem[];
}

export function LibraryList({ bucketLabel, items }: LibraryListProps) {
  if (items.length === 0) {
    return (
      <section className="rounded-3xl border border-dashed border-slate-700 bg-slate-950/40 p-8 text-center text-slate-300">
        No entries matched this {bucketLabel.toLowerCase()} view yet.
      </section>
    );
  }

  return (
    <section className="grid gap-4">
      {items.map(({ entry, media }) => {
        const creatorLine = getMediaCreatorLine(media);

        return (
          <Link
            key={entry.id}
            className="block rounded-3xl border border-slate-800 bg-slate-950/70 p-6 transition hover:border-sky-400/50"
            href={`/library/${entry.id}`}
          >
            <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
              <div className="space-y-3">
                <div>
                  <p className="text-sm font-medium uppercase tracking-[0.3em] text-sky-300">
                    {getMediaTypeLabel(media.mediaType)}
                  </p>
                  <h3 className="mt-2 text-2xl font-semibold text-white">
                    {media.title}
                  </h3>
                  {creatorLine ? (
                    <p className="mt-1 text-sm text-slate-300">{creatorLine}</p>
                  ) : null}
                </div>

                <div className="flex flex-wrap gap-2 text-xs text-slate-300">
                  <span className="rounded-full border border-slate-700 px-3 py-1">
                    {entry.bucket}
                  </span>
                  <span className="rounded-full border border-slate-700 px-3 py-1">
                    {getPhysicalFormatLabel(entry.format)}
                  </span>
                  {media.year ? (
                    <span className="rounded-full border border-slate-700 px-3 py-1">
                      {media.year}
                    </span>
                  ) : null}
                  {entry.tags.map((tag) => (
                    <span
                      key={`${entry.id}-${tag}`}
                      className="rounded-full border border-slate-700 px-3 py-1"
                    >
                      #{tag}
                    </span>
                  ))}
                </div>

                {entry.notes ? (
                  <p className="max-w-2xl text-sm text-slate-300">{entry.notes}</p>
                ) : null}
              </div>

              <div className="text-sm text-slate-400">
                Added {formatDateTimeLabel(entry.createdAt)}
              </div>
            </div>
          </Link>
        );
      })}
    </section>
  );
}
