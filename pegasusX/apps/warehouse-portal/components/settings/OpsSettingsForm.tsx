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
  setFeeCurrency: (val: string) => void;
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
  feeCurrency, setFeeCurrency,
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
  return (
    <>
      <PortalSection icon="orders" title="Pre-order lead window" description="Days between order placement and earliest fulfillment.">
        <div className="grid grid-cols-2 gap-3">
          <PortalField id="preorderMinLeadDays" label="Min lead (days)">
            <PortalInput id="preorderMinLeadDays" value={preorderMinLeadDays} onChange={(e: ChangeEvent<HTMLInputElement>) => setPreorderMinLeadDays(e.target.value)} />
          </PortalField>
          <PortalField id="preorderMaxLeadDays" label="Max lead (days)">
            <PortalInput id="preorderMaxLeadDays" value={preorderMaxLeadDays} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setPreorderMaxLeadDays(e.target.value)} />
          </PortalField>
        </div>
      </PortalSection>

      <PortalSection icon="inventory" title="Line quantity limits" description="Leave blank to clear a limit. Applies to standard and scheduled checkout.">
        <div className="grid grid-cols-2 gap-3">
          <PortalField id="orderLineMin" label="Min per SKU">
            <PortalInput id="orderLineMin" value={orderLineMin} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setOrderLineMin(e.target.value)} />
          </PortalField>
          <PortalField id="orderLineMax" label="Max per SKU">
            <PortalInput id="orderLineMax" value={orderLineMax} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setOrderLineMax(e.target.value)} />
          </PortalField>
        </div>
      </PortalSection>

      <PortalSection icon="payment" title="Delivery fee tiers">
        <div className="grid grid-cols-2 gap-3">
          <PortalField id="feeBaseMinor" label="Base fee (minor)">
            <PortalInput id="feeBaseMinor" value={feeBaseMinor} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setFeeBaseMinor(e.target.value)} />
          </PortalField>
          <PortalField id="feeCurrency" label="Currency">
            <PortalInput id="feeCurrency" value={feeCurrency} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setFeeCurrency(e.target.value)} />
          </PortalField>
          <PortalField id="feeTierKm" label="Free within km">
            <PortalInput id="feeTierKm" value={feeTierKm} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setFeeTierKm(e.target.value)} />
          </PortalField>
          <PortalField id="feeTierMinor" label="Beyond-tier fee (minor)">
            <PortalInput id="feeTierMinor" value={feeTierMinor} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setFeeTierMinor(e.target.value)} />
          </PortalField>
        </div>
      </PortalSection>

      <PortalSection icon="settings" title="Out-of-stock orders">
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

      <PortalSection icon="settings" title="Order acceptance hours" description="When enforcement is on, retailers cannot preview or create orders outside the window.">
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={enforceOrderAcceptance} onChange={(e: ChangeEvent<HTMLInputElement>) => setEnforceOrderAcceptance(e.target.checked)} />
          Enforce order acceptance hours
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={scheduleIs24h} onChange={(e: ChangeEvent<HTMLInputElement>) => setScheduleIs24h(e.target.checked)} />
          Open 24 hours
        </label>
        <PortalField id="scheduleTimezone" label="Timezone">
          <PortalInput id="scheduleTimezone" value={scheduleTimezone} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setScheduleTimezone(e.target.value)} />
        </PortalField>
        <div className="grid grid-cols-2 gap-3">
          <PortalField id="weekdayOpen" label="Weekday open">
            <PortalInput id="weekdayOpen" value={weekdayOpen} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setWeekdayOpen(e.target.value)} />
          </PortalField>
          <PortalField id="weekdayClose" label="Weekday close">
            <PortalInput id="weekdayClose" value={weekdayClose} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setWeekdayClose(e.target.value)} />
          </PortalField>
        </div>
        <h3 className="text-xs font-semibold text-[var(--muted)]">Advanced JSON</h3>
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
