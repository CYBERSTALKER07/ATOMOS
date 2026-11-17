"use client";

import { useState } from "react";
import { usePortalT } from "@/lib/i18n";
import { migrationProjectCreateKey } from "@pegasusx/api-core";
import type { MigrationProject } from "@pegasusx/types";
import type { ApiClient } from "@pegasusx/api-core";

export interface MigrationProjectSidebarProps {
  api: Pick<ApiClient, "createMigrationProject">;
  supplierId: string;
  projects: MigrationProject[];
  selectedProjectId: string | null;
  warehouses: Array<{ warehouse_id: string; name: string }>;
  onSelectProject: (projectId: string) => void;
  onProjectCreated: (project: MigrationProject) => void;
  onError: (msg: string) => void;
}

export function MigrationProjectSidebar({
  api,
  supplierId,
  projects,
  selectedProjectId,
  warehouses,
  onSelectProject,
  onProjectCreated,
  onError,
}: MigrationProjectSidebarProps) {
  const t = usePortalT();
  const [name, setName] = useState("");
  const [warehouseId, setWarehouseId] = useState(warehouses[0]?.warehouse_id ?? "");
  const [creating, setCreating] = useState(false);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    const trimmed = name.trim();
    if (!trimmed || creating) return;
    setCreating(true);
    try {
      const created = await api.createMigrationProject(
        {
          name: trimmed,
          mode: "CUTOVER",
          default_warehouse_id: warehouseId || warehouses[0]?.warehouse_id || undefined,
        },
        migrationProjectCreateKey(supplierId, trimmed),
      );
      setName("");
      onProjectCreated(created);
    } catch (err) {
      onError(err instanceof Error ? err.message : String(err));
    } finally {
      setCreating(false);
    }
  }

  return (
    <aside className="rounded-xl border border-[var(--border)] bg-[var(--surface)] p-4 flex flex-col gap-4">
      <div>
        <h2 className="md-typescale-title-medium font-semibold">
          {t("supplier_portal.migrations.create_project")}
        </h2>
        <form onSubmit={(e) => void handleCreate(e)} className="mt-3 flex flex-col gap-3">
          <label className="flex flex-col gap-1 text-xs">
            <span>{t("supplier_portal.migrations.project_name")}</span>
            <input
              type="text"
              className="md-input-outlined px-3 py-2 text-sm"
              placeholder={t("supplier_portal.migrations.project_name_placeholder")}
              value={name}
              onChange={(e) => setName(e.target.value)}
              disabled={creating}
            />
          </label>
          {warehouses.length > 0 && (
            <label className="flex flex-col gap-1 text-xs">
              <span>{t("supplier_portal.migrations.warehouse_label")}</span>
              <select
                className="md-input-outlined px-3 py-2 text-sm"
                value={warehouseId || warehouses[0]?.warehouse_id || ""}
                onChange={(e) => setWarehouseId(e.target.value)}
                disabled={creating}
              >
                {warehouses.map((w) => (
                  <option key={w.warehouse_id} value={w.warehouse_id}>
                    {w.name}
                  </option>
                ))}
              </select>
            </label>
          )}
          <button
            type="submit"
            disabled={creating || name.trim().length === 0}
            className="md-btn md-btn-filled px-4 py-2 text-sm"
          >
            {creating
              ? t("supplier_portal.migrations.creating")
              : t("supplier_portal.migrations.create_button")}
          </button>
        </form>
      </div>

      <div className="border-t border-[var(--border)] pt-3">
        <h3 className="md-typescale-label-large mb-2 text-[var(--color-md-outline)]">
          {t("supplier_portal.migrations.title")} ({projects.length})
        </h3>
        {projects.length === 0 ? (
          <p className="text-xs text-[var(--muted)]">
            {t("supplier_portal.migrations.no_projects")}
          </p>
        ) : (
          <ul className="flex flex-col gap-2">
            {projects.map((p) => {
              const active = p.project_id === selectedProjectId;
              return (
                <li key={p.project_id}>
                  <button
                    type="button"
                    onClick={() => onSelectProject(p.project_id)}
                    className={`w-full text-left rounded-lg border px-3 py-2 transition ${
                      active
                        ? "border-[var(--color-md-primary)] bg-[var(--color-md-surface-variant)]"
                        : "border-[var(--border)] hover:border-[var(--color-md-outline)]"
                    }`}
                  >
                    <div className="flex items-center justify-between gap-2">
                      <span className="text-sm font-medium truncate">{p.name}</span>
                      <span className="text-[11px] font-mono px-1.5 py-0.5 rounded bg-black/10">
                        {p.status}
                      </span>
                    </div>
                  </button>
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </aside>
  );
}
