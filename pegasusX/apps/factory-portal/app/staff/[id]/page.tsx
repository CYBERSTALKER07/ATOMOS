'use client';

import { usePortalT } from "@/lib/i18n";
import Link from 'next/link';
import { useCallback, useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import { apiFetch } from '@/lib/auth';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';

interface StaffDetail {
  id: string;
  staff_id?: string;
  name: string;
  role: string;
  phone?: string;
  status?: string;
  joined_at?: string;
}

export default function StaffDetailPage() {
  const t = usePortalT();
  const { id } = useParams<{ id: string }>();
  const [staff, setStaff] = useState<StaffDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    setError(null);
    try {
      const res = await apiFetch(`/v1/factory/staff/${id}`);
      if (!res.ok) throw new Error(`Unable to load staff member (${res.status})`);
      setStaff(await res.json());
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Unable to load staff member');
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => { void load(); }, [load]);

  useFactorySessionReconcile(() => {
    void load();
  });

  if (loading) {
    return (
      <PageTransition>
        <PageChrome icon="staff" title={t("factory_portal.staff._id_.text.staff_detail")} description={t("factory_portal.residual.text.loading_operator_profile")} loading skeletonVariant="form">
          <span />
        </PageChrome>
      </PageTransition>
    );
  }

  if (error || !staff) {
    return (
      <PageTransition>
        <PageChrome icon="staff" title={t("factory_portal.staff._id_.text.staff_detail")} error={error || 'Not found'}>
          <span />
        </PageChrome>
      </PageTransition>
    );
  }

  return (
    <PageTransition>
      <PageChrome icon="staff" title={staff.name} description={staff.role}>
        <Link href="/staff" className="text-sm text-[var(--muted)] hover:text-[var(--foreground)]">← Back to staff</Link>
        <div className="desk-card mt-6 p-6 space-y-4 max-w-lg">
          <div>
            <p className="text-xs uppercase tracking-wide text-[var(--muted)]">{t("factory_portal.staff._id_.text.staff_id")}</p>
            <p className="font-mono text-sm mt-1">{staff.staff_id || staff.id}</p>
          </div>
          <div>
            <p className="text-xs uppercase tracking-wide text-[var(--muted)]">{t("common.field.phone")}</p>
            <p className="text-sm mt-1">{staff.phone?.trim() || '—'}</p>
          </div>
          <div>
            <p className="text-xs uppercase tracking-wide text-[var(--muted)]">{t("factory_portal.fleet.text.status")}</p>
            <p className="text-sm mt-1">{staff.status || 'ACTIVE'}</p>
          </div>
          <div>
            <p className="text-xs uppercase tracking-wide text-[var(--muted)]">{t("factory_portal.staff._id_.text.joined")}</p>
            <p className="text-sm mt-1">{staff.joined_at?.trim() || '—'}</p>
          </div>
        </div>
        <SetPasswordForm staffId={String(staff.staff_id || staff.id || id)} />
      </PageChrome>
    </PageTransition>
  );
}

function SetPasswordForm({ staffId }: { staffId: string }) {
  const [pin, setPin] = useState("");
  const [busy, setBusy] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);

  return (
    <form
      className="desk-card mt-6 p-6 space-y-3 max-w-lg"
      onSubmit={async (e) => {
        e.preventDefault();
        setBusy(true);
        setMsg(null);
        try {
          const res = await apiFetch(`/v1/factory/staff/${encodeURIComponent(staffId)}/set-password`, {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
              "Idempotency-Key": `factory-staff-set-password:${staffId}:${Date.now()}`,
            },
            body: JSON.stringify({ pin }),
          });
          const body = await res.json().catch(() => ({}));
          if (!res.ok) throw new Error(body.error || `status_${res.status}`);
          setMsg("Password set. Staff can log in with this PIN.");
          setPin("");
        } catch (err) {
          setMsg(err instanceof Error ? err.message : "set_password_failed");
        } finally {
          setBusy(false);
        }
      }}
    >
      <h2 className="text-sm font-semibold">Set login PIN</h2>
      <p className="text-xs text-[var(--muted)]">Never stored in plaintext. Invite rows must set a PIN before login.</p>
      <input
        type="password"
        minLength={4}
        required
        value={pin}
        onChange={(e) => setPin(e.target.value)}
        placeholder="PIN or password (min 4)"
        className="w-full rounded border px-3 py-2 text-sm"
      />
      <button type="submit" disabled={busy || pin.trim().length < 4} className="portal-btn portal-btn--primary text-sm">
        {busy ? "Saving…" : "Set password"}
      </button>
      {msg ? <p className="text-sm">{msg}</p> : null}
    </form>
  );
}
