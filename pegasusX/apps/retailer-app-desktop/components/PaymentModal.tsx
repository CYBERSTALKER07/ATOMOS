"use client";

import { useState, useCallback, useEffect } from "react";
import { usePathname, useRouter } from "next/navigation";
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
import type { PaymentRequired } from "@pegasusx/types";
import { useWsEvent, useWebSocket, type WsMessage } from "../lib/ws";
import { apiFetch } from "../lib/auth";
import type { CardCheckoutResponse, PendingPaymentSession, PendingPaymentsResponse } from "../lib/types";

/* ── Types ── */

type PaymentEvent = Omit<PaymentRequired, "available_card_gateways" | "amount"> & {
  available_card_gateways?: string[];
  amount: number;
  original_amount?: number;
  type: string;
  session_id?: string;
  invoice_id?: string | null;
  message?: string;
  gateway?: string;
};

type PaymentState =
  | "idle"
  | "choosing"
  | "confirm-cash"
  | "processing"
  | "cash-pending"
  | "success"
  | "error";

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
    gateway: gateway,
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
    gateway: ((msg.gateway as string) || "GLOBAL_PAY"),
    currency: (msg.currency as string) || "UZS",
    available_card_gateways: gateways,
    message: (msg.message as string | undefined) ?? "",
  };
}

/* ── Component ── */

