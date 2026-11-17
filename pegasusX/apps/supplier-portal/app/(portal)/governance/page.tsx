"use client";

import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "next/navigation";
import { ApiError } from "@pegasusx/api-core";
import { createSupplierApi } from "@/lib/api";
import { decodeJwtPayload, getSupplierToken, readTokenFromCookie } from "@/lib/auth";
import { resolvePortalSessionRole } from "@/lib/connectors-auditor-api";
import {
  createSupplierAuditIamApi,
  currentUtcDayBucket,
  type AuditActorType,
  type AuditAnchor,
  type AuditEvent,
  type AuditLegalHold,
  type ChainVerifyResult,
  type IamPermissionDefinition,
  type IamRoleDefinition,
  type IamUserRoleAssignment,
  type RoleAssignmentAction,
  type SodPolicy,
  type SodViolation,
} from "@/lib/audit-iam-api";
import { createP2bApi, extractP2bError } from "@/lib/p2b-api";
import { usePortalT } from "@/lib/i18n";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { PageChrome } from "@/components/PageChrome";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { PortalInput, PortalSection, PortalSelect } from "@/components/portal";
import StatusBadge from "@/components/StatusBadge";
import Icon from "@/components/Icon";
import { ActorTypeBadge } from "@/components/audit/ObjectHistoryPanel";
import type { CustomRole, ScimGroup, ScimUser } from "@pegasusx/types";

type GovernanceTab = "verify" | "holds" | "roles" | "sod" | "custom_scim";

const DATE_RE = /^\d{4}-\d{2}-\d{2}$/;

function normalizeTab(raw: string | null): GovernanceTab {
  switch ((raw || "").trim().toLowerCase()) {
    case "holds":
      return "holds";
    case "roles":
      return "roles";
    case "sod":
      return "sod";
    case "custom_scim":
    case "custom_roles":
    case "scim":
      return "custom_scim";
    default:
      return "verify";
  }
}

function shortHash(hash?: string): string {
  const clean = (hash || "").trim();
  if (!clean) return "—";
  if (clean.length <= 16) return clean;
  return `${clean.slice(0, 10)}…${clean.slice(-6)}`;
}

function formatError(err: unknown, fallback: string): string {
  if (err instanceof ApiError && err.payload && typeof err.payload === "object") {
    const p = err.payload as Record<string, unknown>;
    const parts = [p.error, p.code, p.message, p.detail, p.permission, p.policy_id]
      .filter((v): v is string => typeof v === "string" && v.trim() !== "");
    if (parts.length > 0) {
      return parts.join(" · ");
    }
  }
  return err instanceof Error ? err.message : fallback;
}

export default function SupplierGovernancePage() {
  return (
    <Suspense fallback={null}>
      <SupplierGovernanceInner />
    </Suspense>
  );
}

