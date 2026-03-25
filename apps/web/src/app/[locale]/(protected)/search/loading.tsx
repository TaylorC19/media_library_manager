export default function SearchLoading() {
  return (
    <div className="space-y-8" aria-busy="true" aria-live="polite">
      <section className="rounded-3xl border border-slate-800 bg-slate-950/70 p-6">
        <div className="h-5 w-24 rounded bg-slate-800" />
        <div className="mt-4 h-10 w-64 rounded bg-slate-900" />
        <div className="mt-6 grid gap-4 md:grid-cols-[1fr_220px_auto]">
          <div className="h-12 rounded-2xl bg-slate-900" />
          <div className="h-12 rounded-2xl bg-slate-900" />
          <div className="h-12 rounded-2xl bg-slate-900" />
        </div>
      </section>

      <div className="grid gap-4">
        {Array.from({ length: 3 }, (_, index) => (
          <section
            key={index}
            className="rounded-3xl border border-slate-800 bg-slate-950/70 p-5"
          >
            <div className="flex flex-col gap-5 sm:flex-row">
              <div className="h-40 w-full rounded-2xl bg-slate-900 sm:w-28" />
              <div className="flex-1 space-y-4">
                <div className="h-5 w-32 rounded bg-slate-800" />
                <div className="h-8 w-2/3 rounded bg-slate-900" />
                <div className="h-4 w-1/2 rounded bg-slate-900" />
                <div className="h-20 rounded bg-slate-900" />
                <div className="flex gap-3">
                  <div className="h-12 w-36 rounded-2xl bg-slate-900" />
                  <div className="h-12 w-36 rounded-2xl bg-slate-900" />
                </div>
              </div>
            </div>
          </section>
        ))}
      </div>
    </div>
  );
}
