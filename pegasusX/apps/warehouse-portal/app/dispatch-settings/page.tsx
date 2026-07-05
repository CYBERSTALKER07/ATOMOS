'use client';

import { useCallback, useEffect, useState } from 'react';
import { ApiError, warehouseDispatchSettingsKey } from '@pegasusx/api-client';
import type { WarehouseDispatchPreview } from '@pegasusx/types';
import { warehouseApi } from '@/lib/warehouse-api';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';

type DispatchPreviewState = Partial<WarehouseDispatchPreview> & { error?: string };

export default function DispatchSettingsPage() {
  const [autoDispatchEnabled, setAutoDispatchEnabled] = useState<boolean | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saveSuccess, setSaveSuccess] = useState<string | null>(null);
  const [showPreviewModal, setShowPreviewModal] = useState(false);
  const [previewData, setPreviewData] = useState<DispatchPreviewState | null>(null);
  const [previewLoading, setPreviewLoading] = useState(false);

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
      await warehouseApi.patchWarehouseDispatchSettings(
        { auto_dispatch_enabled: next },
        {},
        warehouseDispatchSettingsKey(warehouseHomeNodeId() || 'warehouse', next),
      );
      setAutoDispatchEnabled(next);
      setSaveSuccess(next ? 'Auto dispatch enabled for this warehouse.' : 'Auto dispatch disabled for this warehouse.');
    } catch (err) {
      setSaveError(err instanceof ApiError ? err.message : 'Failed to update dispatch settings');
    } finally {
      setSaving(false);
    }
  }, []);

  const handleToggle = async (next: boolean) => {
    if (next) {
      // Enabling auto-dispatch -> show impact preview
      setPreviewLoading(true);
      setShowPreviewModal(true);
      try {
        const data = await warehouseApi.previewWarehouseDispatch({ warehouse_id: warehouseHomeNodeId() || undefined }, {});
        setPreviewData(data);
      } catch (err) {
        setPreviewData({ error: err instanceof ApiError ? err.message : 'Failed to load preview' });
      } finally {
        setPreviewLoading(false);
      }
    } else {
      // Just disable directly
      void save(false);
    }
  };

  return (
    <PageTransition>
      <PageChrome
        icon="settings"
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
                When enabled, the warehouse auto-dispatch worker commits optimizer assignments on a timed cadence without manual confirmation.
              </p>
            </div>
            <button
              type="button"
              disabled={saving || autoDispatchEnabled === null}
              onClick={() => void handleToggle(!(autoDispatchEnabled ?? false))}
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

        {showPreviewModal && (
          <div className="md-dialog-scrim fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm transition-opacity">
            <div className="md-dialog flex w-full max-w-md flex-col overflow-hidden rounded-[24px] bg-[var(--background)] shadow-2xl">
              <div className="px-6 pb-2 pt-6">
                <h2 className="md-dialog-title text-xl font-semibold tracking-tight text-[var(--foreground)]">
                  Enable Auto Dispatch
                </h2>
              </div>
              <div className="flex-1 overflow-y-auto px-6 py-2 text-sm text-[var(--muted)]">
                {previewLoading ? (
                  <p>Loading impact preview...</p>
                ) : previewData?.error ? (
                  <p className="text-(--error)">{previewData.error}</p>
                ) : (
                  <div className="space-y-4">
                    <p>Enabling auto dispatch will immediately schedule available drivers. Based on the current queue:</p>
                    <ul className="list-disc pl-5">
                      <li>{previewData?.undispatched_orders?.length || 0} orders waiting</li>
                      <li>{previewData?.available_drivers?.length || 0} drivers available</li>
                      <li>{previewData?.window_constrained_count || 0} constrained by delivery windows</li>
                    </ul>
                    <p>Are you sure you want to enable auto dispatch?</p>
                  </div>
                )}
              </div>
              <div className="md-dialog-actions flex items-center justify-end gap-2 p-6">
                <button
                  type="button"
                  className="button--secondary px-4 py-2"
                  onClick={() => setShowPreviewModal(false)}
                  disabled={saving}
                >
                  Cancel
                </button>
                <button
                  type="button"
                  className="button--primary px-4 py-2"
                  onClick={() => {
                    void save(true);
                    setShowPreviewModal(false);
                  }}
                  disabled={previewLoading || saving}
                >
                  {saving ? 'Enabling...' : 'Confirm'}
                </button>
              </div>
            </div>
          </div>
        )}
      </PageChrome>
    </PageTransition>
  );
}
