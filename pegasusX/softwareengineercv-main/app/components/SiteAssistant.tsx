'use client';

import { FormEvent, useEffect, useId, useRef, useState } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Maximize2, Copy, Check, RotateCcw, X } from 'lucide-react';
import { useLanguage } from '../context/LanguageContext';
import GooeyAgent from './visuals/GooeyAgent';

type ChatRole = 'user' | 'assistant';

type ChatMessage = {
 id: string;
 role: ChatRole;
 content: string;
};

type QuickAction = {
 id: string;
 badge: string;
 label: string;
 category: 'versus' | 'tech' | 'roles' | 'business' | 'action';
 prompt?: string;
 href?: string;
 dismiss?: boolean;
};

const QUICK_ACTIONS_RU: QuickAction[] = [
 {
 id: 'versus-giants',
 badge: 'VS',
 label: 'Почему Pegasus превосходит Amazon, o9 и Oracle?',
 category: 'versus',
 prompt: 'Сравните Pegasus с гигантами отрасли и legacy ERP (Amazon AWS Supply Chain, o9 Solutions, Oracle OTM, Google Cloud Twin, Blue Yonder). В чем наши ключевые преимущества?',
 },
 {
 id: 'tech-stack',
 badge: 'TECH',
 label: 'Backend на Go, Spanner и Transactional Outbox',
 category: 'tech',
 prompt: 'Расскажите о технической архитектуре Pegasus: микросервисы на Go, транзакции в Cloud Spanner, паттерн Transactional Outbox, Kafka, WebSockets и оффлайн-синхронизация.',
 },
 {
 id: 'role-parity',
 badge: 'ROLES',
 label: '6 ключевых ролей в единой экосистеме',
 category: 'roles',
 prompt: 'Какие 6 ролей объединены в Pegasus (Поставщик, Завод, Склад, Водитель, Розничный продавец, Служба доставки) и как устроена их единая сеть данных?',
 },
 {
 id: 'business-roi',
 badge: 'ROI',
 label: 'Операционная эффективность и бизнес-метрики',
 category: 'business',
 prompt: 'Какие измеримые экономические показатели и оптимизацию цепочки поставок обеспечивает Pegasus для директоров по логистике?',
 },
 {
 id: 'contact-team',
 badge: 'JOIN',
 label: 'Запросить демо / Связаться с командой',
 category: 'action',
 href: '/join',
 },
];

const WELCOME_RU =
 'Привет! Я ИИ-Ассистент Pegasus. Чем могу помочь? Узнайте о наших возможностях, ролях в сети поставок, диспетчеризации, отслеживании автопарка или сравните нас с альтернативами.';

const QUICK_ACTIONS: QuickAction[] = [
 {
 id: 'versus-giants',
 badge: 'VS',
 label: 'Why Pegasus vs Amazon, o9 & Oracle?',
 category: 'versus',
 prompt: 'Compare Pegasus with tech giants & legacy ERPs (Amazon AWS Supply Chain, o9 Solutions, Oracle OTM, Google Cloud Twin, Blue Yonder). Why choose Pegasus?',
 },
 {
 id: 'tech-stack',
 badge: 'TECH',
 label: 'Go Backend, Spanner & Outbox System',
 category: 'tech',
 prompt: 'Explain the technical architecture of Pegasus: Go microservices, Cloud Spanner transactions, Transactional Outbox pattern, Kafka, WebSockets, and offline mobile sync.',
 },
 {
 id: 'six-roles',
 badge: 'ROLES',
 label: '6 Ecosystem Roles & Capabilities',
 category: 'roles',
 prompt: 'What are the 6 ecosystem roles in Pegasus (Supplier, Warehouse, Retailer, Driver, Factory, Payload/Gate) and their key capabilities?',
 },
 {
 id: 'business-roi',
 badge: 'ROI',
 label: 'Business Benefits & Operational ROI',
 category: 'business',
 prompt: 'What are the core business outcomes, ROI, and workflow improvements Pegasus delivers for supply chain leadership?',
 },
 {
 id: 'demo-tour',
 badge: 'DEMO',
 label: 'Watch Platform Walkthrough',
 category: 'action',
 href: '/demo',
 },
 {
 id: 'contact-expert',
 badge: 'TALK',
 label: 'Talk to a Logistics Expert',
 category: 'action',
 href: '/contact',
 },
 {
 id: 'join-careers',
 badge: 'JOIN',
 label: 'Careers & Schedule Demo',
 category: 'action',
 href: '/join',
 },
 {
 id: 'dismiss',
 badge: 'HIDE',
 label: 'Dismiss prompt assistant',
 category: 'action',
 dismiss: true,
 },
];

