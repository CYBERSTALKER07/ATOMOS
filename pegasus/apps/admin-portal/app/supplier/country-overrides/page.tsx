'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { Button } from '@heroui/react';
import { useToken } from '@/lib/auth';
import { useToast } from '@/components/Toast';
import EmptyState from '@/components/EmptyState';
import {
  buildSupplierCountryOverrideDeleteIdempotencyKey,
  buildSupplierCountryOverrideSaveIdempotencyKey,
} from '../_shared/idempotency';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// ── Types ─────────────────────────────────────────────────────────────────────

interface CountryConfig {
  country_code: string;
  country_name: string;
  timezone: string;
  currency_code: string;
  breach_radius_meters: number;
  shop_closed_grace_minutes: number;
  shop_closed_escalation_minutes: number;
  offline_mode_duration_minutes: number;
  cash_custody_alert_hours: number;
  global_paynt_gateways: string[];
  notification_fallback_order: string[];
  sms_provider: string;
  maps_provider: string;
  llm_provider: string;
}

interface SupplierOverride {
  supplier_id?: string;
  country_code: string;
  breach_radius_meters: number | null;
  shop_closed_grace_minutes: number | null;
  shop_closed_escalation_minutes: number | null;
  offline_mode_duration_minutes: number | null;
  cash_custody_alert_hours: number | null;
  global_paynt_gateways: string[] | null;
  notification_fallback_order: string[] | null;
  sms_provider: string | null;
  maps_provider: string | null;
  llm_provider: string | null;
}

interface OverrideEntry {
  override: SupplierOverride;
  effective: CountryConfig;
}

// ── Helpers ───────────────────────────────────────────────────────────────────

const BLANK_OVERRIDE: Omit<SupplierOverride, 'country_code'> = {
  breach_radius_meters: null,
  shop_closed_grace_minutes: null,
  shop_closed_escalation_minutes: null,
  offline_mode_duration_minutes: null,
  cash_custody_alert_hours: null,
  global_paynt_gateways: null,
  notification_fallback_order: null,
  sms_provider: null,
  maps_provider: null,
  llm_provider: null,
};

function overrideFromEntry(entry: OverrideEntry | null, code: string): SupplierOverride {
  if (!entry) return { country_code: code, ...BLANK_OVERRIDE };
  return { ...entry.override };
}

// ── Component ─────────────────────────────────────────────────────────────────

