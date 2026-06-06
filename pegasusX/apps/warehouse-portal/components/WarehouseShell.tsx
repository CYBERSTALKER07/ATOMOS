'use client';

import { usePathname } from 'next/navigation';
import Link from 'next/link';
import type { Route } from 'next';
import { useState, memo, useMemo, useCallback, useEffect } from 'react';
import Icon from './Icon';
import { useTheme, type ThemeMode } from './ThemeProvider';
import { PanelLeftClose, PanelLeft } from 'lucide-react';
import { motion } from 'framer-motion';

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
      { href: '/transfers', icon: 'transfers', label: 'Transfers' },
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

function buildBreadcrumbs(pathname: string): { label: string; href: string }[] {
  if (pathname === '/') return [{ label: 'Dashboard', href: '/' }];
  const segs = pathname.split('/').filter(Boolean);
  const crumbs: { label: string; href: string }[] = [{ label: 'Home', href: '/' }];
  let path = '';
  for (const seg of segs) {
    path += '/' + seg;
    crumbs.push({ label: seg.charAt(0).toUpperCase() + seg.slice(1).replace(/-/g, ' '), href: path });
  }
  return crumbs;
}

const BARE_ROUTES = ['/auth/'];

/* ── Theme Toggle ── */
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
      className="desk-icon-btn"
      title={`Theme: ${mode}`}
    >
      <Icon name={iconName[mode]} size={18} />
    </button>
  );
});

/* ── Drawer Content ── */
const DrawerContent = memo(function DrawerContent({
  collapsed,
  isMobile,
  pathname,
  onToggle,
  onLogout,
}: {
  collapsed: boolean;
  isMobile: boolean;
  pathname: string;
  onToggle: () => void;
  onLogout: () => void;
}) {
  const isRail = collapsed && !isMobile;

  return (
    <div className="flex flex-col h-full">
      <div className="flex-1 overflow-y-auto overflow-x-hidden">
        {/* Header */}
        <div className={`flex items-center gap-3 transition-all duration-200 ${isRail ? 'justify-center px-2 pt-4 pb-2' : 'px-4 pt-4 pb-2'}`}>
          {isRail ? (
            <button onClick={onToggle} aria-label="Open sidebar" className="desk-icon-btn">
              <PanelLeft size={20} strokeWidth={1.75} />
            </button>
          ) : (
            <motion.div
              initial={{ opacity: 0, x: -10 }}
              animate={{ opacity: 1, x: 0 }}
              className="flex items-center gap-3 w-full"
            >
              <div className="desk-logo-mark">W</div>
              <div className="min-w-0 flex-1">
                <p className="desk-sidebar-section-label" style={{ padding: 0, margin: 0 }}>Warehouse workspace</p>
                <h1 style={{ font: 'var(--type-title)', color: 'var(--desk-text-primary)', margin: 0, letterSpacing: '-0.02em' }}>
                  Warehouse Portal
                </h1>
              </div>
              {!isMobile && (
                <button onClick={onToggle} className="desk-icon-btn" style={{ width: 28, height: 28 }} aria-label="Collapse sidebar">
                  <PanelLeftClose size={16} strokeWidth={1.75} />
                </button>
              )}
            </motion.div>
          )}
        </div>

        {/* Divider */}
        <div style={{ height: 1, background: 'var(--desk-border)', margin: isRail ? '4px 8px' : '4px 16px' }} />

        {/* Navigation */}
        <nav className={`flex flex-col gap-0.5 mt-1 transition-all duration-200 ${isRail ? 'px-1.5' : 'px-2.5'}`}>
          {NAV.map((section, si) => (
            <div key={si}>
              {si > 0 && <div style={{ height: 1, background: 'var(--desk-border)', margin: isRail ? '8px 4px' : '8px 12px' }} />}
              {section.label && !isRail && (
                <div className="desk-sidebar-section-label">{section.label}</div>
              )}
              {section.items.map((item, ii) => {
                const active = isActiveRoute(pathname, item.href);
                return (
                  <motion.div
                    key={item.href}
                    initial={{ opacity: 0, x: -8 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: (si * 0.08) + (ii * 0.02), type: 'spring', stiffness: 320, damping: 28 }}
                  >
                    <Link
                      href={item.href as Route}
                      prefetch={false}
                      className={`desk-sidebar-item ${active ? 'desk-sidebar-item--accent' : ''}`}
                      title={isRail ? item.label : undefined}
                      aria-label={item.label}
                       style={isRail ? { justifyContent: 'center', padding: '0', height: 44 } : undefined}
                    >
                      <Icon name={item.icon} size={18} className="desk-sidebar-item-icon" />
                      {!isRail && <span className="truncate">{item.label}</span>}
                    </Link>
                  </motion.div>
                );
              })}
            </div>
          ))}
        </nav>
      </div>

      {/* Footer */}
      <div className={`py-3 transition-all duration-200 ${isRail ? 'px-2' : 'px-3'}`} style={{ borderTop: '1px solid var(--desk-border)' }}>
        <div className={`flex items-center ${isRail ? 'justify-center' : 'gap-2'}`}>
          <ThemeToggle />
          {!isRail && (
            <Link
              href="/auth/login"
              onClick={onLogout}
              className="desk-sidebar-item flex-1"
              title="Sign Out"
            >
              <Icon name="logout" size={18} />
              <span>Sign Out</span>
            </Link>
          )}
        </div>
      </div>
    </div>
  );
});

