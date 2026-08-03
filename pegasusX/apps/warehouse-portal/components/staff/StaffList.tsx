<<<<<<< HEAD
import type { WarehouseStaffMember } from '@pegasusx/types';
import Icon from '@/components/Icon';

interface StaffListProps {
  staff: WarehouseStaffMember[];
  loading?: boolean;
}

export function StaffList({ staff, loading }: StaffListProps) {
=======
import Icon from '@/components/Icon';
import type { WarehouseStaffMember } from '@pegasusx/types';

interface StaffListProps {
  staff: WarehouseStaffMember[];
  loading: boolean;
}

export default function StaffList({ staff, loading }: StaffListProps) {
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
  if (loading) {
    return (
      <div className="space-y-2">
        {Array.from({ length: 3 }).map((_, i) => (
          <div key={i} className="md-skeleton md-skeleton-row" />
        ))}
      </div>
    );
  }

  if (staff.length === 0) {
    return (
      <div className="text-center py-20 text-(--muted)">
        <Icon name="staff" size={48} className="mx-auto mb-3 opacity-30" />
        <p className="text-sm">No staff members registered</p>
      </div>
    );
  }

  return (
    <div className="border border-(--border) rounded-xl overflow-hidden">
      <table className="desk-table w-full text-sm">
        <thead>
          <tr className="border-b border-(--border)" style={{ background: 'var(--surface)' }}>
            <th className="text-left px-4 py-3 font-semibold text-(--muted)">Name</th>
            <th className="text-left px-4 py-3 font-semibold text-(--muted)">Phone</th>
            <th className="text-left px-4 py-3 font-semibold text-(--muted)">Role</th>
            <th className="text-left px-4 py-3 font-semibold text-(--muted)">Status</th>
          </tr>
        </thead>
        <tbody>
          {staff.map(s => (
            <tr key={s.worker_id} className="border-b border-(--border) last:border-b-0">
              <td className="px-4 py-3">{s.name}</td>
              <td className="px-4 py-3 text-(--muted)">{s.phone || '—'}</td>
              <td className="px-4 py-3">
                <span className="status-chip status-chip--submitted">{s.role}</span>
              </td>
              <td className="px-4 py-3">
                <span className={`text-xs font-semibold ${s.is_active ? 'text-(--success)' : 'text-(--danger)'}`}>
                  {s.is_active ? 'Active' : 'Inactive'}
                </span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