export default function PaymentModal() {
  const router = useRouter();
  const pathname = usePathname();
  const { reconnectEpoch } = useWebSocket();
  const [event, setEvent] = useState<PaymentEvent | null>(null);
  const [state, setState] = useState<PaymentState>("idle");
  const [error, setError] = useState<string | null>(null);
  const [checkoutUrl, setCheckoutUrl] = useState<string | null>(null);
  const hideForAddCard = pathname?.startsWith("/settings/cards") ?? false;

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

  useWsEvent(
    "ORDER_COMPLETED",
    useCallback(
      (msg: WsMessage) => {
        if (event && msg.order_id === event.order_id) {
          setState("success");
          setTimeout(dismiss, 2000);
        }
      },
      [event, dismiss],
    ),
  );

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

  useEffect(() => {
    if (typeof window === "undefined") return;
    const params = new URLSearchParams(window.location.search);
    if (params.get("resume_delivery_payment") !== "1") return;

    let cancelled = false;

    async function resumeDeliveryPayment() {
      try {
        const res = await apiFetch("/v1/retailer/pending-payments");
        if (!res.ok) return;
        const data: PendingPaymentsResponse = await res.json();
        const orderId = params.get("order_id") ?? "";
        const pending =
          data.pending_payments?.find((session) => session.order_id === orderId) ??
          data.pending_payments?.[0];
        if (!cancelled && pending) {
          setEvent(sessionToPaymentEvent(pending));
          setState("choosing");
          setError(null);
          setCheckoutUrl(null);
        }
      } catch {
        // WebSocket delivery remains the primary realtime path.
      } finally {
        if (!cancelled && pathname) {
          router.replace(pathname);
        }
      }
    }

    void resumeDeliveryPayment();
    return () => {
      cancelled = true;
    };
  }, [pathname, router]);

  useEffect(() => {
    if (reconnectEpoch === 0) return;
    let cancelled = false;

    async function reconcilePendingPayment() {
      try {
        const res = await apiFetch("/v1/retailer/pending-payments");
        if (!res.ok) return;
        const data: PendingPaymentsResponse = await res.json();
        const pending = data.pending_payments?.[0];
        if (cancelled) return;

        if (pending) {
          const refreshed = sessionToPaymentEvent(pending);
          setEvent((prev) => {
            if (!prev || prev.order_id === refreshed.order_id) {
              return refreshed;
            }
            return prev;
          });
          setState((prev) => (prev === "processing" ? "choosing" : prev));
          setError(null);
          return;
        }

        setEvent(null);
        setState("idle");
        setError(null);
        setCheckoutUrl(null);
      } catch {
        // WebSocket delivery remains the primary realtime path.
      }
    }

    void reconcilePendingPayment();
    return () => {
      cancelled = true;
    };
  }, [reconnectEpoch]);

  const handleCash = useCallback(async () => {
    if (!event) return;
    setState("processing");
    setError(null);
    try {
      const res = await apiFetch("/v1/delivery/confirm-cash", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": `retailer-confirm-cash:${event.order_id}`,
        },
        body: JSON.stringify({ order_id: event.order_id }),
      });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || "Cash confirmation failed");
      }
      setState("cash-pending");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Cash confirmation failed");
      setState("choosing");
    }
  }, [event, dismiss]);

  const reviewCash = useCallback(() => {
    setError(null);
    setState("confirm-cash");
  }, []);

  const handleAddCard = useCallback(() => {
    if (!event) return;
    const params = new URLSearchParams({
      return_to: "delivery_payment",
      order_id: event.order_id,
    });
    if (event.session_id) {
      params.set("session_id", event.session_id);
    }
    router.push(`/settings/cards?${params.toString()}`);
  }, [event, router]);

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

  if (!event || state === "idle" || hideForAddCard) return null;

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
                <h2 className="md-typescale-title-large font-bold text-[var(--desk-text-primary)]">
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
                    <h2 className="md-typescale-title-large font-bold text-[var(--desk-text-primary)]">
                      Payment Required
                    </h2>
                    <p className="text-[10px] font-bold text-[var(--desk-text-tertiary)] uppercase tracking-widest">
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
                  <p className="text-xs font-bold uppercase tracking-widest opacity-60 mb-2">
                    Node Settlement Amount
                  </p>
                  <h3 className="md-typescale-display-small font-bold tabular-nums">
                    {formatAmount(event.amount)}{" "}
                    <small className="text-sm opacity-40 uppercase ml-0.5">
                      UZS
                    </small>
                  </h3>
                  {amended && (
                    <div className="mt-4 pt-4 border-t border-white/10 flex items-center justify-between">
                      <span className="text-[10px] font-bold opacity-60 uppercase">
                        Amended Baseline
                      </span>
                      <span className="text-xs font-bold line-through opacity-40">
                        {formatAmount(event.original_amount!)}
                      </span>
                    </div>
                  )}
                </div>

                {error && (
                  <div className="p-4 rounded-xl bg-red-50 border border-red-100 flex items-center gap-3 text-red-600">
                    <AlertTriangle size={18} />
                    <p className="text-xs font-bold">{error}</p>
                  </div>
                )}

                {state === "processing" ? (
                  <div className="flex flex-col items-center gap-4 py-8 text-center">
                    <Loader2
                      size={32}
                      className="animate-spin text-[var(--desk-accent)]"
                    />
                    <div>
                      <p className="md-typescale-body-medium font-bold text-[var(--desk-text-primary)]">
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
                        className="flex items-center gap-2 px-6 h-11 rounded-xl bg-[var(--desk-accent)] text-white font-bold transition-all hover:scale-105"
                      >
                        Open Payment Portal <ExternalLink size={16} />
                      </a>
                    )}
                  </div>
                ) : state === "cash-pending" ? (
                  <div className="flex flex-col items-center gap-4 py-8 text-center">
                    <Banknote size={40} className="text-[var(--desk-warning)]" />
                    <div>
                      <p className="md-typescale-body-medium font-bold text-[var(--desk-text-primary)]">
                        Awaiting driver cash collection
                      </p>
                      <p className="text-xs text-[var(--desk-text-tertiary)] mt-1">
                        The driver will collect {formatAmount(event.amount)} UZS and complete the delivery.
                      </p>
                    </div>
                  </div>
                ) : state === "confirm-cash" ? (
                  <div className="space-y-4">
                    <div className="rounded-2xl border border-[var(--desk-warning)]/30 bg-[var(--desk-warning)]/10 p-4 text-left">
                      <p className="text-[10px] font-bold uppercase tracking-widest text-[var(--desk-warning)]">
                        Cash Payment
                      </p>
                      <p className="mt-2 text-sm font-bold text-[var(--desk-text-primary)]">
                        Confirm you will pay {formatAmount(event.amount)} UZS in cash. The driver will collect and complete the delivery.
                      </p>
                    </div>

                    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                      <button
                        onClick={() => setState("choosing")}
                        className="flex h-12 items-center justify-center rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface)] font-bold text-[var(--desk-text-primary)] transition-all hover:border-[var(--desk-border-strong)]"
                      >
                        Back
                      </button>
                      <button
                        onClick={handleCash}
                        className="flex h-12 items-center justify-center rounded-2xl bg-[var(--desk-warning)] px-4 font-bold text-white transition-all hover:scale-[1.01]"
                      >
                        Pay with Cash
                      </button>
                    </div>
                  </div>
                ) : (
                  <div className="space-y-3">
                    <button
                      onClick={reviewCash}
                      className="flex items-center gap-4 w-full p-4 rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface)] hover:border-[var(--desk-border-strong)] hover:shadow-md transition-all group"
                    >
                      <div className="w-12 h-12 rounded-xl bg-[var(--desk-surface-subtle)] text-[var(--desk-text-tertiary)] flex items-center justify-center group-hover:bg-[var(--desk-accent-soft)] group-hover:text-[var(--desk-accent)] transition-colors">
                        <Banknote size={24} />
                      </div>
                      <div className="text-left">
                        <p className="md-typescale-title-small font-bold text-[var(--desk-text-primary)]">
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
                          <p className="md-typescale-title-small font-bold text-[var(--desk-text-primary)]">
                            Pay via {gw.replace(/_/g, " ")}
                          </p>
                          <p className="md-typescale-body-small text-[var(--desk-text-tertiary)]">
                            Secure encrypted network trade
                          </p>
                        </div>
                      </button>
                    ))}

                    <button
                      type="button"
                      onClick={handleAddCard}
                      className="w-full py-3 text-sm font-bold text-[var(--desk-accent)] underline underline-offset-4"
                    >
                      Add payment method
                    </button>
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