/* ── Shell ── */
export default function WarehouseShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const [collapsed, setCollapsed] = useState(false);
  const [refreshEpoch, setRefreshEpoch] = useState(0);

  const breadcrumbs = useMemo(() => buildBreadcrumbs(pathname), [pathname]);

  const handleLogout = useCallback(() => {
    document.cookie = 'pegasus_warehouse_jwt=; Max-Age=0; path=/';
  }, []);

  useEffect(() => {
    const wakeRefresh = () => {
      if (document.visibilityState === 'hidden') return;
      setRefreshEpoch((current) => current + 1);
    };

    window.addEventListener('focus', wakeRefresh);
    window.addEventListener('pageshow', wakeRefresh);
    window.addEventListener('online', wakeRefresh);
    document.addEventListener('visibilitychange', wakeRefresh);

    return () => {
      window.removeEventListener('focus', wakeRefresh);
      window.removeEventListener('pageshow', wakeRefresh);
      window.removeEventListener('online', wakeRefresh);
      document.removeEventListener('visibilitychange', wakeRefresh);
    };
  }, []);

  const isBare = BARE_ROUTES.some(r => pathname.startsWith(r));
  if (isBare) return <>{children}</>;

  return (
    <div className="flex h-dvh overflow-hidden" style={{ background: 'var(--desk-canvas)' }}>
      {/* Desktop Sidebar */}
      <motion.aside
        initial={false}
        animate={{ width: collapsed ? 72 : 264 }}
        transition={{ type: 'spring', stiffness: 300, damping: 30 }}
        className="hidden md:flex flex-col shrink-0 overflow-hidden"
        style={{
          borderRight: '1px solid var(--desk-border)',
          background: 'var(--desk-surface)',
        }}
      >
        <DrawerContent
          collapsed={collapsed}
          isMobile={false}
          pathname={pathname}
          onToggle={() => setCollapsed(c => !c)}
          onLogout={handleLogout}
        />
      </motion.aside>

      {/* Main content */}
      <div className="flex min-w-0 flex-1 flex-col">
        {/* Top bar */}
        <header className="desk-topbar shrink-0">
          <div className="desk-topbar-left">
            <nav className="desk-breadcrumb" aria-label="Breadcrumb">
              {breadcrumbs.map((crumb, i) => (
                <span key={crumb.href} className="flex items-center gap-2 min-w-0">
                  {i > 0 && <span className="desk-breadcrumb-sep">/</span>}
                  {i === breadcrumbs.length - 1 ? (
                    <span className="desk-breadcrumb-current truncate">{crumb.label}</span>
                  ) : (
                    <Link href={crumb.href as Route} prefetch={false} className="truncate">{crumb.label}</Link>
                  )}
                </span>
              ))}
            </nav>
          </div>

          <div className="desk-topbar-right">
            <div className="desk-live-indicator hidden lg:inline-flex">
              <span className="desk-live-dot" />
              Warehouse live
            </div>
          </div>
        </header>

        <main
          key={refreshEpoch}
          className="flex-1 min-w-0 overflow-y-auto"
          style={{ background: 'var(--desk-canvas)' }}
        >
          {children}
        </main>
      </div>
    </div>
  );
}
