"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { Camera, Loader2 } from "lucide-react";
import {
  claimTypeNeedsPhoto,
  fileOrderClaim,
  listOrderClaims,
  uploadClaimPhoto,
  type RetailerClaim,
} from "../lib/api";
import type { Order } from "../lib/types";

const CLAIM_TYPES = [
  "CONCEALED_DAMAGE",
  "DAMAGED",
  "MISSING",
  "TAMPER",
  "TEMPERATURE",
  "OTHER",
] as const;

type Props = {
  order: Order;
  onFiled?: (claim: RetailerClaim) => void;
};

export function FileClaimPanel({ order, onFiled }: Props) {
  const [claimType, setClaimType] = useState<string>("CONCEALED_DAMAGE");
  const [description, setDescription] = useState("");
  const [qtyBySku, setQtyBySku] = useState<Record<string, number>>({});
  const [photoUrl, setPhotoUrl] = useState("");
  const [preview, setPreview] = useState<string | null>(null);
  const [uploading, setUploading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successId, setSuccessId] = useState<string | null>(null);
  const [existing, setExisting] = useState<RetailerClaim[]>([]);

  const items = order.items ?? [];
  const needsPhoto = claimTypeNeedsPhoto(claimType);
  const selectedTotal = useMemo(
    () => Object.values(qtyBySku).reduce((s, n) => s + n, 0),
    [qtyBySku],
  );

  const refreshExisting = useCallback(async () => {
    try {
      const res = await listOrderClaims(order.order_id);
      if (!res.ok) {
        setExisting([]);
        return;
      }
      const body = (await res.json()) as { claims?: RetailerClaim[] };
      setExisting(body.claims ?? []);
    } catch {
      setExisting([]);
    }
  }, [order.order_id]);

  useEffect(() => {
    void refreshExisting();
  }, [refreshExisting]);

  const onPickFile = async (file: File | null) => {
    if (!file) return;
    setError(null);
    setUploading(true);
    try {
      setPreview(URL.createObjectURL(file));
      const url = await uploadClaimPhoto(file, order.order_id);
      setPhotoUrl(url);
    } catch (e) {
      setPhotoUrl("");
      setError(e instanceof Error ? e.message : "Photo upload failed");
    } finally {
      setUploading(false);
    }
  };

  const submit = async () => {
    setError(null);
    setSuccessId(null);
    if (selectedTotal <= 0) {
      setError("Select at least one item quantity.");
      return;
    }
    if (needsPhoto && !photoUrl.trim()) {
      setError("Photo required for this claim type.");
      return;
    }
    setSubmitting(true);
    try {
      const line_items = Object.entries(qtyBySku)
        .filter(([, q]) => q > 0)
        .map(([sku, quantity]) => ({
          sku,
          quantity,
          reason: claimType === "MISSING" ? "MISSING" : "DAMAGED",
        }));
      const evidences = photoUrl.trim()
        ? [
            {
              evidence_type: "PHOTO",
              uri: photoUrl.trim(),
              mime_type: "image/jpeg",
            },
          ]
        : [];
      const res = await fileOrderClaim(
        order.order_id,
        {
          claim_type: claimType,
          description,
          line_items,
          evidences,
        },
        `retailer-claim:${order.order_id}:${Date.now()}`,
      );
      if (!res.ok) {
        const err = await res.json().catch(() => null);
        throw new Error(
          err?.error || err?.message || `Claim failed (${res.status})`,
        );
      }
      const claim = (await res.json()) as RetailerClaim;
      setSuccessId(claim.claim_id);
      onFiled?.(claim);
      await refreshExisting();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Claim failed");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface-subtle)] p-5 space-y-4">
      <div>
        <h3 className="md-typescale-title-small font-light text-[var(--desk-text-primary)]">
          File claim
        </h3>
        <p className="md-typescale-body-small text-[var(--desk-text-tertiary)] mt-1">
          Within 48 hours of delivery. Amounts use your order prices.
        </p>
      </div>

      <label className="block">
        <span className="md-typescale-label-small text-[var(--desk-text-tertiary)]">
          Claim type
        </span>
        <select
          value={claimType}
          onChange={(e) => setClaimType(e.target.value)}
          className="mt-1 w-full rounded-xl border border-[var(--desk-border)] bg-[var(--desk-surface)] px-3 py-2 text-sm"
        >
          {CLAIM_TYPES.map((t) => (
            <option key={t} value={t}>
              {t.replaceAll("_", " ")}
            </option>
          ))}
        </select>
      </label>

      <div className="space-y-2">
        <span className="md-typescale-label-small text-[var(--desk-text-tertiary)]">
          Items
        </span>
        {items.length === 0 ? (
          <p className="text-sm text-[var(--desk-text-tertiary)]">
            No line items on this order snapshot.
          </p>
        ) : (
          items.map((item) => {
            const sku = item.sku_id || item.line_item_id;
            const qty = qtyBySku[sku] ?? 0;
            return (
              <div
                key={sku}
                className="flex items-center justify-between gap-3 rounded-xl border border-[var(--desk-border)] bg-[var(--desk-surface)] px-3 py-2"
              >
                <div className="min-w-0">
                  <p className="text-sm font-medium truncate">
                    {item.sku_name || sku}
                  </p>
                  <p className="text-xs text-[var(--desk-text-tertiary)]">
                    SKU {sku} · ordered {item.quantity}
                  </p>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  <button
                    type="button"
                    className="portal-btn portal-btn--ghost h-8 w-8 rounded-lg"
                    disabled={qty <= 0 || submitting}
                    onClick={() =>
                      setQtyBySku((m) => ({ ...m, [sku]: Math.max(0, qty - 1) }))
                    }
                  >
                    −
                  </button>
                  <span className="w-6 text-center text-sm font-semibold tabular-nums">
                    {qty}
                  </span>
                  <button
                    type="button"
                    className="portal-btn portal-btn--ghost h-8 w-8 rounded-lg"
                    disabled={qty >= item.quantity || submitting}
                    onClick={() =>
                      setQtyBySku((m) => ({
                        ...m,
                        [sku]: Math.min(item.quantity, qty + 1),
                      }))
                    }
                  >
                    +
                  </button>
                </div>
              </div>
            );
          })
        )}
      </div>

      <div className="space-y-2">
        <span className="md-typescale-label-small text-[var(--desk-text-tertiary)]">
          Photo proof
        </span>
        <label className="flex cursor-pointer items-center justify-center gap-2 rounded-xl border border-dashed border-[var(--desk-border-strong)] bg-[var(--desk-surface)] px-4 py-3 text-sm">
          {uploading ? (
            <>
              <Loader2 size={16} className="animate-spin" />
              Uploading…
            </>
          ) : (
            <>
              <Camera size={16} />
              {photoUrl ? "Photo ready — change" : "Choose photo"}
            </>
          )}
          <input
            type="file"
            accept="image/*"
            className="hidden"
            disabled={uploading || submitting}
            onChange={(e) => void onPickFile(e.target.files?.[0] ?? null)}
          />
        </label>
        {preview && (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={preview}
            alt="Claim proof"
            className="max-h-40 w-full rounded-xl object-cover"
          />
        )}
        {needsPhoto && (
          <p className="text-xs text-[var(--desk-text-tertiary)]">
            Required for this claim type.
          </p>
        )}
      </div>

      <label className="block">
        <span className="md-typescale-label-small text-[var(--desk-text-tertiary)]">
          What happened?
        </span>
        <textarea
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          rows={3}
          className="mt-1 w-full rounded-xl border border-[var(--desk-border)] bg-[var(--desk-surface)] px-3 py-2 text-sm"
          disabled={submitting}
        />
      </label>

      {error && (
        <p className="text-sm text-red-600 font-medium">{error}</p>
      )}
      {successId && (
        <p className="text-sm text-green-700 font-medium">
          Claim filed: {successId}
        </p>
      )}

      {existing.length > 0 && (
        <div className="space-y-1">
          <span className="md-typescale-label-small text-[var(--desk-text-tertiary)]">
            Previous claims
          </span>
          {existing.map((c) => (
            <p key={c.claim_id} className="text-xs text-[var(--desk-text-secondary)]">
              {c.claim_type} · {c.status}
              {c.amount_minor != null
                ? ` · ${c.amount_minor} ${c.currency ?? "UZS"}`
                : ""}
            </p>
          ))}
        </div>
      )}

      <button
        type="button"
        disabled={submitting || uploading || selectedTotal <= 0}
        onClick={() => void submit()}
        className="portal-btn portal-btn--primary h-11 w-full rounded-xl font-light"
      >
        {submitting ? (
          <Loader2 size={16} className="animate-spin" />
        ) : (
          "Submit claim"
        )}
      </button>
    </div>
  );
}
