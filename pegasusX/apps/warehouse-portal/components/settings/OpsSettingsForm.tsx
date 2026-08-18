"use client";

import { usePortalT } from "@/lib/i18n";
import { ChangeEvent } from 'react';
import { PortalField, PortalInput, PortalSection } from '@/components/portal';

export interface OpsSettingsFormProps {
  preorderMinLeadDays: string;
  setPreorderMinLeadDays: (val: string) => void;
  preorderMaxLeadDays: string;
  setPreorderMaxLeadDays: (val: string) => void;
  orderLineMin: string;
  setOrderLineMin: (val: string) => void;
  orderLineMax: string;
  setOrderLineMax: (val: string) => void;
  feeBaseMinor: string;
  setFeeBaseMinor: (val: string) => void;
  feeCurrency: string;
  feeTierKm: string;
  setFeeTierKm: (val: string) => void;
  feeTierMinor: string;
  setFeeTierMinor: (val: string) => void;
  showStockCounts: boolean;
  setShowStockCounts: (val: boolean) => void;
  policy: 'REJECT' | 'ACCEPT_BACKORDER';
  setPolicy: (val: 'REJECT' | 'ACCEPT_BACKORDER') => void;
  enforceOrderAcceptance: boolean;
  setEnforceOrderAcceptance: (val: boolean) => void;
  scheduleIs24h: boolean;
  setScheduleIs24h: (val: boolean) => void;
  scheduleTimezone: string;
  setScheduleTimezone: (val: string) => void;
  weekdayOpen: string;
  setWeekdayOpen: (val: string) => void;
  weekdayClose: string;
  setWeekdayClose: (val: string) => void;
  scheduleJSON: string;
  setScheduleJSON: (val: string) => void;
}

