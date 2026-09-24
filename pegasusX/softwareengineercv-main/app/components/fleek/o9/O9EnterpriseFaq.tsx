'use client';

import React, { useState, useMemo, useId } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { ChevronDown, Search, X, CheckCircle2, HelpCircle } from 'lucide-react';
import { useLanguage } from '@/app/context/LanguageContext';
import { cn } from '@/lib/utils';

export interface O9FaqItem {
  id: string;
  question: string;
  answer: string;
  category?: string;
  tag?: string;
  highlights?: string[];
}

export interface O9EnterpriseFaqProps {
  kicker?: string;
  title?: string;
  description?: string;
  items?: O9FaqItem[];
  allowMultiple?: boolean;
  className?: string;
  searchPlaceholder?: string;
  defaultOpenIndex?: number;
}

const DEFAULT_ENTERPRISE_FAQS_EN: O9FaqItem[] = [
  {
    id: 'erp-integration',
    category: 'Integration',
    tag: 'ERP / WMS',
    question: 'How does Pegasus integrate with existing legacy ERPs like SAP, Oracle, and 1C?',
    answer:
      'Pegasus operates as a non-invasive real-time execution overlay using bidirectional REST/gRPC connectors and an asynchronous Kafka event streaming bus. Existing ERPs retain system-of-record authority for financial general ledgers and master catalog records, while Pegasus handles sub-second dispatch, live driver telemetry, and atomic inventory locks without requiring high-risk rip-and-replace migrations.',
    highlights: [
      'Bidirectional REST/gRPC & Kafka connectors',
      'Zero rip-and-replace required for existing systems',
      'Sub-second real-time state synchronization',
    ],
  },
  {
    id: 'tenant-isolation',
    category: 'Security',
    tag: 'CRYPTO CLAIMS',
    question: 'How is cryptographic tenant isolation maintained across competing suppliers in shared warehouse nodes?',
    answer:
      'Every transaction, mutation, and query execution is strictly bound to cryptographic JWT tenant claims. At the persistence layer, Google Cloud Spanner row-level security and tenant-scoped table partitions ensure suppliers and third-party logistics operators cannot view, query, or infer another tenant\'s inventory levels, route plans, or cost structures.',
    highlights: [
      'Cryptographic JWT tenant perimeters',
      'Cloud Spanner row-level security policies',
      'Zero cross-tenant data bleed guaranteed',
    ],
  },
  {
    id: 'offline-connectivity',
    category: 'Reliability',
    tag: 'OFFLINE FIRST',
    question: 'What happens if mobile network connectivity drops during driver transit or in remote warehouse zones?',
    answer:
      'The Pegasus Driver and Warehouse mobile applications operate offline-first using local SQLite transactional queues. Mutations such as barcode scans, digital delivery signatures, and GPS pings are queued locally with cryptographic timestamps and monotonically increasing sequence numbers. Once connectivity is restored, the client replays events through an idempotent outbox pipeline with zero data loss or duplicate charges.',
    highlights: [
      'Offline-first SQLite transactional queue',
      'Cryptographic event replay on reconnect',
      'Zero duplicate mutations or lost manifests',
    ],
  },
  {
    id: 'concurrency-locks',
    category: 'Architecture',
    tag: 'STATE MACHINE',
    question: 'How does Pegasus eliminate race conditions and stockouts during high-volume freight spikes?',
    answer:
      'All state modifications pass through a strict atomic mutation pipeline: Validate JWT Scope → Acquire Distributed Lock → Mutate State Machine → Commit Event Outbox → Fanout WebSocket Sync. Concurrent orders for the same SKU are evaluated sequentially at Cloud Spanner commit time; failed reservations fail fast with deterministic error reasons before financial authorization occurs.',
    highlights: [
      'Atomic mutation pipeline with distributed lock',
      'Deterministic zero-overselling guarantee',
      'Sub-millisecond commit latency under heavy load',
    ],
  },
  {
    id: 'deployment-timeline',
    category: 'Rollout',
    tag: 'ONBOARDING',
    question: 'What is the typical deployment timeline for an enterprise distribution network?',
    answer:
      'A phased rollout typically completes in 60 to 90 days. Phase 1 (Days 1–30) establishes ERP/WMS connectors and shadows live inventory. Phase 2 (Days 31–60) deploys warehouse dispatch boards and driver mobile apps to pilot hubs. Phase 3 (Days 61–90) activates automated billing hard-gates and transitions full production traffic with 24/7 dedicated engineering support.',
    highlights: [
      '60–90 day typical enterprise rollout timeline',
      '3-phase non-disruptive migration strategy',
      '24/7 dedicated engineering support & SLA',
    ],
  },
  {
    id: 'hardware-scanners',
    category: 'Integration',
    tag: 'HARDWARE & IOT',
    question: 'Does Pegasus support industrial handheld barcode scanners and cold-chain temperature sensors?',
    answer:
      'Yes. Pegasus provides native hardware SDK bindings and HID barcode wedge support for Zebra, Honeywell, and Datalogic scanners. For cold-chain operations, BLE and IoT temperature telemetry streams are ingested in real-time, automatically triggering compliance exception workflows if trailer temperatures breach pre-set thresholds.',
    highlights: [
      'Zebra, Honeywell & Datalogic native scanner support',
      'Real-time BLE & IoT temperature sensor ingestion',
      'Automated compliance exception playbooks',
    ],
  },
];

