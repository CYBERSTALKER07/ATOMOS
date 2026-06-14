"use client";

import Link from "next/link";
import { useState } from "react";
import { ApiError } from "@pegasusx/api-client";
import { createSupplierApi } from "@/lib/api";
import type { SupplierInventoryImportResult } from "@pegasusx/types";
import { PortalSurface } from "../../_components/PortalSurface";

const api = createSupplierApi();

const sampleCsv = `product_id,warehouse_id,quantity_on_hand,reorder_threshold
sku-apple,wh-main,120,20
sku-pear,wh-main,80,15`;

export default function InventoryImportPage() {
  const [csvText, setCsvText] = useState(sampleCsv);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<SupplierInventoryImportResult | null>(null);

  async function submitImport() {
    setSubmitting(true);
    setError(null);
    setResult(null);
    try {
      const response = await api.importSupplierInventoryCSV(csvText, crypto.randomUUID());
      setResult(response);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "inventory_import_failed");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <PortalSurface
      title="Inventory CSV import"
      description="Bulk upsert inventory levels for warehouses in your topology."
      error={error}
    >
      <p className="mb-4 md-typescale-body-medium text-[var(--color-md-outline)]">
        Required columns: <code>product_id</code>, <code>warehouse_id</code>, <code>quantity_on_hand</code>.
        Optional: <code>reorder_threshold</code>. Warehouse ids must exist in{" "}
        <Link href="/topology" className="text-[var(--color-md-primary)] underline">
          topology
        </Link>
        .
      </p>

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
          onClick={() => void submitImport()}
        >
          {submitting ? "Importing…" : "Apply import"}
        </button>
        <Link href="/inventory" className="md-btn md-btn-outlined md-typescale-label-large px-4 py-2">
          Back to inventory
        </Link>
      </div>

      {result ? (
        <div className="md-card p-4 mt-6 md-typescale-body-medium">
          <p>
            Applied <strong>{result.applied}</strong> rows; skipped <strong>{result.skipped}</strong>.
          </p>
          {result.errors && result.errors.length > 0 ? (
            <ul className="mt-3 list-disc pl-5 text-[var(--color-md-error)]">
              {result.errors.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          ) : null}
        </div>
      ) : null}
    </PortalSurface>
  );
}
