'use client';

import { FormEvent, useEffect, useId, useRef, useState } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';

type ChatRole = 'user' | 'assistant';

type ChatMessage = {
  id: string;
  role: ChatRole;
  content: string;
};

type QuickAction = {
  id: string;
  emoji: string;
  label: string;
  category: 'versus' | 'tech' | 'roles' | 'business' | 'action';
  prompt?: string;
  href?: string;
  dismiss?: boolean;
};

const QUICK_ACTIONS: QuickAction[] = [
  {
    id: 'versus-giants',
    emoji: '⚡',
    label: 'Why Pegasus vs Amazon, o9 & Oracle?',
    category: 'versus',
    prompt: 'Compare Pegasus with tech giants & legacy ERPs (Amazon AWS Supply Chain, o9 Solutions, Oracle OTM, Google Cloud Twin, Blue Yonder). Why choose Pegasus?',
  },
  {
    id: 'tech-stack',
    emoji: '⚙️',
    label: 'Go Backend, Spanner & Outbox System',
    category: 'tech',
    prompt: 'Explain the technical architecture of Pegasus: Go microservices, Cloud Spanner transactions, Transactional Outbox pattern, Kafka, WebSockets, and offline mobile sync.',
  },
  {
    id: 'six-roles',
    emoji: '👥',
    label: '6 Ecosystem Roles & Capabilities',
    category: 'roles',
    prompt: 'What are the 6 ecosystem roles in Pegasus (Supplier, Warehouse, Retailer, Driver, Factory, Payload/Gate) and their key capabilities?',
  },
  {
    id: 'business-roi',
    emoji: '💼',
    label: 'Business Benefits & Operational ROI',
    category: 'business',
    prompt: 'What are the core business outcomes, ROI, and workflow improvements Pegasus delivers for supply chain leadership?',
  },
  {
    id: 'demo-tour',
    emoji: '📺',
    label: 'Watch Platform Walkthrough',
    category: 'action',
    href: '/demo',
  },
  {
    id: 'contact-expert',
    emoji: '💬',
    label: 'Talk to a Logistics Expert',
    category: 'action',
    href: '/contact',
  },
  {
    id: 'join-careers',
    emoji: '👩‍💻',
    label: 'Careers & Schedule Demo',
    category: 'action',
    href: '/join',
  },
  {
    id: 'dismiss',
    emoji: '🙈',
    label: 'Just browsing',
    category: 'action',
    dismiss: true,
  },
];

const SIDEBAR_CATEGORIES = [
  {
    id: 'all',
    title: 'Featured Prompts',
    icon: '✨',
    filter: () => true,
  },
  {
    id: 'versus',
    title: 'vs. Tech Giants',
    icon: '⚡',
    filter: (a: QuickAction) => a.category === 'versus',
  },
  {
    id: 'tech',
    title: 'Technical Stack',
    icon: '⚙️',
    filter: (a: QuickAction) => a.category === 'tech',
  },
  {
    id: 'roles',
    title: '6 Role Capabilities',
    icon: '👥',
    filter: (a: QuickAction) => a.category === 'roles',
  },
  {
    id: 'business',
    title: 'Business & ROI',
    icon: '💼',
    filter: (a: QuickAction) => a.category === 'business',
  },
];

const HIDDEN_PREFIXES = ['/admin', '/resume'];
const WELCOME =
  'Welcome to Pegasus! Ask me anything about our logistics OS — compare us to tech giants (Amazon, o9, Oracle), explore our Go & Spanner architecture, or dive into our 6 role capabilities.';

