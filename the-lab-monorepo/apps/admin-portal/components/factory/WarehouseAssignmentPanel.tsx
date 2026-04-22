'use client';

import React, { useState, useMemo, useCallback, useEffect } from 'react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';

// ── Types ─────────────────────────────────────────────────────────────────────

interface WarehouseRec {
  warehouse_id: string;
  name: string;
  address: string;
  lat: number;
  lng: number;
  distance_km: number;
  rank: number;
  is_assigned: boolean;
  assigned_to: string;
}

interface WarehouseAssignmentPanelProps {
  factoryLat: number;
  factoryLng: number;
  factoryId?: string;
  selected: string[];
  onChange: (ids: string[]) => void;
}

// ── Component ─────────────────────────────────────────────────────────────────

export default function WarehouseAssignmentPanel({
  factoryLat,
  factoryLng,
  factoryId,
  selected,
  onChange,
}: WarehouseAssignmentPanelProps) {
  const [recommendations, setRecommendations] = useState<WarehouseRec[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  // Fetch recommendations when factory location is available
  useEffect(() => {
    if (!factoryLat && !factoryLng) return;
    let cancelled = false;

    const load = async () => {
      setLoading(true);
      setError('');
      try {
        const params = new URLSearchParams({
          factory_lat: String(factoryLat),
          factory_lng: String(factoryLng),
        });
        if (factoryId) params.set('factory_id', factoryId);

        const res = await apiFetch(`/v1/supplier/factories/recommend-warehouses?${params}`);
        if (!res.ok) throw new Error('Failed to load recommendations');
        const data = await res.json();
        if (!cancelled) setRecommendations(data.recommendations ?? []);
      } catch {
        if (!cancelled) setError('Failed to load warehouse recommendations');
      } finally {
        if (!cancelled) setLoading(false);
      }
    };
    load();
    return () => { cancelled = true; };
  }, [factoryLat, factoryLng, factoryId]);

  const toggleWarehouse = useCallback((whId: string) => {
    onChange(
      selected.includes(whId)
        ? selected.filter(id => id !== whId)
        : [...selected, whId]
    );
  }, [selected, onChange]);

  // Smart assign — select top N nearest warehouses (without existing assignment to another factory)
  const handleSmartAssign = useCallback(() => {
    const available = recommendations.filter(r => !r.assigned_to || r.assigned_to === (factoryId ?? ''));
    // Select nearest warehouses — max 5 or all if fewer
    const smartPicks = available.slice(0, Math.min(5, available.length)).map(r => r.warehouse_id);
    onChange(smartPicks);
  }, [recommendations, factoryId, onChange]);

  const handleSelectAll = useCallback(() => {
    onChange(recommendations.map(r => r.warehouse_id));
  }, [recommendations, onChange]);

  const handleClearAll = useCallback(() => {
    onChange([]);
  }, [onChange]);

  const selectedCount = selected.length;
  const totalCount = recommendations.length;

  if (!factoryLat && !factoryLng) {
    return (
      <div className="flex flex-col items-center justify-center py-12 gap-3">
        <Icon name="pin" size={32} style={{ color: 'var(--muted)' }} />
        <p className="md-typescale-body-medium" style={{ color: 'var(--muted)' }}>
          Set the factory location first to see warehouse recommendations
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Actions row */}
      <div className="flex items-center justify-between">
        <p className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>
          {selectedCount} of {totalCount} warehouses selected
        </p>
        <div className="flex gap-2">
          <button
            type="button"
            onClick={handleSmartAssign}
            className="md-btn md-btn-tonal md-typescale-label-medium px-3 py-1.5 flex items-center gap-1.5"
          >
            <Icon name="hexagon" size={14} />
            Smart Assign
          </button>
          <button
            type="button"
            onClick={selectedCount === totalCount ? handleClearAll : handleSelectAll}
            className="md-btn md-btn-outlined md-typescale-label-medium px-3 py-1.5"
          >
            {selectedCount === totalCount ? 'Clear All' : 'Select All'}
          </button>
        </div>
      </div>

      {error && (
        <div className="flex items-center gap-2 p-3 rounded-lg" style={{ background: 'color-mix(in srgb, var(--danger) 10%, transparent)', color: 'var(--danger)' }}>
          <Icon name="error" size={14} />
          <span className="md-typescale-body-small">{error}</span>
        </div>
      )}

      {/* Warehouse list */}
      <div className="space-y-2 max-h-[360px] overflow-y-auto pr-1">
        {loading ? (
          Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="rounded-xl p-4 animate-pulse" style={{ background: 'var(--surface)', height: 72 }} />
          ))
        ) : recommendations.length === 0 ? (
          <div className="flex flex-col items-center py-8 gap-2">
            <Icon name="warehouse" size={28} style={{ color: 'var(--muted)' }} />
            <p className="md-typescale-body-medium" style={{ color: 'var(--muted)' }}>No warehouses configured yet</p>
          </div>
        ) : (
          recommendations.map(rec => {
            const isSelected = selected.includes(rec.warehouse_id);
            const isAssignedElsewhere = rec.assigned_to && rec.assigned_to !== (factoryId ?? '') && rec.assigned_to !== '';
            return (
              <button
                key={rec.warehouse_id}
                type="button"
                onClick={() => toggleWarehouse(rec.warehouse_id)}
                className="w-full text-left rounded-xl p-4 flex items-center gap-4 transition-all"
                style={{
                  background: isSelected ? 'color-mix(in srgb, var(--accent) 12%, var(--surface))' : 'var(--surface)',
                  border: isSelected ? '2px solid var(--accent)' : '2px solid var(--border)',
                  opacity: isAssignedElsewhere ? 0.6 : 1,
                }}
              >
                {/* Checkbox */}
                <div
                  className="flex items-center justify-center rounded-md shrink-0"
                  style={{
                    width: 22,
                    height: 22,
                    background: isSelected ? 'var(--accent)' : 'transparent',
                    border: isSelected ? 'none' : '2px solid var(--border)',
                  }}
                >
                  {isSelected && (
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="var(--accent-foreground)">
                      <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
                    </svg>
                  )}
                </div>

                {/* Info */}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <Icon name="warehouse" size={14} style={{ color: 'var(--muted)' }} />
                    <span className="md-typescale-body-medium font-medium truncate">{rec.name}</span>
                    {isAssignedElsewhere && (
                      <span
                        className="px-2 py-0.5 rounded text-[10px] font-medium shrink-0"
                        style={{ background: 'color-mix(in srgb, var(--warning) 15%, transparent)', color: 'var(--warning)' }}
                      >
                        Assigned to another factory
                      </span>
                    )}
                  </div>
                  {rec.address && (
                    <p className="md-typescale-body-small truncate mt-0.5" style={{ color: 'var(--muted)' }}>
                      {rec.address}
                    </p>
                  )}
                </div>

                {/* Distance badge */}
                <div className="shrink-0 text-right">
                  <span
                    className="inline-block px-2.5 py-1 rounded-lg text-xs font-semibold tabular-nums"
                    style={{
                      background: rec.distance_km < 50 ? 'color-mix(in srgb, var(--success) 15%, transparent)' : 'var(--surface)',
                      color: rec.distance_km < 50 ? 'var(--success)' : 'var(--muted)',
                    }}
                  >
                    {rec.distance_km.toFixed(1)} km
                  </span>
                  <p className="md-typescale-label-small mt-0.5" style={{ color: 'var(--muted)' }}>
                    #{rec.rank}
                  </p>
                </div>
              </button>
            );
          })
        )}
      </div>
    </div>
  );
}
