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
import { motion, AnimatePresence } from 'framer-motion';

/* ────────── Navigation Config ────────── */

type NavEntry = { href: string; icon: string; label: string; globalOnly?: boolean; factoryHidden?: boolean };
type NavSection = { label?: string; items: NavEntry[] };

type CommandAction = {
  id: string;
  label: string;
  href: string;
  icon: string;
  keywords: string[];
};

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
      { href: '/notifications', icon: 'notifications', label: 'Notifications' },
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
      { href: '/planning', icon: 'analytics', label: 'Planning Federation', globalOnly: true },
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
const splashDurationMs = 1600;

/* ── Splash Screen (Cinematic) ── */
function SplashScreen({ onComplete }: { onComplete: () => void }) {
  useEffect(() => {
    const timer = window.setTimeout(onComplete, splashDurationMs);
    return () => window.clearTimeout(timer);
  }, [onComplete]);

  return (
    <motion.div
      initial={{ opacity: 1 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0, scale: 1.1, filter: 'blur(10px)' }}
      transition={{ duration: 0.8, ease: [0.16, 1, 0.3, 1] }}
      className="fixed inset-0 z-9999 flex items-center justify-center"
      style={{ background: 'var(--desk-canvas)' }}
    >
      <motion.div
        initial={{ opacity: 0, scale: 0.8, y: 20 }}
        animate={{ opacity: 1, scale: 1, y: 0 }}
        transition={{ duration: 1, ease: [0.16, 1, 0.3, 1] }}
        className="relative z-10 flex flex-col items-center gap-6"
      >
        <div
          className="w-16 h-16 flex items-center justify-center"
          style={{
            background: 'var(--desk-accent)',
            color: 'var(--desk-accent-on)',
            borderRadius: 16,
          }}
        >
          <svg width="32" height="32" viewBox="0 0 24 24" fill="currentColor">
            <path d="M20 4H4v2h16V4zm1 10v-2l-1-5H4l-1 5v2h1v6h10v-6h4v6h2v-6h1zm-9 4H6v-4h6v4z"/>
          </svg>
        </div>
        <motion.div 
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.5, duration: 1 }}
          className="flex flex-col items-center"
        >
          <h2 className="text-2xl font-light tracking-tight" style={{ color: 'var(--desk-text-primary)' }}>PEGASUS</h2>
          <p className="text-[10px] uppercase tracking-[0.3em] mt-1" style={{ color: 'var(--desk-text-tertiary)' }}>Enterprise Hub</p>
        </motion.div>
        
        {/* Loading bar */}
        <div className="w-32 h-0.5 rounded-full mt-4 overflow-hidden" style={{ background: 'var(--desk-border)' }}>
          <motion.div 
            initial={{ x: '-100%' }}
            animate={{ x: '100%' }}
            transition={{ repeat: Infinity, duration: 1.5, ease: "easeInOut" }}
            className="w-full h-full"
            style={{ background: 'var(--desk-accent)' }}
          />
        </div>
      </motion.div>
    </motion.div>
  );
}


/* ── Static nav flat list for search ── */
const ALL_NAV_ITEMS = NAV.flatMap(s => s.items);

const COMMAND_ACTIONS: CommandAction[] = [
  { id: 'orders', label: 'Open Orders — bulk approve/dispatch', href: '/supplier/orders', icon: 'orders', keywords: ['approve', 'dispatch', 'cancel', 'delay', 'bulk'] },
  { id: 'dispatch', label: 'Dispatch Control Room', href: '/supplier/dispatch', icon: 'dispatch', keywords: ['fleet', 'manifest', 'truck'] },
  { id: 'notifications', label: 'Notifications Inbox', href: '/notifications', icon: 'notifications', keywords: ['alerts', 'inbox', 'unread'] },
  { id: 'dashboard', label: 'Demand Dashboard', href: '/supplier/dashboard', icon: 'overview', keywords: ['sla', 'metrics', 'analytics'] },
  { id: 'manifests', label: 'Manifest Exceptions', href: '/supplier/manifest-exceptions', icon: 'dlq', keywords: ['dlq', 'exception'] },
  { id: 'returns', label: 'Returns Queue', href: '/supplier/returns', icon: 'returns', keywords: ['damaged', 'dispute'] },
  { id: 'inventory', label: 'Inventory', href: '/supplier/inventory', icon: 'inventory', keywords: ['stock', 'sku'] },
  { id: 'catalog', label: 'Catalog', href: '/supplier/catalog', icon: 'catalog', keywords: ['products', 'pricing'] },
];

