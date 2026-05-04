'use client';

import { usePathname } from 'next/navigation';
import Link from 'next/link';
import { memo, useEffect, useMemo, useState } from 'react';
import { PanelLeft, PanelLeftClose } from 'lucide-react';
import Icon from './Icon';
import { useTheme, type ThemeMode } from './ThemeProvider';
import { apiFetch } from '@/lib/auth';

type NavEntry = { href: string; icon: string; label: string };
type NavSection = { label?: string; items: NavEntry[] };

const NAV: NavSection[] = [
  {
    items: [
      { href: '/', icon: 'dashboard', label: 'Dashboard' },
      { href: '/loading-bay', icon: 'loadingBay', label: 'Loading Bay' },
      { href: '/transfers', icon: 'transfers', label: 'Transfers' },
    ],
  },
  {
    label: 'Operations',
    items: [
      { href: '/fleet', icon: 'fleet', label: 'Fleet' },
      { href: '/staff', icon: 'staff', label: 'Staff' },
      { href: '/insights', icon: 'insights', label: 'Insights' },
    ],
  },
  {
    label: 'Supply Chain',
    items: [
      { href: '/supply-requests', icon: 'transfers', label: 'Supply Requests' },
      { href: '/payload-override', icon: 'loadingBay', label: 'Payload Override' },
    ],
  },
];

const ALL_NAV_ITEMS = NAV.flatMap((section) => section.items);
const BARE_ROUTES = ['/auth/'];
const PAGE_SUMMARIES: Record<string, string> = {
  '/': 'Monitor transfer readiness, staffing, fleet coverage, and dispatch pressure from one desktop command view.',
  '/loading-bay': 'Keep approved payloads moving through loading without losing track of dispatch readiness.',
  '/transfers': 'Review active factory-to-warehouse movements, state transitions, and manifest health.',
  '/fleet': 'Inspect vehicle availability, assignments, and operational readiness.',
  '/staff': 'Track shifts, assigned operators, and coverage gaps across the factory floor.',
  '/insights': 'Review alerts, factory intelligence, and operational drift before it becomes a delay.',
  '/supply-requests': 'Review inbound warehouse demand and convert requests into planned outbound work.',
  '/payload-override': 'Handle controlled manual overrides when payload automation needs an operator decision.',
};

function isActiveRoute(pathname: string, href: string): boolean {
  if (href === '/') return pathname === '/';
  return pathname === href || pathname.startsWith(`${href}/`);
}

const ThemeToggle = memo(function ThemeToggle() {
  const { mode, cycle } = useTheme();
  const iconName: Record<ThemeMode, string> = {
    system: 'autoMode',
    light: 'lightMode',
    dark: 'darkMode',
  };

  return (
    <button
      onClick={cycle}
      className="flex h-9 w-9 items-center justify-center rounded-xl border border-[var(--border)] bg-[var(--background)] transition-colors hover:bg-[var(--surface)]"
      title={`Theme: ${mode}`}
    >
      <Icon name={iconName[mode]} size={18} />
    </button>
  );
});

function ShellActionLink({ href, icon, label }: NavEntry) {
  return (
    <Link
      href={href}
      className="inline-flex h-10 items-center gap-2 rounded-full border border-[var(--border)] bg-[var(--background)] px-4 text-sm font-medium text-[var(--foreground)] transition-colors hover:bg-[var(--surface)]"
    >
      <Icon name={icon} size={16} />
      <span>{label}</span>
    </Link>
  );
}

