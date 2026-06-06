'use client';

import { usePathname } from 'next/navigation';
import Link from 'next/link';
import { memo, useEffect, useMemo, useState, useCallback } from 'react';
import { PanelLeft, PanelLeftClose, Search, Bell } from 'lucide-react';
import Icon from './Icon';
import { useTheme, type ThemeMode } from './ThemeProvider';
import { apiFetch } from '@/lib/auth';
import { motion, AnimatePresence } from 'framer-motion';

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
  '/': 'Monitor transfer readiness, staffing, fleet coverage, and dispatch pressure.',
  '/loading-bay': 'Keep approved payloads moving through loading.',
  '/transfers': 'Review active factory-to-warehouse movements and manifest health.',
  '/fleet': 'Inspect vehicle availability and operational readiness.',
  '/staff': 'Track shifts, assigned operators, and coverage gaps.',
  '/insights': 'Review alerts and operational drift before it becomes a delay.',
  '/supply-requests': 'Review inbound warehouse demand and plan outbound work.',
  '/payload-override': 'Handle controlled manual overrides for payload automation.',
};

function isActiveRoute(pathname: string, href: string): boolean {
  if (href === '/') return pathname === '/';
  return pathname === href || pathname.startsWith(`${href}/`);
}

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
  factoryName,
  onToggle,
  onLogout,
}: {
  collapsed: boolean;
  isMobile: boolean;
  pathname: string;
  factoryName: string;
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
              <div className="desk-logo-mark">F</div>
              <div className="min-w-0 flex-1">
                <p className="desk-sidebar-section-label" style={{ padding: 0, margin: 0 }}>Factory workspace</p>
                <h1 style={{ font: 'var(--type-title)', color: 'var(--desk-text-primary)', margin: 0, letterSpacing: '-0.02em' }}>
                  {factoryName}
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
                      href={item.href}
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
        {!isRail && (
          <div className="desk-card-padded desk-card mb-3" style={{ borderRadius: 'var(--radius-md)' }}>
            <div className="flex items-center gap-2">
              <span className="desk-live-dot" />
              <span style={{ font: 'var(--type-caption-sm)', color: 'var(--desk-text-primary)', fontWeight: 600 }}>Desktop command ready</span>
            </div>
            <p style={{ font: '400 12px/18px var(--desktop-font-sans)', color: 'var(--desk-text-tertiary)', margin: '4px 0 0' }}>
              Dispatch, transfer, and loading views synced.
            </p>
          </div>
        )}

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
export default function FactoryShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const [collapsed, setCollapsed] = useState(false);
  const [factoryName, setFactoryName] = useState('Factory Portal');
  const [refreshEpoch, setRefreshEpoch] = useState(0);

  const currentEntry = useMemo(
    () => ALL_NAV_ITEMS.find((item) => isActiveRoute(pathname, item.href)) ?? ALL_NAV_ITEMS[0],
    [pathname],
  );
  const currentSection = useMemo(
    () => NAV.find((section) => section.items.some((item) => item.href === currentEntry.href))?.label ?? 'Factory workspace',
    [currentEntry.href],
  );

  const handleLogout = useCallback(() => {
    document.cookie = 'pegasus_factory_jwt=; Max-Age=0; path=/';
  }, []);

  const loadFactoryProfile = useCallback(async () => {
    const res = await apiFetch('/v1/factory/profile');
    if (!res.ok) return;
    const payload = (await res.json()) as { name?: string };
    const resolved = payload.name?.trim();
    if (resolved) {
      setFactoryName(resolved);
    }
  }, []);

  useEffect(() => {
    loadFactoryProfile().catch((error) => {
      console.error('[FactoryShell] profile load failed', error);
    });
  }, [loadFactoryProfile]);

  useEffect(() => {
    const wakeRefresh = () => {
      if (document.visibilityState === 'hidden') return;
      setRefreshEpoch((current) => current + 1);
      loadFactoryProfile().catch((error) => {
        console.error('[FactoryShell] profile refresh failed', error);
      });
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
  }, [loadFactoryProfile]);

  const isBare = BARE_ROUTES.some((route) => pathname.startsWith(route));
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
          factoryName={factoryName}
          onToggle={() => setCollapsed((c) => !c)}
          onLogout={handleLogout}
        />
      </motion.aside>

      {/* Main content */}
      <div className="flex min-w-0 flex-1 flex-col">
        {/* Top bar */}
        <header className="desk-topbar shrink-0">
          <div className="desk-topbar-left">
            <nav className="desk-breadcrumb" aria-label="Breadcrumb">
              <span className="desk-breadcrumb-sep">{currentSection}</span>
              <span className="desk-breadcrumb-sep">/</span>
              <span className="desk-breadcrumb-current">{currentEntry.label}</span>
            </nav>
          </div>

          <div className="desk-topbar-right">
            <div className="desk-live-indicator hidden lg:inline-flex">
              <span className="desk-live-dot" />
              Factory network live
            </div>
          </div>
        </header>

        <main key={refreshEpoch} className="min-w-0 flex-1 overflow-y-auto" style={{ background: 'var(--desk-canvas)' }}>
          <div className="min-h-full">{children}</div>
        </main>
      </div>
    </div>
  );
}
