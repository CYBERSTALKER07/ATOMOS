'use client';

import { useEffect, useState, useCallback } from 'react';
import { Button } from '@heroui/react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';
import { Shield, Link2, KeyRound, ChevronDown, ChevronUp, CheckCircle2, XCircle, Clock } from 'lucide-react';

// ── Types ─────────────────────────────────────────────────────────────────

type GatewayConfig = {
  config_id: string;
  gateway_name: string;
  merchant_id: string;
  service_id?: string;
  is_active: boolean;
  has_secret: boolean;
  created_at: string;
};

type OnboardingMode = 'MANUAL_ONLY' | 'REDIRECT_PLUS_MANUAL';

type ManualFieldName = 'merchant_id' | 'service_id' | 'secret_key';

type ManualField = {
  name: ManualFieldName;
  label: string;
  placeholder?: string;
  input_type?: string;
  helper_text?: string;
};

type ProviderCapability = {
  gateway: string;
  display_name: string;
  onboarding_mode: OnboardingMode;
  required_fields: string[];
  manual_fields?: ManualField[];
  connect_label?: string;
  connect_hint?: string;
  manual_hint: string;
};

type GatewayName = 'CASH' | 'GLOBAL_PAY' | 'GLOBAL_PAY';

const DEFAULT_CAPABILITIES: ProviderCapability[] = [
  {
    gateway: 'CASH',
    display_name: 'Cash',
    onboarding_mode: 'MANUAL_ONLY',
    required_fields: ['merchant_id', 'service_id', 'secret_key'],
    manual_fields: [
      { name: 'merchant_id', label: 'Merchant ID', placeholder: 'e.g. 12345' },
      { name: 'service_id', label: 'Service ID', placeholder: 'e.g. 23456' },
      { name: 'secret_key', label: 'Secret Key', placeholder: 'Enter your Cash secret key', input_type: 'password' },
    ],
    manual_hint: 'Enter your Cash merchant credentials from the Cash merchant dashboard.',
  },
  {
    gateway: 'GLOBAL_PAY',
    display_name: 'GlobalPay',
    onboarding_mode: 'MANUAL_ONLY',
    required_fields: ['merchant_id', 'secret_key'],
    manual_fields: [
      { name: 'merchant_id', label: 'Merchant ID', placeholder: 'e.g. 6241a1234567890abc...' },
      { name: 'secret_key', label: 'Secret Key', placeholder: 'Enter your GlobalPay secret key', input_type: 'password' },
    ],
    manual_hint: 'Enter your GlobalPay merchant credentials from the GlobalPay Business cabinet.',
  },
  {
    gateway: 'GLOBAL_PAY',
    display_name: 'Global Pay',
    onboarding_mode: 'MANUAL_ONLY',
    required_fields: ['merchant_id', 'service_id', 'secret_key'],
    manual_fields: [
      { name: 'merchant_id', label: 'OAuth Username', placeholder: 'e.g. gp_checkout_user', helper_text: 'Stored in the merchant ID field for backend compatibility.' },
      { name: 'service_id', label: 'Service ID', placeholder: 'e.g. 200123', helper_text: 'Required to create hosted checkout service tokens.' },
      { name: 'secret_key', label: 'OAuth Password', placeholder: 'Enter your Global Pay OAuth password', input_type: 'password', helper_text: 'Stored encrypted and used for Checkout Service authentication.' },
    ],
    manual_hint: 'Enter the Global Pay Checkout Service OAuth username, OAuth password, and service ID issued for your supplier account.',
  },
];

function getManualFields(cap: ProviderCapability): ManualField[] {
  return cap.manual_fields && cap.manual_fields.length > 0
    ? cap.manual_fields
    : DEFAULT_CAPABILITIES.find((candidate) => candidate.gateway === cap.gateway)?.manual_fields ?? [];
}

function getFieldLabel(cap: ProviderCapability, fieldName: ManualFieldName): string {
  return getManualFields(cap).find((field) => field.name === fieldName)?.label
    ?? DEFAULT_CAPABILITIES.find((candidate) => candidate.gateway === cap.gateway)?.manual_fields?.find((field) => field.name === fieldName)?.label
    ?? fieldName;
}

// ── Component ─────────────────────────────────────────────────────────────

