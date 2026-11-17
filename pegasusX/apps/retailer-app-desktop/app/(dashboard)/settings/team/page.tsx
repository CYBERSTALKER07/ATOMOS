"use client";

import { usePortalLocale, usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import {
  ArrowLeft,
  Loader2,
  UserPlus,
  Users,
  Shield,
  AlertTriangle,
} from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { OwnerApprovalsInbox } from "@/components/OwnerApprovalsInbox";
import { SellerMandateModal } from "@/components/SellerMandateModal";
import { apiFetch } from "@/lib/auth";
import {
  approveRetailerRequest,
  getRetailerApprovals,
  getRetailerMe,
  rejectRetailerRequest,
} from "@/lib/api";
import { W2_COPY, type RetailerApprovalItem } from "@/lib/retailer-w2";

type Member = {
  user_id: string;
  name: string;
  phone: string;
  retailer_role: string;
  is_owner: boolean;
  is_active: boolean;
};

const ROLES = [
  "ADMIN",
  "MANAGER",
  "BUYER",
  "RECEIVER",
  "CASHIER",
  "STOCK_CLERK",
  "SECTION_LEAD",
  "VIEWER",
] as const;

export default function TeamPage() {
  const t = usePortalT();
  const locale = usePortalLocale();
  const w2 = W2_COPY[locale] ?? W2_COPY.en;
  const router = useRouter();
  const [items, setItems] = useState<Member[]>([]);
  const [isOwner, setIsOwner] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [banner, setBanner] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [approvals, setApprovals] = useState<RetailerApprovalItem[]>([]);
  const [approvalStatus, setApprovalStatus] = useState("PENDING_OWNER");
  const [approvalsLoading, setApprovalsLoading] = useState(false);
  const [approvalsError, setApprovalsError] = useState<string | null>(null);
  const [approvalBanner, setApprovalBanner] = useState<string | null>(null);
  const [decidingId, setDecidingId] = useState<string | null>(null);
  const [mandateTarget, setMandateTarget] = useState<Member | null>(null);
  const [form, setForm] = useState({
    name: "",
    phone: "",
    password: "",
    retailer_role: "CASHIER",
  });

  const loadApprovals = useCallback(async (status: string) => {
    setApprovalsLoading(true);
    setApprovalsError(null);
    try {
      const res = await getRetailerApprovals(status, 20);
      setApprovals(res.items ?? []);
    } catch (e) {
      setApprovalsError(
        e instanceof Error ? e.message : "Failed to load approval inbox",
      );
    } finally {
      setApprovalsLoading(false);
    }
  }, []);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [res, me] = await Promise.all([
        apiFetch("/v1/retailer/org/members"),
        getRetailerMe().catch(() => null),
      ]);
      if (!res.ok) throw new Error(`load_failed_${res.status}`);
      const json = (await res.json()) as { items?: Member[] };
      setItems(json.items ?? []);
      const owner = Boolean(
        me?.is_owner || me?.retailer_role?.toUpperCase() === "OWNER",
      );
      setIsOwner(owner);
      if (owner) {
        await loadApprovals(approvalStatus);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : t("retailer_desktop.residual.text.failed_to_load_team"));
    } finally {
      setLoading(false);
    }
  }, [approvalStatus, loadApprovals, t]);

  useEffect(() => {
    void load();
  }, [load]);

  const handleDecision = async (
    requestId: string,
    approve: boolean,
    reason?: string,
  ) => {
    setDecidingId(requestId);
    setApprovalsError(null);
    setApprovalBanner(null);
    try {
      if (approve) {
        await approveRetailerRequest(requestId, reason);
        setApprovalBanner("Order approved and dispatched.");
      } else {
        await rejectRetailerRequest(requestId, reason);
        setApprovalBanner("Approval request rejected.");
      }
      await loadApprovals(approvalStatus);
    } catch (e) {
      setApprovalsError(
        e instanceof Error ? e.message : "Failed to submit approval decision",
      );
    } finally {
      setDecidingId(null);
    }
  };

  const invite = async () => {
    setBusy(true);
    setBanner(null);
    try {
      const res = await apiFetch("/v1/retailer/org/members", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": `team-invite-${Date.now()}`,
        },
        body: JSON.stringify(form),
      });
      const json = await res.json().catch(() => ({}));
      if (!res.ok) {
        throw new Error(
          (json as { error?: string }).error || `invite_failed_${res.status}`,
        );
      }
      setBanner("Team member created");
      setForm({ name: "", phone: "", password: "", retailer_role: "CASHIER" });
      await load();
    } catch (e) {
      setBanner(e instanceof Error ? e.message : t("retailer_desktop.residual.text.invite_failed"));
    } finally {
      setBusy(false);
    }
  };

  const deactivate = async (userId: string) => {
    if (!confirm("Deactivate this team member?")) return;
    setBusy(true);
    try {
      const res = await apiFetch(`/v1/retailer/org/members/${userId}`, {
        method: "DELETE",
        headers: { "Idempotency-Key": `team-deact-${userId}-${Date.now()}` },
      });
      if (!res.ok) {
        const json = await res.json().catch(() => ({}));
        throw new Error(
          (json as { error?: string }).error || `deactivate_failed_${res.status}`,
        );
      }
      setBanner("Member deactivated");
      await load();
    } catch (e) {
      setBanner(e instanceof Error ? e.message : t("retailer_desktop.residual.text.deactivate_failed"));
    } finally {
      setBusy(false);
    }
  };

  return (
    <PageChrome
      title={t("retailer_desktop.settings.team.text.team")}
      description={t("retailer_desktop.residual.text.invite_staff_with_roles_owner_cannot_be_demoted_or_deactivated")}
      actions={
        <button
          type="button"
          onClick={() => router.push("/settings")}
          className="inline-flex items-center gap-2 rounded-lg border border-border px-3 py-2 text-sm hover:bg-muted"
        >
          <ArrowLeft className="h-4 w-4" />
          Settings
        </button>
      }
    >
      <div className="mx-auto max-w-3xl space-y-6 px-4 pb-16 pt-2">
        {banner && (
          <div className="rounded-lg border border-border bg-muted/40 px-3 py-2 text-sm">
            {banner}
          </div>
        )}
        {error && (
          <div className="flex items-center gap-2 text-sm text-red-600">
            <AlertTriangle className="h-4 w-4" />
            {error}
            <button type="button" className="underline" onClick={() => void load()}>
              Retry
            </button>
          </div>
        )}

        <section className="rounded-xl border border-border bg-card p-4">
          <h2 className="mb-3 flex items-center gap-2 font-semibold">
            <UserPlus className="h-4 w-4" />
            Invite staff
          </h2>
          <div className="grid gap-3 sm:grid-cols-2">
            <label htmlFor="invite-staff-name" className="text-sm">
              Name
              <input
                id="invite-staff-name"
                aria-label="Name"
                className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2"
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              />
            </label>
            <label htmlFor="invite-staff-phone" className="text-sm">
              Phone
              <input
                id="invite-staff-phone"
                aria-label="Phone"
                className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2"
                value={form.phone}
                onChange={(e) => setForm((f) => ({ ...f, phone: e.target.value }))}
                placeholder={t("retailer_desktop.settings.team.text.998")}
              />
            </label>
            <label htmlFor="invite-staff-password" className="text-sm">
              Temporary password
              <input
                id="invite-staff-password"
                aria-label="Temporary password"
                type="password"
                className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2"
                value={form.password}
                onChange={(e) => setForm((f) => ({ ...f, password: e.target.value }))}
              />
            </label>
            <label className="text-sm">
              Role
              <select
                className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2"
                value={form.retailer_role}
                onChange={(e) =>
                  setForm((f) => ({ ...f, retailer_role: e.target.value }))
                }
              >
                {ROLES.map((r) => (
                  <option key={r} value={r}>
                    {r}
                  </option>
                ))}
              </select>
            </label>
          </div>
          <button
            type="button"
            disabled={busy}
            onClick={() => void invite()}
            className="mt-4 rounded-lg bg-foreground px-4 py-2 text-sm font-medium text-background disabled:opacity-50"
          >
            {busy ? "…" : "Create member"}
          </button>
          <p className="mt-2 text-xs text-muted-foreground">
            Creating the first staff member auto-enables the TEAM capability pack.
          </p>
        </section>

        <section className="rounded-xl border border-border bg-card p-4">
          <h2 className="mb-3 flex items-center gap-2 font-semibold">
            <Users className="h-4 w-4" />
            Roster
          </h2>
          {loading && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              Loading…
            </div>
          )}
          <ul className="divide-y divide-border">
            {items.map((m) => (
              <li
                key={m.user_id}
                className="flex flex-wrap items-center justify-between gap-2 py-3"
              >
                <div>
                  <div className="font-medium">
                    {m.name}{" "}
                    {m.is_owner && (
                      <span className="ml-1 rounded-full bg-muted px-2 py-0.5 text-[10px] uppercase">
                        Owner
                      </span>
                    )}
                    {!m.is_active && (
                      <span className="ml-1 text-xs text-red-600">{t("retailer_desktop.settings.team.text.inactive")}</span>
                    )}
                  </div>
                  <div className="text-xs text-muted-foreground">
                    {m.phone} ·{" "}
                    <Shield className="inline h-3 w-3" /> {m.retailer_role}
                  </div>
                </div>
                {!m.is_owner && m.is_active && (
                  <div className="flex flex-wrap items-center gap-2">
                    {isOwner && (
                      <button
                        type="button"
                        disabled={busy}
                        onClick={() => setMandateTarget(m)}
                        className="rounded-lg border border-amber-300 bg-amber-50 px-3 py-1.5 text-xs font-medium text-amber-800 hover:bg-amber-100"
                      >
                        {w2.mandateButton}
                      </button>
                    )}
                    <button
                      type="button"
                      disabled={busy}
                      onClick={() => void deactivate(m.user_id)}
                      className="rounded-lg border border-border px-3 py-1.5 text-xs hover:bg-muted"
                    >
                      Deactivate
                    </button>
                  </div>
                )}
              </li>
            ))}
          </ul>
        </section>

        {isOwner && (
          <OwnerApprovalsInbox
            items={approvals}
            statusFilter={approvalStatus}
            loading={approvalsLoading}
            error={approvalsError}
            banner={approvalBanner}
            decidingId={decidingId}
            onStatusFilterChange={(next) => {
              setApprovalStatus(next);
              void loadApprovals(next);
            }}
            onRefresh={() => void loadApprovals(approvalStatus)}
            onApprove={(id, reason) => void handleDecision(id, true, reason)}
            onReject={(id, reason) => void handleDecision(id, false, reason)}
          />
        )}
      </div>

      <SellerMandateModal
        member={mandateTarget}
        onClose={() => setMandateTarget(null)}
        onSaved={(msg) => setBanner(msg)}
      />
    </PageChrome>
  );
}
