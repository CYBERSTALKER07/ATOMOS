"use client";

import { useState, useCallback } from "react";
import { CreditCard, Banknote, X, Loader2, ExternalLink } from "lucide-react";
import type { PaymentRequiredEvent } from "@pegasus/types";
import { useWsEvent, type WsMessage } from "../lib/ws";
import { apiFetch } from "../lib/auth";
import type { CardCheckoutResponse } from "../lib/types";

/* ── Types ── */

type PaymentEvent = PaymentRequiredEvent & {
  available_card_gateways?: string[];
};

type PaymentState = "idle" | "choosing" | "processing" | "success" | "error";

function formatAmount(amount: number): string {
  return amount.toLocaleString("en-US").replace(/,/g, " ") + "";
}

/* ── Component ── */

export default function PaymentModal() {
  const [event, setEvent] = useState<PaymentEvent | null>(null);
  const [state, setState] = useState<PaymentState>("idle");
  const [error, setError] = useState<string | null>(null);
  const [checkoutUrl, setCheckoutUrl] = useState<string | null>(null);

  // Listen for PAYMENT_REQUIRED WS events
  useWsEvent(
    "PAYMENT_REQUIRED",
    useCallback((msg: WsMessage) => {
      const evt: PaymentEvent = {
        order_id: msg.order_id as string,
        type: "PAYMENT_REQUIRED",
        invoice_id: (msg.invoice_id as string | null | undefined) ?? null,
        session_id: msg.session_id as string,
        amount: msg.amount as number,
        original_amount: msg.original_amount as number,
        payment_method: msg.payment_method as string,
        gateway: msg.gateway as PaymentRequiredEvent["gateway"],
        currency: msg.currency as string,
        available_card_gateways: msg.available_card_gateways as string[] | undefined,
        message: (msg.message as string | undefined) ?? "",
      };
      setEvent(evt);
      setState("choosing");
      setError(null);
      setCheckoutUrl(null);
    }, []),
  );

  // Listen for PAYMENT_SETTLED — auto-dismiss
  useWsEvent(
    "PAYMENT_SETTLED",
    useCallback((msg: WsMessage) => {
      if (event && msg.order_id === event.order_id) {
        setState("success");
        setTimeout(() => {
          setEvent(null);
          setState("idle");
        }, 2000);
      }
    }, [event]),
  );

  const dismiss = useCallback(() => {
    setEvent(null);
    setState("idle");
    setError(null);
    setCheckoutUrl(null);
  }, []);

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
        // Stay in processing until PAYMENT_SETTLED WS event arrives
      } catch (err) {
        setError(err instanceof Error ? err.message : "Card checkout failed");
        setState("choosing");
      }
    },
    [event],
  );

  // Don't render unless there's an active payment event
  if (!event || state === "idle") return null;

  const gateways = event.available_card_gateways ?? [];
  const amended = event.original_amount && event.original_amount !== event.amount;

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 z-50 flex items-center justify-center"
        style={{ background: "rgba(0,0,0,0.5)", backdropFilter: "blur(4px)" }}
      >
        {/* Modal */}
        <div
          className="relative w-full max-w-md rounded-2xl p-6"
          style={{
            background: "var(--background)",
            border: "1px solid var(--border)",
            boxShadow: "0 12px 32px rgba(0,0,0,0.14)",
          }}
        >
          {/* Close */}
          <button
            onClick={dismiss}
            className="absolute right-4 top-4 rounded-full p-1 transition-colors hover:bg-[var(--surface)]"
          >
            <X size={18} style={{ color: "var(--muted)" }} />
          </button>

          {/* Success State */}
          {state === "success" ? (
            <div className="flex flex-col items-center gap-4 py-8">
              <div
                className="flex h-16 w-16 items-center justify-center rounded-full"
                style={{ background: "var(--success)", color: "white" }}
              >
                <svg width={32} height={32} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round">
                  <polyline points="20 6 9 17 4 12" />
                </svg>
              </div>
              <p className="md-typescale-title-medium" style={{ color: "var(--foreground)" }}>
                Payment confirmed
              </p>
            </div>
          ) : (
            <>
              {/* Header */}
              <div className="mb-6">
                <p className="md-typescale-title-large" style={{ color: "var(--foreground)" }}>
                  Payment Required
                </p>
                <p className="md-typescale-body-small mt-1" style={{ color: "var(--muted)" }}>
                  Order {event.order_id.slice(0, 8)} · Delivery complete
                </p>
              </div>

              {/* Amount */}
              <div className="mb-6 rounded-xl p-4" style={{ background: "var(--surface)" }}>
                <p className="md-typescale-label-small uppercase tracking-widest" style={{ color: "var(--muted)" }}>
                  Amount Due
                </p>
                <p className="md-typescale-headline-small tabular-nums mt-1" style={{ color: "var(--foreground)" }}>
                  {formatAmount(event.amount)}
                </p>
                {amended && (
                  <p className="md-typescale-body-small mt-1" style={{ color: "var(--warning)" }}>
                    Amended from {formatAmount(event.original_amount!)}
                  </p>
                )}
              </div>

              {/* Error */}
              {error && (
                <div className="mb-4 rounded-xl p-3" style={{ background: "var(--danger)", color: "white" }}>
                  <p className="md-typescale-body-small">{error}</p>
                </div>
              )}

              {/* Processing */}
              {state === "processing" ? (
                <div className="flex flex-col items-center gap-3 py-4">
                  <Loader2 size={24} className="animate-spin" style={{ color: "var(--muted)" }} />
                  <p className="md-typescale-body-medium" style={{ color: "var(--muted)" }}>
                    {checkoutUrl ? "Waiting for payment confirmation..." : "Processing..."}
                  </p>
                  {checkoutUrl && (
                    <a
                      href={checkoutUrl}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex items-center gap-1 md-typescale-label-large"
                      style={{ color: "var(--accent)" }}
                    >
                      Open payment page <ExternalLink size={14} />
                    </a>
                  )}
                </div>
              ) : (
                /* Payment Options */
                <div className="space-y-3">
                  {/* Cash */}
                  <button
                    onClick={handleCash}
                    className="flex w-full items-center gap-3 rounded-xl p-4 text-left transition-colors hover:bg-[var(--surface)]"
                    style={{ border: "1px solid var(--border)" }}
                  >
                    <div
                      className="flex h-10 w-10 items-center justify-center rounded-xl"
                      style={{ background: "var(--surface)" }}
                    >
                      <Banknote size={20} style={{ color: "var(--foreground)" }} />
                    </div>
                    <div>
                      <p className="md-typescale-title-small" style={{ color: "var(--foreground)" }}>Pay Cash</p>
                      <p className="md-typescale-body-small" style={{ color: "var(--muted)" }}>
                        Driver will collect at your location
                      </p>
                    </div>
                  </button>

                  {/* Card Gateways */}
                  {gateways.map((gw) => (
                    <button
                      key={gw}
                      onClick={() => handleCard(gw)}
                      className="flex w-full items-center gap-3 rounded-xl p-4 text-left transition-colors hover:bg-[var(--surface)]"
                      style={{ border: "1px solid var(--border)" }}
                    >
                      <div
                        className="flex h-10 w-10 items-center justify-center rounded-xl"
                        style={{ background: "var(--surface)" }}
                      >
                        <CreditCard size={20} style={{ color: "var(--foreground)" }} />
                      </div>
                      <div>
                        <p className="md-typescale-title-small" style={{ color: "var(--foreground)" }}>
                          Pay via {gw.replace(/_/g, " ")}
                        </p>
                        <p className="md-typescale-body-small" style={{ color: "var(--muted)" }}>
                          Secure card payment
                        </p>
                      </div>
                    </button>
                  ))}
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </>
  );
}
