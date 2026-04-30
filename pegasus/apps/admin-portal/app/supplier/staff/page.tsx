'use client';

import { useEffect, useState, useMemo, useCallback, useRef } from 'react';
import { Button } from '@heroui/react';
import { apiFetch } from '@/lib/auth';
import { useAuth } from '@/hooks/useAuth';
import { usePagination } from '@/lib/usePagination';
import PaginationControls from '@/components/PaginationControls';
import Icon from '@/components/Icon';
import Drawer from '@/components/Drawer';

/* ── Types ─────────────────────────────────────────────────────────────── */

type SupplierRole = 'GLOBAL_ADMIN' | 'NODE_ADMIN' | 'FACTORY_ADMIN' | 'FACTORY_PAYLOADER';

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

interface Warehouse { warehouse_id: string; name: string; }
interface Factory { factory_id: string; name: string; }

type RoleFilter = 'ALL' | SupplierRole;
type StatusFilter = 'ALL' | 'ACTIVE' | 'INACTIVE';

/* ── Helpers ───────────────────────────────────────────────────────────── */

const ROLE_META: Record<string, { label: string; bg: string; fg: string }> = {
  GLOBAL_ADMIN: {
    label: 'Global Admin',
    bg: 'var(--color-md-primary-container)',
    fg: 'var(--color-md-on-primary-container)',
  },
  NODE_ADMIN: {
    label: 'Node Admin',
    bg: 'var(--color-md-tertiary-container)',
    fg: 'var(--color-md-on-tertiary-container)',
  },
  FACTORY_ADMIN: {
    label: 'Factory Admin',
    bg: 'var(--color-md-secondary-container)',
    fg: 'var(--color-md-on-secondary-container)',
  },
  FACTORY_PAYLOADER: {
    label: 'Payloader',
    bg: 'var(--color-md-secondary-container)',
    fg: 'var(--color-md-on-secondary-container)',
  },
};

function roleMeta(role: string) {
  return ROLE_META[role] ?? { label: role, bg: 'var(--color-md-surface-container)', fg: 'var(--foreground)' };
}

/* ── Main Page ─────────────────────────────────────────────────────────── */

