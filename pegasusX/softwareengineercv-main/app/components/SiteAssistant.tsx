'use client';

import { useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { ArrowUpRight, X } from 'lucide-react';

import '@openuidev/react-ui/components.css';
import '@openuidev/react-ui/styles/index.css';

import { AgentInterface } from '@openuidev/react-ui';
import {
  fetchLLM,
  vercelAIAdapter,
  vercelAIMessageFormat,
} from '@openuidev/react-headless';
import { openuiLibrary } from '@openuidev/react-ui/genui-lib';

const HIDDEN_PREFIXES = ['/admin', '/resume', '/platform', '/roles', '/technology', '/solutions'];

const llm = fetchLLM({
  url: '/api/chat',
  streamAdapter: vercelAIAdapter(),
  messageFormat: vercelAIMessageFormat,
});

export default function SiteAssistant() {
  const pathname = usePathname();
  const containerRef = useRef<HTMLDivElement>(null);
  const launcherRef = useRef<HTMLButtonElement>(null);

  const [open, setOpen] = useState(false);

  // Close assistant on route changes
  useEffect(() => {
    setOpen(false);
  }, [pathname]);

  // Keyboard shortcut: Cmd+K / Ctrl+K to toggle assistant
  useEffect(() => {
    function handleGlobalKeyDown(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        setOpen((prev) => !prev);
      }
    }
    window.addEventListener('keydown', handleGlobalKeyDown);
    return () => window.removeEventListener('keydown', handleGlobalKeyDown);
  }, []);

  // Click outside to close assistant
  useEffect(() => {
    if (!open) return;

    function handleClickOutside(e: MouseEvent | TouchEvent) {
      const targetNode = e.target as Node | null;
      if (!targetNode) return;

      const isOutsideContainer = containerRef.current && !containerRef.current.contains(targetNode);
      const isOutsideLauncher = launcherRef.current && !launcherRef.current.contains(targetNode);

      if (isOutsideContainer && isOutsideLauncher) {
        setOpen(false);
      }
    }

    document.addEventListener('mousedown', handleClickOutside);
    document.addEventListener('touchstart', handleClickOutside);

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('touchstart', handleClickOutside);
    };
  }, [open]);

  // Escape key to close
  useEffect(() => {
    if (!open) return;

    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        setOpen(false);
      }
    }

    window.addEventListener('keydown', handleKeyDown);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [open]);

  if (HIDDEN_PREFIXES.some((prefix) => pathname?.startsWith(prefix))) {
    return null;
  }

  return (
    <aside aria-label="Pegasus AI Assistant" className="fixed bottom-6 right-6 z-[10004]">
      {open ? (
        <div
          ref={containerRef}
          className="fixed bottom-6 right-6 z-[10005] w-[95vw] sm:w-[480px] md:w-[560px] h-[85vh] max-h-[740px] bg-[#09090B] border border-white/20 rounded-2xl shadow-2xl flex flex-col overflow-hidden animate-in fade-in zoom-in-95 duration-200"
          role="dialog"
          aria-modal="true"
          aria-label="Pegasus AI Assistant"
        >
          {/* Header Bar */}
          <header className="h-14 border-b border-white/10 px-4 flex items-center justify-between bg-black/90 backdrop-blur-md shrink-0">
            <div className="flex items-center gap-2.5">
              <div className="w-8 h-8 rounded-lg bg-black border border-[#CEFF00]/40 flex items-center justify-center p-1 shadow-[0_0_10px_rgba(206,255,0,0.15)]">
                <img src="/icons/ai-orbit.svg" alt="" width={20} height={20} />
              </div>
              <div>
                <div className="flex items-center gap-2">
                  <span className="text-xs font-mono font-bold tracking-wider text-white uppercase">
                    PEGASUS AI ASSISTANT
                  </span>
                  <span className="px-1.5 py-0.5 rounded bg-[#CEFF00]/15 text-[#CEFF00] font-mono text-[9px] font-bold">
                    GENUI
                  </span>
                </div>
                <p className="text-[10px] font-mono text-white/50 leading-none mt-0.5">
                  Generative UI • xAI / OpenAI Intelligence
                </p>
              </div>
            </div>

            <div className="flex items-center gap-2">
              <Link
                href="/assistant"
                className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md border border-white/15 text-[11px] font-mono text-white/70 hover:text-white hover:border-[#CEFF00] hover:text-[#CEFF00] transition-colors"
                title="Open in Full Workspace"
              >
                <span>Full Workspace</span>
                <ArrowUpRight className="w-3 h-3" />
              </Link>
              <button
                type="button"
                onClick={() => setOpen(false)}
                className="w-8 h-8 rounded-lg hover:bg-white/10 text-white/70 hover:text-white flex items-center justify-center transition-colors"
                aria-label="Close Assistant"
              >
                <X className="w-4 h-4" />
              </button>
            </div>
          </header>

          {/* Main OpenUI Agent Interface */}
          <div className="flex-1 w-full h-full relative overflow-hidden bg-[#09090B]">
            <AgentInterface
              llm={llm}
              componentLibrary={openuiLibrary}
              agentName="Pegasus AI Assistant"
              theme={{ mode: 'dark' }}
            />
          </div>
        </div>
      ) : null}

      {/* Single Glowing AI Assistant Launcher (Matches Reference Design) */}
      {!open ? (
        <button
          ref={launcherRef}
          type="button"
          className="glowing-squircle-launcher group focus:outline-none"
          aria-expanded={open}
          aria-label="Open Pegasus AI Assistant"
          onClick={() => setOpen(true)}
        >
          <img
            src="/icons/ai-orbit.svg"
            alt="AI Assistant"
            width={32}
            height={32}
            className="transition-transform duration-200 group-hover:scale-110 drop-shadow-[0_0_8px_rgba(255,255,255,0.5)]"
          />
          <span className="sr-only">Open Pegasus AI Assistant</span>
        </button>
      ) : null}
    </aside>
  );
}
