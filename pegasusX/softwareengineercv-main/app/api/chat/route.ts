import librarySpec from '@/app/generated/spec.json';
import { promptOptions } from '@/app/lib/prompt-options';
import { createOpenAI } from '@ai-sdk/openai';
import { generateSystemPrompt } from '@openuidev/lang-core';
import { convertToModelMessages, stepCountIs, streamText, type UIMessage } from 'ai';
import { assistantSystemPrompt, buildAssistantKnowledge } from '@/app/lib/assistantKnowledge';

export const runtime = 'nodejs';

let cachedKnowledge: string | null = null;
function getKnowledge(): string {
  if (!cachedKnowledge) cachedKnowledge = buildAssistantKnowledge();
  return cachedKnowledge;
}

// xAI Grok or OpenAI provider (OpenAI compatible endpoint)
const apiKey = process.env.XAI_API_KEY?.trim() || process.env.OPENAI_API_KEY?.trim() || '';
const isXAI = Boolean(process.env.XAI_API_KEY?.trim());

const modelProvider = createOpenAI({
  baseURL: isXAI ? 'https://api.x.ai/v1' : (process.env.OPENAI_BASE_URL || undefined),
  apiKey: apiKey,
});

const defaultModel = isXAI ? (process.env.XAI_MODEL ?? 'grok-3-mini') : (process.env.OPENAI_MODEL ?? 'gpt-4o');

export async function POST(req: Request) {
  const payload = (await req.json()) as { messages?: UIMessage[] };
  if (!Array.isArray(payload.messages)) {
    return Response.json({ error: 'messages must be an array' }, { status: 400 });
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const openuiSystemPrompt = generateSystemPrompt({
    library: librarySpec as any,
    promptOptions: promptOptions as any,
  });

  const baseKnowledge = assistantSystemPrompt(getKnowledge());
  const combinedSystem = `${baseKnowledge}\n\n${openuiSystemPrompt}\n\nYou are the Pegasus Ecosystem AI Assistant. Deliver technical precision, crisp metrics, and generate OpenUI component UI cards, data tables, metrics, charts, or steps whenever helpful to illustrate logistics workflows.`;

  const result = streamText({
    model: modelProvider.chat(defaultModel),
    system: combinedSystem,
    messages: await convertToModelMessages(payload.messages),
    stopWhen: stepCountIs(5),
    abortSignal: req.signal,
  });

  return result.toUIMessageStreamResponse();
}
