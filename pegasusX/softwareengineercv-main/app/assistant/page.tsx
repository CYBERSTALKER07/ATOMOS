'use client';

import '@openuidev/react-ui/components.css';
import '@openuidev/react-ui/styles/index.css';

import {
  AgentInterface,
  useSystemThemeMode,
  useThread,
} from '@openuidev/react-ui';
import {
  fetchLLM,
  vercelAIAdapter,
  vercelAIMessageFormat,
} from '@openuidev/react-headless';
import { openuiLibrary } from '@openuidev/react-ui/genui-lib';
import Link from 'next/link';
import { ArrowLeft } from 'lucide-react';

const llm = fetchLLM({
  url: '/api/chat',
  streamAdapter: vercelAIAdapter(),
  messageFormat: vercelAIMessageFormat,
});

const STARTERS = [
  {
    tag: '[FLEET TELEMETRY]',
    displayText: 'Audit Multi-Tenant Fleet Allocation',
    prompt: 'Audit current fleet allocation across Tashkent and regional hubs, checking vehicle status and active driver pairings.',
  },
  {
    tag: '[SPANNER LEDGER]',
    displayText: 'Verify Double-Entry Ledger Invariants',
    prompt: 'Explain the Cloud Spanner transactional double-entry ledger invariants for supplier-to-retailer balance settlement.',
  },
  {
    tag: '[OR-TOOLS CVRP]',
    displayText: 'Simulate Tashkent-Samarkand Dispatch',
    prompt: 'Simulate an automated dispatch wave from Tashkent to Samarkand using Google OR-Tools CVRP optimizer with capacity and time-window constraints.',
  },
  {
    tag: '[DVIR INSPECTION]',
    displayText: 'Inspect Driver DVIR Pre-Trip Workflow',
    prompt: 'Walk through the DVIR vehicle pre-trip inspection workflow and how critical defects block dispatch ignition.',
  },
];

function AssistantWelcome({ starters }: { starters: typeof STARTERS }) {
  const processMessage = useThread((s) => s.processMessage);
  const isRunning = useThread((s) => s.isRunning);

  return (
    <div className="flex flex-col items-center justify-center max-w-3xl w-full px-4 text-center my-auto py-6">
      {/* Tactical Badge */}
      <div className="inline-flex items-center gap-2 px-3 py-1 mb-4 border border-[#CEFF00]/30 bg-[#CEFF00]/10 text-[#CEFF00] font-mono text-[11px] uppercase tracking-wider">
        <span className="w-1.5 h-1.5 rounded-full bg-[#CEFF00] animate-pulse" />
        Pegasus Autonomous Operations AI
      </div>

      <h1 className="text-2xl sm:text-3xl font-mono font-bold tracking-tight text-white mb-3">
        ECOSYSTEM INTELLIGENCE
      </h1>

      <p className="text-xs sm:text-sm font-mono text-white/60 max-w-xl mb-8 leading-relaxed">
        Real-time multi-tenant fleet telemetry, Spanner transactional ledger verification, and Google OR-Tools CVRP dispatch optimization.
      </p>

      {/* 2x2 Starter Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 w-full text-left">
        {starters.map((starter, idx) => (
          <button
            key={idx}
            type="button"
            disabled={isRunning}
            onClick={() => processMessage({ role: 'user', content: starter.prompt })}
            className="group relative flex flex-col justify-between p-4 bg-[#121216] hover:bg-[#181820] border border-white/10 hover:border-[#CEFF00]/50 transition-all duration-200 text-left cursor-pointer focus:outline-none focus:border-[#CEFF00]"
          >
            <div>
              <div className="flex items-center justify-between gap-2 mb-2">
                <span className="text-[10px] font-mono font-bold tracking-wider text-[#CEFF00] uppercase">
                  {starter.tag}
                </span>
                <span className="text-white/30 group-hover:text-[#CEFF00] group-hover:translate-x-0.5 transition-all text-xs font-mono">
                  →
                </span>
              </div>
              <h3 className="text-xs sm:text-sm font-mono font-bold text-white mb-1.5 group-hover:text-white">
                {starter.displayText}
              </h3>
              <p className="text-[11px] font-mono text-white/50 line-clamp-2 leading-relaxed">
                {starter.prompt}
              </p>
            </div>
          </button>
        ))}
      </div>
    </div>
  );
}

export default function AssistantPage() {
  const mode = useSystemThemeMode();

  return (
    <div className="fixed inset-0 h-[100dvh] w-screen bg-[#09090B] flex flex-col overflow-hidden text-white z-10">
      {/* Top Header Bar */}
      <header className="h-14 border-b border-white/10 px-6 flex items-center justify-between bg-black/80 backdrop-blur-md z-20 shrink-0">
        <div className="flex items-center gap-3">
          <Link
            href="/"
            className="inline-flex items-center gap-1.5 text-xs font-mono text-white/70 hover:text-white transition-colors"
          >
            <ArrowLeft className="w-4 h-4" />
            <span>BACK TO ECOSYSTEM</span>
          </Link>
          <span className="text-white/20">|</span>
          <span className="text-xs font-mono font-bold tracking-wider text-[#CEFF00] uppercase">
            PEGASUS AI ASSISTANT (OPENUI)
          </span>
        </div>
        <div className="flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-[#CEFF00] animate-pulse" />
          <span className="text-[11px] font-mono text-white/50">GENUI ACTIVE</span>
        </div>
      </header>

      {/* Main Agent Interface */}
      <main className="flex-1 w-full h-[calc(100dvh-3.5rem)] relative overflow-hidden bg-[#09090B]">
        <AgentInterface
          llm={llm}
          componentLibrary={openuiLibrary}
          agentName="Pegasus AI Assistant"
          theme={{ mode: 'dark' }}
          starters={STARTERS}
          starterVariant="long"
          scrollVariant="always"
        >
          <AgentInterface.Welcome>
            <AssistantWelcome starters={STARTERS} />
          </AgentInterface.Welcome>
        </AgentInterface>
      </main>
    </div>
  );
}