const HIDDEN_PREFIXES = ['/admin', '/resume', '/assistant'];
const WELCOME =
 'Welcome to Pegasus. Ask anything about our logistics OS — compare us to alternatives, explore our 6 role capabilities, or learn how Pegasus handles dispatch, fleet tracking, and payments.';

function newId() {
 return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

function FormattedContent({ text }: { text: string }) {
 const parts = text.split(/(```[\s\S]*?```)/g);

 return (
 <div className="space-y-2 leading-relaxed">
 {parts.map((part, idx) => {
 if (part.startsWith('```') && part.endsWith('```')) {
 const lines = part.slice(3, -3).trim().split('\n');
 const lang = lines[0]?.match(/^[a-zA-Z0-9_-]+$/) ? lines[0] : '';
 const code = lang ? lines.slice(1).join('\n') : lines.join('\n');
 return (
 <div key={idx} className="my-2 border border-white/20 bg-black p-3 font-mono text-xs overflow-x-auto text-zinc-200">
 {lang && <div className="text-[10px] text-zinc-500 uppercase mb-1">{lang}</div>}
 <code>{code}</code>
 </div>
 );
 }

 const paragraphs = part.split('\n\n');
 return (
 <div key={idx} className="space-y-1.5">
 {paragraphs.map((para, pIdx) => {
 if (!para.trim()) return null;

 if (para.startsWith('### ')) {
 return (
 <h4 key={pIdx} className="font-mono font-bold text-white text-xs mt-2 mb-1 tracking-wider uppercase">
 {para.replace('### ', '')}
 </h4>
 );
 }

 if (para.startsWith('## ')) {
 return (
 <h3 key={pIdx} className="font-mono font-bold text-white text-sm mt-2 mb-1 tracking-wide uppercase">
 {para.replace('## ', '')}
 </h3>
 );
 }

 const formattedLine = para.split('\n').map((line, lIdx) => {
 const isBullet = line.trim().startsWith('- ') || line.trim().startsWith('* ');
 const cleanLine = isBullet ? line.trim().slice(2) : line;

 return (
 <div key={lIdx} className={isBullet ? 'flex items-start gap-1.5 pl-2' : ''}>
 {isBullet && <span className="text-white/60 font-mono select-none">•</span>}
 <span>{cleanLine}</span>
 </div>
 );
 });

 return <div key={pIdx}>{formattedLine}</div>;
 })}
 </div>
 );
 })}
 </div>
 );
}

export default function SiteAssistant() {
 const pathname = usePathname();
 const panelId = useId();
 const { t, language } = useLanguage();
 const listRef = useRef<HTMLDivElement>(null);
 const containerRef = useRef<HTMLDivElement>(null);
 const launcherRef = useRef<HTMLButtonElement>(null);
 const inputRef = useRef<HTMLInputElement>(null);

 const [open, setOpen] = useState(false);
 const [dismissed, setDismissed] = useState(false);
 const [input, setInput] = useState('');
 const [loading, setLoading] = useState(false);
 const [error, setError] = useState<string | null>(null);
 const [copiedId, setCopiedId] = useState<string | null>(null);

 const currentWelcome = language === 'ru' ? WELCOME_RU : WELCOME;
 const currentQuickActions = language === 'ru' ? QUICK_ACTIONS_RU : QUICK_ACTIONS;

 const [messages, setMessages] = useState<ChatMessage[]>([
 { id: 'welcome', role: 'assistant', content: currentWelcome },
 ]);

 useEffect(() => {
 setMessages((prev) => {
 if (prev.length === 1 && prev[0].id === 'welcome') {
 return [{ id: 'welcome', role: 'assistant', content: currentWelcome }];
 }
 return prev;
 });
 }, [language, currentWelcome]);

 // Close assistant on route changes
 useEffect(() => {
 setOpen(false);
 }, [pathname]);

 // Auto-scroll message list when new messages arrive
 useEffect(() => {
 if (!open) return;
 listRef.current?.scrollTo({ top: listRef.current.scrollHeight, behavior: 'smooth' });
 }, [messages, open, loading]);

 // Auto-focus input when chat opens
 useEffect(() => {
 if (open) {
 setTimeout(() => inputRef.current?.focus(), 100);
 }
 }, [open]);

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

 document.addEventListener('mousedown', handleClickOutside, true);
 document.addEventListener('touchstart', handleClickOutside, true);

 return () => {
 document.removeEventListener('mousedown', handleClickOutside, true);
 document.removeEventListener('touchstart', handleClickOutside, true);
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

 if (dismissed) return null;

 async function sendPrompt(text: string) {
 const trimmed = text.trim();
 if (!trimmed || loading) return;

 const userMsg: ChatMessage = { id: newId(), role: 'user', content: trimmed };
 const nextHistory = [...messages, userMsg].filter((m) => m.id !== 'welcome');
 setMessages((prev) => [...prev, userMsg]);
 setInput('');
 setLoading(true);
 setError(null);

 const assistantMsgId = newId();

 try {
 const res = await fetch('/api/assistant', {
 method: 'POST',
 headers: {
 'Content-Type': 'application/json',
 Accept: 'text/plain',
 },
 body: JSON.stringify({
 messages: nextHistory.map(({ role, content }) => ({ role, content })),
 language,
 }),
 });

 if (!res.ok) {
 throw new Error('Assistant unavailable');
 }

 if (res.body && res.headers.get('content-type')?.includes('text/plain')) {
 const reader = res.body.getReader();
 const decoder = new TextDecoder('utf-8');
 let accumulated = '';

 setMessages((prev) => [...prev, { id: assistantMsgId, role: 'assistant', content: '' }]);

 while (true) {
 const { done, value } = await reader.read();
 if (done) break;
 const chunk = decoder.decode(value, { stream: true });
 accumulated += chunk;
 setMessages((prev) =>
 prev.map((msg) => (msg.id === assistantMsgId ? { ...msg, content: accumulated } : msg))
 );
 }
 } else {
 const data = (await res.json()) as { reply?: string; error?: string };
 if (data.error || !data.reply) {
 throw new Error(data.error || 'Assistant unavailable');
 }
 setMessages((prev) => [...prev, { id: assistantMsgId, role: 'assistant', content: data.reply! }]);
 }
 } catch (err) {
 const message = err instanceof Error ? err.message : 'Something went wrong';
 setError(message);
 setMessages((prev) => [
 ...prev,
 {
 id: newId(),
 role: 'assistant',
 content:
 language === 'ru'
 ? `Не удалось получить ответ (${message}). Попробуйте еще раз или свяжитесь с нами.`
 : `I couldn’t answer that just now (${message}). Try again, or reach out to our team.`,
 },
 ]);
 } finally {
 setLoading(false);
 }
 }

 function onSubmit(e: FormEvent) {
 e.preventDefault();
 void sendPrompt(input);
 }

 function clearHistory() {
 setMessages([{ id: newId(), role: 'assistant', content: currentWelcome }]);
 setError(null);
 }

 function copyToClipboard(msgId: string, text: string) {
 void navigator.clipboard.writeText(text);
 setCopiedId(msgId);
 setTimeout(() => setCopiedId(null), 2000);
 }

 const handleExpandToSeparateWindow = () => {
 window.open('/assistant', '_blank');
 };

 return (
 <div className="site-assistant" data-open={open ? 'true' : 'false'}>
 {open ? (
 <aside
 id={panelId}
 ref={containerRef}
 className="site-assistant__panel site-assistant__panel--chat fixed bottom-[76px] right-4 sm:bottom-[88px] sm:right-6 z-[10004] outline-none"
 role="dialog"
 aria-label="Pegasus assistant"
 >
 <div className="site-assistant__chat-card rounded-none border border-white/20 bg-black text-white overflow-hidden flex flex-col w-[380px] sm:w-[420px] max-w-[calc(100vw-2rem)] h-[560px] max-h-[calc(100vh-6rem)] sm:max-h-[calc(100vh-7.5rem)]">
 {/* Header Bar */}
 <header className="site-assistant__chat-head flex items-center justify-between p-3.5 bg-black border-b border-white/10 shrink-0">
 <div className="flex items-center gap-2.5 min-w-0">
 <div className="w-7 h-7 border border-white/20 bg-black flex items-center justify-center shrink-0 p-1">
 <GooeyAgent color="#ffffff" size={20} />
 </div>
 <div className="min-w-0">
 <div className="flex items-center gap-1.5">
 <span className="text-xs font-mono font-bold tracking-wider text-white uppercase truncate">
 PEGASUS OS
 </span>
 <span className="text-[9px] font-mono font-semibold px-1 py-0.2 border border-white/20 text-white/70 uppercase">
 CORNER
 </span>
 </div>
 <p className="text-[10px] font-mono text-white/50 truncate">
 Autonomous Operations & Architecture AI
 </p>
 </div>
 </div>

 <div className="flex items-center gap-1.5 shrink-0">
 {/* Clear History */}
 <button
 type="button"
 onClick={clearHistory}
 title={language === 'ru' ? 'Очистить историю' : 'Clear History'}
 className="p-1.5 text-white/50 hover:text-white border border-transparent hover:border-white/20 transition-colors cursor-pointer"
 >
 <RotateCcw className="w-3.5 h-3.5" />
 </button>

 {/* Open in Separate Window Fullscreen */}
 <button
 type="button"
 className="site-assistant__toggle-fullscreen inline-flex items-center gap-1 px-2 py-1 text-[10px] font-mono font-bold text-white bg-black border border-white/30 hover:border-white hover:bg-white hover:text-black transition-all cursor-pointer uppercase tracking-wider rounded-none"
 title={language === 'ru' ? 'Открыть в отдельном окне на весь экран' : 'Open in separate window fullscreen'}
 onClick={handleExpandToSeparateWindow}
 >
 <Maximize2 className="w-3 h-3" />
 <span>{language === 'ru' ? 'ОКНО' : 'EXPAND'} ↗</span>
 </button>

 {/* Close Button */}
 <button
 type="button"
 className="site-assistant__close-btn text-white/60 hover:text-white text-lg leading-none p-1 cursor-pointer transition-colors"
 title={language === 'ru' ? 'Закрыть (Esc)' : 'Close (Esc)'}
 onClick={() => setOpen(false)}
 >
 ×
 </button>
 </div>
 </header>

 {/* Messages Stream */}
 <div ref={listRef} className="site-assistant__messages flex-1 overflow-y-auto p-3.5 space-y-3 bg-black" aria-live="polite">
 {messages.map((msg) => (
 <div
 key={msg.id}
 className={`site-assistant__msg ${
 msg.role === 'assistant'
 ? 'site-assistant__msg--assistant self-start max-w-[92%] bg-black border border-white/10 text-[#EDEDED] p-3 text-xs leading-relaxed rounded-none'
 : 'site-assistant__msg--user self-end max-w-[85%] bg-white text-black p-3 text-xs font-medium rounded-none '
 }`}
 >
 <div className="flex items-center justify-between gap-2 mb-1.5 pb-1 border-b border-white/5">
 <span className="text-[10px] font-mono font-bold uppercase tracking-wider opacity-60">
 {msg.role === 'assistant' ? 'PEGASUS INTELLIGENCE' : t('asst_you', 'YOU')}
 </span>

 {msg.role === 'assistant' && msg.content && (
 <button
 type="button"
 className="inline-flex items-center gap-1 text-[10px] font-mono text-white/50 hover:text-white transition-colors cursor-pointer"
 onClick={() => copyToClipboard(msg.id, msg.content)}
 >
 {copiedId === msg.id ? (
 <>
 <Check className="w-3 h-3 text-white" />
 <span>COPIED</span>
 </>
 ) : (
 <>
 <Copy className="w-3 h-3" />
 <span>COPY</span>
 </>
 )}
 </button>
 )}
 </div>

 <div className="site-assistant__msg-body">
 <FormattedContent text={msg.content} />
 </div>
 </div>
 ))}

 {loading && (
 <div className="site-assistant__msg site-assistant__msg--assistant self-start max-w-[92%] bg-black border border-white/10 text-white p-3 text-xs rounded-none">
 <div className="flex items-center gap-1.5 mb-1 opacity-60">
 <span className="text-[10px] font-mono font-bold uppercase tracking-wider">PEGASUS INTELLIGENCE</span>
 </div>
 <div className="flex items-center gap-2 text-xs font-mono text-white/70 py-1">
 <span className="w-1.5 h-1.5 rounded-none bg-white" />
 <span>{language === 'ru' ? 'Генерация ответа...' : 'Synthesizing telemetry...'}</span>
 </div>
 </div>
 )}
 </div>

 {error && <p className="text-[11px] font-mono text-zinc-400 px-3.5 py-1 bg-black">{error}</p>}

 {/* Action Pills */}
 <ul className="site-assistant__actions site-assistant__actions--inline flex flex-col gap-1.5 p-2.5 bg-black border-t border-white/10 max-h-[140px] overflow-y-auto">
 {currentQuickActions.map((action) => (
 <li key={action.id} className="w-full">
 {action.dismiss ? (
 <button
 type="button"
 className="w-full flex items-center justify-between p-1.5 bg-black hover:bg-black border border-white/10 hover:border-white/30 text-white text-left transition-colors cursor-pointer text-xs font-mono rounded-none"
 onClick={() => {
 setOpen(false);
 setDismissed(true);
 }}
 >
 <span className="px-1.5 py-0.5 bg-white text-black font-bold text-[10px] uppercase">
 {action.badge}
 </span>
 <span className="text-white/70 truncate flex-1 ml-2">{action.label}</span>
 </button>
 ) : action.prompt ? (
 <button
 type="button"
 className="w-full flex items-center justify-between p-1.5 bg-black hover:bg-black border border-white/10 hover:border-white/30 text-white text-left transition-colors cursor-pointer text-xs font-mono rounded-none"
 disabled={loading}
 onClick={() => void sendPrompt(action.prompt!)}
 >
 <span className="px-1.5 py-0.5 bg-white text-black font-bold text-[10px] uppercase shrink-0">
 {action.badge}
 </span>
 <span className="text-white/80 truncate flex-1 ml-2">{action.label}</span>
 <span className="text-white/40 text-[10px] ml-1">↵</span>
 </button>
 ) : (
 <Link
 href={action.href!}
 className="w-full flex items-center justify-between p-1.5 bg-black hover:bg-black border border-white/10 hover:border-white/30 text-white text-left transition-colors cursor-pointer text-xs font-mono rounded-none"
 onClick={() => setOpen(false)}
 >
 <span className="px-1.5 py-0.5 bg-white text-black font-bold text-[10px] uppercase shrink-0">
 {action.badge}
 </span>
 <span className="text-white/80 truncate flex-1 ml-2">{action.label}</span>
 <span className="text-white/40 text-[10px] ml-1">↗</span>
 </Link>
 )}
 </li>
 ))}
 </ul>

 {/* Message Composer */}
 <form className="site-assistant__composer flex items-center gap-2 p-2.5 bg-black border-t border-white/10 shrink-0" onSubmit={onSubmit}>
 <input
 ref={inputRef}
 type="text"
 value={input}
 onChange={(e) => setInput(e.target.value)}
 placeholder={language === 'ru' ? 'Спросите о платформе, ролях, решениях...' : 'Ask about the platform, roles, solutions...'}
 maxLength={2000}
 disabled={loading}
 className="flex-1 bg-black text-white placeholder-white/40 border border-white/20 focus:border-white px-3 py-2 text-xs font-mono rounded-none outline-none"
 aria-label={language === 'ru' ? 'Сообщение' : 'Message'}
 />
 <button
 type="submit"
 disabled={loading || !input.trim()}
 className="px-3.5 py-2 bg-white text-black hover:bg-zinc-200 disabled:opacity-30 disabled:cursor-not-allowed font-mono font-bold text-xs uppercase tracking-wider transition-colors cursor-pointer rounded-none shrink-0"
 >
 {language === 'ru' ? 'Ввод' : 'Send'}
 </button>
 </form>
 </div>
 </aside>
 ) : null}

 {/* Floating Action Launcher in Corner - Always Visible */}
 <aside aria-label="Pegasus AI Assistant" className="fixed bottom-4 right-4 sm:bottom-6 sm:right-6 z-[10005]">
 <button
 ref={launcherRef}
 type="button"
 className={`glowing-squircle-launcher group focus:outline-none ${open ? 'border-white/90 bg-black' : ''}`}
 title={open ? (language === 'ru' ? 'Закрыть Pegasus AI (Esc)' : 'Close Pegasus AI Assistant (Esc)') : (language === 'ru' ? 'Открыть Pegasus AI (⌘K)' : 'Open Pegasus AI Assistant (⌘K)')}
 aria-label={open ? 'Close Pegasus AI Assistant' : 'Open Pegasus AI Assistant'}
 onClick={() => setOpen((prev) => !prev)}
 >
 {open ? (
 <X className="w-5 h-5 sm:w-6 sm:h-6 text-white transition-transform duration-200 group-hover:scale-110" />
 ) : (
 <GooeyAgent color="#ffffff" size={34} className="group-hover:scale-105 transition-transform duration-200" />
 )}
 <span className="site-assistant__badge" aria-hidden="true">
 {open ? 'ESC' : '⌘K'}
 </span>
 </button>
 </aside>
 </div>
 );
}
