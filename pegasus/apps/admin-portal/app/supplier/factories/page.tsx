'use client';

import React, { useEffect, useState, useCallback, lazy, Suspense } from 'react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';
import Drawer from '@/components/Drawer';
import StatsCard from '@/components/StatsCard';
import CreateFactoryWizard from '@/components/factory/CreateFactoryWizard';
import WarehouseAssignmentPanel from '@/components/factory/WarehouseAssignmentPanel';

const FactoryNetworkMap = lazy(() => import('@/components/factory/FactoryNetworkMap'));

type ViewTab = 'list' | 'network';

interface Factory {
  id: string;
  supplier_id: string;
  name: string;
  address: string;
  lat: number;
  lng: number;
  h3_index: string;
  product_types: string[];
  lead_time_days: number;
  production_capacity_vu: number;
  is_active: boolean;
  created_at: string;
}

export default function FactoriesPage() {
  const [factories, setFactories] = useState<Factory[]>([]);
  const [loading, setLoading] = useState(true);
  const [selected, setSelected] = useState<Factory | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [error, setError] = useState('');
  const [tab, setTab] = useState<ViewTab>('list');

  // Edit warehouse assignment state
  const [editingAssignment, setEditingAssignment] = useState(false);
  const [editWarehouses, setEditWarehouses] = useState<string[]>([]);
  const [savingAssignment, setSavingAssignment] = useState(false);

  const fetchFactories = useCallback(async () => {
    setLoading(true);
    try {
      const res = await apiFetch('/v1/supplier/factories');
      if (!res.ok) throw new Error('Failed to load');
      const data = await res.json();
      setFactories(data.factories ?? []);
    } catch {
      setError('Failed to load factories');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchFactories(); }, [fetchFactories]);

  const handleSaveAssignment = useCallback(async () => {
    if (!selected) return;
    setSavingAssignment(true);
    try {
      const res = await apiFetch(`/v1/supplier/factories/${selected.id}/warehouses`, {
        method: 'PUT',
        body: JSON.stringify({ warehouse_ids: editWarehouses }),
      });
      if (!res.ok) throw new Error('Failed to update');
      setEditingAssignment(false);
      fetchFactories();
    } catch {
      // error visible from network
    } finally {
      setSavingAssignment(false);
    }
  }, [selected, editWarehouses, fetchFactories]);

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="md-typescale-headline-medium">Factories</h1>
          <p className="md-typescale-body-medium" style={{ color: 'var(--muted)' }}>
            Production facilities linked to your warehouse grid
          </p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="md-btn md-btn-filled md-typescale-label-large px-5 py-2.5 flex items-center gap-2"
        >
          <Icon name="plus" size={16} />
          Add Factory
        </button>
      </div>

      {/* KPI row */}
      <div className="grid grid-cols-3 gap-4">
        <StatsCard label="Total Factories" value={String(factories.length)} sub="active facilities" delay={0} />
        <StatsCard label="Active" value={String(factories.filter(f => f.is_active).length)} sub="producing" accent="var(--success)" delay={80} />
        <StatsCard label="Product Types" value={String([...new Set(factories.flatMap(f => f.product_types))].length)} sub="categories" delay={160} />
      </div>

      {error && (
        <div className="flex items-center gap-2 p-3 rounded-lg" style={{ background: 'color-mix(in srgb, var(--danger) 10%, transparent)', color: 'var(--danger)' }}>
          <Icon name="error" size={16} />
          <span className="md-typescale-body-small">{error}</span>
        </div>
      )}

      {/* Tab switcher */}
      <div className="flex items-center gap-1 p-1 rounded-xl w-fit" style={{ background: 'var(--surface)' }}>
        {([
          { key: 'list' as const, label: 'List View', icon: 'orders' },
          { key: 'network' as const, label: 'Network Map', icon: 'global' },
        ]).map(t => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className="flex items-center gap-2 px-4 py-2 rounded-lg md-typescale-label-medium transition-all"
            style={{
              background: tab === t.key ? 'var(--background)' : 'transparent',
              color: tab === t.key ? 'var(--foreground)' : 'var(--muted)',
              boxShadow: tab === t.key ? '0 1px 3px rgba(0,0,0,0.2)' : 'none',
            }}
          >
            <Icon name={t.icon} size={14} />
            {t.label}
          </button>
        ))}
      </div>

      {/* Content based on tab */}
      {tab === 'list' ? (
        <div className="md-card md-card-elevated rounded-xl overflow-hidden">
          <table className="w-full">
            <thead>
              <tr style={{ background: 'var(--surface)', borderBottom: '1px solid var(--border)' }}>
                {['Name', 'Address', 'H3 Cell', 'Products', 'Status'].map(h => (
                  <th key={h} className="px-4 py-3 text-left md-typescale-label-medium" style={{ color: 'var(--muted)' }}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr><td colSpan={5} className="px-4 py-12 text-center md-typescale-body-medium" style={{ color: 'var(--muted)' }}>Loading...</td></tr>
              ) : factories.length === 0 ? (
                <tr><td colSpan={5} className="px-4 py-12 text-center">
                  <div className="space-y-2">
                    <Icon name="factory" size={32} className="mx-auto" style={{ color: 'var(--muted)' }} />
                    <p className="md-typescale-body-medium" style={{ color: 'var(--muted)' }}>No factories yet</p>
                    <button
                      onClick={() => setShowCreate(true)}
                      className="md-btn md-btn-tonal md-typescale-label-medium px-4 py-2 mt-2"
                    >
                      Add your first factory
                    </button>
                  </div>
                </td></tr>
              ) : factories.map(f => (
                <tr
                  key={f.id}
                  onClick={() => { setSelected(f); setEditingAssignment(false); }}
                  className="cursor-pointer transition-colors"
                  style={{ borderBottom: '1px solid var(--border)' }}
                  onMouseEnter={e => (e.currentTarget.style.background = 'var(--surface)')}
                  onMouseLeave={e => (e.currentTarget.style.background = '')}
                >
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <Icon name="factory" size={16} style={{ color: 'var(--accent)' }} />
                      <span className="md-typescale-body-medium font-medium">{f.name}</span>
                    </div>
                  </td>
                  <td className="px-4 py-3 md-typescale-body-small" style={{ color: 'var(--muted)' }}>{f.address || '—'}</td>
                  <td className="px-4 py-3 md-typescale-body-small font-mono" style={{ color: 'var(--muted)' }}>{f.h3_index?.slice(0, 12) || '—'}</td>
                  <td className="px-4 py-3">
                    <div className="flex gap-1 flex-wrap">
                      {(f.product_types ?? []).slice(0, 3).map(t => (
                        <span key={t} className="px-2 py-0.5 rounded text-xs" style={{ background: 'var(--surface)', color: 'var(--foreground)' }}>{t}</span>
                      ))}
                      {(f.product_types?.length ?? 0) > 3 && <span className="text-xs" style={{ color: 'var(--muted)' }}>+{f.product_types.length - 3}</span>}
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <span className="px-2 py-0.5 rounded text-xs font-medium" style={{
                      background: f.is_active ? 'color-mix(in srgb, var(--success) 15%, transparent)' : 'var(--surface)',
                      color: f.is_active ? 'var(--success)' : 'var(--muted)',
                    }}>
                      {f.is_active ? 'Active' : 'Inactive'}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <Suspense fallback={
          <div className="flex items-center justify-center py-20">
            <div className="w-6 h-6 rounded-full border-2 border-t-transparent animate-spin" style={{ borderColor: 'var(--accent)', borderTopColor: 'transparent' }} />
          </div>
        }>
          <FactoryNetworkMap factories={factories.map(f => ({
            factory_id: f.id,
            name: f.name,
            address: f.address,
            lat: f.lat,
            lng: f.lng,
            is_active: f.is_active,
          }))} />
        </Suspense>
      )}

      {/* Detail drawer */}
      <Drawer open={!!selected} onClose={() => { setSelected(null); setEditingAssignment(false); }} title={selected?.name ?? ''}>
        {selected && (
          <div className="space-y-5 p-5">
            {/* Mini map */}
            <div className="rounded-xl overflow-hidden" style={{ height: 160, background: 'var(--surface)' }}>
              <Suspense fallback={<div className="w-full h-full animate-pulse" style={{ background: 'var(--surface)' }} />}>
                <MiniFactoryMap lat={selected.lat} lng={selected.lng} />
              </Suspense>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <DetailField label="Address" value={selected.address || '—'} />
              <DetailField label="H3 Index" value={selected.h3_index?.slice(0, 12) || '—'} mono />
              <DetailField label="Coordinates" value={`${selected.lat.toFixed(5)}, ${selected.lng.toFixed(5)}`} />
              <DetailField label="Status" value={selected.is_active ? 'Active' : 'Inactive'} accent={selected.is_active ? 'var(--success)' : 'var(--muted)'} />
              <DetailField label="Lead Time" value={selected.lead_time_days ? `${selected.lead_time_days} days` : '—'} />
              <DetailField label="Capacity" value={selected.production_capacity_vu ? `${selected.production_capacity_vu.toLocaleString()} VU` : '—'} />
            </div>

            <div>
              <p className="md-typescale-label-small mb-2" style={{ color: 'var(--muted)' }}>Product Types</p>
              <div className="flex gap-2 flex-wrap">
                {(selected.product_types ?? []).map(t => (
                  <span key={t} className="px-3 py-1 rounded-lg text-sm" style={{ background: 'var(--surface)', color: 'var(--foreground)' }}>{t}</span>
                ))}
                {(!selected.product_types || selected.product_types.length === 0) && (
                  <span className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>No product types configured</span>
                )}
              </div>
            </div>

            {/* Warehouse assignment section */}
            <div style={{ borderTop: '1px solid var(--border)', paddingTop: 16 }}>
              <div className="flex items-center justify-between mb-3">
                <p className="md-typescale-title-small">Warehouse Assignments</p>
                {!editingAssignment ? (
                  <button
                    onClick={() => { setEditingAssignment(true); setEditWarehouses([]); }}
                    className="md-btn md-btn-tonal md-typescale-label-medium px-3 py-1.5 flex items-center gap-1.5"
                  >
                    <Icon name="edit" size={14} />
                    Edit
                  </button>
                ) : (
                  <div className="flex gap-2">
                    <button
                      onClick={() => setEditingAssignment(false)}
                      className="md-btn md-btn-outlined md-typescale-label-medium px-3 py-1.5"
                    >
                      Cancel
                    </button>
                    <button
                      onClick={handleSaveAssignment}
                      disabled={savingAssignment}
                      className="md-btn md-btn-filled md-typescale-label-medium px-3 py-1.5"
                    >
                      {savingAssignment ? 'Saving...' : 'Save'}
                    </button>
                  </div>
                )}
              </div>
              {editingAssignment && (
                <WarehouseAssignmentPanel
                  factoryLat={selected.lat}
                  factoryLng={selected.lng}
                  factoryId={selected.id}
                  selected={editWarehouses}
                  onChange={setEditWarehouses}
                />
              )}
            </div>
          </div>
        )}
      </Drawer>

      {/* Create drawer */}
      <Drawer open={showCreate} onClose={() => setShowCreate(false)} title="Add Factory">
        <CreateFactoryWizard
          onCreated={() => { setShowCreate(false); fetchFactories(); }}
          onCancel={() => setShowCreate(false)}
        />
      </Drawer>
    </div>
  );
}

// ── Helper components ─────────────────────────────────────────────────────

function DetailField({ label, value, mono, accent }: { label: string; value: string; mono?: boolean; accent?: string }) {
  return (
    <div>
      <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>{label}</p>
      <p className={`md-typescale-body-medium ${mono ? 'font-mono text-sm' : ''}`} style={accent ? { color: accent } : undefined}>
        {value}
      </p>
    </div>
  );
}

/** Small non-interactive map centered on factory coordinates */
function MiniFactoryMap({ lat, lng }: { lat: number; lng: number }) {
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const MapGL = require('react-map-gl/maplibre').default;
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const { Marker } = require('react-map-gl/maplibre');

  return (
    <MapGL
      initialViewState={{ latitude: lat, longitude: lng, zoom: 14 }}
      style={{ width: '100%', height: '100%' }}
      mapStyle="https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json"
      interactive={false}
    >
      <Marker latitude={lat} longitude={lng} anchor="bottom">
        <svg width="28" height="28" viewBox="0 0 24 24" fill="var(--accent)">
          <path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7zm0 9.5c-1.38 0-2.5-1.12-2.5-2.5s1.12-2.5 2.5-2.5 2.5 1.12 2.5 2.5-1.12 2.5-2.5 2.5z" />
        </svg>
      </Marker>
    </MapGL>
  );
}
