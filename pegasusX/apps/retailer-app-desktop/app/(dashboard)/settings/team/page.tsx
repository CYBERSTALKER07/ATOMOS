"use client";

import { usePortalT } from "@/lib/i18n";
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
import { apiFetch } from "@/lib/auth";

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
  const router = useRouter();
  const [items, setItems] = useState<Member[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [banner, setBanner] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [form, setForm] = useState({
    name: "",
    phone: "",
    password: "",
    retailer_role: "CASHIER",
  });

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await apiFetch("/v1/retailer/org/members");
      if (!res.ok) throw new Error(`load_failed_${res.status}`);
      const json = (await res.json()) as { items?: Member[] };
      setItems(json.items ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : t("retailer_desktop.residual.text.failed_to_load_team"));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

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
            <label className="text-sm">
              Name
              <input
                className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2"
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              />
            </label>
            <label className="text-sm">
              Phone
              <input
                className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2"
                value={form.phone}
                onChange={(e) => setForm((f) => ({ ...f, phone: e.target.value }))}
                placeholder={t("retailer_desktop.settings.team.text.998")}
              />
            </label>
            <label className="text-sm">
              Temporary password
              <input
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
                  <button
                    type="button"
                    disabled={busy}
                    onClick={() => void deactivate(m.user_id)}
                    className="rounded-lg border border-border px-3 py-1.5 text-xs hover:bg-muted"
                  >
                    Deactivate
                  </button>
                )}
              </li>
            ))}
          </ul>
        </section>
      </div>
    </PageChrome>
  );
}
