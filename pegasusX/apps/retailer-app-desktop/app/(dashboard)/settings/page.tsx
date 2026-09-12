"use client";

import { usePortalT } from "@/lib/i18n";
import { useState, useCallback, useEffect, useMemo } from "react";
import { useRetailerSessionReconcile } from "../../../lib/use-retailer-session-reconcile";
import { useRouter } from "next/navigation";
import {
  User,
  Mail,
  MapPin,
  Globe,
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
  RefreshCw,
  WifiOff,
  Clock,
  Users,
} from "lucide-react";
import { Chip, Skeleton } from "@heroui/react";
import { PageChrome } from "@/components/PageChrome";
import { CreditProfileCard } from "@/components/CreditProfileCard";
import { LoyaltyCard } from "@/components/LoyaltyCard";
import { motion, AnimatePresence } from "framer-motion";
import { useLiveData } from "../../../lib/hooks";
import { apiFetch } from "../../../lib/auth";
import { getPricingRules } from "../../../lib/api";
import { retailerProfileUpdateKey } from "@pegasusx/api-core";
import { useOptionalWebSocket } from "../../../lib/ws";
import { BentoGrid, BentoCard } from "../../../components/BentoGrid";
import { getRetailerProfile, getRetailerId, mergeRetailerProfile } from "@/lib/retailer-profile";
import type { AutoOrderSettings } from "../../../lib/types";
import {
  normalizeReceivingWindow,
  validateReceivingWindowField,
} from "../../../lib/receiving-window";
import { Toggle, OverrideRow, OverrideSection } from "../../../components/settings/OverrideComponents";
import { ProfileField, ProfileTimeField } from "../../../components/settings/ProfileFields";

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
  receivingWindowOpen?: string;
  receivingWindowClose?: string;
};

type SaveBanner = {
  kind: "success" | "error";
  message: string;
};

type LoadIssue = "restricted" | "offline" | "error";

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
  receivingWindowOpen: string,
  receivingWindowClose: string,
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

  const openError = validateReceivingWindowField(receivingWindowOpen);
  if (openError) errors.receivingWindowOpen = openError;
  const closeError = validateReceivingWindowField(receivingWindowClose);
  if (closeError) errors.receivingWindowClose = closeError;

  return errors;
}

/* ── Main Page ── */