export default function GlobalPayntConfigPage() {
  const [configs, setConfigs] = useState<GatewayConfig[]>([]);
  const [capabilities, setCapabilities] = useState<ProviderCapability[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  const resolvedCaps: ProviderCapability[] = capabilities.length > 0 ? capabilities : DEFAULT_CAPABILITIES;
  const capabilityByGateway = resolvedCaps.reduce<Record<string, ProviderCapability>>((acc, cap) => {
    acc[cap.gateway] = cap;
    return acc;
  }, {});

  // Manual form state
  const [expandedGateway, setExpandedGateway] = useState<GatewayName | null>(null);
  const [merchantId, setMerchantId] = useState('');
  const [serviceId, setServiceId] = useState('');
  const [secretKey, setSecretKey] = useState('');
  const [isSaving, setIsSaving] = useState(false);
  const [toast, setToast] = useState<{ type: 'success' | 'error'; message: string } | null>(null);

  const fetchConfigs = useCallback(async () => {
    try {
      const res = await apiFetch('/v1/supplier/global_paynt-config');
      if (res.ok) {
        const data = await res.json();
        setConfigs(data.configs ?? []);
        setCapabilities(data.capabilities ?? []);
      }
    } catch {
      // Silent — empty state handles it
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchConfigs();
  }, [fetchConfigs]);

  const resetForm = useCallback(() => {
    setMerchantId('');
    setServiceId('');
    setSecretKey('');
  }, []);

  const handleSave = useCallback(async (gateway: GatewayName) => {
    if (isSaving) return;
    const capability = capabilityByGateway[gateway];
    const fieldValues: Record<ManualFieldName, string> = {
      merchant_id: merchantId,
      service_id: serviceId,
      secret_key: secretKey,
    };
    const missingFields = (capability?.required_fields ?? []).filter((fieldName) => !fieldValues[fieldName as ManualFieldName]?.trim());
    if (missingFields.length > 0) {
      const labels = missingFields.map((fieldName) => capability ? getFieldLabel(capability, fieldName as ManualFieldName) : fieldName);
      setToast({ type: 'error', message: `Required: ${labels.join(', ')}.` });
      return;
    }
    setIsSaving(true);
    try {
      const res = await apiFetch('/v1/supplier/global_paynt-config', {
        method: 'POST',
        body: JSON.stringify({
          gateway_name: gateway,
          merchant_id: merchantId.trim(),
          service_id: serviceId.trim() || undefined,
          secret_key: secretKey.trim(),
        }),
      });
      if (res.ok) {
        setToast({ type: 'success', message: `${gateway} gateway configured successfully.` });
        resetForm();
        setExpandedGateway(null);
        fetchConfigs();
      } else {
        const data = await res.json().catch(() => ({ error: 'Unknown error' }));
        setToast({ type: 'error', message: data.error || 'Failed to save configuration.' });
      }
    } catch {
      setToast({ type: 'error', message: 'Network error. Please try again.' });
    } finally {
      setIsSaving(false);
    }
  }, [isSaving, merchantId, secretKey, serviceId, resetForm, fetchConfigs, capabilityByGateway]);

  const handleDeactivate = useCallback(async (configId: string, gatewayName: string) => {
    try {
      const res = await apiFetch('/v1/supplier/global_paynt-config', {
        method: 'DELETE',
        body: JSON.stringify({ config_id: configId }),
      });
      if (res.ok) {
        setToast({ type: 'success', message: `${gatewayName} gateway deactivated.` });
        fetchConfigs();
      }
    } catch {
      setToast({ type: 'error', message: 'Failed to deactivate.' });
    }
  }, [fetchConfigs]);

  const handleExpand = useCallback((gw: GatewayName) => {
    if (expandedGateway === gw) {
      setExpandedGateway(null);
      resetForm();
      return;
    }
    const existing = configs.find((c) => c.gateway_name === gw);
    setMerchantId(existing?.merchant_id ?? '');
    setServiceId(existing?.service_id ?? '');
    setSecretKey(''); // Never pre-fill secrets
    setExpandedGateway(gw);
  }, [expandedGateway, configs, resetForm]);

  // Auto-clear toast
  useEffect(() => {
    if (!toast) return;
    const t = setTimeout(() => setToast(null), 4000);
    return () => clearTimeout(t);
  }, [toast]);

  // Build a map for quick lookups
  const configByGateway = configs.reduce<Record<string, GatewayConfig>>((acc, c) => {
    acc[c.gateway_name] = c;
    return acc;
  }, {});

  return (
    <div className="max-w-3xl mx-auto py-10 px-4">
      {/* Header */}
      <div className="mb-8">
        <h1 className="md-typescale-headline-medium" style={{ color: 'var(--foreground)' }}>
          GlobalPaynt Gateways
        </h1>
        <p className="md-typescale-body-medium mt-1" style={{ color: 'var(--muted)' }}>
          Configure Cash, GlobalPay, and Global Pay credentials for your global_paynt processing.
        </p>
      </div>

      {/* Toast */}
      {toast && (
        <div
          className="mb-6 px-4 py-3 md-shape-sm md-typescale-body-medium flex items-center gap-2"
          style={{
            background: toast.type === 'success'
              ? 'var(--success)'
              : 'var(--danger)',
            color: toast.type === 'success'
              ? 'var(--success-foreground)'
              : 'var(--danger-foreground)',
          }}
        >
          {toast.type === 'success' ? <CheckCircle2 size={16} /> : <XCircle size={16} />}
          {toast.message}
        </div>
      )}

      {/* Loading */}
      {isLoading ? (
        <div className="flex items-center gap-2 py-16 justify-center" style={{ color: 'var(--muted)' }}>
          <div className="w-5 h-5 border-2 border-current border-t-transparent rounded-full animate-spin" />
          Loading configurations…
        </div>
      ) : (
        <div className="space-y-4">
          {resolvedCaps.map((cap) => {
            const cfg = configByGateway[cap.gateway];
            const isConfigured = !!cfg?.is_active;
            const isExpanded = expandedGateway === cap.gateway;
            const supportsConnect = cap.onboarding_mode === 'REDIRECT_PLUS_MANUAL';
            const manualFields = getManualFields(cap);
            const merchantField = manualFields.find((field) => field.name === 'merchant_id');
            const serviceField = manualFields.find((field) => field.name === 'service_id');

            return (
              <div
                key={cap.gateway}
                className="md-card md-elevation-1 md-shape-md overflow-hidden"
                style={{ background: 'var(--surface)' }}
              >
                {/* Provider Header */}
                <div className="p-5 flex items-center justify-between">
                  <div className="flex items-center gap-4">
                    {/* Gateway Icon */}
                    <div
                      className="w-10 h-10 md-shape-sm flex items-center justify-center"
                      style={{
                        background: isConfigured
                          ? 'color-mix(in srgb, var(--success) 12%, transparent)'
                          : 'color-mix(in srgb, var(--muted) 8%, transparent)',
                      }}
                    >
                      <CreditCardIcon gateway={cap.gateway as GatewayName} configured={isConfigured} />
                    </div>

                    <div>
                      <div className="flex items-center gap-2">
                        <span className="md-typescale-title-medium" style={{ color: 'var(--foreground)' }}>
                          {cap.display_name}
                        </span>
                        {/* Status chip */}
                        {isConfigured ? (
                          <span
                            className="md-typescale-label-small px-2 py-0.5 md-shape-full inline-flex items-center gap-1"
                            style={{
                              background: 'color-mix(in srgb, var(--success) 15%, transparent)',
                              color: 'var(--success)',
                            }}
                          >
                            <CheckCircle2 size={12} />
                            Active
                          </span>
                        ) : (
                          <span
                            className="md-typescale-label-small px-2 py-0.5 md-shape-full inline-flex items-center gap-1"
                            style={{
                              background: 'color-mix(in srgb, var(--muted) 10%, transparent)',
                              color: 'var(--muted)',
                            }}
                          >
                            <Clock size={12} />
                            Not configured
                          </span>
                        )}
                      </div>
                      {/* Merchant ID preview when configured */}
                      {cfg?.merchant_id && (
                        <div className="md-typescale-body-small mt-0.5" style={{ color: 'var(--muted)' }}>
                          {merchantField?.label || 'Merchant ID'}: <span className="font-mono">{cfg.merchant_id}</span>
                          {cfg.service_id && (<> · {serviceField?.label || 'Service ID'}: <span className="font-mono">{cfg.service_id}</span></>)}
                        </div>
                      )}
                    </div>
                  </div>

                  {/* Actions */}
                  <div className="flex items-center gap-2">
                    {/* Connect button — disabled until provider supports it */}
                    {supportsConnect ? (
                      <Button
                        variant="primary"
                        size="sm"
                        className="md-typescale-label-medium"
                      >
                        <Link2 size={14} className="mr-1" />
                        {cap.connect_label || 'Connect'}
                      </Button>
                    ) : (
                      <span
                        className="md-typescale-label-small px-3 py-1.5 md-shape-sm inline-flex items-center gap-1 cursor-default"
                        style={{
                          background: 'color-mix(in srgb, var(--muted) 6%, transparent)',
                          color: 'var(--muted)',
                          border: '1px solid var(--border)',
                        }}
                        title="Connect will be available when the provider enables merchant redirect onboarding"
                      >
                        <Link2 size={12} />
                        Connect — coming soon
                      </span>
                    )}

                    {/* Manual setup toggle */}
                    <Button
                      variant={isExpanded ? 'primary' : 'outline'}
                      size="sm"
                      onPress={() => handleExpand(cap.gateway as GatewayName)}
                      className="md-typescale-label-medium"
                    >
                      <KeyRound size={14} className="mr-1" />
                      {isConfigured ? 'Update' : 'Manual setup'}
                      {isExpanded ? <ChevronUp size={14} className="ml-1" /> : <ChevronDown size={14} className="ml-1" />}
                    </Button>

                    {/* Deactivate */}
                    {cfg?.is_active && (
                      <Button
                        variant="danger-soft"
                        size="sm"
                        onPress={() => handleDeactivate(cfg.config_id, cfg.gateway_name)}
                        className="md-typescale-label-medium"
                        style={{ color: 'var(--danger)' }}
                      >
                        Deactivate
                      </Button>
                    )}
                  </div>
                </div>

                {/* Manual Credential Form (Expandable) */}
                {isExpanded && (
                  <div
                    className="px-5 pb-5 pt-0"
                    style={{ borderTop: '1px solid var(--border)' }}
                  >
                    <p className="md-typescale-body-small mt-4 mb-4 flex items-center gap-1.5" style={{ color: 'var(--muted)' }}>
                      <Shield size={14} />
                      {cap.manual_hint}
                    </p>

                    <div className="space-y-3">
                      {manualFields.map((field) => {
                        const value = field.name === 'merchant_id'
                          ? merchantId
                          : field.name === 'service_id'
                            ? serviceId
                            : secretKey;
                        const onChange = (nextValue: string) => {
                          if (field.name === 'merchant_id') {
                            setMerchantId(nextValue);
                            return;
                          }
                          if (field.name === 'service_id') {
                            setServiceId(nextValue);
                            return;
                          }
                          setSecretKey(nextValue);
                        };

                        return (
                          <div key={field.name}>
                            <label className="md-typescale-label-medium block mb-1" style={{ color: 'var(--muted)' }}>
                              {field.label}
                            </label>
                            <input
                              type={field.input_type || (field.name === 'secret_key' ? 'password' : 'text')}
                              value={value}
                              onChange={(e) => onChange(e.target.value)}
                              placeholder={field.placeholder}
                              className="md-input-outlined w-full"
                              autoComplete={field.name === 'secret_key' ? 'new-password' : 'off'}
                            />
                            {(field.helper_text || field.name === 'secret_key') && (
                              <p className="md-typescale-body-small mt-1 flex items-center gap-1" style={{ color: 'var(--muted)' }}>
                                <Shield size={12} />
                                {field.helper_text || 'Encrypted with AES-256-GCM before storage. Never exposed after saving.'}
                              </p>
                            )}
                          </div>
                        );
                      })}
                    </div>

                    <div className="flex gap-3 mt-5 justify-end">
                      <Button
                        variant="outline"
                        onPress={() => { resetForm(); setExpandedGateway(null); }}
                        className="md-typescale-label-large"
                      >
                        Cancel
                      </Button>
                      <Button
                        variant="primary"
                        onPress={() => handleSave(cap.gateway as GatewayName)}
                        isDisabled={isSaving}
                        className="md-typescale-label-large"
                      >
                        {isSaving ? 'Saving…' : (isConfigured ? 'Update Configuration' : 'Save Configuration')}
                      </Button>
                    </div>
                  </div>
                )}
              </div>
            );
          })}

          {/* Empty state when no capabilities returned and no configs */}
          {resolvedCaps.length === 0 && configs.length === 0 && (
            <div
              className="md-card md-elevation-0 md-shape-lg py-16 flex flex-col items-center gap-3"
              style={{ background: 'var(--background)', border: '1px dashed var(--border)' }}
            >
              <Icon name="global_paynt" size={32} className="opacity-40" />
              <span className="md-typescale-headline-small" style={{ color: 'var(--muted)' }}>
                No global_paynt gateways available
              </span>
              <p className="md-typescale-body-medium" style={{ color: 'var(--muted)' }}>
                GlobalPaynt gateway support will be configured by your system administrator.
              </p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// ── Gateway Icon ──────────────────────────────────────────────────────────
// Simple SVG badge per gateway — avoids emoji per project UX doctrine.

function CreditCardIcon({ gateway, configured }: { gateway: GatewayName; configured: boolean }) {
  const color = configured ? 'var(--success)' : 'var(--muted)';
  if (gateway === 'GLOBAL_PAY') {
    return (
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round">
        <rect x="2" y="5" width="20" height="14" rx="2" />
        <line x1="2" y1="10" x2="22" y2="10" />
      </svg>
    );
  }
  if (gateway === 'GLOBAL_PAY') {
    return (
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round">
        <rect x="2" y="5" width="20" height="14" rx="2" />
        <path d="M7 15.5h4" />
        <path d="M14 15.5h3" />
        <path d="M16.5 8.5a2.5 2.5 0 1 1 0 5a2.5 2.5 0 1 1 0-5Z" />
      </svg>
    );
  }
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round">
      <rect x="2" y="5" width="20" height="14" rx="2" />
      <line x1="2" y1="10" x2="22" y2="10" />
      <circle cx="17" cy="16" r="1.5" fill={color} stroke="none" />
    </svg>
  );
}
