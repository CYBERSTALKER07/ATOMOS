'use client';

import Link from 'next/link';
import { useEffect, useState, useCallback } from 'react';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { motion } from 'framer-motion';

interface StaffMember {
  id: string;
  name: string;
  phone: string;
  role: string;
  status: string;
  created_at: string;
}

export default function StaffPage() {
  const [staff, setStaff] = useState<StaffMember[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    try {
      const res = await apiFetch('/v1/factory/staff');
      if (res.ok) {
        const data = await res.json();
        setStaff(data.staff || []);
      } else {
        setError(`Unable to load staff (${res.status}).`);
      }
    } catch {
      setError('Unable to load staff right now.');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  useEffect(() => {
    const unsubscribe = subscribeFactoryWS({
      onMessage: payload => {
        const event = parseFactoryLiveEvent(payload);
        if (!event) {
          return;
        }
        void load();
      },
    });

    return () => {
      unsubscribe();
    };
  }, [load]);

  return (
    <PageTransition>
      <PageChrome
        icon="staff"
        title="Factory staff"
        description="Operators and shift coverage registered for this factory node."
        loading={loading}
        skeletonVariant="table"
        error={error && staff.length === 0 ? error : null}
        empty={!loading && !error && staff.length === 0}
        emptyMessage="There are no staff members registered for this factory."
        actions={
          <button type="button" className="portal-btn portal-btn--ghost inline-flex items-center gap-1.5" onClick={() => void load()}>
            <Icon name="refresh" size={16} /> Refresh
          </button>
        }
      >
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="desk-table-wrap"
        >
          <table className="w-full text-sm">
            <thead>
              <tr className="table__header border-b border-[var(--border)] bg-[var(--default)]">
                <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Name</th>
                <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Phone</th>
                <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Role</th>
                <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Status</th>
                <th className="table__column text-right py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Joined</th>
                <th className="table__column text-right py-3 px-4 font-medium uppercase tracking-wider text-[11px]"></th>
              </tr>
            </thead>
            <tbody>
              {staff.map((s, index) => (
                <motion.tr
                  key={s.id}
                  initial={{ opacity: 0, x: -10 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ delay: index * 0.05 }}
                  className="table__row border-b border-[var(--border)] last:border-0 hover:bg-[var(--default)]/50 transition-colors"
                >
                  <td className="py-3 px-4 font-medium">{s.name}</td>
                  <td className="py-3 px-4 text-[var(--muted)]">{s.phone}</td>
                  <td className="py-3 px-4">{s.role}</td>
                  <td className="py-3 px-4">
                    <span className={`status-chip ${s.status === 'ACTIVE' ? 'status-chip--stable' : 'status-chip--draft'}`}>
                      {s.status}
                    </span>
                  </td>
                  <td className="py-3 px-4 text-right text-[var(--muted)] tabular-nums font-mono">
                    {new Date(s.created_at).toLocaleDateString()}
                  </td>
                  <td className="py-3 px-4 text-right">
                    <Link href={`/staff/${s.id}`} className="portal-btn portal-btn--ghost text-xs">
                      Open
                    </Link>
                  </td>
                </motion.tr>
              ))}
            </tbody>
          </table>
        </motion.div>
      </PageChrome>
    </PageTransition>
  );
}
