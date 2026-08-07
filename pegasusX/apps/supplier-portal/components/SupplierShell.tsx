'use client';

import { usePathname } from 'next/navigation';
import Link from 'next/link';
import { useEffect, useState, useRef, useMemo, useCallback, memo } from 'react';
import { Button } from '@heroui/react';
import Icon from './Icon';
import { useTheme, type ThemeMode } from './ThemeProvider';
import { PanelLeftClose, PanelLeft } from 'lucide-react';
import NotificationPanel from './NotificationPanel';
import ClientPolicyBanner from './ClientPolicyBanner';
import LanguageSwitcher from './LanguageSwitcher';
import { useNotifications } from '@/lib/useNotifications';
import { clearSession, readTokenFromCookie } from '@/lib/auth';
import { usePortalT } from '@/lib/i18n';
import { motion, AnimatePresence, useReducedMotion } from "framer-motion";

type NavEntry = { href: string; icon: string; labelKey: string; globalOnly?: boolean; factoryHidden?: boolean };
type NavSection = { labelKey?: string; items: NavEntry[] };

const NAV: NavSection[] = [
  {
    items: [{ href: "/dashboard", icon: "overview", labelKey: "portal.nav.overview" }],
  },
  {
    labelKey: "portal.nav.section.operations",
    items: [
      { href: "/orders", icon: "orders", labelKey: "portal.nav.orders" },
      { href: "/dispatch", icon: "dispatch", labelKey: "portal.nav.dispatch" },
      { href: "/ops/map", icon: "dispatch", labelKey: "portal.nav.live_ops_map" },
      { href: "/manifests", icon: "manifests", labelKey: "portal.nav.manifests" },
      { href: "/fleet", icon: "fleet", labelKey: "portal.nav.fleet" },
      { href: "/fleet/orders", icon: "orders", labelKey: "portal.nav.fleet_orders" },
      { href: "/operations", icon: "dispatch", labelKey: "portal.nav.operations" },
      { href: "/operations/replenishment-policies", icon: "inventory", labelKey: "portal.nav.replenishment_policies" },
      { href: "/replenishment/suggestions", icon: "inventory", labelKey: "portal.nav.reorder_suggestions" },
      { href: "/exceptions", icon: "warning", labelKey: "portal.nav.exceptions" },
      { href: "/activity", icon: "overview", labelKey: "portal.nav.activity" },
    ],
  },
  {
    labelKey: "portal.nav.section.catalog",
    items: [
      { href: "/inventory", icon: "inventory", labelKey: "portal.nav.inventory" },
      { href: "/inventory/import", icon: "inventory", labelKey: "portal.nav.import_csv" },
      { href: "/catalog", icon: "catalog", labelKey: "portal.nav.catalog" },
      { href: "/pricing", icon: "pricing", labelKey: "portal.nav.pricing" },
      { href: "/pricing/retailer-overrides", icon: "pricing", labelKey: "portal.nav.retailer_overrides" },
      { href: "/promotions", icon: "pricing", labelKey: "portal.nav.promotions" },
    ],
  },
  {
    labelKey: "portal.nav.section.network",
    items: [
      { href: "/topology", icon: "topology", labelKey: "portal.nav.topology" },
      { href: "/factories", icon: "factory", labelKey: "portal.nav.factories" },
      { href: "/warehouses", icon: "warehouse", labelKey: "portal.nav.warehouses" },
      { href: "/delivery-zones", icon: "global", labelKey: "portal.nav.delivery_zones" },
      { href: "/supply-lanes", icon: "fleet", labelKey: "portal.nav.supply_lanes" },
      { href: "/geo-report", icon: "global", labelKey: "portal.nav.geo_report" },
    ],
  },
  {
    labelKey: "portal.nav.section.analytics",
    items: [
      { href: "/analytics", icon: "overview", labelKey: "portal.nav.analytics" },
      { href: "/control-tower", icon: "global", labelKey: "portal.nav.control_tower" },
      { href: "/analytics/demand", icon: "overview", labelKey: "portal.nav.demand_forecast" },
      { href: "/analytics/demand/flywheel", icon: "campaign", labelKey: "portal.nav.pos_flywheel" },
      { href: "/analytics/route-performance", icon: "overview", labelKey: "portal.nav.route_performance" },
      { href: "/analytics/demand/signals", icon: "campaign", labelKey: "portal.nav.demand_signals" },
      { href: "/demand/payday-calendar", icon: "campaign", labelKey: "portal.nav.payday_calendar" },
      { href: "/analytics/knowledge-graph", icon: "topology", labelKey: "portal.nav.knowledge_graph" },
      { href: "/ai/recommendations", icon: "overview", labelKey: "portal.nav.ai_recommendations" },
    ],
  },
  {
    labelKey: "portal.nav.section.finance",
    items: [
      { href: "/treasury", icon: "treasury", labelKey: "portal.nav.treasury" },
      { href: "/treasury/cash-reconciliations", icon: "reconcile", labelKey: "portal.nav.cash_reconciliations" },
      { href: "/finance/credit-notes", icon: "returns", labelKey: "portal.nav.credit_notes" },
      { href: "/reconciliation", icon: "reconcile", labelKey: "portal.nav.reconciliation" },
      { href: "/compliance", icon: "warning", labelKey: "portal.nav.compliance_audit" },
      { href: "/payments", icon: "payment", labelKey: "portal.nav.payments" },
      { href: "/earnings", icon: "pricing", labelKey: "portal.nav.earnings" },
      { href: "/credit/policy", icon: "treasury", labelKey: "portal.nav.credit_policy" },
      { href: "/credit/collections", icon: "treasury", labelKey: "portal.nav.credit_collections" },
      { href: "/credit/admin-disable", icon: "warning", labelKey: "portal.nav.credit_admin_disable" },
      { href: "/chargebacks", icon: "warning", labelKey: "portal.nav.chargebacks" },
      { href: "/chargebacks/claims", icon: "warning", labelKey: "portal.nav.claim_chargebacks" },
      { href: "/ledger", icon: "orders", labelKey: "portal.nav.ledger" },
    ],
  },
  {
    labelKey: "portal.nav.section.settings",
    items: [
      { href: "/profile", icon: "supplier", labelKey: "portal.nav.profile" },
      { href: "/settings/tax-regimes", icon: "overview", labelKey: "portal.nav.tax_regimes" },
      { href: "/settings/fx-rates", icon: "overview", labelKey: "portal.nav.fx_rates" },
      { href: "/settings/planning", icon: "overview", labelKey: "portal.nav.planning" },
      { href: "/settings/return-policy", icon: "returns", labelKey: "portal.nav.return_policy" },
      { href: "/settings/notification-preferences", icon: "overview", labelKey: "portal.nav.notifications" },
      { href: "/settings/integrations", icon: "overview", labelKey: "portal.nav.integrations" },
      { href: "/settings/segmentation", icon: "overview", labelKey: "portal.nav.segmentation" },
      { href: "/settings/playbooks", icon: "overview", labelKey: "portal.nav.playbooks" },
      { href: "/org-fleet", icon: "person-add", labelKey: "portal.nav.org_fleet" },
      { href: "/returns", icon: "returns", labelKey: "portal.nav.returns" },
    ],
  },
];

