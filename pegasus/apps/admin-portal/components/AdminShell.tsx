'use client';

import { usePathname } from 'next/navigation';
import Link from 'next/link';
import { useEffect, useState, useRef, useMemo, useCallback, memo } from 'react';
import { Button } from '@heroui/react';
import Icon from './Icon';
import { useTheme, type ThemeMode } from './ThemeProvider';
import { PanelLeftClose, PanelLeft } from 'lucide-react';
import NotificationPanel from './NotificationPanel';
import { useNotifications } from '@/lib/useNotifications';
import { useAuth } from '@/hooks/useAuth';

/* ────────── Navigation Config ────────── */

type NavEntry = { href: string; icon: string; label: string; globalOnly?: boolean; factoryHidden?: boolean };
type NavSection = { label?: string; items: NavEntry[] };

const NAV: NavSection[] = [
  {
    items: [
      { href: '/', icon: 'overview', label: 'Overview' },
      { href: '/ledger', icon: 'ledger', label: 'Ledger', globalOnly: true },
      { href: '/reconciliation', icon: 'reconcile', label: 'Reconciliation', globalOnly: true },
      { href: '/treasury', icon: 'treasury', label: 'Treasury', globalOnly: true },
      { href: '/treasury/cash-holdings', icon: 'treasury', label: 'Cash Holdings', globalOnly: true },
      { href: '/treasury/settlement', icon: 'treasury', label: 'Settlement', globalOnly: true },
      { href: '/treasury/refunds', icon: 'treasury', label: 'Refunds', globalOnly: true },
      { href: '/treasury/chargebacks', icon: 'reconcile', label: 'Chargebacks', globalOnly: true },
      { href: '/fleet', icon: 'fleet', label: 'Fleet Radar', factoryHidden: true },
      { href: '/kyc', icon: 'kyc', label: 'KYC', globalOnly: true },
      { href: '/admin/empathy', icon: 'empathy', label: 'Empathy Engine', globalOnly: true },
      { href: '/admin/audit-log', icon: 'reconcile', label: 'Audit Log', globalOnly: true },
      { href: '/analytics', icon: 'analytics', label: 'Intelligence', globalOnly: true },
      { href: '/analytics/advanced', icon: 'analytics', label: 'Advanced Analytics' },
    ],
  },
  {
    label: 'Supplier',
    items: [
      { href: '/supplier/dashboard', icon: 'overview', label: 'Demand Dashboard' },
      { href: '/supplier/analytics', icon: 'analytics', label: 'Analytics' },
      { href: '/supplier/pricing', icon: 'pricing', label: 'Pricing Engine', factoryHidden: true },
      { href: '/supplier/pricing/retailer-overrides', icon: 'pricing', label: 'Retailer Pricing', factoryHidden: true },
      { href: '/supplier/country-overrides', icon: 'global', label: 'Country Overrides', factoryHidden: true },
      { href: '/supplier/catalog', icon: 'catalog', label: 'Catalog' },
      { href: '/supplier/products', icon: 'inventory', label: 'My Products' },
      { href: '/supplier/inventory', icon: 'inventory', label: 'Inventory' },
      { href: '/supplier/orders', icon: 'orders', label: 'Orders' },
      { href: '/supplier/dispatch', icon: 'dispatch', label: 'Dispatch', factoryHidden: true },
      { href: '/supplier/manifests', icon: 'manifests', label: 'Manifests', factoryHidden: true },
      { href: '/supplier/manifest-exceptions', icon: 'dlq', label: 'Manifest Exceptions', factoryHidden: true },
      { href: '/supplier/exceptions/shop-closed', icon: 'dlq', label: 'Shop Closed', factoryHidden: true },
      { href: '/supplier/delivery-zones', icon: 'fleet', label: 'Delivery Zones', factoryHidden: true },
      { href: '/supplier/returns', icon: 'returns', label: 'Returns', factoryHidden: true },
      { href: '/supplier/depot-reconciliation', icon: 'warehouse', label: 'Depot', factoryHidden: true },
      { href: '/supplier/crm', icon: 'crm', label: 'CRM', factoryHidden: true },
      { href: '/supplier/fleet', icon: 'fleet', label: 'Fleet', factoryHidden: true },
      { href: '/supplier/warehouses', icon: 'warehouse', label: 'Warehouses', globalOnly: true },
      { href: '/supplier/factories', icon: 'factory', label: 'Factories', globalOnly: true },
      { href: '/supplier/geo-report', icon: 'global', label: 'Coverage Map', globalOnly: true },
      { href: '/supplier/supply-lanes', icon: 'fleet', label: 'Supply Lanes', globalOnly: true },
      { href: '/supplier/staff', icon: 'warehouse', label: 'Warehouse Staff', factoryHidden: true },
      { href: '/supplier/org', icon: 'supplier', label: 'Org Members', globalOnly: true },
      { href: '/supplier/onboarding', icon: 'kyc', label: 'Onboarding' },
      { href: '/supplier/payment-config', icon: 'payment', label: 'Payment', globalOnly: true },
      { href: '/supplier/profile', icon: 'supplier', label: 'Profile' },
      { href: '/supplier/settings', icon: 'config', label: 'Settings', globalOnly: true },
    ],
  },
  {
    label: 'System',
    items: [
      { href: '/configuration', icon: 'config', label: 'Config', globalOnly: true },
      { href: '/configuration/countries', icon: 'global', label: 'Country Config', globalOnly: true },
      { href: '/dashboard', icon: 'global', label: 'Global Supply', globalOnly: true },
      { href: '/dlq', icon: 'dlq', label: 'DLQ Monitor', globalOnly: true },
      { href: '/admin/control-center', icon: 'config', label: 'Control Center', globalOnly: true },
    ],
  },
];

