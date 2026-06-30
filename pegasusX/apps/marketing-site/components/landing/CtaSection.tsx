import Link from "next/link";
import type { Route } from "next";
import { trustStrip } from "@/content/pegasus";
import { experiencePaths } from "@/content/landing/enterprise";
import { SectionShell } from "@/components/layout/SectionShell";
import { SectionHeader } from "@/components/docs/SpecTable";
import { TextMarquee } from "@/components/void/TextMarquee";

const CTA_MARQUEE = ["CONNECT", "GET IN TOUCH", "REQUEST DEMO"];

export function CtaSection() {
  return (
    <SectionShell id="cta" minHeight="min-h-[70vh]" className="relative overflow-hidden py-24">
      <TextMarquee items={CTA_MARQUEE} separator="✦" className="mb-12" />
      <div className="relative mx-auto max-w-3xl px-4 text-center md:px-6">
        <SectionHeader
          platformFrame
          label="Contact"
          title="See Pegasus with your network in mind."
          description="Explore the platform on your own, or talk to our team about dispatch, tracking, and payments for your operation."
        />

        <div className="mt-10 flex flex-wrap justify-center gap-4">
          {experiencePaths.map((path) => (
            <Link
              key={path.id}
              href={path.href as Route}
              className={path.id === "demo" ? "mkt-btn mkt-btn-primary" : "mkt-btn mkt-btn-outline"}
            >
              {path.cta}
            </Link>
          ))}
        </div>

        <form
          className="mx-auto mt-10 flex max-w-md flex-col gap-3 sm:flex-row"
          action="mailto:demo@pegasus.io"
          method="get"
        >
          <label htmlFor="cta-email" className="sr-only">
            Email address
          </label>
          <input
            id="cta-email"
            type="email"
            name="subject"
            placeholder="Work email"
            className="min-h-[44px] flex-1 rounded-full border border-[var(--mkt-border)] bg-[var(--mkt-surface)] px-4 text-sm outline-none focus:border-[var(--mkt-text)]"
          />
          <button type="submit" className="mkt-btn mkt-btn-primary">
            Send request
          </button>
        </form>

        <div className="mt-8 flex flex-wrap justify-center gap-2">
          {trustStrip.map((item) => (
            <span key={item} className="font-mono text-[10px] uppercase tracking-wider text-[var(--mkt-subtle)]">
              {item}
            </span>
          ))}
        </div>

        <div className="mt-8 flex flex-wrap justify-center gap-6 text-sm">
          <Link href="/platform" className="text-[var(--mkt-muted)] hover:text-[var(--mkt-text)]">
            How it works →
          </Link>
          <Link href="/solutions" className="text-[var(--mkt-muted)] hover:text-[var(--mkt-text)]">
            Solutions →
          </Link>
          <Link href="/customers" className="text-[var(--mkt-muted)] hover:text-[var(--mkt-text)]">
            Customer stories →
          </Link>
        </div>
      </div>
    </SectionShell>
  );
}
