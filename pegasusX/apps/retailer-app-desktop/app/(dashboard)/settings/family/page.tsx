"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { ArrowLeft, Loader2, Plus, Trash2, Users } from "lucide-react";
import { motion } from "framer-motion";
import { PageChrome } from "@/components/PageChrome";
import { apiFetch } from "../../../../lib/auth";

type FamilyMember = {
  member_id: string;
  name: string;
  phone?: string;
  created_at?: string;
};

export default function FamilyMembersPage() {
  const router = useRouter();
  const [members, setMembers] = useState<FamilyMember[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [name, setName] = useState("");
  const [phone, setPhone] = useState("");
  const [saving, setSaving] = useState(false);

  const loadMembers = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await apiFetch("/v1/retailer/family-members");
      if (!res.ok) {
        throw new Error("Could not load family members");
      }
      const data = (await res.json()) as { members?: FamilyMember[] };
      setMembers(Array.isArray(data.members) ? data.members : []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not load family members");
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
      if (!res.ok) {
        throw new Error("Could not add family member");
      }
      setName("");
      setPhone("");
      await loadMembers();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not add family member");
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
      setError(err instanceof Error ? err.message : "Could not remove family member");
    }
  };

  return (
    <PageChrome
      icon="settings"
      title="Family Members"
      description="Manage staff and family who can place orders on this account."
      loading={loading}
      skeletonVariant="form"
      actions={
        <button
          type="button"
          onClick={() => router.push("/settings")}
          className="portal-btn portal-btn--ghost desk-icon-btn"
          aria-label="Back to settings"
        >
          <ArrowLeft size={18} />
        </button>
      }
    >
    <div className="max-w-2xl mx-auto space-y-6">

      <div className="rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface)] p-4 space-y-3">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Name"
            className="h-11 px-4 rounded-xl border border-[var(--desk-border)] bg-[var(--desk-canvas)]"
          />
          <input
            value={phone}
            onChange={(e) => setPhone(e.target.value)}
            placeholder="Phone (optional)"
            className="h-11 px-4 rounded-xl border border-[var(--desk-border)] bg-[var(--desk-canvas)]"
          />
        </div>
        <button
          type="button"
          disabled={saving || !name.trim()}
          onClick={() => void addMember()}
          className="portal-btn portal-btn--primary h-11 px-5 rounded-xl font-bold inline-flex items-center gap-2 disabled:opacity-60"
        >
          {saving ? <Loader2 size={16} className="animate-spin" /> : <Plus size={16} />}
          Add Member
        </button>
      </div>

      {error && (
        <p className="text-sm font-semibold text-red-600">{error}</p>
      )}

      {loading ? (
        <div className="flex justify-center py-16">
          <Loader2 size={24} className="animate-spin text-[var(--desk-text-tertiary)]" />
        </div>
      ) : members.length === 0 ? (
        <div className="rounded-2xl border border-dashed border-[var(--desk-border)] p-10 text-center">
          <Users size={28} className="mx-auto mb-3 text-[var(--desk-text-tertiary)]" />
          <p className="font-bold text-[var(--desk-text-primary)]">No family members yet</p>
          <p className="text-sm text-[var(--desk-text-secondary)] mt-1">
            Add members to delegate ordering access.
          </p>
        </div>
      ) : (
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
                <p className="font-bold text-[var(--desk-text-primary)]">{member.name}</p>
                {member.phone && (
                  <p className="text-xs text-[var(--desk-text-tertiary)]">{member.phone}</p>
                )}
              </div>
              <button
                type="button"
                onClick={() => void removeMember(member.member_id)}
                className="w-9 h-9 rounded-lg border border-[var(--desk-border)] flex items-center justify-center text-red-600"
              >
                <Trash2 size={16} />
              </button>
            </motion.div>
          ))}
        </div>
      )}
    </div>
    </PageChrome>
  );
}
