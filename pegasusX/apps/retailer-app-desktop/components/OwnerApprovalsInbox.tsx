"use client";

import { useState } from "react";
import { Check, Loader2, RefreshCw, ShieldAlert, X } from "lucide-react";
import { usePortalLocale } from "@/lib/i18n";
import {
  formatTrippedRuleLabel,
  formatWholeSom,
  W2_COPY,
  type RetailerApprovalItem,
} from "@/lib/retailer-w2";

interface OwnerApprovalsInboxProps {
  items: RetailerApprovalItem[];
  statusFilter: string;
  loading: boolean;
  error: string | null;
  banner: string | null;
  decidingId: string | null;
  onStatusFilterChange: (status: string) => void;
  onRefresh: () => void;
  onApprove: (requestId: string, reason?: string) => void;
  onReject: (requestId: string, reason?: string) => void;
}

const STATUS_TABS = ["PENDING_OWNER", "APPROVED", "REJECTED", "EXPIRED"] as const;

export function OwnerApprovalsInbox({
  items,
  statusFilter,
  loading,
  error,
  banner,
  decidingId,
  onStatusFilterChange,
  onRefresh,
  onApprove,
  onReject,
}: OwnerApprovalsInboxProps) {
  const locale = usePortalLocale();
  const w2 = W2_COPY[locale] ?? W2_COPY.en;
  const [reasons, setReasons] = useState<Record<string, string>>({});

  return (
    <section
      id="approvals"
      className="rounded-xl border border-border bg-card p-4 space-y-4"
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h2 className="flex items-center gap-2 font-semibold">
            <ShieldAlert className="h-4 w-4 text-amber-600" />
            {w2.inboxTitle}
          </h2>
          <p className="text-xs text-muted-foreground">{w2.inboxSubtitle}</p>
        </div>
        <button
          type="button"
          disabled={loading}
          onClick={onRefresh}
          className="inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs hover:bg-muted disabled:opacity-50"
        >
          <RefreshCw className={`h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} />
          Refresh
        </button>
      </div>

      <div className="flex flex-wrap gap-2">
        {STATUS_TABS.map((status) => (
          <button
            key={status}
            type="button"
            onClick={() => onStatusFilterChange(status)}
            className={`rounded-full border px-3 py-1 text-xs font-medium transition-colors ${
              statusFilter === status
                ? "border-foreground bg-foreground text-background"
                : "border-border bg-background text-muted-foreground hover:bg-muted"
            }`}
          >
            {status.replace(/_/g, " ")}
          </button>
        ))}
      </div>

      {banner && (
        <div className="rounded-lg border border-emerald-200 bg-emerald-50 px-3 py-2 text-xs text-emerald-800">
          {banner}
        </div>
      )}

      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700">
          {error}
        </div>
      )}

      {loading && items.length === 0 ? (
        <div className="flex items-center gap-2 text-sm text-muted-foreground py-2">
          <Loader2 className="h-4 w-4 animate-spin" />
          Loading approvals...
        </div>
      ) : items.length === 0 ? (
        <p className="text-sm text-muted-foreground py-2">{w2.noPendingApprovals}</p>
      ) : (
        <ul className="space-y-3">
          {items.map((item) => {
            const isPending = item.status.toUpperCase() === "PENDING_OWNER";
            const isBusy = decidingId === item.request_id;
            const reasonVal = reasons[item.request_id] ?? "";
            return (
              <li
                key={item.request_id}
                className="rounded-xl border border-border bg-background p-3.5 space-y-2.5"
              >
                <div className="flex flex-wrap items-start justify-between gap-2">
                  <div>
                    <div className="font-medium text-sm">
                      {item.requested_by_name?.trim() || item.requested_by_user}
                    </div>
                    <div className="text-xs text-muted-foreground">
                      {formatTrippedRuleLabel(item.tripped_rule)}
                      {item.payment_method ? ` · ${item.payment_method}` : ""}
                      {item.payload_kind ? ` · ${item.payload_kind}` : ""}
                    </div>
                  </div>
                  <div className="text-right">
                    <div className="font-semibold text-sm">
                      {formatWholeSom(item.amount_minor)}
                    </div>
                    <span className="inline-block rounded-full bg-muted px-2 py-0.5 text-[10px] font-medium uppercase">
                      {item.status}
                    </span>
                  </div>
                </div>

                {item.decision_reason && (
                  <p className="text-xs text-muted-foreground">
                    Note: {item.decision_reason}
                  </p>
                )}

                {isPending && (
                  <div className="flex flex-wrap items-center gap-2 pt-1">
                    <input
                      type="text"
                      value={reasonVal}
                      onChange={(e) =>
                        setReasons((prev) => ({
                          ...prev,
                          [item.request_id]: e.target.value,
                        }))
                      }
                      placeholder={w2.rejectReasonPlaceholder}
                      className="min-w-[180px] flex-1 rounded-lg border border-border bg-card px-3 py-1.5 text-xs"
                    />
                    <button
                      type="button"
                      disabled={isBusy}
                      onClick={() => onApprove(item.request_id, reasonVal)}
                      className="inline-flex items-center gap-1 rounded-lg bg-emerald-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-emerald-700 disabled:opacity-50"
                    >
                      {isBusy ? (
                        <Loader2 className="h-3.5 w-3.5 animate-spin" />
                      ) : (
                        <Check className="h-3.5 w-3.5" />
                      )}
                      {w2.approve}
                    </button>
                    <button
                      type="button"
                      disabled={isBusy}
                      onClick={() => onReject(item.request_id, reasonVal)}
                      className="inline-flex items-center gap-1 rounded-lg border border-red-300 bg-red-50 px-3 py-1.5 text-xs font-medium text-red-700 hover:bg-red-100 disabled:opacity-50"
                    >
                      <X className="h-3.5 w-3.5" />
                      {w2.reject}
                    </button>
                  </div>
                )}
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}
