import { NextRequest, NextResponse } from 'next/server';
import { assistantSystemPrompt, buildAssistantKnowledge } from '@/app/lib/assistantKnowledge';

export const runtime = 'nodejs';

type ChatMessage = {
 role: 'user' | 'assistant' | 'system';
 content: string;
};

const isXAI = Boolean(process.env.XAI_API_KEY?.trim());
const API_URL = isXAI ? 'https://api.x.ai/v1/chat/completions' : (process.env.OPENAI_BASE_URL ? `${process.env.OPENAI_BASE_URL}/chat/completions` : 'https://api.openai.com/v1/chat/completions');
const MODEL = isXAI ? (process.env.XAI_MODEL ?? 'grok-3-mini') : (process.env.OPENAI_MODEL ?? 'gpt-4o');
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
 const apiKey = process.env.XAI_API_KEY?.trim() || process.env.OPENAI_API_KEY?.trim();
 if (!apiKey) {
 return NextResponse.json(
 { error: 'Assistant API key is not configured. Set XAI_API_KEY or OPENAI_API_KEY in the environment.' },
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

 const language = (body as { language?: string })?.language;
 const isRussian = language === 'ru' || messages.some((m) => /[а-яА-ЯёЁ]/.test(m.content));

 let systemPromptContent = assistantSystemPrompt(getKnowledge());
 if (isRussian) {
 systemPromptContent +=
 '\n\nIMPORTANT: The user prefers Russian. Always respond fluently, concisely, and professionally in Russian (русский язык) using standard high-tech logistics terminology. Do NOT use emojis under any circumstances.';
 }

 const shouldStream = request.headers.get('accept')?.includes('text/plain');

 const payload = {
 model: MODEL,
 temperature: 0.35,
 max_tokens: 1000,
 stream: shouldStream,
 messages: [
 { role: 'system', content: systemPromptContent },
 ...messages,
 ],
 };

 try {
 const upstream = await fetch(API_URL, {
 method: 'POST',
 headers: {
 Authorization: `Bearer ${apiKey}`,
 'Content-Type': 'application/json',
 },
 body: JSON.stringify(payload),
 });

 if (!upstream.ok) {
 const errorData = await upstream.json().catch(() => null);
 const detail =
 (errorData as { error?: { message?: string } })?.error?.message ??
 (errorData as { error?: string })?.error ??
 (errorData as { message?: string })?.message ??
 `Upstream AI provider error (${upstream.status})`;
 return NextResponse.json({ error: detail }, { status: upstream.status });
 }

 if (shouldStream && upstream.body) {
 const encoder = new TextEncoder();
 const decoder = new TextDecoder();
 const upstreamReader = upstream.body.getReader();

 const transformStream = new ReadableStream<Uint8Array>({
 async start(controller) {
 let buffer = '';
 try {
 while (true) {
 const { done, value } = await upstreamReader.read();
 if (done) break;
 buffer += decoder.decode(value, { stream: true });
 const lines = buffer.split('\n');
 buffer = lines.pop() || '';

 for (const line of lines) {
 const trimmed = line.trim();
 if (!trimmed.startsWith('data: ')) continue;
 const dataStr = trimmed.slice(6);
 if (dataStr === '[DONE]') continue;
 try {
 const parsed = JSON.parse(dataStr);
 const token = parsed.choices?.[0]?.delta?.content;
 if (token) {
 controller.enqueue(encoder.encode(token));
 }
 } catch {}
 }
 }
 } catch (err) {
 controller.error(err);
 } finally {
 controller.close();
 }
 },
 });

 return new Response(transformStream, {
 headers: {
 'Content-Type': 'text/plain; charset=utf-8',
 'Cache-Control': 'no-cache, no-transform',
 'Transfer-Encoding': 'chunked',
 },
 });
 }

 const data = (await upstream.json().catch(() => null)) as {
 choices?: { message?: { content?: string } }[];
 error?: { message?: string };
 } | null;

 const reply = data?.choices?.[0]?.message?.content?.trim();
 if (!reply) {
 return NextResponse.json({ error: 'Empty response from model' }, { status: 502 });
 }

 return NextResponse.json({ reply });
 } catch (err) {
 const message = err instanceof Error ? err.message : 'Failed to reach AI provider';
 return NextResponse.json({ error: message }, { status: 502 });
 }
}
