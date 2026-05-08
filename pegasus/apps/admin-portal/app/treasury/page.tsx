"use client";

import { useState } from 'react';
import { apiFetch } from '@/lib/auth';
import { useSyncHub } from '@/lib/useSyncHub';
import Link from 'next/link';
import Icon from '@/components/Icon';
import EmptyState from '@/components/EmptyState';

interface TreasuryData {
    platform_revenue: number;
    supplier_payout: number;
    total_volume: number;
}

export default function TreasuryDashboard() {
    const [data, setData] = useState<TreasuryData | null>(null);
    const [isLive, setIsLive] = useState(false);
    const [loading, setLoading] = useState(true);
    const [lastRefreshed, setLastRefreshed] = useState<Date | null>(null);

    useSyncHub("POLL", "default", async (signal) => {
        try {
            const res = await apiFetch('/v1/treasury/ledger', { signal });
            if (!res.ok) throw new Error("Vault disconnected");
            const json = await res.json();
            setData(json);
            setIsLive(true);
            setLastRefreshed(new Date());
        } catch (err) {
            if ((err as Error).name === 'AbortError') return;
            console.error("[LEDGER ERROR]", err);
            setIsLive(false);
        } finally {
            setLoading(false);
        }
    }, 5000);

    const fmt = (n: number) => new Intl.NumberFormat('en-US').format(n);

    return (
        <div className="min-h-full p-6 md:p-10" style={{ background: 'var(--desk-bg)', color: 'var(--desk-text-primary)' }}>
            {/* ── Page Header ── */}
            <header className="md-page-header">
                <div>
                    <h1 className="md-typescale-headline-medium">Treasury</h1>
                    <p className="md-typescale-body-medium mt-1" style={{ color: 'var(--desk-text-secondary)' }}>
                        Executive Command — System Treasury Ledger
                    </p>
                </div>
                <div className="flex items-center gap-3">
                    <Link href="/treasury/cash-holdings" className="desk-btn-secondary inline-flex items-center gap-2 px-4 py-2 rounded-lg md-typescale-label-large">
                        <Icon name="treasury" size={18} />
                        Cash Holdings
                    </Link>
                    {isLive ? (
                        <div className="md-chip" style={{ cursor: 'default' }}>
                            <span className="desk-status-dot desk-status-dot--success" />
                            <span className="md-typescale-label-small">Vault Secure</span>
                        </div>
                    ) : (
                        <div className="md-chip" style={{ cursor: 'default', borderColor: 'var(--danger)' }}>
                            <span className="desk-status-dot desk-status-dot--danger" />
                            <span className="md-typescale-label-small" style={{ color: 'var(--danger)' }}>Vault Offline</span>
                        </div>
                    )}
                </div>
            </header>

            {/* ── KPI Grid ── */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
                {loading ? (
                    Array.from({ length: 3 }).map((_, i) => (
                        <div key={i} className="desk-card md-kpi-card">
                            <div className="skeleton" style={{ height: 12, width: '50%' }} />
                            <div className="skeleton" style={{ height: 36, width: '70%' }} />
                            <div className="skeleton" style={{ height: 12, width: '40%' }} />
                        </div>
                    ))
                ) : data ? (
                    <>
                        <div className="desk-card md-kpi-card relative overflow-hidden">
                            <div className="absolute top-0 right-0 w-1 h-full" style={{ background: 'var(--success)' }} />
                            <p className="md-kpi-label">Net Revenue (5% Commission)</p>
                            <p className="md-kpi-value" style={{ fontVariantNumeric: 'tabular-nums' }}>
                                {fmt(data.platform_revenue)}
                            </p>
                            <p className="md-kpi-sub" style={{ color: 'var(--success)' }}>Liquid — Settled</p>
                        </div>

                        <div className="desk-card md-kpi-card relative overflow-hidden">
                            <div className="absolute top-0 right-0 w-1 h-full" style={{ background: 'var(--warning)' }} />
                            <p className="md-kpi-label">Supplier Payout Liability</p>
                            <p className="md-kpi-value" style={{ fontVariantNumeric: 'tabular-nums' }}>
                                {fmt(data.supplier_payout)}
                            </p>
                            <p className="md-kpi-sub" style={{ color: 'var(--warning)' }}>Pending Clearing</p>
                        </div>

                        <div className="desk-card md-kpi-card relative overflow-hidden">
                            <div className="absolute top-0 right-0 w-1 h-full" style={{ background: 'var(--accent)' }} />
                            <p className="md-kpi-label">Gross System Volume</p>
                            <p className="md-kpi-value" style={{ fontVariantNumeric: 'tabular-nums' }}>
                                {fmt(data.total_volume)}
                            </p>
                            <p className="md-kpi-sub">Total throughput</p>
                        </div>
                    </>
                ) : (
                    <div className="col-span-full">
                        <EmptyState
                            icon="treasury"
                            headline="No treasury data available"
                            body="The vault is currently disconnected. Treasury data will appear when the connection is restored."
                        />
                    </div>
                )}
            </div>

            {/* ── Margin Analysis ── */}
            {data && data.total_volume > 0 && (
                <div className="desk-card p-6">
                    <h2 className="md-typescale-title-medium mb-4">Margin Analysis</h2>
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-6">
                        <div>
                            <p className="md-kpi-label">Commission Rate</p>
                            <p className="md-typescale-title-large mt-1" style={{ fontVariantNumeric: 'tabular-nums' }}>5.0%</p>
                        </div>
                        <div>
                            <p className="md-kpi-label">Revenue / Volume</p>
                            <p className="md-typescale-title-large mt-1" style={{ fontVariantNumeric: 'tabular-nums' }}>
                                {(data.platform_revenue / data.total_volume * 100).toFixed(2)}%
                            </p>
                        </div>
                        <div>
                            <p className="md-kpi-label">Payout Ratio</p>
                            <p className="md-typescale-title-large mt-1" style={{ fontVariantNumeric: 'tabular-nums' }}>
                                {(data.supplier_payout / data.total_volume * 100).toFixed(1)}%
                            </p>
                        </div>
                        <div>
                            <p className="md-kpi-label">Clearing Buffer</p>
                            <p className="md-typescale-title-large mt-1" style={{ fontVariantNumeric: 'tabular-nums' }}>
                                {fmt(data.total_volume - data.platform_revenue - data.supplier_payout)}
                            </p>
                        </div>
                    </div>
                </div>
            )}

            {/* ── Footer ── */}
            {lastRefreshed && (
                <p className="mt-6 md-typescale-label-small" style={{ color: 'var(--desk-text-tertiary)' }}>
                    Last updated: {lastRefreshed.toLocaleTimeString()} · Auto-refresh every 5s
                </p>
            )}
        </div>
    );
}
