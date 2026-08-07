'use client';

import Link from 'next/link';
import FleekSecondaryLayout from '@/app/components/fleek/FleekSecondaryLayout';
import { useLanguage } from '@/app/context/LanguageContext';

export default function DemoPortal() {
  const { t } = useLanguage();

  const personas = [
    {
      name: t('demo_persona_supplier', 'Supplier'),
      href: '/demo/supplier',
      desc: t('demo_persona_supplier_desc', 'Vet orders, preview dispatch, and review treasury on the supplier control plane.'),
      kpi: 'ADMIN role',
    },
    {
      name: t('demo_persona_warehouse', 'Warehouse'),
      href: '/demo/warehouse',
      desc: t('demo_persona_warehouse_desc', 'Visual dispatch board, capacity matching, and live fleet after gate seal.'),
      kpi: 'WAREHOUSE_ADMIN',
    },
    {
      name: t('demo_persona_retailer', 'Retailer'),
      href: '/demo/retailer',
      desc: t('demo_persona_retailer_desc', 'Checkout, live tracking, shop-closed respond, and pay-at-delivery.'),
      kpi: 'RETAILER',
    },
  ];

  return (
    <FleekSecondaryLayout
      activeHref="/demo"
      sectionTitle={t('demo_section_title', 'INTERACTIVE DEMO')}
      title={t('demo_title', 'Pegasus Dashboard')}
      summary={t('demo_summary', 'Select a persona to experience specialized workflows — dispatch, fleet, and receiving — on the same order truth.')}
      primaryHref="/demo/supplier"
      primaryLabel={t('demo_primary', 'SUPPLIER DEMO')}
      secondaryHref="/join"
      secondaryLabel={t('demo_secondary', 'REQUEST ACCESS')}
      hubId="operations"
      showStack={false}
      section06={
        <>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
            {personas.map((p) => (
              <Link
                key={p.href}
                href={p.href}
                className="docs-card group transition-colors hover:border-white/30"
              >
                <p className="font-mono text-[10px] uppercase tracking-widest text-white/45">{p.kpi}</p>
                <h2 className="mt-4 text-xl font-semibold">{p.name}</h2>
                <p className="mt-2 text-sm leading-relaxed text-white/50 group-hover:text-white/70">
                  {p.desc}
                </p>
                <span className="mt-8 inline-flex items-center gap-2 font-mono text-[10px] uppercase tracking-widest text-white/35 group-hover:text-[var(--fleek-accent)]">
                  {t('demo_enter_platform', 'Enter platform →')}
                </span>
              </Link>
            ))}
          </div>

          <p className="mt-10 text-center">
            <Link
              href="/"
              className="text-sm text-white/40 underline-offset-4 hover:text-white hover:underline"
            >
              {t('demo_return_landing', 'Return to landing page')}
            </Link>
          </p>
        </>
      }
    />
  );
}
