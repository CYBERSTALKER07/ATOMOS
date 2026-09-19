"use client";

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { Building2, Download, Loader2, RefreshCw } from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { apiFetch } from "@/lib/auth";

type HqSummary = {
  day?: string;
  location_count?: number;
  sku_count?: number;
  qty_sold?: number;
  qty_voided?: number;
  gross_minor?: number;
  net_minor?: number;
  currency?: string;
  honest_empty?: boolean;
};

type LocItem = {
  location_id: string;
  qty_sold: number;
  qty_voided: number;
  gross_minor: number;
  net_minor: number;
  currency: string;
};

type SkuItem = {
  sku_id: string;
  qty_sold: number;
  qty_voided: number;
  net_minor: number;
  is_local?: boolean;
};

function formatMoney(minor: number) {
  return (minor / 100).toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
}

function todayUTC() {
  return new Date().toISOString().slice(0, 10);
}

/**
 * Wave C2.2 franchise HQ rollup (OWNER/ADMIN).
 * Honest empty when no HQ writers data or HQ_ANALYTICS_ENABLED off (404).
 */
export default function HqPage() {
  const t = usePortalT();
  const [day, setDay] = useState(todayUTC);
  const [summary, setSummary] = useState<HqSummary | null>(null);
  const [byLoc, setByLoc] = useState<LocItem[]>([]);
  const [bySku, setBySku] = useState<SkuItem[]>([]);
  const [balanced, setBalanced] = useState(true);
  const [orgNet, setOrgNet] = useState(0);
  const [disabled, setDisabled] = useState(false);
  const [banner, setBanner] = useState<string | null>(null);
  const [hqError, setHqError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setBusy(true);
    setBanner(null);
    setDisabled(false);
    try {
      const q = `day=${encodeURIComponent(day)}`;
      const [sRes, locRes, skuRes] = await Promise.all([
        apiFetch(`/v1/retailer/hq/summary?${q}`),
        apiFetch(`/v1/retailer/hq/sales-by-location?${q}`),
        apiFetch(`/v1/retailer/hq/sales-by-sku?${q}`),
      ]);
      if (sRes.status === 404 || locRes.status === 404) {
        setDisabled(true);
        setHqError(null);
        setSummary(null);
        setByLoc([]);
        setBySku([]);
        setBanner(
          "HQ analytics is off (HQ_ANALYTICS_ENABLED). Enable flag after backend image with C2.1 writers.",
        );
        return;
      }
      if (sRes.status === 403 || locRes.status === 403) {
        setHqError("HQ requires OWNER or ADMIN with reports.view.");
        return;
      }
      if (!sRes.ok || !locRes.ok || !skuRes.ok) {
        setHqError("hq_failed");
        return;
      }
      setHqError(null);
      setSummary((await sRes.json()) as HqSummary);
      const locJson = (await locRes.json()) as {
        items?: LocItem[];
        balanced?: boolean;
        org_net_minor?: number;
      };
      setByLoc(locJson.items ?? []);
      setBalanced(locJson.balanced !== false);
      setOrgNet(locJson.org_net_minor ?? 0);
      const skuJson = (await skuRes.json()) as { items?: SkuItem[] };
      setBySku(skuJson.items ?? []);
    } catch {
      setHqError("hq_failed");
    } finally {
      setBusy(false);
    }
  }, [day]);

  useEffect(() => {
    void load();
  }, [load]);

  const downloadCSV = async () => {
    const res = await apiFetch(
      `/v1/retailer/hq/export?day=${encodeURIComponent(day)}&format=csv`,
    );
    if (!res.ok) {
      setBanner("Export failed");
      return;
    }
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `hq-sales-${day}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <PageChrome
      title={t("portal.nav.franchise_hq")}
      description={t("retailer_desktop.residual.text.multi_location_pos_sales_rollup_sum_of_locations_equals_org_net_")}
    >
      <div className="mx-auto flex max-w-4xl flex-col gap-6 p-4">
        <div className="flex flex-wrap items-end gap-3">
          <label className="text-sm">
            Day (UTC)
            <input
              type="date"
              className="mt-1 block rounded-lg border border-border bg-background px-3 py-2 text-sm"
              value={day}
              onChange={(e) => setDay(e.target.value)}
            />
          </label>
          <button
            type="button"
            disabled={busy}
            onClick={() => void load()}
            className="inline-flex items-center gap-2 rounded-lg border border-border px-3 py-2 text-sm disabled:opacity-50"
          >
            {busy ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <RefreshCw className="h-4 w-4" />
            )}
            Refresh
          </button>
          <button
            type="button"
            disabled={busy || disabled}
            onClick={() => void downloadCSV()}
            className="inline-flex items-center gap-2 rounded-lg bg-foreground px-3 py-2 text-sm text-background disabled:opacity-50"
          >
            <Download className="h-4 w-4" /> CSV export
          </button>
        </div>

        {banner && (
          <div className="rounded-lg border border-border bg-muted/40 px-3 py-2 text-sm">
            {banner}
          </div>
        )}

        {disabled ? (
          <div className="flex flex-col items-center gap-3 rounded-xl border border-dashed border-border py-16 text-center text-muted-foreground">
            <Building2 className="h-10 w-10 opacity-40" />
            <p className="text-sm">{t("retailer_desktop.hq.text.hq_analytics_not_enabled_for_this_environment")}</p>
          </div>
        ) : hqError ? (
          <p role="alert" className="text-sm text-destructive">
            {hqError}
          </p>
        ) : summary ? (
          <>
            <section className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
              {[
                ["Net sales", formatMoney(summary?.net_minor ?? 0)],
                ["Gross", formatMoney(summary?.gross_minor ?? 0)],
                ["Units sold", String(summary?.qty_sold ?? 0)],
                ["Units voided", String(summary?.qty_voided ?? 0)],
              ].map(([label, val]) => (
                <div
                  key={label}
                  className="rounded-xl border border-border bg-card p-4"
                >
                  <p className="text-xs text-muted-foreground">{label}</p>
                  <p className="mt-1 text-xl font-semibold tabular-nums">{val}</p>
                </div>
              ))}
            </section>

            {summary?.honest_empty && (
              <p className="text-sm text-muted-foreground">
                No HQ rows for {day}. Complete POS sales with C2.1 writers live to populate.
              </p>
            )}

            <section className="rounded-xl border border-border bg-card p-4">
              <div className="mb-3 flex items-center justify-between">
                <h2 className="font-semibold">{t("retailer_desktop.hq.text.sales_by_location")}</h2>
                <span
                  className={`text-xs ${balanced ? "text-emerald-600" : "text-red-600"}`}
                >
                  {balanced
                    ? `Balanced · org net ${formatMoney(orgNet)}`
                    : "Imbalance: sum ≠ org"}
                </span>
              </div>
              {byLoc.length === 0 ? (
                <p className="text-sm text-muted-foreground">{t("retailer_desktop.hq.text.no_locations_with_sales")}</p>
              ) : (
                <table className="w-full text-left text-sm">
                  <thead>
                    <tr className="border-b border-border text-muted-foreground">
                      <th className="py-2 font-medium">{t("portal.nav.location")}</th>
                      <th className="py-2 font-medium">{t("retailer_desktop.hq.text.sold")}</th>
                      <th className="py-2 font-medium">{t("retailer_desktop.hq.text.voided")}</th>
                      <th className="py-2 font-medium">{t("retailer_desktop.hq.text.net")}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {byLoc.map((r) => (
                      <tr key={r.location_id} className="border-b border-border/60">
                        <td className="py-2 font-mono text-xs">{r.location_id}</td>
                        <td className="py-2 tabular-nums">{r.qty_sold}</td>
                        <td className="py-2 tabular-nums">{r.qty_voided}</td>
                        <td className="py-2 tabular-nums">{formatMoney(r.net_minor)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </section>

            <section className="rounded-xl border border-border bg-card p-4">
              <h2 className="mb-3 font-semibold">{t("retailer_desktop.hq.text.sales_by_sku")}</h2>
              {bySku.length === 0 ? (
                <p className="text-sm text-muted-foreground">{t("retailer_desktop.hq.text.no_skus")}</p>
              ) : (
                <table className="w-full text-left text-sm">
                  <thead>
                    <tr className="border-b border-border text-muted-foreground">
                      <th className="py-2 font-medium">SKU</th>
                      <th className="py-2 font-medium">{t("retailer_desktop.hq.text.sold")}</th>
                      <th className="py-2 font-medium">{t("retailer_desktop.hq.text.voided")}</th>
                      <th className="py-2 font-medium">{t("retailer_desktop.hq.text.net")}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {bySku.slice(0, 50).map((r) => (
                      <tr key={r.sku_id} className="border-b border-border/60">
                        <td className="py-2">
                          <span className="font-mono text-xs">{r.sku_id}</span>
                          {r.is_local ? (
                            <span className="ml-2 rounded bg-muted px-1.5 py-0.5 text-[10px]">
                              local
                            </span>
                          ) : null}
                        </td>
                        <td className="py-2 tabular-nums">{r.qty_sold}</td>
                        <td className="py-2 tabular-nums">{r.qty_voided}</td>
                        <td className="py-2 tabular-nums">{formatMoney(r.net_minor)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </section>
          </>
        ) : null}
      </div>
    </PageChrome>
  );
}