export default function FactoryShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const [collapsed, setCollapsed] = useState(false);
  const [factoryName, setFactoryName] = useState('Factory Portal');

  const currentEntry = useMemo(
    () => ALL_NAV_ITEMS.find((item) => isActiveRoute(pathname, item.href)) ?? ALL_NAV_ITEMS[0],
    [pathname],
  );
  const currentSection = useMemo(
    () => NAV.find((section) => section.items.some((item) => item.href === currentEntry.href))?.label ?? 'Factory workspace',
    [currentEntry.href],
  );
  const pageSummary = PAGE_SUMMARIES[currentEntry.href] ?? 'Factory desktop operations workspace.';

  useEffect(() => {
    let active = true;
    async function loadFactoryProfile() {
      const res = await apiFetch('/v1/factory/profile');
      if (!res.ok) return;
      const payload = (await res.json()) as { name?: string };
      const resolved = payload.name?.trim();
      if (active && resolved) {
        setFactoryName(resolved);
      }
    }
    loadFactoryProfile().catch((error) => {
      console.error('[FactoryShell] profile load failed', error);
    });
    return () => {
      active = false;
    };
  }, []);

  const isBare = BARE_ROUTES.some((route) => pathname.startsWith(route));
  if (isBare) return <>{children}</>;

  return (
    <>
      <aside
        className="flex h-screen flex-col border-r border-[var(--border)] bg-[var(--background)] transition-[width] duration-200"
        style={{ width: collapsed ? 72 : 264 }}
      >
        <div className="flex h-16 items-center gap-3 border-b border-[var(--border)] px-4">
          <div className="flex h-10 w-10 items-center justify-center rounded-2xl bg-[var(--accent)] text-[var(--accent-foreground)]">
            <Icon name="loadingBay" size={18} />
          </div>
          {!collapsed && (
            <div className="min-w-0 flex-1">
              <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--muted)]">Factory workspace</p>
               <span className="block truncate text-sm font-bold tracking-tight text-[var(--foreground)]">
                 {factoryName}
               </span>
             </div>
           )}
          <button
            onClick={() => setCollapsed((value) => !value)}
            className="flex h-9 w-9 items-center justify-center rounded-xl border border-[var(--border)] bg-[var(--background)] transition-colors hover:bg-[var(--surface)]"
          >
            {collapsed ? <PanelLeft size={18} /> : <PanelLeftClose size={18} />}
          </button>
        </div>

        <nav className="flex-1 overflow-y-auto px-3 py-3">
          {NAV.map((section, sectionIndex) => (
            <div key={sectionIndex} className={sectionIndex > 0 ? 'mt-4' : ''}>
              {section.label && !collapsed && (
                <div className="px-3 pb-1 text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--muted)]">
                  {section.label}
                </div>
              )}
              <div className="space-y-1">
                {section.items.map((item) => {
                  const active = isActiveRoute(pathname, item.href);
                  return (
                    <Link
                      key={item.href}
                      href={item.href}
                      className={`flex items-center gap-3 rounded-2xl px-3 py-2.5 text-sm font-medium transition-colors ${
                        active
                          ? 'bg-[var(--accent)] text-[var(--accent-foreground)]'
                          : 'text-[var(--muted)] hover:bg-[var(--surface)] hover:text-[var(--foreground)]'
                      }`}
                      title={collapsed ? item.label : undefined}
                    >
                      <Icon name={item.icon} size={20} />
                      {!collapsed && <span className="truncate">{item.label}</span>}
                    </Link>
                  );
                })}
              </div>
            </div>
          ))}
        </nav>

        <div className="border-t border-[var(--border)] px-3 py-3">
          {!collapsed && (
            <div className="mb-3 rounded-2xl border border-[var(--border)] bg-[var(--surface)] p-3">
              <div className="flex items-center gap-2">
                <span className="h-2 w-2 rounded-full bg-[var(--success)]" />
                <p className="text-sm font-semibold text-[var(--foreground)]">Desktop command ready</p>
              </div>
              <p className="mt-1 text-xs leading-5 text-[var(--muted)]">
                Dispatch, transfer, and loading views are synced for shift-level operations.
              </p>
            </div>
          )}

          <div className={`flex items-center ${collapsed ? 'justify-center' : 'justify-between gap-2'}`}>
            <ThemeToggle />
            <Link
              href="/auth/login"
              onClick={() => {
                document.cookie = 'pegasus_factory_jwt=; Max-Age=0; path=/';
              }}
              className={`flex items-center gap-2 rounded-xl px-2 py-2 text-sm text-[var(--muted)] transition-colors hover:bg-[var(--surface)] hover:text-[var(--danger)] ${
                collapsed ? 'justify-center' : ''
              }`}
              title="Sign Out"
            >
              <Icon name="logout" size={18} />
              {!collapsed && <span>Sign Out</span>}
            </Link>
          </div>
        </div>
      </aside>

      <div className="flex min-w-0 flex-1 flex-col bg-[var(--color-md-surface-container-low)]">
        <header className="sticky top-0 z-10 border-b border-[var(--border)] bg-[color:color-mix(in_oklch,var(--background)_92%,transparent)] px-6 py-4 backdrop-blur-md">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div className="min-w-0">
              <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--muted)]">{currentSection}</p>
              <h1 className="truncate text-2xl font-semibold tracking-tight text-[var(--foreground)]">{currentEntry.label}</h1>
              <p className="max-w-3xl text-sm text-[var(--muted)]">{pageSummary}</p>
            </div>

            <div className="flex flex-wrap items-center gap-3">
              <div className="inline-flex items-center gap-2 rounded-full border border-[color:color-mix(in_oklch,var(--success)_28%,var(--border))] bg-[color:color-mix(in_oklch,var(--success)_12%,transparent)] px-3 py-2 text-[11px] font-semibold uppercase tracking-[0.16em] text-[var(--foreground)]">
                <span className="h-2 w-2 rounded-full bg-[var(--success)]" />
                Factory network live
              </div>
              <ShellActionLink href="/loading-bay" icon="loadingBay" label="Loading Bay" />
              <ShellActionLink href="/transfers" icon="transfers" label="Transfers" />
            </div>
          </div>
        </header>

        <main className="min-w-0 flex-1 overflow-y-auto">
          <div className="min-h-full">{children}</div>
        </main>
      </div>
    </>
  );
}
