"use client";

import { useLiveData } from "../lib/hooks";
import type { LoyaltyLedgerResponse, LoyaltyTierView } from "@pegasusx/types";

export function LoyaltyCard() {
  const { data: tier, loading, error } = useLiveData<LoyaltyTierView>("/v1/retailer/loyalty/tier");
  const { data: ledger } = useLiveData<LoyaltyLedgerResponse>("/v1/retailer/loyalty/ledger");

  return (
    <div className="bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl p-4 shadow-[var(--shadow-sm)] mb-3">
      <p className="text-[10px] font-black uppercase tracking-widest text-[var(--desk-text-tertiary)] mb-2">
        Loyalty
      </p>
      {loading && !tier ? (
        <p className="text-sm text-[var(--desk-text-secondary)]">Loading loyalty…</p>
      ) : error ? (
        <p className="text-sm text-orange-700">{error.message || String(error)}</p>
      ) : !tier?.enrolled ? (
        <p className="text-sm text-[var(--desk-text-secondary)]">
          Not enrolled. No fake Bronze — the supplier has not configured a program, or you have no points yet.
        </p>
      ) : (
        <div className="space-y-1 text-sm text-[var(--desk-text-secondary)]">
          <p>
            {tier.tier || "Member"} · {tier.lifetime_points} lifetime · {tier.available_points} available
          </p>
          {tier.next_tier ? (
            <p>
              {tier.points_to_next} points to {tier.next_tier}
            </p>
          ) : null}
          <ul className="list-disc pl-5">
            {(ledger?.entries ?? []).slice(0, 8).map((e) => (
              <li key={e.ledger_id}>
                {e.points} pts · order {e.order_id.slice(0, 8)} · {e.created_at}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
