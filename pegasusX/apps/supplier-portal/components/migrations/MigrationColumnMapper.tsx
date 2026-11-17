"use client";

import { useCallback, useEffect, useState } from "react";
import { usePortalT } from "@/lib/i18n";
import { migrationMappingConfirmKey, type ApiClient } from "@pegasusx/api-core";
import type {
  MigrationCanonicalField,
  MigrationColumnMappingProposal,
  MigrationSessionMappingDocument,
  MigrationStagedRow,
} from "@pegasusx/types";

export interface MigrationColumnMapperProps {
  api: Pick<
    ApiClient,
    "getMigrationSessionMapping" | "getMigrationSessionRows" | "confirmMigrationSessionMapping"
  >;
  sessionId: string;
  onMappingConfirmed: () => void;
  onError: (msg: string) => void;
}

function sampleValuesForColumn(rows: MigrationStagedRow[], col: string): string {
  const vals: string[] = [];
  for (const r of rows.slice(0, 5)) {
    const raw = r.raw_data?.[col];
    if (raw !== undefined && raw !== null && String(raw).trim() !== "") {
      vals.push(String(raw).slice(0, 32));
    }
    if (vals.length >= 3) break;
  }
  return vals.join(" | ") || "—";
}

export function MigrationColumnMapper({
  api,
  sessionId,
  onMappingConfirmed,
  onError,
}: MigrationColumnMapperProps) {
  const t = usePortalT();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [taxonomy, setTaxonomy] = useState<MigrationCanonicalField[]>([]);
  const [doc, setDoc] = useState<MigrationSessionMappingDocument | null>(null);
  const [proposals, setProposals] = useState<MigrationColumnMappingProposal[]>([]);
  const [sampleRows, setSampleRows] = useState<MigrationStagedRow[]>([]);
  const [saveTemplate, setSaveTemplate] = useState(true);
  const [showSampleTable, setShowSampleTable] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [mapRes, rowsRes] = await Promise.all([
        api.getMigrationSessionMapping(sessionId),
        api.getMigrationSessionRows(sessionId, { limit: 20, offset: 0 }),
      ]);
      setTaxonomy(mapRes.taxonomy ?? []);
      setDoc(mapRes.mapping);
      setProposals(mapRes.mapping?.mappings ?? []);
      setSampleRows(rowsRes.rows ?? []);
    } catch (err) {
      onError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, [api, onError, sessionId]);

  useEffect(() => {
    void load();
  }, [load]);

  async function acceptHighConfidence() {
    if (saving || !doc) return;
    setSaving(true);
    try {
      const res = await api.confirmMigrationSessionMapping(
        sessionId,
        {
          mappings: proposals,
          accept_above_confidence: 0.8,
          save_to_library: saveTemplate,
          strict: false,
        },
        migrationMappingConfirmKey(sessionId, `${doc.mapping_hash}-accept80`),
      );
      setDoc(res.mapping);
      setProposals(res.mapping.mappings ?? []);
      onMappingConfirmed();
    } catch (err) {
      onError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  }

  async function confirmMapping() {
    if (saving || !doc) return;
    setSaving(true);
    try {
      const next = proposals.map((p) => ({
        ...p,
        confirmed: Boolean(p.target_field),
      }));
      const res = await api.confirmMigrationSessionMapping(
        sessionId,
        {
          mappings: next,
          save_to_library: saveTemplate,
          strict: true,
        },
        migrationMappingConfirmKey(sessionId, JSON.stringify(next)),
      );
      setDoc(res.mapping);
      setProposals(res.mapping.mappings ?? []);
      onMappingConfirmed();
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
            {t("supplier_portal.migrations.mapper_title")}
          </h3>
          {doc && (
            <p className="text-xs text-[var(--muted)]">
              AI: {doc.ai_model} ({doc.ai_status})
              {doc.missing_required && doc.missing_required.length > 0
                ? ` · Missing: ${doc.missing_required.join(", ")}`
                : ""}
            </p>
          )}
        </div>
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            disabled={loading || saving || proposals.length === 0}
            onClick={() => void acceptHighConfidence()}
            className="md-btn md-btn-outlined px-3 py-1.5 text-xs"
          >
            {t("supplier_portal.migrations.accept_high_confidence")}
          </button>
          <button
            type="button"
            disabled={loading || saving || proposals.length === 0}
            onClick={() => void confirmMapping()}
            className="md-btn md-btn-filled px-3 py-1.5 text-xs"
          >
            {t("supplier_portal.migrations.confirm_mapping")}
          </button>
        </div>
      </div>

      {loading ? (
        <p className="text-xs text-[var(--muted)]">Loading mapping…</p>
      ) : (
        <>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs border-collapse">
              <thead>
                <tr className="border-b border-[var(--border)] text-[var(--color-md-outline)]">
                  <th className="py-2 pr-3">{t("supplier_portal.migrations.source_column")}</th>
                  <th className="py-2 pr-3">{t("supplier_portal.migrations.sample_values")}</th>
                  <th className="py-2 pr-3">{t("supplier_portal.migrations.target_field")}</th>
                </tr>
              </thead>
              <tbody>
                {proposals.map((prop, idx) => {
                  const pct = Math.round((prop.confidence ?? 0) * 100);
                  return (
                    <tr key={prop.source_column} className="border-b border-[var(--border)]/50">
                      <td className="py-2 pr-3 font-mono font-medium">
                        {prop.source_column}
                        <span className="ml-2 text-[11px] font-sans text-[var(--muted)]">
                          {t("supplier_portal.migrations.confidence_label", { pct: String(pct) })}
                        </span>
                      </td>
                      <td className="py-2 pr-3 text-[var(--muted)] max-w-xs truncate">
                        {sampleValuesForColumn(sampleRows, prop.source_column)}
                      </td>
                      <td className="py-2 pr-3">
                        <select
                          className="md-input-outlined px-2 py-1 text-xs"
                          value={prop.target_field || ""}
                          onChange={(e) => {
                            const val = e.target.value;
                            setProposals((prev) =>
                              prev.map((item, i) =>
                                i === idx
                                  ? { ...item, target_field: val, confirmed: Boolean(val) }
                                  : item,
                              ),
                            );
                          }}
                        >
                          <option value="">
                            {t("supplier_portal.migrations.ignore_column")}
                          </option>
                          {taxonomy.map((field) => (
                            <option key={field.key} value={field.key}>
                              {field.label || field.key} ({field.key})
                              {field.required ? " *" : ""}
                            </option>
                          ))}
                        </select>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>

          <div className="flex flex-wrap items-center justify-between gap-2 text-xs pt-2 border-t border-[var(--border)]">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={saveTemplate}
                onChange={(e) => setSaveTemplate(e.target.checked)}
              />
              <span>{t("supplier_portal.migrations.save_template_toggle")}</span>
            </label>
            <button
              type="button"
              onClick={() => setShowSampleTable((v) => !v)}
              className="text-[var(--color-md-primary)] underline"
            >
              {t("supplier_portal.migrations.sample_values")} ({sampleRows.length})
            </button>
          </div>

          {showSampleTable && sampleRows.length > 0 && (
            <div className="max-h-56 overflow-auto rounded border border-[var(--border)] p-2 text-[11px] font-mono">
              {sampleRows.slice(0, 20).map((row) => (
                <div key={row.row_index} className="border-b border-[var(--border)]/40 py-1">
                  #{row.row_index}: {JSON.stringify(row.raw_data ?? {})}
                </div>
              ))}
            </div>
          )}
        </>
      )}
    </section>
  );
}
