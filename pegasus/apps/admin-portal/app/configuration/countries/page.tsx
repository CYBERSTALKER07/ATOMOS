'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { Button } from '@heroui/react';
import { useToken } from '@/lib/auth';
import { useToast } from '@/components/Toast';
import { useLocale } from '@/hooks/useLocale';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

interface CountryConfig {
  country_code: string;
  country_name: string;
  timezone: string;
  currency_code: string;
  breach_radius_meters: number;
  payment_gateways: string[];
  sms_provider: string;
}

interface ApiResponse {
  status: string;
  data: CountryConfig[];
}

export default function CountryConfigsPage() {
  const token = useToken();
  const { toast } = useToast();
  const { t } = useLocale();

  const [rows, setRows] = useState<CountryConfig[]>([]);
  const [selectedCode, setSelectedCode] = useState<string>('');
  const [draft, setDraft] = useState<CountryConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const selected = useMemo(() => rows.find((r) => r.country_code === selectedCode) || null, [rows, selectedCode]);

  const load = useCallback(async () => {
    if (!token) return;
    setLoading(true);
    try {
      const res = await fetch(`${API}/v1/admin/country-configs`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error(t('supplier_portal.configuration.countries.error.load_failed'));
      const payload = (await res.json()) as ApiResponse;
      const data = payload.data || [];
      setRows(data);
      if (data.length > 0) {
        const pick = data[0].country_code;
        setSelectedCode((prev) => prev || pick);
      }
    } catch (e) {
      toast(e instanceof Error ? e.message : t('supplier_portal.configuration.countries.error.load_failed'), 'error');
    } finally {
      setLoading(false);
    }
  }, [t, token, toast]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    if (selected) {
      setDraft({ ...selected });
    }
  }, [selected]);

  const save = useCallback(async () => {
    if (!token || !draft) return;
    setSaving(true);
    try {
      const res = await fetch(`${API}/v1/admin/country-configs`, {
        method: 'PUT',
        headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(draft),
      });
      if (!res.ok) throw new Error(await res.text());
      toast(t('supplier_portal.configuration.countries.toast.saved', { country_code: draft.country_code }), 'success');
      await load();
    } catch (e) {
      toast(e instanceof Error ? e.message : t('supplier_portal.configuration.countries.error.save_failed'), 'error');
    } finally {
      setSaving(false);
    }
  }, [draft, load, t, token, toast]);

  return (
    <div className="flex flex-col gap-6 w-full max-w-7xl mx-auto px-4 py-6">
      <div>
        <h1 className="md-typescale-headline-small" style={{ color: 'var(--color-md-on-surface)' }}>
          {t('supplier_portal.configuration.countries.title')}
        </h1>
        <p className="md-typescale-body-small mt-1" style={{ color: 'var(--color-md-on-surface-variant)' }}>
          {t('supplier_portal.configuration.countries.subtitle')}
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <div className="md-card md-elevation-1 md-shape-md overflow-hidden" style={{ background: 'var(--color-md-surface)' }}>
          <div className="px-4 py-3 border-b" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
            <p className="md-typescale-title-small">{t('supplier_portal.configuration.countries.list_title')}</p>
          </div>
          {loading ? (
            <div className="p-4">{t('supplier_portal.configuration.countries.state.loading')}</div>
          ) : (
            <div className="max-h-[520px] overflow-y-auto">
              {rows.map((r) => (
                <button
                  key={r.country_code}
                  onClick={() => setSelectedCode(r.country_code)}
                  className="w-full text-left px-4 py-3 border-b hover:bg-black/5"
                  style={{
                    borderColor: 'var(--color-md-outline-variant)',
                    background: selectedCode === r.country_code ? 'var(--color-md-surface-container-high)' : 'transparent',
                  }}
                >
                  <div className="font-medium">{r.country_name}</div>
                  <div className="text-xs" style={{ color: 'var(--color-md-on-surface-variant)' }}>{r.country_code} • {r.currency_code}</div>
                </button>
              ))}
            </div>
          )}
        </div>

        <div className="lg:col-span-2 md-card md-elevation-1 md-shape-md p-4" style={{ background: 'var(--color-md-surface)' }}>
          {!draft ? (
            <div className="text-sm" style={{ color: 'var(--color-md-on-surface-variant)' }}>
              {t('supplier_portal.configuration.countries.state.select_country')}
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              <label className="flex flex-col gap-1">
                <span className="text-xs">{t('supplier_portal.configuration.countries.field.country_code')}</span>
                <input className="md-input-outlined px-3 py-2" value={draft.country_code} onChange={(e) => setDraft({ ...draft, country_code: e.target.value.toUpperCase() })} />
              </label>
              <label className="flex flex-col gap-1">
                <span className="text-xs">{t('supplier_portal.configuration.countries.field.country_name')}</span>
                <input className="md-input-outlined px-3 py-2" value={draft.country_name} onChange={(e) => setDraft({ ...draft, country_name: e.target.value })} />
              </label>
              <label className="flex flex-col gap-1">
                <span className="text-xs">{t('supplier_portal.configuration.countries.field.timezone')}</span>
                <input className="md-input-outlined px-3 py-2" value={draft.timezone} onChange={(e) => setDraft({ ...draft, timezone: e.target.value })} />
              </label>
              <label className="flex flex-col gap-1">
                <span className="text-xs">{t('supplier_portal.configuration.countries.field.currency_code')}</span>
                <input className="md-input-outlined px-3 py-2" value={draft.currency_code} onChange={(e) => setDraft({ ...draft, currency_code: e.target.value.toUpperCase() })} />
              </label>
              <label className="flex flex-col gap-1">
                <span className="text-xs">{t('supplier_portal.configuration.countries.field.breach_radius_meters')}</span>
                <input className="md-input-outlined px-3 py-2" type="number" min="1" value={draft.breach_radius_meters} onChange={(e) => setDraft({ ...draft, breach_radius_meters: Number(e.target.value) })} />
              </label>
              <label className="flex flex-col gap-1">
                <span className="text-xs">{t('supplier_portal.configuration.countries.field.sms_provider')}</span>
                <input className="md-input-outlined px-3 py-2" value={draft.sms_provider} onChange={(e) => setDraft({ ...draft, sms_provider: e.target.value })} />
              </label>
              <label className="flex flex-col gap-1 md:col-span-2">
                <span className="text-xs">{t('supplier_portal.configuration.countries.field.payment_gateways')}</span>
                <input
                  className="md-input-outlined px-3 py-2"
                  value={(draft.payment_gateways || []).join(',')}
                  onChange={(e) => setDraft({ ...draft, payment_gateways: e.target.value.split(',').map((s) => s.trim()).filter(Boolean) })}
                />
              </label>
              <div className="md:col-span-2 pt-2">
                <Button variant="primary" onPress={save} isDisabled={saving}>
                  {saving
                    ? t('common.status.saving')
                    : t('supplier_portal.configuration.countries.action.save')}
                </Button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
