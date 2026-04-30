"use client";

import { useEffect, useState } from "react";
import { Button } from '@heroui/react';
import Link from 'next/link';
import { getAdminToken } from "@/lib/auth";
import { useToast } from '@/components/Toast';

type ConfigMap = Record<string, string>;

const CONFIG_KEYS = {
    geofence_radius: { label: "Geofence Lock Radius (Meters)", default: "100" },
    delivery_fee: { label: "Base Delivery Fee (Amount)", default: "15000" },
};

export default function ConfigurationPage() {
    const [config, setConfig] = useState<ConfigMap>({});
    const [platformFeePercent, setPlatformFeePercent] = useState<number>(5);
    const [isLoading, setIsLoading] = useState(true);
    const [isSavingPhysics, setIsSavingPhysics] = useState(false);
    const [isSavingPlatformFee, setIsSavingPlatformFee] = useState(false);
    const { toast } = useToast();

    useEffect(() => {
        (async () => {
            try {
                const token = await getAdminToken();
                const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/v1/admin/config`, {
                    headers: { Authorization: `Bearer ${token}` },
                });
                if (res.ok) {
                    const data: ConfigMap = await res.json();
                    setConfig(data);
                }

                const feeRes = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/v1/admin/config/platform-fee`, {
                    headers: { Authorization: `Bearer ${token}` },
                });
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
            const token = await getAdminToken();
            const entries = Object.entries(CONFIG_KEYS).map(([key]) => ({
                key,
                value: config[key] ?? CONFIG_KEYS[key as keyof typeof CONFIG_KEYS].default,
            }));
            const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/v1/admin/config`, {
                method: "PUT",
                headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
                body: JSON.stringify(entries),
            });
            if (res.ok) {
                toast('Configuration saved.' , 'success');
            } else {
                const err = await res.text();
                toast(`Failed: ${err}`, 'error');
            }
        } catch (e: unknown) {
            toast(`Error: ${e instanceof Error ? e.message : 'Network failure'}`, 'error');
        } finally {
            setIsSavingPhysics(false);
        }
    };

    const savePlatformFee = async () => {
        setIsSavingPlatformFee(true);
        try {
            const token = await getAdminToken();
            const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/v1/admin/config/platform-fee`, {
                method: 'PATCH',
                headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
                body: JSON.stringify({ fee_percent: platformFeePercent }),
            });
            if (!res.ok) {
                throw new Error(await res.text());
            }
            toast('Platform fee updated.', 'success');
        } catch (e: unknown) {
            toast(`Error: ${e instanceof Error ? e.message : 'Network failure'}`, 'error');
        } finally {
            setIsSavingPlatformFee(false);
        }
    };

    return (
        <div className="min-h-full p-6 md:p-10" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>
            <header className="mb-8">
                <h1 className="md-typescale-headline-medium">System Configuration</h1>
                <p className="md-typescale-body-medium mt-2" style={{ color: 'var(--muted)' }}>Logistics physics and gateway parameters</p>
            </header>

            <div className="md-divider mb-8" />

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
                {/* Logistics Physics */}
                <div className="md-card md-card-elevated p-6">
                    <h2 className="md-typescale-title-small pb-3 mb-6" style={{ borderBottom: '1px solid var(--border)' }}>
                        Logistics Physics
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
                                    {CONFIG_KEYS.geofence_radius.label}
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
                                    Minimum proximity for delivery completion confirmation.
                                </p>
                            </div>

                            <div className="mb-6">
                                <label className="md-typescale-label-small block mb-2" style={{ color: 'var(--muted)' }}>
                                    {CONFIG_KEYS.delivery_fee.label}
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
                                    Applied per delivery to the retailer invoice.
                                </p>
                            </div>

                            <Button
                                variant="primary"
                                fullWidth
                                className="mt-2"
                                onPress={savePhysics}
                                isDisabled={isSavingPhysics}
                            >
                                {isSavingPhysics ? "Saving..." : "Update Physics"}
                            </Button>

                            <div className="mt-8 pt-6" style={{ borderTop: '1px solid var(--border)' }}>
                                <label className="md-typescale-label-small block mb-2" style={{ color: 'var(--muted)' }}>
                                    Platform Fee Percent (0-50)
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
                                    {isSavingPlatformFee ? "Saving..." : "Update Platform Fee"}
                                </Button>
                            </div>
                        </>
                    )}
                </div>

                {/* Financial Gateways — sensitive, display-only */}
                <div className="md-card md-card-elevated p-6">
                    <h2 className="md-typescale-title-small pb-3 mb-6" style={{ borderBottom: '1px solid var(--border)' }}>
                        Financial Gateways
                    </h2>

                    <div className="mb-6">
                        <label className="md-typescale-label-small block mb-2" style={{ color: 'var(--muted)' }}>
                            Click Up Merchant ID
                        </label>
                        <input
                            type="text"
                            placeholder="Configured via Secret Manager"
                            disabled
                            className="md-input-outlined w-full font-mono opacity-60"
                        />
                        <p className="md-typescale-label-small mt-1" style={{ color: 'var(--border)' }}>
                            Payment gateway keys are managed via infrastructure secrets.
                        </p>
                    </div>

                    <div className="mb-2">
                        <label className="md-typescale-label-small block mb-2" style={{ color: 'var(--muted)' }}>
                            Payme Secret Key
                        </label>
                        <input
                            type="password"
                            placeholder="Configured via Secret Manager"
                            disabled
                            className="md-input-outlined w-full font-mono opacity-60"
                        />
                    </div>

                    <div className="mt-6 pt-4" style={{ borderTop: '1px solid var(--border)' }}>
                        <Link href="/configuration/countries" className="md-btn md-btn-tonal md-typescale-label-large inline-flex px-4 py-2">
                            Open Country Configuration
                        </Link>
                    </div>
                </div>
            </div>


        </div>
    );
}
