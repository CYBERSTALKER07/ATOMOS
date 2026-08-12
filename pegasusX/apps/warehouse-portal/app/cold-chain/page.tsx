'use client';

import { usePortalT } from '@/lib/i18n';
import { useCallback, useState } from 'react';
import { apiFetch } from '@/lib/auth';
import { PageChrome } from '@/components/PageChrome';
import EmptyState from '@/components/EmptyState';
import { useToast } from '@/components/Toast';

type TempReading = {
  reading_id: string;
  manifest_id: string;
  sensor_id?: string;
  recorded_at: string;
  temp_c: number;
  min_c?: number;
  max_c?: number;
  excursion?: boolean;
};

export default function ColdChainPage() {
  const t = usePortalT();
  const { toast } = useToast();
  const [manifestId, setManifestId] = useState('');
  const [readings, setReadings] = useState<TempReading[]>([]);
  const [enabled, setEnabled] = useState(true);
  const [loading, setLoading] = useState(false);
  const [sensorId, setSensorId] = useState('');
  const [tempC, setTempC] = useState('');
  const [minC, setMinC] = useState('');
  const [maxC, setMaxC] = useState('');
  const [posting, setPosting] = useState(false);

  const load = useCallback(async () => {
    const mid = manifestId.trim();
    if (!mid) {
      toast('Manifest ID required', 'error');
      return;
    }
    setLoading(true);
    try {
      const res = await apiFetch(
        `/v1/warehouse/ops/temperature-readings?manifest_id=${encodeURIComponent(mid)}`,
      );
      if (res.status === 409) {
        setEnabled(false);
        setReadings([]);
        return;
      }
      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        toast(err.error || 'Failed to load readings', 'error');
        return;
      }
      setEnabled(true);
      const data = await res.json();
      setReadings(data.readings || []);
    } catch {
      toast('Failed to load readings', 'error');
    } finally {
      setLoading(false);
    }
  }, [manifestId, toast]);

  const ingest = async () => {
    const mid = manifestId.trim();
    const temp = Number(tempC);
    if (!mid || !Number.isFinite(temp)) {
      toast('Manifest ID and temperature required', 'error');
      return;
    }
    setPosting(true);
    try {
      const body: Record<string, unknown> = {
        manifest_id: mid,
        sensor_id: sensorId.trim() || undefined,
        temp_c: temp,
      };
      if (minC.trim() !== '' && maxC.trim() !== '') {
        body.min_c = Number(minC);
        body.max_c = Number(maxC);
      }
      const res = await apiFetch('/v1/warehouse/ops/temperature-readings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      if (res.status === 409) {
        setEnabled(false);
        toast('Cold chain disabled on this environment', 'error');
        return;
      }
      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        toast(err.error || 'Ingest failed', 'error');
        return;
      }
      toast('Reading recorded', 'success');
      setTempC('');
      await load();
    } finally {
      setPosting(false);
    }
  };

  return (
    <PageChrome
      icon="warning"
      title={t('portal.nav.cold_chain')}
      description="Manifest temperature readings — excursions quarantine lots and raise system breaches."
      loading={loading}
      skeletonVariant="list"
    >
      {!enabled ? (
        <EmptyState
          title="Cold chain disabled"
          description="Set WMS_COLD_CHAIN_ENABLED=true on the API to enable temperature ingest."
        />
      ) : (
        <div className="flex flex-col gap-6">
          <div className="flex flex-wrap gap-3 items-end">
            <label className="flex flex-col gap-1 text-sm">
              Manifest ID
              <input
                className="md-input min-w-[220px]"
                value={manifestId}
                onChange={(e) => setManifestId(e.target.value)}
                placeholder="manifest uuid"
              />
            </label>
            <button type="button" className="md-btn md-btn-filled" onClick={() => void load()} disabled={loading}>
              Load readings
            </button>
          </div>

          <div className="flex flex-wrap gap-3 items-end p-4 rounded border" style={{ borderColor: 'var(--desk-border)' }}>
            <label className="flex flex-col gap-1 text-sm">
              Sensor ID
              <input className="md-input" value={sensorId} onChange={(e) => setSensorId(e.target.value)} />
            </label>
            <label className="flex flex-col gap-1 text-sm">
              Temp °C
              <input className="md-input w-28" value={tempC} onChange={(e) => setTempC(e.target.value)} />
            </label>
            <label className="flex flex-col gap-1 text-sm">
              Min °C
              <input className="md-input w-24" value={minC} onChange={(e) => setMinC(e.target.value)} placeholder="opt" />
            </label>
            <label className="flex flex-col gap-1 text-sm">
              Max °C
              <input className="md-input w-24" value={maxC} onChange={(e) => setMaxC(e.target.value)} placeholder="opt" />
            </label>
            <button type="button" className="md-btn md-btn-tonal" onClick={() => void ingest()} disabled={posting}>
              Record reading
            </button>
          </div>

          {readings.length === 0 ? (
            <EmptyState title="No readings" description="Load a manifest or record the first sample." />
          ) : (
            <div className="overflow-x-auto">
              <table className="md-table w-full text-sm">
                <thead>
                  <tr>
                    <th>Recorded</th>
                    <th>Temp °C</th>
                    <th>Band</th>
                    <th>Sensor</th>
                    <th>Status</th>
                  </tr>
                </thead>
                <tbody>
                  {readings.map((r) => (
                    <tr key={r.reading_id}>
                      <td className="font-mono text-xs">{r.recorded_at}</td>
                      <td>{r.temp_c.toFixed(2)}</td>
                      <td className="font-mono text-xs">
                        {r.min_c != null && r.max_c != null ? `${r.min_c}…${r.max_c}` : '—'}
                      </td>
                      <td>{r.sensor_id || '—'}</td>
                      <td style={{ color: r.excursion ? 'var(--color-md-error, #b3261e)' : undefined }}>
                        {r.excursion ? 'EXCURSION' : 'OK'}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}
    </PageChrome>
  );
}
