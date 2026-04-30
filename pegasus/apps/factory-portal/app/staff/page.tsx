'use client';

import { useEffect, useState, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';

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

  return (
    <div className="p-6 space-y-4 md-animate-in">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold tracking-tight">Factory Staff</h1>
        <button onClick={() => load()} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary">
          <Icon name="refresh" size={16} /> Refresh
        </button>
      </div>

      {loading ? (
        <div className="space-y-1">
          {Array.from({ length: 4 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}
        </div>
      ) : staff.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-[var(--muted)]">
          <Icon name="staff" size={48} className="mb-3 opacity-40" />
          <p className="text-sm">No staff registered</p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="table__header border-b border-[var(--border)]">
                <th className="table__column text-left py-2 px-3 font-medium">Name</th>
                <th className="table__column text-left py-2 px-3 font-medium">Phone</th>
                <th className="table__column text-left py-2 px-3 font-medium">Role</th>
                <th className="table__column text-left py-2 px-3 font-medium">Status</th>
                <th className="table__column text-right py-2 px-3 font-medium">Joined</th>
              </tr>
            </thead>
            <tbody>
              {staff.map(s => (
                <tr key={s.id} className="table__row">
                  <td className="py-2.5 px-3 font-medium">{s.name}</td>
                  <td className="py-2.5 px-3 text-[var(--muted)]">{s.phone}</td>
                  <td className="py-2.5 px-3">{s.role}</td>
                  <td className="py-2.5 px-3">
                    <span className={`status-chip ${s.status === 'ACTIVE' ? 'status-chip--stable' : 'status-chip--draft'}`}>
                      {s.status}
                    </span>
                  </td>
                  <td className="py-2.5 px-3 text-right text-[var(--muted)]">
                    {new Date(s.created_at).toLocaleDateString()}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
