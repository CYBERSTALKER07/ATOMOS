import Link from "next/link";
import type { Route } from "next";
import type { ReactNode } from "react";
import { CAPABILITIES, ROLES, SOLUTIONS } from "@/lib/constants";

type PageIntroProps = {
  label: string;
  title: string;
  description: string;
  back?: { href: Route; label: string };
  children?: ReactNode;
};

export function PageIntro({ label, title, description, back, children }: PageIntroProps) {
  return (
    <header className="mx-auto max-w-4xl">
      {back ? (
        <Link href={back.href} className="text-sm text-[var(--mkt-muted)] hover:text-[var(--mkt-text)]">
          ← {back.label}
        </Link>
      ) : null}
      <p className="void-tag">{label}</p>
      <h1 className="void-section-title mt-4">{title}</h1>
      <p className="mt-4 max-w-2xl text-lg text-[var(--mkt-muted)]">{description}</p>
      {children}
    </header>
  );
}

type DetailSectionProps = {
  title: string;
  children: ReactNode;
  className?: string;
};

export function DetailSection({ title, children, className = "" }: DetailSectionProps) {
  return (
    <section className={`mt-12 ${className}`}>
      <h2 className="text-lg font-semibold">{title}</h2>
      {children}
    </section>
  );
}

type OutcomeListProps = {
  items: string[];
};

export function OutcomeList({ items }: OutcomeListProps) {
  return (
    <ul className="mt-4 space-y-3">
      {items.map((item) => (
        <li key={item} className="flex gap-3 text-sm text-[var(--mkt-muted)]">
          <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-[var(--mkt-text)]" />
          {item}
        </li>
      ))}
    </ul>
  );
}

function titleForCapability(slug: string) {
  return CAPABILITIES.find((c) => c.slug === slug)?.title ?? slug;
}

function titleForRole(slug: string) {
  return ROLES.find((r) => r.slug === slug)?.name ?? slug;
}

function titleForSolution(slug: string) {
  return SOLUTIONS.find((s) => s.slug === slug)?.title ?? slug;
}

type RelatedLinksProps = {
  capabilities?: string[];
  roles?: string[];
  solutions?: string[];
};

export function RelatedLinks({ capabilities, roles, solutions }: RelatedLinksProps) {
  return (
    <div className="mt-4 flex flex-wrap gap-2">
      {capabilities?.map((slug) => (
        <Link key={slug} href={`/capabilities/${slug}`} className="mkt-btn mkt-btn-outline text-xs">
          {titleForCapability(slug)}
        </Link>
      ))}
      {roles?.map((slug) => (
        <Link key={slug} href={`/roles/${slug}`} className="mkt-btn mkt-btn-outline text-xs">
          {titleForRole(slug)}
        </Link>
      ))}
      {solutions?.map((slug) => (
        <Link key={slug} href={`/solutions/${slug}`} className="mkt-btn mkt-btn-outline text-xs">
          {titleForSolution(slug)}
        </Link>
      ))}
    </div>
  );
}
