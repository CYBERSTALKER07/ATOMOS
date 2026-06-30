import { notFound } from "next/navigation";
import Link from "next/link";
import { Nav } from "@/components/layout/Nav";
import { ComponentPreview } from "@/components/docs/ComponentPreview";
import { PropsTable } from "@/components/docs/PropsTable";
import { CodeBlock } from "@/components/docs/CodeBlock";
import { MotionSpec } from "@/components/docs/MotionSpec";
import { COMPONENT_PREVIEW_MAP } from "@/components/docs/preview-map";
import { getComponentDoc } from "@/content/components/registry";

type PageProps = {
  params: Promise<{ slug: string }>;
};

export function generateStaticParams() {
  return [
    "motion-tokens",
    "portal-button",
    "pulse-timeline",
    "portal-card",
    "page-chrome",
    "explain-banner",
    "kpi-stat-card",
    "fleet-route-map",
    "topology-graph",
    "status-chip",
    "scroll-section",
    "role-badge",
  ].map((slug) => ({ slug }));
}

export default async function ComponentDocPage({ params }: PageProps) {
  const { slug } = await params;
  const doc = getComponentDoc(slug);
  if (!doc) notFound();

  const Preview = COMPONENT_PREVIEW_MAP[slug];

  return (
    <>
      <Nav />
      <main className="mx-auto max-w-4xl px-4 py-16 md:px-6">
        <Link href="/components" className="text-sm text-[var(--mkt-muted)] hover:text-[var(--mkt-text)]">
          ← Components
        </Link>
        <h1 className="mkt-display mt-6 text-4xl">{doc.title}</h1>
        <p className="mt-4 text-[var(--mkt-muted)]">{doc.description}</p>

        <section className="mt-12">
          <h2 className="text-lg font-semibold">Preview</h2>
          <div className="mt-4">
            {Preview ? (
              <ComponentPreview>
                <Preview />
              </ComponentPreview>
            ) : (
              <p className="text-sm text-[var(--mkt-text-tertiary)]">Preview unavailable.</p>
            )}
          </div>
        </section>

        <section className="mt-12">
          <h2 className="text-lg font-semibold">Props</h2>
          <div className="mt-4">
            <PropsTable props={doc.props} />
          </div>
        </section>

        <section className="mt-12">
          <MotionSpec spec={doc.motionSpec} />
        </section>

        <section className="mt-12">
          <h2 className="text-lg font-semibold">Usage</h2>
          <div className="mt-4">
            <CodeBlock code={doc.snippet} />
          </div>
        </section>

        <section className="mt-12">
          <h2 className="text-lg font-semibold">Used in</h2>
          <ul className="mt-4 space-y-2 text-sm text-[var(--mkt-text-secondary)]">
            {doc.usedIn.map((item) => (
              <li key={`${item.role}-${item.surface}`}>
                <span className="font-medium text-white">{item.role}</span> — {item.surface}
              </li>
            ))}
          </ul>
        </section>
      </main>
    </>
  );
}
