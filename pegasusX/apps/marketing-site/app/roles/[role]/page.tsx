import { notFound } from "next/navigation";
import Link from "next/link";
import { Nav } from "@/components/layout/Nav";
import { DetailSection, OutcomeList, PageIntro, RelatedLinks } from "@/components/layout/PageIntro";
import { ROLES, type RoleSlug } from "@/lib/constants";
import { roleContent } from "@/content/roles";

type PageProps = {
  params: Promise<{ role: string }>;
};

export function generateStaticParams() {
  return ROLES.map((role) => ({ role: role.slug }));
}

export default async function RoleDetailPage({ params }: PageProps) {
  const { role: roleSlug } = await params;
  if (!ROLES.some((r) => r.slug === roleSlug)) notFound();

  const role = ROLES.find((r) => r.slug === roleSlug)!;
  const content = roleContent[roleSlug as RoleSlug];

  return (
    <>
      <Nav />
      <main className="mx-auto max-w-4xl px-4 py-16 md:px-6">
        <PageIntro
          back={{ href: "/roles", label: "All roles" }}
          label={role.name}
          title={content.headline}
          description={content.summary}
        >
          <p className="mt-4 text-sm text-[var(--mkt-subtle)]">
            Built for {content.persona.toLowerCase()}
          </p>
        </PageIntro>

        <DetailSection title="Apps available">
          <ul className="mt-3 flex flex-wrap gap-2">
            {role.surfaces.map((surface) => (
              <li key={surface} className="role-badge">{surface}</li>
            ))}
          </ul>
        </DetailSection>

        <DetailSection title="Common challenges">
          <OutcomeList items={content.painPoints} />
        </DetailSection>

        <DetailSection title="What changes">
          <OutcomeList items={content.outcomes} />
        </DetailSection>

        <DetailSection title="Key features">
          <ul className="mt-4 space-y-2">
            {content.bullets.map((bullet) => (
              <li key={bullet} className="flex gap-2 text-sm text-[var(--mkt-muted)]">
                <span className="text-[var(--mkt-subtle)]">→</span>
                {bullet}
              </li>
            ))}
          </ul>
        </DetailSection>

        <DetailSection title="A day in the workflow">
          <div className="mt-6 space-y-6">
            {content.workflows.map((workflow) => (
              <div key={workflow.title} className="mkt-card p-6">
                <h3 className="font-semibold">{workflow.title}</h3>
                <ol className="mt-4 list-decimal space-y-2 pl-5 text-sm text-[var(--mkt-muted)]">
                  {workflow.steps.map((step) => (
                    <li key={step}>{step}</li>
                  ))}
                </ol>
              </div>
            ))}
          </div>
        </DetailSection>

        <DetailSection title="Works with">
          <div className="mt-4 overflow-x-auto rounded-xl border border-[var(--mkt-border)]">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-[var(--mkt-border)] text-[var(--mkt-subtle)]">
                  <th className="px-4 py-3">Team</th>
                  <th className="px-4 py-3">How you connect</th>
                </tr>
              </thead>
              <tbody>
                {content.crossRole.map((row) => (
                  <tr key={row.role} className="border-b border-[var(--mkt-border)] last:border-0">
                    <td className="px-4 py-3 font-medium">{row.role}</td>
                    <td className="px-4 py-3 text-[var(--mkt-muted)]">{row.touchpoint}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </DetailSection>

        <DetailSection title="Related capabilities">
          <RelatedLinks capabilities={content.capabilityLinks} />
        </DetailSection>

        <div className="mt-16 flex flex-wrap gap-4">
          <Link href="/contact" className="mkt-btn mkt-btn-primary">
            Request a demo
          </Link>
          <Link href="/solutions" className="mkt-btn mkt-btn-outline">
            See solutions
          </Link>
        </div>
      </main>
    </>
  );
}
