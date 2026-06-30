"use client";

import { useEffect, useRef } from "react";
import Link from "next/link";
import type { Route } from "next";
import gsap from "gsap";

type NavLinkItem =
  | { id: string; label: string; href: Route }
  | { id: string; label: string; anchor: true };

type MobileMenuProps = {
  open: boolean;
  onClose: () => void;
  links: NavLinkItem[];
};

export function MobileMenu({ open, onClose, links }: MobileMenuProps) {
  const overlayRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open || !overlayRef.current) return;
    const linkEls = overlayRef.current.querySelectorAll("[data-menu-link]");
    gsap.fromTo(
      linkEls,
      { opacity: 0, y: 20 },
      { opacity: 1, y: 0, stagger: 0.06, duration: 0.4, ease: "power3.out" },
    );
  }, [open]);

  if (!open) return null;

  return (
    <div
      ref={overlayRef}
      className="fixed inset-0 z-[10002] flex flex-col bg-black pt-20 lg:hidden"
    >
      <nav className="flex flex-col gap-2 px-6 py-8">
        {links.map((item) =>
          "anchor" in item ? (
            <Link
              key={item.id}
              href={`/#${item.id}` as Route}
              data-menu-link
              className="border-b border-white/20 py-4 text-2xl font-bold uppercase tracking-wide"
              onClick={onClose}
            >
              {item.label}
            </Link>
          ) : (
            <Link
              key={item.id}
              href={item.href}
              data-menu-link
              className="border-b border-white/20 py-4 text-2xl font-bold uppercase tracking-wide"
              onClick={onClose}
            >
              {item.label}
            </Link>
          ),
        )}
        <Link
          href="/contact"
          data-menu-link
          className="mkt-btn mkt-btn-primary mt-6 w-full border-2 border-white font-bold uppercase"
          onClick={onClose}
        >
          Request demo
        </Link>
      </nav>
    </div>
  );
}
