import { notFound } from "next/navigation";
import Link from "next/link";
import { Nav } from "@/components/layout/Nav";
import { DetailSection, OutcomeList, PageIntro } from "@/components/layout/PageIntro";
import { CAPABILITIES, type CapabilitySlug } from "@/lib/constants";
import { capabilityContent } from "@/content/capabilities";

type PageProps = {
  params: Promise<{ slug: string }>;
};

export function generateStaticParams() {
  return CAPABILITIES.map((cap) => ({ slug: cap.slug }));
}

export default async function CapabilityPage({ params }: PageProps) {
  const { slug } = await params;
  if (!CAPABILITIES.some((c) => c.slug === slug)) notFound();

  const content = capabilityContent[slug as CapabilitySlug];

  return (
    <>
      <Nav />
      <main className="mx-auto max-w-4xl px-4 py-16 md:px-6">
        <PageIntro
          back={{ href: "/capabilities", label: "All capabilities" }}
          label="Capability"
          title={content.title}
          description={content.summary}
        >
          <p className="mt-4 text-sm text-[var(--mkt-subtle)]">For {content.whoItsFor.toLowerCase()}</p>
        </PageIntro>

        <DetailSection title="What you get">
          <OutcomeList items={content.benefits} />
        </DetailSection>

        <DetailSection title="How it works">
          <ol className="mt-6 space-y-4">
            {content.steps.map((step, i) => (
              <li key={step} className="flex gap-4">
                <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full border border-[var(--mkt-border-strong)] text-sm font-semibold">
                  {i + 1}
                </span>
                <p className="pt-1 text-[var(--mkt-muted)]">{step}</p>
              </li>
            ))}
          </ol>
        </DetailSection>

        <div className="mt-16 flex flex-wrap gap-4">
          <Link href="/contact" className="mkt-btn mkt-btn-primary">
            Request a demo
          </Link>
          <Link href="/solutions" className="mkt-btn mkt-btn-outline">
            See related solutions
          </Link>
        </div>
      </main>
    </>
  );
}
