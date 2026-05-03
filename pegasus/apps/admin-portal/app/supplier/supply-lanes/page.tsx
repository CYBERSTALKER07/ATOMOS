'use client';

import { useCallback, useEffect, useState } from 'react';
import { apiFetch } from '@/lib/auth';

/* ── Types ──────────────────────────────────────────────────────────────────── */

interface SupplyLane {
  lane_id: string;
  supplier_id: string;
  factory_id: string;
  warehouse_id: string;
  transit_time_hours: number;
  dampened_transit_hours: number;
  freight_cost_minor: number;
  carbon_score_kg: number;
  direct_distance_km: number;
  is_active: boolean;
  priority: number;
}

interface SLAEvent {
  event_id: string;
  transfer_id: string;
  factory_id: string;
  warehouse_id: string;
  escalation_level: string;
  breach_minutes: number;
}

interface NetworkAnalytics {
  network_mode: string;
  supply_lanes: SupplyLane[] | null;
  sla_events: SLAEvent[] | null;
}

interface PullMatrixRun {
  run_id: string;
  run_at: string;
  transfers_generated: number;
  skus_processed: number;
  duration_ms: number;
  source: string;
}

/* ── Mode Badge Colors ─────────────────────────────────────────────────────── */

const MODE_COLORS: Record<string, { bg: string; text: string }> = {
  SPEED: { bg: 'var(--color-md-error-container, #ffdad6)', text: 'var(--color-md-on-error-container, #410002)' },
  ECONOMY: { bg: 'var(--color-md-tertiary-container, #ffd8e4)', text: 'var(--color-md-on-tertiary-container, #31111d)' },
  BALANCED: { bg: 'var(--color-md-primary-container, #d3e4ff)', text: 'var(--color-md-on-primary-container, #001c38)' },
  LOW_CARBON: { bg: 'var(--color-md-success-container, #c8f5c8)', text: 'var(--color-md-on-success-container, #002200)' },
  MANUAL_ONLY: { bg: 'var(--color-md-surface-variant, #e0e0e0)', text: 'var(--color-md-on-surface-variant, #444)' },
};

const ESCALATION_COLORS: Record<string, string> = {
  WARNING: 'var(--color-md-warning, #f9a825)',
  CRITICAL: 'var(--color-md-error, #ba1a1a)',
  AUTO_REROUTE: 'var(--color-md-info, #0288d1)',
  FORCE_RECEIVED: 'var(--color-md-tertiary, #7d5260)',
};

function buildSupplyLaneCreateIdempotencyKey(
  factoryId: string,
  warehouseId: string,
  transitTimeHours: string,
  freightCostMinor: string,
  priority: string,
): string {
  return ['supply-lane-create', factoryId.trim(), warehouseId.trim(), transitTimeHours.trim(), freightCostMinor.trim(), priority.trim()].join(':');
}

/* ── Page Component ────────────────────────────────────────────────────────── */

