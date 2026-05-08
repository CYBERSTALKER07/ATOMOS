'use client';

import { usePathname, useRouter } from 'next/navigation';
import Link from 'next/link';
import { useEffect, useState, useMemo, useCallback, memo } from 'react';
import { Button } from '@heroui/react';
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
        <div className={`flex items-center gap-3 pt-4 pb-2 transition-all duration-300 ${isRail ? 'justify-center px-2' : 'px-4'}`}>
          {isRail ? (
            <Button
              variant="ghost"
              isIconOnly
              onPress={onToggle}
              aria-label="Open sidebar"
              className="w-9 h-9 min-w-0 text-muted hover-lift active-press"
            >
              <PanelLeft size={20} strokeWidth={1.75} />
            </Button>
          ) : (
            <motion.div 
              initial={{ opacity: 0, x: -10 }}
              animate={{ opacity: 1, x: 0 }}
              className="flex items-center gap-3 w-full"
            >
              <div className="w-8 h-8 flex items-center justify-center text-xs font-semibold md-shape-full shrink-0 bg-accent text-accent-foreground shadow-lg shadow-desk-accent/20">
                <Store size={18} />
              </div>
              <div className="min-w-0 flex-1">
                <p className="md-typescale-label-small uppercase tracking-[0.16em] text-muted">Retailer workspace</p>
                <h1 className="md-typescale-title-small truncate text-foreground font-bold">V.O.I.D Hub</h1>
              </div>
              {!isMobile && (
                <Button
                  variant="ghost"
                  isIconOnly
                  onPress={onToggle}
                  className="w-7 h-7 min-w-0 text-muted hover-lift active-press"
                  aria-label="Collapse sidebar"
                >
                  <PanelLeftClose size={16} strokeWidth={1.75} />
                </Button>
              )}
            </motion.div>
          )}
        </div>

        <div className={`md-divider my-1.5 transition-all duration-300 ${isRail ? 'mx-2' : 'mx-4'}`} />

        {/* Navigation */}
        <nav className={`flex flex-col gap-0.5 transition-all duration-300 ${isRail ? 'px-1.5' : 'px-2.5'}`}>
          {NAV.map((section, si) => (
            <div key={si}>
              {si > 0 && <div className={`md-divider my-1.5 transition-all duration-300 ${isRail ? 'mx-1' : 'mx-3'}`} />}
              {section.label && !isRail && (
                <motion.p 
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  className="md-nav-section-label"
                >
                  {section.label}
                </motion.p>
              )}
              {section.items.map((item) => {
                const active = isActiveRoute(pathname, item.href);
                const Icon = item.icon;
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    prefetch={false}
                    className={`md-nav-item hover-lift active-press ${active ? 'md-nav-active' : ''}`}
                    data-active={active}
                    title={isRail ? item.label : undefined}
                    aria-label={item.label}
                    style={isRail ? { justifyContent: 'center', padding: '0', height: 42 } : undefined}
                  >
                    <Icon size={20} className={active ? "text-accent-foreground" : "text-muted"} />
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
      <div className={`py-3 transition-all duration-300 ${isRail ? 'px-2' : 'px-4'}`} style={{ borderTop: '1px solid var(--border)' }}>
        <button
          onClick={onLogout}
          className={`md-nav-item w-full hover-lift active-press ${isRail ? 'justify-center' : ''}`}
          style={isRail ? { padding: 0 } : undefined}
          title={isRail ? 'Sign Out' : undefined}
          aria-label="Sign Out"
        >
          <LogOut size={20} className="text-muted" />
          {!isRail && <span className="text-muted">Sign Out</span>}
        </button>
        {!isRail && (
          <motion.div 
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            className="mt-3 rounded-2xl border border-[var(--border)] bg-surface-subtle px-4 py-3"
          >
            <div className="flex items-center gap-2">
              <div className="w-2 h-2 rounded-full bg-success animate-pulse" />
              <p className="md-typescale-label-small text-foreground font-semibold">Desktop workspace ready</p>
            </div>
            <p className="mt-1 md-typescale-label-small text-muted leading-tight">v2.0.0 · notifications, payments, and live tracking enabled</p>
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
  const currentPageLabel = breadcrumbs[breadcrumbs.length - 1]?.label ?? "Hub";

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
    <div className="flex h-dvh bg-background text-foreground overflow-hidden w-full font-sans antialiased">
      
      {/* ── Desktop Sidebar ── */}
      <motion.div
        initial={false}
        animate={{ width: collapsed ? 64 : 240 }}
        transition={{ type: "spring", stiffness: 300, damping: 30 }}
        className="hidden md:flex flex-col shrink-0 border-r border-[var(--border)] bg-background z-10 overflow-hidden"
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
              className="absolute inset-y-0 left-0 w-[280px] bg-background border-r border-[var(--border)] flex flex-col shadow-2xl"
            >
              <div className="flex items-center justify-between px-4 h-14 border-b border-[var(--border)]">
                <span className="md-typescale-title-small font-bold text-foreground">Menu</span>
                <Button variant="ghost" isIconOnly onPress={() => setMobileOpen(false)} className="w-8 h-8 min-w-0 text-muted hover-lift active-press" aria-label="Close menu">
                  <X size={20} />
                </Button>
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
      <div className="flex-1 flex flex-col min-w-0 bg-background relative z-0">
        
        {/* Top App Bar */}
        <header className="h-16 shrink-0 flex items-center justify-between px-4 border-b border-[var(--border)] bg-surface/80 backdrop-blur-md sticky top-0 z-20">
          <div className="flex items-center gap-3 overflow-hidden">
            <Button
              variant="ghost"
              isIconOnly
              className="md:hidden w-10 h-10 min-w-0 -ml-2 hover-lift active-press"
              onPress={() => setMobileOpen(true)}
              aria-label="Open menu"
            >
              <Menu size={24} />
            </Button>

            <nav className="hidden md:flex items-center gap-1.5 min-w-0" aria-label="Breadcrumb">
              {breadcrumbs.map((crumb, i) => (
                <span key={crumb.href} className="flex items-center gap-1.5 min-w-0">
                  {i > 0 && <span className="text-muted opacity-40">/</span>}
                  {i === breadcrumbs.length - 1 ? (
                    <span className="md-typescale-label-large font-bold truncate text-foreground">
                      {crumb.label}
                    </span>
                  ) : (
                    <Link href={crumb.href} className="md-typescale-label-large truncate text-muted hover:text-foreground transition-colors">
                      {crumb.label}
                    </Link>
                  )}
                </span>
              ))}
            </nav>

            <span className="md:hidden md-typescale-title-small truncate text-foreground font-bold">
              {breadcrumbs[breadcrumbs.length - 1]?.label ?? 'Hub'}
            </span>
          </div>

          <div className="flex items-center gap-2">
            <div className="hidden xl:flex flex-col mr-2">
              <span className="md-typescale-label-small uppercase tracking-[0.14em] text-muted font-semibold">Current workspace</span>
              <span className="md-typescale-title-small truncate text-foreground font-bold">{currentPageLabel}</span>
            </div>
            <div className={`hidden md:flex items-center gap-1.5 px-3 py-1 rounded-full text-[10px] font-bold uppercase tracking-widest mr-1 border transition-all ${
              isConnected
                ? 'bg-success/10 text-success border-success/20'
                : 'bg-danger/10 text-danger border-danger/20'
            }`}>
              <div className={`w-1.5 h-1.5 rounded-full ${isConnected ? 'bg-success animate-pulse' : 'bg-danger'}`} />
              {isConnected ? 'Live feed' : 'Offline'}
            </div>

            <Button
              variant="ghost"
              className="hidden md:flex h-10 min-w-[200px] items-center justify-between rounded-full border border-[var(--border)] bg-surface-subtle px-4 text-muted hover-lift active-press"
              aria-label="Open catalog search"
              onPress={() => router.push('/catalog')}
            >
              <span className="flex items-center gap-2">
                <Search size={16} />
                <span className="md-typescale-label-large">Search catalog...</span>
              </span>
              <kbd className="px-1.5 py-0.5 rounded border border-[var(--border)] bg-background text-[10px] font-bold opacity-60">⌘K</kbd>
            </Button>

            <Button variant="ghost" isIconOnly className="md:hidden w-9 h-9 min-w-0 text-muted hover-lift active-press" aria-label="Search catalog" onPress={() => router.push('/catalog')}>
              <Search size={20} />
            </Button>

            <Button
              variant="ghost"
              isIconOnly
              className="w-9 h-9 min-w-0 text-muted relative hover-lift active-press"
              aria-label={`Notifications${unreadCount > 0 ? `, ${unreadCount} unread` : ''}`}
              onPress={() => router.push('/notifications')}
            >
              <Bell size={20} />
              <AnimatePresence>
                {unreadCount > 0 && (
                  <motion.div 
                    initial={{ scale: 0 }}
                    animate={{ scale: 1 }}
                    exit={{ scale: 0 }}
                    className="absolute -top-1 -right-1 min-w-[18px] h-[18px] px-1 rounded-full bg-danger text-[10px] leading-[18px] text-danger-foreground border-2 border-background text-center font-bold"
                  >
                    {unreadCount > 9 ? '9+' : unreadCount}
                  </motion.div>
                )}
              </AnimatePresence>
            </Button>

            <div className="ml-1 flex items-center gap-3 rounded-full border border-[var(--border)] bg-surface-subtle px-2.5 py-1.5 hover:border-desk-accent/40 transition-colors cursor-pointer group">
              <div className="w-8 h-8 rounded-full bg-accent text-accent-foreground flex items-center justify-center font-bold text-xs border shadow-lg shadow-desk-accent/20 group-hover:scale-105 transition-transform">
                {identity.initials}
              </div>
              <div className="hidden lg:flex flex-col min-w-0 max-w-[140px]">
                <span className="md-typescale-label-large truncate text-foreground font-bold leading-none">{identity.name}</span>
                <span className="md-typescale-label-small truncate text-muted mt-1 leading-none">{identity.company}</span>
              </div>
            </div>
          </div>
        </header>

        {/* Main Content scroll area */}
        <main className="flex-1 overflow-y-auto w-full relative">
          <div className="absolute inset-0">
            {children}
          </div>
        </main>
      </div>
    </div>
  );
}
