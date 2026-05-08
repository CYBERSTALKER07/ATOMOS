'use client';

import { usePathname, useRouter } from 'next/navigation';
import Link from 'next/link';
import { useEffect, useState, useMemo, useCallback, memo } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import {
  PanelLeftClose, PanelLeft, Menu, Bell, Search,
  LayoutDashboard, ShoppingCart, PackageSearch, Activity, BarChart3, Settings, LogOut,
  Store, X, MapPin, Container
} from 'lucide-react';
import { useWebSocket } from '../lib/ws';
import { useRetailerNotifications } from '../lib/notifications';
import { clearStoredToken } from '../lib/bridge';

/* ────────── Navigation Config ────────── */

type NavEntry = { href: string; icon: React.ElementType; label: string };
type NavSection = { label?: string; items: NavEntry[] };
type RetailerIdentity = { name: string; company: string; initials: string };

const NAV: NavSection[] = [
  {
    items: [
      { href: '/dashboard', icon: LayoutDashboard, label: 'Dashboard' },
      { href: '/orders', icon: ShoppingCart, label: 'Orders' },
      { href: '/tracking', icon: MapPin, label: 'Tracking' },
      { href: '/dock', icon: Container, label: 'Dock' },
      { href: '/catalog', icon: PackageSearch, label: 'Catalog' },
      { href: '/procurement', icon: Activity, label: 'Procurement' },
      { href: '/insights', icon: BarChart3, label: 'Insights' },
    ],
  },
  {
    label: 'System',
    items: [
      { href: '/settings', icon: Settings, label: 'Settings' },
    ],
  },
];

const DEFAULT_IDENTITY: RetailerIdentity = {
  name: "Retailer",
  company: "Workspace",
  initials: "R",
};

function isActiveRoute(pathname: string, href: string): boolean {
  if (href === '/') return pathname === '/';
  return pathname === href || pathname.startsWith(href + '/');
}

/* ── Breadcrumb helper ── */
function buildBreadcrumbs(pathname: string): { label: string; href: string }[] {
  if (pathname === '/') return [{ label: 'Hub', href: '/' }];
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

/* ── Memoized Drawer Content ── */
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
              <div className="desk-logo-mark">
                <Store size={18} />
              </div>
              <div className="min-w-0 flex-1">
                <p className="desk-sidebar-section-label" style={{ padding: 0, margin: 0 }}>Retailer workspace</p>
                <h1 style={{ font: 'var(--type-title)', color: 'var(--desk-text-primary)', margin: 0, letterSpacing: '-0.02em' }}>
                  V.O.I.D Hub
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
                <motion.div 
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  className="desk-sidebar-section-label"
                >
                  {section.label}
                </motion.div>
              )}
              {section.items.map((item) => {
                const active = isActiveRoute(pathname, item.href);
                const ItemIcon = item.icon;
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    prefetch={false}
                    className={`desk-sidebar-item active-press ${active ? 'desk-sidebar-item--accent' : ''}`}
                    title={isRail ? item.label : undefined}
                    aria-label={item.label}
                    style={isRail ? { justifyContent: 'center', padding: '0', height: 42 } : undefined}
                  >
                    <ItemIcon size={18} className="desk-sidebar-item-icon" style={{ color: active ? 'var(--desk-accent)' : undefined }} />
                    {!isRail && (
                      <motion.span 
                        initial={{ opacity: 0, x: -5 }}
                        animate={{ opacity: 1, x: 0 }}
                        className="truncate"
                      >
                        {item.label}
                      </motion.span>
                    )}
                  </Link>
                );
              })}
            </div>
          ))}
        </nav>
      </div>

      {/* Footer */}
      <div className={`py-3 transition-all duration-200 ${isRail ? 'px-2' : 'px-3'}`} style={{ borderTop: '1px solid var(--desk-border)' }}>
        <button
          onClick={onLogout}
          className={`desk-sidebar-item w-full active-press ${isRail ? 'justify-center' : ''}`}
          style={isRail ? { padding: 0 } : undefined}
          title={isRail ? 'Sign Out' : undefined}
          aria-label="Sign Out"
        >
          <LogOut size={18} className="desk-sidebar-item-icon" />
          {!isRail && <span>Sign Out</span>}
        </button>
        {!isRail && (
          <motion.div 
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            className="mt-3 px-3"
          >
            <div className="flex items-center gap-2">
              <span className="desk-live-dot" />
              <span style={{ font: 'var(--type-caption-sm)', color: 'var(--desk-text-secondary)' }}>v2.0.0 · Desktop ready</span>
            </div>
          </motion.div>
        )}
      </div>
    </div>
  );
});

/* ── Shell ── */

