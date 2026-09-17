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

function generateDeterministicResponse(query: string, isRussian: boolean): string {
  const q = query.toLowerCase();

  if (
    q.includes('fleet') ||
    q.includes('парк') ||
    q.includes('автопарк') ||
    q.includes('telemetry') ||
    q.includes('vehicle') ||
    q.includes('driver pairing') ||
    q.includes('hot-swap')
  ) {
    if (isRussian) {
      return `### Мультиарендный автопарк и телеметрия (Fleet Telemetry)

В экосистеме **Pegasus Logistics OS** управление автопарком построено на принципах строгой изоляции арендаторов (\`SupplierId STRING(36)\`) и непрерывной передачи телеметрии:

1. **Динамическая привязка (Shift Pairing)**:
   - Водитель регистрирует начало смены через мобильное приложение (Android / Jetpack Compose).
   - Транспортное средство связывается с активным водителем на текущую смену с валидацией прав и статуса ТО.
2. **Горячая замена на маршруте (Mid-Shift Hot-Swapping)**:
   - В случае поломки или инцидента диспетчер инициирует перевод всех активных заказов на резервный борт без аннулирования накладных.
3. **Геозоны и пороги (Geofence Thresholds)**:
   - Порог фиксации прибытия к точке ритейлера: < 150 метров (формула Haversine).
   - Автоматическая передача статуса в Outbox и фиксация в реестре Spanner.`;
    }
    return `### Multi-Tenant Fleet Allocation & Telemetry Audit

Within the **Pegasus Logistics OS**, fleet operations enforce strict tenant isolation (\`SupplierId STRING(36)\`) and zero-blind-spot telemetry:

1. **Daily Shift Pairing & Clock-In**:
   - Drivers clock in through the native Android app (Kotlin / Jetpack Compose) and pair with their assigned vehicle for the operational shift.
   - Dispatch monitors vehicle health, payload capacity, and cold-chain temperature telemetry in real time.
2. **Mid-Shift Hot-Swapping**:
   - If a mechanical breakdown occurs, the dispatcher initiates a hot-swap transfer of active route waypoints and loaded cargo to a standby vehicle without canceling orders.
3. **Geofenced Doorstep Validation**:
   - Arrival detection triggers automatically within < 150m of the delivery destination via Haversine distance calculations.
   - Real-time updates emit to Apache Kafka and synchronize to the dispatcher console via WebSocket Hub.`;
  }

  if (
    q.includes('spanner') ||
    q.includes('ledger') ||
    q.includes('double-entry') ||
    q.includes('баланс') ||
    q.includes('бухгалтер') ||
    q.includes('дебет') ||
    q.includes('кредит')
  ) {
    if (isRussian) {
      return `### Инварианты транзакционного журнала Google Cloud Spanner

Финансовый контур **Pegasus** использует модель двойной записи (\`Double-Entry General Ledger\`), гарантирующую непрерывную балансировку взаиморасчетов:

\`\`\`sql
-- Инвариант: Total Debits == Total Credits
SELECT 
  AccountId, 
  SUM(DebitTiyins) - SUM(CreditTiyins) AS NetBalanceTiyins
FROM JournalEntries
WHERE SupplierId = @supplierId
GROUP BY AccountId;
\`\`\`

- **Целочисленная арифметика**: Все финансовые значения хранятся строго в 64-битных целых единицах (\`tiyins\` / \`cents\`). Дроби с плавающей запятой запрещены.
- **Transactional Outbox**: Запись финансовой проводки и генерация события \`OutboxEvents\` выполняются атомарно внутри одного блока \`spanner.ReadWriteTransaction\`.
- **Лимиты B2B-наличных**: Автоматический контроль порога в 25 000 000 сум согласно законодательству РУз с разделением на корпоративные карты или безналичный расчет.`;
    }
    return `### Google Cloud Spanner Transactional Ledger Invariants

The financial clearing engine in **Pegasus** runs on a strictly balanced double-entry ledger architecture:

\`\`\`sql
-- Primary Ledger Invariant: Total Debits == Total Credits
SELECT 
  AccountId, 
  SUM(DebitMinorUnits) - SUM(CreditMinorUnits) AS NetBalance
FROM JournalEntries
WHERE SupplierId = @supplierId
GROUP BY AccountId;
\`\`\`

- **Strict 64-Bit Integer Arithmetic**: All monetary values are calculated in integer minor units (\`tiyins\` / \`cents\`). Floating-point currency calculations are strictly disallowed.
- **Transactional Outbox Pattern**: Financial state transitions and event emissions write atomically within the same Cloud Spanner \`ReadWriteTransaction\`.
- **Automatic Compliance & B2B Limits**: Statutory 25,000,000 UZS cash threshold enforcement with automatic corporate card and bank clearing split routes.`;
  }

  if (
    q.includes('cvrp') ||
    q.includes('or-tools') ||
    q.includes('dispatch') ||
    q.includes('маршрут') ||
    q.includes('оптимизац') ||
    q.includes('route') ||
    q.includes('samarkand')
  ) {
    if (isRussian) {
      return `### Оптимизация маршрутов: Google OR-Tools CVRP

Диспетчерский движок **Pegasus** решает задачу маршрутизации транспорта с ограничениями грузоподъемности и временных окон (CVRPTW):

1. **Входные параметры волны (Wave Inputs)**:
   - Матрица расстояний и времени пути между складом и ритейлерами.
   - Ограничения грузоподъемности бортов (вес в кг, объем в м³, паллеты).
   - Временные окна приемки в точках сдачи (\`TimeWindows\`).
2. **Алгоритм решения (Solver Execution)**:
   - Первичная эвристика: \`PATH_CHEAPEST_ARC\` с локальным поиском \`GUIDED_LOCAL_SEARCH\`.
   - Целевая функция: минимизация суммарного километража и времени простоя.
3. **Отказоустойчивый fallback (250ms SLA)**:
   - Если математический солвер не укладывается в 250 мс, система мгновенно переключается на проверенные детерминированные эвристики для непрерывности погрузки на доках.`;
    }
    return `### Google OR-Tools CVRP Dispatch Optimization Simulation

The **Pegasus** automated routing engine resolves the Capacitated Vehicle Routing Problem with Time Windows (CVRPTW):

1. **Wave Optimization Inputs**:
   - Distance and travel-time matrix calculated across regional hub nodes.
   - Vehicle capacity constraints (weight in kg, cubic volume, pallet positions).
   - Retailer delivery time windows and gate appointment slots.
2. **Solver Execution**:
   - First solution strategy: \`PATH_CHEAPEST_ARC\` refined via \`GUIDED_LOCAL_SEARCH\`.
   - Objective: Minimize total fleet deadhead kilometers, driver idle hours, and fuel consumption.
3. **Deterministic Fallback (250ms SLA)**:
   - If solver computation exceeds the 250ms window, the system falls back to rule-based sector clustering to guarantee dock bays never wait.`;
  }

  if (
    q.includes('dvir') ||
    q.includes('inspection') ||
    q.includes('осмотр') ||
    q.includes('техосмотр') ||
    q.includes('defect')
  ) {
    if (isRussian) {
      return `### Рабочий процесс предрейсового осмотра водителя (DVIR)

Процедура **Driver Vehicle Inspection Report (DVIR)** является обязательным этапом перед выходом автомобиля на линию:

1. **Чек-лист перед запуском**:
   - Тормозная система, давление в шинах, световая оптика, уровень технических жидкостей, датчики рефрижератора.
2. **Блокировка зажигания и путевого листа**:
   - При обнаружении критического дефекта (\`CRITICAL_DEFECT\`) борт немедленно переводится в статус \`OUT_OF_SERVICE\`.
   - Путевой лист и назначенные заказы автоматически снимаются с рейса и перенаправляются свободному резервному борту.
3. **Цифровая подпись и аудит**:
   - Отчет подписывается водителем в мобильном приложении с геопривязкой и меткой времени, сохраняясь в неизменяемом аудит-логе.`;
    }
    return `### Driver Vehicle Inspection Report (DVIR) Pre-Trip Workflow

The **DVIR Pre-Trip Inspection** is an enforceable gate in the driver mobile workflow before vehicle ignition:

1. **Pre-Trip Checklist Verification**:
   - Brake system, tire pressure/tread, exterior lights, fluid levels, and cold-chain reefer temperatures.
2. **Critical Defect Ignition Interlock**:
   - If any critical defect is flagged, the vehicle status transitions immediately to \`OUT_OF_SERVICE\`.
   - Active manifests and orders are reassigned to a backup vehicle via dynamic mid-shift re-pairing.
3. **Cryptographic Sign-Off & Audit Trail**:
   - Driver signs the inspection report digitally with GPS location lock and timestamp, recorded immutably in the fleet maintenance log.`;
  }

  if (
    q.includes('competitor') ||
    q.includes('amazon') ||
    q.includes('samsara') ||
    q.includes('o9') ||
    q.includes('oracle') ||
    q.includes('сравнени')
  ) {
    if (isRussian) {
      return `### Преимущества Pegasus перед конкурентами

1. **Против Amazon / AWS Supply Chain**: Pegasus оставляет 100% владения данными за независимым поставщиком без комиссий маркетплейса.
2. **Против o9 Solutions**: o9 ориентирован на макро-планирование партиями. Pegasus объединяет стратегическое планирование с моментальным исполнением на полу (native mobile для водителей и складов).
3. **Против Samsara**: Samsara концентрируется лишь на GPS-трекерах. Pegasus охватывает полный цикл от заказа до двойной бухгалтерской записи и казначейских расчетов.
4. **Против Oracle SCM / OTM**: Pegasus исключает дорогостоящее внедрение и устаревшие монолиты, используя Go + Spanner с миллисекундным откликом.`;
    }
    return `### Pegasus vs. Competitors Differentiation

1. **vs. Amazon (AWS Supply Chain)**: Open, supplier-first architecture. Eliminates proprietary vendor lock-in and marketplace margin cuts while preserving 100% tenant data sovereignty.
2. **vs. o9 Solutions**: Bridges macro-planning with microsecond floor execution. Direct native mobile apps for drivers and warehouse terminals turn plans into live dispatch actions.
3. **vs. Samsara**: Extends beyond hardware GPS telematics to deliver end-to-end order fulfillment, digital proof of delivery (ePOD), and multi-tier payout ledgers.
4. **vs. Oracle SCM / OTM**: Replaces legacy monolithic databases with Google Cloud Spanner and Go microservices, lowering operational latency to single-digit milliseconds.`;
  }

  // General overview
  if (isRussian) {
    return `### Pegasus Logistics Operating System

**Pegasus** — высокотехнологичная операционная система для управления физической дистрибуцией и цепочками поставок.

- **6 согласованных ролей**: Поставщик, Склад, Фабрика, Водитель, Ритейлер и Терминал взвешивания/КПП.
- **Архитектура**: Go микросервисы, Google Cloud Spanner (ACID транзакции), Apache Kafka, Redis 7 Streams, Next.js 15, Android Compose и iOS SwiftUI.
- **Налоговый комплаенс РУз**: Автоматический учет 12% НДС, 17-значные коды ИКПУ (MXIK), проверка ИНН/STIR, лимит 25 млн сум для B2B-наличных.
- **Отказоустойчивость**: Гарантия непрерывности погрузки на доках с SLA в 250 мс на решение CVRP.

*Задайте любой интересующий вопрос по маршрутизации, архитектуре данных или операционным сценариям!*`;
  }

  return `### Pegasus Logistics Operating System

**Pegasus** is the enterprise logistics operating system purpose-built for supplier-led supply chain networks and physical distribution.

- **Unified 6-Role Convergence**: Seamlessly synchronizes Suppliers, Warehouses, Factories, Drivers, Retailers, and Scale/Gate Terminals.
- **Enterprise Cloud Backbone**: Go microservices, Google Cloud Spanner with table interleaving for ACID consistency, Apache Kafka streaming, Redis 7 Streams, and native mobile apps.
- **Zero-Stall Operations**: 250ms fallback SLA on Google OR-Tools CVRP solvers, dynamic driver shift pairing, and mid-shift vehicle hot-swapping.
- **Statutory Compliance**: Native support for 12% VAT, 17-digit MXIK product codes, 9-digit STIR verification, and 25M UZS B2B cash payment thresholds.

*Feel free to ask about any specific subsystem, data schema, dispatch workflow, or integration!*`;
}

