'use client';

import Link from 'next/link';

type AdminShellProps = {
  title: string;
  subtitle: string;
  badge?: React.ReactNode;
  nav?: { label: string; href: string; active?: boolean }[];
  children: React.ReactNode;
};

export default function AdminShell({ title, subtitle, badge, nav, children }: AdminShellProps) {
  return (
    <div className="min-h-screen bg-black text-white">
      <header className="border-b border-white/10 bg-[#0a0a0a]">
        <div className="mx-auto flex max-w-7xl flex-wrap items-center justify-between gap-4 px-4 py-4">
          <div className="flex flex-wrap items-center gap-6">
            <Link href="/" className="editorial-btn editorial-btn--sm">
              ← Home
            </Link>
            {nav?.map((item) => (
              <Link
                key={item.href}
                href={item.href}
                className={`inline-flex min-h-11 items-center text-sm transition-colors duration-200 ${
                  item.active ? 'text-white' : 'text-white/45 hover:text-white'
                }`}
              >
                {item.label}
              </Link>
            ))}
          </div>
          <div className="flex items-center gap-4 font-mono text-[10px] uppercase tracking-widest text-white/40">
            <span>6 roles</span>
            <span className="text-white/20">·</span>
            <span>One source of truth</span>
            <span className="text-white/20">·</span>
            <span>Ops console</span>
          </div>
        </div>
      </header>

      <div className="mx-auto max-w-7xl px-4 py-10 md:py-14">
        <div className="mb-10 flex flex-wrap items-start justify-between gap-4 border-b border-white/10 pb-8">
          <div>
            <p className="font-mono text-[10px] uppercase tracking-widest text-white/45">Admin</p>
            <h1 className="mt-2 text-3xl font-semibold tracking-tight md:text-4xl">{title}</h1>
            <p className="mt-2 max-w-2xl text-white/55">{subtitle}</p>
          </div>
          {badge}
        </div>
        {children}
      </div>
    </div>
  );
}
