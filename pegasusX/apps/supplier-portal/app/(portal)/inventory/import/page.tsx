"use client";

import Link from "next/link";
import { useCallback, useMemo, useState } from "react";
import { ApiError } from "@pegasusx/api-client";
import type {
  SupplierImportApplyResponse,
  SupplierImportMappingCandidate,
  SupplierImportSession,
} from "@pegasusx/types";
import { createSupplierApi } from "@/lib/api";
import { PortalSurface } from "../../_components/PortalSurface";

const api = createSupplierApi();

type WizardStep = "upload" | "mapping" | "review" | "done";

const sampleCsv = `product_id,warehouse_id,quantity_on_hand,reorder_threshold
sku-apple,wh-main,120,20
sku-pear,wh-main,80,15`;

function prettyStatus(value: string): string {
  return value.replaceAll("_", " ");
}

export default function InventoryImportPage() {
  const [step, setStep] = useState<WizardStep>("upload");
  const [csvText, setCsvText] = useState(sampleCsv);
  const [fileName, setFileName] = useState("inventory.csv");
  const [session, setSession] = useState<SupplierImportSession | null>(null);
  const [mappings, setMappings] = useState<SupplierImportMappingCandidate[]>([]);
  const [applyResult, setApplyResult] = useState<SupplierImportApplyResponse | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const sessionStatus = session?.status?.toUpperCase() ?? "";

  const canAdvanceMapping = useMemo(
    () => sessionStatus === "DISCOVERED" || sessionStatus === "MAPPING_REQUIRED",
    [sessionStatus],
  );

  const runWizard = useCallback(async () => {
    setSubmitting(true);
    setError(null);
    setApplyResult(null);
    try {
      const body = csvText.trim();
      if (!body) {
        throw new Error("csv_required");
      }
      const bytes = new TextEncoder().encode(body).length;
      const created = await api.createSupplierImportSession(fileName, bytes, crypto.randomUUID());
      const ingested = await api.ingestSupplierImportSession(created.session_id, body, crypto.randomUUID());
      const loaded = await api.getSupplierImportSession(created.session_id);
      const mapping = await api.getSupplierImportMapping(created.session_id);
      setSession({ ...loaded, status: ingested.status || loaded.status });
      setMappings(mapping.mapping_json?.mappings ?? []);
      setStep("mapping");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "import_wizard_failed");
    } finally {
      setSubmitting(false);
    }
  }, [csvText, fileName]);

  async function approveSession() {
    if (!session?.session_id) return;
    setSubmitting(true);
    setError(null);
    try {
      await api.approveSupplierImportSession(session.session_id, crypto.randomUUID());
      const refreshed = await api.getSupplierImportSession(session.session_id);
      setSession(refreshed);
      setStep("review");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "import_approve_failed");
    } finally {
      setSubmitting(false);
    }
  }

  async function applySession() {
    if (!session?.session_id) return;
    setSubmitting(true);
    setError(null);
    try {
      const result = await api.applySupplierImportSession(session.session_id, crypto.randomUUID());
      setApplyResult(result);
      const refreshed = await api.getSupplierImportSession(session.session_id);
      setSession(refreshed);
      setStep("done");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "import_apply_failed");
    } finally {
      setSubmitting(false);
    }
  }

  function onFileSelect(event: React.ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) return;
    setFileName(file.name);
    const reader = new FileReader();
    reader.onload = () => {
      if (typeof reader.result === "string") {
        setCsvText(reader.result);
      }
    };
    reader.readAsText(file);
  }

  return (
    <PortalSurface
      title="Inventory import wizard"
      description="Upload CSV, review column mapping, approve, and apply to warehouse inventory."
      error={error}
    >
      <div className="mb-4 flex flex-wrap gap-2 md-typescale-label-large">
        {(["upload", "mapping", "review", "done"] as WizardStep[]).map((item) => (
          <span
            key={item}
            className={`md-chip px-3 py-1 ${step === item ? "md-btn-filled" : "md-btn-outlined"}`}
          >
            {prettyStatus(item)}
          </span>
        ))}
      </div>

      {step === "upload" ? (
        <>
          <p className="mb-4 md-typescale-body-medium text-[var(--color-md-outline)]">
            Required columns: <code>product_id</code>, <code>warehouse_id</code>,{" "}
            <code>quantity_on_hand</code>. Optional: <code>reorder_threshold</code>. Warehouse ids
            must exist in{" "}
            <Link href="/topology" className="text-[var(--color-md-primary)] underline">
              topology
            </Link>
            .
          </p>

          <label className="md-btn md-btn-outlined md-typescale-label-large px-4 py-2 inline-block cursor-pointer mb-4">
            Choose CSV file
            <input type="file" accept=".csv,.tsv,text/csv" className="hidden" onChange={onFileSelect} />
          </label>

          <textarea
            className="md-input-outlined w-full min-h-48 font-mono text-sm p-3"
            value={csvText}
            onChange={(event) => setCsvText(event.target.value)}
            disabled={submitting}
          />

          <div className="flex flex-wrap gap-3 mt-4">
            <button
              type="button"
              className="md-btn md-btn-filled md-typescale-label-large px-4 py-2"
              disabled={submitting || csvText.trim() === ""}
              onClick={() => void runWizard()}
            >
              {submitting ? "Processing…" : "Upload and map columns"}
            </button>
            <Link href="/inventory" className="md-btn md-btn-outlined md-typescale-label-large px-4 py-2">
              Back to inventory
            </Link>
          </div>
        </>
      ) : null}

      {step === "mapping" && session ? (
        <div className="md-card p-4 md-typescale-body-medium">
          <p>
            Session <code>{session.session_id}</code> — status{" "}
            <strong>{prettyStatus(session.status)}</strong>
            {session.total_rows ? ` · ${session.total_rows} rows staged` : null}
          </p>
          {mappings.length === 0 ? (
            <p className="mt-3 text-[var(--color-md-outline)]">No mapping suggestions returned.</p>
          ) : (
            <table className="w-full mt-4 text-sm">
              <thead>
                <tr className="text-left border-b border-[var(--color-md-outline-variant)]">
                  <th className="py-2 pr-4">Source column</th>
                  <th className="py-2 pr-4">Target field</th>
                  <th className="py-2">Confidence</th>
                </tr>
              </thead>
              <tbody>
                {mappings.map((row) => (
                  <tr key={`${row.source_column}-${row.target_field}`} className="border-b border-[var(--color-md-outline-variant)]">
                    <td className="py-2 pr-4 font-mono">{row.source_column}</td>
                    <td className="py-2 pr-4 font-mono">{row.target_field}</td>
                    <td className="py-2">{(row.confidence * 100).toFixed(0)}%</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
          <div className="flex flex-wrap gap-3 mt-6">
            <button
              type="button"
              className="md-btn md-btn-filled md-typescale-label-large px-4 py-2"
              disabled={submitting || !canAdvanceMapping}
              onClick={() => void approveSession()}
            >
              {submitting ? "Approving…" : "Approve mapping"}
            </button>
            <button
              type="button"
              className="md-btn md-btn-outlined md-typescale-label-large px-4 py-2"
              disabled={submitting}
              onClick={() => setStep("upload")}
            >
              Start over
            </button>
          </div>
        </div>
      ) : null}

      {step === "review" && session ? (
        <div className="md-card p-4 md-typescale-body-medium">
          <p>
            Session <code>{session.session_id}</code> is <strong>APPROVED</strong>. Apply will write
            valid rows to inventory levels.
          </p>
          <div className="flex flex-wrap gap-3 mt-6">
            <button
              type="button"
              className="md-btn md-btn-filled md-typescale-label-large px-4 py-2"
              disabled={submitting}
              onClick={() => void applySession()}
            >
              {submitting ? "Applying…" : "Apply to inventory"}
            </button>
          </div>
        </div>
      ) : null}

      {step === "done" && applyResult ? (
        <div className="md-card p-4 mt-2 md-typescale-body-medium">
          <p>
            Applied <strong>{applyResult.applied_rows}</strong> rows across{" "}
            <strong>{applyResult.affected_warehouses}</strong> warehouse(s).
          </p>
          {applyResult.created_products ? (
            <p className="mt-2">Created products: {applyResult.created_products}</p>
          ) : null}
          <Link href="/inventory" className="md-btn md-btn-tonal md-typescale-label-large px-4 py-2 mt-4 inline-block">
            View inventory
          </Link>
        </div>
      ) : null}
    </PortalSurface>
  );
}
