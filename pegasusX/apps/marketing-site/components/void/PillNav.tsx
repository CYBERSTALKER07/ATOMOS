"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import type { Route } from "next";
import { usePathname } from "next/navigation";
import { Menu, X } from "lucide-react";
import { LANDING_SECTIONS, SITE_NAME } from "@/lib/constants";
import { MobileMenu } from "@/components/layout/MobileMenu";

type PillNavProps = {
  activeSection?: string;
};

type NavLinkItem =
  | { id: string; label: string; href: Route }
  | { id: string; label: string; anchor: true };

const STATIC_LINKS: NavLinkItem[] = [
  { id: "solutions", label: "Solutions", href: "/solutions" },
  { id: "platform", label: "Platform", href: "/platform" },
  { id: "roles", label: "Roles", href: "/roles" },
  { id: "customers", label: "Customers", href: "/customers" },
  { id: "contact", label: "Contact", href: "/contact" },
];

export function PillNav({ activeSection }: PillNavProps) {
  const pathname = usePathname();
  const isLanding = pathname === "/";
  const [menuOpen, setMenuOpen] = useState(false);

  const navLinks: NavLinkItem[] = isLanding
    ? LANDING_SECTIONS.filter((s) =>
        ["hero", "experience", "six-roles", "dispatch-engine", "component-system", "cta"].includes(s.id),
      ).map((section) => ({
        id: section.id,
        label: section.label,
        anchor: true as const,
      }))
    : STATIC_LINKS;

  useEffect(() => {
    setMenuOpen(false);
  }, [pathname]);

  return (
    <>
      <a
        href="#main-content"
        className="fixed left-4 top-4 z-[10001] -translate-y-20 rounded-md border-2 border-white bg-white px-4 py-2 font-bold text-black transition-transform focus:translate-y-0"
      >
        Skip to content
      </a>

      <div className="pointer-events-none fixed inset-x-0 top-6 z-[1000] flex justify-center px-4">
        <div className="pointer-events-auto flex w-full max-w-7xl items-center justify-between gap-3 md:justify-center">
          <Link
            href="/"
            aria-label="Home"
            className="inline-flex h-[42px] w-[42px] shrink-0 items-center justify-center overflow-hidden rounded-full border-2 border-white bg-black md:absolute md:left-4"
          >
            <span className="text-sm font-black text-white">P</span>
          </Link>

          <nav
            className="hidden items-center rounded-full bg-black md:flex"
            style={{ height: "var(--void-nav-h, 42px)" }}
            aria-label="Primary"
          >
            <ul className="flex h-full list-none items-stretch gap-[3px] p-[3px]">
              {navLinks.map((item) => {
                const isActive =
                  "anchor" in item
                    ? activeSection === item.id
                    : pathname.startsWith(item.href);

                const className = `void-pill-link relative inline-flex h-full items-center justify-center overflow-hidden rounded-full px-[18px] text-[13px] font-semibold uppercase tracking-[0.2px] no-underline transition-colors ${
                  isActive
                    ? "bg-white text-black"
                    : "text-white hover:bg-white/10"
                }`;

                return (
                  <li key={item.id} className="flex h-full">
                    {"anchor" in item ? (
                      <Link href={`/#${item.id}` as Route} className={className}>
                        {item.label}
                      </Link>
                    ) : (
                      <Link href={item.href} className={className}>
                        {item.label}
                      </Link>
                    )}
                  </li>
                );
              })}
            </ul>
          </nav>

          <div className="flex items-center gap-2 md:absolute md:right-4">
            <Link
              href="/contact"
              className="void-pill-link hidden h-[42px] items-center rounded-full border-2 border-white bg-white px-5 text-[13px] font-semibold uppercase tracking-wide text-black sm:inline-flex"
            >
              Request demo →
            </Link>
            <button
              type="button"
              className="inline-flex h-10 w-10 items-center justify-center rounded-full border-2 border-white text-white md:hidden"
              aria-label={menuOpen ? "Close menu" : "Open menu"}
              onClick={() => setMenuOpen((v) => !v)}
            >
              {menuOpen ? <X size={18} /> : <Menu size={18} />}
            </button>
          </div>
        </div>
      </div>

      <span className="sr-only">{SITE_NAME}</span>
      <MobileMenu open={menuOpen} onClose={() => setMenuOpen(false)} links={navLinks} />
    </>
  );
}
