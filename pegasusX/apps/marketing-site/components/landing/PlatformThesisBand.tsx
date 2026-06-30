import { platformThesis } from "@/content/landing/enterprise";

export function PlatformThesisBand() {
  return (
    <section className="border-b border-[var(--mkt-border)] bg-[var(--mkt-bg)] py-16 md:py-20">
      <div className="mx-auto max-w-4xl px-4 text-center md:px-6">
        <p className="font-mono text-[10px] uppercase tracking-[0.2em] text-[var(--mkt-muted)]">
          {platformThesis.eyebrow}
        </p>
        <h2 className="mkt-display mt-4 text-3xl md:text-5xl">{platformThesis.headline}</h2>
        <p className="mx-auto mt-6 max-w-3xl text-base leading-relaxed text-[var(--mkt-muted)] md:text-lg">
          {platformThesis.body}
        </p>
      </div>
    </section>
  );
}