const DEFAULT_ENTERPRISE_FAQS_RU: O9FaqItem[] = [
  {
    id: 'erp-integration',
    category: 'Интеграция',
    tag: 'ERP / WMS',
    question: 'Как Pegasus интегрируется с существующими ERP-системами, такими как SAP, Oracle и 1C?',
    answer:
      'Pegasus работает как неинвазивный контур исполнения в реальном времени через двунаправленные коннекторы REST/gRPC и шину событий Kafka. Существующие ERP сохраняют статус основной учетной системы, в то время как Pegasus берет на себя субсекундную диспетчеризацию, телеметрию водителей и атомарные блокировки складских остатков без рискованной замены всей инфраструктуры.',
    highlights: [
      'Двунаправленные REST/gRPC и Kafka коннекторы',
      'Без необходимости замены существующих систем',
      'Субсекундная синхронизация состояния',
    ],
  },
  {
    id: 'tenant-isolation',
    category: 'Безопасность',
    tag: 'КРИПТО-ИЗОЛЯЦИЯ',
    question: 'Как обеспечивается криптографическая изоляция арендаторов между конкурирующими поставщиками на общих складах?',
    answer:
      'Каждая транзакция, мутация и выполнение запроса строго привязаны к криптографическим утверждениям JWT. На уровне хранения защита Cloud Spanner на уровне строк и партиционирование таблиц гарантируют, что поставщики и 3PL-операторы не могут видеть или запрашивать данные других арендаторов.',
    highlights: [
      'Криптографический периметр JWT',
      'Изоляция строк в Cloud Spanner',
      'Нулевая утечка межклиентских данных',
    ],
  },
  {
    id: 'offline-connectivity',
    category: 'Надежность',
    tag: 'OFFLINE FIRST',
    question: 'Что происходит при потере мобильной связи во время рейса водителя или на удаленных складах?',
    answer:
      'Мобильные приложения Pegasus для водителей и складов работают по принципу offline-first с локальными очередями SQLite. Сканирования штрихкодов, подписи и GPS-отметки сохраняются локально с криптографическими метками времени. При восстановлении связи очередь идемпотентно синхронизируется через outbox без потерь и дубликатов.',
    highlights: [
      'Локальная очередь транзакций SQLite',
      'Идемпотентная синхронизация outbox',
      'Гарантия сохранности 100% данных',
    ],
  },
  {
    id: 'concurrency-locks',
    category: 'Архитектура',
    tag: 'СТЕЙТ-МАШИНА',
    question: 'Как Pegasus предотвращает состояние гонки и перепродажу остатков при пиковых нагрузках?',
    answer:
      'Все изменения состояния проходят строгий атомарный пайплайн: Проверка JWT → Захват распределенного лока → Мутация стейт-машины → Коммит в Outbox → Рассылка через WebSocket. Конкурентные заказы на один артикул валидируются последовательно при коммите; повторные бронирования мгновенно отклоняются с понятной причиной.',
    highlights: [
      'Атомарный пайплайн мутаций',
      'Распределенная блокировка',
      'Гарантия нулевого оверселлинга',
    ],
  },
  {
    id: 'deployment-timeline',
    category: 'Внедрение',
    tag: 'ОНБОРДИНГ',
    question: 'Каковы типичные сроки внедрения для корпоративной распределительной сети?',
    answer:
      'Поэтапное развертывание обычно занимает от 60 до 90 дней. Фаза 1 (дни 1–30): настройка коннекторов ERP/WMS и теневой учет. Фаза 2 (дни 31–60): запуск диспетчерских досок и мобильных приложений на пилотных узлах. Фаза 3 (дни 61–90): активация финансовых шлюзов и перевод полного трафика под контролем команды 24/7.',
    highlights: [
      'Типичное внедрение за 60–90 дней',
      '3-фазный безопасный переход',
      'Круглосуточная инженерная поддержка',
    ],
  },
  {
    id: 'hardware-scanners',
    category: 'Интеграция',
    tag: 'ТСД И IOT',
    question: 'Поддерживает ли Pegasus промышленные терминалы сбора данных и датчики температурного режима?',
    answer:
      'Да. Pegasus включает нативные биндинги для сканеров Zebra, Honeywell и Datalogic. Для холодовой цепи телеметрия датчиков температуры BLE и IoT принимается в реальном времени, автоматически запуская сценарий реагирования при выходе температуры за допустимые границы.',
    highlights: [
      'Нативная поддержка ТСД Zebra и Honeywell',
      'Телеметрия BLE и IoT датчиков',
      'Автоматические сценарии термоконтроля',
    ],
  },
];

