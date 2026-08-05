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
  prompt?: string;
  href?: string;
  dismiss?: boolean;
};

const QUICK_ACTIONS: QuickAction[] = [
  {
    id: 'expert',
    emoji: '💬',
    label: 'Talk to a logistics expert at Pegasus',
    href: '/contact',
  },
  {
    id: 'platform',
    emoji: '📖',
    label: 'Learn how Pegasus runs dispatch, fleet, and payments',
    prompt: 'How does Pegasus run dispatch, fleet tracking, and payments across the six roles?',
  },
  {
    id: 'demo',
    emoji: '📺',
    label: 'Watch a platform walkthrough',
    href: '/demo',
  },
  {
    id: 'tour',
    emoji: '💻',
    label: 'Take a platform tour',
    prompt: 'Give me a short platform tour and the best pages to visit first.',
  },
  {
    id: 'careers',
    emoji: '👩‍💻',
    label: 'Careers at Pegasus',
    href: '/join',
  },
  {
    id: 'browse',
    emoji: '🙈',
    label: 'Just browsing',
    dismiss: true,
  },
];

const HIDDEN_PREFIXES = ['/admin', '/resume'];
const WELCOME =
  'Thanks for visiting Pegasus! Ask about dispatch, roles, payments, or how the platform works — or pick a shortcut below.';

function newId() {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

export default function SiteAssistant() {
  const pathname = usePathname();
  const panelId = useId();
  const listRef = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);
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

  return (
    <div className="site-assistant" data-open={open ? 'true' : 'false'}>
      {open ? (
        <div
          id={panelId}
          className="site-assistant__panel site-assistant__panel--chat"
          role="dialog"
          aria-label="Pegasus assistant"
        >
          <div className="site-assistant__chat-card">
            <header className="site-assistant__chat-head">
              <img src="/pegasus.jpg" alt="" width={28} height={28} className="site-assistant__avatar" />
              <div>
                <p className="site-assistant__chat-title">Pegasus bot</p>
                <p className="site-assistant__meta">Trained on Pegasus product & site data</p>
              </div>
            </header>

            <div ref={listRef} className="site-assistant__messages" aria-live="polite">
              {messages.map((msg) => (
                <div
                  key={msg.id}
                  className={`site-assistant__msg site-assistant__msg--${msg.role}`}
                >
                  {msg.content}
                </div>
              ))}
              {loading ? <div className="site-assistant__msg site-assistant__msg--assistant">Thinking…</div> : null}
            </div>

            {error ? <p className="site-assistant__error">{error}</p> : null}

            <ul className="site-assistant__actions site-assistant__actions--inline">
              {QUICK_ACTIONS.map((action) => (
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

            <form className="site-assistant__composer" onSubmit={onSubmit}>
              <input
                type="text"
                value={input}
                onChange={(e) => setInput(e.target.value)}
                placeholder="Ask about Pegasus…"
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
      ) : null}

      <button
        type="button"
        className="site-assistant__launcher"
        aria-expanded={open}
        aria-controls={panelId}
        aria-label={open ? 'Close assistant' : 'Open assistant'}
        onClick={() => setOpen((value) => !value)}
      >
        {open ? (
          <span className="site-assistant__launcher-x" aria-hidden="true">
            ×
          </span>
        ) : (
          <>
            <img src="/pegasus.jpg" alt="" width={40} height={40} className="site-assistant__launcher-logo" />
            <span className="site-assistant__badge" aria-hidden="true">
              1
            </span>
          </>
        )}
      </button>
    </div>
  );
}
