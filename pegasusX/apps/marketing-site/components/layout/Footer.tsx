import Link from "next/link";
import { ROLES, SITE_NAME } from "@/lib/constants";
import { TextMarquee } from "@/components/void/TextMarquee";

const FOOTER_MARQUEE = [
  "DISPATCH",
  "TRACK",
  "DELIVER",
  "COORDINATE",
  "SCALE",
  "OPERATE",
  "CONNECT",
  "RUN",
];

export function Footer() {
  return (
    <footer className="border-t-2 border-white/20 bg-black">
      <div className="void-kinetic-footer border-b border-white/10 py-8">
        <TextMarquee items={FOOTER_MARQUEE} separator="·" speed="slow" />
      </div>

      <div className="mx-auto grid max-w-7xl gap-10 px-4 py-16 md:grid-cols-5 md:px-6">
        <div className="md:col-span-2">
          <p className="text-2xl font-black uppercase tracking-tight">{SITE_NAME}</p>
          <p className="mt-3 max-w-md text-sm leading-relaxed text-[var(--mkt-muted)]">
            The logistics operating system for teams that move physical goods — from supplier
            networks to last-mile delivery.
          </p>
          <Link
            href="/contact"
            className="mkt-btn mkt-btn-primary mt-6 inline-flex border-2 border-white font-bold uppercase tracking-wide"
          >
            Get in touch →
          </Link>
        </div>
        <div>
          <p className="mb-3 font-mono text-[10px] font-bold uppercase tracking-[0.18em] text-[var(--mkt-subtle)]">
            Product
          </p>
          <ul className="space-y-2 text-sm text-[var(--mkt-muted)]">
            <li><Link href="/solutions" className="hover:text-white">Solutions</Link></li>
            <li><Link href="/platform" className="hover:text-white">Platform</Link></li>
            <li><Link href="/capabilities" className="hover:text-white">Capabilities</Link></li>
            <li><Link href="/customers" className="hover:text-white">Customers</Link></li>
          </ul>
        </div>
        <div>
          <p className="mb-3 font-mono text-[10px] font-bold uppercase tracking-[0.18em] text-[var(--mkt-subtle)]">
            Roles
          </p>
          <ul className="space-y-2 text-sm text-[var(--mkt-muted)]">
            {ROLES.map((role) => (
              <li key={role.slug}>
                <Link href={`/roles/${role.slug}`} className="hover:text-white">
                  {role.name}
                </Link>
              </li>
            ))}
          </ul>
        </div>
        <div>
          <p className="mb-3 font-mono text-[10px] font-bold uppercase tracking-[0.18em] text-[var(--mkt-subtle)]">
            Company
          </p>
          <ul className="space-y-2 text-sm text-[var(--mkt-muted)]">
            <li><Link href="/about" className="hover:text-white">About</Link></li>
            <li><Link href="/contact" className="hover:text-white">Contact</Link></li>
            <li><Link href="/components" className="hover:text-white">Developer docs</Link></li>
          </ul>
        </div>
      </div>
      <div className="border-t border-white/10 px-4 py-6 text-center text-xs uppercase tracking-wider text-[var(--mkt-subtle)] md:px-6">
        © {new Date().getFullYear()} {SITE_NAME}. All rights reserved.
      </div>
    </footer>
  );
}
