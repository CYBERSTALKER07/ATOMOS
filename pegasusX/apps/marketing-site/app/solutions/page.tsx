import Link from "next/link";
import { Nav } from "@/components/layout/Nav";
import { PageIntro } from "@/components/layout/PageIntro";
import { SOLUTIONS } from "@/lib/constants";
import { solutionContent } from "@/content/solutions";

export default function SolutionsIndexPage() {
  return (
    <>
      <Nav />
      <main className="mx-auto max-w-7xl px-4 py-16 md:px-6">
        <PageIntro
          label="Solutions"
          title="Problems we help operators solve"
          description="Whether dispatch is your bottleneck, tracking is your blind spot, or payments keep finance up at night — start with the outcome that matters most."
        />
        <div className="mt-16 grid gap-8 md:grid-cols-2">
          {SOLUTIONS.map((solution) => {
            const content = solutionContent[solution.slug];
            return (
              <Link
                key={solution.slug}
                href={`/solutions/${solution.slug}`}
                className="mkt-card block p-8 transition hover:border-[var(--mkt-border-strong)]"
              >
                <h2 className="text-2xl font-semibold">{solution.title}</h2>
                <p className="mt-3 text-[var(--mkt-muted)]">{solution.summary}</p>
                <ul className="mt-6 space-y-2">
                  {content.outcomes.slice(0, 2).map((outcome) => (
                    <li key={outcome} className="flex gap-2 text-sm text-[var(--mkt-muted)]">
                      <span className="text-[var(--mkt-subtle)]">→</span>
                      {outcome}
                    </li>
                  ))}
                </ul>
                <span className="mt-6 inline-block text-sm font-medium">Learn more →</span>
              </Link>
            );
          })}
        </div>
      </main>
    </>
  );
}