function isActiveRoute(pathname: string, href: string): boolean {
  if (href === '/') return pathname === '/';
  return pathname === href || pathname.startsWith(href + '/');
}

/* ── Breadcrumb helper ── */
function buildBreadcrumbs(pathname: string): { label: string; href: string }[] {
  if (pathname === '/') return [{ label: 'Overview', href: '/' }];
  const segs = pathname.split('/').filter(Boolean);
  const crumbs: { label: string; href: string }[] = [{ label: 'Home', href: '/' }];
  let path = '';
  for (const seg of segs) {
    path += '/' + seg;
    const label = seg.charAt(0).toUpperCase() + seg.slice(1).replace(/-/g, ' ');
    crumbs.push({ label, href: path });
  }
  return crumbs;
}

// Routes where the navigation drawer should NOT render
const BARE_ROUTES = ['/login', '/signup', '/auth/'];

/* ── Splash Screen (X-style) ── */
function SplashScreen({ onComplete }: { onComplete: () => void }) {
  const [phase, setPhase] = useState<'enter' | 'exit'>('enter');

  useEffect(() => {
    const t1 = setTimeout(() => setPhase('exit'), 1200);
    const t2 = setTimeout(() => onComplete(), 1800);
    return () => { clearTimeout(t1); clearTimeout(t2); };
  }, [onComplete]);

  return (
    <div
      className="fixed inset-0 z-9999 flex items-center justify-center bg-accent"
      style={{
        opacity: phase === 'exit' ? 0 : 1,
        transform: phase === 'exit' ? 'scale(1.15)' : 'scale(1)',
        transition: 'opacity 500ms cubic-bezier(0.4, 0, 0.2, 1), transform 500ms cubic-bezier(0.4, 0, 0.2, 1)',
        pointerEvents: phase === 'exit' ? 'none' : 'auto',
      }}
    >
      <div
        style={{
          animation: 'splash-logo-in 600ms cubic-bezier(0.16, 1, 0.3, 1) forwards',
        }}
      >
        <svg width="72" height="72" viewBox="0 0 24 24" fill="var(--accent-foreground)">
          <path d="M20 4H4v2h16V4zm1 10v-2l-1-5H4l-1 5v2h1v6h10v-6h4v6h2v-6h1zm-9 4H6v-4h6v4z"/>
        </svg>
      </div>
      <style>{`
        @keyframes splash-logo-in {
          0% { opacity: 0; transform: scale(0.6); }
          100% { opacity: 1; transform: scale(1); }
        }
      `}</style>
    </div>
  );
}


/* ── Static nav flat list for search ── */
const ALL_NAV_ITEMS = NAV.flatMap(s => s.items);

