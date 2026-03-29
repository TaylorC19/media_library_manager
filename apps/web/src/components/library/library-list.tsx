import type { LibraryEntryListItem } from "@media-library/types";
import { useFormatter, useTranslations } from "next-intl";
import { Link } from "../../i18n/navigation";
import {
  formatDateTimeLabel,
  getBucketLabel,
  getMediaTypeLabel,
  getPhysicalFormatLabel
} from "../../i18n/ui";
import { getMediaCreatorLine } from "../../lib/media-api";

interface LibraryListProps {
  emptyMessage: string;
  items: LibraryEntryListItem[];
}

export function LibraryList({
  emptyMessage,
  items
}: LibraryListProps) {
  const format = useFormatter();
  const tBucket = useTranslations("enums.bucket");
  const tCommon = useTranslations("common");
  const tLibrary = useTranslations("library.detail");
  const tMediaType = useTranslations("enums.mediaType");
  const tPhysicalFormat = useTranslations("enums.physicalFormat");

  if (items.length === 0) {
    return (
      <section className="rounded-3xl border border-dashed border-slate-700 bg-slate-950/40 p-8 text-center text-slate-300">
        {emptyMessage}
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
              <div className="flex gap-4">
                <div className="flex h-28 w-20 shrink-0 items-center justify-center overflow-hidden rounded-2xl border border-slate-800 bg-slate-900 text-xs text-slate-500">
                  {media.imageUrl ? (
                    <img
                      alt={media.title}
                      className="h-full w-full object-cover"
                      src={media.imageUrl}
                    />
                  ) : (
                    <span>{tCommon("states.noImage")}</span>
                  )}
                </div>

                <div className="space-y-3">
                  <div>
                    <p className="text-sm font-medium uppercase tracking-[0.3em] text-sky-300">
                      {getMediaTypeLabel(tMediaType, media.mediaType)}
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
                      {getBucketLabel(tBucket, entry.bucket)}
                    </span>
                    <span className="rounded-full border border-slate-700 px-3 py-1">
                      {getPhysicalFormatLabel(tPhysicalFormat, tCommon, entry.format)}
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
              </div>

              <div className="text-sm text-slate-400">
                {tLibrary("added")}{" "}
                {formatDateTimeLabel(format.dateTime, entry.createdAt)}
              </div>
            </div>
          </Link>
        );
      })}
    </section>
  );
}
