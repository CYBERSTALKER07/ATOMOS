"use client";

import React, { useCallback, useEffect, useState } from "react";
import type { ControlTowerPlaybook } from "@pegasusx/types";
import { createSupplierApi } from "@/lib/api";
import { PageChrome } from "@/components/PageChrome";

const api = createSupplierApi();

export default function PlaybooksSettingsPage() {
  const [playbooks, setPlaybooks] = useState<ControlTowerPlaybook[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [acting, setActing] = useState<string | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    setError(null);
    api
      .listPlaybooks()
      .then((resp) => setPlaybooks(resp.playbooks ?? []))
      .catch((err) => setError(err instanceof Error ? err.message : "load_playbooks_failed"))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const deactivate = async (id: string) => {
    setActing(id);
    try {
      await api.deactivatePlaybook(id);
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "deactivate_failed");
    } finally {
      setActing(null);
    }
  };

  return (
    <PageChrome
      title="Control tower playbooks"
      description="Review automated exception playbooks. Deactivate rules you do not want suggested."
    >
      {error ? <p className="text-sm text-[var(--color-md-error)]">{error}</p> : null}
      {loading ? (
        <p className="text-sm text-[var(--color-md-outline)]">Loading playbooks…</p>
      ) : playbooks.length === 0 ? (
        <p className="text-sm text-[var(--color-md-outline)]">No active playbooks.</p>
      ) : (
        <ul className="md-card divide-y divide-[var(--color-md-outline-variant)]">
          {playbooks.map((pb) => (
            <li key={pb.playbook_id} className="p-4 md-typescale-body-medium">
              <div className="flex flex-wrap items-center gap-2">
                <span className="font-medium">{pb.name}</span>
                <span className="md-chip h-6 text-xs">priority {pb.priority}</span>
                {pb.auto_execute ? <span className="md-chip h-6 text-xs">auto</span> : null}
              </div>
              {pb.description ? (
                <p className="mt-1 text-sm text-[var(--color-md-outline)]">{pb.description}</p>
              ) : null}
              <pre className="mt-2 text-xs overflow-x-auto rounded bg-[var(--color-md-surface-variant)] p-2">
                {JSON.stringify(pb.match_rules, null, 2)}
              </pre>
              <button
                type="button"
                className="md-btn md-btn-outlined mt-3"
                disabled={acting === pb.playbook_id}
                onClick={() => void deactivate(pb.playbook_id)}
              >
                Deactivate
              </button>
            </li>
          ))}
        </ul>
      )}
    </PageChrome>
  );
}
