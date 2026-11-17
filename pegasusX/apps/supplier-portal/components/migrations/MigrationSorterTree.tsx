"use client";

import { useEffect, useState } from "react";
import { usePortalT } from "@/lib/i18n";
import { migrationBundleSortKey, type ApiClient } from "@pegasusx/api-core";
import type {
  MigrationBundleSortOverrideResponse,
  MigrationBundleTree,
  MigrationPack,
  MigrationSession,
  MigrationSortSummary,
  MigrationSortedStreamDecision,
} from "@pegasusx/types";
import {
  ALL_MIGRATION_PACKS,
  ENCODING_OPTIONS,
  packLabelKey,
} from "./pack-labels";

export interface MigrationSorterTreeProps {
  api: Pick<ApiClient, "overrideMigrationBundleSort">;
  projectId: string;
  bundleTree: MigrationBundleTree;
  sessions: MigrationSession[];
  selectedSessionId: string | null;
  onSelectSession: (sessionId: string) => void;
  onSortUpdated: (res: MigrationBundleSortOverrideResponse) => void;
  onError: (msg: string) => void;
}

function parseSortSummary(raw: unknown): MigrationSortSummary | null {
  if (!raw || typeof raw !== "object") return null;
  const obj = raw as Partial<MigrationSortSummary>;
  if (!Array.isArray(obj.streams)) return null;
  return obj as MigrationSortSummary;
}