const THEME_META: Record<ThemeMode, { icon: string; label: string; next: ThemeMode }> = {
  system: { icon: 'autoMode', label: 'System theme', next: 'light' },
  light: { icon: 'lightMode', label: 'Light theme', next: 'dark' },
  dark: { icon: 'darkMode', label: 'Dark theme', next: 'system' },
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
      className="desk-btn-ghost w-10 h-10 min-w-0 p-0"
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
    <div className="flex flex-col h-full">
      <div className="flex-1 overflow-y-auto overflow-x-hidden">
        {/* Header */}
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
                <p className="desk-sidebar-section-label" style={{ padding: 0, margin: 0 }}>Enterprise</p>
                <h1 style={{ font: 'var(--type-title)', color: 'var(--desk-text-primary)', margin: 0, letterSpacing: '-0.02em' }}>
                  Pegasus Hub
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

        {/* Search bar */}
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

        {/* Navigation */}
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
                      href={item.href}
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
              <span style={{ font: 'var(--type-caption-sm)', color: 'var(--desk-text-secondary)' }}>v2.0.0 · System ready</span>
            </div>
          </motion.div>
        )}
      </div>
    </div>
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
  const searchResults = useMemo(() => {
    const q = searchQuery.trim().toLowerCase();
    if (!q) return [];
    const pages = ALL_NAV_ITEMS.filter(item =>
      item.label.toLowerCase().includes(q) || item.href.toLowerCase().includes(q)
    );
    const actions = COMMAND_ACTIONS.filter(item =>
      item.label.toLowerCase().includes(q) ||
      item.keywords.some(k => k.includes(q) || q.includes(k))
    );
    const merged = [...actions, ...pages.filter(p => !actions.some(a => a.href === p.href))];
    return merged;
  }, [searchQuery]);

  if (isBare) return <>{children}</>;

  return (
    <>
      <AnimatePresence>
        {!splashDone && <SplashScreen onComplete={dismissSplash} />}
      </AnimatePresence>

      {/* ── Desktop: M3 Navigation Rail / Drawer ─────────────────────── */}
      <motion.aside
        animate={{ width: collapsed ? 72 : 264 }}
        transition={{ type: 'spring', stiffness: 200, damping: 25 }}
        data-shell-sidebar
        className="hidden md:flex flex-col justify-between shrink-0 overflow-hidden"
        style={{
          borderRight: '1px solid var(--desk-border)',
          background: 'var(--desk-surface)',
        }}
      >
        <DrawerContent isMobile={false} collapsed={collapsed} pathname={pathname} isGlobalAdmin={isGlobalAdmin} isFactoryStaff={isFactoryStaff} onToggle={toggleSidebar} onLogout={handleLogout} />
      </motion.aside>

      {/* ── Mobile: Scrim + Slide Drawer ─────────────────────────────── */}
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
              <DrawerContent isMobile={true} collapsed={collapsed} pathname={pathname} isGlobalAdmin={isGlobalAdmin} isFactoryStaff={isFactoryStaff} onToggle={toggleSidebar} onLogout={handleLogout} />
            </motion.aside>
          </>
        )}
      </AnimatePresence>

      {/* ── Main Content Area ── */}
      <div className="flex-1 flex flex-col h-screen overflow-hidden">
        {/* ── Top App Bar ── */}
        <header className="desk-topbar shrink-0">
          {/* Left section */}
          <div className="desk-topbar-left">
            <button
              className="desk-icon-btn md:hidden"
              onClick={() => setMobileOpen(true)}
              aria-label="Open navigation"
            >
              <Icon name="menu" />
            </button>

            {/* Breadcrumbs */}
            <nav className="desk-breadcrumb hidden md:flex" aria-label="Breadcrumb">
              {breadcrumbs.map((crumb, i) => (
                <span key={crumb.href} className="flex items-center gap-2 min-w-0">
                  {i > 0 && <span className="desk-breadcrumb-sep">/</span>}
                  {i === breadcrumbs.length - 1 ? (
                    <span className="desk-breadcrumb-current truncate">{crumb.label}</span>
                  ) : (
                    <Link href={crumb.href} className="truncate">{crumb.label}</Link>
                  )}
                </span>
              ))}
            </nav>

            {/* Mobile: page title only */}
            <span className="md:hidden truncate" style={{ font: 'var(--type-title)', color: 'var(--desk-text-primary)' }}>
              {breadcrumbs[breadcrumbs.length - 1]?.label || 'Supplier Portal'}
            </span>
          </div>

          {/* Right section */}
          <div className="desk-topbar-right">
            {/* Search trigger */}
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

            {/* Live indicator */}
            <div className="desk-live-indicator hidden lg:inline-flex">
              <span className="desk-live-dot" />
              Live
            </div>

            {/* Notifications */}
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

            {/* Profile pill */}
            <div className="relative" ref={profileRef}>
              <button
                onClick={() => setProfileOpen(p => !p)}
                className="desk-profile-pill"
                aria-label="Profile menu"
              >
                <div className="desk-profile-avatar">AS</div>
                <div className="desk-profile-info hidden lg:flex">
                  <span className="desk-profile-name">Admin</span>
                  <span className="desk-profile-role">
                    {supplierRole === 'NODE_ADMIN' ? 'Node Admin' : supplierRole === 'FACTORY_ADMIN' ? 'Factory Admin' : supplierRole === 'FACTORY_PAYLOADER' ? 'Payloader' : 'Supplier'}
                  </span>
                </div>
              </button>
              {profileOpen && (
                <div className="md-menu" style={{ right: 0, top: 48, minWidth: 220 }}>
                  <div className="px-4 py-3" style={{ borderBottom: '1px solid var(--desk-border)' }}>
                    <p style={{ font: 'var(--type-title)', color: 'var(--desk-text-primary)', margin: 0 }}>Admin Supplier</p>
                    <p style={{ font: 'var(--type-caption-sm)', color: 'var(--desk-text-tertiary)', margin: '4px 0 0' }}>admin@void.pegasus.uz</p>
                  </div>
                  <Link href="/supplier/profile" className="md-menu-item" onClick={() => setProfileOpen(false)}>
                    <Icon name="supplier" />
                    <span>Profile</span>
                  </Link>
                  <Link href="/supplier/settings" className="md-menu-item" onClick={() => setProfileOpen(false)}>
                    <Icon name="config" />
                    <span>Settings</span>
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

        {/* ── Search overlay ── */}
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
                    placeholder="Search pages and actions..."
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
                    {searchResults.slice(0, 10).map(item => {
                      const href = 'href' in item ? item.href : (item as NavEntry).href;
                      const label = 'label' in item ? item.label : (item as NavEntry).label;
                      const icon = 'icon' in item ? item.icon : (item as NavEntry).icon;
                      const isAction = COMMAND_ACTIONS.some(a => a.href === href && a.label === label);
                      return (
                      <Link
                        key={`${href}-${label}`}
                        href={href}
                        className="md-menu-item active-press"
                        onClick={() => { setSearchOpen(false); setSearchQuery(''); }}
                      >
                        <Icon name={icon} />
                        <span>{label}</span>
                        <span className="ml-auto md-typescale-label-small text-muted">
                          {isAction ? 'Action' : href}
                        </span>
                      </Link>
                    );})}
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

        {/* ── Page content ── */}
        <main className="flex-1 overflow-y-auto" style={{ background: 'var(--desk-canvas)' }}>
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
