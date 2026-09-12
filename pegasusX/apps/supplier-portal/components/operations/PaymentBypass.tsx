"use client";

import { usePortalT } from "@/lib/i18n";
import React from 'react';
import { PageSection } from '@/components/PageSection';

interface PaymentBypassProps {
  orderId: string;
  bypassReason: string;
  bypassToken: string | null;
  confirmBypass: boolean;
  bypassing: boolean;
  onOrderIdChange: (v: string) => void;
  onBypassReasonChange: (v: string) => void;
  onConfirmBypassChange: (v: boolean) => void;
  onPaymentBypass: () => void;
}

export function PaymentBypass({
  orderId,
  bypassReason,
  bypassToken,
  confirmBypass,
  bypassing,
  onOrderIdChange,
  onBypassReasonChange,
  onConfirmBypassChange,
  onPaymentBypass,
}: PaymentBypassProps) {
  const t = usePortalT();
  return (
    <PageSection title={t("supplier_portal.operations.payment_bypass.text.payment_bypass")} description={t("supplier_portal.residual.text.issue_a_one_time_driver_token_for_awaiting_payment_orders")}>
      <div className="space-y-3">
        <label className="block space-y-1">
          <span className="md-typescale-label-medium" style={{ color: "var(--desk-text-secondary)" }}>
            Order ID
          </span>
          <input
            className="md-input-outlined w-full font-mono"
            placeholder={t("supplier_portal.operations.payment_bypass.text.order_id_awaiting_payment")}
            value={orderId}
            onChange={(e) => onOrderIdChange(e.target.value)}
          />
        </label>
        <label className="block space-y-1">
          <span className="md-typescale-label-medium" style={{ color: "var(--desk-text-secondary)" }}>
            Reason (optional)
          </span>
          <input
            className="md-input-outlined w-full"
            placeholder={t("supplier_portal.admin.control_center.field.reason")}
            value={bypassReason}
            onChange={(e) => onBypassReasonChange(e.target.value)}
          />
        </label>
        {confirmBypass ? (
          <div
            className="space-y-3 p-4 rounded-xl"
            style={{ border: "1px solid var(--desk-border)", background: "var(--desk-surface-raised)" }}
          >
            <p className="md-typescale-body-medium">
              Issue bypass for <span className="font-mono">{orderId.trim()}</span>? Order must be AWAITING_PAYMENT.
            </p>
            <div className="flex flex-wrap gap-2">
              <button type="button" className="md-btn md-btn-filled" onClick={onPaymentBypass} disabled={bypassing}>
                {bypassing ? "Issuing…" : "Confirm issue"}
              </button>
              <button
                type="button"
                className="md-btn md-btn-outlined"
                onClick={() => onConfirmBypassChange(false)}
                disabled={bypassing}
              >
                Cancel
              </button>
            </div>
          </div>
        ) : (
          <button
            type="button"
            className="md-btn md-btn-outlined"
            onClick={() => onConfirmBypassChange(true)}
            disabled={!orderId.trim()}
          >
            Issue bypass token
          </button>
        )}
        {bypassToken ? (
          <p className="md-typescale-body-medium">
            Driver token: <span className="font-mono">{bypassToken}</span>
          </p>
        ) : null}
      </div>
    </PageSection>
  );
}
