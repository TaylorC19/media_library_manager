import type { LibraryEntryResponse } from "@media-library/types";
import { useFormatter, useTranslations } from "next-intl";
import {
  formatDateLabel,
  formatDateTimeLabel,
  getBucketLabel,
  getMediaTypeLabel,
  getPhysicalFormatLabel
} from "../../i18n/ui";
import { getMediaCreatorLine } from "../../lib/media-api";

interface EntryDetailPanelsProps {
  item: LibraryEntryResponse;
}

export function EntryDetailPanels({ item }: EntryDetailPanelsProps) {
  const { entry, media } = item;
  const creatorLine = getMediaCreatorLine(media);
  const format = useFormatter();
  const tCommon = useTranslations("common");
  const tBucket = useTranslations("enums.bucket");
  const tLibrary = useTranslations("library.detail");
  const tMediaType = useTranslations("enums.mediaType");
  const tPhysicalFormat = useTranslations("enums.physicalFormat");

  return (
    <div className="grid gap-6 lg:grid-cols-2">
      <section className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6">
        <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
          {tLibrary("myCopy")}
        </p>
        <div className="mt-6 grid gap-4 text-sm text-slate-200">
          <DetailRow label={tLibrary("bucket")} value={getBucketLabel(tBucket, entry.bucket)} />
          <DetailRow
            label={tLibrary("format")}
            value={getPhysicalFormatLabel(tPhysicalFormat, tCommon, entry.format)}
          />
          <DetailRow
            label={tLibrary("barcode")}
            value={entry.barcode ?? tCommon("notSet")}
          />
          <DetailRow
            label={tLibrary("purchaseDate")}
            value={formatDateLabel(format.dateTime, tCommon, entry.purchaseDate)}
          />
          <DetailRow
            label={tLibrary("added")}
            value={formatDateTimeLabel(format.dateTime, entry.createdAt)}
          />
          <DetailRow
            label={tLibrary("updated")}
            value={formatDateTimeLabel(format.dateTime, entry.updatedAt)}
          />
          <div>
            <p className="text-xs uppercase tracking-[0.25em] text-slate-500">
              {tLibrary("tags")}
            </p>
            <div className="mt-2 flex flex-wrap gap-2">
              {entry.tags.length > 0 ? (
                entry.tags.map((tag) => (
                  <span
                    key={tag}
                    className="rounded-full border border-slate-700 px-3 py-1 text-xs text-slate-300"
                  >
                    #{tag}
                  </span>
                ))
              ) : (
                <span className="text-sm text-slate-400">
                  {tCommon("states.noTags")}
                </span>
              )}
            </div>
          </div>
          <div>
            <p className="text-xs uppercase tracking-[0.25em] text-slate-500">
              {tLibrary("notes")}
            </p>
            <p className="mt-2 whitespace-pre-wrap text-sm text-slate-300">
              {entry.notes ?? tCommon("states.noPersonalNotes")}
            </p>
          </div>
        </div>
      </section>

      <section className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6">
        <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
          {tLibrary("mediaDetails")}
        </p>
        <div className="mt-4">
          <p className="text-sm uppercase tracking-[0.25em] text-slate-500">
            {getMediaTypeLabel(tMediaType, media.mediaType)}
          </p>
          <h1 className="mt-2 text-3xl font-semibold text-white">{media.title}</h1>
          {creatorLine ? (
            <p className="mt-2 text-sm text-slate-300">{creatorLine}</p>
          ) : null}
        </div>

        <div className="mt-6 grid gap-4 text-sm text-slate-200">
          <DetailRow
            label={tLibrary("year")}
            value={media.year ? String(media.year) : tCommon("unknown")}
          />
          <DetailRow
            label={tLibrary("releaseDate")}
            value={formatDateLabel(format.dateTime, tCommon, media.releaseDate)}
          />
          <DetailRow
            label={tLibrary("lastSynced")}
            value={formatDateLabel(format.dateTime, tCommon, media.lastSyncedAt)}
          />
          <div>
            <p className="text-xs uppercase tracking-[0.25em] text-slate-500">
              {tLibrary("summary")}
            </p>
            <p className="mt-2 whitespace-pre-wrap text-sm text-slate-300">
              {media.summary ?? tCommon("states.noImportedSummary")}
            </p>
          </div>
          <div>
            <p className="text-xs uppercase tracking-[0.25em] text-slate-500">
              {tLibrary("providerRefs")}
            </p>
            <pre className="mt-2 overflow-x-auto rounded-2xl border border-slate-800 bg-slate-900 p-4 text-xs text-slate-300">
              {JSON.stringify(media.providerRefs, null, 2)}
            </pre>
          </div>
        </div>
      </section>
    </div>
  );
}

interface DetailRowProps {
  label: string;
  value: string;
}

function DetailRow({ label, value }: DetailRowProps) {
  return (
    <div>
      <p className="text-xs uppercase tracking-[0.25em] text-slate-500">{label}</p>
      <p className="mt-1 text-sm text-slate-200">{value}</p>
    </div>
  );
}
