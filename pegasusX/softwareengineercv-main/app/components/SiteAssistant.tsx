'use client';

import { FormEvent, useEffect, useId, useRef, useState } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
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
    id: 'contact-team',
    badge: 'JOIN',
    label: 'Запросить демо / Написать в Telegram',
    category: 'action',
    href: 'https://t.me/DominusMunerum',
  },
];

const WELCOME_RU =
  'Привет! Я ИИ-Ассистент Pegasus. Чем могу помочь? Узнайте о нашей Go-архитектуре, Cloud Spanner outbox или почему клиенты выбирают нас вместо Amazon AWS Supply Chain и o9.';

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

const SIDEBAR_CATEGORIES = [
  {
    id: 'all',
    title: 'Featured Prompts',
    badge: 'ALL',
    filter: () => true,
  },
  {
    id: 'versus',
    title: 'vs. Tech Giants',
    badge: 'VS',
    filter: (a: QuickAction) => a.category === 'versus',
  },
  {
    id: 'tech',
    title: 'Technical Stack',
    badge: 'STACK',
    filter: (a: QuickAction) => a.category === 'tech',
  },
  {
    id: 'roles',
    title: '6 Role Capabilities',
    badge: 'ROLES',
    filter: (a: QuickAction) => a.category === 'roles',
  },
  {
    id: 'business',
    title: 'Business & ROI',
    badge: 'ROI',
    filter: (a: QuickAction) => a.category === 'business',
  },
];

const HIDDEN_PREFIXES = ['/admin', '/resume'];
const WELCOME =
  'Welcome to Pegasus. Ask anything about our logistics OS — compare us to tech giants (Amazon, o9, Oracle), explore our Go & Spanner architecture, or dive into our 6 role capabilities.';

