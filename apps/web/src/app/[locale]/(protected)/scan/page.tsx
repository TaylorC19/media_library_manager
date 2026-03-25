import { getTranslations } from "next-intl/server";

export default async function ScanPlaceholderPage() {
  const tScan = await getTranslations("scan");

  return (
    <section className="rounded-3xl border border-slate-800 bg-slate-950/70 p-8">
      <h1 className="text-3xl font-semibold text-white">{tScan("title")}</h1>
      <p className="mt-3 max-w-2xl text-sm text-slate-300">
        {tScan("description")}
      </p>
    </section>
  );
}
