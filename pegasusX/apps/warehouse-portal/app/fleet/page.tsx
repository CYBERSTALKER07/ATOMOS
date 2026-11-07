'use client';

import React, { useState } from 'react';
import { FleetDVIRScreen } from '@/components/fleet/FleetDVIRScreen';
import { YardGateCheckScreen } from '@/components/fleet/YardGateCheckScreen';
import { PredictiveReeferCard } from '@/components/fleet/PredictiveReeferCard';
import { MidRouteRescueModal } from '@/components/fleet/MidRouteRescueModal';

export default function FleetPage() {
  const [activeTab, setActiveTab] = useState<'dvir' | 'gate' | 'reefer'>('dvir');
  const [isRescueOpen, setIsRescueOpen] = useState(false);

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-100 flex items-center gap-2">
            <span>🚛</span>
            <span>Role 9: Fleet, Yard & Maintenance Operations</span>
          </h1>
          <p className="text-sm text-slate-400 mt-1">
            Pre-trip 8-point DVIR inspection, Red-Tag Grounding Gate, ALPR yard security check-in/out, predictive reefer maintenance, and mid-route breakdown rescues.
          </p>
        </div>

        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={() => setIsRescueOpen(true)}
            className="px-4 py-2 rounded-xl text-xs font-bold bg-amber-600 hover:bg-amber-500 text-white shadow transition-all flex items-center gap-1.5"
          >
            <span>🚨</span>
            <span>Mid-Route Rescue Console</span>
          </button>

          {/* Tab selector */}
          <div className="flex items-center gap-1 p-1 bg-slate-900 rounded-xl border border-slate-800">
            <button
              type="button"
              onClick={() => setActiveTab('dvir')}
              className={`px-4 py-2 rounded-lg text-xs font-semibold transition-all ${
                activeTab === 'dvir'
                  ? 'bg-blue-600 text-white shadow'
                  : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              8-Point DVIR
            </button>
            <button
              type="button"
              onClick={() => setActiveTab('gate')}
              className={`px-4 py-2 rounded-lg text-xs font-semibold transition-all ${
                activeTab === 'gate'
                  ? 'bg-blue-600 text-white shadow'
                  : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              Yard Gate ALPR
            </button>
            <button
              type="button"
              onClick={() => setActiveTab('reefer')}
              className={`px-4 py-2 rounded-lg text-xs font-semibold transition-all ${
                activeTab === 'reefer'
                  ? 'bg-blue-600 text-white shadow'
                  : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              Predictive Reefer
            </button>
          </div>
        </div>
      </div>

      {/* Main Tab Panels */}
      {activeTab === 'dvir' && <FleetDVIRScreen />}
      {activeTab === 'gate' && <YardGateCheckScreen />}
      {activeTab === 'reefer' && <PredictiveReeferCard />}

      {/* Mid-Route Breakdown Rescue Modal */}
      <MidRouteRescueModal
        isOpen={isRescueOpen}
        onClose={() => setIsRescueOpen(false)}
      />
    </div>
  );
}
