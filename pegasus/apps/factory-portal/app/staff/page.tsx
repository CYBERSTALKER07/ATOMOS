'use client';

import { useEffect, useState, useCallback } from 'react';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import EmptyState from '@/components/EmptyState';
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

  const load = useCallback(async () => {
    try {
      const res = await apiFetch('/v1/factory/staff');
      if (res.ok) {
        const data = await res.json();
        setStaff(data.staff || []);
      }
    } catch { /* handled */ } finally {
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
      <div className="p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h1 className="text-xl font-bold tracking-tight text-[var(--foreground)]">Factory Staff</h1>
          <motion.button 
            whileHover={{ scale: 1.05 }}
            whileTap={{ scale: 0.95 }}
            onClick={() => load()} 
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary hover-lift active-press"
          >
            <Icon name="refresh" size={16} /> Refresh
          </motion.button>
        </div>

        {loading ? (
          <div className="space-y-1">
            {Array.from({ length: 4 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}
          </div>
        ) : staff.length === 0 ? (
          <EmptyState
            imageUrl="/images/empty-products.png"
            headline="No staff registered"
            body="There are no staff members registered for this factory."
          />
        ) : (
          <motion.div 
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            className="overflow-x-auto rounded-xl border border-[var(--border)] bg-[var(--surface)]"
          >
            <table className="w-full text-sm">
              <thead>
                <tr className="table__header border-b border-[var(--border)] bg-[var(--default)]">
                  <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Name</th>
                  <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Phone</th>
                  <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Role</th>
                  <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Status</th>
                  <th className="table__column text-right py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Joined</th>
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
                  </motion.tr>
                ))}
              </tbody>
            </table>
          </motion.div>
        )}
      </div>
    </PageTransition>
  );
}
