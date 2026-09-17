'use client';

import React, { useState, useRef, useEffect } from 'react';
import { usePathname } from 'next/navigation';
import {
  ArrowUp,
  Maximize2,
  Minimize2,
  X,
  RotateCcw,
  Search,
  Sparkles,
  Square,
  Copy,
  Check,
} from 'lucide-react';
import { useLanguage } from '../context/LanguageContext';

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: Date;
}

const STARTERS = [
  {
    tag: 'FLEET TELEMETRY',
    title: 'Audit Multi-Tenant Fleet Allocation',
    prompt: 'Audit current fleet allocation across Tashkent and regional hubs, checking vehicle status and active driver pairings.',
  },
  {
    tag: 'SPANNER LEDGER',
    title: 'Verify Double-Entry Ledger Invariants',
    prompt: 'Explain the Cloud Spanner transactional double-entry ledger invariants for supplier-to-retailer balance settlement.',
  },
  {
    tag: 'OR-TOOLS CVRP',
    title: 'Simulate Regional Route Dispatch',
    prompt: 'Simulate an automated dispatch wave using Google OR-Tools CVRP optimizer with capacity and time-window constraints.',
  },
  {
    tag: 'DVIR INSPECTION',
    title: 'Inspect Driver DVIR Pre-Trip Workflow',
    prompt: 'Walk through the DVIR vehicle pre-trip inspection workflow and how critical defects block dispatch ignition.',
  },
];

