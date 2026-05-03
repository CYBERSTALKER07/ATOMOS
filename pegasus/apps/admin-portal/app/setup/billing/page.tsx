'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@heroui/react';
import { translateProblemDetail } from '@pegasus/i18n';
import { useLocale } from '@/hooks/useLocale';
import { apiFetch } from '@/lib/auth';

const gwIcons: Record<string, string> = {
  GLOBAL_PAY: 'M20 4H4c-1.11 0-1.99.89-1.99 2L2 18c0 1.11.89 2 2 2h16c1.11 0 2-.89 2-2V6c0-1.11-.89-2-2-2zm0 14H4v-6h16v6zm0-10H4V6h16v2z',
  CASH: 'M13 1.07V9h7c0-4.08-3.05-7.44-7-7.93zM4 15c0 4.42 3.58 8 8 8s8-3.58 8-8v-4H4v4zm7-13.93C7.05 1.56 4 4.92 4 9h7V1.07z',
  GLOBAL_PAY_DUP: 'M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.76l6.15 6.15c-.15 1.13-.27 2.37-.36 3.54zM2 12c0-5.52 4.48-10 10-10s10 4.48 10 10-4.48 10-10 10S2 17.52 2 12z',
  CASH_DUP: 'M11.8 10.9c-2.27-.59-3-1.2-3-2.15 0-1.09 1.01-1.85 2.7-1.85 1.78 0 2.44.85 2.5 2.1h2.21c-.07-1.72-1.12-3.3-3.21-3.81V3h-3v2.16c-1.94.42-3.5 1.68-3.5 3.61 0 2.31 1.91 3.46 4.7 4.13 2.5.6 3 1.48 3 2.41 0 .69-.49 1.79-2.7 1.79-2.06 0-2.87-.92-2.98-2.1h-2.2c.12 2.19 1.76 3.42 3.68 3.83V21h3v-2.15c1.95-.37 3.5-1.5 3.5-3.55 0-2.84-2.43-3.81-4.7-4.4z',
  CARD: 'M21 18v1c0 1.1-.9 2-2 2H5c-1.11 0-2-.9-2-2V5c0-1.1.89-2 2-2h14c1.1 0 2 .9 2 2v1h-9c-1.11 0-2 .9-2 2v8c0 1.1.89 2 2 2h9zm-9-2h10V8H12v8zm4-2.5c-.83 0-1.5-.67-1.5-1.5s.67-1.5 1.5-1.5 1.5.67 1.5 1.5-.67 1.5-1.5 1.5z',
  BANK: 'M4 10v7h3v-7H4zm6 0v7h3v-7h-3zM2 22h19v-3H2v3zm14-12v7h3v-7h-3zm-4.5-9L2 6v2h19V6l-9.5-5z',
};

const GLOBAL_PAYNT_GATEWAYS = [
  {
    id: 'GLOBAL_PAY',
    labelKey: 'supplier_portal.billing_setup.gateway.global_pay.label',
    descKey: 'supplier_portal.billing_setup.gateway.global_pay.description',
  },
  {
    id: 'CASH',
    labelKey: 'supplier_portal.billing_setup.gateway.cash.label',
    descKey: 'supplier_portal.billing_setup.gateway.cash.description',
  },
  {
    id: 'CARD',
    labelKey: 'supplier_portal.billing_setup.gateway.card.label',
    descKey: 'supplier_portal.billing_setup.gateway.card.description',
  },
  {
    id: 'BANK',
    labelKey: 'supplier_portal.billing_setup.gateway.bank.label',
    descKey: 'supplier_portal.billing_setup.gateway.bank.description',
  },
];

function trailingDigits(value: string): string {
  const digitsOnly = value.replace(/\D/g, '');
  return digitsOnly.slice(-4);
}

function buildBillingSetupIdempotencyKey(
  bankName: string,
  accountNumber: string,
  cardNumber: string,
  paymentGateway: string,
): string {
  return [
    'supplier-billing-setup',
    bankName.trim().toUpperCase(),
    trailingDigits(accountNumber),
    trailingDigits(cardNumber),
    paymentGateway.trim().toUpperCase(),
  ].join(':');
}

function InputField({ label, ...props }: React.InputHTMLAttributes<HTMLInputElement> & { label: string }) {
  return (
    <div>
      <label className="md-typescale-label-medium block mb-1.5" style={{ color: 'var(--foreground)' }}>{label}</label>
      <input {...props} className="md-input-outlined w-full" />
    </div>
  );
}