function newId() {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
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
  const [fullscreen, setFullscreen] = useState(false);
  const [activeCategory, setActiveCategory] = useState('all');
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

  // Lock body scroll when fullscreen modal is active
  useEffect(() => {
    if (open && fullscreen) {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = '';
    }
    return () => {
      document.body.style.overflow = '';
    };
  }, [open, fullscreen]);

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
        if (fullscreen) {
          setFullscreen(false);
        } else {
          setOpen(false);
        }
      }
    }

    window.addEventListener('keydown', handleKeyDown);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [open, fullscreen]);

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

    try {
      const res = await fetch('/api/assistant', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          messages: nextHistory.map(({ role, content }) => ({ role, content })),
          language,
        }),
      });
      const data = (await res.json()) as { reply?: string; error?: string };
      if (!res.ok || !data.reply) {
        throw new Error(data.error || 'Assistant unavailable');
      }
      setMessages((prev) => [...prev, { id: newId(), role: 'assistant', content: data.reply! }]);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Something went wrong';
      setError(message);
      setMessages((prev) => [
        ...prev,
        {
          id: newId(),
          role: 'assistant',
          content: language === 'ru' 
            ? `Не удалось получить ответ (${message}). Попробуйте еще раз или напишите в /contact.`
            : `I couldn’t answer that just now (${message}). Try again, or visit /contact.`,
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

  const currentCategoryObj = SIDEBAR_CATEGORIES.find((c) => c.id === activeCategory) || SIDEBAR_CATEGORIES[0];
  const filteredQuickActions = currentQuickActions.filter(currentCategoryObj.filter);

  return (
    <div 
      className={`site-assistant ${fullscreen ? 'site-assistant--fullscreen' : ''}`} 
      data-open={open ? 'true' : 'false'}
    >
      {open ? (
        <div
          id={panelId}
          ref={containerRef}
          className={`site-assistant__panel ${fullscreen ? 'site-assistant__panel--grok-modal' : 'site-assistant__panel--chat'}`}
          role="dialog"
          aria-label="Pegasus assistant"
          onClick={(e) => {
            if (fullscreen && e.target === e.currentTarget) {
              setOpen(false);
            }
          }}
        >
          <div className="site-assistant__chat-card">
            
            {/* Grok Fullscreen Left Sidebar (Only in fullscreen mode) */}
            {fullscreen ? (
              <aside className="site-assistant__grok-sidebar">
                <div className="site-assistant__sidebar-header">
                  <img src="/pegasus.jpg" alt="" width={28} height={28} className="site-assistant__avatar" />
                  <span className="site-assistant__grok-logo-text">PEGASUS OS</span>
                </div>

                <div className="site-assistant__sidebar-section">
                  <p className="site-assistant__sidebar-title">Categories</p>
                  <ul className="site-assistant__sidebar-menu">
                    {SIDEBAR_CATEGORIES.map((cat) => (
                      <li key={cat.id}>
                        <button
                          type="button"
                          className={`site-assistant__sidebar-btn ${activeCategory === cat.id ? 'is-active' : ''}`}
                          onClick={() => setActiveCategory(cat.id)}
                        >
                          <span className="site-assistant__tag-badge">{cat.badge}</span>
                          <span>{cat.title}</span>
                        </button>
                      </li>
                    ))}
                  </ul>
                </div>

                <div className="site-assistant__sidebar-footer">
                  <button type="button" className="site-assistant__clear-btn" onClick={clearHistory}>
                    Clear History
                  </button>
                </div>
              </aside>
            ) : null}

            {/* Main Chat Content Window */}
            <div className="site-assistant__grok-main">
              
              {/* Header Bar */}
              <header className="site-assistant__chat-head">
                <div className="site-assistant__head-info">
                  <img src="/pegasus.jpg" alt="" width={28} height={28} className="site-assistant__avatar" />
                  <div>
                    <p className="site-assistant__chat-title">
                      Pegasus Bot <span className="site-assistant__grok-badge">Full Mode</span>
                    </p>
                    <p className="site-assistant__meta">Architecture, Competitors & Role System Knowledge</p>
                  </div>
                </div>

                <div className="site-assistant__head-actions">
                  <button
                    type="button"
                    className="site-assistant__toggle-fullscreen"
                    title={fullscreen ? 'Compact View' : 'Fullscreen View'}
                    onClick={() => setFullscreen((v) => !v)}
                  >
                    {fullscreen ? 'Compact' : 'Fullscreen'}
                  </button>
                  
                  <button
                    type="button"
                    className="site-assistant__close-btn"
                    title="Close (Esc)"
                    onClick={() => setOpen(false)}
                  >
                    ×
                  </button>
                </div>
              </header>

              {/* Message Stream */}
              <div ref={listRef} className="site-assistant__messages" aria-live="polite">
                {messages.map((msg) => (
                  <div
                    key={msg.id}
                    className={`site-assistant__msg site-assistant__msg--${msg.role}`}
                  >
                    <div className="site-assistant__msg-header-bar">
                      <span className="site-assistant__msg-header">
                        {msg.role === 'assistant' ? 'PEGASUS OS' : 'YOU'}
                      </span>
                      
                      {msg.role === 'assistant' ? (
                        <button
                          type="button"
                          className="site-assistant__copy-btn"
                          onClick={() => copyToClipboard(msg.id, msg.content)}
                        >
                          {copiedId === msg.id ? 'Copied' : 'Copy'}
                        </button>
                      ) : null}
                    </div>
                    
                    <div className="site-assistant__msg-body">
                      {msg.content}
                    </div>
                  </div>
                ))}
                
                {loading ? (
                  <div className="site-assistant__msg site-assistant__msg--assistant">
                    <div className="site-assistant__msg-header-bar">
                      <span className="site-assistant__msg-header">PEGASUS OS</span>
                    </div>
                    <div className="site-assistant__status-indicator">
                      <span className="site-assistant__pulse-dot"></span>
                      <span>Processing response...</span>
                    </div>
                  </div>
                ) : null}
              </div>

              {error ? <p className="site-assistant__error">{error}</p> : null}

              {/* Action Pills */}
              <ul className="site-assistant__actions site-assistant__actions--inline">
                {filteredQuickActions.map((action) => (
                  <li key={action.id}>
                    {action.dismiss ? (
                      <button
                        type="button"
                        className="site-assistant__pill"
                        onClick={() => {
                          setOpen(false);
                          setDismissed(true);
                        }}
                      >
                        <span className="site-assistant__pill-badge">{action.badge}</span>
                        <span>{action.label}</span>
                      </button>
                    ) : action.prompt ? (
                      <button
                        type="button"
                        className="site-assistant__pill"
                        disabled={loading}
                        onClick={() => void sendPrompt(action.prompt!)}
                      >
                        <span className="site-assistant__pill-badge">{action.badge}</span>
                        <span>{action.label}</span>
                      </button>
                    ) : (
                      <Link href={action.href!} className="site-assistant__pill" onClick={() => setOpen(false)}>
                        <span className="site-assistant__pill-badge">{action.badge}</span>
                        <span>{action.label}</span>
                      </Link>
                    )}
                  </li>
                ))}
              </ul>

              {/* Message Composer */}
              <form className="site-assistant__composer" onSubmit={onSubmit}>
                <input
                  ref={inputRef}
                  type="text"
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  placeholder={t('assistant_placeholder')}
                  maxLength={2000}
                  disabled={loading}
                  aria-label="Message"
                />
                <button type="submit" disabled={loading || !input.trim()}>
                  {language === 'ru' ? 'Отправить' : 'Send'}
                </button>
              </form>

            </div>
          </div>
        </div>
      ) : null}

      {/* Floating Action Button Launcher */}
      {!open ? (
        <button
          ref={launcherRef}
          type="button"
          className="site-assistant__launcher"
          aria-expanded={open}
          aria-controls={panelId}
          aria-label="Open assistant"
          onClick={() => setOpen(true)}
        >
          <img src="/pegasus.jpg" alt="" width={40} height={40} className="site-assistant__launcher-logo" />
          <span className="site-assistant__badge" aria-hidden="true">
            ⌘K
          </span>
        </button>
      ) : null}
    </div>
  );
}
