"use client";

import { useState } from "react";
import {
  X,
  Ticket,
  CreditCard,
  Loader2,
  ShieldCheck,
  Truck,
  ChevronRight,
} from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { Button } from "@heroui/react";
import { useCart } from "../lib/cart";
import { apiFetch } from "../lib/auth";
import { useRouter } from "next/navigation";
import type {
  CardCheckoutResponse,
  CashCheckoutResponse,
  UnifiedCheckoutResponse,
  RetailerProfile,
} from "../lib/types";

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
  const [method, setMethod] = useState<"global_pay" | "adyen" | "cash">("global_pay");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [oosItems, setOosItems] = useState<string[]>([]);
  const router = useRouter();

  const handleCheckout = async () => {
    if (items.length === 0) return;
    setLoading(true);
    setError("");
    setOosItems([]);

    try {
      const profile = getProfile();
      if (!profile?.id)
        throw new Error("Retailer profile not found. Please log in again.");

      const gatewayMap: Record<string, string> = {
        cash: "CASH",
        global_pay: "GLOBAL_PAY",
        adyen: "ADYEN",
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

      const cartRes = await apiFetch("/v1/checkout/unified", {
        method: "POST",
        headers: {
          "Idempotency-Key": `retailer-checkout:${method}:${cartKey}`,
        },
        body: JSON.stringify({
          retailer_id: profile.id,
          payment_gateway: gatewayMap[method] || "GLOBAL_PAY",
          latitude: 0,
          longitude: 0,
          items: lineItems,
        }),
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
        throw new Error(errBody?.error || "Failed to create orders from cart");
      }
      const cartData: UnifiedCheckoutResponse = await cartRes.json();
      const supplierOrders = cartData.supplier_orders || [];

      if (["global_pay", "adyen"].includes(method)) {
        for (const so of supplierOrders) {
          const payRes = await apiFetch("/v1/order/card-checkout", {
            method: "POST",
            headers: {
              "Idempotency-Key": `retailer-card-checkout:${so.order_id}:${gatewayMap[method]}`,
            },
            body: JSON.stringify({
              order_id: so.order_id,
              gateway: gatewayMap[method],
              amount: so.total,
              return_url: "retailer-app://orders",
            }),
          });
          if (!payRes.ok)
            throw new Error(
              `Payment initiation failed for order ${so.order_id}`,
            );
          const payData: CardCheckoutResponse = await payRes.json();
          if (payData.payment_url) {
            window.open(payData.payment_url, "_blank");
          }
        }
      } else if (method === "cash") {
        for (const so of supplierOrders) {
          const payRes = await apiFetch("/v1/order/cash-checkout", {
            method: "POST",
            headers: {
              "Idempotency-Key": `retailer-cash-checkout:${so.order_id}`,
            },
            body: JSON.stringify({ order_id: so.order_id }),
          });
          if (!payRes.ok)
            throw new Error(`Cash checkout failed for order ${so.order_id}`);
          await ((await payRes.json()) as Promise<CashCheckoutResponse>);
        }
      }

      clearCart();
      onClose();
      router.push("/orders");
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Checkout failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <AnimatePresence>
      {isOpen && (
        <div className="fixed inset-0 z-[100] flex items-center justify-center p-4">
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="absolute inset-0 bg-[#0a0a0a]/60 backdrop-blur-md"
            onClick={onClose}
          />

          <motion.div
            initial={{ scale: 0.95, opacity: 0, y: 20 }}
            animate={{ scale: 1, opacity: 1, y: 0 }}
            exit={{ scale: 0.95, opacity: 0, y: 20 }}
            className="relative w-full max-w-2xl bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-3xl shadow-2xl overflow-hidden flex flex-col max-h-[90vh]"
          >
            {/* Header */}
            <div className="px-8 py-6 border-b border-[var(--desk-border)] flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-xl bg-[var(--desk-accent-soft)] text-[var(--desk-accent)] flex items-center justify-center">
                  <ShieldCheck size={22} />
                </div>
                <div>
                  <h2 className="md-typescale-title-large font-bold text-[var(--desk-text-primary)]">
                    Trade Settlement
                  </h2>
                  <p className="text-[10px] font-bold text-[var(--desk-text-tertiary)] uppercase tracking-widest">
                    Secure Operational Protocol
                  </p>
                </div>
              </div>
              <button
                onClick={onClose}
                className="w-10 h-10 rounded-full hover:bg-[var(--desk-surface-subtle)] flex items-center justify-center transition-colors"
              >
                <X size={20} />
              </button>
            </div>

            <div className="flex-1 overflow-y-auto p-8 space-y-8">
              {error && (
                <div className="p-4 rounded-xl bg-red-50 border border-red-100 flex items-start gap-3">
                  <XCircle className="text-red-500 mt-0.5" size={18} />
                  <div>
                    <p className="text-red-800 font-bold text-sm">
                      Execution Interrupted
                    </p>
                    <p className="text-red-600 text-xs mt-0.5">{error}</p>
                    {oosItems.length > 0 && (
                      <ul className="mt-2 text-xs text-red-500 list-disc pl-4 space-y-1">
                        {oosItems.map((sku) => (
                          <li key={sku}>{sku}</li>
                        ))}
                      </ul>
                    )}
                  </div>
                </div>
              )}

              <div className="p-6 rounded-2xl bg-[var(--desk-text-primary)] text-white shadow-xl flex items-center justify-between">
                <div>
                  <p className="text-xs font-bold uppercase tracking-widest opacity-60 mb-2">
                    Total Settlement Amount
                  </p>
                  <h3 className="md-typescale-display-small font-bold tabular-nums">
                    {total.toLocaleString()}{" "}
                    <small className="text-sm opacity-40 ml-0.5 uppercase">
                      UZS
                    </small>
                  </h3>
                </div>
                <div className="text-right">
                  <div className="flex items-center gap-2 justify-end mb-2">
                    <Truck size={16} className="opacity-60" />
                    <span className="text-xs font-bold uppercase tracking-widest opacity-60">
                      Estimated Arrival
                    </span>
                  </div>
                  <p className="md-typescale-title-medium font-bold uppercase tracking-tighter">
                    Immediate Route
                  </p>
                </div>
              </div>

              <div>
                <h3 className="md-typescale-label-small uppercase tracking-[0.2em] text-[var(--desk-text-tertiary)] mb-4">
                  Select Settlement Gateway
                </h3>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div
                    onClick={() => setMethod("cash")}
                    className={`p-5 rounded-2xl border-2 transition-all cursor-pointer group ${method === "cash" ? "border-[var(--desk-accent)] bg-[var(--desk-accent-soft)] shadow-md" : "border-[var(--desk-border)] hover:border-[var(--desk-border-strong)] bg-[var(--desk-surface)]"}`}
                  >
                    <div
                      className={`w-12 h-12 rounded-xl flex items-center justify-center mb-4 transition-colors ${method === "cash" ? "bg-[var(--desk-accent)] text-white" : "bg-[var(--desk-surface-subtle)] text-[var(--desk-text-tertiary)] group-hover:text-[var(--desk-text-primary)]"}`}
                    >
                      <Ticket size={24} />
                    </div>
                    <h4
                      className={`md-typescale-title-small font-bold transition-colors ${method === "cash" ? "text-[var(--desk-accent)]" : "text-[var(--desk-text-primary)]"}`}
                    >
                      Physical Settlement
                    </h4>
                    <p className="md-typescale-body-small text-[var(--desk-text-tertiary)] mt-1">
                      Cash payment on node arrival.
                    </p>
                  </div>

                  <div
                    onClick={() => setMethod("global_pay")}
                    className={`p-5 rounded-2xl border-2 transition-all cursor-pointer group ${method === "global_pay" ? "border-[var(--desk-accent)] bg-[var(--desk-accent-soft)] shadow-md" : "border-[var(--desk-border)] hover:border-[var(--desk-border-strong)] bg-[var(--desk-surface)]"}`}
                  >
                    <div
                      className={`w-12 h-12 rounded-xl flex items-center justify-center mb-4 transition-colors ${method === "global_pay" ? "bg-[var(--desk-accent)] text-white" : "bg-[var(--desk-surface-subtle)] text-[var(--desk-text-tertiary)] group-hover:text-[var(--desk-text-primary)]"}`}
                    >
                      <CreditCard size={24} />
                    </div>
                    <h4
                      className={`md-typescale-title-small font-bold transition-colors ${method === "global_pay" ? "text-[var(--desk-accent)]" : "text-[var(--desk-text-primary)]"}`}
                    >
                      Network Transaction
                    </h4>
                    <p className="md-typescale-body-small text-[var(--desk-text-tertiary)] mt-1">
                      Instant digital credit settlement.
                    </p>
                  </div>

                  <div
                    onClick={() => setMethod("adyen")}
                    className={`p-5 rounded-2xl border-2 transition-all cursor-pointer group ${method === "adyen" ? "border-[var(--desk-accent)] bg-[var(--desk-accent-soft)] shadow-md" : "border-[var(--desk-border)] hover:border-[var(--desk-border-strong)] bg-[var(--desk-surface)]"}`}
                  >
                    <div
                      className={`w-12 h-12 rounded-xl flex items-center justify-center mb-4 transition-colors ${method === "adyen" ? "bg-[var(--desk-accent)] text-white" : "bg-[var(--desk-surface-subtle)] text-[var(--desk-text-tertiary)] group-hover:text-[var(--desk-text-primary)]"}`}
                    >
                      <CreditCard size={24} />
                    </div>
                    <h4
                      className={`md-typescale-title-small font-bold transition-colors ${method === "adyen" ? "text-[var(--desk-accent)]" : "text-[var(--desk-text-primary)]"}`}
                    >
                      Adyen Checkout
                    </h4>
                    <p className="md-typescale-body-small text-[var(--desk-text-tertiary)] mt-1">
                      International card settlement via Adyen.
                    </p>
                  </div>
                </div>
              </div>
            </div>

            <div className="px-8 py-6 border-t border-[var(--desk-border)] bg-[var(--desk-surface-subtle)] flex items-center justify-between">
              <button
                onClick={onClose}
                disabled={loading}
                className="text-[var(--desk-text-tertiary)] font-bold md-typescale-label-large hover:text-[var(--desk-text-primary)] transition-colors disabled:opacity-30"
              >
                Cancel Protocol
              </button>

              <Button
                onClick={handleCheckout}
                disabled={loading || items.length === 0}
                className="bg-[var(--desk-text-primary)] text-white font-bold h-12 px-10 rounded-xl shadow-xl transition-all hover:scale-105 active:scale-95 disabled:opacity-30 disabled:hover:scale-100 flex items-center gap-3"
              >
                {loading ? (
                  <Loader2 size={18} className="animate-spin" />
                ) : (
                  <>
                    Execute Order <ChevronRight size={18} />
                  </>
                )}
              </Button>
            </div>
          </motion.div>
        </div>
      )}
    </AnimatePresence>
  );
}

function XCircle({ className, size }: { className?: string; size?: number }) {
  return <X className={className} size={size} />;
}
