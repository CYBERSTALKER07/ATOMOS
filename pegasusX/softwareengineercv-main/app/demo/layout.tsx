'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import FleekNav from '@/app/components/fleek/FleekNav';

export default function DemoLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const isHome = pathname === '/demo';

  const persona = pathname.includes('/supplier')
    ? 'Supplier'
    : pathname.includes('/warehouse')
      ? 'Warehouse'
      : pathname.includes('/retailer')
        ? 'Retailer'
        : 'Select Persona';

  const navLinks = [
    { name: 'Supplier', href: '/demo/supplier', active: persona === 'Supplier' },
    { name: 'Warehouse', href: '/demo/warehouse', active: persona === 'Warehouse' },
    { name: 'Retailer', href: '/demo/retailer', active: persona === 'Retailer' },
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
            <p className="font-mono text-[10px] uppercase tracking-widest text-white/45">Demo portal</p>
            <p className="mt-2 text-sm font-medium text-white/80">Persona workspaces</p>
          </div>
          <nav className="flex-1 p-3" aria-label="Demo personas">
            <p className="mb-2 px-2 font-mono text-[10px] uppercase tracking-widest text-white/35">
              Personas
            </p>
            {navLinks.map((link) => (
              <Link
                key={link.name}
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
              ← All personas
            </Link>
          </div>
        </aside>

        <main className="flex min-h-0 flex-1 flex-col overflow-hidden">
          <header className="flex h-14 shrink-0 items-center justify-between border-b border-white/10 bg-[#0a0a0a] px-6">
            <div className="flex items-center gap-2 text-sm">
              <span className="h-2 w-2 rounded-full bg-[var(--fleek-accent)]" />
              <span className="font-medium">{persona} dashboard</span>
            </div>
            <span className="font-mono text-[10px] uppercase tracking-widest text-white/40">Demo mode</span>
          </header>

          <div className="relative flex-1 overflow-y-auto p-4 md:p-8">
            <div className="relative z-10 mx-auto max-w-7xl">{children}</div>
          </div>
        </main>
      </div>
    </div>
  );
}
