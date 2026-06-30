import Link from "next/link";
import { Nav } from "@/components/layout/Nav";
import { DetailSection, PageIntro } from "@/components/layout/PageIntro";
import { aboutContent } from "@/content/company";

export default function AboutPage() {
  return (
    <>
      <Nav />
      <main className="mx-auto max-w-4xl px-4 py-16 md:px-6">
        <PageIntro
          label="About"
          title={aboutContent.headline}
          description={aboutContent.summary}
        />

        <div className="mt-12 space-y-6 text-[var(--mkt-muted)]">
          {aboutContent.story.map((paragraph) => (
            <p key={paragraph.slice(0, 40)}>{paragraph}</p>
          ))}
        </div>

        <DetailSection title="What we believe">
          <div className="mt-6 grid gap-6 md:grid-cols-3">
            {aboutContent.values.map((value) => (
              <div key={value.title} className="mkt-card p-6">
                <h3 className="font-semibold">{value.title}</h3>
                <p className="mt-3 text-sm text-[var(--mkt-muted)]">{value.body}</p>
              </div>
            ))}
          </div>
        </DetailSection>

        <div className="mt-16 flex flex-wrap gap-4">
          <Link href="/contact" className="mkt-btn mkt-btn-primary">
            Get in touch
          </Link>
          <Link href="/platform" className="mkt-btn mkt-btn-outline">
            See the platform
          </Link>
        </div>
      </main>
    </>
  );
}
