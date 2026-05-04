"use client";

import { useEffect, useState } from "react";
import { Button } from '@heroui/react';
import Link from 'next/link';
import { apiFetch, apiFetchNoQueue } from "@/lib/auth";
import { useToast } from '@/components/Toast';
import { useLocale } from '@/hooks/useLocale';

type ConfigMap = Record<string, string>;

const CONFIG_KEYS = {
    geofence_radius: { label_key: "supplier_portal.configuration.system.physics.geofence_radius_label", default: "100" },
    delivery_fee: { label_key: "supplier_portal.configuration.system.physics.delivery_fee_label", default: "15000" },
};

export default function ConfigurationPage() {
    const [config, setConfig] = useState<ConfigMap>({});
    const [platformFeePercent, setPlatformFeePercent] = useState<number>(5);
    const [isLoading, setIsLoading] = useState(true);
    const [isSavingPhysics, setIsSavingPhysics] = useState(false);
    const [isSavingPlatformFee, setIsSavingPlatformFee] = useState(false);
    const { toast } = useToast();
    const { t } = useLocale();

    useEffect(() => {
        (async () => {
            try {
                const res = await apiFetch('/v1/admin/config');
                if (res.ok) {
                    const data: ConfigMap = await res.json();
                    setConfig(data);
                }

                const feeRes = await apiFetch('/v1/admin/config/platform-fee');
                if (feeRes.ok) {
                    const feeData = await feeRes.json() as { platform_fee_percent?: number };
                    setPlatformFeePercent(feeData.platform_fee_percent ?? 5);
                }
            } catch {
                // Config load failed — use defaults
            } finally {
                setIsLoading(false);
            }
        })();
    }, []);

    const updateField = (key: string, value: string) => {
        setConfig(prev => ({ ...prev, [key]: value }));
    };

    const savePhysics = async () => {
        setIsSavingPhysics(true);
        try {
            const entries = Object.entries(CONFIG_KEYS).map(([key]) => ({
                key,
                value: config[key] ?? CONFIG_KEYS[key as keyof typeof CONFIG_KEYS].default,
            }));
            const res = await apiFetchNoQueue('/v1/admin/config', {
                method: "PUT",
                body: JSON.stringify(entries),
            });
            if (res.ok) {
                toast(t('supplier_portal.configuration.system.toast.config_saved') , 'success');
            } else {
                const err = await res.text();
                toast(t('supplier_portal.configuration.system.error.failed_with_reason', { detail: err }), 'error');
            }
        } catch (e: unknown) {
            toast(
                t('supplier_portal.configuration.system.error.generic_with_reason', {
                    detail: e instanceof Error ? e.message : t('supplier_portal.configuration.system.error.network_failure'),
                }),
                'error',
            );
        } finally {
            setIsSavingPhysics(false);
        }
    };

    const savePlatformFee = async () => {
        setIsSavingPlatformFee(true);
        try {
            const res = await apiFetchNoQueue('/v1/admin/config/platform-fee', {
                method: 'PATCH',
                body: JSON.stringify({ fee_percent: platformFeePercent }),
            });
            if (!res.ok) {
                throw new Error(await res.text());
            }
            toast(t('supplier_portal.configuration.system.toast.platform_fee_updated'), 'success');
        } catch (e: unknown) {
            toast(
                t('supplier_portal.configuration.system.error.generic_with_reason', {
                    detail: e instanceof Error ? e.message : t('supplier_portal.configuration.system.error.network_failure'),
                }),
                'error',
            );
        } finally {
            setIsSavingPlatformFee(false);
        }
    };

    return (
        <div className="min-h-full p-6 md:p-10" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>
            <header className="mb-8">
                <h1 className="md-typescale-headline-medium">{t('supplier_portal.configuration.system.title')}</h1>
                <p className="md-typescale-body-medium mt-2" style={{ color: 'var(--muted)' }}>
                    {t('supplier_portal.configuration.system.subtitle')}
                </p>
            </header>

            <div className="md-divider mb-8" />

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
                {/* Logistics Physics */}
                <div className="md-card md-card-elevated p-6">
                    <h2 className="md-typescale-title-small pb-3 mb-6" style={{ borderBottom: '1px solid var(--border)' }}>
                        {t('supplier_portal.configuration.system.physics.title')}
                    </h2>

                    {isLoading ? (
                        <div className="space-y-6">
                            {Array.from({ length: 2 }).map((_, i) => (
                                <div key={i}>
                                    <div className="w-1/3 h-3 rounded animate-pulse mb-2" style={{ background: 'var(--surface)' }} />
                                    <div className="w-full h-10 rounded animate-pulse" style={{ background: 'var(--surface)' }} />
                                </div>
                            ))}
                        </div>
                    ) : (
                        <>
                            <div className="mb-6">
                                <label className="md-typescale-label-small block mb-2" style={{ color: 'var(--muted)' }}>
                                    {t(CONFIG_KEYS.geofence_radius.label_key)}
                                </label>
                                <input
                                    type="number"
                                    min="10"
                                    max="5000"
                                    value={config.geofence_radius ?? CONFIG_KEYS.geofence_radius.default}
                                    onChange={e => updateField("geofence_radius", e.target.value)}
                                    className="md-input-outlined w-full font-mono"
                                    style={{ fontVariantNumeric: 'tabular-nums' }}
                                />
                                <p className="md-typescale-label-small mt-1.5" style={{ color: 'var(--border)' }}>
                                    {t('supplier_portal.configuration.system.physics.geofence_help')}
                                </p>
                            </div>

                            <div className="mb-6">
                                <label className="md-typescale-label-small block mb-2" style={{ color: 'var(--muted)' }}>
                                    {t(CONFIG_KEYS.delivery_fee.label_key)}
                                </label>
                                <input
                                    type="number"
                                    min="0"
                                    value={config.delivery_fee ?? CONFIG_KEYS.delivery_fee.default}
                                    onChange={e => updateField("delivery_fee", e.target.value)}
                                    className="md-input-outlined w-full font-mono"
                                    style={{ fontVariantNumeric: 'tabular-nums' }}
                                />
                                <p className="md-typescale-label-small mt-1.5" style={{ color: 'var(--border)' }}>
                                    {t('supplier_portal.configuration.system.physics.delivery_fee_help')}
                                </p>
                            </div>

                            <Button
                                variant="primary"
                                fullWidth
                                className="mt-2"
                                onPress={savePhysics}
                                isDisabled={isSavingPhysics}
                            >
                                {isSavingPhysics
                                    ? t('common.status.saving')
                                    : t('supplier_portal.configuration.system.physics.update_action')}
                            </Button>

                            <div className="mt-8 pt-6" style={{ borderTop: '1px solid var(--border)' }}>
                                <label className="md-typescale-label-small block mb-2" style={{ color: 'var(--muted)' }}>
                                    {t('supplier_portal.configuration.system.platform_fee.label')}
                                </label>
                                <input
                                    type="number"
                                    min="0"
                                    max="50"
                                    value={platformFeePercent}
                                    onChange={e => setPlatformFeePercent(Number(e.target.value || 0))}
                                    className="md-input-outlined w-full font-mono"
                                    style={{ fontVariantNumeric: 'tabular-nums' }}
                                />
                                <Button
                                    variant="outline"
                                    fullWidth
                                    className="mt-2"
                                    onPress={savePlatformFee}
                                    isDisabled={isSavingPlatformFee}
                                >
                                    {isSavingPlatformFee
                                        ? t('common.status.saving')
                                        : t('supplier_portal.configuration.system.platform_fee.update_action')}
                                </Button>
                            </div>
                        </>
                    )}
                </div>

                {/* Financial Gateways — sensitive, display-only */}
                <div className="md-card md-card-elevated p-6">
                    <h2 className="md-typescale-title-small pb-3 mb-6" style={{ borderBottom: '1px solid var(--border)' }}>
                        {t('supplier_portal.configuration.system.gateway.title')}
                    </h2>

                    <div className="mb-6">
                        <label className="md-typescale-label-small block mb-2" style={{ color: 'var(--muted)' }}>
                            {t('supplier_portal.configuration.system.gateway.click_up_merchant_id')}
                        </label>
                        <input
                            type="text"
                            placeholder={t('supplier_portal.configuration.system.gateway.configured_via_secret_manager')}
                            disabled
                            className="md-input-outlined w-full font-mono opacity-60"
                        />
                        <p className="md-typescale-label-small mt-1" style={{ color: 'var(--border)' }}>
                            {t('supplier_portal.configuration.system.gateway.keys_help')}
                        </p>
                    </div>

                    <div className="mb-2">
                        <label className="md-typescale-label-small block mb-2" style={{ color: 'var(--muted)' }}>
                            {t('supplier_portal.configuration.system.gateway.payme_secret_key')}
                        </label>
                        <input
                            type="password"
                            placeholder={t('supplier_portal.configuration.system.gateway.configured_via_secret_manager')}
                            disabled
                            className="md-input-outlined w-full font-mono opacity-60"
                        />
                    </div>

                    <div className="mt-6 pt-4" style={{ borderTop: '1px solid var(--border)' }}>
                        <Link href="/configuration/countries" className="md-btn md-btn-tonal md-typescale-label-large inline-flex px-4 py-2">
                            {t('supplier_portal.configuration.system.action.open_country_configuration')}
                        </Link>
                    </div>
                </div>
            </div>


        </div>
    );
}
