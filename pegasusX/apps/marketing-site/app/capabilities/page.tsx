import Link from "next/link";
import { Nav } from "@/components/layout/Nav";
import { PageIntro } from "@/components/layout/PageIntro";
import { CAPABILITIES } from "@/lib/constants";
import { capabilityContent } from "@/content/capabilities";

export default function CapabilitiesIndexPage() {
  return (
    <>
      <Nav />
      <main className="mx-auto max-w-7xl px-4 py-16 md:px-6">
        <PageIntro
          label="Capabilities"
          title="What Pegasus does for your network"
          description="From dispatch to payments to live tracking — each capability solves a real operations problem, in language your team already uses."
        />
        <div className="mt-16 grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {CAPABILITIES.map((cap) => {
            const content = capabilityContent[cap.slug];
            return (
              <Link
                key={cap.slug}
                href={`/capabilities/${cap.slug}`}
                className="mkt-card block p-6 transition hover:border-[var(--mkt-border-strong)]"
              >
                <h2 className="text-xl font-semibold">{cap.title}</h2>
                <p className="mt-3 text-sm text-[var(--mkt-muted)]">{content.summary}</p>
                <p className="mt-4 text-xs text-[var(--mkt-subtle)]">For {content.whoItsFor.toLowerCase()}</p>
              </Link>
            );
          })}
        </div>
      </main>
    </>
  );
}