export function OpsSettingsForm({
  preorderMinLeadDays, setPreorderMinLeadDays,
  preorderMaxLeadDays, setPreorderMaxLeadDays,
  orderLineMin, setOrderLineMin,
  orderLineMax, setOrderLineMax,
  feeBaseMinor, setFeeBaseMinor,
  feeCurrency,
  feeTierKm, setFeeTierKm,
  feeTierMinor, setFeeTierMinor,
  showStockCounts, setShowStockCounts,
  policy, setPolicy,
  enforceOrderAcceptance, setEnforceOrderAcceptance,
  scheduleIs24h, setScheduleIs24h,
  scheduleTimezone, setScheduleTimezone,
  weekdayOpen, setWeekdayOpen,
  weekdayClose, setWeekdayClose,
  scheduleJSON, setScheduleJSON,
}: OpsSettingsFormProps) {
  const t = usePortalT();
  return (
    <>
      <PortalSection icon="orders" title={t("warehouse_portal.settings.ops_settings_form.text.pre_order_lead_window")} description={t("warehouse_portal.residual.text.days_between_order_placement_and_earliest_fulfillment")}>
        <div className="grid grid-cols-2 gap-3">
          <PortalField id="preorderMinLeadDays" label={t("warehouse_portal.residual.text.min_lead_days")}>
            <PortalInput id="preorderMinLeadDays" value={preorderMinLeadDays} onChange={(e: ChangeEvent<HTMLInputElement>) => setPreorderMinLeadDays(e.target.value)} />
          </PortalField>
          <PortalField id="preorderMaxLeadDays" label={t("warehouse_portal.residual.text.max_lead_days")}>
            <PortalInput id="preorderMaxLeadDays" value={preorderMaxLeadDays} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setPreorderMaxLeadDays(e.target.value)} />
          </PortalField>
        </div>
      </PortalSection>

      <PortalSection icon="inventory" title={t("warehouse_portal.settings.ops_settings_form.text.line_quantity_limits")} description={t("warehouse_portal.residual.text.leave_blank_to_clear_a_limit_applies_to_standard_and_scheduled_c")}>
        <div className="grid grid-cols-2 gap-3">
          <PortalField id="orderLineMin" label={t("warehouse_portal.residual.text.min_per_sku")}>
            <PortalInput id="orderLineMin" value={orderLineMin} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setOrderLineMin(e.target.value)} />
          </PortalField>
          <PortalField id="orderLineMax" label={t("warehouse_portal.residual.text.max_per_sku")}>
            <PortalInput id="orderLineMax" value={orderLineMax} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setOrderLineMax(e.target.value)} />
          </PortalField>
        </div>
      </PortalSection>

      <PortalSection icon="payment" title={t("warehouse_portal.settings.ops_settings_form.text.delivery_fee_tiers")}>
        <div className="grid grid-cols-2 gap-3">
          <PortalField id="feeBaseMinor" label={t("warehouse_portal.residual.text.base_fee_minor")}>
            <PortalInput id="feeBaseMinor" value={feeBaseMinor} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setFeeBaseMinor(e.target.value)} />
          </PortalField>
          <PortalField id="feeCurrency" label={t("supplier_portal.chargebacks.text.currency")}>
            <PortalInput id="feeCurrency" value={feeCurrency} readOnly />
          </PortalField>
          <PortalField id="feeTierKm" label={t("warehouse_portal.residual.text.free_within_km")}>
            <PortalInput id="feeTierKm" value={feeTierKm} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setFeeTierKm(e.target.value)} />
          </PortalField>
          <PortalField id="feeTierMinor" label={t("warehouse_portal.residual.text.beyond_tier_fee_minor")}>
            <PortalInput id="feeTierMinor" value={feeTierMinor} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setFeeTierMinor(e.target.value)} />
          </PortalField>
        </div>
      </PortalSection>

      <PortalSection icon="settings" title={t("warehouse_portal.settings.ops_settings_form.text.out_of_stock_orders")}>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={showStockCounts} onChange={(e: ChangeEvent<HTMLInputElement>) => setShowStockCounts(e.target.checked)} />
          Show stock counts to retailers
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input type="radio" checked={policy === 'REJECT'} onChange={() => setPolicy('REJECT')} />
          Reject orders when out of stock
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input type="radio" checked={policy === 'ACCEPT_BACKORDER'} onChange={() => setPolicy('ACCEPT_BACKORDER')} />
          Accept orders — warn retailer, fulfill when stock arrives
        </label>
      </PortalSection>

      <PortalSection icon="settings" title={t("warehouse_portal.settings.ops_settings_form.text.order_acceptance_hours")} description={t("warehouse_portal.residual.text.when_enforcement_is_on_retailers_cannot_preview_or_create_orders")}>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={enforceOrderAcceptance} onChange={(e: ChangeEvent<HTMLInputElement>) => setEnforceOrderAcceptance(e.target.checked)} />
          Enforce order acceptance hours
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={scheduleIs24h} onChange={(e: ChangeEvent<HTMLInputElement>) => setScheduleIs24h(e.target.checked)} />
          Open 24 hours
        </label>
        <PortalField id="scheduleTimezone" label={t("supplier_portal.configuration.countries.field.timezone")}>
          <PortalInput id="scheduleTimezone" value={scheduleTimezone} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setScheduleTimezone(e.target.value)} />
        </PortalField>
        <div className="grid grid-cols-2 gap-3">
          <PortalField id="weekdayOpen" label={t("warehouse_portal.residual.text.weekday_open")}>
            <PortalInput id="weekdayOpen" value={weekdayOpen} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setWeekdayOpen(e.target.value)} />
          </PortalField>
          <PortalField id="weekdayClose" label={t("warehouse_portal.residual.text.weekday_close")}>
            <PortalInput id="weekdayClose" value={weekdayClose} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setWeekdayClose(e.target.value)} />
          </PortalField>
        </div>
        <h3 className="text-xs font-semibold text-[var(--muted)]">{t("warehouse_portal.settings.ops_settings_form.text.advanced_json")}</h3>
        <textarea
          className="w-full min-h-[140px] font-mono text-xs rounded-lg border p-3"
          style={{ borderColor: 'var(--field-border)', background: 'var(--field-background)' }}
          value={scheduleJSON}
          onChange={(e: ChangeEvent<HTMLTextAreaElement>) => setScheduleJSON(e.target.value)}
        />
      </PortalSection>
    </>
  );
}
