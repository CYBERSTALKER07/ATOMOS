import { NextRequest, NextResponse } from 'next/server';
import { assistantSystemPrompt, buildAssistantKnowledge } from '@/app/lib/assistantKnowledge';

export const runtime = 'nodejs';

type ChatMessage = {
  role: 'user' | 'assistant' | 'system';
  content: string;
};

const XAI_URL = 'https://api.x.ai/v1/chat/completions';
const MODEL = process.env.XAI_MODEL ?? 'grok-3-mini';
const MAX_HISTORY = 12;
const MAX_CONTENT_LEN = 2_000;

let cachedKnowledge: string | null = null;

function getKnowledge(): string {
  if (!cachedKnowledge) cachedKnowledge = buildAssistantKnowledge();
  return cachedKnowledge;
}

function sanitizeMessages(input: unknown): ChatMessage[] {
  if (!Array.isArray(input)) return [];
  const out: ChatMessage[] = [];
  for (const item of input) {
    if (!item || typeof item !== 'object') continue;
    const role = (item as { role?: string }).role;
    const content = (item as { content?: string }).content;
    if ((role !== 'user' && role !== 'assistant') || typeof content !== 'string') continue;
    const trimmed = content.trim().slice(0, MAX_CONTENT_LEN);
    if (!trimmed) continue;
    out.push({ role, content: trimmed });
    if (out.length >= MAX_HISTORY) break;
  }
  return out;
}

export async function POST(request: NextRequest) {
  const apiKey = process.env.XAI_API_KEY?.trim();
  if (!apiKey) {
    return NextResponse.json(
      { error: 'Assistant is not configured. Set XAI_API_KEY in the environment.' },
      { status: 503 },
    );
  }

  let body: unknown;
  try {
    body = await request.json();
  } catch {
    return NextResponse.json({ error: 'Invalid JSON body' }, { status: 400 });
  }

  const messages = sanitizeMessages((body as { messages?: unknown })?.messages);
  if (messages.length === 0 || messages[messages.length - 1]?.role !== 'user') {
    return NextResponse.json({ error: 'Send at least one user message' }, { status: 400 });
  }

  const payload = {
    model: MODEL,
    temperature: 0.35,
    max_tokens: 700,
    messages: [
      { role: 'system', content: assistantSystemPrompt(getKnowledge()) },
      ...messages,
    ],
  };

  try {
    const upstream = await fetch(XAI_URL, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${apiKey}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(payload),
    });

    const data = (await upstream.json().catch(() => null)) as {
      choices?: { message?: { content?: string } }[];
      error?: { message?: string };
    } | null;

    if (!upstream.ok) {
      const detail = data?.error?.message ?? `xAI error ${upstream.status}`;
      return NextResponse.json({ error: detail }, { status: 502 });
    }

    const reply = data?.choices?.[0]?.message?.content?.trim();
    if (!reply) {
      return NextResponse.json({ error: 'Empty response from model' }, { status: 502 });
    }

    return NextResponse.json({ reply });
  } catch {
    return NextResponse.json({ error: 'Failed to reach xAI' }, { status: 502 });
  }
}