export default function O9EnterpriseFaq({
  kicker,
  title,
  description,
  items,
  allowMultiple = false,
  className,
  searchPlaceholder,
  defaultOpenIndex = 0,
}: O9EnterpriseFaqProps) {
  const { language } = useLanguage();
  const searchInputId = useId();

  const resolvedKicker =
    kicker ??
    (language === 'ru' ? 'ВОПРОСЫ И ОТВЕТЫ ПО АРХИТЕКТУРЕ' : 'ENTERPRISE ARCHITECTURE FAQ');

  const resolvedTitle =
    title ??
    (language === 'ru'
      ? 'Часто задаваемые вопросы об интеграции и надежности'
      : 'Frequently Answered Questions on Integration & Reliability');

  const resolvedDescription =
    description ??
    (language === 'ru'
      ? 'Технические детали интеграции с существующими ERP/WMS, изоляции арендаторов, отказоустойчивости и сроков внедрения.'
      : 'Technical details regarding legacy ERP/WMS overlays, cryptographic tenant isolation, fault-tolerant offline sync, and rollout timelines.');

  const resolvedPlaceholder =
    searchPlaceholder ??
    (language === 'ru' ? 'Поиск по архитектуре, ERP, безопасности...' : 'Search architecture, ERP, security, offline...');

  const allCategoryLabel = language === 'ru' ? 'Все вопросы' : 'All Topics';

  // Base list of items
  const baseItems = useMemo(() => {
    if (items && items.length > 0) return items;
    return language === 'ru' ? DEFAULT_ENTERPRISE_FAQS_RU : DEFAULT_ENTERPRISE_FAQS_EN;
  }, [items, language]);

  // Extract distinct categories
  const categories = useMemo(() => {
    const set = new Set<string>();
    baseItems.forEach((item) => {
      if (item.category) set.add(item.category);
    });
    return [allCategoryLabel, ...Array.from(set)];
  }, [baseItems, allCategoryLabel]);

  const [selectedCategory, setSelectedCategory] = useState<string>(allCategoryLabel);
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [openIds, setOpenIds] = useState<Set<string>>(() => {
    const initial = new Set<string>();
    if (defaultOpenIndex >= 0 && baseItems[defaultOpenIndex]) {
      initial.add(baseItems[defaultOpenIndex].id);
    }
    return initial;
  });

  // Filter items by category and search term
  const filteredItems = useMemo(() => {
    const q = searchQuery.trim().toLowerCase();
    return baseItems.filter((item) => {
      const matchesCategory =
        selectedCategory === allCategoryLabel || item.category === selectedCategory;
      if (!matchesCategory) return false;
      if (!q) return true;
      const inQuestion = item.question.toLowerCase().includes(q);
      const inAnswer = item.answer.toLowerCase().includes(q);
      const inCategory = item.category?.toLowerCase().includes(q) ?? false;
      const inTag = item.tag?.toLowerCase().includes(q) ?? false;
      return inQuestion || inAnswer || inCategory || inTag;
    });
  }, [baseItems, selectedCategory, allCategoryLabel, searchQuery]);

  const toggleItem = (id: string) => {
    setOpenIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        if (!allowMultiple) {
          next.clear();
        }
        next.add(id);
      }
      return next;
    });
  };

  return (
    <section
      className={cn(
        'w-full pt-12 md:pt-16 border-t border-zinc-200 dark:border-white/10 transition-colors',
        className
      )}
      aria-label={resolvedTitle}
    >
      {/* Header Eyebrow & Headline */}
      <div className="flex flex-col gap-4 max-w-4xl">
        <div className="inline-flex items-center gap-2">
          <span className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" aria-hidden />
          <span className="font-mono text-[10px] sm:text-[11px] uppercase tracking-widest text-emerald-600 dark:text-emerald-400 font-medium">
            {resolvedKicker}
          </span>
        </div>

        <h2 className="text-2xl sm:text-3xl md:text-4xl lg:text-5xl font-semibold tracking-tight text-zinc-900 dark:text-white leading-[1.12]">
          {resolvedTitle}
        </h2>

        {resolvedDescription && (
          <p className="text-sm sm:text-base text-zinc-600 dark:text-zinc-400 leading-relaxed max-w-3xl">
            {resolvedDescription}
          </p>
        )}
      </div>

      {/* Controls Bar: Category Tabs & Real-Time Search */}
      <div className="mt-8 flex flex-col md:flex-row md:items-center justify-between gap-4">
        {/* Category Tabs */}
        {categories.length > 1 && (
          <div
            className="flex flex-wrap items-center gap-1.5 p-1 bg-zinc-100 dark:bg-white/[0.04] border border-zinc-200 dark:border-white/10 rounded-none"
            role="tablist"
            aria-label="FAQ categories"
          >
            {categories.map((cat) => {
              const isSelected = selectedCategory === cat;
              return (
                <button
                  key={cat}
                  type="button"
                  role="tab"
                  aria-selected={isSelected}
                  onClick={() => setSelectedCategory(cat)}
                  className={cn(
                    'px-3 py-1.5 font-mono text-[10px] sm:text-[11px] uppercase tracking-wider transition-all duration-200 cursor-pointer',
                    isSelected
                      ? 'bg-zinc-900 text-white dark:bg-emerald-500 dark:text-black font-semibold shadow-sm'
                      : 'text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-white hover:bg-white/50 dark:hover:bg-white/5'
                  )}
                >
                  {cat}
                </button>
              );
            })}
          </div>
        )}

        {/* Search Filter Input */}
        <div className="relative w-full md:w-80">
          <label htmlFor={searchInputId} className="sr-only">
            {resolvedPlaceholder}
          </label>
          <Search
            className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-400 dark:text-zinc-500 pointer-events-none"
            aria-hidden
          />
          <input
            id={searchInputId}
            type="search"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder={resolvedPlaceholder}
            className="w-full pl-9 pr-8 py-2 text-xs font-mono bg-white dark:bg-zinc-950/80 border border-zinc-200 dark:border-white/10 text-zinc-900 dark:text-zinc-100 placeholder:text-zinc-400 dark:placeholder:text-zinc-600 focus:outline-none focus:border-emerald-500 dark:focus:border-emerald-400 transition-colors rounded-none"
          />
          {searchQuery && (
            <button
              type="button"
              onClick={() => setSearchQuery('')}
              className="absolute right-2.5 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200 p-0.5"
              aria-label="Clear search query"
            >
              <X className="w-3.5 h-3.5" />
            </button>
          )}
        </div>
      </div>

      {/* Search results summary count */}
      {searchQuery.trim() && (
        <div className="mt-3 flex items-center justify-between text-xs font-mono text-zinc-500 dark:text-zinc-400">
          <span>
            {language === 'ru'
              ? `Найдено вопросов: ${filteredItems.length} из ${baseItems.length}`
              : `Showing ${filteredItems.length} of ${baseItems.length} questions`}
          </span>
          <button
            type="button"
            onClick={() => setSearchQuery('')}
            className="text-emerald-600 dark:text-emerald-400 hover:underline"
          >
            {language === 'ru' ? 'Сбросить поиск' : 'Reset search'}
          </button>
        </div>
      )}

      {/* Accordion Items List */}
      <div className="mt-6 flex flex-col divide-y divide-zinc-200 dark:divide-white/10 border-y border-zinc-200 dark:border-white/10">
        {filteredItems.length === 0 ? (
          <div className="py-12 px-4 text-center flex flex-col items-center justify-center gap-3">
            <HelpCircle className="w-8 h-8 text-zinc-400 dark:text-zinc-600" />
            <p className="text-sm font-mono text-zinc-600 dark:text-zinc-400">
              {language === 'ru'
                ? 'По вашему запросу вопросов не найдено.'
                : 'No questions matched your search criteria.'}
            </p>
            <button
              type="button"
              onClick={() => {
                setSearchQuery('');
                setSelectedCategory(allCategoryLabel);
              }}
              className="font-mono text-xs text-emerald-600 dark:text-emerald-400 hover:underline uppercase tracking-wider"
            >
              {language === 'ru' ? 'Показать все вопросы' : 'Show all questions'}
            </button>
          </div>
        ) : (
          filteredItems.map((item, idx) => {
            const isOpen = openIds.has(item.id);
            const triggerId = `faq-trigger-${item.id}`;
            const contentId = `faq-content-${item.id}`;
            const indexNumber = String(idx + 1).padStart(2, '0');

            return (
              <article
                key={item.id}
                className={cn(
                  'relative transition-all duration-200 group',
                  isOpen
                    ? 'bg-zinc-50/80 dark:bg-white/[0.02] border-l-2 border-emerald-500 pl-4 sm:pl-6 -ml-[2px]'
                    : 'hover:bg-zinc-50/50 dark:hover:bg-white/[0.01]'
                )}
              >
                {/* Tactical Corner Bracket on Open */}
                {isOpen && (
                  <>
                    <span
                      className="absolute -top-1 -right-1 w-2 h-2 border-t border-r border-emerald-500/60 pointer-events-none"
                      aria-hidden
                    />
                    <span
                      className="absolute -bottom-1 -right-1 w-2 h-2 border-b border-r border-emerald-500/60 pointer-events-none"
                      aria-hidden
                    />
                  </>
                )}

                {/* Accordion Trigger Header */}
                <button
                  type="button"
                  id={triggerId}
                  aria-expanded={isOpen}
                  aria-controls={contentId}
                  onClick={() => toggleItem(item.id)}
                  className="w-full py-4 sm:py-5 md:py-6 flex items-start sm:items-center justify-between gap-3 sm:gap-4 text-left cursor-pointer focus:outline-none focus-visible:ring-1 focus-visible:ring-emerald-500"
                >
                  <div className="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-4 flex-1 pr-1 sm:pr-2 min-w-0">
                    {/* Index & Category Badges */}
                    <div className="flex items-center gap-2 shrink-0">
                      <span className="font-mono text-xs text-zinc-400 dark:text-zinc-500 group-hover:text-emerald-600 dark:group-hover:text-emerald-400 transition-colors">
                        [{indexNumber}]
                      </span>
                      {item.tag && (
                        <span className="font-mono text-[9px] uppercase tracking-wider px-1.5 py-0.5 border border-zinc-300 dark:border-white/10 text-zinc-600 dark:text-zinc-400 bg-white dark:bg-white/5 shrink-0">
                          {item.tag}
                        </span>
                      )}
                    </div>

                    {/* Question Text */}
                    <h3
                      className={cn(
                        'text-sm sm:text-base md:text-lg lg:text-xl font-medium tracking-tight transition-colors duration-200 break-words',
                        isOpen
                          ? 'text-emerald-600 dark:text-emerald-400 font-semibold'
                          : 'text-zinc-900 dark:text-zinc-100 group-hover:text-emerald-600 dark:group-hover:text-emerald-400'
                      )}
                    >
                      {item.question}
                    </h3>
                  </div>

                  {/* Expand / Collapse Icon indicator */}
                  <div className="shrink-0 pt-1 sm:pt-0">
                    <span
                      className={cn(
                        'inline-flex items-center justify-center w-7 h-7 border transition-all duration-300',
                        isOpen
                          ? 'border-emerald-500 bg-emerald-500 text-black dark:text-black rotate-180 shadow-[0_0_10px_rgba(16,185,129,0.2)]'
                          : 'border-zinc-300 dark:border-white/20 text-zinc-500 dark:text-zinc-400 group-hover:border-emerald-500 group-hover:text-emerald-500'
                      )}
                      aria-hidden
                    >
                      <ChevronDown className="w-4 h-4" />
                    </span>
                  </div>
                </button>

                {/* Accordion Content Panel */}
                <AnimatePresence initial={false}>
                  {isOpen && (
                    <motion.div
                      id={contentId}
                      role="region"
                      aria-labelledby={triggerId}
                      initial={{ height: 0, opacity: 0 }}
                      animate={{ height: 'auto', opacity: 1 }}
                      exit={{ height: 0, opacity: 0 }}
                      transition={{ duration: 0.28, ease: [0.16, 1, 0.3, 1] }}
                      className="overflow-hidden"
                    >
                      <div className="pb-5 sm:pb-6 pt-1 sm:pl-8 md:pl-10 pr-2 sm:pr-8 flex flex-col gap-4">
                        {/* Answer Body */}
                        <p className="text-sm sm:text-base leading-relaxed text-zinc-700 dark:text-zinc-300 break-words">
                          {item.answer}
                        </p>

                        {/* Optional Tactical Highlights */}
                        {item.highlights && item.highlights.length > 0 && (
                          <div className="mt-2 pt-3 border-t border-zinc-200/80 dark:border-white/5 flex flex-wrap gap-x-4 sm:gap-x-6 gap-y-2">
                            {item.highlights.map((hl) => (
                              <div
                                key={hl}
                                className="inline-flex items-start sm:items-center gap-2 font-mono text-[11px] text-zinc-600 dark:text-zinc-400 break-words"
                              >
                                <CheckCircle2 className="w-3.5 h-3.5 text-emerald-500 shrink-0 mt-0.5 sm:mt-0" />
                                <span>{hl}</span>
                              </div>
                            ))}
                          </div>
                        )}
                      </div>
                    </motion.div>
                  )}
                </AnimatePresence>
              </article>
            );
          })
        )}
      </div>
    </section>
  );
}