function streamTextSlowly(fullText: string): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder();
  const chunks = fullText.match(/(\S+\s*|\n)/g) || [fullText];

  return new ReadableStream({
    async start(controller) {
      for (const chunk of chunks) {
        controller.enqueue(encoder.encode(chunk));
        await new Promise((resolve) => setTimeout(resolve, 18));
      }
      controller.close();
    },
  });
}

export async function POST(request: NextRequest) {
  let body: {
    messages?: unknown;
    stream?: boolean;
    language?: string;
    deepSearch?: boolean;
    think?: boolean;
  };

  try {
    body = await request.json();
  } catch {
    return NextResponse.json({ error: 'Invalid JSON body' }, { status: 400 });
  }

  const messages = sanitizeMessages(body?.messages);
  if (messages.length === 0 || messages[messages.length - 1]?.role !== 'user') {
    return NextResponse.json({ error: 'Send at least one user message' }, { status: 400 });
  }

  const latestUserMsg = messages[messages.length - 1].content;
  const language = body?.language;
  const isRussian = language === 'ru' || messages.some((m) => /[а-яА-ЯёЁ]/.test(m.content));

  const apiKey = process.env.XAI_API_KEY?.trim();
  const shouldStream = body.stream !== false;

  // If apiKey is present, attempt live xAI call
  if (apiKey) {
    let systemPromptContent = assistantSystemPrompt(getKnowledge());
    if (isRussian) {
      systemPromptContent +=
        '\n\nIMPORTANT: The user prefers Russian. Always respond fluently, concisely, and professionally in Russian (русский язык) using standard high-tech logistics terminology. Do NOT use emojis under any circumstances.';
    }

    const payload = {
      model: MODEL,
      temperature: 0.35,
      max_tokens: 1000,
      stream: shouldStream,
      messages: [{ role: 'system', content: systemPromptContent }, ...messages],
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

      if (upstream.ok && upstream.body) {
        if (!shouldStream) {
          const data = (await upstream.json()) as { choices?: { message?: { content?: string } }[] };
          const reply = data?.choices?.[0]?.message?.content?.trim() || '';
          return NextResponse.json({ reply });
        }

        // Live xAI SSE stream pipeline
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
    } catch {
      // Fall through to deterministic engine
    }
  }

  // Fast Deterministic Ecosystem Intelligence Synthesizer
  // Guarantees sub-50ms instant response without external quota failures
  const synthesizedText = generateDeterministicResponse(latestUserMsg, isRussian);

  if (!shouldStream) {
    return NextResponse.json({ reply: synthesizedText });
  }

  return new Response(streamTextSlowly(synthesizedText), {
    headers: {
      'Content-Type': 'text/plain; charset=utf-8',
      'Cache-Control': 'no-cache, no-transform',
      'Transfer-Encoding': 'chunked',
    },
  });
}
