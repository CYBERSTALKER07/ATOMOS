'use client';

import { useEffect, useState } from 'react';
import { warehouseCreateStaffKey } from '@pegasusx/api-client';
import { apiFetch } from '@/lib/auth';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { useToast } from '@/components/Toast';
import { StaffList } from '@/components/staff/StaffList';
import type {
  CreateWarehouseStaffRequest,
  CreateWarehouseStaffResponse,
  WarehouseStaffListResponse,
  WarehouseStaffMember,
  WarehouseStaffRole,
} from '@pegasusx/types';

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
        headers: {
          'Idempotency-Key': warehouseCreateStaffKey(warehouseHomeNodeId() || 'warehouse', formPhone),
        },
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
    <PageTransition>
      <PageChrome
        icon="staff"
        title="Staff"
        description="Warehouse staff and payloader accounts for terminal execution."
        actions={
          <button
            type="button"
            onClick={() => setShowForm(!showForm)}
            className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-semibold button--primary"
          >
            <Icon name="plus" size={16} />
            Add Staff
          </button>
        }
      >
      <div className="space-y-6">
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

      <StaffList staff={staff} loading={loading} />
      </div>
      </PageChrome>
    </PageTransition>
  );
}
