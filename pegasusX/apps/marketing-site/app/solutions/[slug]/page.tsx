import { notFound } from "next/navigation";
import Link from "next/link";
import { Nav } from "@/components/layout/Nav";
import { DetailSection, OutcomeList, PageIntro, RelatedLinks } from "@/components/layout/PageIntro";
import { SOLUTIONS, type SolutionSlug } from "@/lib/constants";
import { solutionContent } from "@/content/solutions";

type PageProps = {
  params: Promise<{ slug: string }>;
};

export function generateStaticParams() {
  return SOLUTIONS.map((s) => ({ slug: s.slug }));
}

export default async function SolutionDetailPage({ params }: PageProps) {
  const { slug } = await params;
  if (!SOLUTIONS.some((s) => s.slug === slug)) notFound();

  const content = solutionContent[slug as SolutionSlug];

  return (
    <>
      <Nav />
      <main className="mx-auto max-w-4xl px-4 py-16 md:px-6">
        <PageIntro
          back={{ href: "/solutions", label: "All solutions" }}
          label="Solution"
          title={content.title}
          description={content.summary}
        />

        <DetailSection title="The problem">
          <p className="mt-4 text-[var(--mkt-muted)]">{content.problem}</p>
        </DetailSection>

        <DetailSection title="What changes">
          <OutcomeList items={content.outcomes} />
        </DetailSection>

        <DetailSection title="How it works">
          <div className="mt-6 space-y-6">
            {content.howItWorks.map((step, i) => (
              <div key={step.title} className="mkt-card p-6">
                <p className="text-xs font-mono uppercase tracking-wider text-[var(--mkt-subtle)]">
                  Step {i + 1}
                </p>
                <h3 className="mt-2 font-semibold">{step.title}</h3>
                <p className="mt-2 text-sm text-[var(--mkt-muted)]">{step.description}</p>
              </div>
            ))}
          </div>
        </DetailSection>

        <DetailSection title="Related capabilities">
          <RelatedLinks capabilities={content.relatedCapabilities} />
        </DetailSection>

        <DetailSection title="Who benefits">
          <RelatedLinks roles={content.relatedRoles} />
        </DetailSection>

        <div className="mt-16 flex flex-wrap gap-4">
          <Link href="/contact" className="mkt-btn mkt-btn-primary">
            Request a demo
          </Link>
          <Link href="/platform" className="mkt-btn mkt-btn-outline">
            See the platform
          </Link>
        </div>
      </main>
    </>
  );
}
