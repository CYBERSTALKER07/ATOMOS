"use client";

import { useState, useCallback, useEffect } from "react";
import {
  CreditCard,
  Banknote,
  X,
  Loader2,
  ExternalLink,
  ShieldCheck,
  CheckCircle2,
  AlertTriangle,
} from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import type { PaymentRequiredEvent } from "@pegasus/types";
import { useWsEvent, type WsMessage } from "../lib/ws";
import { apiFetch } from "../lib/auth";
import type { CardCheckoutResponse, PendingPaymentSession, PendingPaymentsResponse } from "../lib/types";

/* ── Types ── */

type PaymentEvent = Omit<PaymentRequiredEvent, "available_card_gateways"> & {
  available_card_gateways?: string[];
};

type PaymentState = "idle" | "choosing" | "processing" | "success" | "error";

function formatAmount(amount: number): string {
  return amount.toLocaleString("en-US").replace(/,/g, " ");
}

function toFiniteNumber(value: unknown, fallback: number): number {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  return fallback;
}

function sessionToPaymentEvent(session: PendingPaymentSession): PaymentEvent {
  const gateway = session.gateway || "GLOBAL_PAY";
  return {
    type: "PAYMENT_REQUIRED",
    order_id: session.order_id,
    invoice_id: session.invoice_id ?? null,
    session_id: session.session_id,
    amount: session.locked_amount,
    original_amount: session.locked_amount,
    payment_method: gateway === "CASH" ? "CASH" : "CARD",
    gateway: gateway as PaymentRequiredEvent["gateway"],
    currency: session.currency || "UZS",
    available_card_gateways: gateway === "CASH" ? [] : [gateway],
    message: "Pending payment requires completion.",
  };
}

function wsMessageToPaymentEvent(msg: WsMessage): PaymentEvent {
  const rawGateways = msg.available_card_gateways;
  const gateways = Array.isArray(rawGateways)
    ? rawGateways.filter((value): value is string => typeof value === "string")
    : undefined;

  const amount = toFiniteNumber(msg.amount, 0);
  const originalAmount = toFiniteNumber(msg.original_amount, amount);

  return {
    order_id: (msg.order_id as string) || "",
    type: "PAYMENT_REQUIRED",
    invoice_id: (msg.invoice_id as string | null | undefined) ?? null,
    session_id: (msg.session_id as string) || "",
    amount,
    original_amount: originalAmount,
    payment_method: (msg.payment_method as string) || "CARD",
    gateway: ((msg.gateway as string) || "GLOBAL_PAY") as PaymentRequiredEvent["gateway"],
    currency: (msg.currency as string) || "UZS",
    available_card_gateways: gateways,
    message: (msg.message as string | undefined) ?? "",
  };
}

/* ── Component ── */

