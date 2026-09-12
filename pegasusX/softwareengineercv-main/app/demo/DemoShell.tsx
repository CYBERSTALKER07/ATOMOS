'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import FleekNav from '@/app/components/fleek/FleekNav';
import { useLanguage } from '@/app/context/LanguageContext';

export default function DemoShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const { language, t } = useLanguage();
  const isHome = pathname === '/demo';

  const personaKey = pathname.includes('/supplier')
    ? 'supplier'
    : pathname.includes('/warehouse')
      ? 'warehouse'
      : pathname.includes('/retailer')
        ? 'retailer'
        : 'select';

  const personaLabels =
    language === 'ru'
      ? {
          supplier: 'Поставщик',
          warehouse: 'Склад',
          retailer: 'Ритейлер',
          select: 'Выберите персону',
        }
      : {
          supplier: 'Supplier',
          warehouse: 'Warehouse',
          retailer: 'Retailer',
          select: 'Select Persona',
        };

  const persona = personaLabels[personaKey];

  const navLinks = [
    {
      name: personaLabels.supplier,
      href: '/demo/supplier',
      active: personaKey === 'supplier',
    },
    {
      name: personaLabels.warehouse,
      href: '/demo/warehouse',
      active: personaKey === 'warehouse',
    },
    {
      name: personaLabels.retailer,
      href: '/demo/retailer',
      active: personaKey === 'retailer',
    },
  ];

  if (isHome) {
    return <>{children}</>;
  }

  return (
    <div className="fleek-docs min-h-screen bg-black text-white">
      <FleekNav activeHref="/demo" />
      <div className="flex min-h-[calc(100vh-5rem)] flex-col md:flex-row pt-[4.5rem] md:pt-20">
        <aside className="flex w-full shrink-0 flex-col border-b border-white/10 bg-[#0a0a0a] md:w-56 md:border-b-0 md:border-r md:border-white/10">
          <div className="border-b border-white/10 p-5">
            <p className="font-mono text-[10px] uppercase tracking-widest text-white/45">
              {t('demo_portal', 'Demo portal')}
            </p>
            <p className="mt-2 text-sm font-medium text-white/80">
              {t('demo_persona_workspaces', 'Persona workspaces')}
            </p>
          </div>
          <nav className="flex-1 p-3" aria-label={t('demo_personas', 'Demo personas')}>
            <p className="mb-2 px-2 font-mono text-[10px] uppercase tracking-widest text-white/35">
              {t('demo_personas_label', 'Personas')}
            </p>
            {navLinks.map((link) => (
              <Link
                key={link.href}
                href={link.href}
                className={`mb-1 block min-h-11 border-l-2 px-3 py-2.5 text-sm transition-colors duration-200 ${
                  link.active
                    ? 'border-[var(--fleek-accent)] bg-white/10 font-medium text-white'
                    : 'border-transparent text-white/50 hover:bg-white/5 hover:text-white'
                }`}
              >
                {link.name}
              </Link>
            ))}
          </nav>
          <div className="border-t border-white/10 p-4">
            <Link
              href="/demo"
              className="inline-flex min-h-11 items-center text-xs text-white/45 hover:text-white"
            >
              {t('demo_all_personas', '← All personas')}
            </Link>
          </div>
        </aside>

        <main className="flex min-h-0 flex-1 flex-col overflow-hidden">
          <header className="flex h-14 shrink-0 items-center justify-between border-b border-white/10 bg-[#0a0a0a] px-6">
            <div className="flex items-center gap-2 text-sm">
              <span className="h-2 w-2 rounded-full bg-[var(--fleek-accent)]" />
              <span className="font-medium">
                {persona} {t('demo_dashboard_suffix', 'dashboard')}
              </span>
            </div>
            <span className="font-mono text-[10px] uppercase tracking-widest text-white/40">
              {t('demo_mode', 'Demo mode')}
            </span>
          </header>

          <div className="relative flex-1 overflow-y-auto p-4 md:p-8">
            <div className="relative z-10 mx-auto max-w-7xl">{children}</div>
          </div>
        </main>
      </div>
    </div>
  );
}