const BARE_ROUTES = ["/auth/", "/setup/"];

const ALL_NAV_ITEMS = NAV.flatMap(s => s.items);

function isActiveRoute(pathname: string, href: string): boolean {
  if (href === "/dashboard") {
    return pathname === "/dashboard" || pathname === "/";
  }
  return pathname === href || pathname.startsWith(`${href}/`);
}

function buildBreadcrumbs(
  pathname: string,
  t: (key: string) => string,
): { label: string; href: string }[] {
  if (pathname === "/dashboard" || pathname === "/") {
    return [{ label: t("portal.nav.overview"), href: "/dashboard" }];
  }
  const crumbs: { label: string; href: string }[] = [
    { label: t("portal.chrome.home"), href: "/dashboard" },
  ];
  let path = "";
  for (const seg of pathname.split("/").filter(Boolean)) {
    path += `/${seg}`;
    const match = ALL_NAV_ITEMS.find((item) => item.href === path);
    crumbs.push({
      label: match ? t(match.labelKey) : seg.charAt(0).toUpperCase() + seg.slice(1).replace(/-/g, " "),
      href: path,
    });
  }
  return crumbs;
}

const THEME_META: Record<ThemeMode, { icon: string; labelKey: string; next: ThemeMode }> = {
  system: { icon: 'autoMode', labelKey: 'portal.chrome.theme_system', next: 'light' },
  light: { icon: 'lightMode', labelKey: 'portal.chrome.theme_light', next: 'dark' },
  dark: { icon: 'darkMode', labelKey: 'portal.chrome.theme_dark', next: 'system' },
};

