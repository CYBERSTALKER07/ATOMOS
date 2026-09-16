'use client';

import Link from 'next/link';
import {
  ArrowUpRight,
  ClipboardList,
  TrendingUp,
  DollarSign,
  ChevronRight,
  Zap,
} from 'lucide-react';
import { useLanguage } from '@/app/context/LanguageContext';
import { useTheme } from '@/app/context/ThemeContext';

/* =========================================================================
   Card 1 Visual: Connected Node Flow Architecture Diagram
   Supply Chain -> Central Green Glowing AI Node -> Finance -> Customer Ops
   ========================================================================= */
function ArchitectureFlowDiagram({ isRu, isLight }: { isRu: boolean; isLight: boolean }) {
  const strokeColor = isLight ? 'rgba(0, 0, 0, 0.22)' : 'rgba(255, 255, 255, 0.25)';

  return (
    <div className="relative w-full max-w-[320px] h-[200px] mx-auto flex items-center justify-center select-none">
      {/* Dashed connector SVG lines */}
      <svg
        className="absolute inset-0 w-full h-full pointer-events-none"
        viewBox="0 0 320 200"
        fill="none"
        preserveAspectRatio="xMidYMid meet"
      >
        {/* Left dashed line entering Supply Chain */}
        <line
          x1="0"
          y1="102"
          x2="24"
          y2="102"
          stroke={strokeColor}
          strokeWidth="1.5"
          strokeDasharray="4 4"
        />

        {/* Up from Supply Chain (x: 75), turning right into circle node */}
        <path
          d="M 75 88 L 75 42 L 138 42"
          stroke={strokeColor}
          strokeWidth="1.5"
          strokeDasharray="4 4"
        />

        {/* Right from circle node (x: 182), turning down into Finance node */}
        <path
          d="M 182 42 L 245 42 L 245 74"
          stroke={strokeColor}
          strokeWidth="1.5"
          strokeDasharray="4 4"
        />

        {/* Straight down from Finance to Customer Ops */}
        <line
          x1="245"
          y1="104"
          x2="245"
          y2="138"
          stroke={strokeColor}
          strokeWidth="1.5"
          strokeDasharray="4 4"
        />

        {/* Exiting right from Customer Ops */}
        <line
          x1="300"
          y1="154"
          x2="320"
          y2="154"
          stroke={strokeColor}
          strokeWidth="1.5"
          strokeDasharray="4 4"
        />
      </svg>

      {/* Node 1: Supply Chain (stays high-contrast black badge in both modes matching reference) */}
      <div className="absolute left-[24px] top-[88px] z-10">
        <div className="px-3 py-1.5 rounded-md bg-black/90 border border-white/20 text-white text-[11px] font-semibold tracking-tight shadow-lg whitespace-nowrap">
          {isRu ? 'Цепь поставок' : 'Supply Chain'}
        </div>
      </div>

      {/* Center Monochrome Node */}
      <div className="absolute left-[160px] top-[24px] -translate-x-1/2 z-10">
        <div className="w-10 h-10 rounded-full bg-white dark:bg-black border border-black dark:border-white flex items-center justify-center shadow-[0_0_15px_rgba(0,0,0,0.1)] dark:shadow-[0_0_20px_rgba(255,255,255,0.25)]">
          <span className="font-mono text-sm font-bold text-black dark:text-white leading-none select-none">&gt;_</span>
        </div>
      </div>

      {/* Node 2: Finance */}
      <div className="absolute right-[44px] top-[74px] z-10">
        <div className="px-3 py-1.5 rounded-md bg-black/90 border border-white/20 text-white text-[11px] font-semibold tracking-tight shadow-lg whitespace-nowrap">
          {isRu ? 'Финансы' : 'Finance'}
        </div>
      </div>

      {/* Node 3: Customer Ops */}
      <div className="absolute right-[20px] top-[138px] z-10">
        <div className="px-3 py-1.5 rounded-md bg-black/90 border border-white/20 text-white text-[11px] font-semibold tracking-tight shadow-lg whitespace-nowrap">
          {isRu ? 'Операции с клиентами' : 'Customer Ops'}
        </div>
      </div>
    </div>
  );
}