export default function SupplyLanesPage() {
  const [analytics, setAnalytics] = useState<NetworkAnalytics | null>(null);
  const [audit, setAudit] = useState<{ current_mode: string; pull_matrix_runs: PullMatrixRun[] | null } | null>(null);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState<'lanes' | 'sla' | 'audit'>('lanes');
  const [modeChanging, setModeChanging] = useState(false);
  const [creating, setCreating] = useState(false);
  const [createForm, setCreateForm] = useState({ factory_id: '', warehouse_id: '', transit_time_hours: '24', freight_cost_minor: '0', priority: '1' });

  const fetchData = useCallback(async () => {
    try {
      const [analyticsRes, auditRes] = await Promise.all([
        apiFetch('/v1/supplier/network-analytics'),
        apiFetch('/v1/supplier/replenishment/audit'),
      ]);
      if (analyticsRes.ok) setAnalytics(await analyticsRes.json());
      if (auditRes.ok) setAudit(await auditRes.json());
    } catch (e) {
      console.error('Failed to fetch supply lane data:', e);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  /* ── Mode Switcher ── */
  const changeMode = async (newMode: string) => {
    setModeChanging(true);
    try {
      const res = await apiFetch('/v1/supplier/network-mode', {
        method: 'PUT',
        body: JSON.stringify({ mode: newMode, reason: 'Admin portal mode change' }),
      });
      if (res.ok) fetchData();
    } finally {
      setModeChanging(false);
    }
  };

  /* ── Kill Switch ── */
  const triggerKillSwitch = async () => {
    if (!confirm('This will cancel ALL automated transfers and set mode to MANUAL_ONLY. Continue?')) return;
    const reason = prompt('Enter reason for kill switch activation:');
    if (!reason) return;
    await apiFetch('/v1/supplier/replenishment/kill-switch', {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
    fetchData();
  };

  /* ── Pull Matrix Manual Trigger ── */
  const triggerPullMatrix = async () => {
    await apiFetch('/v1/supplier/replenishment/pull-matrix', { method: 'POST' });
    fetchData();
  };

  /* ── Create Lane ── */
  const handleCreateLane = async () => {
    const body = {
      factory_id: createForm.factory_id,
      warehouse_id: createForm.warehouse_id,
      transit_time_hours: parseFloat(createForm.transit_time_hours),
      freight_cost_minor: parseInt(createForm.freight_cost_minor),
      carbon_score_kg: 0, // Auto-seeded by backend via Haversine (DirectDistanceKm × 0.1)
      priority: parseInt(createForm.priority),
    };
    const res = await apiFetch('/v1/supplier/supply-lanes', {
      method: 'POST',
      headers: {
        'Idempotency-Key': buildSupplyLaneCreateIdempotencyKey(
          createForm.factory_id,
          createForm.warehouse_id,
          createForm.transit_time_hours,
          createForm.freight_cost_minor,
          createForm.priority,
        ),
      },
      body: JSON.stringify(body),
    });
    if (res.ok) {
      setCreating(false);
      setCreateForm({ factory_id: '', warehouse_id: '', transit_time_hours: '24', freight_cost_minor: '0', priority: '1' });
      fetchData();
    }
  };

  /* ── Deactivate Lane ── */
  const deactivateLane = async (laneId: string) => {
    await apiFetch(`/v1/supplier/supply-lanes/${laneId}`, { method: 'DELETE' });
    fetchData();
  };

  if (loading) {
    return (
      <div className="p-6">
        <div className="md-typescale-headline-small" style={{ color: 'var(--foreground)' }}>Supply Lanes</div>
        <div className="mt-4" style={{ color: 'var(--muted)' }}>Loading network topology...</div>
      </div>
    );
  }

  const lanes = analytics?.supply_lanes ?? [];
  const slaEvents = analytics?.sla_events ?? [];
  const runs = audit?.pull_matrix_runs ?? [];
  const mode = analytics?.network_mode ?? 'BALANCED';
  const modeColor = MODE_COLORS[mode] ?? MODE_COLORS.BALANCED;

  return (
    <div className="p-6 space-y-6">
      {/* ── Header ── */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="md-typescale-headline-small" style={{ color: 'var(--foreground)' }}>
            Supply Lanes
          </h1>
          <p className="md-typescale-body-medium mt-1" style={{ color: 'var(--muted)' }}>
            Factory → Warehouse routing edges, network optimization mode, and SLA enforcement
          </p>
        </div>
        <div className="flex gap-2">
          <button className="md-btn md-btn-tonal md-typescale-label-large px-4 py-2" onClick={triggerPullMatrix}>
            Run Pull Matrix
          </button>
          <button
            className="md-btn md-btn-outlined md-typescale-label-large px-4 py-2"
            style={{ borderColor: 'var(--color-md-error)', color: 'var(--color-md-error)' }}
            onClick={triggerKillSwitch}
          >
            Kill Switch
          </button>
          <button className="md-btn md-btn-filled md-typescale-label-large px-4 py-2" onClick={() => setCreating(true)}>
            Add Lane
          </button>
        </div>
      </div>

      {/* ── KPI Row ── */}
      <div className="grid grid-cols-4 gap-4">
        <div className="md-card md-card-elevated md-shape-md p-4" style={{ background: 'var(--surface)' }}>
          <div className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Network Mode</div>
          <div className="mt-1 inline-block px-3 py-1 rounded-full md-typescale-label-large"
            style={{ background: modeColor.bg, color: modeColor.text }}>
            {mode}
          </div>
        </div>
        <div className="md-card md-card-elevated md-shape-md p-4" style={{ background: 'var(--surface)' }}>
          <div className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Active Lanes</div>
          <div className="md-typescale-headline-medium mt-1" style={{ color: 'var(--foreground)' }}>
            {lanes.filter(l => l.is_active).length}
          </div>
        </div>
        <div className="md-card md-card-elevated md-shape-md p-4" style={{ background: 'var(--surface)' }}>
          <div className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>SLA Events (Recent)</div>
          <div className="md-typescale-headline-medium mt-1" style={{ color: slaEvents.length > 0 ? 'var(--color-md-error)' : 'var(--foreground)' }}>
            {slaEvents.length}
          </div>
        </div>
        <div className="md-card md-card-elevated md-shape-md p-4" style={{ background: 'var(--surface)' }}>
          <div className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Pull Matrix Runs</div>
          <div className="md-typescale-headline-medium mt-1" style={{ color: 'var(--foreground)' }}>
            {runs.length}
          </div>
        </div>
      </div>

      {/* ── Mode Switcher ── */}
      <div className="md-card md-card-elevated md-shape-md p-4" style={{ background: 'var(--surface)' }}>
        <div className="md-typescale-title-medium mb-3" style={{ color: 'var(--foreground)' }}>Optimization Objective</div>
        <div className="flex gap-2 flex-wrap">
          {['SPEED', 'ECONOMY', 'BALANCED', 'LOW_CARBON', 'MANUAL_ONLY'].map(m => {
            const c = MODE_COLORS[m] ?? MODE_COLORS.BALANCED;
            return (
              <button
                key={m}
                disabled={modeChanging}
                className={`px-4 py-2 rounded-full md-typescale-label-large transition-opacity ${mode === m ? 'ring-2 ring-offset-2' : 'opacity-60 hover:opacity-100'}`}
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                style={{ background: c.bg, color: c.text, '--ringColor': c.text } as any}
                onClick={() => changeMode(m)}
              >
                {m.replace('_', ' ')}
              </button>
            );
          })}
        </div>
      </div>

      {/* ── Tabs ── */}
      <div className="flex gap-1 border-b" style={{ borderColor: 'var(--border)' }}>
        {(['lanes', 'sla', 'audit'] as const).map(t => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`px-4 py-2 md-typescale-label-large capitalize ${tab === t ? 'border-b-2' : ''}`}
            style={{
              color: tab === t ? 'var(--color-md-primary)' : 'var(--muted)',
              borderColor: tab === t ? 'var(--color-md-primary)' : 'transparent',
            }}
          >
            {t === 'sla' ? 'SLA Events' : t === 'audit' ? 'Pull Matrix Audit' : 'Lanes'}
          </button>
        ))}
      </div>

      {/* ── Tab Content ── */}
      {tab === 'lanes' && (
        <div className="md-card md-card-elevated md-shape-md overflow-hidden" style={{ background: 'var(--surface)' }}>
          {lanes.length === 0 ? (
            <div className="p-8 text-center" style={{ color: 'var(--muted)' }}>
              <div className="md-typescale-title-medium">No Supply Lanes</div>
              <div className="md-typescale-body-medium mt-1">Create a lane to connect a factory to a warehouse.</div>
            </div>
          ) : (
            <table className="w-full">
              <thead>
                <tr style={{ background: 'var(--color-md-surface-container, #f0f0f0)' }}>
                  <th className="md-typescale-label-medium text-left p-3" style={{ color: 'var(--muted)' }}>Factory</th>
                  <th className="md-typescale-label-medium text-left p-3" style={{ color: 'var(--muted)' }}>Warehouse</th>
                  <th className="md-typescale-label-medium text-right p-3" style={{ color: 'var(--muted)' }}>Transit (h)</th>
                  <th className="md-typescale-label-medium text-right p-3" style={{ color: 'var(--muted)' }}>Dampened (h)</th>
                  <th className="md-typescale-label-medium text-right p-3" style={{ color: 'var(--muted)' }}>Dist. (km)</th>
                  <th className="md-typescale-label-medium text-right p-3" style={{ color: 'var(--muted)' }}>Cost</th>
                  <th className="md-typescale-label-medium text-right p-3" style={{ color: 'var(--muted)' }}>Est. Impact (kg CO₂)</th>
                  <th className="md-typescale-label-medium text-center p-3" style={{ color: 'var(--muted)' }}>Priority</th>
                  <th className="md-typescale-label-medium text-center p-3" style={{ color: 'var(--muted)' }}>Status</th>
                  <th className="md-typescale-label-medium text-right p-3" style={{ color: 'var(--muted)' }}>Actions</th>
                </tr>
              </thead>
              <tbody>
                {lanes.map(lane => (
                  <tr key={lane.lane_id} className="border-t" style={{ borderColor: 'var(--border)' }}>
                    <td className="md-typescale-body-medium p-3 font-mono" style={{ color: 'var(--foreground)' }}>
                      {lane.factory_id.substring(0, 8)}...
                    </td>
                    <td className="md-typescale-body-medium p-3 font-mono" style={{ color: 'var(--foreground)' }}>
                      {lane.warehouse_id.substring(0, 8)}...
                    </td>
                    <td className="md-typescale-body-medium p-3 text-right" style={{ color: 'var(--foreground)' }}>
                      {lane.transit_time_hours.toFixed(1)}
                    </td>
                    <td className="md-typescale-body-medium p-3 text-right" style={{ color: 'var(--foreground)' }}>
                      {lane.dampened_transit_hours.toFixed(1)}
                    </td>
                    <td className="md-typescale-body-medium p-3 text-right" style={{ color: 'var(--muted)' }}>
                      {lane.direct_distance_km > 0 ? lane.direct_distance_km.toFixed(1) : '—'}
                    </td>
                    <td className="md-typescale-body-medium p-3 text-right" style={{ color: 'var(--foreground)' }}>
                      {(lane.freight_cost_minor / 100).toLocaleString()} UZS
                    </td>
                    <td className="md-typescale-body-medium p-3 text-right" style={{ color: 'var(--foreground)' }}>
                      {lane.carbon_score_kg.toFixed(1)}
                    </td>
                    <td className="md-typescale-body-medium p-3 text-center" style={{ color: 'var(--foreground)' }}>
                      {lane.priority}
                    </td>
                    <td className="p-3 text-center">
                      <span className="px-2 py-1 rounded-full md-typescale-label-small"
                        style={{
                          background: lane.is_active ? 'var(--color-md-success-container, #c8f5c8)' : 'var(--color-md-surface-variant, #e0e0e0)',
                          color: lane.is_active ? 'var(--color-md-on-success-container, #002200)' : 'var(--muted)',
                        }}>
                        {lane.is_active ? 'Active' : 'Inactive'}
                      </span>
                    </td>
                    <td className="p-3 text-right">
                      {lane.is_active && (
                        <button
                          className="md-typescale-label-small px-2 py-1"
                          style={{ color: 'var(--color-md-error)' }}
                          onClick={() => deactivateLane(lane.lane_id)}
                        >
                          Deactivate
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}

      {tab === 'sla' && (
        <div className="md-card md-card-elevated md-shape-md overflow-hidden" style={{ background: 'var(--surface)' }}>
          {slaEvents.length === 0 ? (
            <div className="p-8 text-center" style={{ color: 'var(--muted)' }}>
              <div className="md-typescale-title-medium">No SLA Events</div>
              <div className="md-typescale-body-medium mt-1">All transfers are within expected delivery windows.</div>
            </div>
          ) : (
            <table className="w-full">
              <thead>
                <tr style={{ background: 'var(--color-md-surface-container, #f0f0f0)' }}>
                  <th className="md-typescale-label-medium text-left p-3" style={{ color: 'var(--muted)' }}>Transfer</th>
                  <th className="md-typescale-label-medium text-left p-3" style={{ color: 'var(--muted)' }}>Factory</th>
                  <th className="md-typescale-label-medium text-left p-3" style={{ color: 'var(--muted)' }}>Warehouse</th>
                  <th className="md-typescale-label-medium text-center p-3" style={{ color: 'var(--muted)' }}>Level</th>
                  <th className="md-typescale-label-medium text-right p-3" style={{ color: 'var(--muted)' }}>Breach (min)</th>
                </tr>
              </thead>
              <tbody>
                {slaEvents.map(evt => (
                  <tr key={evt.event_id} className="border-t" style={{ borderColor: 'var(--border)' }}>
                    <td className="md-typescale-body-medium p-3 font-mono" style={{ color: 'var(--foreground)' }}>
                      {evt.transfer_id.substring(0, 8)}...
                    </td>
                    <td className="md-typescale-body-medium p-3 font-mono" style={{ color: 'var(--foreground)' }}>
                      {evt.factory_id.substring(0, 8)}...
                    </td>
                    <td className="md-typescale-body-medium p-3 font-mono" style={{ color: 'var(--foreground)' }}>
                      {evt.warehouse_id.substring(0, 8)}...
                    </td>
                    <td className="p-3 text-center">
                      <span className="px-2 py-1 rounded-full md-typescale-label-small font-medium"
                        style={{ color: ESCALATION_COLORS[evt.escalation_level] ?? 'var(--muted)' }}>
                        {evt.escalation_level}
                      </span>
                    </td>
                    <td className="md-typescale-body-medium p-3 text-right" style={{ color: 'var(--foreground)' }}>
                      {evt.breach_minutes}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}

      {tab === 'audit' && (
        <div className="md-card md-card-elevated md-shape-md overflow-hidden" style={{ background: 'var(--surface)' }}>
          {runs.length === 0 ? (
            <div className="p-8 text-center" style={{ color: 'var(--muted)' }}>
              <div className="md-typescale-title-medium">No Pull Matrix Runs</div>
              <div className="md-typescale-body-medium mt-1">The automated aggregation engine hasn&apos;t run yet.</div>
            </div>
          ) : (
            <table className="w-full">
              <thead>
                <tr style={{ background: 'var(--color-md-surface-container, #f0f0f0)' }}>
                  <th className="md-typescale-label-medium text-left p-3" style={{ color: 'var(--muted)' }}>Run At</th>
                  <th className="md-typescale-label-medium text-center p-3" style={{ color: 'var(--muted)' }}>Source</th>
                  <th className="md-typescale-label-medium text-right p-3" style={{ color: 'var(--muted)' }}>Transfers</th>
                  <th className="md-typescale-label-medium text-right p-3" style={{ color: 'var(--muted)' }}>SKUs</th>
                  <th className="md-typescale-label-medium text-right p-3" style={{ color: 'var(--muted)' }}>Duration</th>
                </tr>
              </thead>
              <tbody>
                {runs.map(run => (
                  <tr key={run.run_id} className="border-t" style={{ borderColor: 'var(--border)' }}>
                    <td className="md-typescale-body-medium p-3" style={{ color: 'var(--foreground)' }}>
                      {new Date(run.run_at).toLocaleString()}
                    </td>
                    <td className="p-3 text-center">
                      <span className="px-2 py-1 rounded-full md-typescale-label-small"
                        style={{ background: 'var(--color-md-surface-variant, #e0e0e0)', color: 'var(--foreground)' }}>
                        {run.source}
                      </span>
                    </td>
                    <td className="md-typescale-body-medium p-3 text-right" style={{ color: 'var(--foreground)' }}>
                      {run.transfers_generated}
                    </td>
                    <td className="md-typescale-body-medium p-3 text-right" style={{ color: 'var(--foreground)' }}>
                      {run.skus_processed}
                    </td>
                    <td className="md-typescale-body-medium p-3 text-right" style={{ color: 'var(--muted)' }}>
                      {run.duration_ms}ms
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}

      {/* ── Create Lane Drawer ── */}
      {creating && (
        <div className="fixed inset-0 z-50 flex justify-end" style={{ background: 'rgba(0,0,0,0.4)' }}>
          <div className="w-[400px] h-full p-6 overflow-auto" style={{ background: 'var(--surface)' }}>
            <div className="flex items-center justify-between mb-6">
              <h2 className="md-typescale-title-large" style={{ color: 'var(--foreground)' }}>New Supply Lane</h2>
              <button className="md-btn md-btn-icon" onClick={() => setCreating(false)}>✕</button>
            </div>
            <div className="space-y-4">
              <div>
                <label className="md-typescale-label-medium block mb-1" style={{ color: 'var(--muted)' }}>Factory ID</label>
                <input className="md-input-outlined w-full px-3 py-2" placeholder="UUID" value={createForm.factory_id}
                  onChange={e => setCreateForm(f => ({ ...f, factory_id: e.target.value }))} />
              </div>
              <div>
                <label className="md-typescale-label-medium block mb-1" style={{ color: 'var(--muted)' }}>Warehouse ID</label>
                <input className="md-input-outlined w-full px-3 py-2" placeholder="UUID" value={createForm.warehouse_id}
                  onChange={e => setCreateForm(f => ({ ...f, warehouse_id: e.target.value }))} />
              </div>
              <div>
                <label className="md-typescale-label-medium block mb-1" style={{ color: 'var(--muted)' }}>Transit Time (hours)</label>
                <input className="md-input-outlined w-full px-3 py-2" type="number" value={createForm.transit_time_hours}
                  onChange={e => setCreateForm(f => ({ ...f, transit_time_hours: e.target.value }))} />
              </div>
              <div>
                <label className="md-typescale-label-medium block mb-1" style={{ color: 'var(--muted)' }}>Freight Cost (minor units)</label>
                <input className="md-input-outlined w-full px-3 py-2" type="number" value={createForm.freight_cost_minor}
                  onChange={e => setCreateForm(f => ({ ...f, freight_cost_minor: e.target.value }))} />
              </div>
              <div className="md-card p-3 md-shape-sm" style={{ background: 'var(--color-md-surface-variant, #e7e0ec)', color: 'var(--muted)' }}>
                <div className="md-typescale-label-medium">Carbon Score</div>
                <div className="md-typescale-body-small mt-1">
                  Estimated automatically from factory/warehouse coordinates using Haversine distance (0.1 kg CO₂/km).
                  Displayed as &quot;Est. Impact&quot; in the table.
                </div>
              </div>
              <div>
                <label className="md-typescale-label-medium block mb-1" style={{ color: 'var(--muted)' }}>Priority</label>
                <input className="md-input-outlined w-full px-3 py-2" type="number" value={createForm.priority}
                  onChange={e => setCreateForm(f => ({ ...f, priority: e.target.value }))} />
              </div>
              <button className="md-btn md-btn-filled md-typescale-label-large w-full py-3 mt-4" onClick={handleCreateLane}>
                Create Lane
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
