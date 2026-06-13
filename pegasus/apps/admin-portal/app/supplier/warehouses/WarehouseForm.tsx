'use client';

import React, { useState, useCallback } from 'react';
import { Button } from '@heroui/react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';
import dynamic from 'next/dynamic';
import { buildSupplierWarehouseCreateIdempotencyKey } from '../_shared/idempotency';

const MapLocationPicker = dynamic(() => import('@/components/MapLocationPicker'), { ssr: false });

interface Props {
  onSuccess: () => void;
  onCancel: () => void;
}

export default function WarehouseForm({ onSuccess, onCancel }: Props) {
  const [name, setName] = useState('');
  const [address, setAddress] = useState('');
  const [lat, setLat] = useState(0);
  const [lng, setLng] = useState(0);
  const [radius, setRadius] = useState('50');
  const [isDefault, setIsDefault] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const handleLocationChange = useCallback((newLat: number, newLng: number, newAddress: string) => {
    setLat(newLat);
    setLng(newLng);
    setAddress(newAddress);
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) { setError('Name is required'); return; }
    if (lat === 0 && lng === 0) { setError('Click the map to place the warehouse pin'); return; }

    setSaving(true);
    setError('');
    try {
      const payload = {
        name: name.trim(),
        address: address.trim(),
        lat,
        lng,
        coverage_radius_km: parseFloat(radius) || 50,
        is_default: isDefault,
      };
      const res = await apiFetch('/v1/supplier/warehouses', {
        method: 'POST',
        headers: {
          'Idempotency-Key': buildSupplierWarehouseCreateIdempotencyKey(payload),
        },
        body: JSON.stringify(payload),
      });
      if (!res.ok) {
        const data = await res.json();
        throw new Error(data.error || 'Create failed');
      }
      onSuccess();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setSaving(false);
    }
  };

  const fieldStyle = {
    background: 'var(--field-background)',
    color: 'var(--field-foreground)',
    border: '1px solid var(--field-border)',
    borderRadius: '8px',
  };

  return (
    <form onSubmit={handleSubmit} className="p-6 space-y-5">
      {error && (
        <div className="flex items-center gap-2 p-3 rounded-lg" style={{ background: 'color-mix(in srgb, var(--danger) 10%, transparent)', color: 'var(--danger)' }}>
          <Icon name="error" size={16} />
          <p className="md-typescale-body-small">{error}</p>
        </div>
      )}

      <div className="space-y-1.5">
        <label className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Name *</label>
        <input
          type="text"
          value={name}
          onChange={e => setName(e.target.value)}
          placeholder="Main Warehouse"
          className="w-full px-3 py-2.5 md-typescale-body-medium outline-none focus:ring-2 focus:ring-offset-0"
          style={{ ...fieldStyle, '--tw-ring-color': 'var(--accent)' } as React.CSSProperties}
        />
      </div>

      <div className="space-y-1.5">
        <label className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Location *</label>
        <MapLocationPicker
          latitude={lat}
          longitude={lng}
          addressText={address}
          onChange={handleLocationChange}
        />
      </div>

      <div className="space-y-1.5">
        <label className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Coverage Radius (km)</label>
        <input
          type="number"
          step="0.1"
          value={radius}
          onChange={e => setRadius(e.target.value)}
          placeholder="50"
          className="w-full px-3 py-2.5 md-typescale-body-medium outline-none focus:ring-2 focus:ring-offset-0"
          style={{ ...fieldStyle, '--tw-ring-color': 'var(--accent)' } as React.CSSProperties}
        />
      </div>

      <label className="flex items-center gap-3 cursor-pointer py-1">
        <div
          onClick={() => setIsDefault(!isDefault)}
          className="w-5 h-5 rounded flex items-center justify-center transition-colors"
          style={{
            background: isDefault ? 'var(--accent)' : 'transparent',
            border: isDefault ? 'none' : '2px solid var(--border)',
          }}
        >
          {isDefault && <Icon name="verified" size={14} className="text-white" />}
        </div>
        <span className="md-typescale-body-medium">Set as default warehouse</span>
      </label>

      <div className="flex gap-3 pt-4" style={{ borderTop: '1px solid var(--border)' }}>
        <Button type="submit" className="button--primary flex-1" isPending={saving}>
          Create Warehouse
        </Button>
        <Button type="button" variant="outline" className="flex-1" onPress={onCancel}>
          Cancel
        </Button>
      </div>
    </form>
  );
}
