/* eslint-disable @typescript-eslint/no-explicit-any */
'use client';

import { usePathname } from 'next/navigation';
import Link from 'next/link';
import { useEffect, useState, useRef, useMemo, useCallback, memo } from 'react';
import Icon from './Icon';
import { useTheme, type ThemeMode } from './ThemeProvider';
import { PanelLeftClose, PanelLeft } from 'lucide-react';
import NotificationPanel from './NotificationPanel';
import ClientPolicyBanner from './ClientPolicyBanner';
import { useNotifications, type WarehouseWsState } from '@/lib/useNotifications';
import { clearSession, decodeJwtPayload, readTokenFromCookie } from '@/lib/auth';
import { motion, AnimatePresence, useReducedMotion } from 'framer-motion';

type NavEntry = { href: string; icon: string; label: string; globalOnly?: boolean; factoryHidden?: boolean };
type NavSection = { label?: string; items: NavEntry[] };

const NAV: NavSection[] = [
  {
    items: [
      { href: '/', icon: 'dashboard', label: 'Dashboard' },
      { href: '/orders', icon: 'orders', label: 'Orders' },
      { href: '/preorders', icon: 'orders', label: 'Pre-orders' },
      { href: '/dispatch', icon: 'dispatch', label: 'Dispatch' },
      { href: '/dispatch-settings', icon: 'settings', label: 'Dispatch Settings' },
      { href: '/manifests', icon: 'manifests', label: 'Manifests' },
    ],
  },
  {
    label: 'Inventory',
    items: [
      { href: '/inventory', icon: 'inventory', label: 'Stock' },
      { href: '/stock-commitments', icon: 'inventory', label: 'Stock commitments' },
      { href: '/products', icon: 'catalog', label: 'Products' },
      { href: '/supply-requests', icon: 'supplyRequests', label: 'Supply Requests' },
      { href: '/settings', icon: 'settings', label: 'Settings' },
      { href: '/replenishment', icon: 'forecast', label: 'Replenishment' },
      { href: '/demand-forecast', icon: 'forecast', label: 'Demand Forecast' },
    ],
  },
  {
    label: 'Fleet',
    items: [
      { href: '/drivers', icon: 'fleet', label: 'Drivers' },
      { href: '/vehicles', icon: 'fleet', label: 'Trucks' },
      { href: '/dispatch-locks', icon: 'lock', label: 'Dispatch Locks' },
    ],
  },
  {
    label: 'Operations',
    items: [
      { href: '/staff', icon: 'staff', label: 'Staff' },
      { href: '/crm', icon: 'crm', label: 'Retailers' },
      { href: '/operations', icon: 'send', label: 'Operations' },
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
  return pathname === href || pathname.startsWith(`${href}/`);
}

function buildBreadcrumbs(pathname: string): { label: string; href: string }[] {
  if (pathname === '/') return [{ label: 'Dashboard', href: '/' }];
  const crumbs: { label: string; href: string }[] = [
    { label: 'Home', href: '/' },
  ];
  let path = '';
  for (const seg of pathname.split('/').filter(Boolean)) {
    path += `/${seg}`;
    crumbs.push({
      label: seg.charAt(0).toUpperCase() + seg.slice(1).replace(/-/g, ' '),
      href: path,
    });
  }
  return crumbs;
}

const BARE_ROUTES = ["/auth/", "/setup/"];

function liveIndicatorCopy(state: WarehouseWsState): { label: string; stale: boolean } {
  switch (state) {
    case 'connected':
      return { label: 'Warehouse live', stale: false };
    case 'reconnecting':
      return { label: 'Reconnecting…', stale: true };
    case 'connecting':
      return { label: 'Connecting…', stale: true };
    default:
      return { label: 'Offline', stale: true };
  }
}

const ALL_NAV_ITEMS = NAV.flatMap(s => s.items);

const THEME_META: Record<ThemeMode, { icon: string; label: string; next: ThemeMode }> = {
  system: { icon: 'autoMode', label: 'System theme', next: 'light' },
  light: { icon: 'lightMode', label: 'Light theme', next: 'dark' },
  dark: { icon: 'darkMode', label: 'Dark theme', next: 'system' },
};

function ThemeToggle() {
  const { mode, cycle } = useTheme();
  const meta = THEME_META[mode];
  return (
    <button
      type="button"
      className="portal-btn portal-btn--ghost desk-icon-btn w-10 h-10 min-w-0 p-0"
      onClick={cycle}
      aria-label={`${meta.label} — switch to ${meta.next}`}
    >
      <Icon name={meta.icon} />
    </button>
  );
}

const DrawerContent = memo(function DrawerContent({
  isMobile,
  collapsed,
  pathname,
  onToggle,
  onLogout,
}: {
  isMobile: boolean;
  collapsed: boolean;
  pathname: string;
  onToggle: () => void;
  onLogout: () => void;
}) {
  const isRail = collapsed && !isMobile;
  const filteredNav = NAV;
  return (
    <div className="flex flex-col h-full">
      <div className="flex-1 overflow-y-auto overflow-x-hidden">
        <div className={`flex items-center gap-3 transition-all duration-200 ${isRail ? 'justify-center px-2 pt-4 pb-2' : 'px-4 pt-4 pb-2'}`}>
          {isRail ? (
            <button
              onClick={onToggle}
              aria-label="Open sidebar"
              className="desk-icon-btn"
            >
              <PanelLeft size={20} strokeWidth={1.75} />
            </button>
          ) : (
            <motion.div
              initial={{ opacity: 0, x: -10 }}
              animate={{ opacity: 1, x: 0 }}
              className="flex items-center gap-3 w-full"
            >
              <div className="desk-logo-mark">
                P
              </div>
              <div className="min-w-0 flex-1">
                <p className="desk-sidebar-section-label" style={{ padding: 0, margin: 0 }}>Node ops</p>
                <h1 style={{ font: 'var(--type-title)', color: 'var(--desk-text-primary)', margin: 0, letterSpacing: '-0.02em' }}>
                  Warehouse
                </h1>
              </div>
              {!isMobile && (
                <button
                  onClick={onToggle}
                  className="desk-icon-btn"
                  style={{ width: 28, height: 28 }}
                  aria-label="Collapse sidebar"
                >
                  <PanelLeftClose size={16} strokeWidth={1.75} />
                </button>
              )}
            </motion.div>
          )}
        </div>

        {!isRail && (
          <button
            className="desk-sidebar-search"
            onClick={() => {
              document.dispatchEvent(new KeyboardEvent('keydown', { key: 'k', metaKey: true }));
            }}
            aria-label="Search pages (⌘K)"
          >
            <Icon name="search" size={16} className="desk-sidebar-search-icon" />
            <span className="desk-sidebar-search-text">Search…</span>
            <kbd className="desk-sidebar-search-kbd">⌘K</kbd>
          </button>
        )}

        <nav className={`flex flex-col gap-0.5 mt-1 transition-all duration-200 ${isRail ? 'px-1.5' : 'px-2.5'}`}>
          {filteredNav.map((section, si) => (
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
                    transition={{
                      delay: (si * 0.08) + (ii * 0.02),
                      type: 'spring',
                      stiffness: 320,
                      damping: 28
                    }}
                  >
                    <Link
                      href={item.href as any}
                      className={`desk-sidebar-item desk-sidebar-link${active ? ' desk-sidebar-link--active' : ''}`}
                      data-active={active ? 'true' : undefined}
                      title={isRail ? item.label : undefined}
                      aria-label={item.label}
                      aria-current={active ? 'page' : undefined}
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

      <div className={`py-3 transition-all duration-200 ${isRail ? 'px-2' : 'px-3'}`} style={{ borderTop: '1px solid var(--desk-border)' }}>
        <div className={`flex items-center ${isRail ? 'justify-center' : 'gap-2'}`}>
          <ThemeToggle />
          {!isRail && (
            <button
              onClick={onLogout}
              className="desk-sidebar-item flex-1"
              title="Sign Out"
              aria-label="Sign Out"
            >
              <Icon name="logout" size={18} />
              <span>Sign Out</span>
            </button>
          )}
        </div>
        {!isRail && (
          <motion.div
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            className="mt-3 px-3"
          >
            <div className="flex items-center gap-2">
              <span className="desk-live-dot" />
              <span style={{ font: 'var(--type-caption-sm)', color: 'var(--desk-text-secondary)' }}>Single-tenant · Live sync</span>
            </div>
          </motion.div>
        )}
      </div>
    </div>
  );
});

export default function WarehouseShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const isBare = BARE_ROUTES.some(r => pathname === r || pathname.startsWith(r));
  const reducedMotion = useReducedMotion();

  const [collapsed, setCollapsed] = useState(true);
  const [mobileOpen, setMobileOpen] = useState(false);
  const toggleSidebar = useCallback(() => setCollapsed(c => !c), []);

  const [searchOpen, setSearchOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const searchRef = useRef<HTMLInputElement>(null);

  const [profileOpen, setProfileOpen] = useState(false);
  const profileRef = useRef<HTMLDivElement>(null);
  const [hasSession, setHasSession] = useState(false);

  const [notifOpen, setNotifOpen] = useState(false);
  const notifRef = useRef<HTMLDivElement>(null);
  const isConfigured = useMemo(() => {
    const token = readTokenFromCookie();
    const claims = token ? decodeJwtPayload(token) : null;
    return claims?.is_configured === true;
  }, [pathname]);
  const notificationsEnabled = !isBare && isConfigured;
  const { items: notifItems, unreadCount, markRead, markAllRead, wsState } = useNotifications({
    enabled: notificationsEnabled,
  });
  const liveIndicator = liveIndicatorCopy(wsState);

  useEffect(() => { setMobileOpen(false); }, [pathname]);

  useEffect(() => {
    setHasSession(Boolean(readTokenFromCookie()));
  }, []);

  useEffect(() => {
    if (!profileOpen) return;
    const handler = (e: MouseEvent | TouchEvent) => {
      if (profileRef.current && !profileRef.current.contains(e.target as Node)) {
        setProfileOpen(false);
      }
    };
    document.addEventListener('mousedown', handler);
    document.addEventListener('touchstart', handler);
    return () => {
      document.removeEventListener('mousedown', handler);
      document.removeEventListener('touchstart', handler);
    }
  }, [profileOpen]);

  const mobileMenuRef = useRef<HTMLElement>(null);
  useEffect(() => {
    if (!mobileOpen) return;
    const handler = (e: MouseEvent | TouchEvent) => {
      if (mobileMenuRef.current && !mobileMenuRef.current.contains(e.target as Node)) {
        setMobileOpen(false);
      }
    };
    setTimeout(() => {
      document.addEventListener('mousedown', handler);
      document.addEventListener('touchstart', handler);
    }, 10);
    return () => {
      document.removeEventListener('mousedown', handler);
      document.removeEventListener('touchstart', handler);
    }
  }, [mobileOpen]);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setSearchOpen(s => !s);
        setTimeout(() => searchRef.current?.focus(), 100);
      }
      if (e.key === 'Escape') {
        setSearchOpen(false);
        setSearchQuery('');
      }
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, []);

  const handleLogout = useCallback(() => {
    clearSession();
    window.location.href = '/auth/login';
  }, []);

  const breadcrumbs = useMemo(() => buildBreadcrumbs(pathname), [pathname]);

  const searchResults = useMemo(() =>
    searchQuery.trim()
      ? ALL_NAV_ITEMS.filter(item =>
          item.label.toLowerCase().includes(searchQuery.toLowerCase())
        )
    : []
  , [searchQuery]);

  if (isBare) return <>{children}</>;

  return (
    <>
      <motion.aside
        animate={{ width: collapsed ? 72 : 264 }}
        transition={reducedMotion ? { duration: 0 } : { type: 'spring', stiffness: 200, damping: 25 }}
        data-shell-sidebar
        className="hidden md:flex flex-col justify-between shrink-0 overflow-hidden"
        style={{
          borderRight: '1px solid var(--desk-border)',
          background: 'var(--desk-surface)',
        }}
      >
        <DrawerContent isMobile={false} collapsed={collapsed} pathname={pathname} onToggle={toggleSidebar} onLogout={handleLogout} />
      </motion.aside>

      <AnimatePresence>
        {mobileOpen && (
          <>
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              className="fixed inset-0 z-40 md:hidden bg-black/30"
              onClick={() => setMobileOpen(false)}
            />
            <motion.aside
              ref={mobileMenuRef}
              initial={{ x: '-100%' }}
              animate={{ x: 0 }}
              exit={{ x: '-100%' }}
              transition={{ type: 'spring', stiffness: 300, damping: 30 }}
              data-shell-sidebar
              className="fixed top-0 left-0 z-50 h-full flex flex-col md:hidden overflow-hidden"
              style={{
                width: 264,
                borderRight: '1px solid var(--desk-border)',
                background: 'var(--desk-surface)',
              }}
            >
              <DrawerContent isMobile={true} collapsed={collapsed} pathname={pathname} onToggle={toggleSidebar} onLogout={handleLogout} />
            </motion.aside>
          </>
        )}
      </AnimatePresence>

      <div className="flex-1 flex flex-col h-screen overflow-hidden">
        <header className="desk-topbar shrink-0">
          <div className="desk-topbar-left">
            <button
              className="desk-icon-btn md:hidden"
              onClick={() => setMobileOpen(true)}
              aria-label="Open navigation"
            >
              <Icon name="menu" />
            </button>

            <nav className="desk-breadcrumb hidden md:flex" aria-label="Breadcrumb">
              {breadcrumbs.map((crumb, i) => (
                <span key={crumb.href} className="flex items-center gap-2 min-w-0">
                  {i > 0 && <span className="desk-breadcrumb-sep">/</span>}
                  {i === breadcrumbs.length - 1 ? (
                    <span className="desk-breadcrumb-current truncate">{crumb.label}</span>
                  ) : (
                    <Link href={crumb.href as any} className="truncate">{crumb.label}</Link>
                  )}
                </span>
              ))}
            </nav>

            <span className="md:hidden truncate" style={{ font: 'var(--type-title)', color: 'var(--desk-text-primary)' }}>
              {breadcrumbs[breadcrumbs.length - 1]?.label || 'Warehouse Portal'}
            </span>
          </div>

          <div className="desk-topbar-right">
            <button
              className="desk-topbar-search hidden md:flex"
              onClick={() => { setSearchOpen(true); setTimeout(() => searchRef.current?.focus(), 100); }}
              aria-label="Search (Ctrl+K)"
            >
              <Icon name="search" size={16} />
              <span style={{ flex: 1, textAlign: 'left' }}>Search…</span>
              <kbd className="desk-sidebar-search-kbd">⌘K</kbd>
            </button>

            <button
              className="desk-icon-btn md:hidden"
              onClick={() => { setSearchOpen(true); setTimeout(() => searchRef.current?.focus(), 100); }}
              aria-label="Search"
            >
              <Icon name="search" />
            </button>

            <div
              className="desk-live-indicator hidden lg:inline-flex"
              style={liveIndicator.stale ? {
                borderColor: 'color-mix(in oklch, var(--desk-warning) 35%, var(--desk-border))',
                background: 'color-mix(in oklch, var(--desk-warning) 10%, transparent)',
              } : undefined}
            >
              <span
                className="desk-live-dot"
                style={liveIndicator.stale ? {
                  background: 'var(--desk-warning)',
                  animation: 'none',
                } : undefined}
              />
              {liveIndicator.label}
            </div>

            <div className="relative" ref={notifRef}>
              <button
                className="desk-icon-btn"
                aria-label="Notifications"
                onClick={() => setNotifOpen(p => !p)}
              >
                <Icon name="notifications" />
                {unreadCount > 0 && (
                  <span className="desk-notif-badge">
                    {unreadCount > 99 ? '99+' : unreadCount}
                  </span>
                )}
              </button>
              <NotificationPanel
                open={notifOpen}
                onClose={() => setNotifOpen(false)}
                items={notifItems}
                unreadCount={unreadCount}
                onMarkRead={markRead}
                onMarkAllRead={markAllRead}
              />
            </div>

            <div className="relative" ref={profileRef}>
              <button
                onClick={() => setProfileOpen(p => !p)}
                className="desk-profile-pill"
                aria-label="Profile menu"
              >
                <div className="desk-profile-avatar">WH</div>
                <div className="desk-profile-info hidden lg:flex">
                  <span className="desk-profile-name">Warehouse</span>
                  <span className="desk-profile-role">
                    {hasSession ? "Administrator" : "Guest"}
                  </span>
                </div>
              </button>
              {profileOpen && (
                <div className="md-menu" style={{ right: 0, top: 48, minWidth: 220 }}>
                  <div className="px-4 py-3" style={{ borderBottom: '1px solid var(--desk-border)' }}>
                    <p style={{ font: 'var(--type-title)', color: 'var(--desk-text-primary)', margin: 0 }}>Warehouse Admin</p>
                    <p style={{ font: 'var(--type-caption-sm)', color: 'var(--desk-text-tertiary)', margin: '4px 0 0' }}>Single-tenant control plane</p>
                  </div>
                  <Link href={"/profile" as any} className="md-menu-item" onClick={() => setProfileOpen(false)}>
                    <Icon name="warehouse" />
                    <span>Profile</span>
                  </Link>
                  <div style={{ height: 1, background: 'var(--desk-border)', margin: '4px 12px' }} />
                  <button className="md-menu-item" style={{ color: 'var(--desk-danger)' }} onClick={() => { setProfileOpen(false); handleLogout(); }}>
                    <Icon name="logout" />
                    <span>Sign Out</span>
                  </button>
                </div>
              )}
            </div>
          </div>
        </header>

        <AnimatePresence>
          {searchOpen && (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              className="fixed inset-0 z-200 flex items-start justify-center pt-20 bg-black/30"
              onClick={(e) => { if (e.target === e.currentTarget) { setSearchOpen(false); setSearchQuery(''); } }}
            >
              <motion.div
                initial={{ opacity: 0, scale: 0.95, y: -20 }}
                animate={{ opacity: 1, scale: 1, y: 0 }}
                exit={{ opacity: 0, scale: 0.95, y: -20 }}
                className="w-full max-w-lg mx-4 overflow-hidden desk-card"
                style={{ borderRadius: 16 }}
              >
                <div className="md-search-bar" style={{ borderRadius: '16px 16px 0 0', height: 56, borderBottom: '1px solid var(--desk-border)' }}>
                  <Icon name="search" />
                  <input
                    ref={searchRef}
                    type="text"
                    placeholder="Search pages..."
                    value={searchQuery}
                    onChange={e => setSearchQuery(e.target.value)}
                    autoFocus
                  />
                  <kbd
                    className="hidden sm:inline-flex items-center px-1.5 h-5 md-typescale-label-small md-shape-xs text-muted"
                    style={{
                      border: '1px solid var(--desk-border)',
                      fontSize: 10,
                    }}
                  >
                    ESC
                  </kbd>
                </div>
                {searchResults.length > 0 && (
                  <div className="py-1" style={{ borderTop: '1px solid var(--border)' }}>
                    {searchResults.slice(0, 8).map(item => (
                      <Link
                        key={item.href}
                        href={item.href as any}
                        className="md-menu-item active-press"
                        onClick={() => { setSearchOpen(false); setSearchQuery(''); }}
                      >
                        <Icon name={item.icon} />
                        <span>{item.label}</span>
                        <span className="ml-auto md-typescale-label-small text-muted">
                          {item.href}
                        </span>
                      </Link>
                    ))}
                  </div>
                )}
                {searchQuery.trim() && searchResults.length === 0 && (
                  <div className="px-4 py-6 text-center md-typescale-body-small text-muted">
                    No pages match &ldquo;{searchQuery}&rdquo;
                  </div>
                )}
              </motion.div>
            </motion.div>
          )}
        </AnimatePresence>

        <main className="flex-1 overflow-y-auto" style={{ background: 'var(--desk-canvas)' }}>
          <ClientPolicyBanner />
          <AnimatePresence mode="wait" initial={false}>
            <motion.div
              key={pathname}
              initial={{ opacity: 0, y: 8 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -8 }}
              transition={{ duration: 0.25, ease: [0.16, 1, 0.3, 1] }}
              className="h-full"
            >
              {children}
            </motion.div>
          </AnimatePresence>
        </main>
      </div>
    </>
  );
}
