'use client';

import {
 Layers,
 TrendingDown,
 Network,
 Gauge,
 ShieldCheck,
 Columns2,
 Repeat,
 UserCheck,
 Activity,
 Scale,
} from 'lucide-react';
import { useLanguage } from '@/app/context/LanguageContext';

type TrapItem = {
 id: string;
 titleEn: string;
 titleRu: string;
 descEn: string;
 descRu: string;
 icon: React.ComponentType<{ className?: string }>;
};

const TRAPS: TrapItem[] = [
 {
 id: 'layer',
 titleEn: 'Adding AI instead of rebuilding around it',
 titleRu: 'Встраивание ИИ вместо перестройки процессов',
 descEn: 'If AI is just a layer, nothing really changes.',
 descRu: 'Если ИИ — это просто надстройка, реальных изменений не происходит.',
 icon: Layers,
 },
 {
 id: 'pilots',
 titleEn: 'Living in pilots',
 titleRu: 'Застревание в пилотах',
 descEn: "If it's not in production, it's not creating value.",
 descRu: 'Если решение не работает в проде, оно не создает ценности.',
 icon: TrendingDown,
 },
 {
 id: 'orchestration',
 titleEn: 'No orchestration layer',
 titleRu: 'Отсутствие слоя оркестрации',
 descEn: 'Without a control layer, everything becomes disconnected and hard to manage.',
 descRu: 'Без центрального слоя управления система фрагментируется.',
 icon: Network,
 },
 {
 id: 'data',
 titleEn: 'Slow, static data',
 titleRu: 'Медленные статичные данные',
 descEn: 'AI is only as good as the speed of your data. Batch = lag.',
 descRu: 'ИИ эффективен ровно настолько, насколько быстры данные. Пакетная выгрузка = отставание.',
 icon: Gauge,
 },
 {
 id: 'governance',
 titleEn: 'Governance as an afterthought',
 titleRu: 'Управление как запоздалая мысль',
 descEn: 'Trust, risk, and control have to be built in - not added later.',
 descRu: 'Доверие, безопасность и контроль должны быть заложены изначально.',
 icon: ShieldCheck,
 },
 {
 id: 'one-size',
 titleEn: 'One-size-fits-all thinking',
 titleRu: 'Универсальный шаблон мышления',
 descEn: 'Forcing one model or tool to do everything limits performance.',
 descRu: 'Попытка закрыть все задачи одной моделью снижает эффективность.',
 icon: Columns2,
 },
 {
 id: 'old-ways',
 titleEn: 'Keeping old ways of working',
 titleRu: 'Сохранение старых рабочих привычек',
 descEn: "AI doesn't just improve functions - it changes how they run.",
 descRu: 'ИИ не просто оптимизирует функции — он меняет сам принцип их работы.',
 icon: Repeat,
 },
 {
 id: 'humans',
 titleEn: 'Humans doing the work',
 titleRu: 'Люди вместо контроля',
 descEn: 'The shift is from doing to overseeing. Missing that limits scale.',
 descRu: 'Смысл в переходе от рутины к надзору. Игнорирование этого ограничивает масштаб.',
 icon: UserCheck,
 },
 {
 id: 'trends',
 titleEn: 'Chasing trends',
 titleRu: 'Погоня за трендами',
 descEn: "If it doesn't tie to margin or growth, it's noise.",
 descRu: 'Если решение не связано с маржой или ростом — это просто шум.',
 icon: Activity,
 },
 {
 id: 'owner',
 titleEn: 'No clear owner',
 titleRu: 'Отсутствие ответственного',
 descEn: 'Without accountability, the stack stalls.',
 descRu: 'Без персональной ответственности стек останавливается.',
 icon: Scale,
 },
];

export type O9TrapsToAvoidProps = {
 eyebrow?: string;
 title?: string;
};

export default function O9TrapsToAvoid({
 eyebrow,
 title,
}: O9TrapsToAvoidProps) {
 const { language } = useLanguage();
 const isRu = language === 'ru';

 const resolvedEyebrow = eyebrow ?? (isRu ? 'ОШИБКИ ВНЕДРЕНИЯ' : 'TRAPS TO AVOID');
 const resolvedTitle = title ?? (isRu ? 'Ошибки, которых следует избегать' : 'Traps to Avoid');

 return (
 <section className="w-full py-12 md:py-16">
 {/* Section Header */}
 <div className="mb-8 md:mb-12">
 <p className="font-mono text-[10px] uppercase tracking-[0.2em] text-zinc-500 dark:text-zinc-400 mb-3">
 {resolvedEyebrow}
 </p>
 <h2 className="text-3xl md:text-4xl lg:text-5xl font-normal tracking-tight text-zinc-900 dark:text-white">
 {resolvedTitle}
 </h2>
 </div>

 {/* 10-Card Adaptive Grid (5 cols x 2 rows on desktop) */}
 <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4 md:gap-5">
 {TRAPS.map((trap) => {
 const Icon = trap.icon;
 return (
 <article
 key={trap.id}
 className="border border-black/8 bg-[#F7F7F6] dark:border-white/10 dark:bg-black p-5 rounded-none flex flex-col justify-between min-h-[200px] hover:border-black/20 dark:hover:border-white/25 dark:shadow-none transition-all duration-300 group"
 >
 <div>
 <h3 className="text-sm md:text-[15px] font-medium tracking-tight text-zinc-900 dark:text-white leading-snug mb-2.5">
 {isRu ? trap.titleRu : trap.titleEn}
 </h3>
 <p className="text-xs leading-relaxed text-zinc-600 dark:text-white/55">
 {isRu ? trap.descRu : trap.descEn}
 </p>
 </div>

 <div className="pt-6">
 <Icon className="w-5 h-5 text-zinc-400 group-hover:text-black dark:text-white/40 dark:group-hover:text-white transition-colors" />
 </div>
 </article>
 );
 })}
 </div>
 </section>
 );
}
