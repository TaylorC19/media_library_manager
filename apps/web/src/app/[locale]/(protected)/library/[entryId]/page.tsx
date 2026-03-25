import { notFound } from "next/navigation";
import { getTranslations } from "next-intl/server";
import { DeleteEntryButton } from "../../../../../components/library/delete-entry-button";
import { EntryDetailPanels } from "../../../../../components/library/entry-detail-panels";
import { LibraryEntryForm } from "../../../../../components/library/library-entry-form";
import { getLibraryEntry } from "../../../../../lib/library-api";

interface LibraryEntryPageProps {
  params: Promise<{
    entryId: string;
  }>;
}

export default async function LibraryEntryPage({
  params
}: LibraryEntryPageProps) {
  const { entryId } = await params;
  const item = await getLibraryEntry(entryId);
  const tLibrary = await getTranslations("library.detail");

  if (!item) {
    notFound();
  }

  return (
    <div className="space-y-8">
      <EntryDetailPanels item={item} />

      <div className="grid gap-6 lg:grid-cols-[2fr_1fr]">
        <section className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6">
          <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
            {tLibrary("editHeading")}
          </p>
          <h2 className="mt-3 text-2xl font-semibold text-white">
            {tLibrary("editTitle")}
          </h2>
          <div className="mt-6">
            <LibraryEntryForm entry={item.entry} />
          </div>
        </section>

        <section className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6">
          <p className="text-sm font-medium uppercase tracking-[0.35em] text-sky-300">
            {tLibrary("entryActions")}
          </p>
          <h2 className="mt-3 text-2xl font-semibold text-white">
            {tLibrary("manageTitle")}
          </h2>
          <p className="mt-3 text-sm text-slate-300">
            {tLibrary("manageDescription")}
          </p>
          <div className="mt-6">
            <DeleteEntryButton entryId={item.entry.id} />
          </div>
        </section>
      </div>
    </div>
  );
}