export default function RetailerShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const { isConnected } = useWebSocket();
  const { unreadCount } = useRetailerNotifications();
  const [collapsed, setCollapsed] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);
  const [identity, setIdentity] = useState<RetailerIdentity>(DEFAULT_IDENTITY);

  const handleLogout = useCallback(async () => {
    document.cookie = 'pegasus_retailer_jwt=; Max-Age=0; path=/';
    await clearStoredToken();
    router.push('/');
  }, [router]);

  const breadcrumbs = useMemo(() => buildBreadcrumbs(pathname), [pathname]);

  /* Close mobile drawer on route change */
  useEffect(() => { setMobileOpen(false); }, [pathname]);
  useEffect(() => {
    if (typeof localStorage === "undefined") return;
    try {
      const profile = JSON.parse(localStorage.getItem("retailer_profile") || "{}");
      const source = (profile.name || profile.company || "Retailer").trim();
      const initials = source
        .split(" ")
        .filter(Boolean)
        .map((part: string) => part[0])
        .join("")
        .slice(0, 2)
        .toUpperCase() || "R";
      setIdentity({
        name: profile.name || "Retailer",
        company: profile.company || "Workspace",
        initials,
      });
    } catch {
      setIdentity(DEFAULT_IDENTITY);
    }
  }, []);

  return (
    <div className="flex h-dvh overflow-hidden w-full" style={{ background: 'var(--desk-canvas)', color: 'var(--desk-text-primary)' }}>
      
      {/* ── Desktop Sidebar ── */}
      <motion.div
        initial={false}
        animate={{ width: collapsed ? 64 : 240 }}
        transition={{ type: "spring", stiffness: 300, damping: 30 }}
        className="hidden md:flex flex-col shrink-0 overflow-hidden z-10"
        style={{
          borderRight: '1px solid var(--desk-border)',
          background: 'var(--desk-surface)',
        }}
      >
        <DrawerContent
          isMobile={false}
          collapsed={collapsed}
          pathname={pathname}
          onToggle={() => setCollapsed((p) => !p)}
          onLogout={handleLogout}
        />
      </motion.div>

      {/* ── Mobile Drawer Overlay ── */}
      <AnimatePresence>
        {mobileOpen && (
          <div className="fixed inset-0 z-50 md:hidden">
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              className="absolute inset-0 bg-black/60 backdrop-blur-sm"
              onClick={() => setMobileOpen(false)}
            />
            <motion.div 
              initial={{ x: "-100%" }}
              animate={{ x: 0 }}
              exit={{ x: "-100%" }}
              transition={{ type: "spring", stiffness: 300, damping: 30 }}
              className="absolute inset-y-0 left-0 w-[280px] flex flex-col shadow-2xl"
              style={{
                background: 'var(--desk-surface)',
                borderRight: '1px solid var(--desk-border)',
              }}
            >
              <div className="flex items-center justify-between px-4 h-14" style={{ borderBottom: '1px solid var(--desk-border)' }}>
                <span style={{ font: 'var(--type-title)', color: 'var(--desk-text-primary)' }}>Menu</span>
                <button className="desk-icon-btn active-press" onClick={() => setMobileOpen(false)} aria-label="Close menu">
                  <X size={20} />
                </button>
              </div>
              <DrawerContent
                isMobile={true}
                collapsed={false}
                pathname={pathname}
                onToggle={() => {}}
                onLogout={handleLogout}
              />
            </motion.div>
          </div>
        )}
      </AnimatePresence>

      {/* ── Main Flow ── */}
      <div className="flex-1 flex flex-col min-w-0 relative z-0">
        
        {/* Top App Bar */}
        <header className="desk-topbar shrink-0">
          <div className="desk-topbar-left">
            <button
              className="desk-icon-btn md:hidden -ml-2 active-press"
              onClick={() => setMobileOpen(true)}
              aria-label="Open menu"
            >
              <Menu size={24} />
            </button>

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

            <span className="md:hidden truncate" style={{ font: 'var(--type-title)', color: 'var(--desk-text-primary)' }}>
              {breadcrumbs[breadcrumbs.length - 1]?.label ?? 'Hub'}
            </span>
          </div>

          <div className="desk-topbar-right">
            {/* Live indicator */}
            <div className={`desk-live-indicator hidden md:inline-flex ${!isConnected ? 'opacity-50' : ''}`}
              style={!isConnected ? { borderColor: 'var(--desk-border)', background: 'transparent' } : undefined}
            >
              <span className={isConnected ? 'desk-live-dot' : ''} style={!isConnected ? { width: 6, height: 6, borderRadius: '50%', background: 'var(--desk-danger)' } : undefined} />
              {isConnected ? 'Live feed' : 'Offline'}
            </div>

            {/* Search */}
            <button
              className="desk-topbar-search hidden md:flex"
              onClick={() => router.push('/catalog')}
              aria-label="Open catalog search"
            >
              <Search size={16} />
              <span style={{ flex: 1, textAlign: 'left' }}>Search catalog...</span>
              <kbd className="desk-sidebar-search-kbd">⌘K</kbd>
            </button>

            <button className="desk-icon-btn md:hidden active-press" onClick={() => router.push('/catalog')} aria-label="Search catalog">
              <Search size={20} />
            </button>

            {/* Notifications */}
            <button
              className="desk-icon-btn relative active-press"
              aria-label={`Notifications${unreadCount > 0 ? `, ${unreadCount} unread` : ''}`}
              onClick={() => router.push('/notifications')}
            >
              <Bell size={20} />
              <AnimatePresence>
                {unreadCount > 0 && (
                  <motion.span 
                    initial={{ scale: 0 }}
                    animate={{ scale: 1 }}
                    exit={{ scale: 0 }}
                    className="desk-notif-badge"
                  >
                    {unreadCount > 9 ? '9+' : unreadCount}
                  </motion.span>
                )}
              </AnimatePresence>
            </button>

            {/* Profile pill */}
            <div className="desk-profile-pill">
              <div className="desk-profile-avatar">
                {identity.initials}
              </div>
              <div className="desk-profile-info hidden lg:flex">
                <span className="desk-profile-name">{identity.name}</span>
                <span className="desk-profile-role">{identity.company}</span>
              </div>
            </div>
          </div>
        </header>

        {/* Main Content scroll area */}
        <main className="flex-1 overflow-y-auto w-full relative" style={{ background: 'var(--desk-canvas)' }}>
          <div className="absolute inset-0">
            {children}
          </div>
        </main>
      </div>
    </div>
  );
}
