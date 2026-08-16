'use client';

import { usePathname } from 'next/navigation';
import Link from 'next/link';
import { memo, useEffect, useMemo, useState, useCallback, useRef } from 'react';
import { PanelLeft, PanelLeftClose } from 'lucide-react';
import Icon from './Icon';
import { useTheme, type ThemeMode } from './ThemeProvider';
import { apiFetch, decodeJwtPayload, readTokenFromCookie } from '@/lib/auth';
import { SessionPackChip } from './SessionPackChip';
import ClientPolicyBanner from './ClientPolicyBanner';
import NotificationPanel from './NotificationPanel';
import LanguageSwitcher from './LanguageSwitcher';
import { useNotifications } from '@/lib/useNotifications';
import { usePortalT } from '@/lib/i18n';
import { motion, useReducedMotion } from 'framer-motion';

type NavEntry = { href: string; icon: string; labelKey: string };
type NavSection = { labelKey?: string; items: NavEntry[] };

const NAV: NavSection[] = [
  {
    items: [
      { href: '/', icon: 'dashboard', labelKey: 'portal.nav.dashboard' },
      { href: '/loading-bay', icon: 'loadingBay', labelKey: 'portal.nav.loading_bay' },
      { href: '/transfers', icon: 'transfers', labelKey: 'portal.nav.transfers' },
    ],
  },
  {
    labelKey: 'portal.nav.section.operations',
    items: [
      { href: '/fleet', icon: 'fleet', labelKey: 'portal.nav.fleet' },
      { href: '/staff', icon: 'staff', labelKey: 'portal.nav.staff' },
      { href: '/settings/location', icon: 'loadingBay', labelKey: 'portal.nav.location' },
      { href: '/insights', icon: 'insights', labelKey: 'portal.nav.insights' },
      { href: '/analytics', icon: 'analytics', labelKey: 'portal.nav.analytics' },
    ],
  },
  {
    labelKey: 'portal.nav.section.supply_chain',
    items: [
      { href: '/supply-requests', icon: 'transfers', labelKey: 'portal.nav.supply_requests' },
      { href: '/payload', icon: 'loadingBay', labelKey: 'portal.nav.payload' },
      { href: '/payload-override', icon: 'loadingBay', labelKey: 'portal.nav.payload_override' },
      { href: '/manifests', icon: 'manifests', labelKey: 'portal.nav.manifests' },
      { href: '/manifest-exceptions', icon: 'insights', labelKey: 'portal.nav.gate_exceptions' },
    ],
  },
];

const ALL_NAV_ITEMS = NAV.flatMap((section) => section.items);
const BARE_ROUTES = ['/auth/', '/setup/'];

function isActiveRoute(pathname: string, href: string): boolean {
  if (href === '/') return pathname === '/';
  return pathname === href || pathname.startsWith(`${href}/`);
}

