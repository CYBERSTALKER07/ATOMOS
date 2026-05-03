'use client';

import { useEffect, useState } from 'react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';
import { useToast } from '@/components/Toast';
import type {
  CreateWarehouseStaffRequest,
  CreateWarehouseStaffResponse,
  WarehouseStaffListResponse,
  WarehouseStaffMember,
  WarehouseStaffRole,
} from '@pegasus/types/warehouse';

export default function StaffPage() {
  const { toast } = useToast();
  const [staff, setStaff] = useState<WarehouseStaffMember[]>([]);
  const [loading, setLoading] = useState(true);
  const [createdStaff, setCreatedStaff] = useState<CreateWarehouseStaffResponse | null>(null);

  // Registration form state
  const [showForm, setShowForm] = useState(false);
  const [formName, setFormName] = useState('');
  const [formPhone, setFormPhone] = useState('');
  const [formRole, setFormRole] = useState<WarehouseStaffRole>('WAREHOUSE_STAFF');
  const [submitting, setSubmitting] = useState(false);

  async function loadStaff() {
    setLoading(true);
    try {
      const res = await apiFetch('/v1/warehouse/ops/staff');
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to load staff');
      }
      const data = await res.json() as WarehouseStaffListResponse;
      setStaff(Array.isArray(data.staff) ? data.staff : []);
    } catch (error) {
      toast(error instanceof Error ? error.message : 'Failed to load staff', 'error');
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { loadStaff(); }, []); // eslint-disable-line react-hooks/exhaustive-deps

  async function handleRegister(e: React.FormEvent) {
    e.preventDefault();
    setSubmitting(true);

    try {
      const body: CreateWarehouseStaffRequest = {
        name: formName,
        phone: formPhone,
        role: formRole,
      };
      const res = await apiFetch('/v1/warehouse/ops/staff', {
        method: 'POST',
        body: JSON.stringify(body),
      });

      if (res.ok) {
        const data = await res.json() as CreateWarehouseStaffResponse;
        setCreatedStaff(data);
        toast('Staff member created', 'success');
        setShowForm(false);
        setFormName('');
        setFormPhone('');
        setFormRole('WAREHOUSE_STAFF');
        loadStaff();
      } else {
        const data = await res.json().catch(() => ({}));
        toast(data.error || 'Failed to create staff member', 'error');
      }
    } catch {
      toast('Network error', 'error');
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="p-6 space-y-6 md-animate-in">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold tracking-tight">Staff</h1>
        <button
          onClick={() => setShowForm(!showForm)}
          className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-semibold button--primary"
        >
          <Icon name="plus" size={16} />
          Add Staff
        </button>
      </div>

      {createdStaff && (
        <div
          className="rounded-xl border border-(--border) p-4"
          style={{ background: 'var(--surface)' }}
        >
          <div className="flex items-start justify-between gap-4">
            <div>
              <p className="text-sm font-semibold">One-time PIN generated</p>
              <p className="mt-1 text-sm text-(--muted)">
                Save this now for {createdStaff.name || 'the new staff member'}.
              </p>
              <p className="mt-3 font-mono text-lg tracking-[0.2em]">{createdStaff.pin}</p>
            </div>
            <button
              type="button"
              onClick={() => setCreatedStaff(null)}
              className="px-3 py-1.5 rounded-lg text-xs button--secondary border border-(--border)"
            >
              Dismiss
            </button>
          </div>
        </div>
      )}

      {/* Registration form */}
      {showForm && (
        <form
          onSubmit={handleRegister}
          className="rounded-xl border border-(--border) p-6 space-y-4"
          style={{ background: 'var(--surface)' }}
        >
          <h2 className="text-sm font-semibold">Create New Staff Member</h2>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-medium mb-1.5 text-(--muted)">Name</label>
              <input
                type="text"
                value={formName}
                onChange={e => setFormName(e.target.value)}
                required
                className="w-full px-3 py-2.5 rounded-lg border text-sm outline-none"
                style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)' }}
              />
            </div>
            <div>
              <label className="block text-xs font-medium mb-1.5 text-(--muted)">Phone</label>
              <input
                type="tel"
                value={formPhone}
                onChange={e => setFormPhone(e.target.value)}
                required
                className="w-full px-3 py-2.5 rounded-lg border text-sm outline-none"
                style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)' }}
              />
            </div>
            <div>
              <label className="block text-xs font-medium mb-1.5 text-(--muted)">PIN</label>
              <div
                className="w-full px-3 py-2.5 rounded-lg border text-sm text-(--muted)"
                style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)' }}
              >
                Generated by server
              </div>
            </div>
            <div>
              <label className="block text-xs font-medium mb-1.5 text-(--muted)">Role</label>
              <select
                value={formRole}
                onChange={e => setFormRole(e.target.value as WarehouseStaffRole)}
                className="w-full px-3 py-2.5 rounded-lg border text-sm outline-none"
                style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)' }}
              >
                <option value="WAREHOUSE_STAFF">Warehouse Staff</option>
                <option value="PAYLOADER">Payloader</option>
              </select>
            </div>
          </div>
          <div className="flex gap-2">
            <button
              type="submit"
              disabled={submitting}
              className="px-4 py-2 rounded-lg text-sm font-semibold button--primary disabled:opacity-50"
            >
              {submitting ? 'Creating...' : 'Create'}
            </button>
            <button
              type="button"
              onClick={() => setShowForm(false)}
              className="px-4 py-2 rounded-lg text-sm button--secondary border border-(--border)"
            >
              Cancel
            </button>
          </div>
        </form>
      )}

      {loading ? (
        <div className="space-y-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="md-skeleton md-skeleton-row" />
          ))}
        </div>
      ) : staff.length === 0 ? (
        <div className="text-center py-20 text-(--muted)">
          <Icon name="staff" size={48} className="mx-auto mb-3 opacity-30" />
          <p className="text-sm">No staff members registered</p>
        </div>
      ) : (
        <div className="border border-(--border) rounded-xl overflow-hidden">
          <table className="w-full text-sm">
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
      )}
    </div>
  );
}