export default function StaffManagementPage() {
  const { userId, isGlobalAdmin } = useAuth();

  const [members, setMembers] = useState<OrgMember[]>([]);
  const [warehouses, setWarehouses] = useState<Warehouse[]>([]);
  const [factories, setFactories] = useState<Factory[]>([]);
  const [loading, setLoading] = useState(true);

  // Filters
  const [search, setSearch] = useState('');
  const [roleFilter, setRoleFilter] = useState<RoleFilter>('ALL');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('ALL');

  // Invite drawer
  const [showInvite, setShowInvite] = useState(false);
  const [formName, setFormName] = useState('');
  const [formEmail, setFormEmail] = useState('');
  const [formPhone, setFormPhone] = useState('');
  const [formPassword, setFormPassword] = useState('');
  const [formRole, setFormRole] = useState<SupplierRole>('NODE_ADMIN');
  const [formWarehouse, setFormWarehouse] = useState('');
  const [formFactory, setFormFactory] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState('');

  // Edit drawer
  const [editTarget, setEditTarget] = useState<OrgMember | null>(null);
  const [editRole, setEditRole] = useState<SupplierRole>('NODE_ADMIN');
  const [editWarehouse, setEditWarehouse] = useState('');
  const [editFactory, setEditFactory] = useState('');
  const [editName, setEditName] = useState('');
  const [editSaving, setEditSaving] = useState(false);
  const [editError, setEditError] = useState('');

  const isFactoryFormRole = formRole === 'FACTORY_ADMIN' || formRole === 'FACTORY_PAYLOADER';
  const isFactoryEditRole = editRole === 'FACTORY_ADMIN' || editRole === 'FACTORY_PAYLOADER';

  // Debounce search
  const searchTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const [debouncedSearch, setDebouncedSearch] = useState('');
  useEffect(() => {
    searchTimer.current = setTimeout(() => setDebouncedSearch(search), 250);
    return () => clearTimeout(searchTimer.current);
  }, [search]);

  /* ── Data Fetching ── */

  const fetchMembers = useCallback(async () => {
    setLoading(true);
    try {
      const res = await apiFetch('/v1/supplier/org/members');
      if (res.ok) {
        const data = await res.json();
        setMembers(data.data || []);
      }
    } catch { /* handled */ } finally { setLoading(false); }
  }, []);

  const fetchWarehouses = useCallback(async () => {
    try {
      const res = await apiFetch('/v1/supplier/warehouses');
      if (res.ok) {
        const data = await res.json();
        setWarehouses(Array.isArray(data) ? data : data.data || []);
      }
    } catch { /* handled */ }
  }, []);

  const fetchFactories = useCallback(async () => {
    try {
      const res = await apiFetch('/v1/supplier/factories');
      if (res.ok) {
        const data = await res.json();
        setFactories(Array.isArray(data) ? data : data.data || []);
      }
    } catch { /* handled */ }
  }, []);

  useEffect(() => { fetchMembers(); fetchWarehouses(); fetchFactories(); }, [fetchMembers, fetchWarehouses, fetchFactories]);

  /* ── Filtered + Sorted ── */

  const filtered = useMemo(() => {
    let list = members;
    if (roleFilter !== 'ALL') list = list.filter(m => m.supplier_role === roleFilter);
    if (statusFilter === 'ACTIVE') list = list.filter(m => m.is_active);
    if (statusFilter === 'INACTIVE') list = list.filter(m => !m.is_active);
    if (debouncedSearch) {
      const q = debouncedSearch.toLowerCase();
      list = list.filter(m =>
        m.name.toLowerCase().includes(q) ||
        m.email.toLowerCase().includes(q) ||
        m.phone.includes(q)
      );
    }
    return list;
  }, [members, roleFilter, statusFilter, debouncedSearch]);

  const pagination = usePagination(filtered, 25);

  /* ── Node Name Resolvers ── */

  function warehouseName(id: string) {
    return warehouses.find(w => w.warehouse_id === id)?.name || id || '—';
  }
  function factoryName(id: string) {
    return factories.find(f => f.factory_id === id)?.name || id || '—';
  }
  function assignmentLabel(m: OrgMember) {
    if (m.assigned_warehouse_id) return warehouseName(m.assigned_warehouse_id);
    if (m.assigned_factory_id) return factoryName(m.assigned_factory_id);
    return '—';
  }
  function assignmentType(m: OrgMember) {
    if (m.assigned_warehouse_id) return 'Warehouse';
    if (m.assigned_factory_id) return 'Factory';
    return '';
  }

  /* ── KPIs ── */

  const kpi = useMemo(() => ({
    total: members.length,
    active: members.filter(m => m.is_active).length,
    inactive: members.filter(m => !m.is_active).length,
    global: members.filter(m => m.supplier_role === 'GLOBAL_ADMIN').length,
    node: members.filter(m => m.supplier_role === 'NODE_ADMIN').length,
    factory: members.filter(m => m.supplier_role === 'FACTORY_ADMIN' || m.supplier_role === 'FACTORY_PAYLOADER').length,
  }), [members]);

  /* ── Invite ── */

  function resetInviteForm() {
    setFormName(''); setFormEmail(''); setFormPhone(''); setFormPassword('');
    setFormRole('NODE_ADMIN'); setFormWarehouse(''); setFormFactory(''); setFormError('');
  }

  async function handleInvite(e: React.FormEvent) {
    e.preventDefault();
    setFormError('');
    if (!formName.trim() || !formPassword) { setFormError('Name and password are required.'); return; }
    if (!formEmail && !formPhone) { setFormError('Email or phone is required.'); return; }
    if (formPassword.length < 8) { setFormError('Password must be at least 8 characters.'); return; }
    if (formRole === 'NODE_ADMIN' && !formWarehouse) { setFormError('Assigned warehouse is required for Node Admin.'); return; }
    if (isFactoryFormRole && !formFactory) { setFormError('Assigned factory is required for factory roles.'); return; }

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
          assigned_factory_id: isFactoryFormRole ? formFactory : '',
        }),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: 'Invite failed' }));
        setFormError(err.error || 'Invite failed');
        return;
      }
      setShowInvite(false);
      resetInviteForm();
      fetchMembers();
    } catch { setFormError('Network error'); } finally { setSubmitting(false); }
  }

  /* ── Toggle Active ── */

  async function toggleActive(m: OrgMember) {
    if (m.user_id === userId) return; // Self-protection
    if (m.is_active) {
      const res = await apiFetch(`/v1/supplier/org/members/${m.user_id}`, { method: 'DELETE' });
      if (res.ok) fetchMembers();
      else {
        const err = await res.json().catch(() => ({ error: 'Failed' }));
        alert(err.error || 'Failed to deactivate');
      }
    } else {
      const res = await apiFetch(`/v1/supplier/org/members/${m.user_id}`, {
        method: 'PUT',
        body: JSON.stringify({ is_active: true }),
      });
      if (res.ok) fetchMembers();
    }
  }

  /* ── Edit Member ── */

  function openEdit(m: OrgMember) {
    setEditTarget(m);
    setEditRole(m.supplier_role as SupplierRole);
    setEditWarehouse(m.assigned_warehouse_id);
    setEditFactory(m.assigned_factory_id);
    setEditName(m.name);
    setEditError('');
  }

  async function handleEditSave(e: React.FormEvent) {
    e.preventDefault();
    if (!editTarget) return;
    setEditError('');

    const isSelf = editTarget.user_id === userId;
    if (isSelf && editRole !== 'GLOBAL_ADMIN') {
      setEditError('Cannot demote your own account.');
      return;
    }

    const body: Record<string, unknown> = {};
    if (editName.trim() !== editTarget.name) body.name = editName.trim();
    if (editRole !== editTarget.supplier_role) body.supplier_role = editRole;
    if (editRole === 'NODE_ADMIN' && editWarehouse !== editTarget.assigned_warehouse_id) {
      body.assigned_warehouse_id = editWarehouse;
    }
    if ((editRole === 'FACTORY_ADMIN' || editRole === 'FACTORY_PAYLOADER') && editFactory !== editTarget.assigned_factory_id) {
      body.assigned_factory_id = editFactory;
    }

    if (Object.keys(body).length === 0) { setEditTarget(null); return; }

    setEditSaving(true);
    try {
      const res = await apiFetch(`/v1/supplier/org/members/${editTarget.user_id}`, {
        method: 'PUT',
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: 'Update failed' }));
        setEditError(err.error || 'Update failed');
        return;
      }
      setEditTarget(null);
      fetchMembers();
    } catch { setEditError('Network error'); } finally { setEditSaving(false); }
  }

  /* ── Guard: Global Admin only ── */

  if (!isGlobalAdmin) {
    return (
      <div className="flex-1 flex items-center justify-center p-8" style={{ background: 'var(--background)' }}>
        <div className="text-center">
          <Icon name="warning" size={48} style={{ color: 'var(--color-md-error)' }} />
          <h2 className="md-typescale-headline-small text-foreground mt-4">Access Restricted</h2>
          <p className="md-typescale-body-medium text-muted mt-2">Staff management requires Global Admin privileges.</p>
        </div>
      </div>
    );
  }

  /* ── Render ── */

  return (
    <div className="flex-1 overflow-y-auto p-6 md:p-8" style={{ background: 'var(--background)' }}>
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
        <div>
          <h1 className="md-typescale-headline-small text-foreground">Staff Management</h1>
          <p className="md-typescale-body-medium text-muted mt-1">
            Manage organization members, assign roles and node scopes.
          </p>
        </div>
        <Button className="md-btn md-btn-filled" onPress={() => { resetInviteForm(); setShowInvite(true); }}>
          <Icon name="add" size={18} />
          <span>Add Member</span>
        </Button>
      </div>

      {/* KPI Row */}
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3 mb-6">
        {[
          { label: 'Total', value: kpi.total, color: 'var(--foreground)' },
          { label: 'Active', value: kpi.active, color: 'var(--color-md-success)' },
          { label: 'Inactive', value: kpi.inactive, color: 'var(--color-md-error)' },
          { label: 'Global', value: kpi.global, color: 'var(--color-md-primary)' },
          { label: 'Node', value: kpi.node, color: 'var(--color-md-tertiary)' },
          { label: 'Factory', value: kpi.factory, color: 'var(--color-md-secondary)' },
        ].map(k => (
          <div key={k.label} className="md-card md-elevation-0 p-3" style={{ border: '1px solid var(--border)' }}>
            <p className="md-typescale-label-small text-muted">{k.label}</p>
            <p className="md-typescale-headline-small" style={{ color: k.color }}>{k.value}</p>
          </div>
        ))}
      </div>

      {/* Filters */}
      <div className="flex flex-col sm:flex-row gap-3 mb-4">
        <div className="relative flex-1">
          <Icon name="search" size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted" />
          <input
            className="md-input-outlined w-full pl-9"
            placeholder="Search by name, email, or phone..."
            value={search}
            onChange={e => setSearch(e.target.value)}
          />
        </div>
        <select className="md-input-outlined" value={roleFilter}
          onChange={e => setRoleFilter(e.target.value as RoleFilter)}>
          <option value="ALL">All Roles</option>
          <option value="GLOBAL_ADMIN">Global Admin</option>
          <option value="NODE_ADMIN">Node Admin</option>
          <option value="FACTORY_ADMIN">Factory Admin</option>
          <option value="FACTORY_PAYLOADER">Payloader</option>
        </select>
        <select className="md-input-outlined" value={statusFilter}
          onChange={e => setStatusFilter(e.target.value as StatusFilter)}>
          <option value="ALL">All Status</option>
          <option value="ACTIVE">Active</option>
          <option value="INACTIVE">Inactive</option>
        </select>
      </div>

      {/* Table */}
      {loading ? (
        <div className="flex items-center justify-center py-20 text-muted">
          <div className="w-8 h-8 border-3 rounded-full animate-spin"
            style={{ borderColor: 'var(--border)', borderTopColor: 'var(--color-md-primary)' }} />
          <span className="ml-3 md-typescale-body-medium">Loading staff...</span>
        </div>
      ) : filtered.length === 0 ? (
        <div className="text-center py-20">
          <Icon name="supplier" size={40} style={{ color: 'var(--muted)' }} />
          <p className="md-typescale-body-large text-muted mt-3">
            {members.length === 0 ? 'No organization members yet.' : 'No members match your filters.'}
          </p>
          {members.length === 0 && (
            <p className="md-typescale-body-medium text-muted mt-1">Add your first team member to get started.</p>
          )}
        </div>
      ) : (
        <div className="md-card md-elevation-0 overflow-hidden" style={{ border: '1px solid var(--border)' }}>
          <div className="overflow-x-auto">
            <table className="w-full min-w-[800px]">
              <thead>
                <tr style={{ borderBottom: '1px solid var(--border)' }}>
                  <th className="text-left md-typescale-label-large px-4 py-3 text-muted">Member</th>
                  <th className="text-left md-typescale-label-large px-4 py-3 text-muted">Contact</th>
                  <th className="text-left md-typescale-label-large px-4 py-3 text-muted">Role</th>
                  <th className="text-left md-typescale-label-large px-4 py-3 text-muted">Assignment</th>
                  <th className="text-left md-typescale-label-large px-4 py-3 text-muted">Status</th>
                  <th className="text-left md-typescale-label-large px-4 py-3 text-muted">Joined</th>
                  <th className="text-right md-typescale-label-large px-4 py-3 text-muted">Actions</th>
                </tr>
              </thead>
              <tbody>
                {pagination.pageItems.map(m => {
                  const isSelf = m.user_id === userId;
                  const isRoot = m.user_id === m.supplier_id;
                  const rm = roleMeta(m.supplier_role);
                  return (
                    <tr key={m.user_id}
                      style={{ borderBottom: '1px solid var(--border)' }}
                      className={!m.is_active ? 'opacity-50' : ''}>
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-2">
                          <p className="md-typescale-body-medium text-foreground">{m.name}</p>
                          {isSelf && (
                            <span className="md-typescale-label-small px-1.5 py-0.5 md-shape-sm"
                              style={{ background: 'var(--color-md-primary)', color: 'var(--color-md-on-primary)', fontSize: '10px' }}>
                              YOU
                            </span>
                          )}
                          {isRoot && (
                            <span className="md-typescale-label-small px-1.5 py-0.5 md-shape-sm"
                              style={{ background: 'var(--color-md-error-container)', color: 'var(--color-md-on-error-container)', fontSize: '10px' }}>
                              FOUNDER
                            </span>
                          )}
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        <p className="md-typescale-body-small text-foreground">{m.email || '—'}</p>
                        <p className="md-typescale-body-small text-muted">{m.phone || '—'}</p>
                      </td>
                      <td className="px-4 py-3">
                        <span className="md-typescale-label-medium px-2 py-0.5 md-shape-sm inline-block"
                          style={{ background: rm.bg, color: rm.fg }}>
                          {rm.label}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <p className="md-typescale-body-small text-foreground">{assignmentLabel(m)}</p>
                        {assignmentType(m) && (
                          <p className="md-typescale-label-small text-muted">{assignmentType(m)}</p>
                        )}
                      </td>
                      <td className="px-4 py-3">
                        <span className="inline-flex items-center gap-1.5">
                          <span className={`w-2 h-2 rounded-full ${m.is_active ? 'bg-success' : 'bg-muted'}`} />
                          <span className="md-typescale-body-small">{m.is_active ? 'Active' : 'Inactive'}</span>
                        </span>
                      </td>
                      <td className="px-4 py-3 md-typescale-body-small text-muted">
                        {m.created_at ? new Date(m.created_at).toLocaleDateString() : '—'}
                      </td>
                      <td className="px-4 py-3 text-right">
                        <div className="inline-flex gap-1">
                          <Button variant="ghost" isIconOnly className="w-8 h-8 min-w-0"
                            onPress={() => openEdit(m)} aria-label="Edit member">
                            <Icon name="edit" size={16} />
                          </Button>
                          {!isSelf && !isRoot && (
                            <Button variant="ghost" isIconOnly className="w-8 h-8 min-w-0"
                              onPress={() => toggleActive(m)}
                              aria-label={m.is_active ? 'Deactivate' : 'Reactivate'}>
                              <Icon name={m.is_active ? 'close' : 'refresh'} size={16} />
                            </Button>
                          )}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
          <PaginationControls pagination={pagination} />
        </div>
      )}

      {/* ── Invite Drawer ── */}
      <Drawer open={showInvite} onClose={() => setShowInvite(false)} title="Add Organization Member">
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
              onChange={e => setFormRole(e.target.value as SupplierRole)}>
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

          {isFactoryFormRole && (
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
              {submitting ? 'Creating...' : 'Create Member'}
            </Button>
            <Button className="md-btn md-btn-outlined" variant="ghost" onPress={() => setShowInvite(false)}>
              Cancel
            </Button>
          </div>
        </form>
      </Drawer>

      {/* ── Edit Drawer ── */}
      <Drawer open={!!editTarget} onClose={() => setEditTarget(null)} title="Edit Member">
        {editTarget && (
          <form onSubmit={handleEditSave} className="flex flex-col gap-4 p-4">
            {editError && (
              <div className="md-typescale-body-small px-3 py-2 md-shape-sm"
                style={{ background: 'var(--color-md-error-container)', color: 'var(--color-md-on-error-container)' }}>
                {editError}
              </div>
            )}

            {editTarget.user_id === editTarget.supplier_id && (
              <div className="md-typescale-body-small px-3 py-2 md-shape-sm"
                style={{ background: 'var(--color-md-error-container)', color: 'var(--color-md-on-error-container)' }}>
                <Icon name="warning" size={14} className="inline-block align-text-bottom mr-1" />
                Founder account — role and status are locked.
              </div>
            )}

            {editTarget.user_id === userId && (
              <div className="md-typescale-body-small px-3 py-2 md-shape-sm"
                style={{ background: 'var(--color-md-warning-container, var(--color-md-tertiary-container))', color: 'var(--color-md-on-warning-container, var(--color-md-on-tertiary-container))' }}>
                <Icon name="warning" size={14} className="inline-block align-text-bottom mr-1" />
                This is your account — role cannot be changed.
              </div>
            )}

            <label className="flex flex-col gap-1">
              <span className="md-typescale-label-medium text-foreground">Name</span>
              <input className="md-input-outlined" value={editName}
                onChange={e => setEditName(e.target.value)} />
            </label>

            <label className="flex flex-col gap-1">
              <span className="md-typescale-label-medium text-foreground">Role</span>
              <select className="md-input-outlined" value={editRole}
                onChange={e => setEditRole(e.target.value as SupplierRole)}
                disabled={editTarget.user_id === userId || editTarget.user_id === editTarget.supplier_id}>
                <option value="GLOBAL_ADMIN">Global Admin (Full access)</option>
                <option value="NODE_ADMIN">Node Admin (Warehouse-scoped)</option>
                <option value="FACTORY_ADMIN">Factory Admin (Factory-scoped)</option>
                <option value="FACTORY_PAYLOADER">Factory Payloader (Loading/Offloading)</option>
              </select>
            </label>

            {editRole === 'NODE_ADMIN' && (
              <label className="flex flex-col gap-1">
                <span className="md-typescale-label-medium text-foreground">Assigned Warehouse</span>
                <select className="md-input-outlined" value={editWarehouse}
                  onChange={e => setEditWarehouse(e.target.value)}>
                  <option value="">Select warehouse...</option>
                  {warehouses.map(w => (
                    <option key={w.warehouse_id} value={w.warehouse_id}>{w.name}</option>
                  ))}
                </select>
              </label>
            )}

            {isFactoryEditRole && (
              <label className="flex flex-col gap-1">
                <span className="md-typescale-label-medium text-foreground">Assigned Factory</span>
                <select className="md-input-outlined" value={editFactory}
                  onChange={e => setEditFactory(e.target.value)}>
                  <option value="">Select factory...</option>
                  {factories.map(f => (
                    <option key={f.factory_id} value={f.factory_id}>{f.name}</option>
                  ))}
                </select>
              </label>
            )}

            {editRole !== editTarget.supplier_role && editRole !== 'GLOBAL_ADMIN' && (
              <div className="md-typescale-body-small px-3 py-2 md-shape-sm"
                style={{ background: 'var(--color-md-surface-container)', color: 'var(--foreground)' }}>
                <Icon name="warning" size={14} className="inline-block align-text-bottom mr-1" />
                Role change takes effect on next login (max 1h TTL).
              </div>
            )}

            <div className="flex gap-3 pt-2">
              <Button className="md-btn md-btn-filled flex-1" type="submit" isDisabled={editSaving}>
                {editSaving ? 'Saving...' : 'Save Changes'}
              </Button>
              <Button className="md-btn md-btn-outlined" variant="ghost" onPress={() => setEditTarget(null)}>
                Cancel
              </Button>
            </div>
          </form>
        )}
      </Drawer>
    </div>
  );
}