export function MigrationSorterTree({
  api,
  projectId,
  bundleTree,
  sessions,
  selectedSessionId,
  onSelectSession,
  onSortUpdated,
  onError,
}: MigrationSorterTreeProps) {
  const t = usePortalT();
  const summary = parseSortSummary(bundleTree.bundle.sort_summary);
  const streams: MigrationSortedStreamDecision[] = summary?.streams ?? [];

  const [packOverrides, setPackOverrides] = useState<Record<string, string>>({});
  const [encodingOverrides, setEncodingOverrides] = useState<Record<string, string>>({});
  const [ignoredStreams, setIgnoredStreams] = useState<Record<string, boolean>>({});
  const [inventoryMode, setInventoryMode] = useState<"stock" | "lots">("lots");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    const p: Record<string, string> = {};
    const enc: Record<string, string> = {};
    const ign: Record<string, boolean> = {};
    for (const st of streams) {
      const key = `${st.artifact_id}::${st.stream_key}`;
      p[key] = st.detected_packs[0] ?? "";
      enc[key] = st.encoding || "utf-8";
      ign[key] = Boolean(st.ignored);
    }
    setPackOverrides(p);
    setEncodingOverrides(enc);
    setIgnoredStreams(ign);
  }, [bundleTree.bundle.bundle_id, streams.length]);

  const bundleSessions = sessions.filter(
    (s) => s.bundle_id === bundleTree.bundle.bundle_id,
  );

  async function saveOverrides(confirmReview = true) {
    if (saving) return;
    setSaving(true);
    try {
      const overrides = streams.map((st) => {
        const k = `${st.artifact_id}::${st.stream_key}`;
        const chosenPack = packOverrides[k];
        return {
          artifact_id: st.artifact_id,
          stream_key: st.stream_key,
          packs: chosenPack ? [chosenPack] : undefined,
          encoding: encodingOverrides[k] || st.encoding,
          ignore: Boolean(ignoredStreams[k]),
          confirmed: true,
        };
      });
      const fp = JSON.stringify({ overrides, inventoryMode });
      const res = await api.overrideMigrationBundleSort(
        projectId,
        bundleTree.bundle.bundle_id,
        {
          overrides,
          resolved_inventory_mode: summary?.conflicting_inventory_mode
            ? inventoryMode
            : undefined,
          confirm_review: confirmReview,
        },
        migrationBundleSortKey(bundleTree.bundle.bundle_id, fp),
      );
      onSortUpdated(res);
    } catch (err) {
      onError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  }

  return (
    <section className="rounded-xl border border-[var(--border)] bg-[var(--surface)] p-4 flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h3 className="md-typescale-title-medium font-semibold">
            {t("supplier_portal.migrations.sorter_title")}
          </h3>
          <p className="text-xs text-[var(--muted)]">
            {t("supplier_portal.migrations.bundle_status", {
              status: bundleTree.bundle.status,
            })}
          </p>
        </div>
        {streams.length > 0 && (
          <button
            type="button"
            disabled={saving}
            onClick={() => void saveOverrides(true)}
            className="md-btn md-btn-outlined px-3 py-1.5 text-xs"
          >
            {t("supplier_portal.migrations.apply_sort_override")}
          </button>
        )}
      </div>

      {summary?.conflicting_inventory_mode && (
        <div className="rounded-lg border border-amber-500/40 bg-amber-500/10 p-3 text-sm">
          <p className="font-medium text-amber-800 dark:text-amber-300">
            {t("supplier_portal.migrations.stock_vs_lots_warning")}
          </p>
          <div className="mt-2 flex flex-wrap items-center gap-4 text-xs">
            <label className="flex items-center gap-1.5 cursor-pointer">
              <input
                type="radio"
                name={`inv-mode-${bundleTree.bundle.bundle_id}`}
                checked={inventoryMode === "lots"}
                onChange={() => setInventoryMode("lots")}
              />
              <span>{t("supplier_portal.migrations.pack_lots")}</span>
            </label>
            <label className="flex items-center gap-1.5 cursor-pointer">
              <input
                type="radio"
                name={`inv-mode-${bundleTree.bundle.bundle_id}`}
                checked={inventoryMode === "stock"}
                onChange={() => setInventoryMode("stock")}
              />
              <span>{t("supplier_portal.migrations.pack_stock")}</span>
            </label>
          </div>
        </div>
      )}

      {streams.length > 0 && (
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs border-collapse">
            <thead>
              <tr className="border-b border-[var(--border)] text-[var(--color-md-outline)]">
                <th className="py-2 pr-3">{t("supplier_portal.migrations.stream_label")}</th>
                <th className="py-2 pr-3">{t("supplier_portal.migrations.encoding_label")}</th>
                <th className="py-2 pr-3">{t("supplier_portal.migrations.reassign_stream")}</th>
                <th className="py-2 pr-3">{t("supplier_portal.migrations.ignore_column")}</th>
              </tr>
            </thead>
            <tbody>
              {streams.map((st) => {
                const k = `${st.artifact_id}::${st.stream_key}`;
                const pct = Math.round((st.confidence ?? 0) * 100);
                return (
                  <tr key={k} className="border-b border-[var(--border)]/50">
                    <td className="py-2 pr-3">
                      <div className="font-medium">
                        {st.file_name} · {st.stream_key}
                      </div>
                      <div className="text-[11px] text-[var(--muted)]">
                        {st.total_rows} rows ·{" "}
                        {t("supplier_portal.migrations.confidence_label", {
                          pct: String(pct),
                        })}
                        {st.review_reason ? ` · ${st.review_reason}` : ""}
                      </div>
                    </td>
                    <td className="py-2 pr-3">
                      <select
                        className="md-input-outlined px-2 py-1 text-xs"
                        value={encodingOverrides[k] ?? st.encoding}
                        onChange={(e) =>
                          setEncodingOverrides((prev) => ({ ...prev, [k]: e.target.value }))
                        }
                      >
                        {ENCODING_OPTIONS.map((enc) => (
                          <option key={enc} value={enc}>
                            {enc}
                          </option>
                        ))}
                      </select>
                    </td>
                    <td className="py-2 pr-3">
                      <select
                        className="md-input-outlined px-2 py-1 text-xs"
                        value={packOverrides[k] ?? ""}
                        onChange={(e) =>
                          setPackOverrides((prev) => ({ ...prev, [k]: e.target.value }))
                        }
                      >
                        <option value="">
                          {t("supplier_portal.migrations.pack_quarantine")}
                        </option>
                        {ALL_MIGRATION_PACKS.map((pk: MigrationPack) => (
                          <option key={pk} value={pk}>
                            {t(packLabelKey(pk))} ({pk})
                          </option>
                        ))}
                      </select>
                    </td>
                    <td className="py-2 pr-3">
                      <input
                        type="checkbox"
                        checked={Boolean(ignoredStreams[k])}
                        onChange={(e) =>
                          setIgnoredStreams((prev) => ({ ...prev, [k]: e.target.checked }))
                        }
                      />
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {bundleSessions.length > 0 && (
        <div className="flex flex-wrap gap-2 pt-2 border-t border-[var(--border)]">
          {bundleSessions.map((sess) => {
            const selected = sess.session_id === selectedSessionId;
            return (
              <button
                key={sess.session_id}
                type="button"
                onClick={() => onSelectSession(sess.session_id)}
                className={`rounded-lg border px-3 py-2 text-left text-xs transition ${
                  selected
                    ? "border-[var(--color-md-primary)] bg-[var(--color-md-primary)]/10 font-semibold"
                    : "border-[var(--border)] hover:border-[var(--color-md-outline)]"
                }`}
              >
                <div>{t(packLabelKey(sess.pack ?? ""))}</div>
                <div className="text-[11px] text-[var(--muted)]">
                  {sess.stream_split_key || sess.sheet_name || sess.file_name} ·{" "}
                  {sess.total_rows} rows · {sess.status}
                </div>
              </button>
            );
          })}
        </div>
      )}
    </section>
  );
}
