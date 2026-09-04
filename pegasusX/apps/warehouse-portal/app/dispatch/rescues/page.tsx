'use client';

import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { warehouseApi } from '@/lib/warehouse-api';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
import { useToast } from '@/components/Toast';
import { PageChrome } from '@/components/PageChrome';
import { PageSection } from '@/components/PageSection';
import EmptyState from '@/components/EmptyState';
import type { WarehouseDispatchDriver } from '@pegasusx/types';

export default function RescuesPage() {
  const t = usePortalT();
  const router = useRouter();
  const { toast } = useToast();
  
  const [warehouseId, setWarehouseId] = useState(() => warehouseHomeNodeId() || 'warehouse');
  const [drivers, setDrivers] = useState<WarehouseDispatchDriver[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  
  const [selectedBrokenDriver, setSelectedBrokenDriver] = useState<WarehouseDispatchDriver | null>(null);
  const [rescueOptions, setRescueOptions] = useState<any[]>([]);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [proposeLoading, setProposeLoading] = useState(false);

  useEffect(() => {
    loadDrivers();
  }, [warehouseId]);

  async function loadDrivers() {
    setLoading(true);
    setError(null);
    try {
      const data = await warehouseApi.previewWarehouseDispatch({ warehouse_id: warehouseId }, {});
      setDrivers(data.drivers || []);
    } catch (e: any) {
      setError(e.message || 'Failed to load drivers');
    } finally {
      setLoading(false);
    }
  }

  async function handleSelectBrokenDriver(driver: WarehouseDispatchDriver) {
    setSelectedBrokenDriver(driver);
    setPreviewLoading(true);
    setRescueOptions([]);
    try {
      const result = await warehouseApi.postWarehousePreviewRescue({
        broken_driver_id: driver.driver_id
      }, { warehouse_id: warehouseId });
      setRescueOptions(result.rescue_options || []);
    } catch (e: any) {
      toast(e.message || 'Failed to preview rescues', 'error');
      setSelectedBrokenDriver(null);
    } finally {
      setPreviewLoading(false);
    }
  }

  async function handleProposeRescue(rescueDriverId: string) {
    if (!selectedBrokenDriver) return;
    setProposeLoading(true);
    try {
      await warehouseApi.postWarehouseProposeRescue({
        broken_driver_id: selectedBrokenDriver.driver_id,
        rescue_driver_id: rescueDriverId,
        rescue_id: crypto.randomUUID()
      }, { warehouse_id: warehouseId });
      toast('Rescue proposed successfully', 'success');
      setSelectedBrokenDriver(null);
      setRescueOptions([]);
      loadDrivers();
    } catch (e: any) {
      toast(e.message || 'Failed to propose rescue', 'error');
    } finally {
      setProposeLoading(false);
    }
  }

  const brokenDrivers = drivers.filter(d => d.truck_status === 'NEEDS_RESCUE');

  return (
    <PageChrome
      title={t("warehouse_portal.dispatch.rescues.text.fleet_rescues")}
      description={t("warehouse_portal.residual.text.manage_truck_breakdowns_and_propose_rescue_operations")}
      icon="dispatch"
      loading={loading}
      error={error}
    >
      <PageSection title={t("warehouse_portal.dispatch.rescues.text.needs_rescue")}>
        {brokenDrivers.length === 0 ? (
          <EmptyState headline={t("warehouse_portal.residual.text.all_good")} body={t("warehouse_portal.residual.text.no_trucks_currently_require_a_rescue")} icon="check" />
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {brokenDrivers.map(d => (
              <div key={d.driver_id} className="border p-4 rounded-xl flex items-center justify-between bg-card text-card-foreground">
                <div>
                  <div className="font-semibold">{d.name}</div>
                  <div className="text-sm text-muted-foreground">{d.vehicle_label || d.vehicle_id || '—'} · {d.truck_status}</div>
                </div>
                <button
                  className="px-4 py-2 bg-primary text-primary-foreground rounded-lg font-medium"
                  onClick={() => handleSelectBrokenDriver(d)}
                >
                  Find Rescue
                </button>
              </div>
            ))}
          </div>
        )}
      </PageSection>

      {selectedBrokenDriver && (
        <PageSection title={`Rescue Options for ${selectedBrokenDriver.name}`}>
          {previewLoading ? (
            <div>{t("warehouse_portal.dispatch.rescues.text.finding_available_trucks")}</div>
          ) : rescueOptions.length === 0 ? (
            <EmptyState headline={t("warehouse_portal.residual.text.no_rescuers_available")} body={t("warehouse_portal.residual.text.there_are_no_active_drivers_with_enough_capacity")} />
          ) : (
            <div className="space-y-4">
              {rescueOptions.map(opt => (
                <div key={opt.driver_id} className="border p-4 rounded-xl flex items-center justify-between">
                  <div>
                    <div className="font-semibold">{opt.name}</div>
                    <div className="text-sm text-muted-foreground">
                      {opt.license_plate} · Capacity: {opt.effective_capacity_vu.toFixed(1)} VU
                      {opt.is_capacity_exceeded && <span className="text-destructive ml-2 font-medium">{t("warehouse_portal.dispatch.rescues.text.insufficient_capacity")}</span>}
                    </div>
                  </div>
                  <button
                    className="px-4 py-2 bg-accent text-accent-foreground rounded-lg font-medium disabled:opacity-50"
                    disabled={opt.is_capacity_exceeded || proposeLoading}
                    onClick={() => handleProposeRescue(opt.driver_id)}
                  >
                    Propose
                  </button>
                </div>
              ))}
            </div>
          )}
        </PageSection>
      )}
    </PageChrome>
  );
}
