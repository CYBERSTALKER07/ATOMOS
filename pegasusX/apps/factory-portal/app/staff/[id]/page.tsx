'use client';

import Link from 'next/link';
import { useCallback, useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import { apiFetch } from '@/lib/auth';
import PageTransition from '@/components/PageTransition';
import FactoryPageState from '@/components/FactoryPageState';

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

  if (loading) {
    return (
      <PageTransition>
        <div className="p-6">
          <FactoryPageState kind="loading" title="Staff detail" subtitle="Loading operator profile." />
        </div>
      </PageTransition>
    );
  }

  if (error || !staff) {
    return (
      <PageTransition>
        <div className="p-6">
          <FactoryPageState kind="error" headline="Unable to load staff" body={error || 'Not found'} actionLabel="Retry" onAction={() => void load()} />
        </div>
      </PageTransition>
    );
  }

  return (
    <PageTransition className="space-y-6 p-6 md:p-8">
      <div>
        <Link href="/staff" className="text-sm text-[var(--muted)] hover:text-[var(--foreground)]">← Back to staff</Link>
        <h1 className="mt-2 text-2xl font-semibold tracking-tight text-[var(--foreground)]">{staff.name}</h1>
        <p className="mt-1 text-sm text-[var(--muted)]">{staff.role}</p>
      </div>
      <div className="md-card md-elevation-1 md-shape-md p-6 space-y-4 max-w-lg">
        <div>
          <p className="text-xs uppercase tracking-wide text-[var(--muted)]">Staff ID</p>
          <p className="font-mono text-sm mt-1">{staff.staff_id || staff.id}</p>
        </div>
        <div>
          <p className="text-xs uppercase tracking-wide text-[var(--muted)]">Phone</p>
          <p className="text-sm mt-1">{staff.phone?.trim() || '—'}</p>
        </div>
        <div>
          <p className="text-xs uppercase tracking-wide text-[var(--muted)]">Status</p>
          <p className="text-sm mt-1">{staff.status || 'ACTIVE'}</p>
        </div>
        <div>
          <p className="text-xs uppercase tracking-wide text-[var(--muted)]">Joined</p>
          <p className="text-sm mt-1">{staff.joined_at?.trim() || '—'}</p>
        </div>
      </div>
    </PageTransition>
  );
}
