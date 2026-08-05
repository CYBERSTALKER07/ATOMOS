"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { AlertTriangle, Camera, Loader2 } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { apiFetch } from "../lib/auth";
import { uploadClaimPhoto } from "../lib/api";
import { retailerShopClosedResponseKey } from "@pegasusx/api-client";
import { useWebSocket, useWsEvent, type WsMessage } from "../lib/ws";

type ShopClosedAlert = {
  order_id: string;
  driver_name: string;
  attempt_id?: string;
  options: string[];
};

const DEFAULT_OPTIONS = [
  "OPEN_NOW",
  "5_MIN",
  "CALL_ME",
  "CLOSED_TODAY",
  "RESCHEDULE",
  "CREDIT_LEAVE",
  "CANCEL",
  "AUTHORIZE_BYPASS",
];

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
    case "RESCHEDULE":
      return "Reschedule delivery";
    case "CREDIT_LEAVE":
      return "Leave on credit";
    case "CANCEL":
      return "Cancel remaining";
    case "AUTHORIZE_BYPASS":
      return "Authorize bypass offload";
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

function formatShopClosedError(raw: string): string {
  if (raw.includes("photo_url_required_for_bypass")) {
    return "Doorway / drop-off photo is required to authorize bypass.";
  }
  return raw || "Could not submit shop status";
}

export default function ShopClosedModal() {
  const { reconnectEpoch } = useWebSocket();
  const [alert, setAlert] = useState<ShopClosedAlert | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [bypassPending, setBypassPending] = useState(false);
  const [bypassPhotoUrl, setBypassPhotoUrl] = useState<string | null>(null);
  const [uploading, setUploading] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (reconnectEpoch > 0 && submitting) {
      setSubmitting(false);
      setError("Connection restored — verify response before retrying.");
    }
  }, [reconnectEpoch, submitting]);

  const resetBypass = useCallback(() => {
    setBypassPending(false);
    setBypassPhotoUrl(null);
    setUploading(false);
  }, []);

  const openAlert = useCallback(
    (msg: WsMessage) => {
      const next = messageToAlert(msg);
      if (next) {
        setAlert(next);
        setError(null);
        resetBypass();
      }
    },
    [resetBypass],
  );

  useWsEvent("SHOP_CLOSED", openAlert);
  useWsEvent("SHOP_CLOSED_ALERT", openAlert);

  useWsEvent(
    "SHOP_CLOSED_RESOLVED",
    useCallback((msg: WsMessage) => {
      const orderId = typeof msg.order_id === "string" ? msg.order_id : "";
      setAlert((current) => {
        if (!current) return current;
        if (!orderId || orderId === current.order_id) return null;
        return current;
      });
      resetBypass();
    }, [resetBypass]),
  );

  useWsEvent(
    "SHOP_CLOSED_RESPONSE",
    useCallback((msg: WsMessage) => {
      const orderId = typeof msg.order_id === "string" ? msg.order_id : "";
      setAlert((current) => {
        if (!current) return current;
        if (!orderId || orderId === current.order_id) return null;
        return current;
      });
      resetBypass();
    }, [resetBypass]),
  );

  const respond = useCallback(
    async (option: string, photoUrl?: string) => {
      if (!alert || submitting) return;
      if (option === "AUTHORIZE_BYPASS" && !photoUrl?.trim()) {
        setBypassPending(true);
        setError("Doorway / drop-off photo is required to authorize bypass.");
        return;
      }
      setSubmitting(true);
      setError(null);
      try {
        const body: Record<string, string> = {
          order_id: alert.order_id,
          response: option,
        };
        if (photoUrl?.trim()) {
          body.photo_url = photoUrl.trim();
        }
        const res = await apiFetch("/v1/retailer/shop-closed-response", {
          method: "POST",
          headers: {
            "Idempotency-Key": retailerShopClosedResponseKey(
              alert.order_id,
              option,
            ),
          },
          body: JSON.stringify(body),
        });
        if (!res.ok) {
          const text = await res.text();
          throw new Error(formatShopClosedError(text));
        }
        setAlert(null);
        resetBypass();
      } catch (err) {
        setError(
          err instanceof Error
            ? formatShopClosedError(err.message)
            : "Could not submit shop status",
        );
      } finally {
        setSubmitting(false);
      }
    },
    [alert, submitting, resetBypass],
  );

  const onBypassFile = useCallback(
    async (file: File | null) => {
      if (!file || !alert) return;
      setUploading(true);
      setError(null);
      try {
        const url = await uploadClaimPhoto(file, alert.order_id);
        setBypassPhotoUrl(url);
      } catch (err) {
        setBypassPhotoUrl(null);
        setError(
          err instanceof Error ? err.message : "Photo upload failed",
        );
      } finally {
        setUploading(false);
      }
    },
    [alert],
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
            {bypassPending && (
              <div className="mt-4 rounded-xl border border-[var(--desk-border)] bg-[var(--desk-bg)] p-3">
                <p className="text-sm text-[var(--desk-text-secondary)]">
                  Doorway / drop-off proof is required for authorize bypass.
                </p>
                <input
                  ref={fileRef}
                  type="file"
                  accept="image/*"
                  capture="environment"
                  className="hidden"
                  onChange={(e) => {
                    void onBypassFile(e.target.files?.[0] ?? null);
                    e.target.value = "";
                  }}
                />
                <div className="mt-3 flex flex-wrap items-center gap-2">
                  <button
                    type="button"
                    disabled={uploading || submitting}
                    onClick={() => fileRef.current?.click()}
                    className="portal-btn portal-btn--secondary inline-flex h-10 items-center gap-2 rounded-xl px-3 text-sm"
                  >
                    <Camera size={16} />
                    {uploading ? "Uploading…" : bypassPhotoUrl ? "Replace photo" : "Take or choose photo"}
                  </button>
                  {bypassPhotoUrl && (
                    <button
                      type="button"
                      disabled={submitting || uploading}
                      onClick={() => void respond("AUTHORIZE_BYPASS", bypassPhotoUrl)}
                      className="portal-btn portal-btn--primary h-10 rounded-xl px-3 text-sm"
                    >
                      Confirm bypass
                    </button>
                  )}
                  <button
                    type="button"
                    disabled={submitting}
                    onClick={resetBypass}
                    className="text-sm text-[var(--desk-text-secondary)] underline"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            )}
            <div className="mt-5 flex flex-col gap-2">
              {alert.options.map((option) => (
                <button
                  key={option}
                  type="button"
                  disabled={submitting || uploading}
                  onClick={() => {
                    if (option === "AUTHORIZE_BYPASS") {
                      setBypassPending(true);
                      setError(null);
                      return;
                    }
                    void respond(option);
                  }}
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
