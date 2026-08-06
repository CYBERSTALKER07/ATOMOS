"use client";

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { createSupplierApi } from "@/lib/api";
import { supplierScopeId } from "@/lib/supplier-scope";
import {
  supplierPromotionCreateKey,
  supplierPromotionDeactivateKey,
  supplierPromotionUpdateKey,
} from "@pegasusx/api-client";
import type {
  PromoSimulateResult,
  SupplierPromotion,
  SupplierPromotionScopeType,
  SupplierPromotionUpsertRequest,
} from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";

const api = createSupplierApi();

const SCOPE_OPTIONS: SupplierPromotionScopeType[] = [
  "ALL_PRODUCTS",
  "PRODUCT",
  "CATEGORY",
];

const emptyForm = (): SupplierPromotionUpsertRequest => ({
  name: "",
  description: "",
  discount_bps: 500,
  scope_type: "ALL_PRODUCTS",
  retailer_scope: "ALL",
  retailer_ids: [],
  min_line_quantity: 0,
  min_order_amount_minor: 0,
  priority: 0,
});

export default function PromotionsPage() {
  const t = usePortalT();
  const [promotions, setPromotions] = useState<SupplierPromotion[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState<SupplierPromotionUpsertRequest>(emptyForm);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [retailerIdsText, setRetailerIdsText] = useState("");
  const [simResults, setSimResults] = useState<Record<string, PromoSimulateResult>>({});
  const [simulatingId, setSimulatingId] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.listSupplierPromotions();
      setPromotions(res.promotions ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.failed_to_load_promotions"));
    } finally {
      setLoading(false);
    }
  }, []);

  useSupplierSessionReconcile(load);

  useEffect(() => {
    void load();
  }, [load]);

  const activeCount = useMemo(
    () => promotions.filter((p) => p.is_active).length,
    [promotions],
  );

  function resetForm() {
    setForm(emptyForm());
    setEditingId(null);
    setRetailerIdsText("");
  }

  async function simulatePromo(promo: SupplierPromotion) {
    setSimulatingId(promo.promotion_id);
    setError(null);
    try {
      const result = await api.simulatePromotionPandL(
        {
          promotion_id: promo.promotion_id,
          discount_pct: promo.discount_bps / 100,
          expected_units: 500,
          avg_unit_margin_minor: 1000,
        },
        crypto.randomUUID(),
      );
      setSimResults((prev) => ({ ...prev, [promo.promotion_id]: result }));
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.p_and_l_simulation_failed"));
    } finally {
      setSimulatingId(null);
    }
  }

  function startEdit(promo: SupplierPromotion) {
    setEditingId(promo.promotion_id);
    setForm({
      name: promo.name,
      description: promo.description ?? "",
      discount_bps: promo.discount_bps,
      scope_type: promo.scope_type,
      scope_product_id: promo.scope_product_id,
      scope_category_id: promo.scope_category_id,
      retailer_scope: promo.retailer_scope,
      retailer_ids: promo.retailer_ids ?? [],
      min_line_quantity: promo.min_line_quantity ?? 0,
      min_order_amount_minor: promo.min_order_amount_minor ?? 0,
      starts_at: promo.starts_at ?? undefined,
      ends_at: promo.ends_at ?? undefined,
      priority: promo.priority ?? 0,
    });
    setRetailerIdsText((promo.retailer_ids ?? []).join("\n"));
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setError(null);
    const payload: SupplierPromotionUpsertRequest = {
      ...form,
      retailer_ids:
        form.retailer_scope === "ALLOWLIST"
          ? retailerIdsText
              .split(/[\n,]+/)
              .map((id) => id.trim())
              .filter(Boolean)
          : [],
      min_line_quantity: form.min_line_quantity || undefined,
      min_order_amount_minor: form.min_order_amount_minor || undefined,
    };
    try {
      const scope = supplierScopeId();
      const fingerprint = JSON.stringify(payload);
      if (editingId) {
        await api.updateSupplierPromotion(
          editingId,
          payload,
          supplierPromotionUpdateKey(scope, editingId, fingerprint),
        );
      } else {
        await api.createSupplierPromotion(
          payload,
          supplierPromotionCreateKey(scope, fingerprint),
        );
      }
      resetForm();
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.configuration.countries.error.save_failed"));
    } finally {
      setSaving(false);
    }
  }

  async function deactivate(promotionId: string) {
    setError(null);
    try {
      await api.deactivateSupplierPromotion(
        promotionId,
        supplierPromotionDeactivateKey(supplierScopeId(), promotionId),
      );
      if (editingId === promotionId) {
        resetForm();
      }
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.deactivate_failed"));
    }
  }

  return (
    <PageChrome
      icon="pricing"
      title={t("portal.nav.promotions")}
      description={t("supplier_portal.residual.text.product_category_and_volume_based_sales_with_retailer_targeting_")}
      loading={loading}
      error={error}
      empty={!loading && promotions.length === 0}
    >
      <div className="grid grid-cols-1 xl:grid-cols-2 gap-6">
        <section className="md-card p-6 space-y-4">
          <div className="flex items-center justify-between gap-4">
            <h2 className="md-typescale-title-medium font-semibold">
              {editingId ? "Edit promotion" : "New promotion"}
            </h2>
            {editingId ? (
              <button type="button" className="md-btn md-btn-text" onClick={resetForm}>
                Cancel edit
              </button>
            ) : null}
          </div>
          <form className="space-y-3" onSubmit={handleSubmit}>
            <Field label={t("supplier_portal.analytics.knowledge_graph.text.name")}>
              <input
                className="md-input-outlined w-full px-3 py-2"
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                required
              />
            </Field>
            <Field label={t("supplier_portal.analytics.demand.signals.text.description")}>
              <input
                className="md-input-outlined w-full px-3 py-2"
                value={form.description ?? ""}
                onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
              />
            </Field>
            <Field label={t("supplier_portal.residual.text.discount_bps_500_5")}>
              <input
                type="number"
                min={1}
                max={10000}
                className="md-input-outlined w-full px-3 py-2"
                value={form.discount_bps}
                onChange={(e) =>
                  setForm((f) => ({ ...f, discount_bps: Number(e.target.value) }))
                }
                required
              />
            </Field>
            <Field label={t("supplier_portal.analytics.demand.signals.text.scope")}>
              <select
                className="md-input-outlined w-full px-3 py-2"
                value={form.scope_type}
                onChange={(e) =>
                  setForm((f) => ({
                    ...f,
                    scope_type: e.target.value as SupplierPromotionScopeType,
                  }))
                }
              >
                {SCOPE_OPTIONS.map((scope) => (
                  <option key={scope} value={scope}>
                    {scope}
                  </option>
                ))}
              </select>
            </Field>
            {form.scope_type === "PRODUCT" ? (
              <Field label={t("supplier_portal.analytics.demand.signals.text.product_id")}>
                <input
                  className="md-input-outlined w-full px-3 py-2"
                  value={form.scope_product_id ?? ""}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, scope_product_id: e.target.value }))
                  }
                  required
                />
              </Field>
            ) : null}
            {form.scope_type === "CATEGORY" ? (
              <Field label={t("supplier_portal.residual.text.category_id")}>
                <input
                  className="md-input-outlined w-full px-3 py-2"
                  value={form.scope_category_id ?? ""}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, scope_category_id: e.target.value }))
                  }
                  required
                />
              </Field>
            ) : null}
            <Field label={t("supplier_portal.residual.text.retailer_scope")}>
              <select
                className="md-input-outlined w-full px-3 py-2"
                value={form.retailer_scope ?? "ALL"}
                onChange={(e) =>
                  setForm((f) => ({
                    ...f,
                    retailer_scope: e.target.value as "ALL" | "ALLOWLIST",
                  }))
                }
              >
                <option value="ALL">{t("supplier_portal.promotions.text.all_retailers")}</option>
                <option value="ALLOWLIST">{t("supplier_portal.promotions.text.specific_retailers")}</option>
              </select>
            </Field>
            {form.retailer_scope === "ALLOWLIST" ? (
              <Field label={t("supplier_portal.residual.text.retailer_ids_one_per_line")}>
                <textarea
                  className="md-input-outlined w-full px-3 py-2 min-h-[88px]"
                  value={retailerIdsText}
                  onChange={(e) => setRetailerIdsText(e.target.value)}
                />
              </Field>
            ) : null}
            <div className="grid grid-cols-2 gap-3">
              <Field label={t("supplier_portal.residual.text.min_line_qty_packs")}>
                <input
                  type="number"
                  min={0}
                  className="md-input-outlined w-full px-3 py-2"
                  value={form.min_line_quantity ?? 0}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, min_line_quantity: Number(e.target.value) }))
                  }
                />
              </Field>
              <Field label={t("supplier_portal.residual.text.min_order_minor_units")}>
                <input
                  type="number"
                  min={0}
                  className="md-input-outlined w-full px-3 py-2"
                  value={form.min_order_amount_minor ?? 0}
                  onChange={(e) =>
                    setForm((f) => ({
                      ...f,
                      min_order_amount_minor: Number(e.target.value),
                    }))
                  }
                />
              </Field>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <Field label={t("supplier_portal.residual.text.starts_at_rfc3339")}>
                <input
                  className="md-input-outlined w-full px-3 py-2"
                  placeholder="2026-06-11T00:00:00Z"
                  value={form.starts_at ?? ""}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, starts_at: e.target.value || undefined }))
                  }
                />
              </Field>
              <Field label={t("supplier_portal.residual.text.ends_at_rfc3339")}>
                <input
                  className="md-input-outlined w-full px-3 py-2"
                  placeholder="2026-12-31T23:59:59Z"
                  value={form.ends_at ?? ""}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, ends_at: e.target.value || undefined }))
                  }
                />
              </Field>
            </div>
            <Field label={t("supplier_portal.residual.text.priority_higher_wins_ties")}>
              <input
                type="number"
                className="md-input-outlined w-full px-3 py-2"
                value={form.priority ?? 0}
                onChange={(e) =>
                  setForm((f) => ({ ...f, priority: Number(e.target.value) }))
                }
              />
            </Field>
            <button
              type="submit"
              className="md-btn md-btn-filled w-full"
              disabled={saving}
            >
              {saving ? "Saving…" : editingId ? "Update promotion" : "Create promotion"}
            </button>
          </form>
        </section>

        <section className="md-card p-6 space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="md-typescale-title-medium font-semibold">{t("supplier_portal.promotions.text.active_rules")}</h2>
            <span className="md-chip">{activeCount} active</span>
          </div>
          <div className="space-y-3 max-h-[70vh] overflow-y-auto">
            {promotions.map((promo) => (
              <article
                key={promo.promotion_id}
                className="border border-[var(--color-md-outline-variant)] p-4 space-y-2"
              >
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="md-typescale-title-small font-semibold">{promo.name}</div>
                    <div className="md-typescale-body-small text-[var(--color-md-outline)]">
                      {(promo.discount_bps / 100).toFixed(2)}% · {promo.scope_type}
                      {promo.is_active ? "" : " · inactive"}
                    </div>
                  </div>
                  <div className="flex gap-2">
                    <button
                      type="button"
                      className="md-btn md-btn-tonal"
                      disabled={simulatingId === promo.promotion_id}
                      onClick={() => void simulatePromo(promo)}
                    >
                      {simulatingId === promo.promotion_id ? "Simulating…" : "Simulate P&L"}
                    </button>
                    <button
                      type="button"
                      className="md-btn md-btn-tonal"
                      onClick={() => startEdit(promo)}
                    >
                      Edit
                    </button>
                    {promo.is_active ? (
                      <button
                        type="button"
                        className="md-btn md-btn-outlined"
                        onClick={() => void deactivate(promo.promotion_id)}
                      >
                        End sale
                      </button>
                    ) : null}
                  </div>
                </div>
                {promo.description ? (
                  <p className="md-typescale-body-small">{promo.description}</p>
                ) : null}
                <dl className="grid grid-cols-2 gap-x-4 gap-y-1 md-typescale-label-small text-[var(--color-md-outline)]">
                  {promo.scope_product_id ? (
                    <>
                      <dt>{t("supplier_portal.admin.empathy.hierarchy.product.level")}</dt>
                      <dd className="text-[var(--color-md-on-surface)]">{promo.scope_product_id}</dd>
                    </>
                  ) : null}
                  {promo.scope_category_id ? (
                    <>
                      <dt>{t("supplier_portal.catalog.components.catalog_table.text.category")}</dt>
                      <dd className="text-[var(--color-md-on-surface)]">{promo.scope_category_id}</dd>
                    </>
                  ) : null}
                  {promo.min_line_quantity ? (
                    <>
                      <dt>{t("supplier_portal.promotions.text.min_qty")}</dt>
                      <dd className="text-[var(--color-md-on-surface)]">{promo.min_line_quantity}</dd>
                    </>
                  ) : null}
                  {promo.min_order_amount_minor ? (
                    <>
                      <dt>{t("supplier_portal.promotions.text.min_order")}</dt>
                      <dd className="text-[var(--color-md-on-surface)]">
                        {promo.min_order_amount_minor}
                      </dd>
                    </>
                  ) : null}
                  {promo.starts_at ? (
                    <>
                      <dt>{t("supplier_portal.promotions.text.starts")}</dt>
                      <dd className="text-[var(--color-md-on-surface)]">{promo.starts_at}</dd>
                    </>
                  ) : null}
                  {promo.ends_at ? (
                    <>
                      <dt>{t("supplier_portal.promotions.text.ends")}</dt>
                      <dd className="text-[var(--color-md-on-surface)]">{promo.ends_at}</dd>
                    </>
                  ) : null}
                </dl>
                {simResults[promo.promotion_id] ? (
                  <div className="mt-3 rounded-lg border border-[var(--border)] p-3 md-typescale-body-small">
                    <div className="font-semibold mb-1">{t("supplier_portal.promotions.text.sandbox_p_and_l_projection")}</div>
                    <dl className="grid grid-cols-2 gap-x-4 gap-y-1 text-[var(--color-md-outline)]">
                      <dt>{t("supplier_portal.promotions.text.volume")}</dt>
                      <dd className="text-[var(--color-md-on-surface)]">
                        {simResults[promo.promotion_id].projected_volume}
                      </dd>
                      <dt>{t("supplier_portal.promotions.text.margin")}</dt>
                      <dd className="text-[var(--color-md-on-surface)]">
                        {simResults[promo.promotion_id].projected_margin_minor}
                      </dd>
                      <dt>{t("supplier_portal.promotions.text.margin_2")}</dt>
                      <dd className="text-[var(--color-md-on-surface)]">
                        {simResults[promo.promotion_id].margin_delta_pct.toFixed(1)}%
                      </dd>
                    </dl>
                  </div>
                ) : null}
              </article>
            ))}
          </div>
        </section>
      </div>
    </PageChrome>
  );
}

function Field({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <label className="block space-y-1">
      <span className="md-typescale-label-medium text-[var(--color-md-outline)]">{label}</span>
      {children}
    </label>
  );
}