export default function CountryOverridesPage() {
  const token = useToken();
  const { toast } = useToast();

  const [countries, setCountries] = useState<CountryConfig[]>([]);
  const [entries, setEntries] = useState<OverrideEntry[]>([]);
  const [selectedCode, setSelectedCode] = useState<string>('');
  const [draft, setDraft] = useState<SupplierOverride | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [nullFields, setNullFields] = useState<Set<keyof SupplierOverride>>(new Set());

  // ── Fetch all country configs (for picker) and any existing supplier overrides ──
  const load = useCallback(async () => {
    if (!token) return;
    setLoading(true);
    try {
      const headers = { Authorization: `Bearer ${token}` };
      const [cfgRes, overrideRes] = await Promise.all([
        fetch(`${API}/v1/admin/country-configs`, { headers }),
        fetch(`${API}/v1/supplier/country-overrides`, { headers }),
      ]);

      if (cfgRes.ok) {
        const cfgData = await cfgRes.json() as { status: string; data: CountryConfig[] };
        setCountries(cfgData.data || []);
        if ((cfgData.data || []).length > 0 && !selectedCode) {
          setSelectedCode(cfgData.data[0].country_code);
        }
      }
      if (overrideRes.ok) {
        const ovData = await overrideRes.json() as { status: string; data: OverrideEntry[] };
        setEntries(ovData.data || []);
      }
    } catch (e) {
      toast(e instanceof Error ? e.message : 'Failed to load data', 'error');
    } finally {
      setLoading(false);
    }
  }, [token, toast, selectedCode]);

  useEffect(() => { load(); }, [token]); // eslint-disable-line react-hooks/exhaustive-deps

  // ── Derive selected entry and reset draft when country changes ──
  const selectedEntry = useMemo(
    () => entries.find((e) => e.override.country_code === selectedCode) ?? null,
    [entries, selectedCode],
  );

  useEffect(() => {
    const initial = overrideFromEntry(selectedEntry, selectedCode);
    setDraft(initial);
    // Track which fields are explicitly null (use platform default)
    const nulls = new Set<keyof SupplierOverride>();
    if (initial) {
      (Object.keys(initial) as (keyof SupplierOverride)[]).forEach((k) => {
        if (k === 'country_code' || k === 'supplier_id') return;
        if (initial[k] === null) nulls.add(k);
      });
    }
    setNullFields(nulls);
  }, [selectedEntry, selectedCode]);

  const selectedCountryConfig = useMemo(
    () => countries.find((c) => c.country_code === selectedCode) ?? null,
    [countries, selectedCode],
  );

  // ── Save (PUT) ──────────────────────────────────────────────────────────────
  const save = useCallback(async () => {
    if (!token || !draft) return;
    setSaving(true);
    try {
      // Re-null any fields the user marked as "use platform default"
      const payload: SupplierOverride = { ...draft };
      (Array.from(nullFields) as (keyof SupplierOverride)[]).forEach((k) => {
        if (k !== 'country_code' && k !== 'supplier_id') {
          (payload as unknown as Record<string, unknown>)[k] = null;
        }
      });

      const res = await fetch(`${API}/v1/supplier/country-overrides`, {
        method: 'PUT',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
          'Idempotency-Key': buildSupplierCountryOverrideSaveIdempotencyKey(payload as unknown as Record<string, unknown>),
        },
        body: JSON.stringify(payload),
      });
      if (!res.ok) throw new Error(await res.text());
      toast(`Override saved for ${selectedCode}`, 'success');
      await load();
    } catch (e) {
      toast(e instanceof Error ? e.message : 'Save failed', 'error');
    } finally {
      setSaving(false);
    }
  }, [token, draft, nullFields, selectedCode, load, toast]);

  // ── Delete (revert to platform defaults) ───────────────────────────────────
  const revert = useCallback(async () => {
    if (!token || !selectedCode) return;
    setDeleting(true);
    try {
      const res = await fetch(`${API}/v1/supplier/country-overrides/${selectedCode}`, {
        method: 'DELETE',
        headers: {
          Authorization: `Bearer ${token}`,
          'Idempotency-Key': buildSupplierCountryOverrideDeleteIdempotencyKey(selectedCode),
        },
      });
      if (!res.ok) throw new Error(await res.text());
      toast(`Override for ${selectedCode} removed — platform defaults restored`, 'success');
      await load();
    } catch (e) {
      toast(e instanceof Error ? e.message : 'Delete failed', 'error');
    } finally {
      setDeleting(false);
    }
  }, [token, selectedCode, load, toast]);

  // ── Field helpers ──────────────────────────────────────────────────────────
  function setNullable<K extends keyof SupplierOverride>(key: K, value: SupplierOverride[K]) {
    setDraft((prev) => prev ? { ...prev, [key]: value } : prev);
    setNullFields((prev) => {
      const next = new Set(prev);
      next.delete(key);
      return next;
    });
  }

  function toggleNull(key: keyof SupplierOverride) {
    setNullFields((prev) => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
        setDraft((prev) => prev ? { ...(prev as SupplierOverride), [key]: null } : prev);
      }
      return next;
    });
  }

  const isNull = (k: keyof SupplierOverride) => nullFields.has(k);

  // ── Render helpers ─────────────────────────────────────────────────────────
  const labelClass = 'md-typescale-label-small block mb-1';
  const inputClass = 'md-input-outlined w-full font-mono';
  const mutedColor = { color: 'var(--muted)' };
  const platformDefault = (val: string | number) => (
    <span className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Platform: {val}</span>
  );

  // ── States ─────────────────────────────────────────────────────────────────
  if (loading) {
    return (
      <div className="min-h-full p-6 md:p-10 flex items-center justify-center" style={{ background: 'var(--background)' }}>
        <div className="md-typescale-body-medium animate-pulse" style={mutedColor}>Loading country configs…</div>
      </div>
    );
  }

  if (countries.length === 0) {
    return (
      <div className="min-h-full p-6 md:p-10" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>
        <header className="mb-10">
          <h1 className="md-typescale-headline-medium">Country Overrides</h1>
          <p className="md-typescale-body-medium mt-2" style={mutedColor}>No country configurations found in the system.</p>
        </header>
        <EmptyState
          headline="No Countries"
          body="Platform country configs must be seeded before setting supplier overrides."
        />
      </div>
    );
  }

  return (
    <div className="min-h-full p-6 md:p-10" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>
      {/* Header */}
      <header className="mb-8">
        <h1 className="md-typescale-headline-medium">Country Overrides</h1>
        <p className="md-typescale-body-medium mt-2" style={mutedColor}>
          Customize operational parameters per country. Overrides take precedence over platform defaults.
          Set a field to blank to inherit the platform value.
        </p>
      </header>

      {/* Active overrides summary row */}
      {entries.length > 0 && (
        <div className="mb-8 flex flex-wrap gap-2">
          {entries.map((e) => (
            <button
              key={e.override.country_code}
              type="button"
              onClick={() => setSelectedCode(e.override.country_code)}
              className={e.override.country_code === selectedCode ? 'md-chip md-chip-selected' : 'md-chip'}
            >
              {e.override.country_code} override active
            </button>
          ))}
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-5 gap-6">
        {/* Left: Country picker + platform reference */}
        <div className="lg:col-span-2 space-y-4">
          <div className="md-card md-card-elevated p-5 md-animate-in">
            <h2 className="md-typescale-title-small mb-4">Select Country</h2>
            <select
              className={inputClass}
              value={selectedCode}
              onChange={(e) => setSelectedCode(e.target.value)}
            >
              {countries.map((c) => (
                <option key={c.country_code} value={c.country_code}>
                  {c.country_name} ({c.country_code})
                </option>
              ))}
            </select>

            {selectedEntry && (
              <div
                className="mt-4 px-3 py-2 rounded-lg md-typescale-label-small"
                style={{ background: 'var(--color-md-secondary-container)', color: 'var(--color-md-on-secondary-container)' }}
              >
                Override active for {selectedCode}
              </div>
            )}
          </div>

          {/* Platform reference panel */}
          {selectedCountryConfig && (
            <div className="md-card p-5 space-y-2" style={{ background: 'var(--color-md-surface-container)' }}>
              <h3 className="md-typescale-label-large mb-3">Platform Defaults ({selectedCode})</h3>
              {[
                ['Geofence Radius', `${selectedCountryConfig.breach_radius_meters} m`],
                ['Shop Closed Grace', `${selectedCountryConfig.shop_closed_grace_minutes} min`],
                ['Shop Closed Escalation', `${selectedCountryConfig.shop_closed_escalation_minutes} min`],
                ['Offline Mode Duration', `${selectedCountryConfig.offline_mode_duration_minutes} min`],
                ['Cash Custody Alert', `${selectedCountryConfig.cash_custody_alert_hours} h`],
                ['GlobalPaynt Gateways', (selectedCountryConfig.global_paynt_gateways || []).join(', ')],
                ['SMS Provider', selectedCountryConfig.sms_provider],
                ['Maps Provider', selectedCountryConfig.maps_provider],
                ['LLM Provider', selectedCountryConfig.llm_provider],
              ].map(([label, val]) => (
                <div key={label as string} className="flex justify-between items-center py-1 border-b" style={{ borderColor: 'var(--border)' }}>
                  <span className="md-typescale-label-small" style={mutedColor}>{label}</span>
                  <span className="md-typescale-label-small font-mono">{val}</span>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Right: Override form */}
        {draft && (
          <div className="lg:col-span-3">
            <div className="md-card md-card-elevated p-6 md-animate-in">
              <div className="flex items-center justify-between mb-6">
                <h2 className="md-typescale-title-small">
                  Override — {selectedCode}
                </h2>
                {selectedEntry && (
                  <button
                    type="button"
                    onClick={revert}
                    disabled={deleting}
                    className="md-typescale-label-small px-3 py-1 rounded"
                    style={{ color: 'var(--color-md-error)', background: 'var(--color-md-error-container)' }}
                  >
                    {deleting ? 'Reverting…' : 'Revert to Platform Defaults'}
                  </button>
                )}
              </div>

              <div className="space-y-5">
                {/* ── Operational Timings ──────────────────────────────── */}
                <section>
                  <h3 className="md-typescale-label-large mb-3" style={mutedColor}>Operational Timings</h3>

                  <NullableNumberField
                    label="Geofence Radius (meters)"
                    fieldKey="breach_radius_meters"
                    value={draft.breach_radius_meters}
                    isNull={isNull('breach_radius_meters')}
                    platformDefault={selectedCountryConfig?.breach_radius_meters}
                    onToggleNull={() => toggleNull('breach_radius_meters')}
                    onChange={(v) => setNullable('breach_radius_meters', v)}
                    step={1}
                    min={10}
                  />

                  <NullableNumberField
                    label="Shop Closed Grace (minutes)"
                    fieldKey="shop_closed_grace_minutes"
                    value={draft.shop_closed_grace_minutes}
                    isNull={isNull('shop_closed_grace_minutes')}
                    platformDefault={selectedCountryConfig?.shop_closed_grace_minutes}
                    onToggleNull={() => toggleNull('shop_closed_grace_minutes')}
                    onChange={(v) => setNullable('shop_closed_grace_minutes', typeof v === 'number' ? Math.round(v) : v)}
                    step={1}
                    min={1}
                  />

                  <NullableNumberField
                    label="Shop Closed Escalation (minutes)"
                    fieldKey="shop_closed_escalation_minutes"
                    value={draft.shop_closed_escalation_minutes}
                    isNull={isNull('shop_closed_escalation_minutes')}
                    platformDefault={selectedCountryConfig?.shop_closed_escalation_minutes}
                    onToggleNull={() => toggleNull('shop_closed_escalation_minutes')}
                    onChange={(v) => setNullable('shop_closed_escalation_minutes', typeof v === 'number' ? Math.round(v) : v)}
                    step={1}
                    min={1}
                  />

                  <NullableNumberField
                    label="Offline Mode Duration (minutes)"
                    fieldKey="offline_mode_duration_minutes"
                    value={draft.offline_mode_duration_minutes}
                    isNull={isNull('offline_mode_duration_minutes')}
                    platformDefault={selectedCountryConfig?.offline_mode_duration_minutes}
                    onToggleNull={() => toggleNull('offline_mode_duration_minutes')}
                    onChange={(v) => setNullable('offline_mode_duration_minutes', typeof v === 'number' ? Math.round(v) : v)}
                    step={1}
                    min={5}
                  />

                  <NullableNumberField
                    label="Cash Custody Alert (hours)"
                    fieldKey="cash_custody_alert_hours"
                    value={draft.cash_custody_alert_hours}
                    isNull={isNull('cash_custody_alert_hours')}
                    platformDefault={selectedCountryConfig?.cash_custody_alert_hours}
                    onToggleNull={() => toggleNull('cash_custody_alert_hours')}
                    onChange={(v) => setNullable('cash_custody_alert_hours', typeof v === 'number' ? Math.round(v) : v)}
                    step={1}
                    min={1}
                  />
                </section>

                {/* ── GlobalPaynt Gateways ──────────────────────────────────── */}
                <section>
                  <h3 className="md-typescale-label-large mb-3" style={mutedColor}>GlobalPaynt & Notifications</h3>

                  <div className="mb-4">
                    <div className="flex items-center justify-between mb-2">
                      <label className={labelClass} style={mutedColor}>GlobalPaynt Gateways (comma-separated)</label>
                      <UseDefaultToggle
                        isNull={isNull('global_paynt_gateways')}
                        onToggle={() => toggleNull('global_paynt_gateways')}
                        platformDefault={(selectedCountryConfig?.global_paynt_gateways || []).join(', ')}
                      />
                    </div>
                    {!isNull('global_paynt_gateways') && (
                      <input
                        type="text"
                        className={inputClass}
                        value={(draft.global_paynt_gateways || []).join(', ')}
                        placeholder="e.g. GLOBAL_PAY, CASH, CASH"
                        onChange={(e) =>
                          setNullable(
                            'global_paynt_gateways',
                            e.target.value
                              .split(',')
                              .map((s) => s.trim().toUpperCase())
                              .filter(Boolean),
                          )
                        }
                      />
                    )}
                    {isNull('global_paynt_gateways') && platformDefault((selectedCountryConfig?.global_paynt_gateways || []).join(', '))}
                  </div>

                  <div className="mb-4">
                    <div className="flex items-center justify-between mb-2">
                      <label className={labelClass} style={mutedColor}>Notification Fallback Order (comma-separated)</label>
                      <UseDefaultToggle
                        isNull={isNull('notification_fallback_order')}
                        onToggle={() => toggleNull('notification_fallback_order')}
                        platformDefault={(selectedCountryConfig?.notification_fallback_order || []).join(', ')}
                      />
                    </div>
                    {!isNull('notification_fallback_order') && (
                      <input
                        type="text"
                        className={inputClass}
                        value={(draft.notification_fallback_order || []).join(', ')}
                        placeholder="e.g. FCM, TELEGRAM, SMS"
                        onChange={(e) =>
                          setNullable(
                            'notification_fallback_order',
                            e.target.value
                              .split(',')
                              .map((s) => s.trim())
                              .filter(Boolean),
                          )
                        }
                      />
                    )}
                    {isNull('notification_fallback_order') && platformDefault((selectedCountryConfig?.notification_fallback_order || []).join(', '))}
                  </div>
                </section>

                {/* ── Provider Overrides (Advanced) ─────────────────────── */}
                <details>
                  <summary
                    className="md-typescale-label-large cursor-pointer select-none py-2"
                    style={mutedColor}
                  >
                    Advanced Provider Overrides
                  </summary>
                  <div className="space-y-3 mt-3">
                    <NullableStringField
                      label="SMS Provider"
                      fieldKey="sms_provider"
                      value={draft.sms_provider}
                      isNull={isNull('sms_provider')}
                      platformDefault={selectedCountryConfig?.sms_provider}
                      onToggleNull={() => toggleNull('sms_provider')}
                      onChange={(v) => setNullable('sms_provider', v)}
                      placeholder="e.g. ESKIZ, TWILIO"
                    />
                    <NullableStringField
                      label="Maps Provider"
                      fieldKey="maps_provider"
                      value={draft.maps_provider}
                      isNull={isNull('maps_provider')}
                      platformDefault={selectedCountryConfig?.maps_provider}
                      onToggleNull={() => toggleNull('maps_provider')}
                      onChange={(v) => setNullable('maps_provider', v)}
                      placeholder="e.g. GOOGLE, MAPBOX"
                    />
                    <NullableStringField
                      label="LLM Provider"
                      fieldKey="llm_provider"
                      value={draft.llm_provider}
                      isNull={isNull('llm_provider')}
                      platformDefault={selectedCountryConfig?.llm_provider}
                      onToggleNull={() => toggleNull('llm_provider')}
                      onChange={(v) => setNullable('llm_provider', v)}
                      placeholder="e.g. GEMINI, OPENAI"
                    />
                  </div>
                </details>

                {/* ── Save button ───────────────────────────────────────── */}
                <div className="pt-4 flex justify-end">
                  <Button
                    variant="primary"
                    isDisabled={saving}
                    onPress={save}
                  >
                    {saving ? 'Saving…' : selectedEntry ? 'Update Override' : 'Create Override'}
                  </Button>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Active overrides table */}
      {entries.length > 0 && (
        <div className="mt-10">
          <h2 className="md-typescale-title-small mb-4">Active Overrides</h2>
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse md-typescale-body-small">
              <thead>
                <tr style={{ background: 'var(--color-md-surface-container)', color: 'var(--muted)' }}>
                  {['Country', 'Geofence (m)', 'Grace (min)', 'Escalation (min)', 'Offline (min)', 'Cash Alert (h)', 'Gateways'].map((h) => (
                    <th key={h} className="px-4 py-3 md-typescale-label-small font-semibold uppercase tracking-wide">
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {entries.map((e, i) => {
                  const o = e.override;
                  const eff = e.effective;
                  const row: (string | number | null)[] = [
                    o.country_code,
                    o.breach_radius_meters !== null ? o.breach_radius_meters : `${eff.breach_radius_meters} ↓`,
                    o.shop_closed_grace_minutes !== null ? o.shop_closed_grace_minutes : `${eff.shop_closed_grace_minutes} ↓`,
                    o.shop_closed_escalation_minutes !== null ? o.shop_closed_escalation_minutes : `${eff.shop_closed_escalation_minutes} ↓`,
                    o.offline_mode_duration_minutes !== null ? o.offline_mode_duration_minutes : `${eff.offline_mode_duration_minutes} ↓`,
                    o.cash_custody_alert_hours !== null ? o.cash_custody_alert_hours : `${eff.cash_custody_alert_hours} ↓`,
                    (o.global_paynt_gateways || eff.global_paynt_gateways || []).join(', '),
                  ];
                  return (
                    <tr
                      key={o.country_code}
                      className="border-b cursor-pointer"
                      style={{
                        borderColor: 'var(--border)',
                        background: i % 2 === 0 ? 'var(--color-md-surface)' : 'var(--color-md-surface-container)',
                        transition: 'background 150ms',
                      }}
                      onClick={() => setSelectedCode(o.country_code)}
                    >
                      {row.map((v, ci) => (
                        <td key={ci} className="px-4 py-3 font-mono">
                          {typeof v === 'string' && v.endsWith(' ↓') ? (
                            <span style={{ color: 'var(--muted)' }}>{v}</span>
                          ) : (
                            v ?? <span style={{ color: 'var(--muted)' }}>—</span>
                          )}
                        </td>
                      ))}
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
          <p className="mt-2 md-typescale-label-small" style={{ color: 'var(--muted)' }}>
            Values marked ↓ inherit the platform default. Cash a row to edit.
          </p>
        </div>
      )}
    </div>
  );
}

// ── Sub-components ────────────────────────────────────────────────────────────

interface UseDefaultToggleProps {
  isNull: boolean;
  onToggle: () => void;
  platformDefault?: string | number;
}
function UseDefaultToggle({ isNull, onToggle, platformDefault }: UseDefaultToggleProps) {
  return (
    <button
      type="button"
      onClick={onToggle}
      className="md-typescale-label-small px-2 py-0.5 rounded"
      style={{
        background: isNull ? 'var(--color-md-surface-container-high)' : 'var(--color-md-primary-container)',
        color: isNull ? 'var(--muted)' : 'var(--color-md-on-primary-container)',
      }}
      title={`Platform default: ${platformDefault ?? '—'}`}
    >
      {isNull ? 'Using platform default' : 'Override active'}
    </button>
  );
}

interface NullableNumberFieldProps {
  label: string;
  fieldKey?: keyof SupplierOverride;
  value: number | null;
  isNull: boolean;
  platformDefault?: number;
  onToggleNull: () => void;
  onChange: (v: number | null) => void;
  step?: number;
  min?: number;
}
function NullableNumberField({
  label,
  value,
  isNull,
  platformDefault,
  onToggleNull,
  onChange,
  step = 1,
  min = 0,
}: NullableNumberFieldProps) {
  const labelClass = 'md-typescale-label-small block mb-1';
  const inputClass = 'md-input-outlined w-full font-mono';
  return (
    <div className="mb-3">
      <div className="flex items-center justify-between mb-1">
        <label className={labelClass} style={{ color: 'var(--muted)' }}>{label}</label>
        <UseDefaultToggle isNull={isNull} onToggle={onToggleNull} platformDefault={platformDefault} />
      </div>
      {!isNull ? (
        <input
          type="number"
          className={inputClass}
          value={value ?? platformDefault ?? ''}
          step={step}
          min={min}
          onChange={(e) => {
            const n = parseFloat(e.target.value);
            onChange(isNaN(n) ? null : n);
          }}
        />
      ) : (
        <span className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
          Platform default: {platformDefault ?? '—'}
        </span>
      )}
    </div>
  );
}

interface NullableStringFieldProps {
  label: string;
  fieldKey?: keyof SupplierOverride;
  value: string | null;
  isNull: boolean;
  platformDefault?: string;
  onToggleNull: () => void;
  onChange: (v: string | null) => void;
  placeholder?: string;
}
function NullableStringField({
  label,
  value,
  isNull,
  platformDefault,
  onToggleNull,
  onChange,
  placeholder,
}: NullableStringFieldProps) {
  const labelClass = 'md-typescale-label-small block mb-1';
  const inputClass = 'md-input-outlined w-full font-mono';
  return (
    <div className="mb-3">
      <div className="flex items-center justify-between mb-1">
        <label className={labelClass} style={{ color: 'var(--muted)' }}>{label}</label>
        <UseDefaultToggle isNull={isNull} onToggle={onToggleNull} platformDefault={platformDefault} />
      </div>
      {!isNull ? (
        <input
          type="text"
          className={inputClass}
          value={value ?? ''}
          placeholder={placeholder}
          onChange={(e) => onChange(e.target.value || null)}
        />
      ) : (
        <span className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
          Platform default: {platformDefault ?? '—'}
        </span>
      )}
    </div>
  );
}
