"use client";

import { usePathname, useRouter } from "next/navigation";
import Link from "next/link";
import { useEffect, useState, useMemo, useCallback, memo } from "react";
import { motion, AnimatePresence, useReducedMotion } from "framer-motion";
import {
  PanelLeftClose,
  PanelLeft,
  Menu,
  Bell,
  Search,
  LayoutDashboard,
  ShoppingCart,
  PackageSearch,
  Activity,
  BarChart3,
  Settings,
  LogOut,
  Store,
  X,
  MapPin,
  Container,
} from "lucide-react";
import { getRetailerProfile } from "@/lib/retailer-profile";
import { useWebSocket } from "../lib/ws";
import { useRetailerNotifications } from "../lib/notifications";
import { clearStoredToken } from "../lib/bridge";
import { useTheme, type ThemeMode } from "./ThemeProvider";
import Icon from "./Icon";

/* ────────── Navigation Config ────────── */

type NavEntry = { href: string; icon: React.ElementType; label: string };
type NavSection = { label?: string; items: NavEntry[] };
type RetailerIdentity = { name: string; company: string; initials: string };

const NAV: NavSection[] = [
  {
    items: [
      { href: "/dashboard", icon: LayoutDashboard, label: "Dashboard" },
      { href: "/orders", icon: ShoppingCart, label: "Orders" },
      { href: "/tracking", icon: MapPin, label: "Tracking" },
      { href: "/dock", icon: Container, label: "Dock" },
      { href: "/catalog", icon: PackageSearch, label: "Catalog" },
      { href: "/procurement", icon: Activity, label: "Procurement" },
      { href: "/insights", icon: BarChart3, label: "Insights" },
    ],
  },
  {
    label: "System",
    items: [{ href: "/settings", icon: Settings, label: "Settings" }],
  },
];

const DEFAULT_IDENTITY: RetailerIdentity = {
  name: "Retailer",
  company: "Workspace",
  initials: "R",
};

function isActiveRoute(pathname: string, href: string): boolean {
  if (href === "/") return pathname === "/";
  return pathname === href || pathname.startsWith(href + "/");
}

/* ── Breadcrumb helper ── */
function buildBreadcrumbs(pathname: string): { label: string; href: string }[] {
  if (pathname === "/") return [{ label: "Hub", href: "/" }];
  const segs = pathname.split("/").filter(Boolean);
  const crumbs: { label: string; href: string }[] = [
    { label: "Home", href: "/" },
  ];
  let path = "";
  for (const seg of segs) {
    path += "/" + seg;
    const label = seg.charAt(0).toUpperCase() + seg.slice(1).replace(/-/g, " ");
    crumbs.push({ label, href: path });
  }
  return crumbs;
}

