'use client';

import { useEffect, useState } from 'react';
import { Button } from '@heroui/react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';
import Drawer from '@/components/Drawer';
import { normalizeCollectionResponse } from '../_shared/referenceData';

/* ── Types ─────────────────────────────────────────────────────────────── */

interface OrgMember {
  user_id: string;
  supplier_id: string;
  name: string;
  email: string;
  phone: string;
  supplier_role: string;
  assigned_warehouse_id: string;
  assigned_factory_id: string;
  is_active: boolean;
  created_at: string;
}

interface Warehouse {
  warehouse_id: string;
  name: string;
}

interface Factory {
  factory_id: string;
  name: string;
}

/* ── Main Page ─────────────────────────────────────────────────────────── */

export default function OrgMembersPage() {
  const [members, setMembers] = useState<OrgMember[]>([]);
  const [warehouses, setWarehouses] = useState<Warehouse[]>([]);
  const [factories, setFactories] = useState<Factory[]>([]);
  const [loading, setLoading] = useState(true);
  const [showInvite, setShowInvite] = useState(false);

  // Invite form state
  const [formName, setFormName] = useState('');
  const [formEmail, setFormEmail] = useState('');
  const [formPhone, setFormPhone] = useState('');
  const [formPassword, setFormPassword] = useState('');
  const [formRole, setFormRole] = useState<'GLOBAL_ADMIN' | 'NODE_ADMIN' | 'FACTORY_ADMIN' | 'FACTORY_PAYLOADER'>('NODE_ADMIN');
  const [formWarehouse, setFormWarehouse] = useState('');
  const [formFactory, setFormFactory] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState('');

  const isFactoryRole = formRole === 'FACTORY_ADMIN' || formRole === 'FACTORY_PAYLOADER';

  useEffect(() => {
    fetchMembers();
    fetchWarehouses();
    fetchFactories();
  }, []);

  async function fetchMembers() {
    setLoading(true);
    try {
      const res = await apiFetch('/v1/supplier/org/members');
      if (res.ok) {
        const data = await res.json();
        setMembers(data.data || []);
      }
    } catch {
      // handled
    } finally {
      setLoading(false);
    }
  }

  async function fetchWarehouses() {
    try {
      const res = await apiFetch('/v1/supplier/warehouses');
      if (res.ok) {
        const data = await res.json();
        setWarehouses(normalizeCollectionResponse<Warehouse>(data, ['warehouses', 'data']));
      }
    } catch {
      // handled
    }
  }

  async function fetchFactories() {
    try {
      const res = await apiFetch('/v1/supplier/factories');
      if (res.ok) {
        const data = await res.json();
        setFactories(normalizeCollectionResponse<Factory>(data, ['data', 'factories']));
      }
    } catch {
      // handled
    }
  }

  async function handleInvite(e: React.FormEvent) {
    e.preventDefault();
    setFormError('');
    if (!formName.trim() || !formPassword) {
      setFormError('Name and password are required.');
      return;
    }
    if (!formEmail && !formPhone) {
      setFormError('Email or phone is required.');
      return;
    }
    if (formPassword.length < 8) {
      setFormError('Password must be at least 8 characters.');
      return;
    }
    if (formRole === 'NODE_ADMIN' && !formWarehouse) {
      setFormError('Assigned warehouse is required for Node Admin.');
      return;
    }
    if (isFactoryRole && !formFactory) {
      setFormError('Assigned factory is required for factory roles.');
      return;
    }

    setSubmitting(true);
    try {
      const res = await apiFetch('/v1/supplier/org/members/invite', {
        method: 'POST',
        body: JSON.stringify({
          name: formName.trim(),
          email: formEmail.trim().toLowerCase(),
          phone: formPhone.trim(),
          password: formPassword,
          supplier_role: formRole,
          assigned_warehouse_id: formRole === 'NODE_ADMIN' ? formWarehouse : '',
          assigned_factory_id: isFactoryRole ? formFactory : '',
        }),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: 'Invite failed' }));
        setFormError(err.error || 'Invite failed');
        return;
      }
      setShowInvite(false);
      resetForm();
      fetchMembers();
    } catch {
      setFormError('Network error');
    } finally {
      setSubmitting(false);
    }
  }

  async function toggleActive(member: OrgMember) {
    if (member.is_active) {
      const res = await apiFetch(`/v1/supplier/org/members/${member.user_id}`, {
        method: 'DELETE',
      });
      if (res.ok) fetchMembers();
    } else {
      const res = await apiFetch(`/v1/supplier/org/members/${member.user_id}`, {
        method: 'PUT',
        body: JSON.stringify({ is_active: true }),
      });
      if (res.ok) fetchMembers();
    }
  }

  function resetForm() {
    setFormName('');
    setFormEmail('');
    setFormPhone('');
    setFormPassword('');
    setFormRole('NODE_ADMIN');
    setFormWarehouse('');
    setFormFactory('');
    setFormError('');
  }

  function warehouseName(id: string): string {
    return warehouses.find(w => w.warehouse_id === id)?.name || id || '—';
  }

  function factoryName(id: string): string {
    return factories.find(f => f.factory_id === id)?.name || id || '—';
  }

  function assignmentLabel(m: OrgMember): string {
    if (m.assigned_warehouse_id) return warehouseName(m.assigned_warehouse_id);
    if (m.assigned_factory_id) return factoryName(m.assigned_factory_id);
    return '—';
  }

  function roleLabel(role: string): string {
    switch (role) {
      case 'GLOBAL_ADMIN': return 'Global Admin';
      case 'NODE_ADMIN': return 'Node Admin';
      case 'FACTORY_ADMIN': return 'Factory Admin';
      case 'FACTORY_PAYLOADER': return 'Payloader';
      default: return role;
    }
  }

  function roleColor(role: string) {
    switch (role) {
      case 'GLOBAL_ADMIN': return { bg: 'var(--color-md-primary-container)', fg: 'var(--color-md-on-primary-container)' };
      case 'NODE_ADMIN': return { bg: 'var(--color-md-tertiary-container)', fg: 'var(--color-md-on-tertiary-container)' };
      case 'FACTORY_ADMIN':
      case 'FACTORY_PAYLOADER': return { bg: 'var(--color-md-secondary-container)', fg: 'var(--color-md-on-secondary-container)' };
      default: return { bg: 'var(--color-md-surface-container)', fg: 'var(--foreground)' };
    }
  }

  /* ── Render ── */

  return (
    <div className="p-6 max-w-6xl mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="md-typescale-headline-small text-foreground">Organization Members</h1>
          <p className="md-typescale-body-medium text-muted mt-1">
            Manage admin accounts and assign warehouse-scoped roles.
          </p>
        </div>
        <Button className="md-btn md-btn-filled" onPress={() => { resetForm(); setShowInvite(true); }}>
          <Icon name="add" size={18} />
          <span>Invite Member</span>
        </Button>
      </div>

      {/* Table */}
      {loading ? (
        <div className="flex items-center justify-center py-20 text-muted">
          <Icon name="refresh" size={20} />
          <span className="ml-2 md-typescale-body-medium">Loading members...</span>
        </div>
      ) : members.length === 0 ? (
        <div className="text-center py-20">
          <Icon name="supplier" size={40} />
          <p className="md-typescale-body-large text-muted mt-3">No organization members yet.</p>
          <p className="md-typescale-body-medium text-muted mt-1">Invite your first team member to get started.</p>
        </div>
      ) : (
        <div className="md-card md-elevation-0 overflow-hidden" style={{ border: '1px solid var(--border)' }}>
          <table className="w-full">
            <thead>
              <tr style={{ borderBottom: '1px solid var(--border)' }}>
                <th className="text-left md-typescale-label-large px-4 py-3 text-muted">Name</th>
                <th className="text-left md-typescale-label-large px-4 py-3 text-muted">Contact</th>
                <th className="text-left md-typescale-label-large px-4 py-3 text-muted">Role</th>
                <th className="text-left md-typescale-label-large px-4 py-3 text-muted">Assignment</th>
                <th className="text-left md-typescale-label-large px-4 py-3 text-muted">Status</th>
                <th className="text-right md-typescale-label-large px-4 py-3 text-muted">Actions</th>
              </tr>
            </thead>
            <tbody>
              {members.map(m => (
                <tr key={m.user_id} style={{ borderBottom: '1px solid var(--border)' }}
                  className={!m.is_active ? 'opacity-50' : ''}>
                  <td className="px-4 py-3">
                    <p className="md-typescale-body-medium text-foreground">{m.name}</p>
                  </td>
                  <td className="px-4 py-3">
                    <p className="md-typescale-body-small text-foreground">{m.email || '—'}</p>
                    <p className="md-typescale-body-small text-muted">{m.phone || '—'}</p>
                  </td>
                  <td className="px-4 py-3">
                    <span className="md-typescale-label-medium px-2 py-0.5 md-shape-sm inline-block"
                      style={{
                        background: roleColor(m.supplier_role).bg,
                        color: roleColor(m.supplier_role).fg,
                      }}>
                      {roleLabel(m.supplier_role)}
                    </span>
                  </td>
                  <td className="px-4 py-3 md-typescale-body-small text-foreground">
                    {assignmentLabel(m)}
                  </td>
                  <td className="px-4 py-3">
                    <span className={`w-2 h-2 rounded-full inline-block mr-2 ${m.is_active ? 'bg-success' : 'bg-muted'}`} />
                    <span className="md-typescale-body-small">{m.is_active ? 'Active' : 'Inactive'}</span>
                  </td>
                  <td className="px-4 py-3 text-right">
                    <Button
                      variant="ghost"
                      isIconOnly
                      className="w-8 h-8 min-w-0"
                      onPress={() => toggleActive(m)}
                      aria-label={m.is_active ? 'Deactivate' : 'Reactivate'}
                    >
                      <Icon name={m.is_active ? 'close' : 'refresh'} size={16} />
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Invite Drawer */}
      <Drawer open={showInvite} onClose={() => setShowInvite(false)} title="Invite Organization Member">
        <form onSubmit={handleInvite} className="flex flex-col gap-4 p-4">
          {formError && (
            <div className="md-typescale-body-small px-3 py-2 md-shape-sm"
              style={{ background: 'var(--color-md-error-container)', color: 'var(--color-md-on-error-container)' }}>
              {formError}
            </div>
          )}

          <label className="flex flex-col gap-1">
            <span className="md-typescale-label-medium text-foreground">Full Name</span>
            <input className="md-input-outlined" value={formName}
              onChange={e => setFormName(e.target.value)} placeholder="John Doe" required />
          </label>

          <label className="flex flex-col gap-1">
            <span className="md-typescale-label-medium text-foreground">Email</span>
            <input className="md-input-outlined" type="email" value={formEmail}
              onChange={e => setFormEmail(e.target.value)} placeholder="john@example.com" />
          </label>

          <label className="flex flex-col gap-1">
            <span className="md-typescale-label-medium text-foreground">Phone</span>
            <input className="md-input-outlined" type="tel" value={formPhone}
              onChange={e => setFormPhone(e.target.value)} placeholder="+998901234567" />
          </label>

          <label className="flex flex-col gap-1">
            <span className="md-typescale-label-medium text-foreground">Password</span>
            <input className="md-input-outlined" type="password" value={formPassword}
              onChange={e => setFormPassword(e.target.value)} placeholder="Minimum 8 characters" required />
          </label>

          <label className="flex flex-col gap-1">
            <span className="md-typescale-label-medium text-foreground">Role</span>
            <select className="md-input-outlined" value={formRole}
              onChange={e => setFormRole(e.target.value as typeof formRole)}>
              <option value="NODE_ADMIN">Node Admin (Warehouse-scoped)</option>
              <option value="GLOBAL_ADMIN">Global Admin (Full access)</option>
              <option value="FACTORY_ADMIN">Factory Admin (Factory-scoped)</option>
              <option value="FACTORY_PAYLOADER">Factory Payloader (Loading/Offloading)</option>
            </select>
          </label>

          {formRole === 'NODE_ADMIN' && (
            <label className="flex flex-col gap-1">
              <span className="md-typescale-label-medium text-foreground">Assigned Warehouse</span>
              <select className="md-input-outlined" value={formWarehouse}
                onChange={e => setFormWarehouse(e.target.value)} required>
                <option value="">Select warehouse...</option>
                {warehouses.map(w => (
                  <option key={w.warehouse_id} value={w.warehouse_id}>{w.name}</option>
                ))}
              </select>
            </label>
          )}

          {isFactoryRole && (
            <label className="flex flex-col gap-1">
              <span className="md-typescale-label-medium text-foreground">Assigned Factory</span>
              <select className="md-input-outlined" value={formFactory}
                onChange={e => setFormFactory(e.target.value)} required>
                <option value="">Select factory...</option>
                {factories.map(f => (
                  <option key={f.factory_id} value={f.factory_id}>{f.name}</option>
                ))}
              </select>
            </label>
          )}

          <div className="flex gap-3 pt-2">
            <Button className="md-btn md-btn-filled flex-1" type="submit" isDisabled={submitting}>
              {submitting ? 'Inviting...' : 'Send Invite'}
            </Button>
            <Button className="md-btn md-btn-outlined" variant="ghost" onPress={() => setShowInvite(false)}>
              Cancel
            </Button>
          </div>
        </form>
      </Drawer>
    </div>
  );
}
