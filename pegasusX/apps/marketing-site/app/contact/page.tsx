import { Nav } from "@/components/layout/Nav";
import { PageIntro } from "@/components/layout/PageIntro";
import { contactContent } from "@/content/company";

export default function ContactPage() {
  return (
    <>
      <Nav />
      <main className="mx-auto max-w-4xl px-4 py-16 md:px-6">
        <PageIntro
          label="Contact"
          title={contactContent.headline}
          description={contactContent.summary}
        />

        <div className="mt-12 grid gap-12 md:grid-cols-2">
          <form
            className="flex flex-col gap-4"
            action={`mailto:${contactContent.email}`}
            method="get"
          >
            <div>
              <label htmlFor="name" className="mb-2 block text-sm font-medium">
                Your name
              </label>
              <input
                id="name"
                name="body"
                type="text"
                placeholder="Alex Morgan"
                className="w-full min-h-[44px] rounded-xl border border-[var(--mkt-border)] bg-[var(--mkt-surface)] px-4 text-sm outline-none focus:border-[var(--mkt-text)]"
              />
            </div>
            <div>
              <label htmlFor="email" className="mb-2 block text-sm font-medium">
                Work email
              </label>
              <input
                id="email"
                type="email"
                name="subject"
                placeholder="you@company.com"
                className="w-full min-h-[44px] rounded-xl border border-[var(--mkt-border)] bg-[var(--mkt-surface)] px-4 text-sm outline-none focus:border-[var(--mkt-text)]"
              />
            </div>
            <div>
              <label htmlFor="message" className="mb-2 block text-sm font-medium">
                Tell us about your operation
              </label>
              <textarea
                id="message"
                name="body"
                rows={5}
                placeholder="How many warehouses, what you deliver, where things break down today..."
                className="w-full rounded-xl border border-[var(--mkt-border)] bg-[var(--mkt-surface)] px-4 py-3 text-sm outline-none focus:border-[var(--mkt-text)]"
              />
            </div>
            <button type="submit" className="mkt-btn mkt-btn-primary w-full sm:w-auto">
              Send message
            </button>
          </form>

          <aside>
            <h2 className="text-lg font-semibold">What we can cover</h2>
            <ul className="mt-4 space-y-3">
              {contactContent.topics.map((topic) => (
                <li key={topic} className="flex gap-2 text-sm text-[var(--mkt-muted)]">
                  <span className="text-[var(--mkt-subtle)]">→</span>
                  {topic}
                </li>
              ))}
            </ul>
            <p className="mt-8 text-sm text-[var(--mkt-subtle)]">
              Or email us directly at{" "}
              <a href={`mailto:${contactContent.email}`} className="text-[var(--mkt-text)] hover:underline">
                {contactContent.email}
              </a>
            </p>
          </aside>
        </div>
      </main>
    </>
  );
}
