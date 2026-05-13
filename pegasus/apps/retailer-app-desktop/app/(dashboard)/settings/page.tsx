"use client";

import { useState, useCallback, useEffect } from "react";
import {
  User,
  Mail,
  MapPin,
  Bell,
  CreditCard,
  Settings,
  Brain,
  AlertTriangle,
  Loader2,
  Building2,
  Layers,
  Package,
  Boxes,
  ChevronDown,
  ChevronRight,
  Info,
  CheckCircle2,
} from "lucide-react";
import { Button, Chip, Skeleton } from "@heroui/react";
import { motion, AnimatePresence } from "framer-motion";
import { useLiveData } from "../../../lib/hooks";
import { apiFetch } from "../../../lib/auth";
import { BentoGrid, BentoCard } from "../../../components/BentoGrid";
import type { AutoOrderSettings, RetailerProfile } from "../../../lib/types";

function getBrowserStorage(): Storage | null {
  if (
    typeof window === "undefined" ||
    typeof window.localStorage?.getItem !== "function"
  ) {
    return null;
  }
  return window.localStorage;
}

type ProfileFieldErrors = {
  name?: string;
  company?: string;
};

type SaveBanner = {
  kind: "success" | "error";
  message: string;
};

function normalizeErrorMessage(error: unknown, fallback: string): string {
  if (typeof error === "string" && error.trim().length > 0) {
    return error;
  }
  if (error instanceof Error && error.message.trim().length > 0) {
    return error.message;
  }
  return fallback;
}

function validateProfileFields(
  name: string,
  company: string,
): ProfileFieldErrors {
  const errors: ProfileFieldErrors = {};
  const trimmedName = name.trim();
  const trimmedCompany = company.trim();

  if (trimmedName.length < 2) {
    errors.name = "Entity name must be at least 2 characters.";
  }

  if (trimmedCompany.length < 2) {
    errors.company = "Company name must be at least 2 characters.";
  }

  return errors;
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
      className="w-11 h-6 rounded-full flex items-center p-1 cursor-pointer transition-all duration-200 disabled:opacity-50 shrink-0"
      style={{ background: on ? "var(--desk-accent)" : "var(--desk-border)" }}
    >
      {disabled ? (
        <Loader2 size={14} className="animate-spin mx-auto text-white" />
      ) : (
        <motion.div
          layout
          className="w-4 h-4 rounded-full bg-white shadow-sm"
          initial={false}
          animate={{ x: on ? 20 : 0 }}
          transition={{ type: "spring", stiffness: 500, damping: 30 }}
        />
      )}
    </button>
  );
}