function ThemeToggle() {
  const { mode, cycle } = useTheme();
  const t = usePortalT();
  const meta = THEME_META[mode];
  return (
    <Button
      variant="ghost"
      isIconOnly
      onPress={cycle}
      aria-label={t(meta.labelKey)}
      className="desk-btn-ghost w-10 h-10 min-w-0 p-0"
    >
      <Icon name={meta.icon} />
    </Button>
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
  const t = usePortalT();
  return (
    <div className="flex flex-col h-full">
      <div className="flex-1 overflow-y-auto overflow-x-hidden">
        <div className={`flex items-center gap-3 transition-all duration-200 ${isRail ? 'justify-center px-2 pt-4 pb-2' : 'px-4 pt-4 pb-2'}`}>
          {isRail ? (
            <button
              onClick={onToggle}
              aria-label={t("portal.chrome.open_sidebar")}
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
                <p className="desk-sidebar-section-label" style={{ padding: 0, margin: 0 }}>{t("supplier_portal.residual.text.pegasusx")}</p>
                <h1 style={{ font: 'var(--type-title)', color: 'var(--desk-text-primary)', margin: 0, letterSpacing: '-0.02em' }}>
                  {t("portal.chrome.supplier_hub")}
                </h1>
              </div>
              {!isMobile && (
                <button
                  onClick={onToggle}
                  className="desk-icon-btn"
                  style={{ width: 28, height: 28 }}
                  aria-label={t("portal.chrome.collapse_sidebar")}
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
            aria-label={t("portal.chrome.search")}
          >
            <Icon name="search" size={16} className="desk-sidebar-search-icon" />
            <span className="desk-sidebar-search-text">{t("portal.chrome.search")}</span>
            <kbd className="desk-sidebar-search-kbd">⌘K</kbd>
          </button>
        )}

        <nav className={`flex flex-col gap-0.5 mt-1 transition-all duration-200 ${isRail ? 'px-1.5' : 'px-2.5'}`}>
          {filteredNav.map((section, si) => (
            <div key={si}>
              {si > 0 && <div style={{ height: 1, background: 'var(--desk-border)', margin: isRail ? '8px 4px' : '8px 12px' }} />}
              {section.labelKey && !isRail && (
                <div className="desk-sidebar-section-label">{t(section.labelKey)}</div>
              )}
              {section.items.map((item, ii) => {
                const active = isActiveRoute(pathname, item.href);
                const label = t(item.labelKey);
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
                      className={`desk-sidebar-item ${active ? 'desk-sidebar-item--accent' : ''}`}
                      title={isRail ? label : undefined}
                      aria-label={label}
                      style={isRail ? { justifyContent: 'center', padding: '0', height: 44 } : undefined}
                    >
                      <Icon name={item.icon} size={18} className="desk-sidebar-item-icon" />
                      {!isRail && <span className="truncate">{label}</span>}
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
              title={t("common.action.sign_out")}
              aria-label={t("common.action.sign_out")}
            >
              <Icon name="logout" size={18} />
              <span>{t("common.action.sign_out")}</span>
            </button>
          )}
        </div>
        {!isRail && <div className="mt-2"><LanguageSwitcher /></div>}
        {!isRail && (
          <motion.div
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            className="mt-3 px-3"
          >
            <div className="flex items-center gap-2">
              <span className="desk-live-dot" />
              <span style={{ font: 'var(--type-caption-sm)', color: 'var(--desk-text-secondary)' }}>{t("supplier_portal.supplier_shell.text.single_tenant_system_ready")}</span>
            </div>
          </motion.div>
        )}
      </div>
    </div>
  );
});

export default function SupplierShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const isBare = BARE_ROUTES.some(r => pathname === r || pathname.startsWith(r));
  if (isBare) return <>{children}</>;
  return <SupplierAppChrome>{children}</SupplierAppChrome>;
}

function SupplierAppChrome({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const reduceMotion = useReducedMotion();

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
  const { items: notifItems, unreadCount, markRead, markAllRead } = useNotifications();
  const t = usePortalT();

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

  const breadcrumbs = useMemo(() => buildBreadcrumbs(pathname, t), [pathname, t]);

  const searchResults = useMemo(() =>
    searchQuery.trim()
      ? ALL_NAV_ITEMS.filter(item =>
          t(item.labelKey).toLowerCase().includes(searchQuery.toLowerCase())
        )
    : []
  , [searchQuery, t]);

  return (
    <>
      <motion.aside
        animate={{ width: collapsed ? 72 : 264 }}
        transition={reduceMotion ? { duration: 0 } : { type: "spring", stiffness: 200, damping: 25 }}
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
              aria-label={t("supplier_portal.supplier_shell.text.open_navigation")}
            >
              <Icon name="menu" />
            </button>

            <nav className="desk-breadcrumb hidden md:flex" aria-label={t("supplier_portal.supplier_shell.text.breadcrumb")}>
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
              {breadcrumbs[breadcrumbs.length - 1]?.label || 'Supplier Portal'}
            </span>
          </div>

          <div className="desk-topbar-right">
            <button
              className="desk-topbar-search hidden md:flex"
              onClick={() => { setSearchOpen(true); setTimeout(() => searchRef.current?.focus(), 100); }}
              aria-label={t("supplier_portal.supplier_shell.text.search_ctrl_k")}
            >
              <Icon name="search" size={16} />
              <span style={{ flex: 1, textAlign: 'left' }}>{t("portal.chrome.search")}</span>
              <kbd className="desk-sidebar-search-kbd">⌘K</kbd>
            </button>

            <button
              className="desk-icon-btn md:hidden"
              onClick={() => { setSearchOpen(true); setTimeout(() => searchRef.current?.focus(), 100); }}
              aria-label={t("supplier_portal.supplier_shell.text.search")}
            >
              <Icon name="search" />
            </button>

            <div className="desk-live-indicator hidden lg:inline-flex">
              <span className="desk-live-dot" />
              Live
            </div>

            <div className="relative" ref={notifRef}>
              <button
                className="desk-icon-btn"
                aria-label={t("portal.nav.notifications")}
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
                aria-label={t("supplier_portal.supplier_shell.text.profile_menu")}
              >
                <div className="desk-profile-avatar">SP</div>
                <div className="desk-profile-info hidden lg:flex">
                  <span className="desk-profile-name">{t("supplier_portal.admin.empathy.hierarchy.supplier.level")}</span>
                  <span className="desk-profile-role">
                    {hasSession ? "Administrator" : "Guest"}
                  </span>
                </div>
              </button>
              {profileOpen && (
                <div className="md-menu" style={{ right: 0, top: 48, minWidth: 220 }}>
                  <div className="px-4 py-3" style={{ borderBottom: '1px solid var(--desk-border)' }}>
                    <p style={{ font: 'var(--type-title)', color: 'var(--desk-text-primary)', margin: 0 }}>{t("supplier_portal.supplier_shell.text.supplier_admin")}</p>
                    <p style={{ font: 'var(--type-caption-sm)', color: 'var(--desk-text-tertiary)', margin: '4px 0 0' }}>{t("supplier_portal.supplier_shell.text.single_tenant_control_plane")}</p>
                  </div>
                  <Link href={"/profile" as any} className="md-menu-item" onClick={() => setProfileOpen(false)}>
                    <Icon name="supplier" />
                    <span>{t("portal.nav.profile")}</span>
                  </Link>
                  <div style={{ height: 1, background: 'var(--desk-border)', margin: '4px 12px' }} />
                  <button className="md-menu-item" style={{ color: 'var(--desk-danger)' }} onClick={() => { setProfileOpen(false); handleLogout(); }}>
                    <Icon name="logout" />
                    <span>{t("common.action.sign_out")}</span>
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
                    placeholder={t("supplier_portal.supplier_shell.text.search_pages")}
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
                        <span>{t(item.labelKey)}</span>
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
