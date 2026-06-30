"use client";

import { useCallback, useEffect, useState } from "react";
import { AlertTriangle, Loader2 } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { apiFetch } from "../lib/auth";
import { retailerShopClosedResponseKey } from "@pegasusx/api-client";
import { useWebSocket, useWsEvent, type WsMessage } from "../lib/ws";

type ShopClosedAlert = {
  order_id: string;
  driver_name: string;
  attempt_id?: string;
  options: string[];
};

const DEFAULT_OPTIONS = ["OPEN_NOW", "5_MIN", "CALL_ME", "CLOSED_TODAY"];

function optionLabel(option: string): string {
  switch (option) {
    case "OPEN_NOW":
      return "I am open now";
    case "5_MIN":
      return "I will be back in 5 mins";
    case "CALL_ME":
      return "Call me";
    case "CLOSED_TODAY":
      return "Closed for the day";
    default:
      return option;
  }
}

function messageToAlert(msg: WsMessage): ShopClosedAlert | null {
  const orderId = typeof msg.order_id === "string" ? msg.order_id : "";
  if (!orderId) return null;
  const driverName =
    (typeof msg.driver_name === "string" && msg.driver_name) ||
    (typeof msg.driver_id === "string" && msg.driver_id) ||
    "Driver";
  const options = Array.isArray(msg.options)
    ? msg.options.filter((value): value is string => typeof value === "string")
    : DEFAULT_OPTIONS;
  return {
    order_id: orderId,
    driver_name: driverName,
    attempt_id: typeof msg.attempt_id === "string" ? msg.attempt_id : undefined,
    options: options.length > 0 ? options : DEFAULT_OPTIONS,
  };
}

export default function ShopClosedModal() {
  const { reconnectEpoch } = useWebSocket();
  const [alert, setAlert] = useState<ShopClosedAlert | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (reconnectEpoch > 0 && submitting) {
      setSubmitting(false);
      setError("Connection restored — verify response before retrying.");
    }
  }, [reconnectEpoch, submitting]);

  const openAlert = useCallback((msg: WsMessage) => {
    const next = messageToAlert(msg);
    if (next) {
      setAlert(next);
      setError(null);
    }
  }, []);

  useWsEvent("SHOP_CLOSED", openAlert);
  useWsEvent("SHOP_CLOSED_ALERT", openAlert);

  useWsEvent(
    "SHOP_CLOSED_RESOLVED",
    useCallback(
      (msg: WsMessage) => {
        const orderId = typeof msg.order_id === "string" ? msg.order_id : "";
        setAlert((current) => {
          if (!current) return current;
          if (!orderId || orderId === current.order_id) return null;
          return current;
        });
      },
      [],
    ),
  );

  useWsEvent(
    "SHOP_CLOSED_RESPONSE",
    useCallback(
      (msg: WsMessage) => {
        const orderId = typeof msg.order_id === "string" ? msg.order_id : "";
        setAlert((current) => {
          if (!current) return current;
          if (!orderId || orderId === current.order_id) return null;
          return current;
        });
      },
      [],
    ),
  );

  const respond = useCallback(
    async (option: string) => {
      if (!alert || submitting) return;
      setSubmitting(true);
      setError(null);
      try {
        const res = await apiFetch("/v1/retailer/shop-closed-response", {
          method: "POST",
          headers: {
            "Idempotency-Key": retailerShopClosedResponseKey(alert.order_id, option),
          },
          body: JSON.stringify({
            order_id: alert.order_id,
            response: option,
          }),
        });
        if (!res.ok) {
          const text = await res.text();
          throw new Error(text || "Could not submit shop status");
        }
        setAlert(null);
      } catch (err) {
        setError(
          err instanceof Error ? err.message : "Could not submit shop status",
        );
      } finally {
        setSubmitting(false);
      }
    },
    [alert, submitting],
  );

  return (
    <AnimatePresence>
      {alert && (
        <motion.div
          className="fixed inset-0 z-[120] flex items-end justify-center bg-black/45 p-4 md:items-center"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
        >
          <motion.div
            className="w-full max-w-lg rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface)] p-6 shadow-[var(--shadow-lg)]"
            initial={{ opacity: 0, y: 24 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 16 }}
            transition={{ type: "spring", stiffness: 420, damping: 32 }}
          >
            <div className="mb-4 flex items-center gap-2 text-[var(--desk-warning)]">
              <AlertTriangle size={18} />
              <h2 className="md-typescale-title-medium font-light text-[var(--desk-text-primary)]">
                Shop Status Required
              </h2>
            </div>
            <p className="md-typescale-body-medium text-[var(--desk-text-secondary)]">
              Driver {alert.driver_name} reported your shop is closed. Confirm
              your status for order #{alert.order_id.slice(-6)}.
            </p>
            {error && (
              <p className="mt-3 text-sm font-semibold text-red-600">{error}</p>
            )}
            <div className="mt-5 flex flex-col gap-2">
              {alert.options.map((option) => (
                <button
                  key={option}
                  type="button"
                  disabled={submitting}
                  onClick={() => void respond(option)}
                  className="portal-btn portal-btn--primary h-11 rounded-xl font-light disabled:opacity-60"
                >
                  {submitting ? (
                    <span className="inline-flex items-center gap-2">
                      <Loader2 size={16} className="animate-spin" />
                      Submitting...
                    </span>
                  ) : (
                    optionLabel(option)
                  )}
                </button>
              ))}
            </div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}
