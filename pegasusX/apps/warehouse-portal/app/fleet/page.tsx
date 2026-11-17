'use client';

import React, { useState } from 'react';
import { FleetDVIRScreen } from '@/components/fleet/FleetDVIRScreen';
import { YardGateCheckScreen } from '@/components/fleet/YardGateCheckScreen';
import { MidRouteRescueModal } from '@/components/fleet/MidRouteRescueModal';
import { Truck, ClipboardCheck, ArrowRightLeft, ShieldAlert } from 'lucide-react';

export default function FleetPage() {
  const [activeTab, setActiveTab] = useState<'dvir' | 'gate'>('dvir');
  const [isRescueOpen, setIsRescueOpen] = useState(false);

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-neutral-100 flex items-center gap-2">
            <Truck className="text-blue-400" size={24} />
            <span>Role 9: Fleet, Yard & Maintenance Operations</span>
          </h1>
          <p className="text-xs text-neutral-400 mt-1">
            Pre-trip 8-point DVIR inspection, Red-Tag Grounding Gate, ALPR yard security check-in/out, and mid-route breakdown rescues.
          </p>
        </div>

        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={() => setIsRescueOpen(true)}
            className="px-4 py-2 rounded-xl text-xs font-bold bg-amber-600 hover:bg-amber-500 text-white shadow transition-all flex items-center gap-1.5"
          >
            <ArrowRightLeft size={14} />
            <span>Mid-Route Rescue Console</span>
          </button>

          {/* Tab selector */}
          <div className="flex items-center gap-1 p-1 bg-neutral-900/90 rounded-xl border border-neutral-800">
            <button
              type="button"
              onClick={() => setActiveTab('dvir')}
              className={`flex items-center gap-1.5 px-3.5 py-2 rounded-lg text-xs font-bold transition-all ${
                activeTab === 'dvir'
                  ? 'bg-blue-600 text-white shadow'
                  : 'text-neutral-400 hover:text-neutral-200'
              }`}
            >
              <ClipboardCheck size={14} />
              <span>8-Point DVIR</span>
            </button>
            <button
              type="button"
              onClick={() => setActiveTab('gate')}
              className={`flex items-center gap-1.5 px-3.5 py-2 rounded-lg text-xs font-bold transition-all ${
                activeTab === 'gate'
                  ? 'bg-blue-600 text-white shadow'
                  : 'text-neutral-400 hover:text-neutral-200'
              }`}
            >
              <Truck size={14} />
              <span>Yard Gate ALPR</span>
            </button>
          </div>
        </div>
      </div>

      {/* Main Tab Panels */}
      {activeTab === 'dvir' && <FleetDVIRScreen />}
      {activeTab === 'gate' && <YardGateCheckScreen />}

      {/* Mid-Route Breakdown Rescue Modal */}
      <MidRouteRescueModal
        isOpen={isRescueOpen}
        onClose={() => setIsRescueOpen(false)}
      />
    </div>
  );
}
