"use client";

import React, { useCallback, useState, useEffect } from 'react';
import { Button } from '@heroui/react';
import { apiFetch, apiFetchNoQueue } from '@/lib/auth';
import Dialog from '@/components/Dialog';
import EmptyState from '@/components/EmptyState';
import { useToast } from '@/components/Toast';

interface RetailerEntry {
    retailer_id: string;
    shop_name: string;
    owner_name: string;
    stir_tax_id: string;
    status: string;
    submitted_at: string;
}

export default function KYCTerminal() {
    const [queue, setQueue] = useState<RetailerEntry[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [pendingAction, setPendingAction] = useState<{ id: string; action: 'approve' | 'reject'; shopName: string } | null>(null);
    const { toast } = useToast();

    const fetchPendingQueue = useCallback(async () => {
        try {
            setIsLoading(true);

            const res = await apiFetch('/v1/admin/retailer/pending');

            if (!res.ok) throw new Error("Failed to fetch pending retailers");
            const data = await res.json();
            setQueue(data || []);
        } catch (err: unknown) {
            if (err instanceof Error) {
                toast(err.message || 'Failed to load pending queue' , 'error');
            } else {
                toast('Failed to load pending queue' , 'error');
            }
        } finally {
            setIsLoading(false);
        }
    }, [toast]);

    useEffect(() => {
        fetchPendingQueue();
    }, [fetchPendingQueue]);

    const executeDecision = async (id: string, action: 'approve' | 'reject') => {
        try {
            const res = await apiFetchNoQueue(`/v1/admin/retailer/${action}`, {
                method: 'POST',
                body: JSON.stringify({ retailer_id: id })
            });

            if (!res.ok) throw new Error(`Failed to ${action} retailer`);

            // Optimistically remove from queue instead of refetching
            setQueue(prev => prev.filter(req => req.retailer_id !== id));
        } catch (err) {
            console.error(err);
            toast('KYC Operation Failed. Check connection.' , 'error');
        }
    };

    const approveRetailer = (id: string, shopName: string) => setPendingAction({ id, action: 'approve', shopName });
    const rejectRetailer = (id: string, shopName: string) => setPendingAction({ id, action: 'reject', shopName });

    const confirmAction = async () => {
        if (!pendingAction) return;
        await executeDecision(pendingAction.id, pendingAction.action);
        setPendingAction(null);
    };

    if (isLoading) {
        return (
            <div className="min-h-full p-6 md:p-10" style={{ background: 'var(--background)' }}>
                <div className="mb-10 pb-6" style={{ borderBottom: '1px solid var(--border)' }}>
                    <div className="w-48 h-7 rounded animate-pulse mb-2" style={{ background: 'var(--surface)' }} />
                    <div className="w-72 h-4 rounded animate-pulse" style={{ background: 'var(--surface)' }} />
                </div>
                <div className="w-40 h-6 rounded animate-pulse mb-4" style={{ background: 'var(--surface)' }} />
                <div className="grid gap-4">
                    {Array.from({ length: 3 }).map((_, i) => (
                        <div key={i} className="md-card md-card-elevated p-6 flex flex-col md:flex-row justify-between items-start md:items-center">
                            <div className="flex-1">
                                <div className="flex items-center gap-3 mb-3">
                                    <div className="w-16 h-6 rounded-full animate-pulse" style={{ background: 'var(--surface)' }} />
                                    <div className="w-40 h-5 rounded animate-pulse" style={{ background: 'var(--surface)' }} />
                                </div>
                                <div className="grid grid-cols-2 gap-x-8 gap-y-2">
                                    <div className="w-32 h-3 rounded animate-pulse" style={{ background: 'var(--surface)' }} />
                                    <div className="w-40 h-3 rounded animate-pulse" style={{ background: 'var(--surface)' }} />
                                    <div className="w-36 h-3 rounded animate-pulse" style={{ background: 'var(--surface)' }} />
                                    <div className="w-28 h-3 rounded animate-pulse" style={{ background: 'var(--surface)' }} />
                                </div>
                            </div>
                            <div className="flex gap-3 mt-4 md:mt-0">
                                <div className="w-20 h-10 rounded-full animate-pulse" style={{ background: 'var(--surface)' }} />
                                <div className="w-32 h-10 rounded-full animate-pulse" style={{ background: 'var(--surface)' }} />
                            </div>
                        </div>
                    ))}
                </div>
            </div>
        );
    }

    return (
        <div className="min-h-full p-6 md:p-10" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>
            <header className="mb-10 pb-6" style={{ borderBottom: '1px solid var(--border)' }}>
                <h1 className="md-typescale-headline-medium">KYC Clearance</h1>
                <p className="mt-2 md-typescale-body-medium" style={{ color: 'var(--muted)' }}>Retailer network access verification queue</p>
            </header>

            <div className="flex justify-between items-end mb-6">
                <div>
                    <h2 className="md-typescale-title-large">Pending Verifications</h2>
                    <p className="md-typescale-body-small mt-1" style={{ color: 'var(--muted)' }}>{queue.length} shops awaiting network access</p>
                </div>
            </div>

            {
                queue.length === 0 ? (
                    <EmptyState
                        icon="kyc"
                        headline="Queue Clear"
                        body="All network access requests have been processed."
                    />
                ) : (
                    <div className="grid gap-4">
                        {queue.map((req) => (
                            <div key={req.retailer_id} className="md-card md-card-elevated p-6 flex flex-col md:flex-row justify-between items-start md:items-center">

                                <div className="mb-4 md:mb-0">
                                    <div className="flex items-center gap-3 mb-1">
                                        <span className="md-chip md-chip-selected md-typescale-label-small" style={{ height: 26, cursor: 'default' }}>
                                            {req.status}
                                        </span>
                                        <span className="md-typescale-title-medium">{req.shop_name}</span>
                                    </div>
                                    <div className="grid grid-cols-2 gap-x-8 gap-y-1 mt-3">
                                        <p className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>Owner: <span style={{ color: 'var(--foreground)' }}>{req.owner_name}</span></p>
                                        <p className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>ID: <span className="font-mono" style={{ color: 'var(--accent)' }}>{req.retailer_id}</span></p>
                                        <p className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>STIR: <span className="font-mono" style={{ color: 'var(--foreground)' }}>{req.stir_tax_id}</span></p>
                                        <p className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>Time: <span style={{ color: 'var(--foreground)' }}>{req.submitted_at}</span></p>
                                    </div>
                                </div>

                                <div className="flex gap-3 w-full md:w-auto">
                                    <Button
                                        variant="outline"
                                        className="text-danger border-danger flex-1 md:flex-none"
                                        onPress={() => rejectRetailer(req.retailer_id, req.shop_name)}
                                    >
                                        Reject
                                    </Button>
                                    <Button
                                        variant="primary"
                                        className="flex-1 md:flex-none"
                                        onPress={() => approveRetailer(req.retailer_id, req.shop_name)}
                                    >
                                        Issue Clearance
                                    </Button>
                                </div>

                            </div>
                        ))}
                    </div>
                )
            }
            {/* Confirmation Dialog */}
            <Dialog
                open={!!pendingAction}
                onClose={() => setPendingAction(null)}
                title={pendingAction?.action === 'approve' ? 'Confirm Clearance' : 'Confirm Rejection'}
                actions={
                    <>
                        <Button variant="outline" onPress={() => setPendingAction(null)}>
                            Cancel
                        </Button>
                        <Button
                            variant={pendingAction?.action === 'reject' ? 'danger' : 'primary'}
                            onPress={confirmAction}
                        >
                            {pendingAction?.action === 'approve' ? 'Issue Clearance' : 'Reject'}
                        </Button>
                    </>
                }
            >
                <p className="md-typescale-body-medium" style={{ color: 'var(--muted)' }}>
                    {pendingAction?.action === 'approve'
                        ? `Grant network access to "${pendingAction?.shopName}"? This will activate their retailer account.`
                        : `Reject "${pendingAction?.shopName}"? They will need to re-submit KYC documents.`}
                </p>
            </Dialog>
        </div>
    );
}
