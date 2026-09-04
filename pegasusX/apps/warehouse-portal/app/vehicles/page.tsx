'use client';

import { usePortalT } from "@/lib/i18n";
import { useState } from 'react';
import { warehouseCreateVehicleKey } from '@pegasusx/api-core';
import { apiFetch } from '@/lib/auth';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
import { useWarehouseVehiclesLive } from '@/lib/use-warehouse-vehicles-live';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { useWarehouseSessionReconcile } from '@/lib/use-warehouse-session-reconcile';
import { VehiclesList } from '../../components/vehicles/VehiclesList';

export default function VehiclesPage() {
  const t = usePortalT();
  const { vehicles, loading, error, liveMessage, reload } = useWarehouseVehiclesLive();
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState({ label: '', license_plate: '', vehicle_class: 'CLASS_A' });
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState('');

  useWarehouseSessionReconcile(() => {
    void reload();
  });

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setCreating(true);
    setCreateError('');
    const warehouseId = warehouseHomeNodeId() || 'warehouse';
    try {
      const res = await apiFetch('/v1/warehouse/ops/vehicles', {
        method: 'POST',
        body: JSON.stringify(form),
        headers: {
          'Idempotency-Key': warehouseCreateVehicleKey(warehouseId, form.license_plate),
        },
      });
      if (!res.ok) {
        throw new Error('Unable to create truck');
      }
      setForm({ label: '', license_plate: '', vehicle_class: 'CLASS_A' });
      setShowCreate(false);
      await reload();
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : 'Unable to create truck');
    } finally {
      setCreating(false);
    }
  }

  return (
    <PageTransition>
      <PageChrome
        icon="fleet"
        title={t("portal.nav.trucks")}
        description={t("warehouse_portal.residual.text.fleet_trucks_with_capacity_driver_assignment_and_live_availabili")}
        loading={loading}
        skeletonVariant="table"
        error={error}
        actions={
          <div className="flex gap-2">
            <button type="button" onClick={() => setShowCreate(!showCreate)} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--primary">
              <Icon name="plus" size={16} /> Add truck
            </button>
            <button type="button" onClick={() => { void reload(); }} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary">
              <Icon name="refresh" size={16} />
            </button>
          </div>
        }
      >
        {liveMessage && (
          <p className="text-sm mb-4" style={{ color: 'var(--warning)' }}>{liveMessage}</p>
        )}

        {showCreate && (
          <form onSubmit={handleCreate} className="p-4 rounded-xl border border-(--border) space-y-3 mb-4" style={{ background: 'var(--surface)' }}>
            <h2 className="text-sm font-semibold">{t("warehouse_portal.vehicles.text.new_truck")}</h2>
            {createError && <p className="text-sm" style={{ color: 'var(--danger)' }}>{createError}</p>}
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
              <input
                placeholder={t("warehouse_portal.vehicles.text.label_e_g_truck_01")}
                value={form.label}
                onChange={e => setForm({ ...form, label: e.target.value })}
                required
                className="px-3 py-2 rounded-lg border text-sm"
                style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)', color: 'var(--field-foreground)' }}
              />
              <input
                placeholder={t("warehouse_portal.vehicles._vehicle_id_.text.license_plate")}
                value={form.license_plate}
                onChange={e => setForm({ ...form, license_plate: e.target.value })}
                required
                className="px-3 py-2 rounded-lg border text-sm"
                style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)', color: 'var(--field-foreground)' }}
              />
              <select
                value={form.vehicle_class}
                onChange={e => setForm({ ...form, vehicle_class: e.target.value })}
                className="px-3 py-2 rounded-lg border text-sm"
                style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)', color: 'var(--field-foreground)' }}
              >
                <option value="CLASS_A">{t("warehouse_portal.vehicles.text.class_a_50_vu")}</option>
                <option value="CLASS_B">{t("warehouse_portal.vehicles.text.class_b_150_vu")}</option>
                <option value="CLASS_C">{t("warehouse_portal.vehicles.text.class_c_400_vu")}</option>
              </select>
            </div>
            <button type="submit" disabled={creating} className="px-4 py-2 rounded-lg text-sm font-semibold button--primary disabled:opacity-50">
              {creating ? 'Creating…' : 'Create truck'}
            </button>
          </form>
        )}

        <VehiclesList vehicles={vehicles} loading={loading} />
      </PageChrome>
    </PageTransition>
  );
}
