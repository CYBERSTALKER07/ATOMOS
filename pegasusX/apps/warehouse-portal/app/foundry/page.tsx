'use client';

import React, { useState } from 'react';
import { OntologyBlastRadiusViewer } from '@/components/foundry/OntologyBlastRadiusViewer';
import { ScenarioSandboxWorkbench } from '@/components/foundry/ScenarioSandboxWorkbench';

export default function FoundryPage() {
  const [activeTab, setActiveTab] = useState<'blast' | 'sandbox'>('blast');

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-100 flex items-center gap-2">
            <span>🔮</span>
            <span>Palantir Foundry Semantic-Kinetic-Dynamic Triad</span>
          </h1>
          <p className="text-sm text-slate-400 mt-1">
            Explore multi-hop lineage, trigger 1-click contaminated lot recalls across SSCC pallets, and stage Copy-on-Write scenarios.
          </p>
        </div>

        {/* Tab switch */}
        <div className="flex items-center gap-1.5 p-1 bg-slate-900 rounded-xl border border-slate-800">
          <button
            type="button"
            onClick={() => setActiveTab('blast')}
            className={`px-4 py-2 rounded-lg text-xs font-semibold transition-all ${
              activeTab === 'blast'
                ? 'bg-rose-600 text-white shadow'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            Ontology Blast-Radius
          </button>
          <button
            type="button"
            onClick={() => setActiveTab('sandbox')}
            className={`px-4 py-2 rounded-lg text-xs font-semibold transition-all ${
              activeTab === 'sandbox'
                ? 'bg-purple-600 text-white shadow'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            Scenario Sandbox
          </button>
        </div>
      </div>

      {activeTab === 'blast' && <OntologyBlastRadiusViewer />}
      {activeTab === 'sandbox' && <ScenarioSandboxWorkbench />}
    </div>
  );
}
