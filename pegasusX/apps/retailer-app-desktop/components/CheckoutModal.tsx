"use client";

import { useState, useEffect } from "react";
import {
  X,
  Ticket,
  CreditCard,
  Loader2,
  ShieldCheck,
  Truck,
  ChevronRight,
  AlertTriangle,
} from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { Button } from "@heroui/react";
import { useCart } from "../lib/cart";
import { apiFetch } from "../lib/auth";
import { useWebSocket } from "../lib/ws";
import { useRouter } from "next/navigation";
import type {
  ActiveFulfillmentsResponse,
  CheckoutPreviewResponse,
  PendingPaymentsResponse,
  UnifiedCheckoutResponse,
  RetailerProfile,
  StockWarning,
} from "../lib/types";
import type { PaymentGatewayDegradedPayload } from "@pegasusx/types";
import { retailerUnifiedCheckoutKey } from "@pegasusx/api-client";
import {
  enqueuePendingCheckout,
  pendingCheckoutQueuedMessage,
} from "../lib/pending-checkout";

function getProfile(): RetailerProfile | null {
  if (typeof localStorage === "undefined") return null;
  try {
    const raw = localStorage.getItem("retailer_profile");
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

interface CheckoutModalProps {
  isOpen: boolean;
  onClose: () => void;
  total: number;
}

export default function CheckoutModal({
  isOpen,
  onClose,
  total,
}: CheckoutModalProps) {
  const { items, clearCart } = useCart();
  const [method, setMethod] = useState<"global_pay" | "cash">("cash");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [degradedBanner, setDegradedBanner] = useState<{ gateway: string; reason: string } | null>(null);
  const [oosItems, setOosItems] = useState<string[]>([]);
  const [stockWarnings, setStockWarnings] = useState<StockWarning[]>([]);
  const [deliveryFeeMinor, setDeliveryFeeMinor] = useState(0);
  const [deliveryMode, setDeliveryMode] = useState<"STANDARD" | "SCHEDULED">("STANDARD");
  const [deliveryDate, setDeliveryDate] = useState("");
  const [expressPriority, setExpressPriority] = useState(false);
  const [hasCardConfigured, setHasCardConfigured] = useState(false);
  const [addingCard, setAddingCard] = useState(false);
  const [pendingCardToken, setPendingCardToken] = useState<string | null>(null);
  const [cardOtpCode, setCardOtpCode] = useState("");
  const [cardSetupError, setCardSetupError] = useState("");
  const router = useRouter();
  const { subscribe, reconnectEpoch } = useWebSocket();

  useEffect(() => {
    if (isOpen) {
      apiFetch("/v1/retailer/cards")
        .then((res) => res.json())
        .then((data) => {
          if (data.cards && data.cards.length > 0) {
            setHasCardConfigured(true);
          } else {
            setHasCardConfigured(false);
          }
        })
        .catch(() => setHasCardConfigured(false));
    }
  }, [isOpen]);

  useEffect(() => {
    if (!isOpen || reconnectEpoch === 0) return;
    let cancelled = false;

    async function reconcileCheckout() {
      try {
        const [fulfillmentRes, pendingRes] = await Promise.all([
          apiFetch("/v1/retailer/active-fulfillment"),
          apiFetch("/v1/retailer/pending-payments"),
        ]);
        if (cancelled) return;

        const fulfillments = fulfillmentRes.ok
          ? ((await fulfillmentRes.json()) as ActiveFulfillmentsResponse).fulfillments ?? []
          : [];
        const pending = pendingRes.ok
          ? ((await pendingRes.json()) as PendingPaymentsResponse).pending_payments?.[0]
          : undefined;

        if (pending) {
          setLoading(false);
          setError("");
          onClose();
          return;
        }

        if (fulfillments.length > 0) {
          setLoading(false);
          setError("");
          clearCart();
          onClose();
          router.push("/orders");
          return;
        }

        if (loading) {
          setLoading(false);
          setError("Connection restored. Confirm checkout status before retrying.");
        }
      } catch {
        if (!cancelled && loading) {
          setLoading(false);
          setError("Connection restored. Confirm checkout status before retrying.");
        }
      }
    }

    void reconcileCheckout();
    return () => {
      cancelled = true;
    };
  }, [isOpen, loading, reconnectEpoch, onClose, clearCart, router]);

  const handleInitiateCard = async () => {
    setAddingCard(true);
    setCardSetupError("");
    try {
      const res = await apiFetch("/v1/retailer/card/initiate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
      });
      if (!res.ok) {
        throw new Error("Could not start card tokenization");
      }
      const data = (await res.json()) as { card_token?: string };
      if (!data.card_token) {
        throw new Error("Card tokenization session missing");
      }
      setPendingCardToken(data.card_token);
    } catch (err) {
      setCardSetupError(err instanceof Error ? err.message : "Could not start card tokenization");
    } finally {
      setAddingCard(false);
    }
  };

  const handleConfirmCard = async () => {
    if (!pendingCardToken || !cardOtpCode.trim()) return;
    setAddingCard(true);
    setCardSetupError("");
    try {
      const res = await apiFetch("/v1/retailer/card/confirm", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          card_token: pendingCardToken,
          otp_code: cardOtpCode.trim(),
        }),
      });
      if (!res.ok) {
        throw new Error("Could not confirm card");
      }
      setPendingCardToken(null);
      setCardOtpCode("");
      setHasCardConfigured(true);
    } catch (err) {
      setCardSetupError(err instanceof Error ? err.message : "Could not confirm card");
    } finally {
      setAddingCard(false);
    }
  };

  useEffect(() => {
    const unsub = subscribe("PAYMENT_GATEWAY_DEGRADED", (msg) => {
      const payload = msg as unknown as PaymentGatewayDegradedPayload;
      if (payload.gateway) {
        setDegradedBanner({ gateway: payload.gateway, reason: payload.reason || "Outage detected" });
        if (method !== "cash") {
          setMethod("cash");
        }
      }
    });
    return () => unsub();
  }, [subscribe, method]);

  const handleCheckout = async () => {
    if (items.length === 0) return;
    setLoading(true);
    setError("");
    setOosItems([]);
    setStockWarnings([]);

    try {
      const profile = getProfile();
      if (!profile) {
        throw new Error("Authentication required. Please log in again.");
      }

      const gatewayMap: Record<string, string> = {
        global_pay: "GLOBAL_PAY",
        adyen: "ADYEN",
        airwallex: "AIRWALLEX",
        cash: "CASH",
      };

      const lineItems = items.map((item) => ({
        sku_id: item.product_id,
        quantity: item.quantity,
        unit_price: item.price,
      }));

      const previewRes = await apiFetch("/v1/checkout/preview", {
        method: "POST",
        body: JSON.stringify({
          retailer_id: profile.id,
          latitude: 0,
          longitude: 0,
          items: lineItems,
        }),
      });
      if (!previewRes.ok) {
        const previewErr = await previewRes.json().catch(() => null);
        throw new Error(previewErr?.error || "Checkout preview failed");
      }
      const preview = (await previewRes.json()) as CheckoutPreviewResponse;
      setDeliveryFeeMinor(preview.delivery_fee_minor ?? 0);
      if (preview.blocked) {
        if (preview.line_errors && Object.keys(preview.line_errors).length > 0) {
          throw new Error(Object.values(preview.line_errors).join("; "));
        }
        setOosItems(preview.rejected_skus || preview.oos_items || []);
        throw new Error(
          preview.message || "Some items cannot be ordered with current stock policy.",
        );
      }
      if (
        preview.stock_warnings &&
        preview.stock_warnings.length > 0 &&
        !window.confirm(
          `Some items will be backordered (${preview.backordered_item_count ?? preview.stock_warnings.length} line(s)). Delivery may be delayed. Proceed?`,
        )
      ) {
        setLoading(false);
        return;
      }
      if (preview.stock_warnings?.length) {
        setStockWarnings(preview.stock_warnings);
      }

      const cartKey = lineItems
        .map((item) => `${item.sku_id}:${item.quantity}:${item.unit_price}`)
        .sort()
        .join("|");
      const idempotencyKey = retailerUnifiedCheckoutKey(method, cartKey);

      const checkoutPayload: Record<string, unknown> = {
        retailer_id: profile.id,
        payment_gateway: gatewayMap[method] || "GLOBAL_PAY",
        latitude: 0,
        longitude: 0,
        items: lineItems,
        delivery_mode: deliveryMode,
        delivery_priority: expressPriority ? "EXPRESS" : "STANDARD",
      };
      if (deliveryDate) {
        const iso = new Date(`${deliveryDate}T12:00:00+05:00`).toISOString();
        if (deliveryMode === "SCHEDULED") {
          checkoutPayload.requested_delivery_date = iso;
        } else {
          checkoutPayload.deliver_before = iso;
        }
      }

      const cartRes = await apiFetch("/v1/checkout/unified", {
        method: "POST",
        headers: {
          "Idempotency-Key": idempotencyKey,
        },
        body: JSON.stringify(checkoutPayload),
      });

      if (!cartRes.ok) {
        const errBody = await cartRes.json().catch(() => null);
        if (
          cartRes.status === 409 &&
          errBody?.code === "ALL_ITEMS_OUT_OF_STOCK"
        ) {
          setOosItems(errBody.oos_items || []);
          throw new Error(
            "All items are out of stock. Please update your cart.",
          );
        }
        if (
          cartRes.status === 409 &&
          errBody?.code === "PARTIAL_OUT_OF_STOCK_REJECTED"
        ) {
          setOosItems(errBody.oos_items || []);
          throw new Error(
            "Some items are out of stock and cannot be backordered. Please update your cart.",
          );
        }
        if (cartRes.status === 422 && errBody?.error === "payment_gateway_policy_violation") {
          // This happens when unified checkout policy triggers 3C Fallback.
          // We can read it here, although WS will arrive shortly.
          setDegradedBanner({ gateway: gatewayMap[method], reason: errBody.message || "Gateway temporarily blocked." });
          if (method !== "cash") {
             setMethod("cash");
          }
          throw new Error(errBody.message || "Payment gateway policy violation. Switched to cash fallback.");
        }
        throw new Error(errBody?.error || "Failed to create orders from cart");
      }

      const resData = (await cartRes.json()) as UnifiedCheckoutResponse;
      if (resData.stock_warnings && resData.stock_warnings.length > 0) {
        setStockWarnings(resData.stock_warnings);
      } else {
        setStockWarnings([]);
      }

      clearCart();
      onClose();
      router.push("/orders");
    } catch (err: unknown) {
      const retryable =
        err instanceof TypeError ||
        (err instanceof Error &&
          /failed to fetch|network|load failed|offline/i.test(err.message));
      if (retryable) {
        try {
          const profile = getProfile();
          if (profile) {
            const gatewayMap: Record<string, string> = {
              global_pay: "GLOBAL_PAY",
              adyen: "ADYEN",
              airwallex: "AIRWALLEX",
              cash: "CASH",
            };
            const lineItems = items.map((item) => ({
              sku_id: item.product_id,
              quantity: item.quantity,
              unit_price: item.price,
            }));
            const cartKey = lineItems
              .map((item) => `${item.sku_id}:${item.quantity}:${item.unit_price}`)
              .sort()
              .join("|");
            enqueuePendingCheckout(
              {
                retailer_id: profile.id,
                payment_gateway: gatewayMap[method] || "GLOBAL_PAY",
                latitude: 0,
                longitude: 0,
                items: lineItems,
              },
              retailerUnifiedCheckoutKey(method, cartKey),
            );
          }
        } catch {
          // Best-effort queue.
        }
        setError(pendingCheckoutQueuedMessage(err));
      } else {
        setError(err instanceof Error ? err.message : "Checkout failed");
      }
    } finally {
      setLoading(false);
    }
  };

  const isCardDisabled = degradedBanner != null;

  return (
    <AnimatePresence>
      {isOpen && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          className="fixed inset-0 z-50 bg-black/40 backdrop-blur-sm flex items-center justify-center p-4"
        >
          <motion.div
            initial={{ scale: 0.95, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            exit={{ scale: 0.95, opacity: 0 }}
            className="w-full max-w-lg bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-[32px] shadow-2xl overflow-hidden flex flex-col"
          >
            <div className="flex items-center justify-between p-6 border-b border-[var(--desk-border)]">
              <h2 className="md-typescale-title-large font-bold text-[var(--desk-text-primary)]">
                Secure Checkout
              </h2>
              <button
                onClick={onClose}
                className="w-10 h-10 rounded-full flex items-center justify-center text-[var(--desk-text-tertiary)] hover:bg-[var(--desk-surface-subtle)] hover:text-[var(--desk-text-primary)] transition-colors"
              >
                <X size={20} />
              </button>
            </div>

            <div className="p-6 flex flex-col gap-6 max-h-[70vh] overflow-y-auto">
              <div className="flex items-center justify-between p-6 bg-[var(--desk-accent)]/5 border border-[var(--desk-accent)]/10 rounded-3xl">
                <div>
                  <span className="text-[10px] font-black uppercase tracking-[0.2em] text-[var(--desk-accent)] mb-1 block">
                    Total Authorization
                  </span>
                  <div className="md-typescale-display-small font-bold text-[var(--desk-text-primary)]">
                    UZS {(total + deliveryFeeMinor).toLocaleString()}
                  </div>
                  {deliveryFeeMinor > 0 && (
                    <p className="text-xs text-[var(--desk-text-tertiary)] mt-1">
                      Includes {deliveryFeeMinor.toLocaleString()} UZS delivery fee
                    </p>
                  )}
                </div>
                <Ticket
                  size={32}
                  className="text-[var(--desk-accent)] opacity-20"
                />
              </div>

              {degradedBanner && (
                <div className="p-4 bg-orange-50 border border-orange-200 rounded-2xl flex gap-3 text-orange-800">
                  <AlertTriangle size={20} className="shrink-0 mt-0.5" />
                  <div>
                     <h3 className="font-bold text-sm uppercase tracking-wide">Card Payments Temporarily Unavailable</h3>
                     <p className="text-xs mt-1 font-medium opacity-90">
                       {degradedBanner.gateway} is currently experiencing issues ({degradedBanner.reason}). We have automatically switched your payment method to cash.
                     </p>
                  </div>
                </div>
              )}

              <div className="space-y-3">
                <span className="text-[10px] font-black uppercase tracking-[0.2em] text-[var(--desk-text-tertiary)] pl-2">
                  Delivery Intent
                </span>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  <button
                    type="button"
                    onClick={() => setDeliveryMode("STANDARD")}
                    className={`p-4 rounded-2xl border text-left ${deliveryMode === "STANDARD" ? "border-[var(--desk-accent)] bg-[var(--desk-accent)]/5" : "border-[var(--desk-border)]"}`}
                  >
                    <span className="font-bold text-sm">Standard (ASAP)</span>
                    <span className="block text-xs text-[var(--desk-text-tertiary)] mt-1">Earliest T+1 business day</span>
                  </button>
                  <button
                    type="button"
                    onClick={() => setDeliveryMode("SCHEDULED")}
                    className={`p-4 rounded-2xl border text-left ${deliveryMode === "SCHEDULED" ? "border-[var(--desk-accent)] bg-[var(--desk-accent)]/5" : "border-[var(--desk-border)]"}`}
                  >
                    <span className="font-bold text-sm">Scheduled pre-order</span>
                    <span className="block text-xs text-[var(--desk-text-tertiary)] mt-1">Delivery day T+3 or later</span>
                  </button>
                </div>
                <input
                  type="date"
                  value={deliveryDate}
                  onChange={(e) => setDeliveryDate(e.target.value)}
                  className="w-full rounded-xl border border-[var(--desk-border)] px-4 py-3 bg-[var(--desk-canvas)]"
                />
                <label className="flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={expressPriority}
                    onChange={(e) => setExpressPriority(e.target.checked)}
                  />
                  Express priority (+fee)
                </label>
              </div>

              <div className="space-y-3">
                <span className="text-[10px] font-black uppercase tracking-[0.2em] text-[var(--desk-text-tertiary)] pl-2">
                  Payment Protocol
                </span>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  <button
                    onClick={() => setMethod("cash")}
                    className={`relative p-5 rounded-2xl border text-left transition-all ${
                      method === "cash"
                        ? "border-[var(--desk-accent)] bg-[var(--desk-accent)]/5 ring-1 ring-[var(--desk-accent)]"
                        : "border-[var(--desk-border)] bg-[var(--desk-canvas)] hover:border-[var(--desk-text-tertiary)]"
                    }`}
                  >
                    <div className="flex items-center justify-between mb-3">
                      <Truck
                        size={20}
                        className={
                          method === "cash"
                            ? "text-[var(--desk-accent)]"
                            : "text-[var(--desk-text-tertiary)]"
                        }
                      />
                      {method === "cash" && (
                        <div className="w-2 h-2 rounded-full bg-[var(--desk-accent)] shadow-[0_0_8px_var(--desk-accent)]" />
                      )}
                    </div>
                    <span className="block md-typescale-body-large font-bold text-[var(--desk-text-primary)]">
                      Cash on Delivery
                    </span>
                    <span className="text-[10px] font-bold text-[var(--desk-text-tertiary)] uppercase tracking-widest mt-1 block">
                      Physical tender
                    </span>
                  </button>

                  <button
                    disabled={isCardDisabled}
                    onClick={() => setMethod("global_pay")}
                    className={`relative p-5 rounded-2xl border text-left transition-all ${isCardDisabled ? "opacity-40 cursor-not-allowed grayscale" : ""} ${
                      method === "global_pay"
                        ? "border-[var(--desk-accent)] bg-[var(--desk-accent)]/5 ring-1 ring-[var(--desk-accent)]"
                        : "border-[var(--desk-border)] bg-[var(--desk-canvas)] hover:border-[var(--desk-text-tertiary)]"
                    }`}
                  >
                    <div className="flex items-center justify-between mb-3">
                      <CreditCard
                        size={20}
                        className={
                          method === "global_pay"
                            ? "text-[var(--desk-accent)]"
                            : "text-[var(--desk-text-tertiary)]"
                        }
                      />
                      {method === "global_pay" && (
                        <div className="w-2 h-2 rounded-full bg-[var(--desk-accent)] shadow-[0_0_8px_var(--desk-accent)]" />
                      )}
                    </div>
                    <span className="block md-typescale-body-large font-bold text-[var(--desk-text-primary)]">
                      Card (Global Pay)
                    </span>
                    <span className="text-[10px] font-bold text-[var(--desk-text-tertiary)] uppercase tracking-widest mt-1 block">
                      Secure digital payment
                    </span>
                  </button>
                </div>
              </div>

              {method === "global_pay" && !hasCardConfigured && (
                <div className="p-5 border border-[var(--desk-border)] rounded-2xl bg-[var(--desk-surface-subtle)] space-y-4">
                  <div>
                    <h3 className="md-typescale-body-large font-bold text-[var(--desk-text-primary)]">Setup Payment Card</h3>
                    <p className="md-typescale-body-small text-[var(--desk-text-secondary)] mt-1">
                      Tokenize a card via OTP confirmation (same flow as Settings → Saved Cards).
                    </p>
                  </div>
                  {cardSetupError && (
                    <p className="text-xs font-semibold text-red-600">{cardSetupError}</p>
                  )}
                  {!pendingCardToken ? (
                    <Button
                      onPress={() => void handleInitiateCard()}
                      isDisabled={addingCard}
                      className="w-full h-11 bg-[var(--desk-accent)] text-white font-bold rounded-xl shadow-md flex items-center justify-center gap-2"
                    >
                      {addingCard ? <Loader2 size={18} className="animate-spin" /> : "Start card setup"}
                    </Button>
                  ) : (
                    <div className="space-y-3">
                      <input
                        type="text"
                        inputMode="numeric"
                        placeholder="OTP code"
                        value={cardOtpCode}
                        onChange={(e) => setCardOtpCode(e.target.value)}
                        className="w-full px-4 py-3 rounded-xl border border-[var(--desk-border)] bg-[var(--desk-surface)] text-sm"
                      />
                      <div className="flex gap-3">
                        <Button
                          onPress={() => {
                            setPendingCardToken(null);
                            setCardOtpCode("");
                          }}
                          className="flex-1 h-11 rounded-xl border border-[var(--desk-border)] font-bold"
                        >
                          Cancel
                        </Button>
                        <Button
                          onPress={() => void handleConfirmCard()}
                          isDisabled={addingCard || !cardOtpCode.trim()}
                          className="flex-1 h-11 bg-[var(--desk-accent)] text-white font-bold rounded-xl"
                        >
                          {addingCard ? <Loader2 size={18} className="animate-spin" /> : "Confirm card"}
                        </Button>
                      </div>
                    </div>
                  )}
                </div>
              )}

              {stockWarnings.length > 0 && (
                <div className="p-4 bg-amber-50 border border-amber-200 rounded-2xl flex gap-3 text-amber-900">
                  <AlertTriangle size={20} className="shrink-0 mt-0.5" />
                  <div className="space-y-2">
                    <h3 className="font-bold text-sm uppercase tracking-wide">
                      Partial backorder
                    </h3>
                    <p className="text-xs font-medium opacity-90">
                      Some items are out of stock but your warehouse accepts backorders. Fulfillment will proceed when stock arrives.
                    </p>
                    <ul className="text-xs space-y-1 font-mono">
                      {stockWarnings.map((warning) => (
                        <li key={warning.sku}>
                          {warning.sku}: {warning.backorder_qty} of {warning.requested} units backordered
                        </li>
                      ))}
                    </ul>
                  </div>
                </div>
              )}

              {error && (
                <div className="p-4 bg-red-50 border border-red-100 text-red-600 text-xs font-bold rounded-xl flex items-center gap-3">
                  <ShieldCheck size={16} className="shrink-0" />
                  {error}
                </div>
              )}

              {oosItems.length > 0 && (
                <div className="p-4 bg-orange-50 border border-orange-100 text-orange-700 text-xs font-bold rounded-xl flex flex-col gap-2">
                  <div className="flex items-center gap-2">
                    <ShieldCheck size={16} />
                    <span>The following items are out of stock:</span>
                  </div>
                  <ul className="list-disc list-inside pl-6 space-y-1">
                    {oosItems.map((id) => (
                      <li key={id}>{id}</li>
                    ))}
                  </ul>
                </div>
              )}
            </div>

            <div className="p-6 bg-[var(--desk-surface-subtle)] border-t border-[var(--desk-border)] flex items-center justify-between gap-4">
              <div className="flex items-center gap-2 text-[var(--desk-text-tertiary)]">
                <ShieldCheck size={14} />
                <span className="text-[10px] font-bold uppercase tracking-widest">
                  End-to-end encrypted
                </span>
              </div>
              <Button
                onPress={handleCheckout}
                isDisabled={loading || items.length === 0 || (method === "global_pay" && !hasCardConfigured)}
                className="h-12 px-8 bg-[var(--desk-text-primary)] text-[var(--desk-surface)] font-bold rounded-xl shadow-lg flex items-center gap-2 transition-all hover:scale-105 active:scale-95 disabled:opacity-50 disabled:cursor-not-allowed"
              >

                {loading ? (
                  <Loader2 size={18} className="animate-spin" />
                ) : (
                  <>
                    Confirm Execution <ChevronRight size={18} />
                  </>
                )}
              </Button>
            </div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}