function newId() {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

export default function SiteAssistant() {
  const pathname = usePathname();
  const panelId = useId();
  const listRef = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);
  const [fullscreen, setFullscreen] = useState(false);
  const [activeCategory, setActiveCategory] = useState('all');
  const [dismissed, setDismissed] = useState(false);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([
    { id: 'welcome', role: 'assistant', content: WELCOME },
  ]);

  useEffect(() => {
    setOpen(false);
  }, [pathname]);

  useEffect(() => {
    if (!open) return;
    listRef.current?.scrollTo({ top: listRef.current.scrollHeight, behavior: 'smooth' });
  }, [messages, open, loading]);

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
          content: `I couldn’t answer that just now (${message}). Try again, or visit /contact.`,
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
    setMessages([{ id: newId(), role: 'assistant', content: WELCOME }]);
    setError(null);
  }

  const currentCategoryObj = SIDEBAR_CATEGORIES.find((c) => c.id === activeCategory) || SIDEBAR_CATEGORIES[0];
  const filteredQuickActions = QUICK_ACTIONS.filter(currentCategoryObj.filter);

  return (
    <div 
      className={`site-assistant ${fullscreen ? 'site-assistant--fullscreen' : ''}`} 
      data-open={open ? 'true' : 'false'}
    >
      {open ? (
        <div
          id={panelId}
          className={`site-assistant__panel ${fullscreen ? 'site-assistant__panel--grok-modal' : 'site-assistant__panel--chat'}`}
          role="dialog"
          aria-label="Pegasus assistant"
        >
          <div className="site-assistant__chat-card">
            
            {/* Grok Fullscreen Left Sidebar (Only visible in fullscreen) */}
            {fullscreen ? (
              <aside className="site-assistant__grok-sidebar">
                <div className="site-assistant__sidebar-header">
                  <img src="/pegasus.jpg" alt="" width={28} height={28} className="site-assistant__avatar" />
                  <span className="site-assistant__grok-logo-text">Pegasus Grok</span>
                </div>

                <div className="site-assistant__sidebar-section">
                  <p className="site-assistant__sidebar-title">Explore Topics</p>
                  <ul className="site-assistant__sidebar-menu">
                    {SIDEBAR_CATEGORIES.map((cat) => (
                      <li key={cat.id}>
                        <button
                          type="button"
                          className={`site-assistant__sidebar-btn ${activeCategory === cat.id ? 'is-active' : ''}`}
                          onClick={() => setActiveCategory(cat.id)}
                        >
                          <span className="site-assistant__sidebar-icon">{cat.icon}</span>
                          <span>{cat.title}</span>
                        </button>
                      </li>
                    ))}
                  </ul>
                </div>

                <div className="site-assistant__sidebar-footer">
                  <button type="button" className="site-assistant__clear-btn" onClick={clearHistory}>
                    <span>🗑️</span> Clear Chat History
                  </button>
                </div>
              </aside>
            ) : null}

            {/* Main Chat Area */}
            <div className="site-assistant__grok-main">
              
              {/* Header Bar */}
              <header className="site-assistant__chat-head">
                <div className="site-assistant__head-info">
                  <img src="/pegasus.jpg" alt="" width={28} height={28} className="site-assistant__avatar" />
                  <div>
                    <p className="site-assistant__chat-title">
                      Pegasus Bot <span className="site-assistant__grok-badge">Grok Mode</span>
                    </p>
                    <p className="site-assistant__meta">Trained on Architecture, Competitors & 6 Roles</p>
                  </div>
                </div>

                <div className="site-assistant__head-actions">
                  <button
                    type="button"
                    className="site-assistant__toggle-fullscreen"
                    title={fullscreen ? 'Exit Fullscreen' : 'Expand Grok Layout'}
                    onClick={() => setFullscreen((v) => !v)}
                  >
                    {fullscreen ? '↙ Compact' : '⤢ Fullscreen'}
                  </button>
                  
                  <button
                    type="button"
                    className="site-assistant__close-btn"
                    title="Close Assistant"
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
                    <div className="site-assistant__msg-header">
                      {msg.role === 'assistant' ? '🤖 Pegasus AI' : '👤 You'}
                    </div>
                    <div className="site-assistant__msg-body">
                      {msg.content}
                    </div>
                  </div>
                ))}
                {loading ? (
                  <div className="site-assistant__msg site-assistant__msg--assistant">
                    <div className="site-assistant__msg-header">🤖 Pegasus AI</div>
                    <div className="site-assistant__typing-dots">
                      <span></span><span></span><span></span> Thinking...
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
                        <span aria-hidden="true">{action.emoji}</span>
                        <span>{action.label}</span>
                      </button>
                    ) : action.prompt ? (
                      <button
                        type="button"
                        className="site-assistant__pill"
                        disabled={loading}
                        onClick={() => void sendPrompt(action.prompt!)}
                      >
                        <span aria-hidden="true">{action.emoji}</span>
                        <span>{action.label}</span>
                      </button>
                    ) : (
                      <Link href={action.href!} className="site-assistant__pill" onClick={() => setOpen(false)}>
                        <span aria-hidden="true">{action.emoji}</span>
                        <span>{action.label}</span>
                      </Link>
                    )}
                  </li>
                ))}
              </ul>

              {/* Message Composer */}
              <form className="site-assistant__composer" onSubmit={onSubmit}>
                <input
                  type="text"
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  placeholder="Ask about architecture, competitors (Amazon/o9), 6 role capabilities..."
                  maxLength={2000}
                  disabled={loading}
                  aria-label="Message"
                />
                <button type="submit" disabled={loading || !input.trim()}>
                  Send
                </button>
              </form>

            </div>
          </div>
        </div>
      ) : null}

      {/* Floating Action Button Launcher */}
      {!open ? (
        <button
          type="button"
          className="site-assistant__launcher"
          aria-expanded={open}
          aria-controls={panelId}
          aria-label="Open assistant"
          onClick={() => setOpen(true)}
        >
          <img src="/pegasus.jpg" alt="" width={40} height={40} className="site-assistant__launcher-logo" />
          <span className="site-assistant__badge" aria-hidden="true">
            1
          </span>
        </button>
      ) : null}
    </div>
  );
}
