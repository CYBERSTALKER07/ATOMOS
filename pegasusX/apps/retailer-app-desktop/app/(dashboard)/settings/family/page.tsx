"use client";

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import {
  ArrowLeft,
  ArrowRightLeft,
  Copy,
  Loader2,
  Plus,
  Trash2,
  Users,
} from "lucide-react";
import { motion } from "framer-motion";
import { PageChrome } from "@/components/PageChrome";
import { apiFetch } from "../../../../lib/auth";

type FamilyMember = {
  member_id: string;
  name: string;
  phone?: string;
  created_at?: string;
};

type MigrateItem = {
  member_id: string;
  user_id: string;
  phone: string;
  name: string;
  retailer_role: string;
  temp_password?: string;
};

type MigrateSkipped = {
  member_id: string;
  phone?: string;
  reason: string;
};

type MigrateResult = {
  retailer_id: string;
  migrated: MigrateItem[];
  skipped: MigrateSkipped[];
  family_remaining: number;
  family_writes: string;
};

export default function FamilyMembersPage() {
  const t = usePortalT();
  const router = useRouter();
  const [members, setMembers] = useState<FamilyMember[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [name, setName] = useState("");
  const [phone, setPhone] = useState("");
  const [saving, setSaving] = useState(false);
  const [migrating, setMigrating] = useState(false);
  const [familyGone, setFamilyGone] = useState(false);
  const [migrateResult, setMigrateResult] = useState<MigrateResult | null>(null);
  const [copiedId, setCopiedId] = useState<string | null>(null);

  const loadMembers = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await apiFetch("/v1/retailer/family-members");
      if (res.status === 410) {
        setFamilyGone(true);
        setMembers([]);
        return;
      }
      if (!res.ok) {
        throw new Error("Could not load family members");
      }
      const data = (await res.json()) as {
        members?: FamilyMember[];
        family_writes?: string;
      };
      setMembers(Array.isArray(data.members) ? data.members : []);
      if (data.family_writes === "gone") {
        setFamilyGone(true);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : t("retailer_desktop.residual.text.could_not_load_family_members"));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadMembers();
  }, [loadMembers]);

  const addMember = async () => {
    const trimmedName = name.trim();
    if (!trimmedName) return;
    setSaving(true);
    setError(null);
    try {
      const res = await apiFetch("/v1/retailer/family-members", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: trimmedName,
          phone: phone.trim(),
        }),
      });
      if (res.status === 410) {
        setFamilyGone(true);
        const body = (await res.json().catch(() => ({}))) as {
          message?: string;
          migrate?: string;
        };
        setError(
          body.message ||
            "Family writes are closed. Use Team staff, or run Migrate to Team for legacy rows.",
        );
        return;
      }
      if (!res.ok) {
        throw new Error("Could not add family member");
      }
      setName("");
      setPhone("");
      await loadMembers();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("retailer_desktop.residual.text.could_not_add_family_member"));
    } finally {
      setSaving(false);
    }
  };

  const removeMember = async (memberId: string) => {
    setError(null);
    try {
      const res = await apiFetch(`/v1/retailer/family-members/${memberId}`, {
        method: "DELETE",
      });
      if (!res.ok) {
        throw new Error("Could not remove family member");
      }
      await loadMembers();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("retailer_desktop.residual.text.could_not_remove_family_member"));
    }
  };

  const migrateToTeam = async () => {
    if (
      !confirm(
        "Migrate all family members with a phone number into Team staff (RECEIVER)?\n\n" +
          "Each person gets a one-time temporary password — copy them before leaving this page.\n" +
          "Family add will be closed after migration.",
      )
    ) {
      return;
    }
    setMigrating(true);
    setError(null);
    setMigrateResult(null);
    try {
      const res = await apiFetch("/v1/retailer/family-members/migrate-to-team", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ retailer_role: "RECEIVER" }),
      });
      const json = (await res.json().catch(() => ({}))) as MigrateResult & {
        error?: string;
        detail?: string;
      };
      if (!res.ok) {
        throw new Error(json.error || json.detail || `migrate_failed_${res.status}`);
      }
      setMigrateResult(json);
      setFamilyGone(json.family_writes === "gone");
      await loadMembers();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("retailer_desktop.residual.text.migration_failed"));
    } finally {
      setMigrating(false);
    }
  };

  const copyPassword = async (userId: string, password: string) => {
    try {
      await navigator.clipboard.writeText(password);
      setCopiedId(userId);
      window.setTimeout(() => setCopiedId(null), 2000);
    } catch {
      setError(t("retailer_desktop.residual.text.could_not_copy_password_to_clipboard"));
    }
  };

  return (
    <PageChrome
      icon="settings"
      title={t("retailer_desktop.settings.family.text.family_members")}
      description={t("retailer_desktop.residual.text.legacy_family_list_prefer_team_staff_migrate_when_ready")}
      loading={loading}
      skeletonVariant="form"
      actions={
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => router.push("/settings/team")}
            className="portal-btn portal-btn--ghost h-11 px-4 rounded-xl font-light text-sm"
          >
            Open Team
          </button>
          <button
            type="button"
            onClick={() => router.push("/settings")}
            className="portal-btn portal-btn--ghost desk-icon-btn"
            aria-label={t("retailer_desktop.settings.family.text.back_to_settings")}
          >
            <ArrowLeft size={18} />
          </button>
        </div>
      }
    >
      <div className="max-w-2xl mx-auto space-y-6">
        {/* Migrate banner */}
        <div className="rounded-2xl border border-[var(--desk-accent)]/30 bg-[var(--desk-accent)]/5 p-4 space-y-3">
          <div className="flex items-start gap-3">
            <ArrowRightLeft
              size={20}
              className="mt-0.5 shrink-0 text-[var(--desk-accent)]"
            />
            <div className="flex-1 space-y-1">
              <p className="font-light text-[var(--desk-text-primary)]">
                Migrate Family → Team
              </p>
              <p className="text-sm text-[var(--desk-text-secondary)]">
                Converts contacts with a phone into Team RECEIVER accounts. Temporary
                passwords are shown once. After migrate, new family adds return 410 —
                use Settings → Team instead.
              </p>
            </div>
          </div>
          <button
            type="button"
            disabled={migrating || familyGone}
            onClick={() => void migrateToTeam()}
            className="portal-btn portal-btn--primary h-11 px-5 rounded-xl font-light inline-flex items-center gap-2 disabled:opacity-60"
          >
            {migrating ? (
              <Loader2 size={16} className="animate-spin" />
            ) : (
              <ArrowRightLeft size={16} />
            )}
            {familyGone ? "Already migrated — use Team" : migrating ? "Migrating…" : "Migrate to Team"}
          </button>
          {familyGone && (
            <p className="text-xs text-[var(--desk-text-tertiary)]">
              Family writes are closed (durable). Manage staff under Settings → Team.
            </p>
          )}
        </div>

        {migrateResult && (
          <div className="rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface)] p-4 space-y-4">
            <div>
              <p className="font-light text-[var(--desk-text-primary)]">{t("retailer_desktop.settings.family.text.migration_result")}</p>
              <p className="text-sm text-[var(--desk-text-secondary)] mt-1">
                {migrateResult.migrated?.length ?? 0} migrated ·{" "}
                {migrateResult.skipped?.length ?? 0} skipped ·{" "}
                {migrateResult.family_remaining ?? 0} remaining on family list
              </p>
            </div>

            {(migrateResult.migrated?.length ?? 0) > 0 && (
              <div className="space-y-2">
                <p className="text-xs uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                  Temporary passwords (copy now)
                </p>
                {migrateResult.migrated.map((m) => (
                  <div
                    key={m.user_id}
                    className="flex items-center justify-between gap-3 rounded-xl border border-[var(--desk-border)] bg-[var(--desk-canvas)] px-3 py-2"
                  >
                    <div className="min-w-0">
                      <p className="text-sm font-light text-[var(--desk-text-primary)] truncate">
                        {m.name}
                      </p>
                      <p className="text-xs text-[var(--desk-text-tertiary)]">
                        {m.phone} · {m.retailer_role}
                      </p>
                      {m.temp_password && (
                        <p className="text-sm font-mono text-[var(--desk-text-primary)] mt-0.5">
                          {m.temp_password}
                        </p>
                      )}
                    </div>
                    {m.temp_password && (
                      <button
                        type="button"
                        onClick={() => void copyPassword(m.user_id, m.temp_password!)}
                        className="portal-btn portal-btn--ghost h-9 px-3 rounded-lg text-xs shrink-0 inline-flex items-center gap-1.5"
                      >
                        <Copy size={14} />
                        {copiedId === m.user_id ? "Copied" : "Copy"}
                      </button>
                    )}
                  </div>
                ))}
              </div>
            )}

            {(migrateResult.skipped?.length ?? 0) > 0 && (
              <div className="space-y-1">
                <p className="text-xs uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                  Skipped
                </p>
                {migrateResult.skipped.map((s) => (
                  <p
                    key={s.member_id}
                    className="text-sm text-[var(--desk-text-secondary)]"
                  >
                    {s.phone || s.member_id}: {s.reason}
                  </p>
                ))}
              </div>
            )}

            <button
              type="button"
              onClick={() => router.push("/settings/team")}
              className="portal-btn portal-btn--ghost h-10 px-4 rounded-xl text-sm font-light"
            >
              Manage in Team →
            </button>
          </div>
        )}

        {!familyGone && (
          <div className="rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface)] p-4 space-y-3">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              <input
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder={t("retailer_desktop.pos.text.name")}
                className="h-11 px-4 rounded-xl border border-[var(--desk-border)] bg-[var(--desk-canvas)]"
              />
              <input
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                placeholder={t("retailer_desktop.settings.family.text.phone_required_for_team_migrate")}
                className="h-11 px-4 rounded-xl border border-[var(--desk-border)] bg-[var(--desk-canvas)]"
              />
            </div>
            <button
              type="button"
              disabled={saving || !name.trim()}
              onClick={() => void addMember()}
              className="portal-btn portal-btn--primary h-11 px-5 rounded-xl font-light inline-flex items-center gap-2 disabled:opacity-60"
            >
              {saving ? <Loader2 size={16} className="animate-spin" /> : <Plus size={16} />}
              Add Member
            </button>
          </div>
        )}

        {familyGone && !migrateResult && (
          <div className="rounded-2xl border border-dashed border-[var(--desk-border)] p-6 text-center space-y-2">
            <p className="font-light text-[var(--desk-text-primary)]">
              Family writes are closed
            </p>
            <p className="text-sm text-[var(--desk-text-secondary)]">
              Manage staff under Settings → Team.
            </p>
            <button
              type="button"
              onClick={() => router.push("/settings/team")}
              className="portal-btn portal-btn--primary h-10 px-4 rounded-xl text-sm font-light mt-2"
            >
              Open Team
            </button>
          </div>
        )}

        {error && (
          <p className="text-sm font-semibold text-red-600">{error}</p>
        )}

        {loading ? (
          <div className="flex justify-center py-16">
            <Loader2 size={24} className="animate-spin text-[var(--desk-text-tertiary)]" />
          </div>
        ) : members.length === 0 && !familyGone ? (
          <div className="rounded-2xl border border-dashed border-[var(--desk-border)] p-10 text-center">
            <Users size={28} className="mx-auto mb-3 text-[var(--desk-text-tertiary)]" />
            <p className="font-light text-[var(--desk-text-primary)]">{t("retailer_desktop.settings.family.text.no_family_members_yet")}</p>
            <p className="text-sm text-[var(--desk-text-secondary)] mt-1">
              Add members with a phone number, then migrate them to Team.
            </p>
          </div>
        ) : members.length > 0 ? (
          <div className="space-y-2">
            {members.map((member, index) => (
              <motion.div
                key={member.member_id}
                initial={{ opacity: 0, y: 8 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: index * 0.03 }}
                className="flex items-center justify-between rounded-xl border border-[var(--desk-border)] bg-[var(--desk-surface)] px-4 py-3"
              >
                <div>
                  <p className="font-light text-[var(--desk-text-primary)]">{member.name}</p>
                  {member.phone ? (
                    <p className="text-xs text-[var(--desk-text-tertiary)]">{member.phone}</p>
                  ) : (
                    <p className="text-xs text-amber-600">{t("retailer_desktop.settings.family.text.no_phone_skipped_on_migrate")}</p>
                  )}
                </div>
                <button
                  type="button"
                  onClick={() => void removeMember(member.member_id)}
                  className="w-9 h-9 rounded-lg border border-[var(--desk-border)] flex items-center justify-center text-red-600"
                  aria-label={`Remove ${member.name}`}
                >
                  <Trash2 size={16} />
                </button>
              </motion.div>
            ))}
          </div>
        ) : null}
      </div>
    </PageChrome>
  );
}
