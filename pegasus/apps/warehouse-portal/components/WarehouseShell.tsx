'use client';

import { usePathname } from 'next/navigation';
import Link from 'next/link';
import { useState, memo } from 'react';
import Icon from './Icon';
import { useTheme, type ThemeMode } from './ThemeProvider';
import { PanelLeftClose, PanelLeft } from 'lucide-react';

type NavEntry = { href: string; icon: string; label: string };
type NavSection = { label?: string; items: NavEntry[] };

const NAV: NavSection[] = [
  {
    items: [
      { href: '/', icon: 'dashboard', label: 'Dashboard' },
      { href: '/orders', icon: 'orders', label: 'Orders' },
      { href: '/dispatch', icon: 'dispatch', label: 'Dispatch' },
      { href: '/manifests', icon: 'manifests', label: 'Manifests' },
    ],
  },
  {
    label: 'Inventory',
    items: [
      { href: '/inventory', icon: 'inventory', label: 'Stock' },
      { href: '/products', icon: 'catalog', label: 'Products' },
      { href: '/supply-requests', icon: 'supplyRequests', label: 'Supply Requests' },
      { href: '/demand-forecast', icon: 'forecast', label: 'Demand Forecast' },
    ],
  },
  {
    label: 'Fleet',
    items: [
      { href: '/drivers', icon: 'fleet', label: 'Drivers' },
      { href: '/vehicles', icon: 'fleet', label: 'Vehicles' },
      { href: '/dispatch-locks', icon: 'lock', label: 'Dispatch Locks' },
    ],
  },
  {
    label: 'Operations',
    items: [
      { href: '/staff', icon: 'staff', label: 'Staff' },
      { href: '/crm', icon: 'crm', label: 'Retailers' },
      { href: '/returns', icon: 'returns', label: 'Returns' },
      { href: '/analytics', icon: 'analytics', label: 'Analytics' },
    ],
  },
  {
    label: 'Finance',
    items: [
      { href: '/treasury', icon: 'treasury', label: 'Treasury' },
      { href: '/payment-config', icon: 'payment', label: 'Payment Config' },
    ],
  },
];

function isActiveRoute(pathname: string, href: string): boolean {
  if (href === '/') return pathname === '/';
  return pathname === href || pathname.startsWith(href + '/');
}

const BARE_ROUTES = ['/auth/'];

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
      className="flex items-center justify-center w-9 h-9 rounded-lg transition-colors hover:bg-[var(--surface)]"
      title={`Theme: ${mode}`}
    >
      <Icon name={iconName[mode]} size={18} />
    </button>
  );
});

export default function WarehouseShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const [collapsed, setCollapsed] = useState(false);

  const isBare = BARE_ROUTES.some(r => pathname.startsWith(r));
  if (isBare) return <>{children}</>;

  return (
    <>
      <aside
        className="h-screen flex flex-col border-r border-[var(--border)] bg-[var(--background)] transition-[width] duration-200"
        style={{ width: collapsed ? 64 : 240 }}
      >
        <div className="flex items-center gap-3 px-4 h-14 border-b border-[var(--border)]">
          {!collapsed && (
            <span className="text-sm font-bold tracking-tight truncate flex-1">
              Warehouse Portal
            </span>
          )}
          <button
            onClick={() => setCollapsed(c => !c)}
            className="flex items-center justify-center w-8 h-8 rounded-lg hover:bg-[var(--surface)] transition-colors"
          >
            {collapsed ? <PanelLeft size={18} /> : <PanelLeftClose size={18} />}
          </button>
        </div>

        <nav className="flex-1 overflow-y-auto py-2 px-2">
          {NAV.map((section, si) => (
            <div key={si}>
              {section.label && !collapsed && (
                <div className="px-2 pt-4 pb-1 text-[11px] font-semibold uppercase tracking-wider text-[var(--muted)]">
                  {section.label}
                </div>
              )}
              {section.items.map(item => {
                const active = isActiveRoute(pathname, item.href);
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    className={`
                      flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors
                      ${active
                        ? 'bg-[var(--accent)] text-[var(--accent-foreground)]'
                        : 'text-[var(--muted)] hover:bg-[var(--surface)] hover:text-[var(--foreground)]'
                      }
                    `}
                    title={collapsed ? item.label : undefined}
                  >
                    <Icon name={item.icon} size={20} />
                    {!collapsed && <span className="truncate">{item.label}</span>}
                  </Link>
                );
              })}
            </div>
          ))}
        </nav>

        <div className="px-3 py-3 border-t border-[var(--border)] flex items-center gap-2">
          <ThemeToggle />
          {!collapsed && (
            <Link
              href="/auth/login"
              onClick={() => {
                document.cookie = 'pegasus_warehouse_jwt=; Max-Age=0; path=/';
              }}
              className="flex items-center gap-2 text-sm text-[var(--muted)] hover:text-[var(--danger)]"
            >
              <Icon name="logout" size={18} />
              <span>Sign Out</span>
            </Link>
          )}
        </div>
      </aside>

      <main className="flex-1 min-w-0 overflow-y-auto">
        {children}
      </main>
    </>
  );
}