const ThemeToggle = memo(function ThemeToggle() {
  const { mode, cycle } = useTheme();
  const iconName: Record<ThemeMode, string> = {
    system: "autoMode",
    light: "lightMode",
    dark: "darkMode",
  };

  return (
    <button
      type="button"
      onClick={cycle}
      className="portal-btn portal-btn--ghost desk-icon-btn"
      title={`Theme: ${mode}`}
      aria-label={`Theme: ${mode}`}
    >
      <Icon name={iconName[mode]} size={18} />
    </button>
  );
});

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
        <div
          className={`flex items-center gap-3 transition-all duration-200 ${isRail ? "justify-center px-2 pt-4 pb-2" : "px-4 pt-4 pb-2"}`}
        >
          {isRail ? (
            <button
              onClick={onToggle}
              aria-label="Open sidebar"
              className="desk-icon-btn active-press"
            >
              <PanelLeft
                size={20}
                strokeWidth={1.5}
                style={{ color: "var(--desk-text-secondary)" }}
              />
            </button>
          ) : (
            <motion.div
              layout
              initial={{ opacity: 0, x: -10 }}
              animate={{ opacity: 1, x: 0 }}
              className="flex items-center gap-3 w-full"
            >
              <div
                className="desk-logo-mark"
                style={{
                  background: "var(--desk-accent)",
                  color: "white",
                  borderRadius: "var(--radius-sm)",
                }}
              >
                <Store size={18} />
              </div>
              <div className="min-w-0 flex-1">
                <p
                  className="desk-sidebar-section-label"
                  style={{
                    padding: 0,
                    margin: 0,
                    color: "var(--desk-text-tertiary)",
                    fontSize: "10px",
                    textTransform: "uppercase",
                    letterSpacing: "0.05em",
                  }}
                >
                  Retailer Workspace
                </p>
                <h1
                  style={{
                    font: "var(--type-title)",
                    color: "var(--desk-text-primary)",
                    margin: 0,
                    letterSpacing: "-0.02em",
                    fontWeight: 600,
                  }}
                >
                  V.O.I.D Hub
                </h1>
              </div>
              {!isMobile && (
                <button
                  onClick={onToggle}
                  className="desk-icon-btn active-press"
                  style={{ width: 28, height: 28 }}
                  aria-label="Collapse sidebar"
                >
                  <PanelLeftClose
                    size={16}
                    strokeWidth={1.5}
                    style={{ color: "var(--desk-text-tertiary)" }}
                  />
                </button>
              )}
            </motion.div>
          )}
        </div>

        {/* Divider */}
        <div
          style={{
            height: 1,
            background: "var(--desk-border)",
            margin: isRail ? "4px 8px" : "4px 16px",
          }}
        />

        {/* Navigation */}
        <nav
          className={`flex flex-col gap-0.5 mt-1 transition-all duration-200 ${isRail ? "px-1.5" : "px-2.5"}`}
        >
          {NAV.map((section, si) => (
            <div key={si}>
              {si > 0 && (
                <div
                  style={{
                    height: 1,
                    background: "var(--desk-border)",
                    margin: isRail ? "8px 4px" : "8px 12px",
                  }}
                />
              )}
              {section.label && !isRail && (
                <motion.div
                  layout
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  className="desk-sidebar-section-label"
                  style={{
                    color: "var(--desk-text-tertiary)",
                    paddingLeft: "12px",
                    marginBottom: "4px",
                  }}
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
                    data-active={active ? "true" : undefined}
                    className={`desk-sidebar-link desk-sidebar-item active-press ${active ? "desk-sidebar-link--active" : ""}`}
                    title={isRail ? item.label : undefined}
                    aria-label={item.label}
                    style={{
                      justifyContent: isRail ? "center" : "flex-start",
                      padding: isRail ? "0" : "0 12px",
                      height: 40,
                    }}
                  >
                    <ItemIcon
                      size={18}
                      className="desk-sidebar-item-icon"
                      style={{
                        color: active
                          ? "var(--desk-accent)"
                          : "var(--desk-text-tertiary)",
                      }}
                    />
                    {!isRail && (
                      <motion.span
                        layout
                        initial={{ opacity: 0, x: -5 }}
                        animate={{ opacity: 1, x: 0 }}
                        className="truncate ml-3"
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
      <div
        className={`py-3 transition-all duration-200 ${isRail ? "px-2" : "px-3"}`}
        style={{
          borderTop: "1px solid var(--desk-border)",
          background: "var(--desk-surface-subtle)",
        }}
      >
        <div
          className={`flex items-center ${isRail ? "justify-center" : "gap-2"}`}
        >
          <ThemeToggle />
          {!isRail && (
            <button
              onClick={onLogout}
              className="portal-btn portal-btn--ghost desk-sidebar-item active-press flex-1"
              title="Sign Out"
              aria-label="Sign Out"
              type="button"
              style={{
                height: 40,
                borderRadius: "var(--radius-md)",
                padding: "0 12px",
              }}
            >
              <LogOut
                size={18}
                className="desk-sidebar-item-icon"
                style={{ color: "var(--desk-text-tertiary)" }}
              />
              <span
                style={{
                  color: "var(--desk-text-secondary)",
                  marginLeft: "12px",
                }}
              >
                Sign Out
              </span>
            </button>
          )}
        </div>
        {!isRail && (
          <motion.div
            layout
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            className="mt-3 px-3"
          >
            <div className="flex items-center gap-2">
              <span
                className="desk-live-dot"
                style={{ background: "var(--desk-success)" }}
              />
              <span
                style={{
                  font: "var(--type-caption-sm)",
                  color: "var(--desk-text-tertiary)",
                  fontSize: "10px",
                }}
              >
                v2.0.0 · DESKTOP READY
              </span>
            </div>
          </motion.div>
        )}
      </div>
    </div>
  );
});

/* ── Shell ── */

export default function RetailerShell({
  children,
}: {
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const router = useRouter();
  const { isConnected } = useWebSocket();
  const { unreadCount } = useRetailerNotifications();
  const [collapsed, setCollapsed] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);
  const [identity, setIdentity] = useState<RetailerIdentity>(DEFAULT_IDENTITY);
  const reduceMotion = useReducedMotion();

  const handleLogout = useCallback(async () => {
    document.cookie = "pegasus_retailer_jwt=; Max-Age=0; path=/";
    await clearStoredToken();
    router.push("/");
  }, [router]);

  const breadcrumbs = useMemo(() => buildBreadcrumbs(pathname), [pathname]);

  /* Close mobile drawer on route change */
  useEffect(() => {
    setMobileOpen(false);
  }, [pathname]);
  useEffect(() => {
    const profile = getRetailerProfile();
    if (!profile) {
      setIdentity(DEFAULT_IDENTITY);
      return;
    }
    const source = (profile.name || profile.company || "Retailer").trim();
    const initials =
      source
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
  }, []);

  return (
    <div
      className="flex h-dvh overflow-hidden w-full"
      style={{
        background: "var(--desk-canvas)",
        color: "var(--desk-text-primary)",
      }}
    >
      {/* ── Desktop Sidebar ── */}
      <motion.div
        layout={!reduceMotion}
        initial={false}
        animate={{ width: collapsed ? 72 : 264 }}
        transition={reduceMotion ? { duration: 0 } : { type: "spring", stiffness: 400, damping: 40 }}
        className="hidden md:flex flex-col shrink-0 overflow-hidden z-10"
        style={{
          borderRight: "1px solid var(--desk-border)",
          background: "var(--desk-surface)",
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
              className="absolute inset-0 bg-[#0a0a0a]/40 backdrop-blur-sm"
              onClick={() => setMobileOpen(false)}
            />
            <motion.div
              layout
              initial={{ x: "-100%" }}
              animate={{ x: 0 }}
              exit={{ x: "-100%" }}
              transition={{ type: "spring", stiffness: 400, damping: 40 }}
              className="absolute inset-y-0 left-0 w-[264px] flex flex-col shadow-[var(--shadow-overlay)]"
              style={{
                background: "var(--desk-surface)",
                borderRight: "1px solid var(--desk-border)",
              }}
            >
              <div
                className="flex items-center justify-between px-4 h-14"
                style={{ borderBottom: "1px solid var(--desk-border)" }}
              >
                <span
                  style={{
                    font: "var(--type-title)",
                    color: "var(--desk-text-primary)",
                    fontWeight: 600,
                  }}
                >
                  Menu
                </span>
                <button
                  className="desk-icon-btn active-press"
                  onClick={() => setMobileOpen(false)}
                  aria-label="Close menu"
                >
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
        <header
          className="desk-topbar shrink-0"
          style={{
            background: "var(--desk-surface)",
            borderBottom: "1px solid var(--desk-border)",
            height: "var(--desk-topbar-height)",
          }}
        >
          <div className="desk-topbar-left px-6">
            <button
              className="desk-icon-btn md:hidden -ml-2 active-press"
              onClick={() => setMobileOpen(true)}
              aria-label="Open menu"
            >
              <Menu size={24} />
            </button>

            <nav
              className="desk-breadcrumb hidden md:flex"
              aria-label="Breadcrumb"
            >
              {breadcrumbs.map((crumb, i) => (
                <motion.span
                  layout
                  key={crumb.href}
                  className="flex items-center gap-2 min-w-0"
                >
                  {i > 0 && (
                    <span
                      className="desk-breadcrumb-sep"
                      style={{ color: "var(--desk-text-tertiary)" }}
                    >
                      /
                    </span>
                  )}
                  {i === breadcrumbs.length - 1 ? (
                    <span
                      className="desk-breadcrumb-current truncate"
                      style={{
                        color: "var(--desk-text-primary)",
                        fontWeight: 500,
                      }}
                    >
                      {crumb.label}
                    </span>
                  ) : (
                    <Link
                      href={crumb.href}
                      className="truncate"
                      style={{ color: "var(--desk-text-secondary)" }}
                    >
                      {crumb.label}
                    </Link>
                  )}
                </motion.span>
              ))}
            </nav>

            <span
              className="md:hidden truncate"
              style={{
                font: "var(--type-title)",
                color: "var(--desk-text-primary)",
                fontWeight: 600,
              }}
            >
              {breadcrumbs[breadcrumbs.length - 1]?.label ?? "Hub"}
            </span>
          </div>

          <div className="desk-topbar-right px-6 gap-4">
            {/* Live indicator */}
            <div
              className={`desk-live-indicator hidden md:inline-flex ${!isConnected ? "opacity-50" : ""}`}
              style={{
                borderColor: "var(--desk-border)",
                background: "var(--desk-surface-subtle)",
                color: "var(--desk-text-secondary)",
                fontSize: "12px",
                fontWeight: 500,
              }}
            >
              <span
                className={isConnected ? "desk-live-dot" : ""}
                style={
                  !isConnected
                    ? {
                        width: 6,
                        height: 6,
                        borderRadius: "50%",
                        background: "var(--desk-danger)",
                      }
                    : undefined
                }
              />
              {isConnected ? "LIVE FEED" : "OFFLINE"}
            </div>

            {/* Search */}
            <button
              className="desk-topbar-search hidden md:flex"
              style={{
                background: "var(--desk-surface-subtle)",
                border: "1px solid var(--desk-border)",
                borderRadius: "var(--radius-md)",
              }}
              onClick={() => router.push("/catalog")}
              aria-label="Open catalog search"
            >
              <Search
                size={16}
                style={{ color: "var(--desk-text-tertiary)" }}
              />
              <span
                style={{
                  flex: 1,
                  textAlign: "left",
                  color: "var(--desk-text-secondary)",
                }}
              >
                Search catalog...
              </span>
              <kbd
                className="desk-sidebar-search-kbd"
                style={{
                  background: "var(--desk-surface)",
                  border: "1px solid var(--desk-border)",
                  color: "var(--desk-text-tertiary)",
                }}
              >
                ⌘K
              </kbd>
            </button>

            <button
              className="desk-icon-btn md:hidden active-press"
              onClick={() => router.push("/catalog")}
              aria-label="Search catalog"
            >
              <Search size={20} />
            </button>

            {/* Notifications */}
            <button
              className="desk-icon-btn relative active-press"
              style={{ color: "var(--desk-text-secondary)" }}
              aria-label={`Notifications${unreadCount > 0 ? `, ${unreadCount} unread` : ""}`}
              onClick={() => router.push("/notifications")}
            >
              <Bell size={20} />
              <AnimatePresence>
                {unreadCount > 0 && (
                  <motion.span
                    initial={{ scale: 0 }}
                    animate={{ scale: 1 }}
                    exit={{ scale: 0 }}
                    className="desk-notif-badge"
                    style={{ background: "var(--desk-accent)", color: "white" }}
                  >
                    {unreadCount > 9 ? "9+" : unreadCount}
                  </motion.span>
                )}
              </AnimatePresence>
            </button>

            {/* Profile pill */}
            <div
              className="desk-profile-pill"
              style={{
                background: "var(--desk-surface-subtle)",
                border: "1px solid var(--desk-border)",
                borderRadius: "var(--radius-pill)",
              }}
            >
              <div
                className="desk-profile-avatar"
                style={{ background: "var(--desk-accent)", color: "white" }}
              >
                {identity.initials}
              </div>
              <div className="desk-profile-info hidden lg:flex">
                <span
                  className="desk-profile-name"
                  style={{ color: "var(--desk-text-primary)", fontWeight: 600 }}
                >
                  {identity.name}
                </span>
                <span
                  className="desk-profile-role"
                  style={{
                    color: "var(--desk-text-tertiary)",
                    fontSize: "11px",
                    textTransform: "uppercase",
                    letterSpacing: "0.05em",
                  }}
                >
                  {identity.company}
                </span>
              </div>
            </div>
          </div>
        </header>

        {/* Main Content scroll area */}
        <main
          className="flex-1 overflow-y-auto w-full relative"
          style={{ background: "var(--desk-canvas)" }}
        >
          <div className="absolute inset-0">{children}</div>
        </main>
      </div>
    </div>
  );
}