export default function SettingsPage() {
  const t = usePortalT();
  const {
    data: autoOrder,
    loading,
    error,
    isRefreshing,
    mutate: mutateAutoOrder,
  } = useLiveData<AutoOrderSettings>("/v1/retailer/settings/auto-order");
  const ws = useOptionalWebSocket();
  const router = useRouter();
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
  const [profileCountryCode, setProfileCountryCode] = useState("");
  const [profileCompany, setProfileCompany] = useState("");
  const [profileReceivingWindowOpen, setProfileReceivingWindowOpen] =
    useState("");
  const [profileReceivingWindowClose, setProfileReceivingWindowClose] =
    useState("");
  const [profileErrors, setProfileErrors] = useState<ProfileFieldErrors>({});
  const [saveBanner, setSaveBanner] = useState<SaveBanner | null>(null);
  const [savingProfile, setSavingProfile] = useState(false);
  const [profileRetailerId, setProfileRetailerId] = useState("");
  const [profileLoading, setProfileLoading] = useState(false);
  const [profileError, setProfileError] = useState<string | null>(null);
  const [pricingRulesSummary, setPricingRulesSummary] = useState<string | null>(null);
  const [pricingRulesError, setPricingRulesError] = useState<string | null>(null);

  useEffect(() => {
    void getPricingRules()
      .then(async (res) => {
        if (!res.ok) {
          throw new Error(`Pricing rules unavailable (${res.status})`);
        }
        const data = (await res.json()) as { summary?: string; status?: string };
        setPricingRulesSummary(
          data.summary || data.status || "Supplier pricing rules are active for your account.",
        );
        setPricingRulesError(null);
      })
      .catch((err: unknown) => {
        setPricingRulesSummary(null);
        setPricingRulesError(
          err instanceof Error ? err.message : t("retailer_desktop.residual.text.pricing_rules_unavailable"),
        );
      });
  }, []);

  const clearProfileFieldError = useCallback((field: keyof ProfileFieldErrors) => {
    setProfileErrors((prev) => ({ ...prev, [field]: undefined }));
  }, []);

  const loadProfile = useCallback(async () => {
    setProfileError(null);
    setProfileLoading(true);
    try {
      const res = await apiFetch("/v1/retailer/profile");
      if (!res.ok) {
        if (res.status === 401 || res.status === 403) {
          throw new Error("Profile access restricted.");
        }
        const text = await res.text();
        throw new Error(text || `Profile fetch failed (${res.status}).`);
      }
      const data = await res.json();
      if (typeof data.retailer_id === "string") {
        setProfileRetailerId(data.retailer_id);
      } else if (typeof data.id === "string") {
        setProfileRetailerId(data.id);
      }
      if (data.name) setProfileName(data.name);
      if (data.company) setProfileCompany(data.company);
      if (data.phone) setProfileEmail(data.phone);
      if (data.location) setProfileLocation(data.location);
      if (data.country_code) setProfileCountryCode(data.country_code);
      setProfileReceivingWindowOpen(
        typeof data.receiving_window_open === "string"
          ? data.receiving_window_open
          : "",
      );
      setProfileReceivingWindowClose(
        typeof data.receiving_window_close === "string"
          ? data.receiving_window_close
          : "",
      );
      await mergeRetailerProfile({
        id: typeof data.retailer_id === "string" ? data.retailer_id : data.id,
        name: data.name,
        company: data.company,
        email: data.phone ?? data.email,
        country_code: data.country_code,
        receiving_window_open: data.receiving_window_open,
        receiving_window_close: data.receiving_window_close,
      });
    } catch (err) {
      setProfileError(
        normalizeErrorMessage(err, "Unable to load retailer profile."),
      );
    } finally {
      setProfileLoading(false);
    }
  }, []);

  const refreshAll = useCallback(() => {
    void mutateAutoOrder();
    void loadProfile();
  }, [loadProfile, mutateAutoOrder]);

  useEffect(() => {
    const p = getRetailerProfile();
    if (p) {
      if (p.name) setProfileName(p.name);
      if (p.company) setProfileCompany(p.company);
      if (p.email) setProfileEmail(p.email);
      if (p.country_code) setProfileCountryCode(p.country_code);
    }
    void loadProfile();
  }, [loadProfile]);

  const saveProfile = useCallback(async () => {
    const validation = validateProfileFields(
      profileName,
      profileCompany,
      profileReceivingWindowOpen,
      profileReceivingWindowClose,
    );
    if (
      validation.name ||
      validation.company ||
      validation.receivingWindowOpen ||
      validation.receivingWindowClose
    ) {
      setProfileErrors(validation);
      setSaveBanner({
        kind: "error",
        message: t("retailer_desktop.residual.text.fix_validation_errors_before_saving_profile_changes"),
      });
      return;
    }

    setProfileErrors({});
    setSaveBanner(null);
    setSavingProfile(true);
    try {
      const payload = {
        name: profileName,
        company: profileCompany,
        location: profileLocation,
        country_code: profileCountryCode,
        receiving_window_open: normalizeReceivingWindow(
          profileReceivingWindowOpen,
        ),
        receiving_window_close: normalizeReceivingWindow(
          profileReceivingWindowClose,
        ),
      };
      const retailerId =
        profileRetailerId || getRetailerId();
      const fingerprint = Object.entries(payload)
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([key, value]) => `${key}=${value}`)
        .join("|");
      const res = await apiFetch("/v1/retailer/profile", {
        method: "PUT",
        headers: retailerId
          ? {
              "Idempotency-Key": retailerProfileUpdateKey(retailerId, fingerprint),
            }
          : undefined,
        body: JSON.stringify(payload),
      });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || "Profile update failed.");
      }

      setProfileEditing(false);
      setSaveBanner({ kind: "success", message: t("retailer_desktop.residual.text.profile_saved_successfully") });
      await mergeRetailerProfile({
        name: profileName,
        company: profileCompany,
        country_code: profileCountryCode,
        receiving_window_open: normalizeReceivingWindow(profileReceivingWindowOpen),
        receiving_window_close: normalizeReceivingWindow(profileReceivingWindowClose),
      });
      await loadProfile();
    } catch (err) {
      setSaveBanner({
        kind: "error",
        message: normalizeErrorMessage(err, "Failed to save profile. Please retry."),
      });
    } finally {
      setSavingProfile(false);
    }
  }, [
    profileName,
    profileCompany,
    profileLocation,
    profileCountryCode,
    profileReceivingWindowOpen,
    profileReceivingWindowClose,
    loadProfile,
  ]);

  const toggleGlobal = useCallback(async () => {
    if (!autoOrder) return;
    setSavingGlobal(true);
    try {
      await apiFetch("/v1/retailer/settings/auto-order/global", {
        method: "PATCH",
        body: JSON.stringify({
          global_auto_order_enabled: !autoOrder.global_enabled,
        }),
      });
      mutateAutoOrder();
      setSaveBanner({ kind: "success", message: t("retailer_desktop.residual.text.global_ai_settings_updated") });
    } catch (err) {
      setSaveBanner({
        kind: "error",
        message: normalizeErrorMessage(err, "Failed to update global AI settings."),
      });
    } finally {
      setSavingGlobal(false);
    }
  }, [autoOrder, mutateAutoOrder]);

  const toggleOverride = useCallback(
    async (level: string, id: string, enabled: boolean) => {
      setSavingId(id);
      try {
        await apiFetch(`/v1/retailer/settings/auto-order/${level}/${id}`, {
          method: "PATCH",
          body: JSON.stringify({ enabled }),
        });
        mutateAutoOrder();
      } catch (err) {
        setSaveBanner({
          kind: "error",
          message: normalizeErrorMessage(err, "Failed to update override settings."),
        });
      } finally {
        setSavingId(null);
      }
    },
    [mutateAutoOrder],
  );

  const loadIssue = useMemo<LoadIssue | null>(() => {
    const message = error?.message ?? profileError ?? "";
    const status = (error as (Error & { status?: number }) | null)?.status;
    if (!message && status == null) return null;
    if (status === 401 || status === 403 || /forbidden|restricted|access/i.test(message)) {
      return "restricted";
    }
    if (
      (typeof navigator !== "undefined" && !navigator.onLine) ||
      /failed to fetch|network|load failed|offline/i.test(message)
    ) {
      return "offline";
    }
    return "error";
  }, [error, profileError]);

  const isSyncing = isRefreshing || profileLoading;

  const syncBanner = useMemo(() => {
    if (loadIssue === "restricted") {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: t("retailer_desktop.residual.text.settings_access_is_partially_restricted_for_this_account"),
      };
    }
    if (loadIssue === "offline") {
      return {
        kind: "warning" as const,
        icon: WifiOff,
        message: t("retailer_desktop.residual.text.offline_mode_active_showing_latest_cached_configuration"),
      };
    }
    if (loadIssue === "error") {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: t("retailer_desktop.residual.text.settings_sync_degraded_auto_retry_is_active"),
      };
    }
    if (ws && !ws.isConnected) {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: t("retailer_desktop.residual.text.live_socket_reconnecting_setting_events_may_be_delayed"),
      };
    }
    if (isSyncing && !loading) {
      return {
        kind: "refreshing" as const,
        icon: RefreshCw,
        message: t("retailer_desktop.residual.text.syncing_settings_feeds"),
      };
    }
    return null;
  }, [isSyncing, loadIssue, loading, ws]);

  useRetailerSessionReconcile(() => {
    void mutateAutoOrder();
  });

  return (
    <div
      className="min-h-full p-6 md:p-8"
      style={{ background: "var(--desk-canvas)" }}
    >
      <PageChrome
        icon="settings"
        title={t("supplier_portal.configuration.system.title")}
        description={t("retailer_desktop.residual.text.account_identity_notification_parameters_and_ai_driven_logic_con")}
        loading={loading}
        skeletonVariant="form"
        actions={
          <button
            type="button"
            disabled={isSyncing}
            onClick={refreshAll}
            className="portal-btn portal-btn--ghost h-10 px-5 rounded-xl font-light"
          >
            <RefreshCw
              size={16}
              className={`mr-2 ${isSyncing ? "animate-spin" : ""}`}
            />
            {isSyncing ? "Syncing" : "Sync"}
          </button>
        }
      >
      <div className="max-w-5xl mx-auto mb-6 space-y-3">
        {syncBanner && (
          <div
            className={`flex items-center justify-between gap-3 rounded-2xl border px-4 py-3 ${
              syncBanner.kind === "refreshing"
                ? "border-[var(--desk-info)]/30 bg-[var(--desk-info)]/5 text-[var(--desk-info)]"
                : "border-[var(--desk-warning)]/30 bg-[var(--desk-warning)]/10 text-[var(--desk-warning)]"
            }`}
          >
            <div className="flex items-center gap-2">
              <syncBanner.icon
                size={16}
                className={syncBanner.kind === "refreshing" ? "animate-spin" : ""}
              />
              <p className="md-typescale-body-small font-light uppercase tracking-wide">
                {syncBanner.message}
              </p>
            </div>
            {syncBanner.kind !== "refreshing" && (
              <button
                onClick={refreshAll}
                className="rounded-lg border border-current/30 px-3 py-1 text-[11px] font-light uppercase tracking-wide hover:bg-current/10"
              >
                Retry
              </button>
            )}
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
            <p className="md-typescale-body-small font-light">{saveBanner.message}</p>
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
                  <h2 className="md-typescale-title-large font-light text-[var(--desk-text-primary)] flex items-center gap-2">
                    <User
                      size={20}
                      className="text-[var(--desk-text-tertiary)]"
                    />{" "}
                    Profile Node
                  </h2>
                  <button
                    onClick={() => setProfileEditing(!profileEditing)}
                    className="text-[var(--desk-accent)] md-typescale-label-small font-light uppercase tracking-widest hover:underline"
                  >
                    {profileEditing ? "Cancel" : "Edit"}
                  </button>
                </div>
                <div className="bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl p-6 shadow-[var(--shadow-sm)] flex flex-col gap-6">
                  <div className="grid grid-cols-1 gap-6">
                    <ProfileField
                      label={t("retailer_desktop.residual.text.entity_name")}
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
                      label={t("retailer_desktop.residual.text.company")}
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
                      label={t("retailer_desktop.residual.text.primary_contact")}
                      value={profileEmail}
                      icon={Mail}
                      editing={false}
                      onChange={setProfileEmail}
                    />
                    <ProfileField
                      label={t("retailer_desktop.residual.text.operational_region")}
                      value={profileLocation}
                      icon={MapPin}
                      editing={profileEditing}
                      onChange={setProfileLocation}
                    />
                    <ProfileField
                      label={t("supplier_portal.configuration.countries.field.country_code")}
                      value={profileCountryCode}
                      icon={Globe}
                      editing={profileEditing}
                      onChange={setProfileCountryCode}
                    />
                    <ProfileTimeField
                      label={t("retailer_desktop.residual.text.receiving_window_opens")}
                      value={profileReceivingWindowOpen}
                      icon={Clock}
                      editing={profileEditing}
                      errorMessage={profileErrors.receivingWindowOpen}
                      onChange={(value) => {
                        setProfileReceivingWindowOpen(value);
                        clearProfileFieldError("receivingWindowOpen");
                      }}
                    />
                    <ProfileTimeField
                      label={t("retailer_desktop.residual.text.receiving_window_closes")}
                      value={profileReceivingWindowClose}
                      icon={Clock}
                      editing={profileEditing}
                      errorMessage={profileErrors.receivingWindowClose}
                      onChange={(value) => {
                        setProfileReceivingWindowClose(value);
                        clearProfileFieldError("receivingWindowClose");
                      }}
                    />
                    {!profileEditing && (
                      <p className="md-typescale-label-small text-[var(--desk-text-tertiary)] -mt-2">
                        Dispatch SLA uses this window when scheduling deliveries.
                      </p>
                    )}
                  </div>
                  {profileEditing && (
                    <button
                      type="button"
                      onClick={saveProfile}
                      disabled={savingProfile}
                      className="portal-btn portal-btn--primary w-full font-light h-11 rounded-xl shadow-lg transition-all active:scale-95"
                    >
                      {savingProfile ? (
                        <Loader2 size={18} className="animate-spin" />
                      ) : (
                        "Verify & Save Changes"
                      )}
                    </button>
                  )}
                </div>
              </section>

              <section className="flex flex-col gap-4 md:col-span-2">
                <h2 className="md-typescale-title-large font-light text-[var(--desk-text-primary)] flex items-center gap-2">
                  <CreditCard
                    size={20}
                    className="text-[var(--desk-text-tertiary)]"
                  />{" "}
                  Billing & Access
                </h2>
                <CreditProfileCard className="mb-1" />
                <LoyaltyCard />
                <div className="bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl p-4 shadow-[var(--shadow-sm)] mb-3">
                  <p className="text-[10px] font-black uppercase tracking-widest text-[var(--desk-text-tertiary)] mb-2">
                    Pricing rules (read-only)
                  </p>
                  {pricingRulesError ? (
                    <p className="text-sm text-orange-700">{pricingRulesError}</p>
                  ) : (
                    <p className="text-sm text-[var(--desk-text-secondary)]">
                      {pricingRulesSummary ?? "Loading pricing rules…"}
                    </p>
                  )}
                </div>
                <div className="bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl p-2 shadow-[var(--shadow-sm)]">
                  <button
                    type="button"
                    onClick={() => router.push("/settings/capabilities")}
                    className="w-full flex items-center justify-between px-4 py-4 rounded-xl hover:bg-[var(--desk-surface-subtle)] transition-colors"
                  >
                    <div className="flex items-center gap-3 text-left">
                      <div className="w-10 h-10 rounded-xl bg-[var(--desk-surface-subtle)] flex items-center justify-center text-[var(--desk-text-tertiary)]">
                        <Layers size={18} />
                      </div>
                      <div>
                        <span className="md-typescale-body-medium font-light text-[var(--desk-text-primary)] block">
                          Store capabilities
                        </span>
                        <span className="text-[10px] font-light text-[var(--desk-text-tertiary)] uppercase tracking-widest">
                          Team, stock, POS packs (Retail OS)
                        </span>
                      </div>
                    </div>
                    <ChevronRight size={18} className="text-[var(--desk-text-tertiary)]" />
                  </button>
                  <button
                    type="button"
                    onClick={() => router.push("/settings/locations")}
                    className="w-full flex items-center justify-between px-4 py-4 rounded-xl hover:bg-[var(--desk-surface-subtle)] transition-colors"
                  >
                    <div className="flex items-center gap-3 text-left">
                      <div className="w-10 h-10 rounded-xl bg-[var(--desk-surface-subtle)] flex items-center justify-center text-[var(--desk-text-tertiary)]">
                        <MapPin size={18} />
                      </div>
                      <div>
                        <span className="md-typescale-body-medium font-light text-[var(--desk-text-primary)] block">
                          Locations
                        </span>
                        <span className="text-[10px] font-light text-[var(--desk-text-tertiary)] uppercase tracking-widest">
                          Branches, primary store, checkout branch
                        </span>
                      </div>
                    </div>
                    <ChevronRight size={18} className="text-[var(--desk-text-tertiary)]" />
                  </button>
                  <button
                    type="button"
                    onClick={() => router.push("/settings/team")}
                    className="w-full flex items-center justify-between px-4 py-4 rounded-xl hover:bg-[var(--desk-surface-subtle)] transition-colors"
                  >
                    <div className="flex items-center gap-3 text-left">
                      <div className="w-10 h-10 rounded-xl bg-[var(--desk-surface-subtle)] flex items-center justify-center text-[var(--desk-text-tertiary)]">
                        <Users size={18} />
                      </div>
                      <div>
                        <span className="md-typescale-body-medium font-light text-[var(--desk-text-primary)] block">
                          Team
                        </span>
                        <span className="text-[10px] font-light text-[var(--desk-text-tertiary)] uppercase tracking-widest">
                          Staff roles, invites, deactivate
                        </span>
                      </div>
                    </div>
                    <ChevronRight size={18} className="text-[var(--desk-text-tertiary)]" />
                  </button>
                  <button
                    type="button"
                    onClick={() => router.push("/settings/family")}
                    className="w-full flex items-center justify-between px-4 py-4 rounded-xl hover:bg-[var(--desk-surface-subtle)] transition-colors"
                  >
                    <div className="flex items-center gap-3 text-left">
                      <div className="w-10 h-10 rounded-xl bg-[var(--desk-surface-subtle)] flex items-center justify-center text-[var(--desk-text-tertiary)]">
                        <Users size={18} />
                      </div>
                      <div>
                        <span className="md-typescale-body-medium font-light text-[var(--desk-text-primary)] block">
                          Family contacts
                        </span>
                        <span className="text-[10px] font-light text-[var(--desk-text-tertiary)] uppercase tracking-widest">
                          Legacy name/phone list (use Team for logins)
                        </span>
                      </div>
                    </div>
                    <ChevronRight size={18} className="text-[var(--desk-text-tertiary)]" />
                  </button>
                </div>
              </section>

              {/* Preferences */}
              <section className="flex flex-col gap-4">
                <h2 className="md-typescale-title-large font-light text-[var(--desk-text-primary)] flex items-center gap-2">
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
                        <span className="md-typescale-body-medium font-light text-[var(--desk-text-primary)] block">
                          Push Notifications
                        </span>
                        <span className="text-[10px] font-light text-[var(--desk-text-tertiary)] uppercase tracking-widest">
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
                  {/* G3.C: local browser mute only — not a server preference API. */}
                  <p className="text-[10px] font-light text-[var(--desk-text-tertiary)] px-1 pb-2">
                    Desktop mute stores locally. Server push registration uses device-token
                    APIs; there is no durable notification-preferences profile yet.
                  </p>
                  {/* Settlement Priority decorative toggle removed (G3.C — was disabled theatre). */}
                </div>
              </section>
            </div>

            {/* AI Settings */}
            <div className="space-y-6">
              <h2 className="md-typescale-title-large font-light text-[var(--desk-text-primary)] flex items-center gap-2">
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
                        <h3 className="md-typescale-title-large font-light text-[var(--desk-text-primary)] mb-2">
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
                        <span className="md-typescale-label-small uppercase font-light tracking-widest">
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
                      title={t("retailer_desktop.settings.text.node_overrides")}
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
                      title={t("retailer_desktop.settings.text.category_overrides")}
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
      </PageChrome>
    </div>
  );
}

