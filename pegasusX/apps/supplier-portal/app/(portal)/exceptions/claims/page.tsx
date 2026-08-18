"use client";

import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import type { Route } from "next";
import { useCallback, useEffect, useState } from "react";
import type {
  Claim,
  ClaimSettlementMode,
  ClaimSettlementResult,
} from "@pegasusx/types";
import { CLAIM_SETTLEMENT_MODES } from "@pegasusx/types";
import { createSupplierApi } from "@/lib/api";
import StatusBadge from "@/components/StatusBadge";
import { PageChrome } from "@/components/PageChrome";
import { moneyCurrency } from "@pegasusx/api-client";

const api = createSupplierApi();

export default function ClaimsQueuePage() {
  const t = usePortalT();
  const [claims, setClaims] = useState<Claim[]>([]);
  const [statusFilter, setStatusFilter] = useState("OPEN");
  const [settlementMode, setSettlementMode] = useState<ClaimSettlementMode>("LEDGER_ONLY");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [lastSettlement, setLastSettlement] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const body = await api.listSupplierClaims({
        status: statusFilter || undefined,
        limit: 50,
      });
      setClaims(body.claims ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.load_claims_failed"));
    } finally {
      setLoading(false);
    }
  }, [statusFilter]);

  useEffect(() => {
    void load();
  }, [load]);

  async function approve(claimId: string) {
    setBusyId(claimId);
    setLastSettlement(null);
    try {
      const body = await api.approveClaim(claimId, {
        resolution_note: "approved_via_supplier_portal",
        settlement_mode: settlementMode,
        skip_gateway_refund: settlementMode !== "GATEWAY_REFUND",
      });
      const settlement = body.settlement as ClaimSettlementResult | undefined;
      if (settlement) {
        setLastSettlement(
          `${settlement.mode} · ${settlement.amount_minor} · refund=${Boolean(settlement.gateway_refunded)} · id=${settlement.chargeback_id ?? "—"}`,
        );
      }
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.approve_failed"));
    } finally {
      setBusyId(null);
    }
  }

  async function reject(claimId: string) {
    setBusyId(claimId);
    try {
      await api.rejectClaim(claimId, {
        resolution_note: "rejected_via_supplier_portal",
      });
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.reject_failed"));
    } finally {
      setBusyId(null);
    }
  }

  return (
    <PageChrome
      title={t("supplier_portal.exceptions.claims.text.claims_queue")}
      description={t("supplier_portal.residual.text.post_delivery_damage_missing_and_os_and_d_claims_choose_settleme")}
      icon="warning"
      loading={loading}
      error={error}
      empty={!loading && claims.length === 0}
      emptyMessage={t("supplier_portal.residual.text.no_claims_in_this_filter_retailer_post_delivery_claims_appear_he")}
    >
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <label className="md-typescale-label-large flex items-center gap-2">
          Status
          <select
            className="md-btn md-btn-outlined px-3 py-1"
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
          >
            <option value="OPEN">OPEN</option>
            <option value="UNDER_REVIEW">UNDER_REVIEW</option>
            <option value="RESOLVED">RESOLVED</option>
            <option value="REJECTED">REJECTED</option>
            <option value="">ALL</option>
          </select>
        </label>
        <label className="md-typescale-label-large flex items-center gap-2">
          Settlement
          <select
            className="md-btn md-btn-outlined px-3 py-1 max-w-[220px]"
            value={settlementMode}
            onChange={(e) => setSettlementMode(e.target.value as ClaimSettlementMode)}
          >
            {CLAIM_SETTLEMENT_MODES.map((m) => (
              <option key={m.value} value={m.value}>
                {m.label}
              </option>
            ))}
          </select>
        </label>
        <button type="button" className="md-btn md-btn-outlined px-3 py-1" onClick={() => void load()}>
          Refresh
        </button>
        {lastSettlement ? (
          <span className="md-typescale-body-small text-[var(--color-md-outline)]">
            Last settle: {lastSettlement}
          </span>
        ) : null}
      </div>
      <p className="mb-4 text-xs text-[var(--color-md-outline)]">
        {CLAIM_SETTLEMENT_MODES.find((m) => m.value === settlementMode)?.hint}
        {" · "}
        <Link href={"/chargebacks/claims" as Route} className="underline text-[var(--color-md-primary)]">
          Claim chargebacks ledger
        </Link>
      </p>

      <ul className="md-card divide-y divide-[var(--color-md-outline-variant)]">
        {claims.map((c) => (
          <li key={c.claim_id} className="p-4 md-typescale-body-medium">
            <div className="flex flex-wrap items-center gap-2">
              <span className="md-chip h-6 text-xs">{c.claim_type}</span>
              <StatusBadge state={c.status} />
              <span className="font-mono text-[var(--color-md-primary)]">{c.claim_id}</span>
            </div>
            <p className="mt-1 text-sm">
              Order <span className="font-mono">{c.order_id}</span> · Retailer{" "}
              <span className="font-mono">{c.retailer_id}</span>
            </p>
            <p className="mt-1 text-sm text-[var(--color-md-outline)]">
              Amount {c.amount_minor ?? 0} {moneyCurrency(c.currency)} ·{" "}
              {c.created_at ? new Date(c.created_at).toLocaleString() : "—"}
            </p>
            {c.description ? <p className="mt-2">{c.description}</p> : null}
            {c.line_items && c.line_items.length > 0 ? (
              <ul className="mt-2 text-sm text-[var(--color-md-outline)]">
                {c.line_items.map((li) => (
                  <li key={`${c.claim_id}-${li.sku}`}>
                    {li.sku} × {li.quantity}
                    {li.amount_minor != null ? ` · ${li.amount_minor}` : ""}
                    {li.reason ? ` · ${li.reason}` : ""}
                  </li>
                ))}
              </ul>
            ) : null}
            {c.evidences && c.evidences.length > 0 ? (
              <div className="mt-2 flex flex-wrap gap-2">
                {c.evidences.map((e, i) => (
                  <a
                    key={`${c.claim_id}-ev-${i}`}
                    href={e.uri}
                    target="_blank"
                    rel="noreferrer"
                    className="text-[var(--color-md-primary)] underline text-sm"
                  >
                    Evidence {i + 1}
                  </a>
                ))}
              </div>
            ) : null}
            {(c.status === "OPEN" || c.status === "UNDER_REVIEW") && (
              <div className="mt-3 flex flex-wrap gap-2">
                <button
                  type="button"
                  className="md-btn md-btn-filled px-3 py-1"
                  disabled={busyId === c.claim_id}
                  onClick={() => void approve(c.claim_id)}
                >
                  Approve ({CLAIM_SETTLEMENT_MODES.find((m) => m.value === settlementMode)?.label})
                </button>
                <button
                  type="button"
                  className="md-btn md-btn-outlined px-3 py-1"
                  disabled={busyId === c.claim_id}
                  onClick={() => void reject(c.claim_id)}
                >
                  Reject
                </button>
              </div>
            )}
          </li>
        ))}
      </ul>

      <div className="mt-4 flex flex-wrap gap-4">
        <Link href={"/exceptions" as Route} className="text-[var(--color-md-primary)] underline">
          All exceptions
        </Link>
        <Link href={"/credit/collections" as Route} className="text-[var(--color-md-primary)] underline">
          Credit collections
        </Link>
        <Link href={"/chargebacks/claims" as Route} className="text-[var(--color-md-primary)] underline">
          Claim chargebacks
        </Link>
      </div>
    </PageChrome>
  );
}
