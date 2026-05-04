"use client";

import { useState, useCallback, useEffect } from "react";
import {
  User, Mail, MapPin, Bell, CreditCard, Settings, Brain,
  AlertTriangle, Loader2, Building2, Layers, Package, Boxes,
  ChevronDown, ChevronRight, Info, CheckCircle2,
} from "lucide-react";
import { Button, Chip, Skeleton } from "@heroui/react";
import { useLiveData } from "../../../lib/hooks";
import { apiFetch } from "../../../lib/auth";
import type {
  AutoOrderSettings, SupplierOverride, CategoryOverride,
  ProductOverride, VariantOverride, RetailerProfile,
} from "../../../lib/types";

function getBrowserStorage(): Storage | null {
  if (typeof window === "undefined" || typeof window.localStorage?.getItem !== "function") {
    return null;
  }
  return window.localStorage;
}

/* ── Toggle Switch ── */
function Toggle({
  on,
  onToggle,
  disabled = false,
}: {
  on: boolean;
  onToggle: () => void;
  disabled?: boolean;
}) {
  return (
    <button
      onClick={onToggle}
      disabled={disabled}
      className="w-12 h-7 rounded-full flex items-center p-1 cursor-pointer transition-colors duration-200 disabled:opacity-50 shrink-0"
      style={{ background: on ? "var(--accent)" : "var(--surface)" }}
    >
      {disabled ? (
        <Loader2 size={16} className="animate-spin mx-auto" style={{ color: "var(--muted)" }} />
      ) : (
        <div
          className="w-5 h-5 rounded-full transition-transform duration-200"
          style={{
            background: on ? "var(--accent-foreground)" : "var(--muted)",
            transform: on ? "translateX(20px)" : "translateX(0)",
          }}
        />
      )}
    </button>
  );
}

/* ── Override Row ── */
function OverrideRow({
  id,
  label,
  enabled,
  hasHistory,
  analyticsDate,
  icon: Icon,
  onToggle,
  saving,
}: {
  id: string;
  label: string;
  enabled: boolean;
  hasHistory?: boolean;
  analyticsDate?: string;
  icon: React.ElementType;
  onToggle: () => void;
  saving: boolean;
}) {
  return (
    <div className="flex items-center justify-between py-3 border-b border-[var(--border)] last:border-0 gap-3">
      <div className="flex items-center gap-3 min-w-0">
        <div className="w-8 h-8 rounded-lg flex items-center justify-center shrink-0" style={{ background: "var(--surface)" }}>
          <Icon size={16} style={{ color: "var(--muted)" }} />
        </div>
        <div className="min-w-0">
          <span className="md-typescale-body-medium font-medium text-foreground block truncate">{label}</span>
          <div className="flex items-center gap-2">
            {hasHistory && (
              <span className="md-typescale-label-small text-muted flex items-center gap-1">
                <CheckCircle2 size={10} style={{ color: "var(--success)" }} /> Has history
              </span>
            )}
            {analyticsDate && (
              <span className="md-typescale-label-small text-muted">
                Since {new Date(analyticsDate).toLocaleDateString()}
              </span>
            )}
          </div>
        </div>
      </div>
      <Toggle on={enabled} onToggle={onToggle} disabled={saving} />
    </div>
  );
}

/* ── Collapsible Section ── */
function OverrideSection<T extends { enabled: boolean }>({
  title,
  icon: Icon,
  items,
  getId,
  getLabel,
  getHasHistory,
  getAnalyticsDate,
  rowIcon,
  onToggle,
  savingId,
}: {
  title: string;
  icon: React.ElementType;
  items: T[];
  getId: (item: T) => string;
  getLabel: (item: T) => string;
  getHasHistory?: (item: T) => boolean;
  getAnalyticsDate?: (item: T) => string | undefined;
  rowIcon: React.ElementType;
  onToggle: (id: string, enabled: boolean) => void;
  savingId: string | null;
}) {
  const [open, setOpen] = useState(items.length <= 6);

  if (items.length === 0) return null;

  const enabledCount = items.filter((i) => i.enabled).length;

  return (
    <div className="bento-card">
      <button
        onClick={() => setOpen((p) => !p)}
        className="flex items-center justify-between w-full cursor-pointer"
      >
        <div className="flex items-center gap-2">
          <Icon size={18} style={{ color: "var(--accent)" }} />
          <span className="md-typescale-title-small font-semibold text-foreground">{title}</span>
          <Chip size="sm" color="default" variant="soft" className="ml-1">
            {enabledCount}/{items.length}
          </Chip>
        </div>
        {open ? <ChevronDown size={18} style={{ color: "var(--muted)" }} /> : <ChevronRight size={18} style={{ color: "var(--muted)" }} />}
      </button>

      {open && (
        <div className="mt-3">
          {items.map((item) => {
            const id = getId(item);
            return (
              <OverrideRow
                key={id}
                id={id}
                label={getLabel(item)}
                enabled={item.enabled}
                hasHistory={getHasHistory?.(item)}
                analyticsDate={getAnalyticsDate?.(item)}
                icon={rowIcon}
                onToggle={() => onToggle(id, !item.enabled)}
                saving={savingId === id}
              />
            );
          })}
        </div>
      )}
    </div>
  );
}

