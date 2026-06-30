import Link from "next/link";
import { Nav } from "@/components/layout/Nav";
import { PageIntro } from "@/components/layout/PageIntro";
import { customerStories, customersPageContent } from "@/content/customers";

export default function CustomersPage() {
  return (
    <>
      <Nav />
      <main className="mx-auto max-w-7xl px-4 py-16 md:px-6">
        <PageIntro
          label="Customers"
          title={customersPageContent.headline}
          description={customersPageContent.summary}
        />

        <div className="mt-16 space-y-12">
          {customerStories.map((story) => (
            <article key={story.slug} className="mkt-card p-8 md:p-10">
              <div className="flex flex-wrap items-start justify-between gap-4">
                <div>
                  <p className="mkt-section-label">{story.industry}</p>
                  <h2 className="mt-2 text-2xl font-semibold">{story.headline}</h2>
                  <p className="mt-1 text-sm text-[var(--mkt-subtle)]">{story.company}</p>
                </div>
                <div className="flex gap-6">
                  {story.metrics.map((metric) => (
                    <div key={metric.label} className="text-center">
                      <p className="text-2xl font-semibold">{metric.value}</p>
                      <p className="mt-1 text-xs text-[var(--mkt-subtle)]">{metric.label}</p>
                    </div>
                  ))}
                </div>
              </div>

              <p className="mt-6 text-[var(--mkt-muted)]">{story.summary}</p>

              <div className="mt-8 grid gap-6 md:grid-cols-2">
                <div>
                  <h3 className="text-sm font-semibold">The challenge</h3>
                  <p className="mt-2 text-sm text-[var(--mkt-muted)]">{story.challenge}</p>
                </div>
                <div>
                  <h3 className="text-sm font-semibold">The result</h3>
                  <p className="mt-2 text-sm text-[var(--mkt-muted)]">{story.result}</p>
                </div>
              </div>

              <blockquote className="mt-8 border-l-2 border-[var(--mkt-border-strong)] pl-6">
                <p className="text-[var(--mkt-muted)] italic">&ldquo;{story.quote.text}&rdquo;</p>
                <footer className="mt-3 text-sm">
                  <span className="font-medium">{story.quote.author}</span>
                  <span className="text-[var(--mkt-subtle)]"> — {story.quote.role}</span>
                </footer>
              </blockquote>
            </article>
          ))}
        </div>

        <div className="mt-16 text-center">
          <Link href={customersPageContent.cta.href} className="mkt-btn mkt-btn-primary">
            {customersPageContent.cta.label}
          </Link>
        </div>
      </main>
    </>
  );
}