export default function PaymentModal() {
  const [event, setEvent] = useState<PaymentEvent | null>(null);
  const [state, setState] = useState<PaymentState>("idle");
  const [error, setError] = useState<string | null>(null);
  const [checkoutUrl, setCheckoutUrl] = useState<string | null>(null);

  const openPaymentEvent = useCallback((msg: WsMessage) => {
    const evt = wsMessageToPaymentEvent(msg);
    setEvent(evt);
    setState("choosing");
    setError(null);
    setCheckoutUrl(null);
  }, []);

  useWsEvent(
    "PAYMENT_REQUIRED",
    useCallback((msg: WsMessage) => {
      openPaymentEvent(msg);
    }, [openPaymentEvent]),
  );

  useWsEvent(
    "GLOBAL_PAYNT_REQUIRED",
    useCallback((msg: WsMessage) => {
      openPaymentEvent(msg);
    }, [openPaymentEvent]),
  );

  useWsEvent(
    "SETTLEMENT_REQUIRED",
    useCallback((msg: WsMessage) => {
      openPaymentEvent(msg);
    }, [openPaymentEvent]),
  );

  useWsEvent(
    "DELIVERY_SESSION_UPDATED",
    useCallback((msg: WsMessage) => {
      const orderId = typeof msg.order_id === "string" ? msg.order_id : "";
      setEvent((prev) => {
        if (!prev || !orderId || prev.order_id !== orderId) {
          return prev;
        }

        const adjustedAmount = toFiniteNumber(msg.adjusted_amount, prev.amount);
        const originalAmount = toFiniteNumber(
          msg.original_amount,
          prev.original_amount || adjustedAmount,
        );

        return {
          ...prev,
          amount: adjustedAmount,
          original_amount: originalAmount,
          message:
            typeof msg.message === "string" && msg.message.length > 0
              ? msg.message
              : prev.message,
        };
      });
    }, []),
  );

  useWsEvent(
    "PAYMENT_SETTLED",
    useCallback(
      (msg: WsMessage) => {
        if (event && msg.order_id === event.order_id) {
          setState("success");
          setTimeout(() => {
            setEvent(null);
            setState("idle");
          }, 2000);
        }
      },
      [event],
    ),
  );

  useWsEvent(
    "GLOBAL_PAYNT_SETTLED",
    useCallback(
      (msg: WsMessage) => {
        if (event && msg.order_id === event.order_id) {
          setState("success");
          setTimeout(() => {
            setEvent(null);
            setState("idle");
          }, 2000);
        }
      },
      [event],
    ),
  );

  const dismiss = useCallback(() => {
    setEvent(null);
    setState("idle");
    setError(null);
    setCheckoutUrl(null);
  }, []);

  useEffect(() => {
    let cancelled = false;

    async function loadPendingPayments() {
      if (event) return;
      try {
        const res = await apiFetch("/v1/retailer/pending-payments");
        if (!res.ok) return;
        const data: PendingPaymentsResponse = await res.json();
        const pending = data.pending_payments?.[0];
        if (!cancelled && pending) {
          setEvent(sessionToPaymentEvent(pending));
          setState("choosing");
          setError(null);
          setCheckoutUrl(null);
        }
      } catch {
        // WebSocket delivery remains the primary realtime path.
      }
    }

    void loadPendingPayments();
    return () => {
      cancelled = true;
    };
  }, [event]);

  const handleCash = useCallback(async () => {
    if (!event) return;
    setState("processing");
    setError(null);
    try {
      const res = await apiFetch("/v1/order/cash-checkout", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": `retailer-cash-checkout:${event.order_id}`,
        },
        body: JSON.stringify({ order_id: event.order_id }),
      });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || "Cash checkout failed");
      }
      setState("success");
      setTimeout(dismiss, 2000);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Cash checkout failed");
      setState("choosing");
    }
  }, [event, dismiss]);

  const handleCard = useCallback(
    async (gateway: string) => {
      if (!event) return;
      setState("processing");
      setError(null);
      try {
        const res = await apiFetch("/v1/order/card-checkout", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "Idempotency-Key": `retailer-card-checkout:${event.order_id}:${gateway}`,
          },
          body: JSON.stringify({ order_id: event.order_id, gateway }),
        });
        if (!res.ok) {
          const text = await res.text();
          throw new Error(text || "Card checkout failed");
        }
        const data: CardCheckoutResponse = await res.json();
        if (data.payment_url) {
          setCheckoutUrl(data.payment_url);
          window.open(data.payment_url, "_blank", "noopener");
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : "Card checkout failed");
        setState("choosing");
      }
    },
    [event],
  );

  if (!event || state === "idle") return null;

  const gateways = event.available_card_gateways ?? [];
  const amended =
    event.original_amount && event.original_amount !== event.amount;

  return (
    <AnimatePresence>
      <div className="fixed inset-0 z-[120] flex items-center justify-center p-4">
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          className="absolute inset-0 bg-[#0a0a0a]/60 backdrop-blur-md"
          onClick={dismiss}
        />

        <motion.div
          initial={{ scale: 0.95, opacity: 0, y: 20 }}
          animate={{ scale: 1, opacity: 1, y: 0 }}
          exit={{ scale: 0.95, opacity: 0, y: 20 }}
          className="relative w-full max-w-md bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-3xl shadow-2xl overflow-hidden"
        >
          {state === "success" ? (
            <div className="flex flex-col items-center gap-4 py-16 px-6 text-center">
              <div className="w-20 h-20 rounded-full bg-[var(--desk-success)] flex items-center justify-center text-white shadow-xl">
                <CheckCircle2 size={40} />
              </div>
              <div>
                <h2 className="md-typescale-title-large font-light text-[var(--desk-text-primary)]">
                  Payment Settled
                </h2>
                <p className="md-typescale-body-medium text-[var(--desk-text-secondary)] mt-1">
                  Order #{event.order_id.slice(-8)} transition complete
                </p>
              </div>
            </div>
          ) : (
            <>
              {/* Header */}
              <div className="p-6 border-b border-[var(--desk-border)] flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-xl bg-[var(--desk-accent-soft)] text-[var(--desk-accent)] flex items-center justify-center">
                    <ShieldCheck size={22} />
                  </div>
                  <div>
                    <h2 className="md-typescale-title-large font-light text-[var(--desk-text-primary)]">
                      Payment Required
                    </h2>
                    <p className="text-[10px] font-light text-[var(--desk-text-tertiary)] uppercase tracking-widest">
                      Node ID: #{event.order_id.slice(-8)}
                    </p>
                  </div>
                </div>
                <button
                  onClick={dismiss}
                  className="w-10 h-10 rounded-full hover:bg-[var(--desk-surface-subtle)] flex items-center justify-center transition-colors"
                >
                  <X size={20} />
                </button>
              </div>

              <div className="p-6 space-y-6">
                {/* Amount */}
                <div className="p-6 rounded-2xl bg-[var(--desk-text-primary)] text-white shadow-xl relative overflow-hidden">
                  <div className="absolute right-0 top-0 w-24 h-24 bg-white opacity-5 rotate-12 translate-x-4 -translate-y-4" />
                  <p className="text-xs font-light uppercase tracking-widest opacity-60 mb-2">
                    Node Settlement Amount
                  </p>
                  <h3 className="md-typescale-display-small font-light tabular-nums">
                    {formatAmount(event.amount)}{" "}
                    <small className="text-sm opacity-40 uppercase ml-0.5">
                      UZS
                    </small>
                  </h3>
                  {amended && (
                    <div className="mt-4 pt-4 border-t border-white/10 flex items-center justify-between">
                      <span className="text-[10px] font-light opacity-60 uppercase">
                        Amended Baseline
                      </span>
                      <span className="text-xs font-light line-through opacity-40">
                        {formatAmount(event.original_amount!)}
                      </span>
                    </div>
                  )}
                </div>

                {error && (
                  <div className="p-4 rounded-xl bg-red-50 border border-red-100 flex items-center gap-3 text-red-600">
                    <AlertTriangle size={18} />
                    <p className="text-xs font-light">{error}</p>
                  </div>
                )}

                {state === "processing" ? (
                  <div className="flex flex-col items-center gap-4 py-8 text-center">
                    <Loader2
                      size={32}
                      className="animate-spin text-[var(--desk-accent)]"
                    />
                    <div>
                      <p className="md-typescale-body-medium font-light text-[var(--desk-text-primary)]">
                        {checkoutUrl
                          ? "Gateway Synchronizing..."
                          : "Initializing..."}
                      </p>
                      <p className="text-xs text-[var(--desk-text-tertiary)] mt-1">
                        Do not close this protocol
                      </p>
                    </div>
                    {checkoutUrl && (
                      <a
                        href={checkoutUrl}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="flex items-center gap-2 px-6 h-11 rounded-xl bg-[var(--desk-accent)] text-white font-light transition-all hover:scale-105"
                      >
                        Open Payment Portal <ExternalLink size={16} />
                      </a>
                    )}
                  </div>
                ) : (
                  <div className="space-y-3">
                    <button
                      onClick={handleCash}
                      className="flex items-center gap-4 w-full p-4 rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface)] hover:border-[var(--desk-border-strong)] hover:shadow-md transition-all group"
                    >
                      <div className="w-12 h-12 rounded-xl bg-[var(--desk-surface-subtle)] text-[var(--desk-text-tertiary)] flex items-center justify-center group-hover:bg-[var(--desk-accent-soft)] group-hover:text-[var(--desk-accent)] transition-colors">
                        <Banknote size={24} />
                      </div>
                      <div className="text-left">
                        <p className="md-typescale-title-small font-light text-[var(--desk-text-primary)]">
                          Physical Cash Settlement
                        </p>
                        <p className="md-typescale-body-small text-[var(--desk-text-tertiary)]">
                          Driver will verify at node location
                        </p>
                      </div>
                    </button>

                    {gateways.map((gw) => (
                      <button
                        key={gw}
                        onClick={() => handleCard(gw)}
                        className="flex items-center gap-4 w-full p-4 rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface)] hover:border-[var(--desk-border-strong)] hover:shadow-md transition-all group"
                      >
                        <div className="w-12 h-12 rounded-xl bg-[var(--desk-surface-subtle)] text-[var(--desk-text-tertiary)] flex items-center justify-center group-hover:bg-[var(--desk-accent-soft)] group-hover:text-[var(--desk-accent)] transition-colors">
                          <CreditCard size={24} />
                        </div>
                        <div className="text-left">
                          <p className="md-typescale-title-small font-light text-[var(--desk-text-primary)]">
                            Pay via {gw.replace(/_/g, " ")}
                          </p>
                          <p className="md-typescale-body-small text-[var(--desk-text-tertiary)]">
                            Secure encrypted network trade
                          </p>
                        </div>
                      </button>
                    ))}
                  </div>
                )}
              </div>
            </>
          )}
        </motion.div>
      </div>
    </AnimatePresence>
  );
}
