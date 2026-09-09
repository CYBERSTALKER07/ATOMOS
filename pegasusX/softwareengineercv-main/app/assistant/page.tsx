'use client';

import '@openuidev/react-ui/components.css';
import '@openuidev/react-ui/styles/index.css';

import {
  AgentInterface,
  useSystemThemeMode,
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

export default function AssistantPage() {
  const mode = useSystemThemeMode();

  return (
    <div className="min-h-screen w-screen bg-[#09090B] flex flex-col overflow-hidden text-white">
      {/* Top Header Bar */}
      <header className="h-14 border-b border-white/10 px-6 flex items-center justify-between bg-black/80 backdrop-blur-md z-20">
        <div className="flex items-center gap-3">
          <Link
            href="/"
            className="inline-flex items-center gap-1.5 text-xs font-mono text-white/70 hover:text-white transition-colors"
          >
            <ArrowLeft className="w-4 h-4" />
            <span>BACK TO ECOSYSTEM</span>
          </Link>
          <span className="text-white/20">|</span>
          <span className="text-xs font-mono font-bold tracking-wider text-[#E2FD52] uppercase">
            PEGASUS AI ASSISTANT (OPENUI)
          </span>
        </div>
        <div className="flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-[#E2FD52] animate-pulse" />
          <span className="text-[11px] font-mono text-white/50">GENUI ACTIVE</span>
        </div>
      </header>

      {/* Main Agent Interface */}
      <main className="flex-1 w-full h-[calc(100vh-3.5rem)] relative overflow-hidden bg-[#09090B]">
        <AgentInterface
          llm={llm}
          componentLibrary={openuiLibrary}
          agentName="Pegasus AI Assistant"
          theme={{ mode: 'dark' }}
        />
      </main>
    </div>
  );
}
