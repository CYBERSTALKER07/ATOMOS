'use client';

import React from 'react';
import type { CashierSessionView as CashierSessionViewType } from '@pegasusx/types';
import { TIYIN_PER_UZS } from '@pegasusx/types';

interface CashierSessionViewProps {
  session: CashierSessionViewType;
  onOpenReclassifyModal?: () => void;
  onRequestSupervisorOverride?: () => void;
}

export function CashierSessionView({
  session,
  onOpenReclassifyModal,
  onRequestSupervisorOverride,
}: CashierSessionViewProps) {
  // STRICT HYGIENE: Never display ExpectedCashMinor when session is in AWAITING_BLIND_COUNT_1 or BLIND_RECOUNT_REQUIRED!
  const isBlindCountActive =
    session.status === 'AWAITING_BLIND_COUNT_1' || session.status === 'BLIND_RECOUNT_REQUIRED';

  const isBalanced = session.status === 'BALANCED' || session.status === 'TOLERANCE_VARIANCE';
  const isQuarantine = session.status === 'DISCREPANT_QUARANTINE';
  const isRecountRequired = session.status === 'BLIND_RECOUNT_REQUIRED';

  return (
    <div className="bg-[var(--color-md-surface-card,#1e293b)] p-6 rounded-2xl border border-slate-700/60 shadow-xl space-y-5">
      {/* Header status bar */}
      <div className="flex flex-wrap items-center justify-between gap-4 pb-4 border-b border-slate-700/60">
        <div>
          <span className="text-xs font-mono uppercase text-slate-400">Session ID</span>
          <h2 className="text-xl font-bold font-mono text-slate-100">{session.session_id}</h2>
        </div>

        <div className="flex items-center gap-2">
          {isBalanced && (
            <span className="px-3 py-1 rounded-full text-xs font-semibold bg-emerald-500/20 text-emerald-300 border border-emerald-500/30 flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
              BALANCED & CLEARED
            </span>
          )}
          {isRecountRequired && (
            <span className="px-3 py-1 rounded-full text-xs font-semibold bg-amber-500/20 text-amber-300 border border-amber-500/30 flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-amber-400 animate-pulse" />
              MANDATORY RECOUNT REQUIRED
            </span>
          )}
          {isQuarantine && (
            <span className="px-3 py-1 rounded-full text-xs font-semibold bg-rose-500/20 text-rose-300 border border-rose-500/30 flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-rose-400 animate-ping" />
              DISCREPANT QUARANTINE
            </span>
          )}
          {session.status === 'AWAITING_BLIND_COUNT_1' && (
            <span className="px-3 py-1 rounded-full text-xs font-semibold bg-blue-500/20 text-blue-300 border border-blue-500/30">
              AWAITING BLIND COUNT (ATTEMPT #1)
            </span>
          )}
        </div>
      </div>

      {/* Micro-skimming Flag Warning Banner */}
      {session.micro_skimming_flagged && (
        <div className="p-4 rounded-xl bg-purple-950/40 border border-purple-500/50 text-purple-200 flex items-start gap-3">
          <div className="text-lg">⚠️</div>
          <div>
            <div className="font-semibold text-sm">Systematic Micro-Skimming Detected</div>
            <p className="text-xs text-purple-300/80 mt-0.5">
              Driver {session.driver_id} has accumulated negative variances across ≥ 80% of recent shifts.
              Security marking MARKING:MICRO_SKIMMING_SUSPECT stamped in Ontology.
            </p>
          </div>
        </div>
      )}

      {/* Recount Ratchet Alert */}
      {isRecountRequired && (
        <div className="p-4 rounded-xl bg-amber-950/40 border border-amber-500/50 text-amber-200 space-y-2">
          <div className="flex items-center gap-2 font-bold text-sm">
            <span>⚠️</span>
            <span>Blind Recount Ratchet: Discrepancy Found on Attempt #1</span>
          </div>
          <p className="text-xs text-amber-300/90 leading-relaxed">
            Count 1 variance exceeded threshold. In accordance with Role 7 SOP, the cashier must now perform
            a <strong>second physical recount</strong>. Expected amount remains strictly masked until Attempt #2 completes.
          </p>
        </div>
      )}

      {/* Discrepant Quarantine Alert */}
      {isQuarantine && (
        <div className="p-4 rounded-xl bg-rose-950/40 border border-rose-500/50 text-rose-200 space-y-2">
          <div className="flex items-center gap-2 font-bold text-sm">
            <span>🚨</span>
            <span>Session Locked in Discrepant Quarantine</span>
          </div>
          <p className="text-xs text-rose-300/90 leading-relaxed">
            Variance persisted after Attempt #2 recount. Driver has been locked with MARKING:DISCREPANT_QUARANTINE.
            Supervisor authorization and physical investigation required.
          </p>
        </div>
      )}

      {/* Session Details Grid */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 p-4 bg-slate-900/50 rounded-xl border border-slate-800">
        <div>
          <span className="text-[11px] text-slate-400 uppercase tracking-wider block font-semibold">Driver</span>
          <span className="text-sm font-medium text-slate-200 font-mono">{session.driver_id}</span>
        </div>
        <div>
          <span className="text-[11px] text-slate-400 uppercase tracking-wider block font-semibold">Route</span>
          <span className="text-sm font-medium text-slate-200 font-mono">{session.route_id}</span>
        </div>
        <div>
          <span className="text-[11px] text-slate-400 uppercase tracking-wider block font-semibold">Bag Seal</span>
          <span className="text-sm font-medium text-slate-200 font-mono">{session.bag_seal_barcode}</span>
        </div>
        <div>
          <span className="text-[11px] text-slate-400 uppercase tracking-wider block font-semibold">Attempt</span>
          <span className="text-sm font-medium text-slate-200 font-mono">#{session.attempt_number}</span>
        </div>
      </div>

      {/* Financial Amounts Display */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {/* Counted Cash */}
        <div className="p-4 bg-slate-900/70 rounded-xl border border-slate-800">
          <span className="text-xs text-slate-400 uppercase tracking-wider block font-semibold mb-1">
            Physical Counted Cash
          </span>
          <div className="text-xl font-bold font-mono text-emerald-400">
            {Math.floor((session.counted_cash_minor || 0) / TIYIN_PER_UZS).toLocaleString()} UZS
          </div>
          <div className="text-xs text-slate-500 font-mono mt-0.5">
            {(session.counted_cash_minor || 0).toLocaleString()} Tiyin
          </div>
        </div>

        {/* Expected Cash: Strictly masked if blind count is active */}
        <div className="p-4 bg-slate-900/70 rounded-xl border border-slate-800">
          <span className="text-xs text-slate-400 uppercase tracking-wider block font-semibold mb-1">
            System Expected Cash
          </span>
          {isBlindCountActive ? (
            <div className="flex items-center gap-2 mt-1">
              <span className="text-sm font-mono text-slate-400 bg-slate-800/80 px-2 py-1 rounded border border-slate-700">
                🔒 MASKED (BLIND COUNT)
              </span>
            </div>
          ) : session.expected_cash_minor !== undefined && session.expected_cash_minor !== null ? (
            <div>
              <div className="text-xl font-bold font-mono text-slate-200">
                {Math.floor(session.expected_cash_minor / TIYIN_PER_UZS).toLocaleString()} UZS
              </div>
              <div className="text-xs text-slate-500 font-mono mt-0.5">
                {session.expected_cash_minor.toLocaleString()} Tiyin
              </div>
            </div>
          ) : (
            <div className="text-sm text-slate-500 font-mono mt-1">Not available</div>
          )}
        </div>

        {/* Variance: Strictly masked if blind count is active */}
        <div className="p-4 bg-slate-900/70 rounded-xl border border-slate-800">
          <span className="text-xs text-slate-400 uppercase tracking-wider block font-semibold mb-1">
            Reconciliation Variance
          </span>
          {isBlindCountActive ? (
            <div className="flex items-center gap-2 mt-1">
              <span className="text-sm font-mono text-slate-400 bg-slate-800/80 px-2 py-1 rounded border border-slate-700">
                🔒 MASKED (BLIND COUNT)
              </span>
            </div>
          ) : session.variance_minor !== undefined && session.variance_minor !== null ? (
            <div>
              <div
                className={`text-xl font-bold font-mono ${
                  session.variance_minor === 0
                    ? 'text-emerald-400'
                    : session.variance_minor < 0
                      ? 'text-rose-400'
                      : 'text-amber-400'
                }`}
              >
                {session.variance_minor > 0 ? '+' : ''}
                {Math.floor(session.variance_minor / TIYIN_PER_UZS).toLocaleString()} UZS
              </div>
              <div className="text-xs text-slate-500 font-mono mt-0.5">
                {session.variance_minor.toLocaleString()} Tiyin
              </div>
            </div>
          ) : (
            <div className="text-sm text-slate-500 font-mono mt-1">Pending count</div>
          )}
        </div>
      </div>

      {/* Clearance Token Badge */}
      {session.clearance_token_hash && (
        <div className="p-3 bg-emerald-950/30 border border-emerald-500/40 rounded-xl flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="text-emerald-400 text-sm">✅</span>
            <span className="text-xs text-emerald-300 font-semibold">Deposit Clearance Token Issued:</span>
            <code className="text-[11px] font-mono text-emerald-200 bg-emerald-900/60 px-2 py-0.5 rounded">
              {session.clearance_token_hash.slice(0, 16)}…{session.clearance_token_hash.slice(-8)}
            </code>
          </div>
          <span className="text-[11px] text-slate-400">Vault safe balance updated</span>
        </div>
      )}

      {/* Action Footer */}
      <div className="flex flex-wrap items-center justify-end gap-3 pt-2">
        {onOpenReclassifyModal && (
          <button
            type="button"
            onClick={onOpenReclassifyModal}
            className="px-4 py-2 rounded-xl text-xs font-semibold bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 transition-colors"
          >
            Reclassify Tender via RRN
          </button>
        )}

        {isQuarantine && onRequestSupervisorOverride && (
          <button
            type="button"
            onClick={onRequestSupervisorOverride}
            className="px-4 py-2 rounded-xl text-xs font-semibold bg-rose-700 hover:bg-rose-600 text-white transition-colors shadow"
          >
            Request Supervisor Investigation Override
          </button>
        )}
      </div>
    </div>
  );
}
