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
 <div className="flex flex-col items-center justify-center max-w-3xl w-full px-4 text-center my-auto py-2">
 {/* Pegasus Logo Icon */}
 <div className="w-12 h-12 mb-3 rounded-2xl border border-white/20 bg-black flex items-center justify-center p-2 select-none shadow-[0_0_24px_rgba(255,255,255,0.08)]">
 <img src="/pegasus.jpg" alt="Pegasus Logo" className="w-full h-full object-contain rounded-xl" />
 </div>

 {/* Tactical Badge */}
 <div className="inline-flex items-center gap-2 px-3 py-0.5 mb-2.5 rounded-full border border-white/20 bg-white/5 text-white font-mono text-[10px] uppercase tracking-wider">
 <span className="w-1.5 h-1.5 rounded-full bg-white " />
 Pegasus Autonomous Operations AI
 </div>

 <h1 className="text-xl sm:text-2xl font-mono font-bold tracking-tight text-white mb-2">
 ECOSYSTEM INTELLIGENCE
 </h1>

 <p className="text-xs font-mono text-white/60 max-w-xl mb-4 leading-relaxed">
 Real-time multi-tenant fleet telemetry, Spanner transactional ledger verification, and Google OR-Tools CVRP dispatch optimization.
 </p>

 {/* 2x2 Starter Grid */}
 <div className="grid grid-cols-1 sm:grid-cols-2 gap-2.5 w-full text-left">
 {starters.map((starter, idx) => (
 <button
 key={idx}
 type="button"
 disabled={isRunning}
 onClick={() => processMessage({ role: 'user', content: starter.prompt })}
 className="group relative flex flex-col justify-between p-4 bg-black hover:bg-white/[0.04] border border-white/15 hover:border-white/40 rounded-2xl transition-all duration-200 text-left cursor-pointer focus:outline-none focus:border-white shadow-sm"
 >
 <div>
 <div className="flex items-center justify-between gap-2 mb-2">
 <span className="px-2 py-0.5 rounded-md border border-white/20 bg-white/5 text-[10px] font-mono font-bold tracking-wider text-white uppercase">
 {starter.tag}
 </span>
 <span className="w-6 h-6 rounded-full border border-white/10 flex items-center justify-center text-white/40 group-hover:text-white group-hover:border-white/40 group-hover:translate-x-0.5 transition-all text-xs font-mono">
 →
 </span>
 </div>
 <h3 className="text-xs sm:text-sm font-mono font-bold text-white mb-1.5 group-hover:text-white">
 {starter.displayText}
 </h3>
 <p className="text-[11px] font-mono text-white/70 line-clamp-2 leading-relaxed">
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
 <div className="fixed inset-0 h-[100dvh] w-screen bg-black flex flex-col overflow-hidden text-white z-10 keep-dark" data-keep-dark data-keep-white>
 {/* Top Header Bar */}
 <header className="h-[calc(3.5rem+env(safe-area-inset-top,0px))] pt-[env(safe-area-inset-top,0px)] border-b border-white/10 px-4 sm:px-6 flex items-center justify-between bg-black z-20 shrink-0">
 <div className="flex items-center gap-3">
 <Link
 href="/"
 className="inline-flex items-center gap-1.5 text-xs font-mono text-white/70 hover:text-white px-3 py-1 rounded-full border border-white/15 hover:border-white/40 bg-black transition-colors"
 >
 <ArrowLeft className="w-4 h-4" />
 <span>BACK TO ECOSYSTEM</span>
 </Link>
 <span className="text-white/20">|</span>
 <div className="flex items-center gap-2">
 <img src="/pegasus.jpg" alt="Pegasus" className="w-5 h-5 object-contain rounded-md" />
 <span className="text-xs font-mono font-bold tracking-wider text-white uppercase">
 PEGASUS AI ASSISTANT
 </span>
 </div>
 </div>
 <div className="flex items-center gap-2">
 <span className="w-2 h-2 rounded-full bg-white " />
 <span className="text-[11px] font-mono text-white/70">GENUI ACTIVE</span>
 </div>
 </header>

 {/* Main Agent Interface */}
 <main className="flex-1 w-full relative overflow-hidden bg-black pb-[env(safe-area-inset-bottom,0px)]">
 <AgentInterface
 llm={llm}
 componentLibrary={openuiLibrary}
 agentName="Pegasus AI Assistant"
 logoUrl="/pegasus.jpg"
 theme={{ mode: 'dark' }}
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
