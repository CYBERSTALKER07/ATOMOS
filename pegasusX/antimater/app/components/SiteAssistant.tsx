'use client';

import { FormEvent, useEffect, useId, useRef, useState } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Maximize2, Copy, Check, RotateCcw, X, ChevronLeft, MoreHorizontal, Send } from 'lucide-react';
import { useLanguage } from '../context/LanguageContext';

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
 <div key={idx} className="my-2 border border-white/20 bg-black p-3 font-mono text-xs overflow-x-auto text-white rounded-xl">
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
 <h4 key={pIdx} className="font-mono font-bold text-xs mt-2 mb-1 tracking-wider uppercase">
 {para.replace('### ', '')}
 </h4>
 );
 }

 if (para.startsWith('## ')) {
 return (
 <h3 key={pIdx} className="font-mono font-bold text-sm mt-2 mb-1 tracking-wide uppercase">
 {para.replace('## ', '')}
 </h3>
 );
 }

 const formattedLine = para.split('\n').map((line, lIdx) => {
 const isBullet = line.trim().startsWith('- ') || line.trim().startsWith('* ');
 const cleanLine = isBullet ? line.trim().slice(2) : line;

 return (
 <div key={lIdx} className={isBullet ? 'flex items-start gap-1.5 pl-2' : ''}>
 {isBullet && <span className="opacity-60 font-mono select-none">•</span>}
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
					className="site-assistant__panel fixed bottom-[calc(env(safe-area-inset-bottom,0px)+1rem)] right-4 sm:bottom-[24px] sm:right-6 z-[10004] outline-none shadow-2xl"
					role="dialog"
					aria-label="Pegasus assistant"
				>
					<div
						className="rounded-[2.5rem] bg-white text-black overflow-hidden flex flex-col w-[380px] sm:w-[420px] max-w-[calc(100vw-2rem)] h-[620px] max-h-[calc(100vh-2rem)] sm:max-h-[calc(100vh-3rem)] shadow-[0_8px_30px_rgb(0,0,0,0.12)]"
					>
						{/* Header Bar */}
						<header className="flex items-center justify-between p-4 bg-white shrink-0">
							<div className="flex items-center gap-3">
								<button
									type="button"
									onClick={() => setOpen(false)}
									className="text-gray-400 hover:text-black transition-colors cursor-pointer"
								>
									<ChevronLeft className="w-5 h-5" />
								</button>
								<div className="w-9 h-9 rounded-xl bg-black flex items-center justify-center shrink-0 overflow-hidden">
									<img
										src="/pegasus.jpg"
										alt="Pegasus"
										className="w-full h-full object-contain invert scale-[1.2]"
									/>
								</div>
								<div className="flex flex-col">
									<span className="text-[15px] font-semibold text-gray-900 leading-tight">
										Pegasus bot
									</span>
									<span className="text-[12px] text-gray-500 leading-tight">
										The team can also help
									</span>
								</div>
							</div>

							<div className="flex items-center gap-2 shrink-0 text-gray-400">
								<button
									type="button"
									onClick={clearHistory}
									title={language === 'ru' ? 'Очистить историю' : 'Clear History'}
									className="p-1 hover:text-black transition-colors cursor-pointer"
								>
									<MoreHorizontal className="w-5 h-5" />
								</button>
								<button
									type="button"
									className="p-1 hover:text-black transition-colors cursor-pointer"
									title={language === 'ru' ? 'Закрыть (Esc)' : 'Close (Esc)'}
									onClick={() => setOpen(false)}
								>
									<X className="w-5 h-5" />
								</button>
							</div>
						</header>

						{/* Messages Stream */}
						<div
							ref={listRef}
							className="flex-1 overflow-y-auto p-4 space-y-4 bg-white"
							aria-live="polite"
						>
							{messages.length <= 1 && (
								<div className="text-center mb-6 mt-2">
									<p className="text-[13px] text-gray-500 max-w-[280px] mx-auto leading-relaxed">
										Empower digital transformation of Supply Chain, Revenue and IBP.
									</p>
								</div>
							)}

							{messages.map((msg) => (
								<div
									key={msg.id}
									className={`flex flex-col ${
										msg.role === 'assistant'
											? 'items-start max-w-[88%]'
											: 'items-end ml-auto max-w-[85%]'
									}`}
								>
									<div
										className={`p-4 text-[14px] leading-relaxed ${
											msg.role === 'assistant'
												? 'bg-gray-100 text-gray-800 rounded-3xl rounded-tl-sm'
												: 'bg-black text-white rounded-3xl rounded-tr-sm shadow-sm'
										}`}
									>
										<div className="">
											<FormattedContent text={msg.content} />
										</div>
									</div>
									{msg.role === 'assistant' && (
										<div className="mt-1.5 ml-2 text-[11px] text-gray-400 flex items-center gap-1">
											Pegasus bot &bull; AI Agent &bull; 5d
										</div>
									)}
								</div>
							))}

							{loading && (
								<div className="items-start max-w-[88%] flex flex-col">
									<div className="bg-gray-100 text-gray-800 p-4 text-[14px] rounded-3xl rounded-tl-sm">
										<div className="flex items-center gap-2 opacity-60">
											<div className="w-1.5 h-1.5 rounded-full bg-gray-500 animate-pulse" />
											<div className="w-1.5 h-1.5 rounded-full bg-gray-500 animate-pulse delay-75" />
											<div className="w-1.5 h-1.5 rounded-full bg-gray-500 animate-pulse delay-150" />
										</div>
									</div>
								</div>
							)}
						</div>

						{error && (
							<p className="text-[12px] text-red-500 px-4 py-2 bg-red-50 text-center">
								{error}
							</p>
						)}

						{/* Action Pills */}
						<div className="flex flex-col items-end gap-2 p-4 bg-white max-h-[220px] overflow-y-auto border-t border-gray-50 shrink-0">
							{currentQuickActions.map((action) => (
								<div key={action.id} className="max-w-[95%]">
									{action.dismiss ? (
										<button
											type="button"
											className="flex items-center justify-start px-4 py-2.5 bg-white hover:bg-gray-50 border border-gray-200 text-gray-800 transition-colors cursor-pointer text-[13px] rounded-full shadow-sm text-left"
											onClick={() => {
												setOpen(false);
												setDismissed(true);
											}}
										>
											<span className="mr-2">👋</span>
											<span className="truncate">{action.label}</span>
										</button>
									) : action.prompt ? (
										<button
											type="button"
											className="flex items-center justify-start px-4 py-2.5 bg-white hover:bg-gray-50 border border-gray-200 text-gray-800 transition-colors cursor-pointer text-[13px] rounded-full shadow-sm text-left"
											disabled={loading}
											onClick={() => void sendPrompt(action.prompt!)}
										>
											<span className="mr-2">💬</span>
											<span className="truncate">{action.label}</span>
										</button>
									) : (
										<Link
											href={action.href!}
											className="flex items-center justify-start px-4 py-2.5 bg-white hover:bg-gray-50 border border-gray-200 text-gray-800 transition-colors cursor-pointer text-[13px] rounded-full shadow-sm text-left"
											onClick={() => setOpen(false)}
										>
											<span className="mr-2">↗️</span>
											<span className="truncate">{action.label}</span>
										</Link>
									)}
								</div>
							))}
						</div>

						{/* Message Composer */}
						<form className="flex items-center gap-2 p-3 bg-white border-t border-gray-100 shrink-0" onSubmit={onSubmit}>
							<input
								ref={inputRef}
								type="text"
								value={input}
								onChange={(e) => setInput(e.target.value)}
								placeholder={language === 'ru' ? 'Написать сообщение...' : 'Reply to Pegasus bot...'}
								maxLength={2000}
								disabled={loading}
								className="flex-1 bg-gray-100 text-gray-800 placeholder-gray-500 border border-transparent focus:border-gray-200 px-4 py-2 text-[13px] rounded-full outline-none"
								aria-label="Message"
							/>
							<button
								type="submit"
								disabled={loading || !input.trim()}
								className="w-9 h-9 bg-black text-white hover:bg-zinc-800 disabled:opacity-30 disabled:cursor-not-allowed transition-colors cursor-pointer rounded-full shrink-0 flex items-center justify-center"
							>
								<Send className="w-4 h-4" />
							</button>
						</form>
					</div>
				</aside>
			) : null}

			{/* Floating Action Launcher in Corner - Always Visible */}
			<aside
				aria-label="Pegasus AI Assistant"
				className="fixed bottom-[calc(env(safe-area-inset-bottom,0px)+1rem)] right-4 sm:bottom-6 sm:right-6 z-[10005]"
			>
				<button
					ref={launcherRef}
					type="button"
					onClick={() => {
						setDismissed(false);
						setOpen((o) => !o);
					}}
					className="group relative flex items-center justify-center w-[52px] h-[52px] bg-black hover:bg-zinc-800 text-white rounded-full shadow-lg transition-transform hover:scale-105 active:scale-95"
					aria-expanded={open}
					aria-haspopup="dialog"
				>
					<span className="sr-only">
						{open ? 'Close Assistant' : 'Open Assistant'}
					</span>
					{open ? (
						<ChevronLeft className="w-6 h-6 transition-transform rotate-90" />
					) : (
						<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
							<path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path>
						</svg>
					)}
					{/* Notification Dot */}
					{!open && messages.length > 0 && (
						<span className="absolute top-0 right-0 w-3 h-3 bg-red-500 border-2 border-white rounded-full"></span>
					)}
				</button>
			</aside>
		</div>
	);
}