/* =========================================================================
   Card 2 Visual: Stacked AI Operations Agents
   Planning Agent, Forecasting Agent, Finance Agent
   ========================================================================= */
function AgentStackVisual({ isRu, isLight }: { isRu: boolean; isLight: boolean }) {
  const cardCls = isLight
    ? 'bg-white border border-black/10 rounded-xl px-3.5 py-2.5 flex items-center gap-3 shadow-[0_2px_8px_rgba(0,0,0,0.06)] hover:border-black/25 transition-colors group'
    : 'bg-[#14141F] border border-white/10 rounded-xl px-3.5 py-2.5 flex items-center gap-3 shadow-[0_4px_16px_rgba(0,0,0,0.4)] hover:border-white/30 transition-colors group';

  const iconCls2 = isLight
    ? 'w-7 h-7 rounded-lg bg-black/5 border border-black/10 flex items-center justify-center text-zinc-700 shrink-0'
    : 'w-7 h-7 rounded-lg bg-white/5 border border-white/10 flex items-center justify-center text-white/80 shrink-0';

  const textCls = isLight
    ? 'text-xs font-medium text-zinc-800 group-hover:text-black'
    : 'text-xs font-medium text-white/90 group-hover:text-white';

  return (
    <div className="w-full max-w-[230px] mx-auto flex flex-col gap-2.5 select-none">
      {/* Agent 1: Planning Agent */}
      <div className={cardCls}>
        <div className={iconCls2}>
          <ClipboardList className="w-3.5 h-3.5" />
        </div>
        <span className={textCls}>
          {isRu ? 'Агент планирования' : 'Planning Agent'}
        </span>
      </div>

      {/* Agent 2: Forecasting Agent */}
      <div className={cardCls}>
        <div className={iconCls2}>
          <TrendingUp className="w-3.5 h-3.5" />
        </div>
        <span className={textCls}>
          {isRu ? 'Агент прогнозирования' : 'Forecasting Agent'}
        </span>
      </div>

      {/* Agent 3: Finance Agent */}
      <div className={cardCls}>
        <div className={iconCls2}>
          <DollarSign className="w-3.5 h-3.5" />
        </div>
        <span className={textCls}>
          {isRu ? 'Финансовый агент' : 'Finance Agent'}
        </span>
      </div>
    </div>
  );
}

/* =========================================================================
   Card 3 Visual: KPI Impact Metric Pill
   Gross Margin +6.2%
   ========================================================================= */
function OutcomeMetricVisual({ isRu }: { isRu: boolean }) {
  return (
    <div className="w-full max-w-[260px] mx-auto select-none">
      <div className="bg-black border border-white/20 rounded-xl px-5 py-4 flex items-center justify-between shadow-[0_8px_32px_rgba(0,0,0,0.6)] relative overflow-hidden group hover:border-white/30 transition-all">
        {/* Ambient subtle monochrome glow */}
        <div className="absolute -right-6 -bottom-6 w-24 h-24 bg-white/10 rounded-full blur-2xl pointer-events-none" />

        <span className="text-xs font-mono font-medium tracking-wider uppercase text-white/60">
          {isRu ? 'Валовая маржа' : 'Gross Margin'}
        </span>
        <span className="text-2xl font-bold font-mono tracking-tight text-white drop-shadow-[0_0_16px_rgba(255,255,255,0.35)]">
          +6.2%
        </span>
      </div>
    </div>
  );
}

/* =========================================================================
   Main Section Component: Adaptive AI Outcomes & Architecture
   ========================================================================= */
export type O9InsightCardsProps = {
  eyebrow?: string;
  title?: string;
  showButtons?: boolean;
};

