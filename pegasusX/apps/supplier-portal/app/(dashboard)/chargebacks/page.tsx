"use client";

import { useState } from "react";
import { PageChrome } from "@/components/PageChrome";
import { PageSection } from "@/components/PageSection";
import { supplierFetch } from "@/lib/auth";
import { RefreshCw, AlertTriangle, CheckCircle2 } from "lucide-react";

export default function ChargebacksPage() {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  // Chargeback state
  const [orderId, setOrderId] = useState("");
  const [retailerId, setRetailerId] = useState("");
  const [gateway, setGateway] = useState("ADYEN");
  const [amount, setAmount] = useState("");
  const [currency, setCurrency] = useState("UZS");
  const [chargebackMessage, setChargebackMessage] = useState<string | null>(null);

  // Reversal state
  const [sessionId, setSessionId] = useState("");
  const [reversalMessage, setReversalMessage] = useState<string | null>(null);

  const gateways = ["ADYEN", "GLOBAL_PAY", "STRIPE", "PAYME", "CLICK", "CASH"];

  const submitChargeback = async (e: React.FormEvent) => {
    e.preventDefault();
    const amountMinor = parseInt(amount, 10);
    if (isNaN(amountMinor)) {
      setError("Amount must be a number.");
      return;
    }
    setBusy(true);
    setError(null);
    setChargebackMessage(null);
    try {
      const idempotencyKey = `cb_${orderId}_${Date.now()}`;
      const res = await supplierFetch("/v1/payment/chargeback", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": idempotencyKey,
        },
        body: JSON.stringify({
          order_id: orderId,
          retailer_id: retailerId,
          gateway,
          amount: amountMinor,
          currency,
        }),
      });

      if (!res.ok) throw new Error("Failed to record chargeback");
      const data = await res.json();
      setChargebackMessage(`Chargeback recorded (${data.status || 'Success'}).`);
      setOrderId("");
      setRetailerId("");
      setAmount("");
    } catch (err: any) {
      setError(err.message || "An error occurred");
    } finally {
      setBusy(false);
    }
  };

  const submitReversal = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    setReversalMessage(null);
    try {
      const idempotencyKey = `rev_${sessionId}_${Date.now()}`;
      const res = await supplierFetch("/v1/payment/chargeback/reversal", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": idempotencyKey,
        },
        body: JSON.stringify({ session_id: sessionId }),
      });

      if (!res.ok) throw new Error("Failed to record reversal");
      const data = await res.json();
      setReversalMessage(`Reversal recorded (${data.status || 'Success'}).`);
      setSessionId("");
    } catch (err: any) {
      setError(err.message || "An error occurred");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="min-h-full p-6 md:p-8" style={{ background: "var(--desk-canvas)" }}>
      <PageChrome
        icon="warning"
        title="Chargebacks"
        description="Record payment disputes and reversals against the durable finance ledger. Logistics claim chargebacks: Finance → Claim chargebacks."
      >
        <div className="mb-6">
          <a
            href="/chargebacks/claims"
            className="text-sm underline text-[var(--desk-accent)]"
          >
            View claim chargebacks ledger →
          </a>
        </div>
        {error && (
          <div className="mb-6 flex items-center gap-2 p-3 rounded-xl border bg-[var(--desk-warning)]/10 text-[var(--desk-warning)] border-[var(--desk-warning)]/30">
            <AlertTriangle size={16} />
            <span className="md-typescale-body-small">{error}</span>
          </div>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
          <PageSection title="Record Chargeback">
            <div className="p-6 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl shadow-[var(--shadow-sm)]">
              <form onSubmit={submitChargeback} className="space-y-4">
                <div>
                  <label className="block text-sm text-[var(--desk-text-secondary)] mb-1">Order ID</label>
                  <input
                    type="text"
                    value={orderId}
                    onChange={(e) => setOrderId(e.target.value)}
                    className="w-full h-10 px-3 bg-[var(--desk-canvas)] border border-[var(--desk-border)] rounded-lg outline-none focus:border-[var(--desk-accent)] text-[var(--desk-text-primary)]"
                    placeholder="Enter Order ID"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm text-[var(--desk-text-secondary)] mb-1">Retailer ID</label>
                  <input
                    type="text"
                    value={retailerId}
                    onChange={(e) => setRetailerId(e.target.value)}
                    className="w-full h-10 px-3 bg-[var(--desk-canvas)] border border-[var(--desk-border)] rounded-lg outline-none focus:border-[var(--desk-accent)] text-[var(--desk-text-primary)]"
                    placeholder="Enter Retailer ID"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm text-[var(--desk-text-secondary)] mb-1">Gateway</label>
                  <select
                    value={gateway}
                    onChange={(e) => setGateway(e.target.value)}
                    className="w-full h-10 px-3 bg-[var(--desk-canvas)] border border-[var(--desk-border)] rounded-lg outline-none focus:border-[var(--desk-accent)] text-[var(--desk-text-primary)]"
                  >
                    {gateways.map(g => <option key={g} value={g}>{g}</option>)}
                  </select>
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm text-[var(--desk-text-secondary)] mb-1">Amount (minor units)</label>
                    <input
                      type="number"
                      value={amount}
                      onChange={(e) => setAmount(e.target.value)}
                      className="w-full h-10 px-3 bg-[var(--desk-canvas)] border border-[var(--desk-border)] rounded-lg outline-none focus:border-[var(--desk-accent)] text-[var(--desk-text-primary)]"
                      placeholder="e.g. 1000"
                      required
                    />
                  </div>
                  <div>
                    <label className="block text-sm text-[var(--desk-text-secondary)] mb-1">Currency</label>
                    <input
                      type="text"
                      value={currency}
                      onChange={(e) => setCurrency(e.target.value)}
                      className="w-full h-10 px-3 bg-[var(--desk-canvas)] border border-[var(--desk-border)] rounded-lg outline-none focus:border-[var(--desk-accent)] text-[var(--desk-text-primary)] uppercase"
                      placeholder="UZS"
                      required
                    />
                  </div>
                </div>
                
                <div className="pt-2">
                  <button
                    type="submit"
                    disabled={busy || !orderId || !retailerId || !amount}
                    className="w-full h-11 flex items-center justify-center gap-2 bg-[var(--desk-accent)] text-white rounded-xl font-medium disabled:opacity-50 disabled:cursor-not-allowed hover:bg-[var(--desk-accent)]/90 transition-colors"
                  >
                    {busy ? <RefreshCw size={18} className="animate-spin" /> : "Record Chargeback"}
                  </button>
                </div>

                {chargebackMessage && (
                  <div className="flex items-center gap-2 mt-4 text-[var(--desk-success)] text-sm font-medium">
                    <CheckCircle2 size={16} />
                    {chargebackMessage}
                  </div>
                )}
              </form>
            </div>
          </PageSection>

          <PageSection title="Reverse Chargeback">
            <div className="p-6 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl shadow-[var(--shadow-sm)]">
              <form onSubmit={submitReversal} className="space-y-4">
                <div>
                  <label className="block text-sm text-[var(--desk-text-secondary)] mb-1">Session ID</label>
                  <input
                    type="text"
                    value={sessionId}
                    onChange={(e) => setSessionId(e.target.value)}
                    className="w-full h-10 px-3 bg-[var(--desk-canvas)] border border-[var(--desk-border)] rounded-lg outline-none focus:border-[var(--desk-accent)] text-[var(--desk-text-primary)]"
                    placeholder="Enter Chargeback Session ID"
                    required
                  />
                </div>
                
                <div className="pt-2">
                  <button
                    type="submit"
                    disabled={busy || !sessionId}
                    className="w-full h-11 flex items-center justify-center gap-2 bg-[var(--desk-accent)] text-white rounded-xl font-medium disabled:opacity-50 disabled:cursor-not-allowed hover:bg-[var(--desk-accent)]/90 transition-colors"
                  >
                    {busy ? <RefreshCw size={18} className="animate-spin" /> : "Record Reversal"}
                  </button>
                </div>

                {reversalMessage && (
                  <div className="flex items-center gap-2 mt-4 text-[var(--desk-success)] text-sm font-medium">
                    <CheckCircle2 size={16} />
                    {reversalMessage}
                  </div>
                )}
              </form>
            </div>
          </PageSection>
        </div>
      </PageChrome>
    </div>
  );
}
