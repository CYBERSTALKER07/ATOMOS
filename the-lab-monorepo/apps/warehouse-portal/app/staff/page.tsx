'use client';

import { useEffect, useState } from 'react';
import { apiFetch, decodeJwtPayload, readTokenFromCookie } from '@/lib/auth';
import Icon from '@/components/Icon';
import { useToast } from '@/components/Toast';

interface StaffMember {
  worker_id: string;
  name: string;
  phone: string;
  role: string;
  is_active: boolean;
  created_at: string;
}

export default function StaffPage() {
  const { toast } = useToast();
  const [staff, setStaff] = useState<StaffMember[]>([]);
  const [loading, setLoading] = useState(true);

  // Registration form state
  const [showForm, setShowForm] = useState(false);
  const [formName, setFormName] = useState('');
  const [formPhone, setFormPhone] = useState('');
  const [formPin, setFormPin] = useState('');
  const [formRole, setFormRole] = useState('WAREHOUSE_STAFF');
  const [submitting, setSubmitting] = useState(false);

  async function loadStaff() {
    setLoading(true);
    try {
      // Use claims to scope — staff list route would need to be a supplier endpoint
      // For now, show current user info from token
      const token = readTokenFromCookie();
      if (token) {
        const claims = decodeJwtPayload(token);
        if (claims) {
          // Placeholder: show current user
          setStaff([{
            worker_id: claims.user_id as string || '',
            name: (claims.name as string) || 'Current User',
            phone: '',
            role: (claims.warehouse_role as string) || 'WAREHOUSE_STAFF',
            is_active: true,
            created_at: new Date().toISOString(),
          }]);
        }
      }
    } catch {
      toast('Failed to load staff', 'error');
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { loadStaff(); }, []); // eslint-disable-line react-hooks/exhaustive-deps

  async function handleRegister(e: React.FormEvent) {
    e.preventDefault();
    setSubmitting(true);

    const token = readTokenFromCookie();
    const claims = token ? decodeJwtPayload(token) : null;
    const warehouseId = (claims?.warehouse_id as string) || '';

    try {
      const res = await apiFetch('/v1/auth/warehouse/register', {
        method: 'POST',
        body: JSON.stringify({
          warehouse_id: warehouseId,
          name: formName,
          phone: formPhone,
          pin: formPin,
          role: formRole,
        }),
      });

      if (res.ok) {
        toast('Staff member registered', 'success');
        setShowForm(false);
        setFormName('');
        setFormPhone('');
        setFormPin('');
        loadStaff();
      } else {
        const data = await res.json().catch(() => ({}));
        toast(data.error || 'Failed to register', 'error');
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
          Register Staff
        </button>
      </div>

      {/* Registration form */}
      {showForm && (
        <form
          onSubmit={handleRegister}
          className="rounded-xl border border-[var(--border)] p-6 space-y-4"
          style={{ background: 'var(--surface)' }}
        >
          <h2 className="text-sm font-semibold">Register New Staff Member</h2>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-medium mb-1.5 text-[var(--muted)]">Name</label>
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
              <label className="block text-xs font-medium mb-1.5 text-[var(--muted)]">Phone</label>
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
              <label className="block text-xs font-medium mb-1.5 text-[var(--muted)]">PIN (6+ digits)</label>
              <input
                type="password"
                value={formPin}
                onChange={e => setFormPin(e.target.value)}
                required
                minLength={6}
                className="w-full px-3 py-2.5 rounded-lg border text-sm outline-none"
                style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)' }}
              />
            </div>
            <div>
              <label className="block text-xs font-medium mb-1.5 text-[var(--muted)]">Role</label>
              <select
                value={formRole}
                onChange={e => setFormRole(e.target.value)}
                className="w-full px-3 py-2.5 rounded-lg border text-sm outline-none"
                style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)' }}
              >
                <option value="WAREHOUSE_ADMIN">Warehouse Admin</option>
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
              {submitting ? 'Registering...' : 'Register'}
            </button>
            <button
              type="button"
              onClick={() => setShowForm(false)}
              className="px-4 py-2 rounded-lg text-sm button--secondary border border-[var(--border)]"
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
        <div className="text-center py-20 text-[var(--muted)]">
          <Icon name="staff" size={48} className="mx-auto mb-3 opacity-30" />
          <p className="text-sm">No staff members registered</p>
        </div>
      ) : (
        <div className="border border-[var(--border)] rounded-xl overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-[var(--border)]" style={{ background: 'var(--surface)' }}>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">Name</th>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">Phone</th>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">Role</th>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">Status</th>
              </tr>
            </thead>
            <tbody>
              {staff.map(s => (
                <tr key={s.worker_id} className="border-b border-[var(--border)] last:border-b-0">
                  <td className="px-4 py-3">{s.name}</td>
                  <td className="px-4 py-3 text-[var(--muted)]">{s.phone || '—'}</td>
                  <td className="px-4 py-3">
                    <span className="status-chip status-chip--submitted">{s.role}</span>
                  </td>
                  <td className="px-4 py-3">
                    <span className={`text-xs font-semibold ${s.is_active ? 'text-[var(--success)]' : 'text-[var(--danger)]'}`}>
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
