"use client";

import { useEffect, useState } from "react";
import { Loader2, X } from "lucide-react";
import { usePortalLocale } from "@/lib/i18n";
import { getSellerMandate, updateSellerMandate } from "@/lib/api";
import {
  formatWholeSom,
  minorToSomInput,
  somInputToMinor,
  W2_COPY,
  type RetailerDailyUsage,
} from "@/lib/retailer-w2";

interface SellerMandateModalProps {
  member: {
    user_id: string;
    name: string;
    retailer_role: string;
  } | null;
  onClose: () => void;
  onSaved: (message: string) => void;
}

export function SellerMandateModal({
  member,
  onClose,
  onSaved,
}: SellerMandateModalProps) {
  const locale = usePortalLocale();
  const w2 = W2_COPY[locale] ?? W2_COPY.en;

  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [canReceiveGoods, setCanReceiveGoods] = useState(true);
  const [canPayCash, setCanPayCash] = useState(true);
  const [canUseCredit, setCanUseCredit] = useState(true);
  const [dailyCreditCapSom, setDailyCreditCapSom] = useState("0");
  const [dailyCashCapSom, setDailyCashCapSom] = useState("0");
  const [maxSingleOrderSom, setMaxSingleOrderSom] = useState("0");
  const [dailyUsage, setDailyUsage] = useState<RetailerDailyUsage | null>(null);

  useEffect(() => {
    if (!member) return;
    let cancelled = false;
    setLoading(true);
    setError(null);
    getSellerMandate(member.user_id)
      .then((res) => {
        if (cancelled) return;
        setCanReceiveGoods(res.mandate.can_receive_goods);
        setCanPayCash(res.mandate.can_pay_cash);
        setCanUseCredit(res.mandate.can_use_credit);
        setDailyCreditCapSom(minorToSomInput(res.mandate.daily_credit_cap_minor));
        setDailyCashCapSom(minorToSomInput(res.mandate.daily_cash_cap_minor));
        setMaxSingleOrderSom(minorToSomInput(res.mandate.max_single_order_minor));
        setDailyUsage(res.daily_usage ?? null);
      })
      .catch((err) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to load mandate");
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [member]);

  if (!member) return null;

  const handleSave = async () => {
    const creditMinor = somInputToMinor(dailyCreditCapSom);
    const cashMinor = somInputToMinor(dailyCashCapSom);
    const singleMinor = somInputToMinor(maxSingleOrderSom);
    if (creditMinor === null || cashMinor === null || singleMinor === null) {
      setError("Enter valid non-negative integer so'm amounts (0 for unlimited).");
      return;
    }

    setSaving(true);
    setError(null);
    try {
      await updateSellerMandate(member.user_id, {
        can_receive_goods: canReceiveGoods,
        can_pay_cash: canPayCash,
        can_use_credit: canUseCredit,
        daily_credit_cap_minor: creditMinor,
        daily_cash_cap_minor: cashMinor,
        max_single_order_minor: singleMinor,
      });
      onSaved(`${w2.mandateSaved}: ${member.name}`);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save mandate");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4 backdrop-blur-sm">
      <div className="w-full max-w-md rounded-2xl border border-border bg-card p-5 shadow-xl space-y-4">
        <div className="flex items-start justify-between gap-3">
          <div>
            <h3 className="text-base font-semibold">{w2.mandateTitle}</h3>
            <p className="text-xs text-muted-foreground">
              {member.name} · {member.retailer_role}
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close mandate editor"
            className="rounded-lg p-1.5 text-muted-foreground hover:bg-muted"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {loading ? (
          <div className="flex items-center gap-2 py-6 text-sm text-muted-foreground">
            <Loader2 className="h-4 w-4 animate-spin" />
            Loading mandate...
          </div>
        ) : (
          <>
            {dailyUsage && (
              <div className="rounded-xl border border-border bg-background p-3 text-xs space-y-1">
                <div className="font-medium text-foreground">
                  Today&apos;s usage ({dailyUsage.usage_date || "today"})
                </div>
                <div className="text-muted-foreground">
                  Credit: {formatWholeSom(dailyUsage.spent_credit_minor)} · Cash:{" "}
                  {formatWholeSom(dailyUsage.spent_cash_minor)} · Orders:{" "}
                  {dailyUsage.order_count}
                </div>
              </div>
            )}

            <div className="space-y-2 text-sm">
              <label className="flex items-center justify-between gap-2 rounded-lg border border-border px-3 py-2">
                <span>{w2.canReceiveGoods}</span>
                <input
                  type="checkbox"
                  checked={canReceiveGoods}
                  onChange={(e) => setCanReceiveGoods(e.target.checked)}
                />
              </label>
              <label className="flex items-center justify-between gap-2 rounded-lg border border-border px-3 py-2">
                <span>{w2.canPayCash}</span>
                <input
                  type="checkbox"
                  checked={canPayCash}
                  onChange={(e) => setCanPayCash(e.target.checked)}
                />
              </label>
              <label className="flex items-center justify-between gap-2 rounded-lg border border-border px-3 py-2">
                <span>{w2.canUseCredit}</span>
                <input
                  type="checkbox"
                  checked={canUseCredit}
                  onChange={(e) => setCanUseCredit(e.target.checked)}
                />
              </label>
            </div>

            <div className="space-y-3 text-xs">
              <label className="block space-y-1">
                <span className="font-medium text-foreground">{w2.dailyCreditCapSom}</span>
                <input
                  type="text"
                  inputMode="numeric"
                  value={dailyCreditCapSom}
                  onChange={(e) => setDailyCreditCapSom(e.target.value)}
                  className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm"
                />
              </label>
              <label className="block space-y-1">
                <span className="font-medium text-foreground">{w2.dailyCashCapSom}</span>
                <input
                  type="text"
                  inputMode="numeric"
                  value={dailyCashCapSom}
                  onChange={(e) => setDailyCashCapSom(e.target.value)}
                  className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm"
                />
              </label>
              <label className="block space-y-1">
                <span className="font-medium text-foreground">{w2.maxSingleOrderSom}</span>
                <input
                  type="text"
                  inputMode="numeric"
                  value={maxSingleOrderSom}
                  onChange={(e) => setMaxSingleOrderSom(e.target.value)}
                  className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm"
                />
              </label>
            </div>

            {error && (
              <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700">
                {error}
              </div>
            )}

            <div className="flex justify-end gap-2 pt-2">
              <button
                type="button"
                onClick={onClose}
                className="rounded-lg border border-border px-4 py-2 text-xs font-medium hover:bg-muted"
              >
                Cancel
              </button>
              <button
                type="button"
                disabled={saving}
                onClick={() => void handleSave()}
                className="inline-flex items-center gap-1.5 rounded-lg bg-foreground px-4 py-2 text-xs font-medium text-background disabled:opacity-50"
              >
                {saving && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
                {w2.mandateSave}
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