/* ── Theme Toggle — cycles system → light → dark → synthwave ── */
const THEME_META: Record<ThemeMode, { icon: string; label: string; next: ThemeMode }> = {
  system: { icon: 'autoMode', label: 'System theme', next: 'light' },
  light:  { icon: 'lightMode', label: 'Light theme', next: 'dark' },
  dark:   { icon: 'darkMode', label: 'Dark theme', next: 'synthwave' },
  synthwave: { icon: 'darkMode', label: 'Synthwave Dark', next: 'system' }, // Recycles darkMode icon, but gives unique label
};

function ThemeToggle() {
  const { mode, cycle } = useTheme();
  const meta = THEME_META[mode];
  return (
    <Button
      variant="ghost"
      isIconOnly
      onPress={cycle}
      aria-label={`${meta.label} — switch to ${meta.next}`}
      className="w-9 h-9 min-w-0 text-muted"
    >
      <Icon name={meta.icon} />
    </Button>
  );
}

/* ── Memoized Drawer Content ── */
const DrawerContent = memo(function DrawerContent({
  isMobile,
  collapsed,
  pathname,
  isGlobalAdmin,
  isFactoryStaff,
  onToggle,
  onLogout,
}: {
  isMobile: boolean;
  collapsed: boolean;
  pathname: string;
  isGlobalAdmin: boolean;
  isFactoryStaff: boolean;
  onToggle: () => void;
  onLogout: () => void;
}) {
  const isRail = collapsed && !isMobile;
  // Filter nav items: globalOnly → GLOBAL_ADMIN only, factoryHidden → hidden for factory staff
  const filteredNav = useMemo(() =>
    NAV.map(section => ({
      ...section,
      items: section.items.filter(item =>
        (!item.globalOnly || isGlobalAdmin) &&
        (!item.factoryHidden || !isFactoryStaff)
      ),
    })).filter(section => section.items.length > 0),
    [isGlobalAdmin, isFactoryStaff],
  );
  return (
    <>
      <div className="flex-1 overflow-y-auto">
        {/* Header */}
        <div className={`flex items-center gap-3 pt-4 pb-2 ${isRail ? 'justify-center px-2' : 'px-4'}`}>
          {isRail ? (
            <Button
              variant="ghost"
              isIconOnly
              onPress={onToggle}
              aria-label="Open sidebar"
              className="w-9 h-9 min-w-0 text-muted"
            >
              <PanelLeft size={20} strokeWidth={1.75} />
            </Button>
          ) : (
            <>
              <div className="w-8 h-8 flex items-center justify-center text-xs font-semibold md-shape-full shrink-0 bg-accent text-accent-foreground">
                L
              </div>
              <h1 className="md-typescale-title-small truncate flex-1 text-foreground">
                Pegasus Hub
              </h1>
              {!isMobile && (
                <Button
                  variant="ghost"
                  isIconOnly
                  onPress={onToggle}
                  className="w-7 h-7 min-w-0 text-muted"
                  aria-label="Collapse sidebar"
                >
                  <PanelLeftClose size={16} strokeWidth={1.75} />
                </Button>
              )}
            </>
          )}
        </div>

        <div className={`md-divider my-1.5 ${isRail ? 'mx-2' : 'mx-4'}`} />

        {/* Navigation */}
        <nav className={`flex flex-col gap-0.5 ${isRail ? 'px-1.5' : 'px-2.5'}`}>
          {filteredNav.map((section, si) => (
            <div key={si}>
              {si > 0 && <div className={`md-divider my-1.5 ${isRail ? 'mx-1' : 'mx-3'}`} />}
              {section.label && !isRail && (
                <p className="md-nav-section-label">{section.label}</p>
              )}
              {section.items.map(item => {
                const active = isActiveRoute(pathname, item.href);
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    prefetch={false}
                    className={`md-nav-item ${active ? 'md-nav-active' : ''}`}
                    data-active={active}
                    title={isRail ? item.label : undefined}
                    aria-label={item.label}
                    style={isRail ? { justifyContent: 'center', padding: '0', height: 42 } : undefined}
                  >
                    <Icon name={item.icon} size={20} />
                    {!isRail && <span className="truncate">{item.label}</span>}
                  </Link>
                );
              })}
            </div>
          ))}
        </nav>
      </div>

      {/* Footer */}
      <div className={`py-3 ${isRail ? 'px-2' : 'px-4'}`} style={{ borderTop: '1px solid var(--border)' }}>
        <button
          onClick={onLogout}
          className={`md-nav-item w-full ${isRail ? 'justify-center' : ''}`}
          style={isRail ? { padding: 0 } : undefined}
          title={isRail ? 'Sign Out' : undefined}
          aria-label="Sign Out"
        >
          <Icon name="logout" />
          {!isRail && <span>Sign Out</span>}
        </button>
        {!isRail && (
          <div className="flex items-center gap-2 mt-3 px-4">
            <div className="w-2 h-2 rounded-full bg-success" />
            <p className="md-typescale-label-small text-muted">v2.0.0</p>
          </div>
        )}
      </div>
    </>
  );
});

