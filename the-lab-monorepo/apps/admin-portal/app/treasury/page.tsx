"use client";

import React, { useState } from 'react';
import { getAdminToken } from '@/lib/auth';
import { usePolling } from '@/lib/usePolling';
import Link from 'next/link';

interface TreasuryData {
    lab_revenue: number;
    supplier_payout: number;
    total_volume: number;
}

export default function TreasuryDashboard() {
    const [data, setData] = useState<TreasuryData | null>(null);
    const [isLive, setIsLive] = useState(false);
    const [loading, setLoading] = useState(true);
    const [lastRefreshed, setLastRefreshed] = useState<Date | null>(null);

    usePolling(async (signal) => {
        try {
            const token = await getAdminToken();
            const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/v1/treasury/ledger`, {
                headers: { Authorization: `Bearer ${token}` }, signal,
            });
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
        <div className="min-h-full p-6 md:p-10" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>
            {/* ── Page Header ── */}
            <header className="md-page-header">
                <div>
                    <h1 className="md-typescale-headline-medium">Treasury</h1>
                    <p className="md-typescale-body-medium mt-1" style={{ color: 'var(--muted)' }}>
                        Executive Command — System Treasury Ledger
                    </p>
                </div>
                <div className="flex items-center gap-3">
                    <Link href="/treasury/cash-holdings" className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-accent-soft text-accent-soft-foreground hover:opacity-80 transition-opacity md-typescale-label-large">
                        <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><path d="M11.8 10.9c-2.27-.59-3-1.2-3-2.15 0-1.09 1.01-1.85 2.7-1.85 1.78 0 2.44.85 2.5 2.1h2.21c-.07-1.72-1.12-3.3-3.21-3.81V3h-3v2.16c-1.94.42-3.5 1.68-3.5 3.61 0 2.31 1.91 3.46 4.7 4.13 2.5.6 3 1.48 3 2.41 0 .69-.49 1.79-2.7 1.79-2.06 0-2.87-.92-2.98-2.1h-2.2c.12 2.19 1.76 3.42 3.68 3.83V21h3v-2.15c1.95-.37 3.5-1.5 3.5-3.55 0-2.84-2.43-3.81-4.7-4.4z"/></svg>
                        Cash Holdings
                    </Link>
                    {isLive ? (
                        <div className="md-chip" style={{ cursor: 'default' }}>
                            <div className="w-2 h-2 rounded-full animate-pulse status-dot-live" />
                            <span className="md-typescale-label-small">Vault Secure</span>
                        </div>
                    ) : (
                        <div className="md-chip" style={{ cursor: 'default', borderColor: 'var(--danger)' }}>
                            <div className="w-2 h-2 rounded-full status-dot-error" />
                            <span className="md-typescale-label-small" style={{ color: 'var(--danger)' }}>Vault Offline</span>
                        </div>
                    )}
                </div>
            </header>

            {/* ── KPI Grid ── */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
                {loading ? (
                    Array.from({ length: 3 }).map((_, i) => (
                        <div key={i} className="md-card md-card-elevated md-kpi-card">
                            <div className="skeleton" style={{ height: 12, width: '50%' }} />
                            <div className="skeleton" style={{ height: 36, width: '70%' }} />
                            <div className="skeleton" style={{ height: 12, width: '40%' }} />
                        </div>
                    ))
                ) : data ? (
                    <>
                        <div className="md-card md-card-elevated md-kpi-card relative overflow-hidden md-animate-in">
                            <div className="absolute top-0 right-0 w-1 h-full" style={{ background: 'var(--success)' }} />
                            <p className="md-kpi-label">Net Revenue (5% Commission)</p>
                            <p className="md-kpi-value" style={{ fontVariantNumeric: 'tabular-nums' }}>
                                {fmt(data.lab_revenue)} <span className="md-typescale-body-small" style={{ color: 'var(--border)' }}></span>
                            </p>
                            <p className="md-kpi-sub" style={{ color: 'var(--success)' }}>Liquid — Settled</p>
                        </div>

                        <div className="md-card md-card-elevated md-kpi-card relative overflow-hidden md-animate-in" style={{ animationDelay: '50ms' }}>
                            <div className="absolute top-0 right-0 w-1 h-full" style={{ background: 'var(--warning)' }} />
                            <p className="md-kpi-label">Supplier Payout Liability</p>
                            <p className="md-kpi-value" style={{ fontVariantNumeric: 'tabular-nums' }}>
                                {fmt(data.supplier_payout)} <span className="md-typescale-body-small" style={{ color: 'var(--border)' }}></span>
                            </p>
                            <p className="md-kpi-sub" style={{ color: 'var(--warning)' }}>Pending Clearing</p>
                        </div>

                        <div className="md-card md-card-elevated md-kpi-card relative overflow-hidden md-animate-in" style={{ animationDelay: '100ms' }}>
                            <div className="absolute top-0 right-0 w-1 h-full" style={{ background: 'var(--accent)' }} />
                            <p className="md-kpi-label">Gross System Volume</p>
                            <p className="md-kpi-value" style={{ fontVariantNumeric: 'tabular-nums' }}>
                                {fmt(data.total_volume)} <span className="md-typescale-body-small" style={{ color: 'var(--border)' }}></span>
                            </p>
                            <p className="md-kpi-sub">Total throughput</p>
                        </div>
                    </>
                ) : (
                    <div className="col-span-full md-empty-state">
                        <svg viewBox="0 0 24 24" fill="currentColor"><path d="M11.8 10.9c-2.27-.59-3-1.2-3-2.15 0-1.09 1.01-1.85 2.7-1.85 1.78 0 2.44.85 2.5 2.1h2.21c-.07-1.72-1.12-3.3-3.21-3.81V3h-3v2.16c-1.94.42-3.5 1.68-3.5 3.61 0 2.31 1.91 3.46 4.7 4.13 2.5.6 3 1.48 3 2.41 0 .69-.49 1.79-2.7 1.79-2.06 0-2.87-.92-2.98-2.1h-2.2c.12 2.19 1.76 3.42 3.68 3.83V21h3v-2.15c1.95-.37 3.5-1.5 3.5-3.55 0-2.84-2.43-3.81-4.7-4.4z"/></svg>
                        <p className="md-typescale-title-medium">No treasury data available</p>
                        <p className="md-typescale-body-small">The vault is currently disconnected. Treasury data will appear when the connection is restored.</p>
                    </div>
                )}
            </div>

            {/* ── Margin Analysis ── */}
            {data && data.total_volume > 0 && (
                <div className="md-card md-card-outlined p-6 md-animate-in" style={{ animationDelay: '150ms' }}>
                    <h2 className="md-typescale-title-medium mb-4">Margin Analysis</h2>
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-6">
                        <div>
                            <p className="md-kpi-label">Commission Rate</p>
                            <p className="md-typescale-title-large mt-1" style={{ fontVariantNumeric: 'tabular-nums' }}>5.0%</p>
                        </div>
                        <div>
                            <p className="md-kpi-label">Revenue / Volume</p>
                            <p className="md-typescale-title-large mt-1" style={{ fontVariantNumeric: 'tabular-nums' }}>
                                {(data.lab_revenue / data.total_volume * 100).toFixed(2)}%
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
                                {fmt(data.total_volume - data.lab_revenue - data.supplier_payout)} <span className="md-typescale-label-small"></span>
                            </p>
                        </div>
                    </div>
                </div>
            )}

            {/* ── Footer ── */}
            {lastRefreshed && (
                <p className="mt-6 md-typescale-label-small" style={{ color: 'var(--border)' }}>
                    Last updated: {lastRefreshed.toLocaleTimeString()} · Auto-refresh every 5s
                </p>
            )}
        </div>
    );
}