function SupplierGovernanceInner() {
  const t = usePortalT();
  const tRef = useRef(t);
  tRef.current = t;
  const searchParams = useSearchParams();
  const supplierApi = useMemo(() => createSupplierApi(), []);
  const auditIamApi = useMemo(() => createSupplierAuditIamApi(supplierApi), [supplierApi]);
  const p2bApi = useMemo(() => createP2bApi(supplierApi), [supplierApi]);

  const [roleResolved, setRoleResolved] = useState(false);
  const [isReadOnlyVisitor, setIsReadOnlyVisitor] = useState(false);

  const [activeTab, setActiveTab] = useState<GovernanceTab>(() =>
    normalizeTab(searchParams.get("tab")),
  );
  useEffect(() => {
    const paramTab = searchParams.get("tab");
    if (paramTab) {
      setActiveTab(normalizeTab(paramTab));
    }
  }, [searchParams]);

  // Chain & Events state
  const [dayBucket, setDayBucket] = useState<string>(() => currentUtcDayBucket());
  const [actorTypeFilter, setActorTypeFilter] = useState<AuditActorType | "">("");
  const [objectFilter, setObjectFilter] = useState<string>("");
  const [verifyResult, setVerifyResult] = useState<ChainVerifyResult | null>(null);
  const [anchors, setAnchors] = useState<AuditAnchor[]>([]);
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [eventsMasked, setEventsMasked] = useState<boolean>(false);

  // Legal Holds state
  const [holds, setHolds] = useState<AuditLegalHold[]>([]);
  const [holdStatusFilter, setHoldStatusFilter] = useState<string>("ALL");
  const [holdObjectType, setHoldObjectType] = useState<string>("order");
  const [holdObjectId, setHoldObjectId] = useState<string>("");
  const [holdReason, setHoldReason] = useState<string>("");
  const [holdExpiresAt, setHoldExpiresAt] = useState<string>("");

  // RBAC & SoD state
  const [roles, setRoles] = useState<IamRoleDefinition[]>([]);
  const [userRoles, setUserRoles] = useState<IamUserRoleAssignment[]>([]);
  const [permissions, setPermissions] = useState<IamPermissionDefinition[]>([]);
  const [assignUserId, setAssignUserId] = useState<string>("");
  const [assignRoleName, setAssignRoleName] = useState<string>("FINANCE_MANAGER");
  const [assignAction, setAssignAction] = useState<RoleAssignmentAction>("ASSIGN");
  const [sodPolicies, setSodPolicies] = useState<SodPolicy[]>([]);
  const [sodViolations, setSodViolations] = useState<SodViolation[]>([]);
  const [violationActorFilter, setViolationActorFilter] = useState<string>("");

  // Wave 4B Custom Roles & SCIM v2 state
  const [customRoles, setCustomRoles] = useState<CustomRole[]>([]);
  const [customRoleName, setCustomRoleName] = useState<string>("");
  const [customRoleDesc, setCustomRoleDesc] = useState<string>("");
  const [customRoleReadOnly, setCustomRoleReadOnly] = useState<boolean>(false);
  const [customRolePerms, setCustomRolePerms] = useState<string[]>([]);
  const [customRoleManualPerms, setCustomRoleManualPerms] = useState<string>("");

  const [scimUsers, setScimUsers] = useState<ScimUser[]>([]);
  const [scimUserName, setScimUserName] = useState<string>("");
  const [scimDisplayName, setScimDisplayName] = useState<string>("");
  const [scimEmail, setScimEmail] = useState<string>("");
  const [scimRoleInput, setScimRoleInput] = useState<string>("");
  const [scimUserActive, setScimUserActive] = useState<boolean>(true);

  const [scimGroups, setScimGroups] = useState<ScimGroup[]>([]);
  const [scimGroupName, setScimGroupName] = useState<string>("");
  const [scimGroupExternalId, setScimGroupExternalId] = useState<string>("");
  const [scimGroupMembersInput, setScimGroupMembersInput] = useState<string>("");

  // Async flags
  const [loading, setLoading] = useState<boolean>(true);
  const [refreshing, setRefreshing] = useState<boolean>(false);
  const [busyAction, setBusyAction] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const syncRoleFromToken = useCallback(() => {
    const token = getSupplierToken() || readTokenFromCookie();
    if (!token) {
      setIsReadOnlyVisitor(false);
      setRoleResolved(true);
      return;
    }
    const roleInfo = resolvePortalSessionRole(decodeJwtPayload(token));
    setIsReadOnlyVisitor(roleInfo.isAuditor || roleInfo.isAgent);
    setRoleResolved(true);
  }, []);

  useEffect(() => {
    syncRoleFromToken();
  }, [syncRoleFromToken]);

  const loadAll = useCallback(
    async (silent = false) => {
      syncRoleFromToken();
      if (!silent) {
        setLoading(true);
      } else {
        setRefreshing(true);
      }
      setError(null);

      const cleanDay = DATE_RE.test(dayBucket.trim()) ? dayBucket.trim() : currentUtcDayBucket();

      const results = await Promise.allSettled([
        auditIamApi.verifyDayChain({ date: cleanDay }),
        auditIamApi.listAnchors({ limit: 30 }),
        auditIamApi.listEvents({
          date: cleanDay,
          actor_type: actorTypeFilter || undefined,
          object: objectFilter.trim() || undefined,
          limit: 100,
        }),
        auditIamApi.listLegalHolds({
          status: holdStatusFilter === "ALL" ? undefined : holdStatusFilter,
          limit: 100,
        }),
        auditIamApi.listRoles(),
        auditIamApi.listPermissions(),
        auditIamApi.listSodPolicies(),
        auditIamApi.listSodViolations({
          actor: violationActorFilter.trim() || undefined,
          limit: 100,
        }),
        p2bApi.listCustomRoles(),
        p2bApi.listScimUsers({ count: 50 }),
        p2bApi.listScimGroups({ count: 50 }),
      ]);

      const [
        verifyRes,
        anchorsRes,
        eventsRes,
        holdsRes,
        rolesRes,
        permsRes,
        policiesRes,
        violationsRes,
        customRolesRes,
        scimUsersRes,
        scimGroupsRes,
      ] = results;

      let firstErr: string | null = null;
      const captureErr = (r: PromiseSettledResult<unknown>) => {
        if (r.status === "rejected" && !firstErr) {
          firstErr = formatError(r.reason, tRef.current("supplier_portal.governance.load_failed"));
        }
      };

      if (verifyRes.status === "fulfilled") setVerifyResult(verifyRes.value);
      else captureErr(verifyRes);

      if (anchorsRes.status === "fulfilled") setAnchors(anchorsRes.value.anchors ?? []);
      else captureErr(anchorsRes);

      if (eventsRes.status === "fulfilled") {
        setEvents(eventsRes.value.events ?? []);
        setEventsMasked(Boolean(eventsRes.value.pii_masked));
      } else captureErr(eventsRes);

      if (holdsRes.status === "fulfilled") setHolds(holdsRes.value.holds ?? []);
      else captureErr(holdsRes);

      if (rolesRes.status === "fulfilled") {
        setRoles(rolesRes.value.roles ?? []);
        setUserRoles(rolesRes.value.user_roles ?? []);
      } else captureErr(rolesRes);

      if (permsRes.status === "fulfilled") setPermissions(permsRes.value.permissions ?? []);
      else captureErr(permsRes);

      if (policiesRes.status === "fulfilled") setSodPolicies(policiesRes.value.policies ?? []);
      else captureErr(policiesRes);

      if (violationsRes.status === "fulfilled") {
        setSodViolations(violationsRes.value.violations ?? []);
      } else captureErr(violationsRes);

      if (customRolesRes.status === "fulfilled" && customRolesRes.value && Array.isArray(customRolesRes.value.custom_roles)) {
        setCustomRoles(customRolesRes.value.custom_roles);
      }

      if (scimUsersRes.status === "fulfilled" && scimUsersRes.value && Array.isArray(scimUsersRes.value.Resources)) {
        setScimUsers(scimUsersRes.value.Resources);
      }

      if (scimGroupsRes.status === "fulfilled" && scimGroupsRes.value && Array.isArray(scimGroupsRes.value.Resources)) {
        setScimGroups(scimGroupsRes.value.Resources);
      }

      if (firstErr) {
        setError(firstErr);
      }

      setLoading(false);
      setRefreshing(false);
    },
    [
      actorTypeFilter,
      auditIamApi,
      dayBucket,
      holdStatusFilter,
      objectFilter,
      p2bApi,
      syncRoleFromToken,
      violationActorFilter,
    ],
  );

  useEffect(() => {
    void loadAll();
  }, [loadAll]);

  useSupplierSessionReconcile(() => {
    void loadAll(true);
  });

  const canMutate = roleResolved && !isReadOnlyVisitor;

  const handleVerifyChain = async () => {
    const cleanDay = dayBucket.trim();
    if (!DATE_RE.test(cleanDay)) return;
    setBusyAction("verify");
    setError(null);
    setNotice(null);
    try {
      const res = await auditIamApi.verifyDayChain({ date: cleanDay });
      setVerifyResult(res);
    } catch (err) {
      setError(formatError(err, t("supplier_portal.governance.load_failed")));
    } finally {
      setBusyAction(null);
    }
  };

  const handleWriteAnchor = async () => {
    if (!canMutate) return;
    const cleanDay = dayBucket.trim();
    if (!DATE_RE.test(cleanDay)) return;
    setBusyAction("anchor");
    setError(null);
    setNotice(null);
    try {
      const anchor = await auditIamApi.writeAnchor({ date: cleanDay });
      setNotice(
        t("supplier_portal.governance.anchor_written", { dayBucket: anchor.day_bucket }),
      );
      await loadAll(true);
    } catch (err) {
      setError(formatError(err, t("supplier_portal.governance.load_failed")));
    } finally {
      setBusyAction(null);
    }
  };

  const handleCreateHold = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canMutate || !holdObjectType.trim() || !holdObjectId.trim() || !holdReason.trim()) return;
    setBusyAction("hold");
    setError(null);
    setNotice(null);
    try {
      let expiresIso: string | undefined;
      if (holdExpiresAt.trim()) {
        const parsed = new Date(holdExpiresAt.trim());
        if (!Number.isNaN(parsed.getTime())) {
          expiresIso = parsed.toISOString();
        }
      }
      const created = await auditIamApi.createLegalHold({
        object_type: holdObjectType.trim(),
        object_id: holdObjectId.trim(),
        reason: holdReason.trim(),
        expires_at: expiresIso,
      });
      setNotice(
        t("supplier_portal.governance.hold_created", { holdId: created.hold_id }),
      );
      setHoldObjectId("");
      setHoldReason("");
      setHoldExpiresAt("");
      await loadAll(true);
    } catch (err) {
      setError(formatError(err, t("supplier_portal.governance.load_failed")));
    } finally {
      setBusyAction(null);
    }
  };

  const handleAssignRole = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canMutate) return;
    const uid = assignUserId.trim();
    if (!uid || !assignRoleName.trim()) return;
    setBusyAction("assign-role");
    setError(null);
    setNotice(null);
    try {
      const res = await auditIamApi.assignUserRole(uid, {
        role_name: assignRoleName.trim(),
        action: assignAction,
      });
      setNotice(
        t("supplier_portal.governance.role_updated", {
          role: res.role_name,
          userId: res.user_id,
        }),
      );
      setAssignUserId("");
      await loadAll(true);
    } catch (err) {
      setError(formatError(err, t("supplier_portal.governance.load_failed")));
    } finally {
      setBusyAction(null);
    }
  };

  const handleTogglePolicy = async (policy: SodPolicy) => {
    if (!canMutate) return;
    setBusyAction(`policy:${policy.policy_id}`);
    setError(null);
    setNotice(null);
    try {
      await auditIamApi.updateSodPolicy(policy.policy_id, {
        enabled: !policy.enabled,
      });
      await loadAll(true);
    } catch (err) {
      setError(formatError(err, t("supplier_portal.governance.load_failed")));
    } finally {
      setBusyAction(null);
    }
  };

  const toggleCustomPermission = (permKey: string) => {
    setCustomRolePerms((prev) =>
      prev.includes(permKey) ? prev.filter((p) => p !== permKey) : [...prev, permKey],
    );
  };

  const handleCreateCustomRole = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canMutate || !customRoleName.trim()) return;
    const extraPerms = customRoleManualPerms
      .split(",")
      .map((s) => s.trim())
      .filter(Boolean);
    const mergedPerms = Array.from(new Set([...customRolePerms, ...extraPerms]));
    if (mergedPerms.length === 0) {
      setError(t("supplier_portal.governance.custom_role_perms_required"));
      return;
    }
    setBusyAction("create-custom-role");
    setError(null);
    setNotice(null);
    try {
      const created = await p2bApi.createCustomRole({
        role_name: customRoleName.trim().toUpperCase(),
        description: customRoleDesc.trim() || undefined,
        permissions: mergedPerms,
        read_only: customRoleReadOnly,
      });
      setCustomRoleName("");
      setCustomRoleDesc("");
      setCustomRolePerms([]);
      setCustomRoleManualPerms("");
      setCustomRoleReadOnly(false);
      setNotice(
        t("supplier_portal.governance.custom_role_created", {
          roleName: created.role_name,
        }),
      );
      await loadAll(true);
    } catch (err) {
      setError(extractP2bError(err, t("supplier_portal.governance.load_failed")).message);
    } finally {
      setBusyAction(null);
    }
  };

  const handleDeleteCustomRole = async (roleName: string) => {
    if (!canMutate) return;
    setBusyAction(`del-custom-role:${roleName}`);
    setError(null);
    setNotice(null);
    try {
      await p2bApi.deleteCustomRole(roleName);
      setNotice(t("supplier_portal.governance.custom_role_deleted", { roleName }));
      await loadAll(true);
    } catch (err) {
      setError(extractP2bError(err, t("supplier_portal.governance.load_failed")).message);
    } finally {
      setBusyAction(null);
    }
  };

  const handleCreateScimUser = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canMutate || !scimUserName.trim()) return;
    const parsedRoles = scimRoleInput
      .split(",")
      .map((s) => s.trim())
      .filter(Boolean)
      .map((value) => ({ value }));
    setBusyAction("create-scim-user");
    setError(null);
    setNotice(null);
    try {
      const created = await p2bApi.createScimUser({
        schemas: ["urn:ietf:params:scim:schemas:core:2.0:User"],
        userName: scimUserName.trim(),
        displayName: scimDisplayName.trim() || scimUserName.trim(),
        emails: scimEmail.trim() ? [{ value: scimEmail.trim(), primary: true, type: "work" }] : undefined,
        active: scimUserActive,
        roles: parsedRoles.length > 0 ? parsedRoles : undefined,
      });
      setScimUserName("");
      setScimDisplayName("");
      setScimEmail("");
      setScimRoleInput("");
      setScimUserActive(true);
      setNotice(
        t("supplier_portal.governance.scim_user_created", {
          userName: created.userName,
          id: created.id,
        }),
      );
      await loadAll(true);
    } catch (err) {
      setError(extractP2bError(err, t("supplier_portal.governance.load_failed")).message);
    } finally {
      setBusyAction(null);
    }
  };

  const handleToggleScimUserActive = async (user: ScimUser) => {
    if (!canMutate) return;
    setBusyAction(`patch-scim-user:${user.id}`);
    setError(null);
    setNotice(null);
    try {
      const updated = await p2bApi.patchScimUser(user.id, {
        schemas: ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
        Operations: [{ op: "replace", path: "active", value: !user.active }],
      });
      setNotice(
        t("supplier_portal.governance.scim_user_patched", {
          userName: updated.userName || user.userName,
          state: updated.active ? "ACTIVE" : "INACTIVE",
        }),
      );
      await loadAll(true);
    } catch (err) {
      setError(extractP2bError(err, t("supplier_portal.governance.load_failed")).message);
    } finally {
      setBusyAction(null);
    }
  };

  const handleDeleteScimUser = async (user: ScimUser) => {
    if (!canMutate) return;
    setBusyAction(`del-scim-user:${user.id}`);
    setError(null);
    setNotice(null);
    try {
      await p2bApi.deleteScimUser(user.id);
      setNotice(t("supplier_portal.governance.scim_user_deleted", { id: user.id }));
      await loadAll(true);
    } catch (err) {
      setError(extractP2bError(err, t("supplier_portal.governance.load_failed")).message);
    } finally {
      setBusyAction(null);
    }
  };

  const handleCreateScimGroup = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canMutate || !scimGroupName.trim()) return;
    const members = scimGroupMembersInput
      .split(",")
      .map((s) => s.trim())
      .filter(Boolean)
      .map((value) => ({ value }));
    setBusyAction("create-scim-group");
    setError(null);
    setNotice(null);
    try {
      const created = await p2bApi.createScimGroup({
        schemas: ["urn:ietf:params:scim:schemas:core:2.0:Group"],
        displayName: scimGroupName.trim(),
        externalId: scimGroupExternalId.trim() || undefined,
        members: members.length > 0 ? members : undefined,
      });
      setScimGroupName("");
      setScimGroupExternalId("");
      setScimGroupMembersInput("");
      setNotice(
        t("supplier_portal.governance.scim_group_created", {
          displayName: created.displayName,
          id: created.id,
        }),
      );
      await loadAll(true);
    } catch (err) {
      setError(extractP2bError(err, t("supplier_portal.governance.load_failed")).message);
    } finally {
      setBusyAction(null);
    }
  };

  const handleDeleteScimGroup = async (group: ScimGroup) => {
    if (!canMutate) return;
    setBusyAction(`del-scim-group:${group.id}`);
    setError(null);
    setNotice(null);
    try {
      await p2bApi.deleteScimGroup(group.id);
      setNotice(t("supplier_portal.governance.scim_group_deleted", { id: group.id }));
      await loadAll(true);
    } catch (err) {
      setError(extractP2bError(err, t("supplier_portal.governance.load_failed")).message);
    } finally {
      setBusyAction(null);
    }
  };

  const activeHoldCount = useMemo(
    () => holds.filter((h) => h.status === "ACTIVE").length,
    [holds],
  );

  const anchorMatchLabel = useMemo(() => {
    if (!verifyResult || verifyResult.anchor_matched === undefined) {
      return t("supplier_portal.governance.anchor_matched_none");
    }
    return verifyResult.anchor_matched
      ? t("supplier_portal.governance.anchor_matched_yes")
      : t("supplier_portal.governance.anchor_matched_no");
  }, [t, verifyResult]);

  return (
    <PageChrome
      icon="kyc"
      title={t("supplier_portal.governance.title")}
      description={t("supplier_portal.governance.subtitle")}
      loading={loading && !verifyResult && roles.length === 0 && !error}
      skeletonVariant="dashboard"
      actions={
        <button
          type="button"
          disabled={loading || refreshing}
          onClick={() => void loadAll(true)}
          className="portal-btn portal-btn--outline"
        >
          <Icon name="refresh" size={16} className={refreshing ? "animate-spin" : ""} />
          <span>{t("common.action.refresh")}</span>
        </button>
      }
    >
      {roleResolved && isReadOnlyVisitor ? (
        <div
          data-testid="governance-read-only-banner"
          className="portal-alert portal-alert--info mb-6"
          role="status"
        >
          <p className="portal-alert-text text-xs">
            {t("supplier_portal.auditor.safe_mode_badge")} —{" "}
            {t("supplier_portal.governance.read_only_notice")}
          </p>
        </div>
      ) : null}

      {error ? (
        <div className="portal-alert portal-alert--error mb-6" role="alert">
          <p className="portal-alert-text">{error}</p>
        </div>
      ) : null}

      {notice ? (
        <div className="portal-alert portal-alert--info mb-6" role="status">
          <p className="portal-alert-text">{notice}</p>
        </div>
      ) : null}

      <KpiStatGrid columns={4}>
        <KpiStatCard
          label={t("supplier_portal.governance.chain_status")}
          value={
            verifyResult
              ? verifyResult.valid
                ? t("supplier_portal.governance.chain_valid_badge")
                : t("supplier_portal.governance.chain_broken_badge")
              : "—"
          }
          sub={verifyResult ? `${verifyResult.day_bucket} · ${anchorMatchLabel}` : undefined}
        />
        <KpiStatCard
          label={t("supplier_portal.governance.events_count")}
          value={verifyResult ? verifyResult.event_count : events.length}
          sub={verifyResult ? shortHash(verifyResult.root_hash) : undefined}
        />
        <KpiStatCard
          label={t("supplier_portal.governance.holds_title")}
          value={activeHoldCount}
          sub={`${holds.length} total`}
        />
        <KpiStatCard
          label={t("supplier_portal.governance.sod_violations_title")}
          value={sodViolations.length}
          sub={`${sodPolicies.filter((p) => p.enabled).length} / ${sodPolicies.length} policies`}
        />
      </KpiStatGrid>

      <div className="mt-6 mb-6 flex flex-wrap gap-2">
        {(
          [
            { id: "verify", labelKey: "supplier_portal.governance.tab_verify" },
            { id: "holds", labelKey: "supplier_portal.governance.tab_holds" },
            { id: "roles", labelKey: "supplier_portal.governance.tab_roles" },
            { id: "sod", labelKey: "supplier_portal.governance.tab_sod" },
            { id: "custom_scim", labelKey: "supplier_portal.governance.tab_custom_scim" },
          ] as const
        ).map((tab) => {
          const active = activeTab === tab.id;
          return (
            <button
              key={tab.id}
              type="button"
              onClick={() => setActiveTab(tab.id)}
              className="rounded-xl border px-4 py-2 text-xs font-semibold transition"
              style={{
                borderColor: active ? "var(--desk-accent)" : "var(--desk-border)",
                background: active ? "var(--desk-accent)" : "var(--desk-surface)",
                color: active ? "var(--desk-accent-on)" : "var(--desk-text-primary)",
              }}
            >
              {t(tab.labelKey)}
            </button>
          );
        })}
      </div>

      {activeTab === "verify" ? (
        <div className="space-y-6">
          <div className="desk-card p-4">
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4 items-end">
              <div>
                <label htmlFor="gov-day-bucket" className="portal-label block mb-2">
                  {t("supplier_portal.governance.day_bucket_label")}
                </label>
                <PortalInput
                  id="gov-day-bucket"
                  type="date"
                  value={dayBucket}
                  onChange={(e) => setDayBucket(e.target.value)}
                />
              </div>
              <div>
                <label htmlFor="gov-actor-type" className="portal-label block mb-2">
                  {t("supplier_portal.governance.actor_type_filter")}
                </label>
                <PortalSelect
                  id="gov-actor-type"
                  value={actorTypeFilter}
                  onChange={(e) => setActorTypeFilter(e.target.value as AuditActorType | "")}
                >
                  <option value="">{t("supplier_portal.governance.all_actors")}</option>
                  <option value="human">{t("supplier_portal.audit.actor_human")}</option>
                  <option value="agent">{t("supplier_portal.audit.actor_agent")}</option>
                  <option value="connector">{t("supplier_portal.audit.actor_connector")}</option>
                  <option value="system">{t("supplier_portal.audit.actor_system")}</option>
                </PortalSelect>
              </div>
              <div>
                <label htmlFor="gov-object-filter" className="portal-label block mb-2">
                  {t("supplier_portal.governance.object_filter")}
                </label>
                <PortalInput
                  id="gov-object-filter"
                  type="text"
                  placeholder="order, journal, recon..."
                  value={objectFilter}
                  onChange={(e) => setObjectFilter(e.target.value)}
                />
              </div>
              <div className="flex flex-wrap gap-2">
                <button
                  type="button"
                  disabled={busyAction === "verify"}
                  onClick={() => void handleVerifyChain()}
                  className="portal-btn portal-btn--primary"
                >
                  <Icon name="verified" size={16} />
                  <span>{t("supplier_portal.governance.verify_button")}</span>
                </button>
                {canMutate ? (
                  <button
                    type="button"
                    disabled={busyAction === "anchor"}
                    onClick={() => void handleWriteAnchor()}
                    className="portal-btn portal-btn--outline"
                  >
                    <span>{t("supplier_portal.governance.write_anchor_button")}</span>
                  </button>
                ) : null}
              </div>
            </div>

            {verifyResult ? (
              <div
                className="mt-4 grid grid-cols-1 md:grid-cols-3 gap-3 rounded-xl border p-3 text-xs"
                style={{
                  borderColor: "var(--desk-border)",
                  background: "var(--desk-surface-subtle)",
                }}
              >
                <div>
                  <div style={{ color: "var(--desk-text-tertiary)" }}>
                    {t("supplier_portal.governance.root_hash")}
                  </div>
                  <div className="font-mono font-semibold truncate mt-1" title={verifyResult.root_hash}>
                    {verifyResult.root_hash || "—"}
                  </div>
                </div>
                <div>
                  <div style={{ color: "var(--desk-text-tertiary)" }}>
                    {t("supplier_portal.governance.anchored_root_hash")}
                  </div>
                  <div
                    className="font-mono font-semibold truncate mt-1"
                    title={verifyResult.anchored_root_hash || ""}
                  >
                    {verifyResult.anchored_root_hash || "—"}
                  </div>
                </div>
                <div>
                  <div style={{ color: "var(--desk-text-tertiary)" }}>
                    {t("supplier_portal.governance.anchor_match")}
                  </div>
                  <div className="font-mono font-semibold mt-1">
                    {anchorMatchLabel}
                    {verifyResult.broken_event_id
                      ? ` · broken: ${verifyResult.broken_event_id}`
                      : ""}
                    {verifyResult.failure_reason ? ` (${verifyResult.failure_reason})` : ""}
                  </div>
                </div>
              </div>
            ) : null}
          </div>

          <PortalSection
            icon="verified"
            title={t("supplier_portal.governance.anchors_title")}
          >
            {anchors.length === 0 ? (
              <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                {t("supplier_portal.governance.anchor_matched_none")}
              </p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse text-xs">
                  <thead>
                    <tr
                      className="border-b uppercase"
                      style={{ borderColor: "var(--desk-border)", color: "var(--desk-text-tertiary)" }}
                    >
                      <th className="py-2 px-3">{t("supplier_portal.governance.day_bucket_label")}</th>
                      <th className="py-2 px-3 text-right">{t("supplier_portal.governance.events_count")}</th>
                      <th className="py-2 px-3">{t("supplier_portal.governance.root_hash")}</th>
                      <th className="py-2 px-3">WORM URI</th>
                      <th className="py-2 px-3">Signer</th>
                      <th className="py-2 px-3">Anchored</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                    {anchors.map((a) => (
                      <tr key={`${a.tenant_id}:${a.day_bucket}`}>
                        <td className="py-2 px-3 font-mono font-semibold">{a.day_bucket}</td>
                        <td className="py-2 px-3 font-mono text-right">{a.event_count}</td>
                        <td className="py-2 px-3 font-mono" title={a.root_hash}>
                          {shortHash(a.root_hash)}
                        </td>
                        <td className="py-2 px-3 font-mono truncate max-w-[220px]" title={a.worm_uri || ""}>
                          {a.worm_uri || "—"}
                        </td>
                        <td className="py-2 px-3 font-mono">{a.signer_key_id || a.anchored_by}</td>
                        <td className="py-2 px-3 whitespace-nowrap">
                          {new Date(a.anchored_at).toLocaleString()}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </PortalSection>

          <PortalSection
            icon="ledger"
            title={t("supplier_portal.governance.events_title")}
            description={
              eventsMasked
                ? t("supplier_portal.audit.pii_masked_note")
                : t("supplier_portal.audit.pii_unmasked_note")
            }
          >
            {events.length === 0 ? (
              <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                {t("supplier_portal.audit.empty")}
              </p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse text-xs">
                  <thead>
                    <tr
                      className="border-b uppercase"
                      style={{ borderColor: "var(--desk-border)", color: "var(--desk-text-tertiary)" }}
                    >
                      <th className="py-2 px-3">Seq</th>
                      <th className="py-2 px-3">Actor</th>
                      <th className="py-2 px-3">Action</th>
                      <th className="py-2 px-3">Object</th>
                      <th className="py-2 px-3">{t("supplier_portal.audit.trace_id")}</th>
                      <th className="py-2 px-3">Hash</th>
                      <th className="py-2 px-3">Time</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                    {events.map((ev) => (
                      <tr key={ev.event_id}>
                        <td className="py-2 px-3 font-mono">#{ev.seq_num}</td>
                        <td className="py-2 px-3">
                          <div className="flex flex-wrap items-center gap-1.5">
                            <ActorTypeBadge actorType={ev.actor_type} />
                            <span className="font-mono">{ev.actor_id}</span>
                          </div>
                        </td>
                        <td className="py-2 px-3 font-mono font-semibold">
                          {ev.action}
                          {ev.legal_hold ? (
                            <span className="ml-2 rounded border px-1.5 py-0.5 text-[10px]">
                              {t("supplier_portal.audit.legal_hold_badge")}
                            </span>
                          ) : null}
                        </td>
                        <td className="py-2 px-3 font-mono">
                          {ev.object_type ? `${ev.object_type}:${ev.object_id || "—"}` : "—"}
                        </td>
                        <td className="py-2 px-3 font-mono">{ev.trace_id || "—"}</td>
                        <td className="py-2 px-3 font-mono" title={ev.entry_hash}>
                          {shortHash(ev.entry_hash)}
                        </td>
                        <td className="py-2 px-3 whitespace-nowrap">
                          {new Date(ev.created_at).toLocaleString()}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </PortalSection>
        </div>
      ) : null}

      {activeTab === "holds" ? (
        <div className="space-y-6">
          {canMutate ? (
            <PortalSection
              icon="warning"
              title={t("supplier_portal.governance.place_hold_title")}
            >
              <form onSubmit={(e) => void handleCreateHold(e)} className="grid grid-cols-1 md:grid-cols-5 gap-4 items-end">
                <div>
                  <label htmlFor="hold-object-type" className="portal-label block mb-2">
                    {t("supplier_portal.governance.object_type_label")}
                  </label>
                  <PortalInput
                    id="hold-object-type"
                    value={holdObjectType}
                    onChange={(e) => setHoldObjectType(e.target.value)}
                    required
                  />
                </div>
                <div>
                  <label htmlFor="hold-object-id" className="portal-label block mb-2">
                    {t("supplier_portal.governance.object_id_label")}
                  </label>
                  <PortalInput
                    id="hold-object-id"
                    value={holdObjectId}
                    onChange={(e) => setHoldObjectId(e.target.value)}
                    required
                  />
                </div>
                <div>
                  <label htmlFor="hold-reason" className="portal-label block mb-2">
                    {t("supplier_portal.governance.hold_reason_label")}
                  </label>
                  <PortalInput
                    id="hold-reason"
                    value={holdReason}
                    onChange={(e) => setHoldReason(e.target.value)}
                    required
                  />
                </div>
                <div>
                  <label htmlFor="hold-expires-at" className="portal-label block mb-2">
                    {t("supplier_portal.governance.expires_at_label")}
                  </label>
                  <PortalInput
                    id="hold-expires-at"
                    type="datetime-local"
                    value={holdExpiresAt}
                    onChange={(e) => setHoldExpiresAt(e.target.value)}
                  />
                </div>
                <div>
                  <button
                    type="submit"
                    disabled={busyAction === "hold" || !holdObjectId.trim() || !holdReason.trim()}
                    className="portal-btn portal-btn--primary w-full"
                  >
                    {t("supplier_portal.governance.place_hold_button")}
                  </button>
                </div>
              </form>
            </PortalSection>
          ) : null}

          <PortalSection
            icon="kyc"
            title={t("supplier_portal.governance.holds_title")}
            actions={
              <PortalSelect
                id="hold-status-filter"
                aria-label={t("supplier_portal.finance.four_way_recon.filter_status")}
                value={holdStatusFilter}
                onChange={(e) => setHoldStatusFilter(e.target.value)}
              >
                <option value="ALL">{t("supplier_portal.finance.four_way_recon.all_filter")}</option>
                <option value="ACTIVE">ACTIVE</option>
                <option value="RELEASED">RELEASED</option>
              </PortalSelect>
            }
          >
            {holds.length === 0 ? (
              <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                {t("supplier_portal.governance.holds_empty")}
              </p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse text-xs">
                  <thead>
                    <tr
                      className="border-b uppercase"
                      style={{ borderColor: "var(--desk-border)", color: "var(--desk-text-tertiary)" }}
                    >
                      <th className="py-2 px-3">Hold ID</th>
                      <th className="py-2 px-3">{t("supplier_portal.governance.object_type_label")}</th>
                      <th className="py-2 px-3">{t("supplier_portal.governance.hold_reason_label")}</th>
                      <th className="py-2 px-3">{t("supplier_portal.compliance.text.status")}</th>
                      <th className="py-2 px-3">Created by</th>
                      <th className="py-2 px-3">Expires</th>
                      <th className="py-2 px-3">Created</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                    {holds.map((h) => (
                      <tr key={h.hold_id}>
                        <td className="py-2 px-3 font-mono font-semibold">{h.hold_id}</td>
                        <td className="py-2 px-3 font-mono">
                          {h.object_type}:{h.object_id}
                        </td>
                        <td className="py-2 px-3">{h.reason}</td>
                        <td className="py-2 px-3">
                          <StatusBadge state={h.status} />
                        </td>
                        <td className="py-2 px-3 font-mono">{h.created_by}</td>
                        <td className="py-2 px-3 whitespace-nowrap">
                          {h.expires_at ? new Date(h.expires_at).toLocaleString() : "—"}
                        </td>
                        <td className="py-2 px-3 whitespace-nowrap">
                          {new Date(h.created_at).toLocaleString()}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </PortalSection>
        </div>
      ) : null}

      {activeTab === "roles" ? (
        <div className="space-y-6">
          <PortalSection
            icon="person-add"
            title={t("supplier_portal.governance.assign_role_title")}
          >
            {canMutate ? (
              <form onSubmit={(e) => void handleAssignRole(e)} className="grid grid-cols-1 md:grid-cols-4 gap-4 items-end">
                <div>
                  <label htmlFor="iam-user-id" className="portal-label block mb-2">
                    {t("supplier_portal.governance.user_id_label")}
                  </label>
                  <PortalInput
                    id="iam-user-id"
                    value={assignUserId}
                    onChange={(e) => setAssignUserId(e.target.value)}
                    required
                  />
                </div>
                <div>
                  <label htmlFor="iam-role-select" className="portal-label block mb-2">
                    {t("supplier_portal.governance.role_name_label")}
                  </label>
                  <PortalSelect
                    id="iam-role-select"
                    value={assignRoleName}
                    onChange={(e) => setAssignRoleName(e.target.value)}
                  >
                    {roles.map((r) => (
                      <option key={r.role_name} value={r.role_name}>
                        {r.role_name}
                      </option>
                    ))}
                  </PortalSelect>
                </div>
                <div>
                  <label htmlFor="iam-role-action" className="portal-label block mb-2">
                    {t("supplier_portal.governance.role_action_label")}
                  </label>
                  <PortalSelect
                    id="iam-role-action"
                    value={assignAction}
                    onChange={(e) => setAssignAction(e.target.value as RoleAssignmentAction)}
                  >
                    <option value="ASSIGN">ASSIGN</option>
                    <option value="REPLACE">REPLACE</option>
                    <option value="REVOKE">REVOKE</option>
                  </PortalSelect>
                </div>
                <div>
                  <button
                    type="submit"
                    disabled={busyAction === "assign-role" || !assignUserId.trim()}
                    className="portal-btn portal-btn--primary w-full"
                  >
                    {t("supplier_portal.governance.submit_role_button")}
                  </button>
                </div>
              </form>
            ) : null}

            {userRoles.length > 0 ? (
              <div className="mt-5 overflow-x-auto">
                <h3 className="text-xs font-semibold uppercase mb-2" style={{ color: "var(--desk-text-tertiary)" }}>
                  {t("supplier_portal.governance.user_roles_title")}
                </h3>
                <table className="w-full text-left border-collapse text-xs">
                  <thead>
                    <tr
                      className="border-b uppercase"
                      style={{ borderColor: "var(--desk-border)", color: "var(--desk-text-tertiary)" }}
                    >
                      <th className="py-2 px-3">{t("supplier_portal.governance.user_id_label")}</th>
                      <th className="py-2 px-3">{t("supplier_portal.governance.role_name_label")}</th>
                      <th className="py-2 px-3">Assigned by</th>
                      <th className="py-2 px-3">Updated</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                    {userRoles.map((ur) => (
                      <tr key={`${ur.tenant_id}:${ur.user_id}:${ur.role_name}`}>
                        <td className="py-2 px-3 font-mono font-semibold">{ur.user_id}</td>
                        <td className="py-2 px-3 font-mono">{ur.role_name}</td>
                        <td className="py-2 px-3 font-mono">{ur.assigned_by || "—"}</td>
                        <td className="py-2 px-3 whitespace-nowrap">
                          {new Date(ur.updated_at).toLocaleString()}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : null}
          </PortalSection>

          <PortalSection
            icon="kyc"
            title={t("supplier_portal.governance.roles_title")}
          >
            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse text-xs">
                <thead>
                  <tr
                    className="border-b uppercase"
                    style={{ borderColor: "var(--desk-border)", color: "var(--desk-text-tertiary)" }}
                  >
                    <th className="py-2 px-3">Permission</th>
                    {roles.map((r) => (
                      <th key={r.role_name} className="py-2 px-2 text-center font-mono whitespace-nowrap">
                        <div>{r.role_name}</div>
                        {r.read_only ? (
                          <span className="inline-block rounded border px-1 py-0.5 text-[9px] mt-0.5">
                            {t("supplier_portal.governance.read_only_badge")}
                          </span>
                        ) : null}
                        {r.agent_role ? (
                          <span className="inline-block rounded border px-1 py-0.5 text-[9px] mt-0.5">
                            {t("supplier_portal.governance.agent_badge")}
                          </span>
                        ) : null}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                  {permissions.map((perm) => (
                    <tr key={perm.key}>
                      <td className="py-2 px-3">
                        <div className="font-mono font-semibold">{perm.key}</div>
                        <div style={{ color: "var(--desk-text-tertiary)" }}>
                          {perm.category} · {perm.description}
                        </div>
                      </td>
                      {roles.map((r) => {
                        const allowed = r.permissions.includes(perm.key);
                        return (
                          <td key={`${r.role_name}:${perm.key}`} className="py-2 px-2 text-center font-mono">
                            {allowed ? "✓" : "—"}
                          </td>
                        );
                      })}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </PortalSection>
        </div>
      ) : null}

      {activeTab === "sod" ? (
        <div className="space-y-6">
          <PortalSection
            icon="warning"
            title={t("supplier_portal.governance.sod_policies_title")}
          >
            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse text-xs">
                <thead>
                  <tr
                    className="border-b uppercase"
                    style={{ borderColor: "var(--desk-border)", color: "var(--desk-text-tertiary)" }}
                  >
                    <th className="py-2 px-3">Policy</th>
                    <th className="py-2 px-3">Conflicting pair</th>
                    <th className="py-2 px-3">Scope</th>
                    <th className="py-2 px-3">{t("supplier_portal.compliance.text.status")}</th>
                    <th className="py-2 px-3 text-right">{t("supplier_portal.finance.gl_ledger.actions")}</th>
                  </tr>
                </thead>
                <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                  {sodPolicies.map((pol) => (
                    <tr key={pol.policy_id}>
                      <td className="py-2 px-3">
                        <div className="font-semibold">{pol.name}</div>
                        <div className="font-mono" style={{ color: "var(--desk-text-tertiary)" }}>
                          {pol.policy_id}
                        </div>
                      </td>
                      <td className="py-2 px-3 font-mono">
                        {pol.permission_a} ↔ {pol.permission_b}
                      </td>
                      <td className="py-2 px-3">
                        {pol.enforce_same_object
                          ? t("supplier_portal.governance.same_object_only")
                          : t("supplier_portal.governance.global_scope")}
                      </td>
                      <td className="py-2 px-3">
                        <StatusBadge state={pol.enabled ? "ENABLED" : "DISABLED"} />
                      </td>
                      <td className="py-2 px-3 text-right">
                        {canMutate ? (
                          <button
                            type="button"
                            disabled={busyAction === `policy:${pol.policy_id}`}
                            onClick={() => void handleTogglePolicy(pol)}
                            className="rounded-lg border px-2.5 py-1 text-xs font-medium hover:bg-[var(--desk-surface-subtle)]"
                            style={{ borderColor: "var(--desk-border)" }}
                          >
                            {pol.enabled
                              ? t("supplier_portal.governance.toggle_disable")
                              : t("supplier_portal.governance.toggle_enable")}
                          </button>
                        ) : (
                          <span>—</span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </PortalSection>

          <PortalSection
            icon="error"
            title={t("supplier_portal.governance.sod_violations_title")}
            actions={
              <PortalInput
                id="sod-actor-filter"
                aria-label={t("supplier_portal.governance.actor_type_filter")}
                placeholder="Filter by actor..."
                value={violationActorFilter}
                onChange={(e) => setViolationActorFilter(e.target.value)}
                className="max-w-xs"
              />
            }
          >
            {sodViolations.length === 0 ? (
              <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                {t("supplier_portal.governance.sod_violations_empty")}
              </p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse text-xs">
                  <thead>
                    <tr
                      className="border-b uppercase"
                      style={{ borderColor: "var(--desk-border)", color: "var(--desk-text-tertiary)" }}
                    >
                      <th className="py-2 px-3">Rule</th>
                      <th className="py-2 px-3">Actor / Maker</th>
                      <th className="py-2 px-3">Action</th>
                      <th className="py-2 px-3">Object</th>
                      <th className="py-2 px-3">{t("supplier_portal.audit.trace_id")}</th>
                      <th className="py-2 px-3">Time</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                    {sodViolations.map((v) => (
                      <tr key={v.violation_id}>
                        <td className="py-2 px-3 font-mono font-semibold">{v.rule_code}</td>
                        <td className="py-2 px-3 font-mono">
                          {v.actor_id}
                          {v.maker_id ? ` (maker: ${v.maker_id})` : ""}
                        </td>
                        <td className="py-2 px-3 font-mono">{v.action}</td>
                        <td className="py-2 px-3 font-mono">
                          {v.object_type ? `${v.object_type}:${v.object_id || "—"}` : "—"}
                        </td>
                        <td className="py-2 px-3 font-mono">{v.trace_id || "—"}</td>
                        <td className="py-2 px-3 whitespace-nowrap">
                          {new Date(v.created_at).toLocaleString()}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </PortalSection>
        </div>
      ) : null}

      {activeTab === "custom_scim" ? (
        <div className="space-y-6" data-testid="governance-custom-scim-panel">
          <PortalSection
            icon="kyc"
            title={t("supplier_portal.governance.custom_roles_title")}
            description={t("supplier_portal.governance.custom_roles_subtitle")}
          >
            {canMutate ? (
              <form onSubmit={(e) => void handleCreateCustomRole(e)} className="space-y-4 mb-6">
                <div className="grid grid-cols-1 md:grid-cols-4 gap-3 items-end">
                  <div>
                    <label htmlFor="custom-role-name" className="portal-label block mb-1">
                      {t("supplier_portal.governance.custom_role_name_label")}
                    </label>
                    <PortalInput
                      id="custom-role-name"
                      value={customRoleName}
                      onChange={(e) => setCustomRoleName(e.target.value)}
                      placeholder="REGIONAL_CONTROLLER"
                      required
                    />
                  </div>
                  <div className="md:col-span-2">
                    <label htmlFor="custom-role-desc" className="portal-label block mb-1">
                      {t("supplier_portal.governance.custom_role_desc_label")}
                    </label>
                    <PortalInput
                      id="custom-role-desc"
                      value={customRoleDesc}
                      onChange={(e) => setCustomRoleDesc(e.target.value)}
                      placeholder={t("supplier_portal.governance.custom_role_desc_placeholder")}
                    />
                  </div>
                  <label className="inline-flex items-center gap-2 text-xs pb-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={customRoleReadOnly}
                      onChange={(e) => setCustomRoleReadOnly(e.target.checked)}
                    />
                    <span>{t("supplier_portal.governance.read_only_badge")}</span>
                  </label>
                </div>

                {permissions.length > 0 ? (
                  <div>
                    <div className="portal-label mb-1.5">
                      {t("supplier_portal.governance.custom_role_perms_label")}
                    </div>
                    <div className="flex flex-wrap gap-2">
                      {permissions.map((perm) => {
                        const checked = customRolePerms.includes(perm.key);
                        return (
                          <label
                            key={perm.key}
                            className="inline-flex items-center gap-1.5 rounded-lg border px-2.5 py-1 text-xs font-mono cursor-pointer"
                            style={{ borderColor: "var(--desk-border)" }}
                          >
                            <input
                              type="checkbox"
                              checked={checked}
                              onChange={() => toggleCustomPermission(perm.key)}
                            />
                            <span>{perm.key}</span>
                          </label>
                        );
                      })}
                    </div>
                  </div>
                ) : null}

                <div className="grid grid-cols-1 md:grid-cols-4 gap-3 items-end">
                  <div className="md:col-span-3">
                    <label htmlFor="custom-role-manual-perms" className="portal-label block mb-1">
                      {t("supplier_portal.governance.custom_role_manual_perms_label")}
                    </label>
                    <PortalInput
                      id="custom-role-manual-perms"
                      value={customRoleManualPerms}
                      onChange={(e) => setCustomRoleManualPerms(e.target.value)}
                      placeholder="audit.read, recon.manage"
                    />
                  </div>
                  <div>
                    <button
                      type="submit"
                      disabled={busyAction !== null || !customRoleName.trim()}
                      className="portal-btn portal-btn--primary w-full text-xs"
                    >
                      {t("supplier_portal.governance.create_custom_role_button")}
                    </button>
                  </div>
                </div>
              </form>
            ) : null}

            {customRoles.length === 0 ? (
              <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                {t("supplier_portal.governance.custom_roles_empty")}
              </p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse text-xs">
                  <thead>
                    <tr
                      className="border-b uppercase"
                      style={{ borderColor: "var(--desk-border)", color: "var(--desk-text-tertiary)" }}
                    >
                      <th className="py-2 px-3">{t("supplier_portal.governance.role_name_label")}</th>
                      <th className="py-2 px-3">{t("supplier_portal.governance.custom_role_desc_label")}</th>
                      <th className="py-2 px-3">{t("supplier_portal.governance.custom_role_perms_label")}</th>
                      <th className="py-2 px-3">{t("supplier_portal.compliance.text.status")}</th>
                      <th className="py-2 px-3 text-right">{t("supplier_portal.finance.gl_ledger.actions")}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                    {customRoles.map((cr) => (
                      <tr key={cr.role_name}>
                        <td className="py-2 px-3 font-mono font-semibold">
                          <div>{cr.role_name}</div>
                          {cr.read_only ? (
                            <span className="inline-block rounded border px-1 py-0.5 text-[9px] mt-0.5">
                              {t("supplier_portal.governance.read_only_badge")}
                            </span>
                          ) : null}
                        </td>
                        <td className="py-2 px-3">{cr.description || "—"}</td>
                        <td className="py-2 px-3 font-mono">{(cr.permissions ?? []).join(", ")}</td>
                        <td className="py-2 px-3">
                          <StatusBadge state={cr.status} />
                        </td>
                        <td className="py-2 px-3 text-right">
                          {canMutate ? (
                            <button
                              type="button"
                              disabled={busyAction !== null}
                              onClick={() => void handleDeleteCustomRole(cr.role_name)}
                              className="rounded-lg border px-2.5 py-1 text-xs font-medium"
                              style={{ borderColor: "var(--desk-border)" }}
                            >
                              {t("supplier_portal.governance.delete_custom_role_button")}
                            </button>
                          ) : (
                            <span>—</span>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </PortalSection>

          <PortalSection
            icon="crm"
            title={t("supplier_portal.governance.scim_users_title")}
            description={t("supplier_portal.governance.scim_subtitle")}
          >
            {canMutate ? (
              <form onSubmit={(e) => void handleCreateScimUser(e)} className="grid grid-cols-1 md:grid-cols-6 gap-3 items-end mb-5">
                <div>
                  <label htmlFor="scim-username" className="portal-label block mb-1">
                    {t("supplier_portal.governance.scim_username_label")}
                  </label>
                  <PortalInput
                    id="scim-username"
                    value={scimUserName}
                    onChange={(e) => setScimUserName(e.target.value)}
                    placeholder="alice@acme.uz"
                    required
                  />
                </div>
                <div>
                  <label htmlFor="scim-displayname" className="portal-label block mb-1">
                    {t("supplier_portal.governance.scim_display_name_label")}
                  </label>
                  <PortalInput
                    id="scim-displayname"
                    value={scimDisplayName}
                    onChange={(e) => setScimDisplayName(e.target.value)}
                    placeholder="Alice Karimova"
                  />
                </div>
                <div>
                  <label htmlFor="scim-email" className="portal-label block mb-1">
                    {t("supplier_portal.governance.scim_email_label")}
                  </label>
                  <PortalInput
                    id="scim-email"
                    type="email"
                    value={scimEmail}
                    onChange={(e) => setScimEmail(e.target.value)}
                    placeholder="alice@acme.uz"
                  />
                </div>
                <div>
                  <label htmlFor="scim-roles" className="portal-label block mb-1">
                    {t("supplier_portal.governance.scim_roles_label")}
                  </label>
                  <PortalInput
                    id="scim-roles"
                    value={scimRoleInput}
                    onChange={(e) => setScimRoleInput(e.target.value)}
                    placeholder="FINANCE_MANAGER"
                  />
                </div>
                <label className="inline-flex items-center gap-2 text-xs pb-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={scimUserActive}
                    onChange={(e) => setScimUserActive(e.target.checked)}
                  />
                  <span>{t("supplier_portal.governance.scim_active_label")}</span>
                </label>
                <div>
                  <button
                    type="submit"
                    disabled={busyAction !== null || !scimUserName.trim()}
                    className="portal-btn portal-btn--primary w-full text-xs"
                  >
                    {t("supplier_portal.governance.scim_create_user_button")}
                  </button>
                </div>
              </form>
            ) : null}

            {scimUsers.length === 0 ? (
              <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                {t("supplier_portal.governance.scim_users_empty")}
              </p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse text-xs">
                  <thead>
                    <tr
                      className="border-b uppercase"
                      style={{ borderColor: "var(--desk-border)", color: "var(--desk-text-tertiary)" }}
                    >
                      <th className="py-2 px-3">{t("supplier_portal.governance.scim_username_label")}</th>
                      <th className="py-2 px-3">{t("supplier_portal.governance.scim_display_name_label")}</th>
                      <th className="py-2 px-3">{t("supplier_portal.governance.scim_email_label")}</th>
                      <th className="py-2 px-3">{t("supplier_portal.governance.scim_roles_label")}</th>
                      <th className="py-2 px-3">{t("supplier_portal.compliance.text.status")}</th>
                      <th className="py-2 px-3 text-right">{t("supplier_portal.finance.gl_ledger.actions")}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                    {scimUsers.map((u) => (
                      <tr key={u.id}>
                        <td className="py-2 px-3 font-mono font-semibold">
                          <div>{u.userName}</div>
                          <div style={{ color: "var(--desk-text-tertiary)" }}>{u.id}</div>
                        </td>
                        <td className="py-2 px-3">{u.displayName || "—"}</td>
                        <td className="py-2 px-3 font-mono">{u.emails?.[0]?.value || "—"}</td>
                        <td className="py-2 px-3 font-mono">
                          {(u.roles ?? []).map((r) => r.value).join(", ") || "—"}
                        </td>
                        <td className="py-2 px-3">
                          <StatusBadge state={u.active ? "ACTIVE" : "INACTIVE"} />
                        </td>
                        <td className="py-2 px-3 text-right">
                          {canMutate ? (
                            <div className="inline-flex items-center gap-1.5">
                              <button
                                type="button"
                                disabled={busyAction !== null}
                                onClick={() => void handleToggleScimUserActive(u)}
                                className="rounded-lg border px-2 py-1 text-xs"
                                style={{ borderColor: "var(--desk-border)" }}
                              >
                                {u.active
                                  ? t("supplier_portal.governance.scim_deactivate_button")
                                  : t("supplier_portal.governance.scim_activate_button")}
                              </button>
                              <button
                                type="button"
                                disabled={busyAction !== null}
                                onClick={() => void handleDeleteScimUser(u)}
                                className="rounded-lg border px-2 py-1 text-xs"
                                style={{ borderColor: "var(--desk-border)" }}
                              >
                                {t("supplier_portal.governance.scim_delete_button")}
                              </button>
                            </div>
                          ) : (
                            <span>—</span>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </PortalSection>

          <PortalSection
            icon="topology"
            title={t("supplier_portal.governance.scim_groups_title")}
          >
            {canMutate ? (
              <form onSubmit={(e) => void handleCreateScimGroup(e)} className="grid grid-cols-1 md:grid-cols-4 gap-3 items-end mb-5">
                <div>
                  <label htmlFor="scim-group-name" className="portal-label block mb-1">
                    {t("supplier_portal.governance.scim_group_name_label")}
                  </label>
                  <PortalInput
                    id="scim-group-name"
                    value={scimGroupName}
                    onChange={(e) => setScimGroupName(e.target.value)}
                    placeholder="FinanceApprovers"
                    required
                  />
                </div>
                <div>
                  <label htmlFor="scim-group-ext-id" className="portal-label block mb-1">
                    {t("supplier_portal.governance.scim_external_id_label")}
                  </label>
                  <PortalInput
                    id="scim-group-ext-id"
                    value={scimGroupExternalId}
                    onChange={(e) => setScimGroupExternalId(e.target.value)}
                    placeholder="okta-grp-1"
                  />
                </div>
                <div>
                  <label htmlFor="scim-group-members" className="portal-label block mb-1">
                    {t("supplier_portal.governance.scim_group_members_label")}
                  </label>
                  <PortalInput
                    id="scim-group-members"
                    value={scimGroupMembersInput}
                    onChange={(e) => setScimGroupMembersInput(e.target.value)}
                    placeholder="usr-1, usr-2"
                  />
                </div>
                <div>
                  <button
                    type="submit"
                    disabled={busyAction !== null || !scimGroupName.trim()}
                    className="portal-btn portal-btn--primary w-full text-xs"
                  >
                    {t("supplier_portal.governance.scim_create_group_button")}
                  </button>
                </div>
              </form>
            ) : null}

            {scimGroups.length === 0 ? (
              <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                {t("supplier_portal.governance.scim_groups_empty")}
              </p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse text-xs">
                  <thead>
                    <tr
                      className="border-b uppercase"
                      style={{ borderColor: "var(--desk-border)", color: "var(--desk-text-tertiary)" }}
                    >
                      <th className="py-2 px-3">{t("supplier_portal.governance.scim_group_name_label")}</th>
                      <th className="py-2 px-3">{t("supplier_portal.governance.scim_external_id_label")}</th>
                      <th className="py-2 px-3">{t("supplier_portal.governance.scim_group_members_label")}</th>
                      <th className="py-2 px-3 text-right">{t("supplier_portal.finance.gl_ledger.actions")}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                    {scimGroups.map((g) => (
                      <tr key={g.id}>
                        <td className="py-2 px-3 font-mono font-semibold">
                          <div>{g.displayName}</div>
                          <div style={{ color: "var(--desk-text-tertiary)" }}>{g.id}</div>
                        </td>
                        <td className="py-2 px-3 font-mono">{g.externalId || "—"}</td>
                        <td className="py-2 px-3 font-mono">
                          {(g.members ?? []).map((m) => m.display || m.value).join(", ") || "—"}
                        </td>
                        <td className="py-2 px-3 text-right">
                          {canMutate ? (
                            <button
                              type="button"
                              disabled={busyAction !== null}
                              onClick={() => void handleDeleteScimGroup(g)}
                              className="rounded-lg border px-2 py-1 text-xs"
                              style={{ borderColor: "var(--desk-border)" }}
                            >
                              {t("supplier_portal.governance.scim_delete_button")}
                            </button>
                          ) : (
                            <span>—</span>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </PortalSection>
        </div>
      ) : null}
    </PageChrome>
  );
}