export default function AdminShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const isBare = BARE_ROUTES.some(r => pathname === r || pathname.startsWith(r));

  /* ── Splash screen ── */
  const [splashDone, setSplashDone] = useState(false);
  const dismissSplash = useCallback(() => setSplashDone(true), []);

  // Only show splash on first mount of non-bare routes  
  useEffect(() => {
    if (isBare) setSplashDone(true);
  }, [isBare]);

  /* ── Auth state ── */
  // Auth cookie check is read-only — use a ref to avoid re-render cycles
  const isAuthRef = useRef(true);
  const { isGlobalAdmin, isFactoryStaff, supplierRole } = useAuth();

  /* ── Sidebar state ── */
  const [collapsed, setCollapsed] = useState(true);
  const [mobileOpen, setMobileOpen] = useState(false);
  const toggleSidebar = useCallback(() => setCollapsed(c => !c), []);

  /* ── Search bar ── */
  const [searchOpen, setSearchOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const searchRef = useRef<HTMLInputElement>(null);

  /* ── Profile menu ── */
  const [profileOpen, setProfileOpen] = useState(false);
  const profileRef = useRef<HTMLDivElement>(null);

  /* ── Notifications ── */
  const [notifOpen, setNotifOpen] = useState(false);
  const notifRef = useRef<HTMLDivElement>(null);
  const { items: notifItems, unreadCount, markRead, markAllRead } = useNotifications();

  useEffect(() => {
    const cookies = document.cookie;
    isAuthRef.current = cookies.includes('pegasus_admin_jwt=') || cookies.includes('pegasus_supplier_jwt=');
  }, [pathname]);

  useEffect(() => { setMobileOpen(false); }, [pathname]);

  /* ── Close profile on outside click ── */
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

  /* ── Mobile menu outside click ── */
  const mobileMenuRef = useRef<HTMLElement>(null);
  useEffect(() => {
    if (!mobileOpen) return;
    const handler = (e: MouseEvent | TouchEvent) => {
      if (mobileMenuRef.current && !mobileMenuRef.current.contains(e.target as Node)) {
        setMobileOpen(false);
      }
    };
    // small timeout so the open-button click doesn't instantly close it
    setTimeout(() => {
      document.addEventListener('mousedown', handler);
      document.addEventListener('touchstart', handler);
    }, 10);
    return () => {
      document.removeEventListener('mousedown', handler);
      document.removeEventListener('touchstart', handler);
    }
  }, [mobileOpen]);

  /* ── Keyboard shortcut: Cmd/Ctrl+K ── */
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
    document.cookie = 'pegasus_admin_jwt=; path=/; max-age=0; SameSite=Lax';
    document.cookie = 'admin_name=; path=/; max-age=0; SameSite=Lax';
    document.cookie = 'pegasus_supplier_jwt=; path=/; max-age=0; SameSite=Lax';
    document.cookie = 'supplier_name=; path=/; max-age=0; SameSite=Lax';
    window.location.href = '/auth/login';
  }, []);

  const breadcrumbs = useMemo(() => buildBreadcrumbs(pathname), [pathname]);

  /* ── Filtered search results ── */
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
      {!splashDone && <SplashScreen onComplete={dismissSplash} />}

      {/* ── Desktop: M3 Navigation Rail / Drawer ─────────────────────── */}
      <aside
        className="hidden md:flex flex-col justify-between shrink-0 bg-background"
        style={{
          width: collapsed ? 64 : 256,
          borderRight: '1px solid var(--border)',
          transition: `width 300ms cubic-bezier(0.2,0,0,1)`,
        }}
      >
        <DrawerContent isMobile={false} collapsed={collapsed} pathname={pathname} isGlobalAdmin={isGlobalAdmin} isFactoryStaff={isFactoryStaff} onToggle={toggleSidebar} onLogout={handleLogout} />
      </aside>

      {/* ── Mobile: Scrim + Slide Drawer ─────────────────────────────── */}
      {mobileOpen && (
        <div
          className="fixed inset-0 z-40 md:hidden"
          style={{ background: 'var(--backdrop)' }}
          onClick={() => setMobileOpen(false)}
        />
      )}
      <aside
        ref={mobileMenuRef}
        className={`fixed top-0 left-0 z-50 h-full flex flex-col md:hidden bg-background ${
          mobileOpen ? 'translate-x-0' : '-translate-x-full'
        }`}
        style={{
          width: 256,
          borderRight: '1px solid var(--border)',
          transition: `transform 300ms cubic-bezier(0.2,0,0,1)`,
        }}
      >
        <DrawerContent isMobile={true} collapsed={collapsed} pathname={pathname} isGlobalAdmin={isGlobalAdmin} isFactoryStaff={isFactoryStaff} onToggle={toggleSidebar} onLogout={handleLogout} />
      </aside>

      {/* ── Main Content Area ── */}
      <div className="flex-1 flex flex-col h-screen overflow-hidden">
        {/* ── Top App Bar ── */}
        <header
          className="h-14 flex items-center justify-between px-4 shrink-0 gap-4 bg-background"
          style={{ borderBottom: '1px solid var(--border)' }}
        >
          {/* Left section */}
          <div className="flex items-center gap-2 min-w-0">
            <Button
              variant="ghost"
              isIconOnly
              className="md:hidden w-9 h-9 min-w-0 text-muted"
              onPress={() => setMobileOpen(true)}
              aria-label="Open navigation"
            >
              <Icon name="menu" />
            </Button>

            {/* Breadcrumbs */}
            <nav className="hidden md:flex items-center gap-1 min-w-0" aria-label="Breadcrumb">
              {breadcrumbs.map((crumb, i) => (
                <span key={crumb.href} className="flex items-center gap-1 min-w-0">
                  {i > 0 && (
                    <span className="md-typescale-body-small text-muted">/</span>
                  )}
                  {i === breadcrumbs.length - 1 ? (
                    <span className="md-typescale-label-large truncate text-foreground">
                      {crumb.label}
                    </span>
                  ) : (
                    <Link
                      href={crumb.href}
                      className="md-typescale-label-large truncate text-muted"
                    >
                      {crumb.label}
                    </Link>
                  )}
                </span>
              ))}
            </nav>

            {/* Mobile: page title only */}
            <span className="md:hidden md-typescale-title-small truncate text-foreground">
              {breadcrumbs[breadcrumbs.length - 1]?.label || 'Supplier Portal'}
            </span>
          </div>

          {/* Right section */}
          <div className="flex items-center gap-1">
            {/* Theme toggle */}
            <ThemeToggle />

            {/* Search trigger */}
            <Button
              variant="ghost"
              isIconOnly
              className="w-9 h-9 min-w-0 text-muted"
              onPress={() => { setSearchOpen(true); setTimeout(() => searchRef.current?.focus(), 100); }}
              aria-label="Search (Ctrl+K)"
            >
              <Icon name="search" />
            </Button>

            {/* Notifications */}
            <div className="relative" ref={notifRef}>
              <Button
                variant="ghost"
                isIconOnly
                className="w-9 h-9 min-w-0 text-muted relative"
                aria-label="Notifications"
                onPress={() => setNotifOpen(p => !p)}
              >
                <Icon name="notifications" />
                {unreadCount > 0 && (
                  <span
                    style={{
                      position: 'absolute',
                      top: 4,
                      right: 4,
                      width: unreadCount > 9 ? 18 : 16,
                      height: 16,
                      borderRadius: 8,
                      background: 'var(--color-md-error)',
                      color: 'var(--color-md-on-error)',
                      fontSize: 10,
                      fontWeight: 600,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                    }}
                  >
                    {unreadCount > 99 ? '99+' : unreadCount}
                  </span>
                )}
              </Button>
              <NotificationPanel
                open={notifOpen}
                onClose={() => setNotifOpen(false)}
                items={notifItems}
                unreadCount={unreadCount}
                onMarkRead={markRead}
                onMarkAllRead={markAllRead}
              />
            </div>

            {/* Profile */}
            <div className="relative" ref={profileRef}>
              <div className="flex items-center gap-2">
                {supplierRole === 'NODE_ADMIN' && (
                  <span className="hidden md:inline-flex md-typescale-label-small px-2 py-0.5 md-shape-sm"
                    style={{ background: 'var(--color-md-tertiary-container)', color: 'var(--color-md-on-tertiary-container)' }}>
                    Node Admin
                  </span>
                )}
                {supplierRole === 'FACTORY_ADMIN' && (
                  <span className="hidden md:inline-flex md-typescale-label-small px-2 py-0.5 md-shape-sm"
                    style={{ background: 'var(--color-md-secondary-container)', color: 'var(--color-md-on-secondary-container)' }}>
                    Factory Admin
                  </span>
                )}
                {supplierRole === 'FACTORY_PAYLOADER' && (
                  <span className="hidden md:inline-flex md-typescale-label-small px-2 py-0.5 md-shape-sm"
                    style={{ background: 'var(--color-md-secondary-container)', color: 'var(--color-md-on-secondary-container)' }}>
                    Payloader
                  </span>
                )}
                <button
                  onClick={() => setProfileOpen(p => !p)}
                  className="w-9 h-9 flex items-center justify-center md-typescale-label-medium md-shape-full cursor-pointer bg-accent text-accent-foreground"
                  aria-label="Profile menu"
                >
                  AS
                </button>
              </div>
              {profileOpen && (
                <div className="md-menu" style={{ right: 0, top: 44, minWidth: 200 }}>
                  <div className="px-3 py-3" style={{ borderBottom: '1px solid var(--border)' }}>
                    <p className="md-typescale-title-small text-foreground">Admin Supplier</p>
                    <p className="md-typescale-body-small text-muted">admin@void.pegasus.uz</p>
                  </div>
                  <Link
                    href="/supplier/profile"
                    className="md-menu-item"
                    onClick={() => setProfileOpen(false)}
                  >
                    <Icon name="supplier" />
                    <span>Profile</span>
                  </Link>
                  <Link
                    href="/supplier/settings"
                    className="md-menu-item"
                    onClick={() => setProfileOpen(false)}
                  >
                    <Icon name="config" />
                    <span>Settings</span>
                  </Link>
                  <div className="md-divider mx-3 my-1" />
                  <button className="md-menu-item text-danger" onClick={() => { setProfileOpen(false); handleLogout(); }}>
                    <Icon name="logout" />
                    <span>Sign Out</span>
                  </button>
                </div>
              )}
            </div>
          </div>
        </header>

        {/* ── Search overlay ── */}
        {searchOpen && (
          <div
            className="fixed inset-0 z-200 flex items-start justify-center pt-20"
            style={{ background: 'var(--backdrop)' }}
            onClick={(e) => { if (e.target === e.currentTarget) { setSearchOpen(false); setSearchQuery(''); } }}
          >
            <div
              className="w-full max-w-lg mx-4 md-shape-xl overflow-hidden bg-surface"
              style={{
                boxShadow: '0 8px 12px 6px rgba(0,0,0,0.15), 0 4px 4px rgba(0,0,0,0.3)',
              }}
            >
              <div className="md-search-bar" style={{ borderRadius: '28px 28px 0 0', height: 56 }}>
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
                    border: '1px solid var(--border)',
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
                      href={item.href}
                      className="md-menu-item"
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
            </div>
          </div>
        )}

        {/* ── Page content ── */}
        <main className="flex-1 overflow-y-auto bg-background">
          {children}
        </main>
      </div>
    </>
  );
}
