'use client';

import { usePathname, useRouter } from 'next/navigation';
import Link from 'next/link';
import { useEffect, useState, useMemo, useCallback, memo } from 'react';
import { Button } from '@heroui/react';
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

const ALL_NAV_ITEMS = NAV.flatMap((s) => s.items);

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
                <Store size={18} />
              </div>
              <h1 className="md-typescale-title-small truncate flex-1 text-foreground">
                V.O.I.D Hub
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
          {NAV.map((section, si) => (
            <div key={si}>
              {si > 0 && <div className={`md-divider my-1.5 ${isRail ? 'mx-1' : 'mx-3'}`} />}
              {section.label && !isRail && (
                <p className="md-nav-section-label">{section.label}</p>
              )}
              {section.items.map((item) => {
                const active = isActiveRoute(pathname, item.href);
                const Icon = item.icon;
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
                    <Icon size={20} />
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
          <LogOut size={20} />
          {!isRail && <span>Sign Out</span>}
        </button>
        {!isRail && (
          <div className="flex items-center gap-2 mt-3 px-4">
            <div className="w-2 h-2 rounded-full bg-success" />
            <p className="md-typescale-label-small text-muted">v2.0.0 (Retailer)</p>
          </div>
        )}
      </div>
    </>
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

  const handleLogout = useCallback(async () => {
    document.cookie = 'pegasus_retailer_jwt=; Max-Age=0; path=/';
    await clearStoredToken();
    router.push('/');
  }, [router]);

  const breadcrumbs = useMemo(() => buildBreadcrumbs(pathname), [pathname]);

  /* Close mobile drawer on route change */
  useEffect(() => { setMobileOpen(false); }, [pathname]);

  const drawerWidth = collapsed ? 64 : 240;

  return (
    <div className="flex h-dvh bg-background text-foreground overflow-hidden w-full font-sans antialiased">
      
      {/* ── Desktop Sidebar ── */}
      <div
        className="hidden md:flex flex-col shrink-0 border-r border-[var(--border)] bg-background transition-all duration-200 z-10"
        style={{ width: drawerWidth }}
      >
        <DrawerContent
          isMobile={false}
          collapsed={collapsed}
          pathname={pathname}
          onToggle={() => setCollapsed((p) => !p)}
          onLogout={handleLogout}
        />
      </div>

      {/* ── Mobile Drawer Overlay ── */}
      {mobileOpen && (
        <div className="fixed inset-0 z-50 md:hidden">
          <div
            className="absolute inset-0 bg-black/50"
            onClick={() => setMobileOpen(false)}
          />
          <div className="absolute inset-y-0 left-0 w-[280px] bg-background border-r border-[var(--border)] flex flex-col shadow-2xl animate-in slide-in-from-left duration-200">
            <div className="flex items-center justify-between px-4 h-14 border-b border-[var(--border)]">
              <span className="md-typescale-title-small font-semibold text-foreground">Menu</span>
              <Button variant="ghost" isIconOnly onPress={() => setMobileOpen(false)} className="w-8 h-8 min-w-0 text-muted" aria-label="Close menu">
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
          </div>
        </div>
      )}

      {/* ── Main Flow ── */}
      <div className="flex-1 flex flex-col min-w-0 bg-background relative z-0">
        
        {/* Top App Bar */}
        <header className="h-14 shrink-0 flex items-center justify-between px-4 border-b border-[var(--border)] bg-surface/80 backdrop-blur-md sticky top-0 z-20">
          <div className="flex items-center gap-3 overflow-hidden">
            <Button
              variant="ghost"
              isIconOnly
              className="md:hidden w-10 h-10 min-w-0 -ml-2"
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
                    <span className="md-typescale-label-large font-semibold truncate text-foreground">
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

            <span className="md:hidden md-typescale-title-small truncate text-foreground">
              {breadcrumbs[breadcrumbs.length - 1]?.label ?? 'Hub'}
            </span>
          </div>

          <div className="flex items-center gap-2">
            <div className={`hidden md:flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-widest mr-2 border ${
              isConnected
                ? 'bg-success/10 text-success border-success/20'
                : 'bg-danger/10 text-danger border-danger/20'
            }`}>
              <div className={`w-1.5 h-1.5 rounded-full ${isConnected ? 'bg-success animate-pulse' : 'bg-danger'}`} />
              {isConnected ? 'Online' : 'Offline'}
            </div>

            <Button variant="ghost" isIconOnly className="w-9 h-9 min-w-0 text-muted" aria-label="Search">
              <Search size={20} />
            </Button>

            <Button
              variant="ghost"
              isIconOnly
              className="w-9 h-9 min-w-0 text-muted relative"
              aria-label={`Notifications${unreadCount > 0 ? `, ${unreadCount} unread` : ''}`}
              onPress={() => router.push('/notifications')}
            >
              <Bell size={20} />
              {unreadCount > 0 && (
                <div className="absolute -top-1 -right-1 min-w-[18px] h-[18px] px-1 rounded-full bg-danger text-[10px] leading-[18px] text-danger-foreground border border-background text-center font-bold">
                  {unreadCount > 9 ? '9+' : unreadCount}
                </div>
              )}
            </Button>

            {(() => {
              let initials = 'R';
              if (typeof localStorage !== 'undefined') {
                try {
                  const p = JSON.parse(localStorage.getItem('retailer_profile') || '{}');
                  if (p.name) initials = p.name.split(' ').map((w: string) => w[0]).join('').slice(0, 2).toUpperCase();
                  else if (p.company) initials = p.company.split(' ').map((w: string) => w[0]).join('').slice(0, 2).toUpperCase();
                } catch { /* fallback */ }
              }
              return (
                <div className="w-8 h-8 rounded-full bg-accent text-accent-foreground flex items-center justify-center font-bold text-xs border shadow-inner cursor-pointer hover:opacity-90 transition-opacity ml-1">
                  {initials}
                </div>
              );
            })()}
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
