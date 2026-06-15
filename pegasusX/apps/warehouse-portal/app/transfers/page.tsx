'use client';

import { useState } from 'react';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { PageSection } from '@/components/PageSection';
import { useToast } from '@/components/Toast';
import { warehouseOps } from '@/lib/warehouse-ops';

export default function TransfersPage() {
  const { toast } = useToast();
  const [volume, setVolume] = useState('20');
  const [notes, setNotes] = useState('');
  const [transferId, setTransferId] = useState('');
  const [acting, setActing] = useState(false);

  async function run(label: string, fn: () => Promise<{ state?: string; transfer_id?: string }>) {
    setActing(true);
    try {
      const resp = await fn();
      toast(`${label} · ${resp.state ?? resp.transfer_id ?? 'ok'}`, 'success');
    } catch {
      toast(`${label} failed`, 'error');
    } finally {
      setActing(false);
    }
  }

  const volumeNum = Number(volume) || 20;

  return (
    <PageTransition>
      <PageChrome
        title="Transfer actions"
        description="Factory inbound transfer controls for warehouse operators."
        skeletonVariant="form"
      >
        <PageSection title="Transfer controls" description="Emergency inbound, force receive, and transfer receipt by ID.">
        <div className="space-y-4 max-w-2xl">
          <label className="block space-y-1">
            <span className="text-sm text-[var(--muted)]">Volume (VU)</span>
            <input
              value={volume}
              onChange={(e) => setVolume(e.target.value)}
              className="w-full px-3 py-2 rounded-lg border text-sm"
              style={{
                background: 'var(--field-background)',
                borderColor: 'var(--field-border)',
                color: 'var(--field-foreground)',
              }}
            />
          </label>
          <label className="block space-y-1">
            <span className="text-sm text-[var(--muted)]">Notes (optional)</span>
            <textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={2}
              className="w-full px-3 py-2 rounded-lg border text-sm"
              style={{
                background: 'var(--field-background)',
                borderColor: 'var(--field-border)',
                color: 'var(--field-foreground)',
              }}
            />
          </label>
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              disabled={acting}
              className="button--secondary px-4 py-2 rounded-lg text-sm"
              onClick={() =>
                run('Emergency transfer', () =>
                  warehouseOps.emergencyTransfer(volumeNum, notes.trim() || undefined),
                )
              }
            >
              Emergency transfer
            </button>
            <button
              type="button"
              disabled={acting}
              className="button--secondary px-4 py-2 rounded-lg text-sm"
              onClick={() =>
                run('Force receive', () =>
                  warehouseOps.forceReceive(volumeNum, notes.trim() || undefined),
                )
              }
            >
              Force receive
            </button>
          </div>
          <label className="block space-y-1">
            <span className="text-sm text-[var(--muted)]">Transfer ID to receive</span>
            <input
              value={transferId}
              onChange={(e) => setTransferId(e.target.value)}
              className="w-full px-3 py-2 rounded-lg border text-sm font-mono"
              style={{
                background: 'var(--field-background)',
                borderColor: 'var(--field-border)',
                color: 'var(--field-foreground)',
              }}
            />
          </label>
          <button
            type="button"
            disabled={acting || !transferId.trim()}
            className="button--primary px-4 py-2 rounded-lg text-sm"
            onClick={() => run('Receive transfer', () => warehouseOps.receiveTransfer(transferId.trim()))}
          >
            Receive transfer
          </button>
        </div>
        </PageSection>
      </PageChrome>
    </PageTransition>
  );
}
