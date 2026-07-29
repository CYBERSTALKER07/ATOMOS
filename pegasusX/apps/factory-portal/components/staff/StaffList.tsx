'use client';

import Link from 'next/link';
import { motion } from 'framer-motion';

export interface StaffMember {
  id: string;
  name: string;
  phone: string;
  role: string;
  status: string;
  created_at: string;
}

interface StaffListProps {
  staff: StaffMember[];
}

export function StaffList({ staff }: StaffListProps) {
  return (
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
  );
}
