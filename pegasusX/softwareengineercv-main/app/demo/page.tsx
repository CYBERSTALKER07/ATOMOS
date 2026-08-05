'use client';

import Link from 'next/link';
import FleekSecondaryLayout from '@/app/components/fleek/FleekSecondaryLayout';
import { O9TourCTA } from '@/app/components/page-sections/o9/O9PageChrome';

const PERSONAS = [
  {
    name: 'Supplier',
    href: '/demo/supplier',
    desc: 'Vet orders, preview dispatch, and review treasury on the supplier control plane.',
    kpi: 'ADMIN role',
  },
  {
    name: 'Warehouse',
    href: '/demo/warehouse',
    desc: 'Visual dispatch board, capacity matching, and live fleet after gate seal.',
    kpi: 'WAREHOUSE_ADMIN',
  },
  {
    name: 'Retailer',
    href: '/demo/retailer',
    desc: 'Checkout, live tracking, shop-closed respond, and pay-at-delivery.',
    kpi: 'RETAILER',
  },
] as const;

export default function DemoPortal() {
  return (
    <FleekSecondaryLayout
      activeHref="/demo"
      sectionTitle="INTERACTIVE DEMO"
      title="Pegasus Dashboard"
      summary="Select a persona to experience specialized workflows — dispatch, fleet, and receiving — on the same order truth."
      primaryHref="/demo/supplier"
      primaryLabel="SUPPLIER DEMO"
      secondaryHref="/join"
      secondaryLabel="REQUEST ACCESS"
      hubId="operations"
      showStack={false}
      section06={
        <>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
            {PERSONAS.map((p) => (
              <Link
                key={p.name}
                href={p.href}
                className="docs-card group transition-colors hover:border-white/30"
              >
                <p className="font-mono text-[10px] uppercase tracking-widest text-white/45">{p.kpi}</p>
                <h2 className="mt-4 text-xl font-semibold">{p.name} Portal</h2>
                <p className="mt-2 text-sm leading-relaxed text-white/50 group-hover:text-white/70">
                  {p.desc}
                </p>
                <span className="mt-8 inline-flex items-center gap-2 font-mono text-[10px] uppercase tracking-widest text-white/35 group-hover:text-[var(--fleek-accent)]">
                  Enter platform →
                </span>
              </Link>
            ))}
          </div>

          <O9TourCTA />

          <p className="mt-10 text-center">
            <Link
              href="/"
              className="text-sm text-white/40 underline-offset-4 hover:text-white hover:underline"
            >
              Return to landing page
            </Link>
          </p>
        </>
      }
    />
  );
}
