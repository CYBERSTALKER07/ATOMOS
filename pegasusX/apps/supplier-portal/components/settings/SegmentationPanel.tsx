"use client";

import React, { useCallback, useEffect, useState } from "react";
import type { RetailerSegmentRow, SkuClassRow } from "@pegasusx/types";
import { createSupplierApi } from "@/lib/api";

const api = createSupplierApi();

const SEGMENTS = ["A", "B", "C"];
const VELOCITY = ["A", "B", "C"];

export function SegmentationPanel() {
  const [retailers, setRetailers] = useState<RetailerSegmentRow[]>([]);
  const [skuClasses, setSkuClasses] = useState<SkuClassRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [bootstrapping, setBootstrapping] = useState(false);
  const [acting, setActing] = useState<string | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    setError(null);
    Promise.all([api.listRetailerSegments(), api.listSkuClasses()])
      .then(([r, s]) => {
        setRetailers(r.retailers ?? []);
        setSkuClasses(s.sku_classes ?? []);
      })
      .catch((err) => setError(err instanceof Error ? err.message : "load_segmentation_failed"))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const bootstrap = async () => {
    setBootstrapping(true);
    setError(null);
    try {
      await api.bootstrapSegmentation();
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "bootstrap_failed");
    } finally {
      setBootstrapping(false);
    }
  };

  const updateRetailer = async (retailerId: string, segment: string) => {
    setActing(`retailer:${retailerId}`);
    try {
      await api.setRetailerSegment(retailerId, { segment, reason: "portal override" });
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "update_retailer_failed");
    } finally {
      setActing(null);
    }
  };

  const updateSku = async (sku: string, velocityClass: string) => {
    setActing(`sku:${sku}`);
    try {
      await api.setSkuClass(sku, { velocity_class: velocityClass });
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "update_sku_failed");
    } finally {
      setActing(null);
    }
  };

  if (loading) {
    return <p className="text-sm text-[var(--color-md-outline)]">Loading segmentation…</p>;
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center gap-3">
        <button
          type="button"
          className="md-btn md-btn-filled"
          disabled={bootstrapping}
          onClick={() => void bootstrap()}
        >
          {bootstrapping ? "Bootstrapping…" : "Run bootstrap"}
        </button>
        <p className="text-sm text-[var(--color-md-outline)]">
          Bootstrap assigns segments from order volume, credit, and claims. Manual overrides are preserved.
        </p>
      </div>

      {error ? <p className="text-sm text-[var(--color-md-error)]">{error}</p> : null}

      <section>
        <h2 className="md-typescale-title-medium mb-3">Retailer segments</h2>
        {retailers.length === 0 ? (
          <p className="text-sm text-[var(--color-md-outline)]">No retailer segments yet. Run bootstrap after orders exist.</p>
        ) : (
          <ul className="md-card divide-y divide-[var(--color-md-outline-variant)]">
            {retailers.map((row) => (
              <li key={row.retailer_id} className="flex flex-wrap items-center gap-3 p-3 md-typescale-body-medium">
                <span className="font-mono text-xs">{row.retailer_id}</span>
                <select
                  className="md-input md-input-sm"
                  value={row.segment}
                  disabled={acting === `retailer:${row.retailer_id}`}
                  onChange={(e) => void updateRetailer(row.retailer_id, e.target.value)}
                >
                  {SEGMENTS.map((s) => (
                    <option key={s} value={s}>{s}</option>
                  ))}
                </select>
                {row.reason ? (
                  <span className="text-xs text-[var(--color-md-outline)]">{row.reason}</span>
                ) : null}
              </li>
            ))}
          </ul>
        )}
      </section>

      <section>
        <h2 className="md-typescale-title-medium mb-3">SKU velocity classes</h2>
        {skuClasses.length === 0 ? (
          <p className="text-sm text-[var(--color-md-outline)]">No SKU classes yet.</p>
        ) : (
          <ul className="md-card divide-y divide-[var(--color-md-outline-variant)]">
            {skuClasses.map((row) => (
              <li key={row.sku} className="flex flex-wrap items-center gap-3 p-3 md-typescale-body-medium">
                <span className="font-mono text-xs">{row.sku}</span>
                <select
                  className="md-input md-input-sm"
                  value={row.velocity_class}
                  disabled={acting === `sku:${row.sku}`}
                  onChange={(e) => void updateSku(row.sku, e.target.value)}
                >
                  {VELOCITY.map((v) => (
                    <option key={v} value={v}>{v}</option>
                  ))}
                </select>
                {row.strategic_flag ? <span className="md-chip h-6 text-xs">Strategic</span> : null}
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
