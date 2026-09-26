"use client";

import { usePortalT } from "@/lib/i18n";
import React, { useCallback, useEffect, useState } from "react";
import type { ControlTowerPlaybookRun } from "@pegasusx/types";
import { createSupplierApi } from "@/lib/api";

const api = createSupplierApi();

export function PlaybookRunsPanel() {
  const t = usePortalT();
  const [runs, setRuns] = useState<ControlTowerPlaybookRun[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [acting, setActing] = useState<string | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    api
      .listPlaybookRuns("SUGGESTED")
      .then((resp) => setRuns(resp.runs ?? []))
      .catch((err) => setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.load_playbook_runs_failed")))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const approve = async (runId: string) => {
    setActing(runId);
    try {
      await api.approvePlaybookRun(runId, `approve-${runId}`);
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.approve_failed"));
    } finally {
      setActing(null);
    }
  };

  const skip = async (runId: string) => {
    setActing(runId);
    try {
      await api.skipPlaybookRun(runId, `skip-${runId}`);
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.skip_failed"));
    } finally {
      setActing(null);
    }
  };

  if (loading) {
    return <p className="text-sm text-[var(--color-md-outline)]">{t("supplier_portal.exceptions.playbook_runs_panel.text.loading_playbook_suggestions")}</p>;
  }

  if (error) {
    return (
      <p className="text-sm text-[var(--color-md-error)]">
        Playbooks unavailable ({error}). Enable CONTROL_TOWER_PLAYBOOKS_ENABLED to surface suggestions.
      </p>
    );
  }

  if (runs.length === 0) {
    return (
      <p className="text-sm text-[var(--color-md-outline)]">
        No suggested playbooks. Open exceptions are evaluated every few minutes when playbooks are enabled.
      </p>
    );
  }

  return (
    <ul className="md-card divide-y divide-[var(--color-md-outline-variant)]">
      {runs.map((run) => (
        <li key={run.run_id} className="p-4 md-typescale-body-medium">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-medium">{run.playbook_name || run.playbook_id}</span>
            <span className="md-chip h-6 text-xs">{run.mode}</span>
            <span className="font-mono text-xs text-[var(--color-md-outline)]">{run.exception_id}</span>
          </div>
          <p className="mt-1 text-sm text-[var(--color-md-outline)]">
            Suggested {new Date(run.created_at).toLocaleString()}
          </p>
          <div className="mt-3 flex flex-wrap gap-2">
            <button
              type="button"
              className="md-btn md-btn-filled"
              disabled={acting === run.run_id}
              onClick={() => approve(run.run_id)}
            >
              Approve
            </button>
            <button
              type="button"
              className="md-btn md-btn-outlined"
              disabled={acting === run.run_id}
              onClick={() => skip(run.run_id)}
            >
              Skip
            </button>
          </div>
        </li>
      ))}
    </ul>
  );
}