export default function O9InsightCards({
  eyebrow,
  title,
  showButtons = true,
}: O9InsightCardsProps) {
  const { language } = useLanguage();
  const { resolvedTheme } = useTheme();
  const isLight = resolvedTheme === 'light';
  const isRu = language === 'ru';

  const resolvedEyebrow = eyebrow ?? (isRu ? 'АРХИТЕКТУРА ИИ' : 'AI-NATIVE ARCHITECTURE');
  const resolvedTitle = title ?? (isRu ? 'ИИ-платформа с самого начала' : 'AI First from the Outset');

  return (
    <section className="w-full py-12 md:py-16">
      {/* Top Header Row with Title and Action CTAs */}
      <div className="flex flex-col md:flex-row md:items-end justify-between gap-6 mb-8 md:mb-12">
        <div>
          <p className="font-mono text-[10px] uppercase tracking-[0.2em] text-zinc-500 dark:text-zinc-400 mb-3">
            {resolvedEyebrow}
          </p>
          <h2 className="text-3xl md:text-4xl lg:text-5xl font-normal tracking-tight text-zinc-900 dark:text-white">
            {resolvedTitle}
          </h2>
        </div>

        {showButtons ? (
          <div className="flex items-center gap-3 shrink-0">
            <Link
              href="/capabilities/payment-confidence"
              className={`inline-flex items-center gap-1.5 px-4 py-2 text-xs font-mono uppercase tracking-wider rounded-lg transition-all ${
                isLight
                  ? 'text-zinc-800 bg-white border border-black/10 hover:border-black/25 hover:bg-zinc-50 shadow-sm'
                  : 'text-white/80 bg-[#121218] border border-white/15 hover:border-white/30 hover:text-white hover:bg-white/5'
              }`}
            >
              <span>{isRu ? 'Метрики' : 'View Metrics'}</span>
              <ArrowUpRight className="w-3.5 h-3.5 text-zinc-900 dark:text-white" />
            </Link>
            <Link
              href="/solutions"
              className={`inline-flex items-center gap-1.5 px-4 py-2 text-xs font-mono uppercase tracking-wider rounded-lg transition-all ${
                isLight
                  ? 'text-zinc-800 bg-white border border-black/10 hover:border-black/25 hover:bg-zinc-50 shadow-sm'
                  : 'text-white/80 bg-[#121218] border border-white/15 hover:border-white/30 hover:text-white hover:bg-white/5'
              }`}
            >
              <span>{isRu ? 'Подробнее' : 'Learn More'}</span>
              <ChevronRight className="w-3.5 h-3.5 text-zinc-400 dark:text-white/50" />
            </Link>
          </div>
        ) : null}
      </div>

      {/* 3-Column Bento Grid */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 lg:gap-8">
        {/* Column 1: Connected Flow Diagram */}
        <article className={`border rounded-2xl p-6 lg:p-7 flex flex-col justify-between transition-all duration-300 ${
          isLight
            ? 'border-black/8 bg-white shadow-[0_2px_12px_rgba(0,0,0,0.04)] hover:border-black/20'
            : 'border-white/10 bg-[#0B0B10] hover:border-white/20'
        }`}>
          <div className={`rounded-xl p-4 min-h-[220px] flex items-center justify-center relative overflow-hidden border ${
            isLight ? 'bg-[#F4F4F6] border-black/5' : 'bg-[#07070B] border-white/8'
          }`}>
            <ArchitectureFlowDiagram isRu={isRu} isLight={isLight} />
          </div>
          <div className="mt-6 flex flex-col flex-1">
            <h3 className="text-lg font-semibold tracking-tight text-zinc-900 dark:text-white mb-2.5">
              {isRu
                ? 'Повышайте маржу. Действуйте в реальном времени.'
                : 'Increase Margin. Operate in Real Time.'}
            </h3>
            <p className="text-sm leading-relaxed text-zinc-600 dark:text-white/60">
              {isRu
                ? 'ИИ в основе, а не сбоку. Мы внедряем его в цепочки поставок, финансы и работу с клиентами, чтобы ваш бизнес работал на ИИ. Никаких бесконечных пилотов. Промышленные системы. Измеримый результат за недели.'
                : 'AI at the core, not bolted on. We embed it in your supply chain, finance, and customer operations, so your business runs on AI, not around it. No pilots. No endless experimentation. Production-grade systems. Measurable impact in weeks.'}
            </p>
          </div>
        </article>

        {/* Column 2: Stacked AI Agents */}
        <article className={`border rounded-2xl p-6 lg:p-7 flex flex-col justify-between transition-all duration-300 ${
          isLight
            ? 'border-black/8 bg-white shadow-[0_2px_12px_rgba(0,0,0,0.04)] hover:border-black/20'
            : 'border-white/10 bg-[#0B0B10] hover:border-white/20'
        }`}>
          <div className={`rounded-xl p-4 min-h-[220px] flex items-center justify-center relative overflow-hidden border ${
            isLight ? 'bg-[#F4F4F6] border-black/5' : 'bg-[#07070B] border-white/8'
          }`}>
            <AgentStackVisual isRu={isRu} isLight={isLight} />
          </div>
          <div className="mt-6 flex flex-col flex-1">
            <h3 className="text-lg font-semibold tracking-tight text-zinc-900 dark:text-white mb-2.5">
              {isRu
                ? 'От экспериментов к операциям на базе ИИ.'
                : 'From AI Experiments to AI-Run Operations.'}
            </h3>
            <p className="text-sm leading-relaxed text-zinc-600 dark:text-white/60">
              {isRu
                ? 'ИИ-агенты ведут процессы: планирование, прогнозирование, финансы и клиентские сервисы — непрерывно и в реальном времени. Ваша команда управляет стратегией, автоматизация исполняет задачи.'
                : 'AI agents run the work. Planning, forecasting, finance, customer engagement—continuous, real-time. Your teams own judgment, strategy, and trust. Execution that\'s automated, optimized, always learning. Not an upgrade. A new operating model.'}
            </p>
          </div>
        </article>

        {/* Column 3: Outcomes KPI Metric */}
        <article className={`border rounded-2xl p-6 lg:p-7 flex flex-col justify-between transition-all duration-300 ${
          isLight
            ? 'border-black/8 bg-white shadow-[0_2px_12px_rgba(0,0,0,0.04)] hover:border-black/20'
            : 'border-white/10 bg-[#0B0B10] hover:border-white/20'
        }`}>
          <div className={`rounded-xl p-4 min-h-[220px] flex items-center justify-center relative overflow-hidden border ${
            isLight ? 'bg-[#F4F4F6] border-black/5' : 'bg-[#07070B] border-white/8'
          }`}>
            <OutcomeMetricVisual isRu={isRu} />
          </div>
          <div className="mt-6 flex flex-col flex-1">
            <h3 className="text-lg font-semibold tracking-tight text-zinc-900 dark:text-white mb-2.5">
              {isRu
                ? 'Создано вокруг результатов, а не кейсов.'
                : 'Built Around Outcomes, Not Use Cases.'}
            </h3>
            <p className="text-sm leading-relaxed text-zinc-600 dark:text-white/60">
              {isRu
                ? 'Результат прежде всего — маржинальность, скорость, надежность. Мы перестраиваем ключевые функции вокруг ИИ с привязкой к ROI. Запуск первых ИИ-процессов за 6–8 недель.'
                : 'Outcomes first—margin, speed, resilience, loyalty. Then we rebuild your core functions around AI. Measurable. Scalable. Tied to ROI. Run your business on AI. Your first AI workflows go live in 6–8 weeks—the foundation for an AI-native enterprise.'}
            </p>
          </div>
        </article>
      </div>
    </section>
  );
}
