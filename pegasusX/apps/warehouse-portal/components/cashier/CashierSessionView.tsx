'use client';

import React from 'react';
import type { CashierSessionView as CashierSessionViewType } from '@pegasusx/types';
import { minorToWholeTrunc } from '@/lib/money';
import { Lock, ShieldAlert, CheckCircle2, AlertTriangle, ShieldCheck, RefreshCw, FileText } from 'lucide-react';

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
    <div className="rounded-2xl border border-neutral-800 bg-neutral-900/90 p-6 shadow-xl space-y-5 backdrop-blur-sm">
      {/* Header status bar */}
      <div className="flex flex-wrap items-center justify-between gap-4 pb-4 border-b border-neutral-800">
        <div>
          <span className="text-[11px] font-mono uppercase tracking-wider text-neutral-500 font-semibold">
            Cashier Reconciliation Session
          </span>
          <h2 className="text-xl font-bold font-mono text-neutral-100 flex items-center gap-2">
            <span>{session.session_id}</span>
          </h2>
        </div>

        <div className="flex items-center gap-2">
          {isBalanced && (
            <span className="px-3 py-1 rounded-full text-xs font-semibold bg-emerald-500/10 text-emerald-300 border border-emerald-500/30 flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
              BALANCED & CLEARED
            </span>
          )}
          {isRecountRequired && (
            <span className="px-3 py-1 rounded-full text-xs font-semibold bg-amber-500/10 text-amber-300 border border-amber-500/30 flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-amber-400 animate-pulse" />
              MANDATORY RECOUNT REQUIRED
            </span>
          )}
          {isQuarantine && (
            <span className="px-3 py-1 rounded-full text-xs font-semibold bg-rose-500/10 text-rose-300 border border-rose-500/30 flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-rose-400 animate-ping" />
              DISCREPANT QUARANTINE
            </span>
          )}
          {session.status === 'AWAITING_BLIND_COUNT_1' && (
            <span className="px-3 py-1 rounded-full text-xs font-semibold bg-blue-500/10 text-blue-300 border border-blue-500/30 flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-blue-400 animate-pulse" />
              AWAITING BLIND COUNT (ATTEMPT #1)
            </span>
          )}
        </div>
      </div>

      {/* Micro-skimming Flag Warning Banner */}
      {session.micro_skimming_flagged && (
        <div className="p-4 rounded-xl bg-purple-950/40 border border-purple-500/50 text-purple-200 flex items-start gap-3">
          <ShieldAlert size={20} className="text-purple-400 shrink-0 mt-0.5" />
          <div>
            <div className="font-bold text-xs uppercase tracking-wide text-purple-200">
              Systematic Micro-Skimming Detected
            </div>
            <p className="text-xs text-purple-300/80 mt-1 leading-relaxed">
              Driver <span className="font-mono font-bold text-white">{session.driver_id}</span> has accumulated negative cash variances across ≥ 80% of recent shifts.
              Security marking <code className="bg-purple-900/60 px-1.5 py-0.5 rounded font-mono text-purple-100">MARKING:MICRO_SKIMMING_SUSPECT</code> stamped in Spanner ontology.
            </p>
          </div>
        </div>
      )}

      {/* Recount Ratchet Alert */}
      {isRecountRequired && (
        <div className="p-4 rounded-xl bg-amber-950/40 border border-amber-500/50 text-amber-200 space-y-2">
          <div className="flex items-center gap-2 font-bold text-xs uppercase tracking-wider text-amber-300">
            <AlertTriangle size={16} className="text-amber-400" />
            <span>Blind Recount Ratchet: Variance Exceeded on Attempt #1</span>
          </div>
          <p className="text-xs text-amber-300/90 leading-relaxed">
            Variance exceeded tolerance. In accordance with PegasusX Role 7 SOP, the cashier must now perform
            a <strong>second physical recount</strong>. Expected amount remains strictly masked until Attempt #2 completes.
          </p>
        </div>
      )}

      {/* Discrepant Quarantine Alert */}
      {isQuarantine && (
        <div className="p-4 rounded-xl bg-rose-950/40 border border-rose-500/50 text-rose-200 space-y-2">
          <div className="flex items-center gap-2 font-bold text-xs uppercase tracking-wider text-rose-300">
            <ShieldAlert size={16} className="text-rose-400" />
            <span>Session Locked in Discrepant Quarantine</span>
          </div>
          <p className="text-xs text-rose-300/90 leading-relaxed">
            Variance persisted after Attempt #2 recount. Driver has been locked with <code className="bg-rose-900/60 px-1.5 py-0.5 rounded font-mono text-rose-100">MARKING:DISCREPANT_QUARANTINE</code>.
            Dual-custody supervisor authorization and physical investigation are required to clear.
          </p>
        </div>
      )}

      {/* Session Metadata Grid */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 p-4 bg-neutral-950/60 rounded-xl border border-neutral-800">
        <div>
          <span className="text-[10px] text-neutral-500 uppercase tracking-wider block font-mono font-semibold">Driver ID</span>
          <span className="text-sm font-semibold text-neutral-200 font-mono">{session.driver_id}</span>
        </div>
        <div>
          <span className="text-[10px] text-neutral-500 uppercase tracking-wider block font-mono font-semibold">Route ID</span>
          <span className="text-sm font-semibold text-neutral-200 font-mono">{session.route_id}</span>
        </div>
        <div>
          <span className="text-[10px] text-neutral-500 uppercase tracking-wider block font-mono font-semibold">Bag Seal Barcode</span>
          <span className="text-sm font-semibold text-neutral-200 font-mono">{session.bag_seal_barcode}</span>
        </div>
        <div>
          <span className="text-[10px] text-neutral-500 uppercase tracking-wider block font-mono font-semibold">Attempt Status</span>
          <span className="text-sm font-semibold text-neutral-200 font-mono">Attempt #{session.attempt_number}</span>
        </div>
      </div>

      {/* Financial Amounts Display */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {/* Counted Cash */}
        <div className="p-4 bg-neutral-950/80 rounded-xl border border-neutral-800">
          <span className="text-[10px] text-neutral-500 uppercase tracking-wider block font-mono font-semibold mb-1">
            Physical Counted Cash
          </span>
          <div className="text-xl font-bold font-mono text-emerald-400">
            {minorToWholeTrunc(session.counted_cash_minor || 0).toLocaleString()} UZS
          </div>
          <div className="text-xs text-neutral-500 font-mono mt-0.5">
            {(session.counted_cash_minor || 0).toLocaleString()} Tiyin
          </div>
        </div>

        {/* Expected Cash: Strictly masked if blind count is active */}
        <div className="p-4 bg-neutral-950/80 rounded-xl border border-neutral-800">
          <span className="text-[10px] text-neutral-500 uppercase tracking-wider block font-mono font-semibold mb-1">
            System Expected Cash
          </span>
          {isBlindCountActive ? (
            <div className="flex items-center gap-2 mt-1">
              <span className="text-xs font-mono text-amber-300 bg-amber-500/10 px-2.5 py-1 rounded-md border border-amber-500/30 flex items-center gap-1.5">
                <Lock size={12} />
                <span>MASKED (BLIND COUNT)</span>
              </span>
            </div>
          ) : session.expected_cash_minor !== undefined && session.expected_cash_minor !== null ? (
            <div>
              <div className="text-xl font-bold font-mono text-neutral-200">
                {minorToWholeTrunc(session.expected_cash_minor).toLocaleString()} UZS
              </div>
              <div className="text-xs text-neutral-500 font-mono mt-0.5">
                {session.expected_cash_minor.toLocaleString()} Tiyin
              </div>
            </div>
          ) : (
            <div className="text-xs text-neutral-500 font-mono mt-1">Pending initial count</div>
          )}
        </div>

        {/* Variance: Strictly masked if blind count is active */}
        <div className="p-4 bg-neutral-950/80 rounded-xl border border-neutral-800">
          <span className="text-[10px] text-neutral-500 uppercase tracking-wider block font-mono font-semibold mb-1">
            Reconciliation Variance
          </span>
          {isBlindCountActive ? (
            <div className="flex items-center gap-2 mt-1">
              <span className="text-xs font-mono text-amber-300 bg-amber-500/10 px-2.5 py-1 rounded-md border border-amber-500/30 flex items-center gap-1.5">
                <Lock size={12} />
                <span>MASKED (BLIND COUNT)</span>
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
                {minorToWholeTrunc(session.variance_minor).toLocaleString()} UZS
              </div>
              <div className="text-xs text-neutral-500 font-mono mt-0.5">
                {session.variance_minor.toLocaleString()} Tiyin
              </div>
            </div>
          ) : (
            <div className="text-xs text-neutral-500 font-mono mt-1">Pending count verification</div>
          )}
        </div>
      </div>

      {/* Clearance Token Badge */}
      {session.clearance_token_hash && (
        <div className="p-3.5 bg-emerald-950/30 border border-emerald-500/40 rounded-xl flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <ShieldCheck size={18} className="text-emerald-400 shrink-0" />
            <span className="text-xs text-emerald-300 font-semibold">Deposit Clearance Token Issued:</span>
            <code className="text-[11px] font-mono text-emerald-200 bg-emerald-900/50 px-2 py-0.5 rounded border border-emerald-800/60">
              {session.clearance_token_hash.slice(0, 16)}…{session.clearance_token_hash.slice(-8)}
            </code>
          </div>
          <span className="text-[11px] text-neutral-400 font-mono">Vault safe balance updated</span>
        </div>
      )}

      {/* Action Footer */}
      <div className="flex flex-wrap items-center justify-end gap-3 pt-2">
        {onOpenReclassifyModal && (
          <button
            type="button"
            onClick={onOpenReclassifyModal}
            className="flex items-center gap-1.5 px-3.5 py-2 rounded-xl text-xs font-semibold bg-neutral-800 hover:bg-neutral-700 text-neutral-200 border border-neutral-700 transition-colors"
          >
            <FileText size={14} />
            <span>Reclassify Tender via RRN</span>
          </button>
        )}

        {isQuarantine && onRequestSupervisorOverride && (
          <button
            type="button"
            onClick={onRequestSupervisorOverride}
            className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-xs font-bold bg-rose-700 hover:bg-rose-600 text-white transition-colors shadow"
          >
            <ShieldAlert size={14} />
            <span>Request Supervisor Override</span>
          </button>
        )}
      </div>
    </div>
  );
}