export default function BillingSetupPage() {
  const router = useRouter();
  const { locale, t } = useLocale();
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [bankName, setBankName] = useState('');
  const [accountNumber, setAccountNumber] = useState('');
  const [cardNumber, setCardNumber] = useState('');
  const [global_payntGateway, setGlobalPayntGateway] = useState('');

  const canSubmit = !!global_payntGateway;

  const submit = async () => {
    setError('');
    setSubmitting(true);
    try {
      const res = await apiFetch('/v1/supplier/billing/setup', {
        method: 'POST',
        headers: {
          'Idempotency-Key': buildBillingSetupIdempotencyKey(
            bankName,
            accountNumber,
            cardNumber,
            global_payntGateway,
          ),
        },
        body: JSON.stringify({
          bank_name: bankName,
          account_number: accountNumber,
          card_number: cardNumber,
          payment_gateway: global_payntGateway,
        }),
      });
      const body = await res.json().catch(() => ({ title: t('supplier_portal.billing_setup.error.setup_failed') }));

      if (body?.queued) {
        setError('Billing setup queued — reconnect to finish onboarding.');
        setSubmitting(false);
        return;
      }

      if (!res.ok) {
        const errorMessage = body?.error === 'rate_limit_exceeded'
          ? t('error.rate_limited')
          : body?.message_key || body?.detail || body?.title
            ? translateProblemDetail(body, locale)
            : body?.error || t('supplier_portal.billing_setup.error.http_error', { status: res.status });
        setError(errorMessage);
        setSubmitting(false);
        return;
      }

      router.push('/supplier/dashboard');
    } catch {
      setError(t('supplier_portal.billing_setup.error.network'));
      setSubmitting(false);
    }
  };

  const skip = () => {
    router.push('/supplier/dashboard');
  };

  return (
    <div className="min-h-screen flex items-center justify-center p-4" style={{ background: 'var(--background)' }}>
      <div className="w-full max-w-lg">
        {/* Header */}
        <div className="flex items-center gap-3 mb-8">
          <div
            className="w-10 h-10 rounded-xl flex items-center justify-center"
            style={{ background: 'var(--accent)', color: 'var(--accent-foreground)' }}
          >
            <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
              <path d="M20 4H4v2h16V4zm1 10v-2l-1-5H4l-1 5v2h1v6h10v-6h4v6h2v-6h1zm-9 4H6v-4h6v4z"/>
            </svg>
          </div>
          <div>
            <h1 className="md-typescale-title-large" style={{ color: 'var(--foreground)' }}>
              {t('supplier_portal.billing_setup.title')}
            </h1>
            <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
              {t('supplier_portal.billing_setup.subtitle')}
            </p>
          </div>
        </div>

        {/* Error */}
        {error && (
          <div className="mb-5 px-4 py-3 md-shape-lg md-typescale-body-small flex items-start gap-3" style={{ background: 'var(--danger)', color: 'var(--danger-foreground)' }} role="alert">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor" className="shrink-0 mt-0.5"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/></svg>
            {error}
          </div>
        )}

        {/* Bank Details */}
        <div className="space-y-4 mb-8">
          <h2 className="md-typescale-headline-small" style={{ color: 'var(--foreground)' }}>
            {t('supplier_portal.billing_setup.bank_details.title')}
          </h2>
          <p className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>
            {t('supplier_portal.billing_setup.bank_details.description')}
          </p>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <InputField label={t('supplier_portal.billing_setup.bank_details.bank_name')} type="text" value={bankName} onChange={e => setBankName(e.target.value)} placeholder="Kapitalbank" />
            <InputField label={t('supplier_portal.billing_setup.bank_details.account_number')} type="text" value={accountNumber} onChange={e => setAccountNumber(e.target.value)} placeholder="20208000900100001010" />
          </div>
          <InputField label={t('supplier_portal.billing_setup.bank_details.card_number')} type="text" value={cardNumber} onChange={e => setCardNumber(e.target.value)} placeholder="8600 1234 5678 9012" />
        </div>

        {/* GlobalPaynt Gateway */}
        <div className="space-y-4 mb-8">
          <h2 className="md-typescale-headline-small" style={{ color: 'var(--foreground)' }}>
            {t('supplier_portal.billing_setup.gateway.title')}
          </h2>
          <p className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>
            {t('supplier_portal.billing_setup.gateway.description')}
          </p>
          <div className="space-y-3">
            {GLOBAL_PAYNT_GATEWAYS.map(gw => {
              const active = global_payntGateway === gw.id;
              return (
                <button
                  key={gw.id}
                  type="button"
                  onClick={() => setGlobalPayntGateway(gw.id)}
                  className="w-full flex items-center gap-4 p-4 md-shape-lg text-left transition-colors"
                  style={{
                    background: active ? 'var(--accent-soft)' : 'var(--surface)',
                    border: `2px solid ${active ? 'var(--accent)' : 'transparent'}`,
                  }}
                >
                  <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor" className="shrink-0"
                    style={{ color: active ? 'var(--accent-soft-foreground)' : 'var(--muted)' }}
                  >
                    <path d={gwIcons[gw.id]} />
                  </svg>
                  <div className="flex-1">
                    <p className="md-typescale-body-medium font-medium" style={{ color: active ? 'var(--accent-soft-foreground)' : 'var(--foreground)' }}>
                      {t(gw.labelKey)}
                    </p>
                    <p className="md-typescale-label-small mt-0.5" style={{ color: active ? 'var(--accent-soft-foreground)' : 'var(--muted)', opacity: 0.8 }}>
                      {t(gw.descKey)}
                    </p>
                  </div>
                  {active && (
                    <span className="flex items-center gap-1 md-typescale-label-small font-medium px-2.5 py-1 rounded-full" style={{ background: 'var(--accent)', color: 'var(--accent-foreground)' }}>
                      <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor"><path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41L9 16.17z"/></svg>
                      {t('common.action.selected')}
                    </span>
                  )}
                </button>
              );
            })}
          </div>
        </div>

        {/* Actions */}
        <div className="flex gap-3">
          <Button variant="outline" className="px-6" onPress={skip}>
            {t('common.action.skip_for_now')}
          </Button>
          <Button
            variant="primary"
            className="flex-1 py-3"
            onPress={submit}
            isDisabled={!canSubmit || submitting}
          >
            {submitting ? (
              <span className="flex items-center justify-center gap-2">
                <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                </svg>
                {t('common.status.saving')}
              </span>
            ) : t('supplier_portal.billing_setup.submit')}
          </Button>
        </div>

        <p className="text-center mt-6 md-typescale-label-small" style={{ color: 'var(--muted)', opacity: 0.6 }}>
          Pegasus &copy; 2026
        </p>
      </div>
    </div>
  );
}