/* ── Theme Toggle ── */
const ThemeToggle = memo(function ThemeToggle() {
  const { mode, cycle } = useTheme();
  const t = usePortalT();
  const iconName: Record<ThemeMode, string> = {
    system: 'autoMode',
    light: 'lightMode',
    dark: 'darkMode',
  };
  const labelKey: Record<ThemeMode, string> = {
    system: 'portal.chrome.theme_system',
    light: 'portal.chrome.theme_light',
    dark: 'portal.chrome.theme_dark',
  };

  return (
    <button
      type="button"
      onClick={cycle}
      className="portal-btn portal-btn--ghost desk-icon-btn"
      title={t(labelKey[mode])}
      aria-label={t(labelKey[mode])}
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
  const t = usePortalT();

  return (
    <div className="flex flex-col h-full">
      <div className="flex-1 overflow-y-auto overflow-x-hidden">
        {/* Header */}
        <div className={`flex items-center gap-3 transition-all duration-200 ${isRail ? 'justify-center px-2 pt-4 pb-2' : 'px-4 pt-4 pb-2'}`}>
          {isRail ? (
            <button onClick={onToggle} aria-label={t("portal.chrome.open_sidebar")} className="desk-icon-btn">
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
                <p className="desk-sidebar-section-label" style={{ padding: 0, margin: 0 }}>{t("portal.chrome.factory_hub")}</p>
                <h1 style={{ font: 'var(--type-title)', color: 'var(--desk-text-primary)', margin: 0, letterSpacing: '-0.02em' }}>
                  {factoryName}
                </h1>
              </div>
              {!isMobile && (
                <button onClick={onToggle} className="desk-icon-btn" style={{ width: 28, height: 28 }} aria-label={t("portal.chrome.collapse_sidebar")}>
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
                    transition={{ delay: (si * 0.08) + (ii * 0.02), type: 'spring', stiffness: 320, damping: 28 }}
                  >
                    <Link
                      href={item.href}
                      prefetch={false}
                      className={`desk-sidebar-item desk-sidebar-link${active ? ' desk-sidebar-link--active' : ''}`}
                      data-active={active ? 'true' : undefined}
                      aria-current={active ? 'page' : undefined}
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

      {/* Footer */}
      <div className={`py-3 transition-all duration-200 ${isRail ? 'px-2' : 'px-3'}`} style={{ borderTop: '1px solid var(--desk-border)' }}>
        {!isRail && (
          <div className="desk-card-padded desk-card mb-3" style={{ borderRadius: 'var(--radius-md)' }}>
            <div className="flex items-center gap-2">
              <span className="desk-live-dot" />
              <span style={{ font: 'var(--type-caption-sm)', color: 'var(--desk-text-primary)', fontWeight: 600 }}>{t("factory_portal.factory_shell.text.desktop_command_ready")}</span>
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
              title={t("common.action.sign_out")}
              aria-label={t("common.action.sign_out")}
            >
              <Icon name="logout" size={18} />
              <span>{t("common.action.sign_out")}</span>
            </Link>
          )}
        </div>
        {!isRail && <div className="mt-2"><LanguageSwitcher /></div>}
      </div>
    </div>
  );
});

/* ── Shell ── */
export default function FactoryShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const reducedMotion = useReducedMotion();
  const isBare = BARE_ROUTES.some((route) => pathname.startsWith(route));
  const isConfigured = useMemo(() => {
    const token = readTokenFromCookie();
    const claims = token ? decodeJwtPayload(token) : null;
    return claims?.is_configured === true;
  }, [pathname]);
  const notificationsEnabled = !isBare && isConfigured;
  const [collapsed, setCollapsed] = useState(false);
  const [factoryName, setFactoryName] = useState('Factory Portal');
  const [refreshEpoch, setRefreshEpoch] = useState(0);
  const [notifOpen, setNotifOpen] = useState(false);
  const notifRef = useRef<HTMLDivElement>(null);
  const { items: notifItems, unreadCount, markRead, markAllRead } = useNotifications({
    enabled: notificationsEnabled,
  });

  const currentEntry = useMemo(
    () => ALL_NAV_ITEMS.find((item) => isActiveRoute(pathname, item.href)) ?? ALL_NAV_ITEMS[0],
    [pathname],
  );
  const t = usePortalT();
  const currentSection = useMemo(
    () => {
      const section = NAV.find((s) => s.items.some((item) => item.href === currentEntry.href));
      return section?.labelKey ? t(section.labelKey) : t('portal.chrome.factory_hub');
    },
    [currentEntry.href, t],
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
    if (!notificationsEnabled) return;
    loadFactoryProfile().catch((error) => {
      console.error('[FactoryShell] profile load failed', error);
    });
  }, [loadFactoryProfile, notificationsEnabled]);

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

  const isBareRoute = BARE_ROUTES.some((route) => pathname.startsWith(route));
  if (isBareRoute) return <>{children}</>;

  return (
    <div className="flex h-dvh overflow-hidden" style={{ background: 'var(--desk-canvas)' }}>
      {/* Desktop Sidebar */}
      <motion.aside
        initial={false}
        animate={{ width: collapsed ? 72 : 264 }}
        transition={reducedMotion ? { duration: 0 } : { type: 'spring', stiffness: 300, damping: 30 }}
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
            <nav className="desk-breadcrumb" aria-label={t("factory_portal.factory_shell.text.breadcrumb")}>
              <span className="desk-breadcrumb-sep">{currentSection}</span>
              <span className="desk-breadcrumb-sep">/</span>
              <span className="desk-breadcrumb-current">{t(currentEntry.labelKey)}</span>
            </nav>
          </div>

          <div className="desk-topbar-right">
            <SessionPackChip />
            <div className="desk-live-indicator hidden lg:inline-flex">
              <span className="desk-live-dot" />
              Factory network live
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
          </div>
        </header>

        <main key={refreshEpoch} className="min-w-0 flex-1 overflow-y-auto" style={{ background: 'var(--desk-canvas)' }}>
          <ClientPolicyBanner />
          <div className="min-h-full">{children}</div>
        </main>
      </div>
    </div>
  );
}