function CodeBlock({ code, language }: { code: string; language?: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="relative my-3 rounded-lg overflow-hidden border border-zinc-800 bg-black/90 font-mono text-xs">
      <div className="flex items-center justify-between px-3 py-1.5 bg-zinc-900/90 border-b border-zinc-800 text-[11px] text-zinc-400">
        <span>{language || 'code'}</span>
        <button
          onClick={handleCopy}
          className="flex items-center gap-1 hover:text-white transition-colors cursor-pointer"
          title="Copy code"
        >
          {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
          <span>{copied ? 'Copied' : 'Copy'}</span>
        </button>
      </div>
      <pre className="p-3.5 overflow-x-auto text-zinc-200 leading-relaxed">
        <code>{code}</code>
      </pre>
    </div>
  );
}

function FormattedMessage({ content, isStreaming }: { content: string; isStreaming?: boolean }) {
  // Simple, high-speed custom parser for headers, code blocks, bold, lists, and inline code
  const parts = content.split(/(```[\s\S]*?```)/g);

  return (
    <div className="text-sm leading-relaxed space-y-2 text-zinc-200">
      {parts.map((part, i) => {
        if (part.startsWith('```') && part.endsWith('```')) {
          const lines = part.slice(3, -3).trim().split('\n');
          const lang = lines[0]?.match(/^[a-zA-Z0-9_-]+$/) ? lines[0] : '';
          const code = lang ? lines.slice(1).join('\n') : lines.join('\n');
          return <CodeBlock key={i} code={code} language={lang} />;
        }

        // Regular text formatting
        const paragraphs = part.split('\n\n');
        return (
          <div key={i} className="space-y-2">
            {paragraphs.map((para, pIdx) => {
              if (!para.trim()) return null;

              // H3 Header
              if (para.startsWith('### ')) {
                return (
                  <h4 key={pIdx} className="text-sm font-semibold text-white mt-3 mb-1 tracking-tight">
                    {para.slice(4)}
                  </h4>
                );
              }

              // Bullet list
              if (para.startsWith('- ') || para.startsWith('* ') || para.includes('\n- ')) {
                const items = para.split(/\n[-*]\s+/).filter(Boolean);
                return (
                  <ul key={pIdx} className="space-y-1 my-1 pl-4 list-disc text-zinc-300">
                    {items.map((item, itemIdx) => (
                      <li key={itemIdx} className="leading-snug">
                        {renderInlineFormatting(item)}
                      </li>
                    ))}
                  </ul>
                );
              }

              // Numbered list
              if (/^\d+\.\s/.test(para)) {
                const items = para.split(/\n(?=\d+\.\s)/).filter(Boolean);
                return (
                  <ol key={pIdx} className="space-y-1.5 my-1 pl-5 list-decimal text-zinc-300">
                    {items.map((item, itemIdx) => {
                      const text = item.replace(/^\d+\.\s+/, '');
                      return (
                        <li key={itemIdx} className="leading-snug">
                          {renderInlineFormatting(text)}
                        </li>
                      );
                    })}
                  </ol>
                );
              }

              return (
                <p key={pIdx} className="text-zinc-300 leading-relaxed">
                  {renderInlineFormatting(para)}
                </p>
              );
            })}
          </div>
        );
      })}

      {isStreaming && (
        <span className="inline-block w-2 h-4 ml-0.5 bg-white align-middle animate-pulse" />
      )}
    </div>
  );
}

function renderInlineFormatting(text: string) {
  // Parse inline code and bold text
  const tokens = text.split(/(`[^`]+`|\*\*[^*]+\*\*)/g);

  return tokens.map((token, idx) => {
    if (token.startsWith('`') && token.endsWith('`')) {
      return (
        <code
          key={idx}
          className="px-1.5 py-0.5 mx-0.5 rounded bg-zinc-800 text-zinc-100 font-mono text-xs border border-zinc-700/60"
        >
          {token.slice(1, -1)}
        </code>
      );
    }
    if (token.startsWith('**') && token.endsWith('**')) {
      return (
        <strong key={idx} className="font-semibold text-white">
          {token.slice(2, -2)}
        </strong>
      );
    }
    return token;
  });
}

export default function SiteAssistant() {
  const pathname = usePathname();
  const { language } = useLanguage();

  const [open, setOpen] = useState(false);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [deepSearch, setDeepSearch] = useState(false);
  const [think, setThink] = useState(false);

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const abortControllerRef = useRef<AbortController | null>(null);

  // Auto-scroll on new messages or stream chunks
  useEffect(() => {
    if (open) {
      messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  }, [messages, open]);

  // Focus textarea when opened
  useEffect(() => {
    if (open) {
      setTimeout(() => {
        textareaRef.current?.focus();
      }, 100);
    }
  }, [open]);

  // Global keyboard shortcuts (Cmd+K / Ctrl+K and Esc)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        setOpen((prev) => !prev);
      } else if (e.key === 'Escape' && open) {
        if (isFullscreen) {
          setIsFullscreen(false);
        } else {
          setOpen(false);
        }
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [open, isFullscreen]);

  // Hide assistant on admin route
  if (pathname?.startsWith('/admin')) {
    return null;
  }

  const handleSend = async (customPrompt?: string) => {
    const promptText = (customPrompt ?? input).trim();
    if (!promptText || isLoading) return;

    // Reset input height
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto';
    }

    const userMsg: Message = {
      id: crypto.randomUUID(),
      role: 'user',
      content: promptText,
      timestamp: new Date(),
    };

    const assistantMsgId = crypto.randomUUID();
    const assistantMsgPlaceholder: Message = {
      id: assistantMsgId,
      role: 'assistant',
      content: '',
      timestamp: new Date(),
    };

    setMessages((prev) => [...prev, userMsg, assistantMsgPlaceholder]);
    setInput('');
    setIsLoading(true);

    const controller = new AbortController();
    abortControllerRef.current = controller;

    try {
      const historyForApi = [...messages, userMsg].map((m) => ({
        role: m.role,
        content: m.content,
      }));

      const res = await fetch('/api/assistant', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          messages: historyForApi,
          stream: true,
          language,
          deepSearch,
          think,
        }),
        signal: controller.signal,
      });

      if (!res.ok || !res.body) {
        throw new Error('Failed to reach assistant');
      }

      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      let accumulated = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        const chunk = decoder.decode(value, { stream: true });
        accumulated += chunk;

        setMessages((prev) =>
          prev.map((m) => (m.id === assistantMsgId ? { ...m, content: accumulated } : m))
        );
      }
    } catch (err: unknown) {
      if ((err as Error)?.name !== 'AbortError') {
        setMessages((prev) =>
          prev.map((m) =>
            m.id === assistantMsgId
              ? {
                  ...m,
                  content:
                    language === 'ru'
                      ? 'Не удалось установить соединение. Попробуйте повторить запрос.'
                      : 'Connection interrupted. Please try asking again.',
                }
              : m
          )
        );
      }
    } finally {
      setIsLoading(false);
      abortControllerRef.current = null;
    }
  };

  const handleStop = () => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      setIsLoading(false);
    }
  };

  const handleReset = () => {
    handleStop();
    setMessages([]);
    setInput('');
  };

  const handleTextareaInput = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setInput(e.target.value);
    e.target.style.height = 'auto';
    e.target.style.height = `${Math.min(e.target.scrollHeight, 140)}px`;
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  // ──────────────────────────────────────────────────────────────────────────
  // Floating Launcher (Closed State)
  // ──────────────────────────────────────────────────────────────────────────
  if (!open) {
    return (
      <aside aria-label="Pegasus AI Assistant" className="fixed bottom-6 right-6 z-[10004]">
        <button
          type="button"
          onClick={() => setOpen(true)}
          className="relative flex items-center justify-center w-14 h-14 rounded-full bg-[#09090b] border border-zinc-700 hover:border-white text-white shadow-2xl shadow-black/80 hover:scale-105 transition-all duration-200 cursor-pointer group focus:outline-none"
          title="Ask Grok (⌘K)"
          aria-label="Open Pegasus AI Assistant"
        >
          {/* Subtle perimeter glow */}
          <div className="absolute inset-0 rounded-full bg-white/5 opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none" />

          {/* Center Pegasus Logo */}
          <img
            src="/pegasus.jpg"
            alt="Pegasus AI"
            className="w-8 h-8 object-contain rounded-full select-none"
          />

          {/* Shortcut Command Badge */}
          <span className="absolute -top-1 -right-1 px-1.5 py-0.5 bg-white text-black font-mono text-[9px] font-bold tracking-tight rounded border border-white shadow-md">
            ⌘K
          </span>
        </button>
      </aside>
    );
  }

  // ──────────────────────────────────────────────────────────────────────────
  // Chat Interface (Corner Dock or Fullscreen Workspace)
  // ──────────────────────────────────────────────────────────────────────────
  const containerClasses = isFullscreen
    ? 'fixed inset-0 z-[10005] bg-black/85 backdrop-blur-xl flex flex-col items-center justify-center p-2 sm:p-6 lg:p-8 animate-in fade-in duration-200'
    : 'fixed bottom-6 right-6 z-[10004] w-[420px] sm:w-[440px] h-[600px] max-w-[calc(100vw-1.5rem)] max-h-[calc(100dvh-4.5rem)] bg-[#09090b] border border-zinc-800 rounded-2xl flex flex-col overflow-hidden shadow-2xl shadow-black/90 animate-in fade-in zoom-in-95 duration-200';

  const innerCardClasses = isFullscreen
    ? 'w-full max-w-4xl h-full max-h-[92vh] bg-[#09090b] border border-zinc-800 rounded-2xl flex flex-col overflow-hidden shadow-2xl'
    : 'flex flex-col w-full h-full';

  return (
    <div className={containerClasses}>
      <div className={innerCardClasses}>
        
        {/* ── Grok Top Navigation Bar ── */}
        <header className="h-14 px-4 border-b border-zinc-800 bg-[#0c0c0e] flex items-center justify-between shrink-0 select-none">
          {/* Left Brand info */}
          <div className="flex items-center gap-2.5">
            <div className="w-6 h-6 rounded border border-white/20 overflow-hidden bg-black flex items-center justify-center">
              <img src="/pegasus.jpg" alt="Pegasus" className="w-full h-full object-contain" />
            </div>
            <div className="flex items-center gap-2 font-mono">
              <span className="text-sm font-semibold text-white tracking-tight">Grok</span>
              <span className="px-2 py-0.5 text-[10px] rounded-full bg-zinc-800 border border-zinc-700 text-zinc-300">
                grok-3-mini
              </span>
            </div>
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse ml-1" title="Online" />
          </div>

          {/* Right Action buttons */}
          <div className="flex items-center gap-1">
            <button
              type="button"
              onClick={handleReset}
              className="p-1.5 text-zinc-400 hover:text-white hover:bg-zinc-800/60 rounded-md transition-colors cursor-pointer"
              title="New Chat"
              aria-label="New Chat"
            >
              <RotateCcw className="w-4 h-4" />
            </button>

            <button
              type="button"
              onClick={() => setIsFullscreen((prev) => !prev)}
              className="p-1.5 text-zinc-400 hover:text-white hover:bg-zinc-800/60 rounded-md transition-colors cursor-pointer"
              title={isFullscreen ? 'Collapse to corner' : 'Expand to full screen'}
              aria-label="Toggle Fullscreen"
            >
              {isFullscreen ? <Minimize2 className="w-4 h-4" /> : <Maximize2 className="w-4 h-4" />}
            </button>

            <button
              type="button"
              onClick={() => setOpen(false)}
              className="p-1.5 text-zinc-400 hover:text-white hover:bg-zinc-800/60 rounded-md transition-colors cursor-pointer"
              title="Close (Esc)"
              aria-label="Close Assistant"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
        </header>

        {/* ── Scrollable Chat Messages Feed ── */}
        <div className="flex-1 overflow-y-auto px-4 py-4 space-y-4 scrollbar-thin scrollbar-thumb-zinc-800">
          {messages.length === 0 ? (
            /* Grok Welcome Empty State */
            <div className={`flex flex-col items-center justify-center text-center my-auto ${isFullscreen ? 'py-12 max-w-2xl mx-auto' : 'py-8'}`}>
              <div className="w-12 h-12 rounded-xl bg-zinc-900 border border-zinc-800 flex items-center justify-center p-2 mb-3 shadow-lg">
                <img src="/pegasus.jpg" alt="Pegasus Grok" className="w-full h-full object-contain rounded-lg" />
              </div>
              
              <h3 className="text-base sm:text-lg font-semibold text-white tracking-tight mb-1">
                {language === 'ru' ? 'Что вы хотите исследовать в Pegasus?' : 'What would you like to explore in Pegasus?'}
              </h3>
              <p className="text-xs text-zinc-400 max-w-sm mb-6">
                {language === 'ru'
                  ? 'Автономная логистическая разведка: телеметрия флота, Spanner ledger и оптимизация CVRP.'
                  : 'Autonomous logistics intelligence: fleet telematics, Spanner ledger invariants, and CVRP dispatch.'}
              </p>

              {/* 2x2 Starter Cards */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-2.5 w-full text-left">
                {STARTERS.map((starter, idx) => (
                  <button
                    key={idx}
                    type="button"
                    onClick={() => handleSend(starter.prompt)}
                    className="p-3 rounded-xl bg-[#131317] hover:bg-[#191920] border border-zinc-800 hover:border-zinc-700 transition-all text-left group cursor-pointer"
                  >
                    <div className="flex items-center justify-between gap-1 mb-1">
                      <span className="text-[10px] font-mono font-bold text-zinc-400 group-hover:text-white uppercase">
                        {starter.tag}
                      </span>
                      <span className="text-zinc-500 group-hover:text-white text-xs transition-transform group-hover:translate-x-0.5">
                        →
                      </span>
                    </div>
                    <div className="text-xs font-medium text-zinc-200 group-hover:text-white line-clamp-1">
                      {starter.title}
                    </div>
                  </button>
                ))}
              </div>
            </div>
          ) : (
            /* Message Thread */
            <div className={isFullscreen ? 'max-w-3xl w-full mx-auto space-y-4' : 'space-y-4'}>
              {messages.map((msg) => {
                const isUser = msg.role === 'user';
                return (
                  <div
                    key={msg.id}
                    className={`flex flex-col ${isUser ? 'items-end' : 'items-start'} gap-1`}
                  >
                    {!isUser && (
                      <div className="flex items-center gap-1.5 text-[11px] font-mono text-zinc-400 mb-0.5 pl-1">
                        <img src="/pegasus.jpg" alt="" className="w-3.5 h-3.5 rounded-full object-contain" />
                        <span className="font-semibold text-zinc-300">Grok</span>
                      </div>
                    )}

                    {isUser ? (
                      <div className="bg-[#1e1e24] border border-zinc-700/80 text-zinc-100 rounded-2xl px-4 py-2.5 text-sm max-w-[85%] break-words whitespace-pre-wrap leading-relaxed shadow-sm">
                        {msg.content}
                      </div>
                    ) : (
                      <div className="bg-transparent text-zinc-200 px-1 py-1 text-sm w-full leading-relaxed">
                        {msg.content ? (
                          <FormattedMessage
                            content={msg.content}
                            isStreaming={isLoading && messages[messages.length - 1]?.id === msg.id}
                          />
                        ) : (
                          <div className="flex items-center gap-1.5 text-zinc-500 text-xs py-1">
                            <span className="inline-block w-1.5 h-1.5 rounded-full bg-zinc-400 animate-pulse" />
                            <span>Thinking...</span>
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                );
              })}
              <div ref={messagesEndRef} />
            </div>
          )}
        </div>

        {/* ── Grok Floating Input Composer ── */}
        <div className={`p-3 shrink-0 ${isFullscreen ? 'max-w-3xl w-full mx-auto' : ''}`}>
          <div className="bg-[#121216] border border-zinc-800 focus-within:border-zinc-600 rounded-2xl p-2.5 shadow-lg transition-colors">
            
            {/* Auto-growing Textarea */}
            <textarea
              ref={textareaRef}
              rows={1}
              value={input}
              onChange={handleTextareaInput}
              onKeyDown={handleKeyDown}
              placeholder={
                language === 'ru'
                  ? 'Спросите Grok о логистике Pegasus...'
                  : 'Ask Grok anything about Pegasus...'
              }
              className="w-full bg-transparent text-white placeholder-zinc-500 text-sm resize-none focus:outline-none max-h-36 px-1.5 py-1 leading-relaxed"
            />

            {/* Bottom Toolbar Row */}
            <div className="flex items-center justify-between pt-2 border-t border-zinc-800/80 mt-1">
              {/* Left Action Toggles */}
              <div className="flex items-center gap-1.5">
                <button
                  type="button"
                  onClick={() => setDeepSearch((prev) => !prev)}
                  className={`inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-[11px] font-mono transition-colors cursor-pointer border ${
                    deepSearch
                      ? 'bg-white text-black border-white font-medium'
                      : 'bg-zinc-900 text-zinc-400 border-zinc-800 hover:border-zinc-700 hover:text-zinc-200'
                  }`}
                  title="DeepSearch Mode"
                >
                  <Search className="w-3 h-3" />
                  <span>DeepSearch</span>
                </button>

                <button
                  type="button"
                  onClick={() => setThink((prev) => !prev)}
                  className={`inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-[11px] font-mono transition-colors cursor-pointer border ${
                    think
                      ? 'bg-white text-black border-white font-medium'
                      : 'bg-zinc-900 text-zinc-400 border-zinc-800 hover:border-zinc-700 hover:text-zinc-200'
                  }`}
                  title="Think Reasoning Mode"
                >
                  <Sparkles className="w-3 h-3" />
                  <span>Think</span>
                </button>
              </div>

              {/* Right Send / Stop Button */}
              <div>
                {isLoading ? (
                  <button
                    type="button"
                    onClick={handleStop}
                    className="w-8 h-8 rounded-full bg-zinc-800 hover:bg-zinc-700 text-white flex items-center justify-center transition-colors cursor-pointer"
                    title="Stop generation"
                    aria-label="Stop generation"
                  >
                    <Square className="w-3.5 h-3.5 fill-current" />
                  </button>
                ) : (
                  <button
                    type="button"
                    onClick={() => handleSend()}
                    disabled={!input.trim()}
                    className={`w-8 h-8 rounded-full flex items-center justify-center transition-all ${
                      input.trim()
                        ? 'bg-white text-black hover:bg-zinc-200 cursor-pointer shadow-md'
                        : 'bg-zinc-800 text-zinc-500 cursor-not-allowed'
                    }`}
                    title="Send message (Enter)"
                    aria-label="Send message"
                  >
                    <ArrowUp className="w-4 h-4 font-bold" />
                  </button>
                )}
              </div>
            </div>

          </div>
        </div>

      </div>
    </div>
  );
}
