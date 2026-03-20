import type { LibraryEntryResponse } from "@media-library/types";
import { formatDateLabel, formatDateTimeLabel, getMediaCreatorLine, getMediaTypeLabel, getPhysicalFormatLabel } from "../../lib/media-api";

interface EntryDetailPanelsProps {
  item: LibraryEntryResponse;
}

export function EntryDetailPanels({ item }: EntryDetailPanelsProps) {
  const { entry, media } = item;
  const creatorLine = getMediaCreatorLine(media);

  return (
    <div className="grid gap-6 lg:grid-cols-2">
      <section className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6">
        <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
          My copy
        </p>
        <div className="mt-6 grid gap-4 text-sm text-slate-200">
          <DetailRow label="Bucket" value={entry.bucket} />
          <DetailRow label="Format" value={getPhysicalFormatLabel(entry.format)} />
          <DetailRow label="Barcode" value={entry.barcode ?? "Not set"} />
          <DetailRow
            label="Purchase date"
            value={formatDateLabel(entry.purchaseDate)}
          />
          <DetailRow
            label="Added"
            value={formatDateTimeLabel(entry.createdAt)}
          />
          <DetailRow
            label="Updated"
            value={formatDateTimeLabel(entry.updatedAt)}
          />
          <div>
            <p className="text-xs uppercase tracking-[0.25em] text-slate-500">Tags</p>
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
                <span className="text-sm text-slate-400">No tags yet.</span>
              )}
            </div>
          </div>
          <div>
            <p className="text-xs uppercase tracking-[0.25em] text-slate-500">Notes</p>
            <p className="mt-2 whitespace-pre-wrap text-sm text-slate-300">
              {entry.notes ?? "No personal notes yet."}
            </p>
          </div>
        </div>
      </section>

      <section className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6">
        <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
          Media details
        </p>
        <div className="mt-4">
          <p className="text-sm uppercase tracking-[0.25em] text-slate-500">
            {getMediaTypeLabel(media.mediaType)}
          </p>
          <h1 className="mt-2 text-3xl font-semibold text-white">{media.title}</h1>
          {creatorLine ? (
            <p className="mt-2 text-sm text-slate-300">{creatorLine}</p>
          ) : null}
        </div>

        <div className="mt-6 grid gap-4 text-sm text-slate-200">
          <DetailRow label="Year" value={media.year ? String(media.year) : "Unknown"} />
          <DetailRow
            label="Release date"
            value={formatDateLabel(media.releaseDate)}
          />
          <DetailRow
            label="Last synced"
            value={formatDateLabel(media.lastSyncedAt)}
          />
          <div>
            <p className="text-xs uppercase tracking-[0.25em] text-slate-500">Summary</p>
            <p className="mt-2 whitespace-pre-wrap text-sm text-slate-300">
              {media.summary ?? "No imported summary available."}
            </p>
          </div>
          <div>
            <p className="text-xs uppercase tracking-[0.25em] text-slate-500">
              Provider refs
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
