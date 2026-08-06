'use client';

import { usePortalT } from "@/lib/i18n";
import React, { useEffect, useState, useCallback } from 'react';
import { ApiError } from '@pegasusx/api-client';
import { payloadApplyReassignKey, payloadRecommendReassignKey } from '@pegasusx/api-client/idempotency';
import type { ReassignmentCandidate } from '@pegasusx/types';
import { createSupplierApi } from '@/lib/api';
import Icon from '@/components/Icon';
import EmptyState from '@/components/EmptyState';

const supplierApi = createSupplierApi();

export type ReDispatchDialogProps = {
  open: boolean;
  orderId: string;
  onClose: () => void;
  onSuccess: () => void;
};

export function ReDispatchDialog({ open, orderId, onClose, onSuccess }: ReDispatchDialogProps) {
  const t = usePortalT();
  const [candidates, setCandidates] = useState<ReassignmentCandidate[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [actingId, setActingId] = useState<string | null>(null);
  const [reassignType, setReassignType] = useState<'COMPLETE' | 'PARTIAL'>('COMPLETE');

  const loadCandidates = useCallback(async () => {
    if (!open || !orderId) return;
    setLoading(true);
    setError(null);
    try {
      const response = await supplierApi.postSupplierRecommendReassign(
        { order_id: orderId },
        payloadRecommendReassignKey(orderId)
      );
      setCandidates(response.candidates ?? []);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'failed_to_load_recommendations');
    } finally {
      setLoading(false);
    }
  }, [open, orderId]);

  useEffect(() => {
    if (open) {
      void loadCandidates();
    } else {
      setCandidates([]);
      setActingId(null);
      setError(null);
      setReassignType('COMPLETE');
    }
  }, [open, loadCandidates]);

  const handleReassign = async (newDriverId: string) => {
    if (!orderId) return;
    setActingId(newDriverId);
    setError(null);
    try {
      await supplierApi.postSupplierApplyReassign(
        {
          order_id: orderId,
          new_driver_id: newDriverId,
          reassign_type: reassignType,
        },
        payloadApplyReassignKey(orderId, newDriverId)
      );
      onSuccess();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'failed_to_reassign');
    } finally {
      setActingId(null);
    }
  };

  if (!open) return null;

  return (
    <dialog
      open
      className="rounded-xl border border-[var(--border)] bg-[var(--surface)] p-5 backdrop:bg-black/40 max-w-lg w-[calc(100%-2rem)] fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-50 flex flex-col max-h-[85vh]"
    >
      <div className="flex items-center justify-between mb-4">
        <div>
          <h2 className="text-lg font-semibold">{t("supplier_portal.orders.re_dispatch_dialog.text.reassign_order")}</h2>
          <p className="text-sm text-[var(--muted)] mt-1 font-mono">{orderId}</p>
        </div>
        <button
          type="button"
          onClick={onClose}
          className="p-2 -mr-2 rounded-full hover:bg-[var(--default)] transition-colors"
          disabled={actingId !== null}
        >
          <Icon name="close" size={20} />
        </button>
      </div>

      <div className="mb-4">
        <label className="block text-sm font-medium mb-2">{t("supplier_portal.orders.re_dispatch_dialog.text.reassignment_type")}</label>
        <div className="flex gap-2">
          <button
            type="button"
            className={`flex-1 px-3 py-2 rounded-lg border text-sm font-medium transition-colors ${
              reassignType === 'COMPLETE'
                ? 'bg-[var(--primary)] text-[var(--on-primary)] border-[var(--primary)]'
                : 'bg-[var(--surface)] text-[var(--foreground)] border-[var(--border)] hover:bg-[var(--default)]'
            }`}
            onClick={() => setReassignType('COMPLETE')}
            disabled={actingId !== null}
          >
            Complete
          </button>
          <button
            type="button"
            className={`flex-1 px-3 py-2 rounded-lg border text-sm font-medium transition-colors ${
              reassignType === 'PARTIAL'
                ? 'bg-[var(--primary)] text-[var(--on-primary)] border-[var(--primary)]'
                : 'bg-[var(--surface)] text-[var(--foreground)] border-[var(--border)] hover:bg-[var(--default)]'
            }`}
            onClick={() => setReassignType('PARTIAL')}
            disabled={actingId !== null}
          >
            Partial
          </button>
        </div>
        <p className="text-xs text-[var(--muted)] mt-2">
          {reassignType === 'COMPLETE'
            ? 'The entire order will be handed over to the new driver.'
            : 'Both drivers will fulfill parts of this order. Requires handshake confirmation.'}
        </p>
      </div>

      <div className="flex-1 overflow-y-auto min-h-0 border-t border-[var(--border)] -mx-5 px-5 pt-4">
        {error ? (
          <div className="p-3 mb-4 rounded-lg bg-[var(--danger)]/10 text-[var(--danger)] text-sm border border-[var(--danger)]/20">
            {error}
          </div>
        ) : null}

        {loading ? (
          <div className="space-y-3">
            {[1, 2, 3].map((i) => (
              <div key={i} className="h-16 rounded-xl bg-[var(--color-md-surface-container-low)] animate-pulse" />
            ))}
          </div>
        ) : candidates.length === 0 && !error ? (
          <EmptyState
            icon="directions_car"
            headline={t("supplier_portal.residual.text.no_nearby_drivers")}
            body={t("supplier_portal.residual.text.there_are_no_eligible_drivers_nearby_to_take_this_order_right_no")}
          />
        ) : (
          <div className="space-y-3">
            {candidates.map((candidate) => (
              <div
                key={candidate.driver_id}
                className="flex items-center justify-between p-3 rounded-xl border border-[var(--border)] bg-[var(--surface)] hover:border-[var(--primary)]/30 transition-colors"
              >
                <div>
                  <h3 className="font-medium text-sm">
                    {candidate.name || 'Unknown Driver'}
                  </h3>
                  <p className="text-xs text-[var(--muted)] font-mono mt-0.5">
                    {candidate.vehicle_id}
                  </p>
                  {candidate.eta_seconds ? (
                    <p className="text-xs text-[var(--primary)] mt-1">
                      {Math.ceil(candidate.eta_seconds / 60)} min away
                    </p>
                  ) : null}
                </div>
                <button
                  type="button"
                  onClick={() => void handleReassign(candidate.driver_id)}
                  disabled={actingId !== null}
                  className="md-btn md-btn-tonal px-4 py-2 shrink-0 disabled:opacity-50"
                >
                  {actingId === candidate.driver_id ? 'Assigning…' : 'Assign'}
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    </dialog>
  );
}
