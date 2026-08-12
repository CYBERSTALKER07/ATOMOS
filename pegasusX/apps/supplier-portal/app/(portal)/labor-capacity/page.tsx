"use client";

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useMemo, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { LaborDriverScore, LaborZoneCapacity } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import EmptyState from "@/components/EmptyState";
import { useToast } from "@/components/Toast";

const api = createSupplierApi();

export default function LaborCapacityPage() {
  const t = usePortalT();
  const { toast } = useToast();
  const [date, setDate] = useState(() => new Date().toISOString().slice(0, 10));
  const [zones, setZones] = useState<LaborZoneCapacity[]>([]);
  const [loading, setLoading] = useState(true);
  const [driverId, setDriverId] = useState("");
  const [score, setScore] = useState<LaborDriverScore | null>(null);
  const [availHours, setAvailHours] = useState("8");
  const [availStatus, setAvailStatus] = useState("AVAILABLE");
  const [zoneH3, setZoneH3] = useState("");
  const [saving, setSaving] = useState(false);

  const loadZones = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.listLaborZoneCapacity(date);
      if (Array.isArray((data as { zones?: LaborZoneCapacity[] }).zones)) {
        setZones((data as { zones: LaborZoneCapacity[] }).zones);
      } else if ((data as LaborZoneCapacity).zoneH3) {
        setZones([data as LaborZoneCapacity]);
      } else {
        setZones([]);
      }
    } catch (err) {
      toast(err instanceof Error ? err.message : "Failed to load zone capacity", "error");
      setZones([]);
    } finally {
      setLoading(false);
    }
  }, [date, toast]);

  useEffect(() => {
    void loadZones();
  }, [loadZones]);

  const loadScore = async () => {
    const id = driverId.trim();
    if (!id) {
      toast("Driver ID required", "error");
      return;
    }
    try {
      setScore(await api.getLaborDriverScore(id));
    } catch {
      setScore(null);
      toast("Driver score not found", "error");
    }
  };

  const setAvailability = async () => {
    const id = driverId.trim();
    if (!id) {
      toast("Driver ID required", "error");
      return;
    }
    setSaving(true);
    try {
      await api.setLaborDriverAvailability({
        driverId: id,
        date,
        availableHours: Number(availHours) || 0,
        zoneH3: zoneH3.trim() || undefined,
        status: availStatus,
      });
      toast("Availability saved", "success");
      await loadZones();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Save failed", "error");
    } finally {
      setSaving(false);
    }
  };

  const zoneRows = useMemo(() => zones, [zones]);

  return (
    <PageChrome
      icon="fleet"
      title={t("portal.nav.labor_capacity")}
      description="Zone delivery capacity and driver reliability scores (labor-capacity API)."
      loading={loading}
      skeletonVariant="table"
    >
      <div className="flex flex-col gap-8">
        <section className="flex flex-wrap gap-3 items-end">
          <label className="flex flex-col gap-1 text-sm">
            Date
            <input className="md-input" type="date" value={date} onChange={(e) => setDate(e.target.value)} />
          </label>
          <button type="button" className="md-btn md-btn-filled" onClick={() => void loadZones()}>
            Refresh zones
          </button>
        </section>

        {zoneRows.length === 0 ? (
          <EmptyState title="No zone capacity rows" description="Workers populate ZoneCapacity after availability is set." />
        ) : (
          <table className="md-table w-full text-sm">
            <thead>
              <tr>
                <th>Zone H3</th>
                <th>Total</th>
                <th>Used</th>
                <th>Utilization</th>
              </tr>
            </thead>
            <tbody>
              {zoneRows.map((z) => {
                const util = z.totalCapacity > 0 ? (z.usedCapacity / z.totalCapacity) * 100 : 0;
                return (
                  <tr key={`${z.zoneH3}-${z.date}`}>
                    <td className="font-mono text-xs">{z.zoneH3}</td>
                    <td>{z.totalCapacity.toFixed(1)}</td>
                    <td>{z.usedCapacity.toFixed(1)}</td>
                    <td>{util.toFixed(0)}%</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}

        <section className="flex flex-col gap-3 p-4 rounded border" style={{ borderColor: "var(--desk-border)" }}>
          <h2 className="md-typescale-title-medium">Driver score & availability</h2>
          <div className="flex flex-wrap gap-3 items-end">
            <label className="flex flex-col gap-1 text-sm">
              Driver ID
              <input className="md-input min-w-[200px]" value={driverId} onChange={(e) => setDriverId(e.target.value)} />
            </label>
            <button type="button" className="md-btn md-btn-tonal" onClick={() => void loadScore()}>
              Load score
            </button>
          </div>
          {score ? (
            <dl className="grid grid-cols-2 md:grid-cols-4 gap-3 text-sm">
              <div>
                <dt className="opacity-70">Score</dt>
                <dd className="font-semibold">{score.score.toFixed(1)}</dd>
              </div>
              <div>
                <dt className="opacity-70">On-time</dt>
                <dd>{(score.onTimeRate * 100).toFixed(0)}%</dd>
              </div>
              <div>
                <dt className="opacity-70">Completion</dt>
                <dd>{(score.completionRate * 100).toFixed(0)}%</dd>
              </div>
              <div>
                <dt className="opacity-70">Stops/hr</dt>
                <dd>{score.stopsPerHour.toFixed(1)}</dd>
              </div>
            </dl>
          ) : null}
          <div className="flex flex-wrap gap-3 items-end">
            <label className="flex flex-col gap-1 text-sm">
              Hours
              <input className="md-input w-24" value={availHours} onChange={(e) => setAvailHours(e.target.value)} />
            </label>
            <label className="flex flex-col gap-1 text-sm">
              Status
              <select className="md-input" value={availStatus} onChange={(e) => setAvailStatus(e.target.value)}>
                <option value="AVAILABLE">AVAILABLE</option>
                <option value="LIMITED">LIMITED</option>
                <option value="OFF">OFF</option>
              </select>
            </label>
            <label className="flex flex-col gap-1 text-sm">
              Zone H3
              <input className="md-input" value={zoneH3} onChange={(e) => setZoneH3(e.target.value)} placeholder="optional" />
            </label>
            <button type="button" className="md-btn md-btn-filled" onClick={() => void setAvailability()} disabled={saving}>
              Save availability
            </button>
          </div>
        </section>
      </div>
    </PageChrome>
  );
}
