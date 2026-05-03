'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { Button } from '@heroui/react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';
import { buildSupplierWarehouseStaffCreateIdempotencyKey, buildSupplierWarehouseStaffToggleIdempotencyKey } from '../_shared/idempotency';

interface StaffMember {
  worker_id: string;
  name: string;
  phone: string;
  role: string;
  is_active: boolean;
  created_at: string;
}

const ROLE_OPTIONS = [
  { value: 'WAREHOUSE_ADMIN', label: 'Admin' },
  { value: 'WAREHOUSE_STAFF', label: 'Staff' },
  { value: 'PAYLOADER', label: 'Payloader' },
];

const fieldStyle = {
  background: 'var(--field-background)',
  color: 'var(--field-foreground)',
  border: '1px solid var(--field-border)',
  borderRadius: '8px',
};

export default function WarehouseStaffPanel({
  warehouseId,
  warehouseName,
}: {
  warehouseId: string;
  warehouseName: string;
}) {
  const [staff, setStaff] = useState<StaffMember[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showCreate, setShowCreate] = useState(false);
  const [createdResult, setCreatedResult] = useState<{ name: string; pin: string } | null>(null);

  // Create form state
  const [formName, setFormName] = useState('');
  const [formPhone, setFormPhone] = useState('');
  const [formPin, setFormPin] = useState('');
  const [formRole, setFormRole] = useState('WAREHOUSE_ADMIN');
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState('');

  const fetchStaff = useCallback(async () => {
    try {
      setLoading(true);
      const res = await apiFetch(`/v1/supplier/warehouse-staff?warehouse_id=${warehouseId}`);
      if (!res.ok) throw new Error('Failed to fetch staff');
      const data = await res.json();
      setStaff(data.staff || []);
      setError('');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  }, [warehouseId]);

  useEffect(() => {
    fetchStaff();
  }, [fetchStaff]);

  const handleCreate = async () => {
    if (!formName.trim() || !formPhone.trim() || formPin.length < 8) return;
    setCreating(true);
    setCreateError('');
    try {
      const payload = {
        warehouse_id: warehouseId,
        name: formName.trim(),
        phone: formPhone.trim(),
        pin: formPin,
        role: formRole,
      };
      const res = await apiFetch('/v1/auth/warehouse/register', {
        method: 'POST',
        headers: {
          'Idempotency-Key': buildSupplierWarehouseStaffCreateIdempotencyKey(payload),
        },
        body: JSON.stringify(payload),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to create staff');
      }
      setCreatedResult({ name: formName.trim(), pin: formPin });
      setFormName('');
      setFormPhone('');
      setFormPin('');
      setFormRole('WAREHOUSE_ADMIN');
      setShowCreate(false);
      fetchStaff();
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setCreating(false);
    }
  };

  const handleToggle = async (member: StaffMember) => {
    const payload = {
      warehouse_id: warehouseId,
      is_active: !member.is_active,
    };
    const res = await apiFetch(`/v1/supplier/warehouse-staff/${member.worker_id}`, {
      method: 'PATCH',
      headers: {
        'Idempotency-Key': buildSupplierWarehouseStaffToggleIdempotencyKey(member.worker_id, payload),
      },
      body: JSON.stringify(payload),
    });
    if (res.ok) fetchStaff();
  };

  return (
    <div className="p-6 space-y-5">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <p className="md-typescale-title-medium font-medium">Warehouse Staff</p>
          <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
            {warehouseName} — {staff.length} member{staff.length !== 1 ? 's' : ''}
          </p>
        </div>
        <Button
          size="sm"
          className="button--primary"
          onPress={() => {
            setShowCreate(!showCreate);
            setCreatedResult(null);
          }}
        >
          <Icon name="person-add" size={16} className="mr-1" />
          Add Staff
        </Button>
      </div>

      {/* Created confirmation */}
      {createdResult && (
        <div
          className="p-4 rounded-lg"
          style={{ background: 'color-mix(in srgb, var(--success) 12%, transparent)', border: '1px solid var(--success)' }}
        >
          <p className="md-typescale-label-medium font-medium" style={{ color: 'var(--success)' }}>
            Staff Created Successfully
          </p>
          <p className="md-typescale-body-small mt-1">
            <strong>{createdResult.name}</strong> has been added. Their one-time PIN is:
          </p>
          <p
            className="md-typescale-headline-small mt-2 font-mono tracking-widest text-center py-2 rounded"
            style={{ background: 'var(--surface)', border: '1px dashed var(--border)' }}
          >
            {createdResult.pin}
          </p>
          <p className="md-typescale-label-small mt-2" style={{ color: 'var(--muted)' }}>
            Save this PIN now. It cannot be recovered later.
          </p>
          <Button
            size="sm"
            variant="outline"
            className="mt-3"
            onPress={() => setCreatedResult(null)}
          >
            Dismiss
          </Button>
        </div>
      )}

      {/* Create form */}
      {showCreate && (
        <div
          className="p-4 rounded-lg space-y-4"
          style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}
        >
          <p className="md-typescale-label-medium font-medium">New Warehouse Staff</p>

          <div>
            <label className="md-typescale-label-small block mb-1" style={{ color: 'var(--muted)' }}>Name</label>
            <input
              className="w-full px-3 py-2.5 md-typescale-body-medium outline-none focus:ring-2 focus:ring-[var(--accent)]"
              style={fieldStyle}
              value={formName}
              onChange={e => setFormName(e.target.value)}
              placeholder="Full name"
            />
          </div>

          <div>
            <label className="md-typescale-label-small block mb-1" style={{ color: 'var(--muted)' }}>Phone</label>
            <input
              className="w-full px-3 py-2.5 md-typescale-body-medium outline-none focus:ring-2 focus:ring-[var(--accent)]"
              style={fieldStyle}
              value={formPhone}
              onChange={e => setFormPhone(e.target.value)}
              placeholder="+998..."
              type="tel"
            />
          </div>

          <div>
            <label className="md-typescale-label-small block mb-1" style={{ color: 'var(--muted)' }}>PIN (8 digits)</label>
            <input
              className="w-full px-3 py-2.5 md-typescale-body-medium outline-none focus:ring-2 focus:ring-[var(--accent)] font-mono tracking-widest"
              style={fieldStyle}
              value={formPin}
              onChange={e => setFormPin(e.target.value.replace(/\D/g, '').slice(0, 8))}
              placeholder="00000000"
              type="text"
              inputMode="numeric"
              maxLength={8}
            />
          </div>

          <div>
            <label className="md-typescale-label-small block mb-1" style={{ color: 'var(--muted)' }}>Role</label>
            <div className="flex gap-2">
              {ROLE_OPTIONS.map(opt => (
                <button
                  key={opt.value}
                  onClick={() => setFormRole(opt.value)}
                  className="px-3 py-1.5 rounded-full md-typescale-label-medium transition-colors"
                  style={{
                    background: formRole === opt.value ? 'var(--accent)' : 'transparent',
                    color: formRole === opt.value ? 'var(--on-accent, #fff)' : 'var(--foreground)',
                    border: `1px solid ${formRole === opt.value ? 'var(--accent)' : 'var(--border)'}`,
                  }}
                >
                  {opt.label}
                </button>
              ))}
            </div>
          </div>

          {createError && (
            <p className="md-typescale-label-small" style={{ color: 'var(--danger)' }}>{createError}</p>
          )}

          <div className="flex gap-2 pt-2">
            <Button
              className="button--primary flex-1"
              isPending={creating}
              isDisabled={!formName.trim() || !formPhone.trim() || formPin.length < 8}
              onPress={handleCreate}
            >
              Create Staff
            </Button>
            <Button variant="outline" onPress={() => setShowCreate(false)}>
              Cancel
            </Button>
          </div>
        </div>
      )}

      {/* Staff list */}
      {loading ? (
        <div className="flex items-center justify-center py-10">
          <div className="w-6 h-6 border-2 border-t-transparent rounded-full animate-spin" style={{ borderColor: 'var(--accent)', borderTopColor: 'transparent' }} />
        </div>
      ) : error ? (
        <div className="text-center py-6">
          <p className="md-typescale-body-small" style={{ color: 'var(--danger)' }}>{error}</p>
          <Button size="sm" variant="outline" className="mt-2" onPress={fetchStaff}>Retry</Button>
        </div>
      ) : staff.length === 0 ? (
        <div className="text-center py-10">
          <Icon name="person" size={32} className="mx-auto mb-2" style={{ color: 'var(--muted)' }} />
          <p className="md-typescale-body-medium" style={{ color: 'var(--muted)' }}>No staff assigned to this warehouse</p>
          <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Add a staff member to enable warehouse operations</p>
        </div>
      ) : (
        <div className="space-y-2">
          {staff.map((member, i) => (
            <div
              key={member.worker_id}
              className="flex items-center justify-between px-4 py-3 rounded-lg md-animate-in"
              style={{
                background: 'var(--surface)',
                border: '1px solid var(--border)',
                animationDelay: `${i * 30}ms`,
              }}
            >
              <div className="flex items-center gap-3">
                <div
                  className="w-8 h-8 rounded-full flex items-center justify-center md-typescale-label-medium"
                  style={{
                    background: member.is_active ? 'var(--accent-soft)' : 'color-mix(in srgb, var(--danger) 15%, transparent)',
                    color: member.is_active ? 'var(--accent)' : 'var(--danger)',
                  }}
                >
                  {member.name.charAt(0).toUpperCase()}
                </div>
                <div>
                  <p className="md-typescale-body-medium font-medium">{member.name}</p>
                  <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
                    {member.phone} · {member.role.replace('WAREHOUSE_', '').replace('_', ' ')}
                  </p>
                </div>
              </div>
              <div className="flex items-center gap-3">
                <span
                  className="md-typescale-label-small px-2 py-0.5 rounded-full"
                  style={{
                    background: member.is_active ? 'color-mix(in srgb, var(--success) 15%, transparent)' : 'color-mix(in srgb, var(--danger) 15%, transparent)',
                    color: member.is_active ? 'var(--success)' : 'var(--danger)',
                  }}
                >
                  {member.is_active ? 'Active' : 'Disabled'}
                </span>
                <button
                  onClick={() => handleToggle(member)}
                  className="w-8 h-5 rounded-full transition-colors relative"
                  style={{ background: member.is_active ? 'var(--success)' : 'var(--border)' }}
                >
                  <div
                    className="absolute top-0.5 w-4 h-4 rounded-full transition-transform"
                    style={{
                      background: 'white',
                      transform: member.is_active ? 'translateX(14px)' : 'translateX(2px)',
                    }}
                  />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
