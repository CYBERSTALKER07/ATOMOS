"use client";

import { usePortalT } from "@/lib/i18n";
import { useState } from "react";
import { PageChrome } from "@/components/PageChrome";
import { PageSection } from "@/components/PageSection";
import { useLiveData } from "@/lib/hooks";
import { supplierFetch } from "@/lib/auth";
import { sessionPackCurrency } from "@pegasusx/api-client";
import {
  RefreshCw,
  Plus,
  Calendar,
  Percent,
  CheckCircle,
  FileJson
} from "lucide-react";

type TaxRegimeVersion = {
  id: string;
  country_code: string;
  effective_from: string;
  effective_to?: string;
  currency: string;
  vat_rate_bps: number;
  simplified: boolean;
  rules_json?: any;
  created_at: string;
};

type TaxRegimeResponse = {
  regimes: TaxRegimeVersion[];
};

export default function TaxRegimesPage() {
  const t = usePortalT();
  const [country] = useState("UZ");
  const { data, loading, isRefreshing, mutate } = useLiveData<TaxRegimeResponse>(
    `/v1/admin/tax-regimes?country=${country}&limit=50`
  );
  const [isCreating, setIsCreating] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  
  const [formData, setFormData] = useState({
    country_code: "UZ",
    currency: sessionPackCurrency(),
    effective_from: "",
    vat_rate_bps: "1200",
    simplified: false
  });

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      const payload = {
        country_code: formData.country_code,
        currency: formData.currency,
        effective_from: new Date(formData.effective_from).toISOString(),
        vat_rate_bps: parseInt(formData.vat_rate_bps, 10),
        simplified: formData.simplified
      };
      
      const res = await supplierFetch("/v1/admin/tax-regimes", {
        method: "POST",
        body: JSON.stringify(payload)
      });
      
      if (!res.ok) throw new Error("Failed to create tax regime");
      
      setIsCreating(false);
      setFormData({
        country_code: "UZ",
        currency: sessionPackCurrency(),
        effective_from: "",
        vat_rate_bps: "1200",
        simplified: false
      });
      mutate();
    } catch (err: any) {
      console.error(err);
      alert("Error creating tax regime");
    } finally {
      setIsSubmitting(false);
    }
  };

  const formatDate = (isoStr: string | undefined) => {
    if (!isoStr) return "Ongoing";
    return new Date(isoStr).toLocaleDateString(undefined, {
      year: "numeric", month: "short", day: "numeric",
      hour: "2-digit", minute: "2-digit"
    });
  };

  return (
    <PageChrome
      title={t("portal.nav.tax_regimes")}
      description={t("supplier_portal.residual.text.manage_historical_and_future_tax_rates")}
      actions={
        <div className="flex gap-2">
          <button
            type="button"
            onClick={() => setIsCreating(!isCreating)}
            className="portal-btn portal-btn--primary h-11 px-5 rounded-xl font-light"
          >
            <Plus size={16} className="mr-2" />
            New Regime
          </button>
          <button
            type="button"
            disabled={loading || isRefreshing}
            onClick={mutate}
            className="portal-btn portal-btn--ghost h-11 px-5 rounded-xl font-light"
          >
            <RefreshCw
              size={16}
              className={`mr-2 ${isRefreshing ? "animate-spin" : ""}`}
            />
            {isRefreshing ? "Syncing" : "Sync"}
          </button>
        </div>
      }
    >
      {isCreating && (
        <div className="mb-6 animate-in slide-in-from-top-4 fade-in duration-200">
          <PageSection title={t("supplier_portal.settings.tax_regimes.text.create_new_tax_regime")}>
            <form onSubmit={handleCreate} className="space-y-4 p-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-[var(--desk-text-secondary)] mb-1">
                    Country Code
                  </label>
                  <input 
                    type="text" 
                    required
                    value={formData.country_code}
                    onChange={(e) => setFormData({...formData, country_code: e.target.value})}
                    className="w-full bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-lg px-4 py-2.5 text-[var(--desk-text-primary)]"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-[var(--desk-text-secondary)] mb-1">
                    Currency
                  </label>
                  <input 
                    type="text" 
                    required
                    value={formData.currency}
                    onChange={(e) => setFormData({...formData, currency: e.target.value})}
                    className="w-full bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-lg px-4 py-2.5 text-[var(--desk-text-primary)]"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-[var(--desk-text-secondary)] mb-1">
                    Effective From
                  </label>
                  <input 
                    type="datetime-local" 
                    required
                    value={formData.effective_from}
                    onChange={(e) => setFormData({...formData, effective_from: e.target.value})}
                    className="w-full bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-lg px-4 py-2.5 text-[var(--desk-text-primary)]"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-[var(--desk-text-secondary)] mb-1">
                    VAT Rate (Basis Points) - e.g. 1200 for 12%
                  </label>
                  <input 
                    type="number" 
                    required
                    min="0"
                    step="1"
                    value={formData.vat_rate_bps}
                    onChange={(e) => setFormData({...formData, vat_rate_bps: e.target.value})}
                    className="w-full bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-lg px-4 py-2.5 text-[var(--desk-text-primary)]"
                  />
                </div>
                <div className="md:col-span-2 flex items-center">
                  <input 
                    type="checkbox" 
                    id="simplified"
                    checked={formData.simplified}
                    onChange={(e) => setFormData({...formData, simplified: e.target.checked})}
                    className="mr-2"
                  />
                  <label htmlFor="simplified" className="text-sm font-medium text-[var(--desk-text-primary)]">
                    Simplified Regime
                  </label>
                </div>
              </div>
              <div className="flex justify-end gap-2 mt-4 pt-4 border-t border-[var(--desk-border)]">
                <button
                  type="button"
                  onClick={() => setIsCreating(false)}
                  className="portal-btn portal-btn--ghost px-4 rounded-lg"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={isSubmitting}
                  className="portal-btn portal-btn--primary px-6 rounded-lg"
                >
                  {isSubmitting ? "Creating..." : "Save Regime"}
                </button>
              </div>
            </form>
          </PageSection>
        </div>
      )}

      <PageSection title={t("supplier_portal.settings.tax_regimes.text.tax_regime_history")} className="p-0">
        {loading && !data ? (
          <div className="p-8 text-center text-[var(--desk-text-secondary)]">{t("supplier_portal.settings.tax_regimes.text.loading_regimes")}</div>
        ) : !data || data.regimes.length === 0 ? (
          <div className="p-8 text-center text-[var(--desk-text-secondary)]">{t("supplier_portal.settings.tax_regimes.text.no_tax_regimes_found")}</div>
        ) : (
          <div className="divide-y divide-[var(--desk-border)]">
            {data.regimes.map((regime, i) => (
              <div key={regime.id} className="p-4 hover:bg-[var(--desk-surface-hover)] transition-colors flex flex-col md:flex-row md:items-center justify-between gap-4">
                <div>
                  <div className="flex items-center gap-2 mb-1">
                    <span className="font-semibold text-[var(--desk-text-primary)] text-lg">
                      {(regime.vat_rate_bps / 100).toFixed(2)}% VAT
                    </span>
                    {i === 0 && !regime.effective_to ? (
                      <span className="px-2 py-0.5 rounded-full bg-green-500/10 text-green-500 text-xs font-medium border border-green-500/20">
                        Active
                      </span>
                    ) : (
                      <span className="px-2 py-0.5 rounded-full bg-[var(--desk-border)] text-[var(--desk-text-secondary)] text-xs font-medium">
                        Archived
                      </span>
                    )}
                    {regime.simplified && (
                      <span className="px-2 py-0.5 rounded-full bg-blue-500/10 text-blue-500 text-xs font-medium border border-blue-500/20">
                        Simplified
                      </span>
                    )}
                  </div>
                  <div className="text-sm text-[var(--desk-text-secondary)] font-mono">
                    ID: {regime.id.substring(0, 8)}...
                  </div>
                </div>

                <div className="flex flex-wrap items-center gap-6">
                  <div className="flex flex-col">
                    <span className="text-xs text-[var(--desk-text-tertiary)] uppercase tracking-wider font-semibold mb-1">
                      Effective From
                    </span>
                    <span className="text-sm text-[var(--desk-text-primary)] flex items-center gap-1.5">
                      <Calendar size={14} className="text-[var(--desk-text-secondary)]" />
                      {formatDate(regime.effective_from)}
                    </span>
                  </div>
                  <div className="flex flex-col">
                    <span className="text-xs text-[var(--desk-text-tertiary)] uppercase tracking-wider font-semibold mb-1">
                      Effective To
                    </span>
                    <span className="text-sm text-[var(--desk-text-primary)] flex items-center gap-1.5">
                      <Calendar size={14} className="text-[var(--desk-text-secondary)]" />
                      {formatDate(regime.effective_to)}
                    </span>
                  </div>
                  <div className="flex flex-col">
                    <span className="text-xs text-[var(--desk-text-tertiary)] uppercase tracking-wider font-semibold mb-1">
                      Region
                    </span>
                    <span className="text-sm text-[var(--desk-text-primary)] font-mono">
                      {regime.country_code} / {regime.currency}
                    </span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </PageSection>
    </PageChrome>
  );
}
