'use client';

import { useCallback, useEffect, useState } from 'react';
import { ApiError } from '@pegasusx/api-client';
import { warehouseApi } from '@/lib/warehouse-api';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';

export default function DispatchSettingsPage() {
  const [autoDispatchEnabled, setAutoDispatchEnabled] = useState<boolean | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saveSuccess, setSaveSuccess] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoadError(null);
    try {
      const data = await warehouseApi.getWarehouseDispatchSettings();
      setAutoDispatchEnabled(data.auto_dispatch_enabled);
    } catch (err) {
      setLoadError(err instanceof ApiError ? err.message : 'Failed to load dispatch settings');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const save = useCallback(async (next: boolean) => {
    setSaving(true);
    setSaveError(null);
    setSaveSuccess(null);
    try {
      await warehouseApi.patchWarehouseDispatchSettings({ auto_dispatch_enabled: next });
      setAutoDispatchEnabled(next);
      setSaveSuccess(next ? 'Auto dispatch enabled for this warehouse.' : 'Auto dispatch disabled for this warehouse.');
    } catch (err) {
      setSaveError(err instanceof ApiError ? err.message : 'Failed to update dispatch settings');
    } finally {
      setSaving(false);
    }
  }, []);

  return (
    <PageTransition>
      <PageChrome
        title="Dispatch settings"
        description="Configure warehouse auto-dispatch policy for this node."
        loading={loading}
        error={loadError}
        actions={
          <button
            type="button"
            onClick={() => {
              setLoading(true);
              void load();
            }}
            className="button--secondary flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm"
          >
            <Icon name="refresh" size={16} /> Refresh
          </button>
        }
      >
        {saveError && (
          <p className="mb-4 text-sm" style={{ color: 'var(--error)' }}>{saveError}</p>
        )}
        {saveSuccess && (
          <p className="mb-4 text-sm" style={{ color: 'var(--success)' }}>{saveSuccess}</p>
        )}

        <div className="rounded-xl border border-(--border) p-4" style={{ background: 'var(--background)' }}>
          <div className="flex items-start justify-between gap-4">
            <div>
              <h2 className="text-sm font-semibold">Auto dispatch</h2>
              <p className="mt-1 text-sm text-(--muted)">
                When enabled, the AI worker may commit dispatch assignments without manual operator confirmation.
              </p>
            </div>
            <button
              type="button"
              disabled={saving || autoDispatchEnabled === null}
              onClick={() => void save(!(autoDispatchEnabled ?? false))}
              className={`relative h-8 w-14 rounded-full transition-colors disabled:opacity-50 ${autoDispatchEnabled ? 'bg-(--accent)' : 'bg-(--border)'}`}
              aria-pressed={autoDispatchEnabled ?? false}
              aria-label="Toggle auto dispatch"
            >
              <span
                className={`absolute top-1 h-6 w-6 rounded-full bg-white transition-transform ${autoDispatchEnabled ? 'left-7' : 'left-1'}`}
              />
            </button>
          </div>
          <p className="mt-3 text-xs text-(--muted)">
            Current state: {autoDispatchEnabled === null ? '—' : autoDispatchEnabled ? 'ENABLED' : 'DISABLED'}
          </p>
        </div>
      </PageChrome>
    </PageTransition>
  );
}
