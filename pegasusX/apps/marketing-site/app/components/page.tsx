import Link from "next/link";
import { Nav } from "@/components/layout/Nav";
import { COMPONENT_DOCS } from "@/content/components/registry";

export default function ComponentsIndexPage() {
  return (
    <>
      <Nav />
      <main className="mx-auto max-w-7xl px-4 py-16 md:px-6">
        <p className="mkt-section-label">Component library</p>
        <h1 className="mkt-display mt-3 text-4xl md:text-5xl">Portal patterns, live previews</h1>
        <p className="mt-4 max-w-2xl text-[var(--mkt-muted)]">
          Marketing-wrapped documentation for shared Pegasus UI contracts.
        </p>
        <div className="mt-12 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {COMPONENT_DOCS.map((doc) => (
            <Link
              key={doc.slug}
              href={`/components/${doc.slug}`}
              className="mkt-card block p-6 transition hover:border-[var(--mkt-border-strong)]"
            >
              <h2 className="font-semibold">{doc.title}</h2>
              <p className="mt-2 text-sm text-[var(--mkt-text-secondary)]">{doc.description}</p>
            </Link>
          ))}
        </div>
      </main>
    </>
  );
}
