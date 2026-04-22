"use client";

import { useState } from "react";
import { getAdminToken } from '@/lib/auth';
import { usePolling } from '@/lib/usePolling';

type LedgerEntry = {
    order_id: string;
    timestamp?: string; // We'll mock if missing from telemetry
    retailer_id: string;
    payment_gateway: string;
    amount: number;
    state: string;
};

type TreasuryReport = {
    lab_revenue: number;
    supplier_payout: number;
    total_volume: number;
};

export default function LedgerPage() {
    const [entries, setEntries] = useState<LedgerEntry[]>([]);
    const [treasury, setTreasury] = useState<TreasuryReport | null>(null);
    const [isLoading, setIsLoading] = useState<boolean>(true);
    const [page, setPage] = useState(1);
    const [pageSize, setPageSize] = useState(25);
    const [hasMore, setHasMore] = useState(false);

    const offset = (page - 1) * pageSize;
    const canPrev = page > 1;
    const canNext = hasMore;

    usePolling(async (signal) => {
        try {
            const token = await getAdminToken();

            const [ordersRes, treasuryRes] = await Promise.all([
                fetch(`${process.env.NEXT_PUBLIC_API_URL}/v1/orders?limit=${pageSize + 1}&offset=${offset}`, {
                    headers: { 'Authorization': `Bearer ${token}` }, signal,
                }),
                fetch(`${process.env.NEXT_PUBLIC_API_URL}/v1/treasury/ledger`, {
                    headers: { 'Authorization': `Bearer ${token}` }, signal,
                }),
            ]);

            if (!ordersRes.ok) throw new Error("HTTP " + ordersRes.status);
            const data: LedgerEntry[] = await ordersRes.json();
            const rows = data || [];
            setHasMore(rows.length > pageSize);
            setEntries(rows.slice(0, pageSize));
            if (treasuryRes.ok) {
                setTreasury(await treasuryRes.json());
            }
            setIsLoading(false);
        } catch (err) {
            if ((err as Error).name === 'AbortError') return;
            console.error("Telemetry Sync Error:", err);
            setHasMore(false);
            if (isLoading) {
                setEntries([]);
                setIsLoading(false);
            }
        }
    }, 3000, [pageSize, offset]);

    const getStatusBadge = (status: string | undefined) => {
        if (!status) {
            return <span className="md-chip md-typescale-label-small" style={{ background: 'var(--danger)', color: 'var(--danger-foreground)', borderColor: 'transparent', cursor: 'default', height: 26 }}>Unknown</span>;
        }
        const map: Record<string, { bg: string; color: string; label: string }> = {
            PENDING:          { bg: 'var(--surface)', color: 'var(--muted)', label: 'Pending' },
            FAILED_HANDSHAKE: { bg: 'var(--danger)', color: 'var(--danger-foreground)', label: 'Fault' },
            COMPLETED:        { bg: 'var(--default)', color: 'var(--default-foreground)', label: 'Completed' },
            EN_ROUTE:         { bg: 'var(--accent)', color: 'var(--accent-foreground)', label: 'Dispatched' },
        };
        const s = map[status.toUpperCase()] ?? { bg: 'var(--accent-soft)', color: 'var(--accent-soft-foreground)', label: status.toUpperCase() };
        return <span className="md-chip md-typescale-label-small" style={{ background: s.bg, color: s.color, borderColor: 'transparent', cursor: 'default', height: 26 }}>{s.label}</span>;
    };

    return (
        <div className="min-h-full p-6 md:p-10" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>
            {/* Page Header */}
            <header className="mb-10">
                <h1 className="md-typescale-headline-medium" style={{ color: 'var(--foreground)' }}>Financial Ledger</h1>
                <p className="md-typescale-body-medium mt-2" style={{ color: 'var(--muted)' }}>System Settlement & Reconciliation Engine</p>
            </header>

            {/* KPI Cards — M3 Filled Cards */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-10">
                {isLoading ? (
                    Array.from({ length: 3 }).map((_, i) => (
                        <div key={i} className="md-card md-card-elevated p-6 h-32 flex flex-col justify-between">
                            <div className="w-1/2 h-3 rounded animate-pulse" style={{ background: 'var(--surface)' }} />
                            <div className="w-2/3 h-8 rounded mt-4 animate-pulse" style={{ background: 'var(--surface)' }} />
                        </div>
                    ))
                ) : (
                    <>
                        {(() => {
                            const grossVolume = treasury?.total_volume
                                ?? entries.reduce((sum, e) => sum + (e.amount ?? 0), 0);
                            const pendingSettlement = treasury
                                ? treasury.total_volume - treasury.lab_revenue - treasury.supplier_payout
                                : entries.filter(e => e.state !== 'COMPLETED').reduce((sum, e) => sum + (e.amount ?? 0), 0);
                            const completedDrops = entries.filter(e => e.state === 'COMPLETED').length;
                            return [
                                { label: "Gross Volume (Amount)", value: grossVolume.toLocaleString() },
                                { label: "Pending Settlement", value: pendingSettlement.toLocaleString() },
                                { label: "Completed Drops", value: String(completedDrops) },
                            ];
                        })().map(({ label, value }, i) => (
                            <div key={i} className="md-card md-card-elevated p-6 flex flex-col justify-between cursor-default md-animate-in" style={{ animationDelay: `${i * 50}ms` }}>
                                <p className="md-typescale-label-small mb-4" style={{ color: 'var(--muted)' }}>{label}</p>
                                <p className="md-typescale-headline-small tracking-tight" style={{ color: 'var(--foreground)', fontVariantNumeric: 'tabular-nums' }}>{value}</p>
                            </div>
                        ))}
                    </>
                )}
            </div>

            {/* M3 Data Table */}
            <main className="md-animate-in" style={{ animationDelay: '150ms' }}>
                <div className="w-full overflow-hidden md-card md-card-outlined p-0">
                    <table className="md-table">
                        <thead>
                            <tr>
                                <th>Order ID</th>
                                <th>Time</th>
                                <th>Retailer</th>
                                <th>Channel</th>
                                <th className="text-right">Settlement (Amount)</th>
                                <th className="text-right">Outcome</th>
                            </tr>
                        </thead>
                        <tbody>
                            {isLoading ? (
                                Array.from({ length: 8 }).map((_, i) => (
                                    <tr key={`skel-${i}`}>
                                        <td><div className="w-24 h-4 rounded animate-pulse" style={{ background: 'var(--surface)' }} /></td>
                                        <td><div className="w-24 h-4 rounded animate-pulse" style={{ background: 'var(--surface)' }} /></td>
                                        <td><div className="w-20 h-4 rounded animate-pulse" style={{ background: 'var(--surface)' }} /></td>
                                        <td><div className="w-16 h-4 rounded animate-pulse" style={{ background: 'var(--surface)' }} /></td>
                                        <td className="flex justify-end"><div className="w-20 h-4 rounded animate-pulse" style={{ background: 'var(--surface)' }} /></td>
                                        <td><div className="w-24 h-6 rounded animate-pulse ml-auto" style={{ background: 'var(--surface)' }} /></td>
                                    </tr>
                                ))
                            ) : entries.length === 0 ? (
                                <tr>
                                    <td colSpan={6} className="p-12 text-center md-typescale-body-medium" style={{ color: 'var(--muted)' }}>
                                        No ledger entries found
                                    </td>
                                </tr>
                            ) : (
                                entries.map((entry, i) => {
                                    return (
                                        <tr
                                            key={entry.order_id || `idx-${i}`}
                                            className="transition-colors cursor-pointer"
                                        >
                                            <td className="font-mono md-typescale-body-small font-medium">
                                                {entry.order_id}
                                            </td>
                                            <td className="md-typescale-body-small whitespace-nowrap" style={{ color: 'var(--muted)' }}>
                                                {entry.timestamp || "14:02:11"}
                                            </td>
                                            <td className="md-typescale-body-medium font-medium">
                                                {entry.retailer_id}
                                            </td>
                                            <td className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>
                                                {entry.payment_gateway}
                                            </td>
                                            <td className="font-mono font-medium text-right" style={{ fontVariantNumeric: 'tabular-nums' }}>
                                                {entry.amount ? entry.amount.toLocaleString() : "0"}
                                            </td>
                                            <td className="text-right">
                                                {getStatusBadge(entry.state)}
                                            </td>
                                        </tr>
                                    );
                                })
                            )}
                        </tbody>
                    </table>
                    {entries.length > 0 && (
                        <div className="flex items-center justify-between px-4 py-3" style={{ borderTop: '1px solid var(--border)' }}>
                            <div className="flex items-center gap-2">
                                <label className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Rows</label>
                                <select
                                    value={pageSize}
                                    onChange={(e) => {
                                        setPageSize(Number(e.target.value));
                                        setPage(1);
                                    }}
                                    className="md-typescale-label-small px-2 py-1 rounded-md"
                                    style={{ border: '1px solid var(--border)', background: 'var(--surface)', color: 'var(--foreground)' }}
                                >
                                    {[10, 25, 50, 100].map((s) => (
                                        <option key={s} value={s}>{s}</option>
                                    ))}
                                </select>
                            </div>
                            <div className="flex items-center gap-3">
                                <span className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
                                    Page {page}
                                </span>
                                <div className="flex gap-1">
                                    <button className="md-btn md-btn-tonal" onClick={() => setPage(1)} disabled={!canPrev}>First</button>
                                    <button className="md-btn md-btn-tonal" onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={!canPrev}>Prev</button>
                                    <button className="md-btn md-btn-tonal" onClick={() => setPage((p) => p + 1)} disabled={!canNext}>Next</button>
                                </div>
                            </div>
                        </div>
                    )}
                </div>
            </main>
        </div>
    );
}