/* ── Main Page ── */

export default function SettingsPage() {
  const { data: autoOrder, loading, error, mutate } = useLiveData<AutoOrderSettings>("/v1/retailer/settings/auto-order");
  const [savingGlobal, setSavingGlobal] = useState(false);
  const [savingId, setSavingId] = useState<string | null>(null);
  const [notifOn, setNotifOn] = useState(() => {
    return getBrowserStorage()?.getItem("retailer_notif") !== "false";
  });

  /* ── Profile Editing ── */
  const [profileEditing, setProfileEditing] = useState(false);
  const [profileName, setProfileName] = useState("");
  const [profileEmail, setProfileEmail] = useState("");
  const [profileLocation, setProfileLocation] = useState("");
  const [profileCompany, setProfileCompany] = useState("");
  const [savingProfile, setSavingProfile] = useState(false);

  // Load profile from localStorage (set at login) + attempt API fetch
  useEffect(() => {
    // Seed from localStorage first for instant display
    const storage = getBrowserStorage();
    if (storage) {
      try {
        const p: RetailerProfile = JSON.parse(storage.getItem('retailer_profile') || '{}');
        if (p.name) setProfileName(p.name);
        if (p.company) setProfileCompany(p.company);
        if (p.email) setProfileEmail(p.email);
      } catch { /* ignore */ }
    }
    // Then try to fetch fresh from API
    apiFetch('/v1/retailer/profile').then(async (res) => {
      if (!res.ok) return;
      const data = await res.json();
      if (data.name) setProfileName(data.name);
      if (data.company) setProfileCompany(data.company);
      if (data.phone) setProfileEmail(data.phone); // phone as contact
      if (data.location) setProfileLocation(data.location);
    }).catch(() => { /* endpoint may not exist yet */ });
  }, []);

  const saveProfile = useCallback(async () => {
    setSavingProfile(true);
    try {
      const res = await apiFetch("/v1/retailer/profile", {
        method: "PUT",
        body: JSON.stringify({ name: profileName, company: profileCompany, location: profileLocation }),
      });
      if (res.ok) {
        setProfileEditing(false);
        // Update localStorage cache so avatar/shell pick up changes
        const storage = getBrowserStorage();
        if (storage) {
          try {
            const existing = JSON.parse(storage.getItem('retailer_profile') || '{}');
            storage.setItem('retailer_profile', JSON.stringify({
              ...existing,
              name: profileName,
              company: profileCompany,
            }));
          } catch { /* ignore */ }
        }
      }
    } catch { /* swallow */ }
    finally { setSavingProfile(false); }
  }, [profileName, profileCompany, profileLocation]);

  const toggleGlobal = useCallback(async () => {
    if (!autoOrder) return;
    setSavingGlobal(true);
    try {
      await apiFetch("/v1/retailer/settings/auto-order/global", {
        method: "PATCH",
        body: JSON.stringify({ enabled: !autoOrder.global_enabled }),
      });
      mutate();
    } catch { /* handled by next poll */ }
    finally { setSavingGlobal(false); }
  }, [autoOrder, mutate]);

  const toggleOverride = useCallback(async (level: string, id: string, enabled: boolean) => {
    setSavingId(id);
    try {
      await apiFetch(`/v1/retailer/settings/auto-order/${level}/${id}`, {
        method: "PATCH",
        body: JSON.stringify({ enabled }),
      });
      mutate();
    } catch { /* handled by next poll */ }
    finally { setSavingId(null); }
  }, [mutate]);

  if (loading) {
    return (
      <div className="min-h-full p-6 md:p-8 max-w-5xl mx-auto">
        <Skeleton className="h-8 w-48 rounded-lg mb-2" />
        <Skeleton className="h-4 w-80 rounded-lg mb-8" />
        <div className="grid grid-cols-2 gap-8">
          <Skeleton className="h-64 rounded-2xl" />
          <Skeleton className="h-64 rounded-2xl" />
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-full p-6 md:p-8 max-w-5xl mx-auto">
      {/* Header */}
      <header className="mb-8">
        <h1 className="md-typescale-headline-large">Profile & Settings</h1>
        <p className="md-typescale-body-medium mt-1" style={{ color: "var(--muted)" }}>
          Manage your retailer profile, notification preferences, and AI auto-ordering.
        </p>
      </header>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
        {/* Profile Card */}
        <div className="flex flex-col gap-4">
          <div className="flex items-center justify-between">
            <h2 className="md-typescale-title-large font-semibold text-foreground flex items-center gap-2">
              <User size={20} style={{ color: "var(--muted)" }} /> My Profile
            </h2>
            {!profileEditing ? (
              <button onClick={() => setProfileEditing(true)} className="md-typescale-label-small font-bold text-accent cursor-pointer hover:underline">
                Edit
              </button>
            ) : (
              <div className="flex items-center gap-2">
                <button onClick={() => setProfileEditing(false)} className="md-typescale-label-small font-bold text-muted cursor-pointer hover:underline">
                  Cancel
                </button>
                <Button variant="primary" size="sm" onPress={saveProfile} isDisabled={savingProfile} className="md-btn md-btn-filled px-4 h-8 flex items-center gap-1.5">
                  {savingProfile && <Loader2 size={14} className="animate-spin" />} Save
                </Button>
              </div>
            )}
          </div>
          <div className="bento-card flex flex-col gap-5">
            <div>
              <label className="md-typescale-label-small uppercase tracking-widest font-semibold flex items-center gap-1.5" style={{ color: "var(--muted)" }}>
                <User size={14} /> Retailer Name
              </label>
              {profileEditing ? (
                <input
                  value={profileName}
                  onChange={(e) => setProfileName(e.target.value)}
                  className="w-full mt-1.5 p-2.5 rounded-lg border border-[var(--border)] bg-transparent text-foreground md-typescale-body-large font-medium focus:outline-none focus:border-[var(--accent)]"
                />
              ) : (
                <p className="md-typescale-body-large font-medium text-foreground pt-1">{profileName}</p>
              )}
            </div>
            <div>
              <label className="md-typescale-label-small uppercase tracking-widest font-semibold flex items-center gap-1.5" style={{ color: "var(--muted)" }}>
                <Mail size={14} /> Contact Email
              </label>
              {profileEditing ? (
                <input
                  type="email"
                  value={profileEmail}
                  onChange={(e) => setProfileEmail(e.target.value)}
                  className="w-full mt-1.5 p-2.5 rounded-lg border border-[var(--border)] bg-transparent text-foreground md-typescale-body-large font-medium focus:outline-none focus:border-[var(--accent)]"
                />
              ) : (
                <p className="md-typescale-body-large font-medium text-foreground pt-1">{profileEmail}</p>
              )}
            </div>
            <div>
              <label className="md-typescale-label-small uppercase tracking-widest font-semibold flex items-center gap-1.5" style={{ color: "var(--muted)" }}>
                <MapPin size={14} /> Location
              </label>
              {profileEditing ? (
                <input
                  value={profileLocation}
                  onChange={(e) => setProfileLocation(e.target.value)}
                  className="w-full mt-1.5 p-2.5 rounded-lg border border-[var(--border)] bg-transparent text-foreground md-typescale-body-large font-medium focus:outline-none focus:border-[var(--accent)]"
                />
              ) : (
                <p className="md-typescale-body-large font-medium text-foreground pt-1">{profileLocation}</p>
              )}
            </div>
          </div>
        </div>

        {/* Preferences Card */}
        <div className="flex flex-col gap-4">
          <h2 className="md-typescale-title-large font-semibold text-foreground flex items-center gap-2">
            <Settings size={20} style={{ color: "var(--muted)" }} /> Preferences
          </h2>
          <div className="bento-card flex flex-col gap-1">
            <div className="flex items-center justify-between py-3 border-b border-[var(--border)]">
              <div className="flex items-center gap-3">
                <Bell size={18} style={{ color: "var(--muted)" }} />
                <span className="md-typescale-body-medium font-semibold text-foreground">Order Push Notifications</span>
              </div>
              <Toggle on={notifOn} onToggle={() => {
                const next = !notifOn;
                setNotifOn(next);
                getBrowserStorage()?.setItem("retailer_notif", String(next));
              }} />
            </div>
            <div className="flex items-center justify-between py-3">
              <div className="flex items-center gap-3">
                <CreditCard size={18} style={{ color: "var(--muted)" }} />
                <span className="md-typescale-body-medium font-semibold text-foreground">Delivery Payment Override</span>
              </div>
              <Toggle on={true} onToggle={() => {}} />
            </div>
          </div>
        </div>
      </div>

      {/* AI Auto-Order Settings */}
      <div className="mt-10">
        <h2 className="md-typescale-title-large font-semibold text-foreground flex items-center gap-2 mb-4">
          <Brain size={20} style={{ color: "var(--accent)" }} /> AI Auto-Ordering
        </h2>

        {error ? (
          <div className="bento-card flex items-center gap-3">
            <AlertTriangle size={18} style={{ color: "var(--warning)" }} />
            <span className="md-typescale-body-medium text-muted">Could not load auto-order settings</span>
            <Button size="sm" variant="secondary" onPress={() => mutate()} className="ml-auto">Retry</Button>
          </div>
        ) : autoOrder ? (
          <div className="flex flex-col gap-4">
            {/* Global Toggle */}
            <div className="bento-card" style={{ borderLeft: `3px solid ${autoOrder.global_enabled ? "var(--accent)" : "var(--border)"}` }}>
              <div className="flex items-center justify-between gap-4">
                <div>
                  <span className="md-typescale-title-small font-semibold text-foreground block">Global Auto-Order</span>
                  <span className="md-typescale-body-small text-muted">
                    {autoOrder.global_enabled
                      ? "AI will automatically reorder all products based on predictions. Individual overrides below fine-tune behavior."
                      : "Auto-ordering is disabled globally. Enable to let AI manage replenishment."}
                  </span>
                </div>
                <Toggle on={autoOrder.global_enabled} onToggle={toggleGlobal} disabled={savingGlobal} />
              </div>
              {autoOrder.analytics_start_date && (
                <p className="md-typescale-label-small text-muted mt-3 flex items-center gap-1">
                  <Info size={12} /> Analytics since {new Date(autoOrder.analytics_start_date).toLocaleDateString()}
                </p>
              )}
            </div>

            {/* How It Works */}
            <div className="bento-card" style={{ background: "var(--surface)" }}>
              <div className="flex items-center gap-2 mb-2">
                <Info size={14} style={{ color: "var(--accent)" }} />
                <span className="md-typescale-label-large font-semibold text-foreground">How Override Levels Work</span>
              </div>
              <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 mt-3">
                {[
                  { label: "Supplier", desc: "All products from this supplier", icon: Building2 },
                  { label: "Category", desc: "All products in this category", icon: Layers },
                  { label: "Product", desc: "This specific product", icon: Package },
                  { label: "Variant", desc: "A specific size/pack variant", icon: Boxes },
                ].map(({ label, desc, icon: I }) => (
                  <div key={label} className="flex items-start gap-2">
                    <I size={14} className="mt-0.5 shrink-0" style={{ color: "var(--muted)" }} />
                    <div>
                      <span className="md-typescale-label-small font-semibold text-foreground block">{label}</span>
                      <span className="md-typescale-label-small text-muted">{desc}</span>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* Supplier Overrides */}
            <OverrideSection
              title="Supplier Overrides"
              icon={Building2}
              items={autoOrder.supplier_overrides}
              getId={(s) => s.supplier_id}
              getLabel={(s) => s.supplier_id}
              getHasHistory={(s) => s.has_history}
              getAnalyticsDate={(s) => s.analytics_start_date}
              rowIcon={Building2}
              onToggle={(id, enabled) => toggleOverride("supplier", id, enabled)}
              savingId={savingId}
            />

            {/* Category Overrides */}
            <OverrideSection
              title="Category Overrides"
              icon={Layers}
              items={autoOrder.category_overrides}
              getId={(c) => c.category_id}
              getLabel={(c) => c.category_id}
              getHasHistory={(c) => c.has_history}
              getAnalyticsDate={(c) => c.analytics_start_date}
              rowIcon={Layers}
              onToggle={(id, enabled) => toggleOverride("category", id, enabled)}
              savingId={savingId}
            />

            {/* Product Overrides */}
            <OverrideSection
              title="Product Overrides"
              icon={Package}
              items={autoOrder.product_overrides}
              getId={(p) => p.product_id}
              getLabel={(p) => p.product_id}
              rowIcon={Package}
              onToggle={(id, enabled) => toggleOverride("product", id, enabled)}
              savingId={savingId}
            />

            {/* Variant Overrides */}
            <OverrideSection
              title="Variant Overrides"
              icon={Boxes}
              items={autoOrder.variant_overrides}
              getId={(v) => v.variant_id}
              getLabel={(v) => v.variant_id}
              rowIcon={Boxes}
              onToggle={(id, enabled) => toggleOverride("variant", id, enabled)}
              savingId={savingId}
            />
          </div>
        ) : null}
      </div>
    </div>
  );
}