/* ── Override Row ── */
function OverrideRow({
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
    <div className="flex items-center justify-between py-3 border-b border-[var(--desk-border)] last:border-0 gap-4">
      <div className="flex items-center gap-3 min-w-0">
        <div className="w-10 h-10 rounded-xl flex items-center justify-center shrink-0 bg-[var(--desk-surface-subtle)] border border-[var(--desk-border)]">
          <Icon size={16} className="text-[var(--desk-text-tertiary)]" />
        </div>
        <div className="min-w-0">
          <span className="md-typescale-body-medium font-bold text-[var(--desk-text-primary)] block truncate">
            {label}
          </span>
          <div className="flex items-center gap-2 mt-0.5">
            {hasHistory && (
              <span className="md-typescale-label-small text-[var(--desk-success)] flex items-center gap-1 font-bold uppercase tracking-tighter">
                <CheckCircle2 size={10} /> Active History
              </span>
            )}
            {analyticsDate && (
              <span className="md-typescale-label-small text-[var(--desk-text-tertiary)] uppercase font-bold tracking-tighter">
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
  storageKey,
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
  storageKey?: string;
}) {
  const [open, setOpen] = useState(() => {
    if (!storageKey) return items.length <= 6;
    const saved = getBrowserStorage()?.getItem(storageKey);
    if (saved === "1") return true;
    if (saved === "0") return false;
    return items.length <= 6;
  });

  useEffect(() => {
    if (!storageKey) return;
    getBrowserStorage()?.setItem(storageKey, open ? "1" : "0");
  }, [open, storageKey]);

  if (items.length === 0) return null;

  const enabledCount = items.filter((i) => i.enabled).length;

  return (
    <div className="bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl p-6 shadow-[var(--shadow-sm)]">
      <button
        onClick={() => setOpen((p) => !p)}
        className="flex items-center justify-between w-full cursor-pointer group"
      >
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-[var(--desk-accent-soft)] text-[var(--desk-accent)] flex items-center justify-center">
            <Icon size={20} />
          </div>
          <span className="md-typescale-title-small font-bold text-[var(--desk-text-primary)]">
            {title}
          </span>
          <Chip
            size="sm"
            variant="secondary"
            className="font-bold text-[10px] ml-1"
          >
            {enabledCount}/{items.length} ACTIVE
          </Chip>
        </div>
        <div className="text-[var(--desk-text-tertiary)] group-hover:text-[var(--desk-text-primary)] transition-colors">
          {open ? <ChevronDown size={20} /> : <ChevronRight size={20} />}
        </div>
      </button>

      <AnimatePresence>
        {open && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            className="mt-6 overflow-hidden"
          >
            <div className="space-y-1">
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
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}

/* ── Main Page ── */

export default function SettingsPage() {
  const {
    data: autoOrder,
    loading,
    error,
    mutate,
  } = useLiveData<AutoOrderSettings>("/v1/retailer/settings/auto-order");
  const [savingGlobal, setSavingGlobal] = useState(false);
  const [savingId, setSavingId] = useState<string | null>(null);
  const [notifOn, setNotifOn] = useState(() => {
    return getBrowserStorage()?.getItem("retailer_notif") !== "false";
  });

  /* ── Profile State ── */
  const [profileEditing, setProfileEditing] = useState(false);
  const [profileName, setProfileName] = useState("");
  const [profileEmail, setProfileEmail] = useState("");
  const [profileLocation, setProfileLocation] = useState("");
  const [profileCompany, setProfileCompany] = useState("");
  const [profileErrors, setProfileErrors] = useState<ProfileFieldErrors>({});
  const [saveBanner, setSaveBanner] = useState<SaveBanner | null>(null);
  const [savingProfile, setSavingProfile] = useState(false);

  const clearProfileFieldError = useCallback((field: keyof ProfileFieldErrors) => {
    setProfileErrors((prev) => ({ ...prev, [field]: undefined }));
  }, []);

  useEffect(() => {
    const storage = getBrowserStorage();
    if (storage) {
      try {
        const p: RetailerProfile = JSON.parse(
          storage.getItem("retailer_profile") || "{}",
        );
        if (p.name) setProfileName(p.name);
        if (p.company) setProfileCompany(p.company);
        if (p.email) setProfileEmail(p.email);
      } catch {
        /* ignore */
      }
    }
    apiFetch("/v1/retailer/profile")
      .then(async (res) => {
        if (!res.ok) return;
        const data = await res.json();
        if (data.name) setProfileName(data.name);
        if (data.company) setProfileCompany(data.company);
        if (data.phone) setProfileEmail(data.phone);
        if (data.location) setProfileLocation(data.location);
      })
      .catch(() => {});
  }, []);

  const saveProfile = useCallback(async () => {
    const validation = validateProfileFields(profileName, profileCompany);
    if (validation.name || validation.company) {
      setProfileErrors(validation);
      setSaveBanner({
        kind: "error",
        message: "Fix validation errors before saving profile changes.",
      });
      return;
    }

    setProfileErrors({});
    setSaveBanner(null);
    setSavingProfile(true);
    try {
      const res = await apiFetch("/v1/retailer/profile", {
        method: "PUT",
        body: JSON.stringify({
          name: profileName,
          company: profileCompany,
          location: profileLocation,
        }),
      });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || "Profile update failed.");
      }

      setProfileEditing(false);
      setSaveBanner({ kind: "success", message: "Profile saved successfully." });
      const storage = getBrowserStorage();
      if (storage) {
        try {
          const existing = JSON.parse(storage.getItem("retailer_profile") || "{}");
          storage.setItem(
            "retailer_profile",
            JSON.stringify({
              ...existing,
              name: profileName,
              company: profileCompany,
            }),
          );
        } catch {
          /* ignore */
        }
      }
    } catch (err) {
      setSaveBanner({
        kind: "error",
        message: normalizeErrorMessage(err, "Failed to save profile. Please retry."),
      });
    } finally {
      setSavingProfile(false);
    }
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
      setSaveBanner({ kind: "success", message: "Global AI settings updated." });
    } catch (err) {
      setSaveBanner({
        kind: "error",
        message: normalizeErrorMessage(err, "Failed to update global AI settings."),
      });
    } finally {
      setSavingGlobal(false);
    }
  }, [autoOrder, mutate]);

  const toggleOverride = useCallback(
    async (level: string, id: string, enabled: boolean) => {
      setSavingId(id);
      try {
        await apiFetch(`/v1/retailer/settings/auto-order/${level}/${id}`, {
          method: "PATCH",
          body: JSON.stringify({ enabled }),
        });
        mutate();
      } catch (err) {
        setSaveBanner({
          kind: "error",
          message: normalizeErrorMessage(err, "Failed to update override settings."),
        });
      } finally {
        setSavingId(null);
      }
    },
    [mutate],
  );

  return (
    <div
      className="min-h-full p-6 md:p-8"
      style={{ background: "var(--desk-canvas)" }}
    >
      <header className="mb-8 max-w-5xl mx-auto">
        <h1 className="md-typescale-display-small font-bold tracking-tight text-[var(--desk-text-primary)]">
          System Configuration
        </h1>
        <p className="mt-1 md-typescale-body-large text-[var(--desk-text-secondary)]">
          Account identity, notification parameters, and AI-driven logic
          controls.
        </p>
      </header>

      <div className="max-w-5xl mx-auto mb-6 space-y-3">
        {error && (
          <div className="flex items-start gap-2 rounded-xl border border-red-200 bg-red-50 p-3 text-red-700">
            <AlertTriangle size={16} className="mt-0.5 shrink-0" />
            <p className="md-typescale-body-small font-bold">
              {normalizeErrorMessage(error, "Unable to load AI settings.")}
            </p>
          </div>
        )}
        {saveBanner && (
          <div
            className={`flex items-start gap-2 rounded-xl border p-3 ${
              saveBanner.kind === "success"
                ? "border-emerald-200 bg-emerald-50 text-emerald-700"
                : "border-red-200 bg-red-50 text-red-700"
            }`}
          >
            {saveBanner.kind === "success" ? (
              <CheckCircle2 size={16} className="mt-0.5 shrink-0" />
            ) : (
              <AlertTriangle size={16} className="mt-0.5 shrink-0" />
            )}
            <p className="md-typescale-body-small font-bold">{saveBanner.message}</p>
          </div>
        )}
      </div>

      <AnimatePresence mode="popLayout">
        {loading ? (
          <div className="max-w-5xl mx-auto grid grid-cols-1 md:grid-cols-2 gap-8">
            <div className="h-64 rounded-3xl animate-pulse bg-[var(--desk-surface-subtle)] border border-[var(--desk-border)]" />
            <div className="h-64 rounded-3xl animate-pulse bg-[var(--desk-surface-subtle)] border border-[var(--desk-border)]" />
          </div>
        ) : (
          <motion.div
            layout
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            className="max-w-5xl mx-auto space-y-12"
          >
            {/* Identity & Preferences */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
              {/* Profile Card */}
              <section className="flex flex-col gap-4">
                <div className="flex items-center justify-between">
                  <h2 className="md-typescale-title-large font-bold text-[var(--desk-text-primary)] flex items-center gap-2">
                    <User
                      size={20}
                      className="text-[var(--desk-text-tertiary)]"
                    />{" "}
                    Profile Node
                  </h2>
                  <button
                    onClick={() => setProfileEditing(!profileEditing)}
                    className="text-[var(--desk-accent)] md-typescale-label-small font-bold uppercase tracking-widest hover:underline"
                  >
                    {profileEditing ? "Cancel" : "Edit"}
                  </button>
                </div>
                <div className="bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl p-6 shadow-[var(--shadow-sm)] flex flex-col gap-6">
                  <div className="grid grid-cols-1 gap-6">
                    <ProfileField
                      label="Entity Name"
                      value={profileName}
                      icon={User}
                      editing={profileEditing}
                      errorMessage={profileErrors.name}
                      onChange={(value) => {
                        setProfileName(value);
                        clearProfileFieldError("name");
                      }}
                    />
                    <ProfileField
                      label="Company"
                      value={profileCompany}
                      icon={Building2}
                      editing={profileEditing}
                      errorMessage={profileErrors.company}
                      onChange={(value) => {
                        setProfileCompany(value);
                        clearProfileFieldError("company");
                      }}
                    />
                    <ProfileField
                      label="Primary Contact"
                      value={profileEmail}
                      icon={Mail}
                      editing={false}
                      onChange={setProfileEmail}
                    />
                    <ProfileField
                      label="Operational Region"
                      value={profileLocation}
                      icon={MapPin}
                      editing={profileEditing}
                      onChange={setProfileLocation}
                    />
                  </div>
                  {profileEditing && (
                    <Button
                      variant="primary"
                      onPress={saveProfile}
                      isDisabled={savingProfile}
                      className="w-full bg-[var(--desk-text-primary)] text-white font-bold h-11 rounded-xl shadow-lg transition-all active:scale-95"
                    >
                      {savingProfile ? (
                        <Loader2 size={18} className="animate-spin" />
                      ) : (
                        "Verify & Save Changes"
                      )}
                    </Button>
                  )}
                </div>
              </section>

              {/* Preferences */}
              <section className="flex flex-col gap-4">
                <h2 className="md-typescale-title-large font-bold text-[var(--desk-text-primary)] flex items-center gap-2">
                  <Settings
                    size={20}
                    className="text-[var(--desk-text-tertiary)]"
                  />{" "}
                  Logic Prefs
                </h2>
                <div className="bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl p-6 shadow-[var(--shadow-sm)] flex flex-col gap-2">
                  <div className="flex items-center justify-between py-4 border-b border-[var(--desk-border)]">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-xl bg-[var(--desk-surface-subtle)] flex items-center justify-center text-[var(--desk-text-tertiary)]">
                        <Bell size={18} />
                      </div>
                      <div>
                        <span className="md-typescale-body-medium font-bold text-[var(--desk-text-primary)] block">
                          Push Notifications
                        </span>
                        <span className="text-[10px] font-bold text-[var(--desk-text-tertiary)] uppercase tracking-widest">
                          Real-time alerts
                        </span>
                      </div>
                    </div>
                    <Toggle
                      on={notifOn}
                      onToggle={() => {
                        const next = !notifOn;
                        setNotifOn(next);
                        getBrowserStorage()?.setItem(
                          "retailer_notif",
                          String(next),
                        );
                      }}
                    />
                  </div>
                  <div className="flex items-center justify-between py-4">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-xl bg-[var(--desk-surface-subtle)] flex items-center justify-center text-[var(--desk-text-tertiary)]">
                        <CreditCard size={18} />
                      </div>
                      <div>
                        <span className="md-typescale-body-medium font-bold text-[var(--desk-text-primary)] block">
                          Settlement Priority
                        </span>
                        <span className="text-[10px] font-bold text-[var(--desk-text-tertiary)] uppercase tracking-widest">
                          Managed by payment policy
                        </span>
                      </div>
                    </div>
                    <Toggle on={true} onToggle={() => {}} disabled={true} />
                  </div>
                </div>
              </section>
            </div>

            {/* AI Settings */}
            <div className="space-y-6">
              <h2 className="md-typescale-title-large font-bold text-[var(--desk-text-primary)] flex items-center gap-2">
                <Brain size={24} className="text-[var(--desk-accent)]" /> Neural
                Replenishment
              </h2>

              {autoOrder && (
                <div className="flex flex-col gap-4">
                  {/* Global Auto-Order */}
                  <div className="bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-3xl p-8 shadow-md relative overflow-hidden group">
                    <div className="absolute right-0 top-0 w-32 h-32 bg-[var(--desk-accent)] opacity-5 rotate-12 translate-x-8 -translate-y-8" />
                    <div className="flex items-center justify-between gap-8 relative z-10">
                      <div className="max-w-2xl">
                        <h3 className="md-typescale-title-large font-bold text-[var(--desk-text-primary)] mb-2">
                          Global AI Orchestration
                        </h3>
                        <p className="md-typescale-body-large text-[var(--desk-text-secondary)]">
                          {autoOrder.global_enabled
                            ? "AI is currently managing node replenishment based on network demand and predictive signals."
                            : "Predictive reordering is disabled. All procurement actions must be executed manually."}
                        </p>
                      </div>
                      <Toggle
                        on={autoOrder.global_enabled}
                        onToggle={toggleGlobal}
                        disabled={savingGlobal}
                      />
                    </div>
                    {autoOrder.analytics_start_date && (
                      <div className="mt-8 pt-6 border-t border-[var(--desk-border)] flex items-center gap-2 text-[var(--desk-text-tertiary)]">
                        <Info size={14} />
                        <span className="md-typescale-label-small uppercase font-bold tracking-widest">
                          Analytics active since{" "}
                          {new Date(
                            autoOrder.analytics_start_date,
                          ).toLocaleDateString()}
                        </span>
                      </div>
                    )}
                  </div>

                  {/* Overrides */}
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <OverrideSection
                      title="Node Overrides"
                      icon={Building2}
                      items={autoOrder.supplier_overrides}
                      getId={(s) => s.supplier_id}
                      getLabel={(s) => s.supplier_id}
                      getHasHistory={(s) => s.has_history}
                      getAnalyticsDate={(s) => s.analytics_start_date}
                      rowIcon={Building2}
                      storageKey="retailer_settings_section_supplier_overrides"
                      onToggle={(id, enabled) =>
                        toggleOverride("supplier", id, enabled)
                      }
                      savingId={savingId}
                    />
                    <OverrideSection
                      title="Category Overrides"
                      icon={Layers}
                      items={autoOrder.category_overrides}
                      getId={(c) => c.category_id}
                      getLabel={(c) => c.category_id}
                      getHasHistory={(c) => c.has_history}
                      getAnalyticsDate={(c) => c.analytics_start_date}
                      rowIcon={Layers}
                      storageKey="retailer_settings_section_category_overrides"
                      onToggle={(id, enabled) =>
                        toggleOverride("category", id, enabled)
                      }
                      savingId={savingId}
                    />
                  </div>
                </div>
              )}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}

function ProfileField({
  label,
  value,
  icon: Icon,
  editing,
  errorMessage,
  onChange,
}: {
  label: string;
  value: string;
  icon: React.ElementType;
  editing: boolean;
  errorMessage?: string;
  onChange: (v: string) => void;
}) {
  return (
    <div className="space-y-1.5">
      <div className="flex items-center gap-2 text-[var(--desk-text-tertiary)]">
        <Icon size={14} />
        <span className="md-typescale-label-small font-bold uppercase tracking-widest">
          {label}
        </span>
      </div>
      {editing ? (
        <>
          <input
            value={value}
            onChange={(e) => onChange(e.target.value)}
            aria-invalid={Boolean(errorMessage)}
            className={`w-full h-11 px-4 bg-[var(--desk-canvas)] rounded-xl outline-none transition-all md-typescale-body-medium font-bold text-[var(--desk-text-primary)] ${
              errorMessage
                ? "ring-2 ring-red-200 border border-red-300"
                : "focus:ring-2 focus:ring-[var(--desk-accent-soft)]"
            }`}
          />
          {errorMessage && (
            <p className="mt-1 flex items-center gap-1 text-[11px] font-bold text-red-600 uppercase tracking-wide">
              <AlertTriangle size={10} />
              {errorMessage}
            </p>
          )}
        </>
      ) : (
        <p className="md-typescale-body-large font-bold text-[var(--desk-text-primary)] pl-0.5">
          {value || "UNSET"}
        </p>
      )}
    </div>
  );
}
