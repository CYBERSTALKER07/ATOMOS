'use client';

import React, { useState } from 'react';
import { Shield, Settings, RotateCcw, CheckCircle2, XCircle, AlertCircle, RefreshCw, KeyRound, Clock, ShieldCheck } from 'lucide-react';
import { useCookieConsent } from '@/app/context/CookieConsentContext';
import { useLanguage } from '@/app/context/LanguageContext';
import { useTheme } from '@/app/context/ThemeContext';

export default function CookiePolicyInteractivePanel() {
 const { consent, openModal, resetConsent, rejectNonEssential, isGPC } = useCookieConsent();
 const { language } = useLanguage();
 const { resolvedTheme } = useTheme();
 const isLight = resolvedTheme === 'light';
 const isRu = language === 'ru';
 const [copied, setCopied] = useState(false);

 const handleCopyId = () => {
 if (!consent?.consentId) return;
 navigator.clipboard.writeText(consent.consentId);
 setCopied(true);
 setTimeout(() => setCopied(false), 2000);
 };

 const formattedDate = consent?.timestamp
 ? new Date(consent.timestamp).toUTCString()
 : isRu
 ? 'Не зафиксировано (до подтверждения)'
 : 'Not recorded (pre-consent default)';

 return (
 <div
 className={`border rounded-none p-5 sm:p-6 transition-colors duration-200 ${
 isLight
 ? 'bg-zinc-50/80 border-black/10 text-zinc-900 '
 : 'bg-black border-white/10 text-white '
 }`}
 >
 {/* Header Bar */}
 <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-5 border-b border-black/10 dark:border-white/10">
 <div className="flex items-center gap-3">
 <div
 className={`w-9 h-9 rounded-none flex items-center justify-center shrink-0 ${
 isLight ? 'bg-zinc-100 text-zinc-900 border border-black/10' : 'bg-white/10 text-white border border-white/20'
 }`}
 >
 <ShieldCheck className="w-5 h-5" />
 </div>
 <div>
 <div className="flex items-center gap-2">
 <h3 className="text-sm font-semibold tracking-tight">
 {isRu ? 'Интерактивная панель управления согласием' : 'Live Consent Management Panel'}
 </h3>
 <span
 className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-none text-[10px] font-mono uppercase tracking-wider font-medium ${
 consent
 ? isLight
 ? 'bg-zinc-200 text-zinc-900'
 : 'bg-white/10 text-white border border-white/20'
 : isLight
 ? 'bg-zinc-100 text-zinc-700'
 : 'bg-white/5 text-zinc-400 border border-white/10'
 }`}
 >
 {consent
 ? isRu
 ? 'Настройки сохранены'
 : 'Configured'
 : isRu
 ? 'По умолчанию (Заблокировано)'
 : 'Pre-Consent Default'}
 </span>
 </div>
 <p className="text-xs text-zinc-500 dark:text-white/50 mt-0.5">
 {isRu
 ? 'Прямой контроль над телеметрией, идентификаторами и файлами cookie в реальном времени.'
 : 'Real-time telemetry, audit record inspection, and persistent preference controls.'}
 </p>
 </div>
 </div>

 {/* Action Controls */}
 <div className="flex flex-wrap items-center gap-2">
 <button
 onClick={openModal}
 className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-none text-xs font-medium bg-black dark:bg-white text-white dark:text-black hover:opacity-90 transition-opacity cursor-pointer "
 >
 <Settings className="w-3.5 h-3.5" />
 <span>{isRu ? 'Изменить настройки' : 'Change Preferences'}</span>
 </button>
 {consent && (
 <button
 onClick={resetConsent}
 className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-none text-xs font-medium border transition-colors cursor-pointer ${
 isLight
 ? 'border-red-200 text-red-700 bg-red-50 hover:bg-red-100'
 : 'border-red-500/30 text-red-400 bg-red-500/10 hover:bg-red-500/20'
 }`}
 title={isRu ? 'Отозвать согласие и сбросить все параметры' : 'Revoke consent and reset all cookies'}
 >
 <RotateCcw className="w-3.5 h-3.5" />
 <span>{isRu ? 'Отозвать согласие' : 'Revoke Consent'}</span>
 </button>
 )}
 </div>
 </div>

 {/* Telemetry & Audit Metadata Grid */}
 <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 py-5 border-b border-black/10 dark:border-white/10 text-xs font-mono">
 <div>
 <span className="text-[11px] text-zinc-500 dark:text-white/50 block uppercase tracking-wider mb-1 flex items-center gap-1">
 <KeyRound className="w-3 h-3" />
 {isRu ? 'ID согласия (Хеш)' : 'Consent Audit ID'}
 </span>
 {consent?.consentId ? (
 <div className="flex items-center gap-1.5">
 <span className="font-semibold truncate max-w-[140px] text-zinc-800 dark:text-white/90">
 {consent.consentId}
 </span>
 <button
 onClick={handleCopyId}
 className="text-[10px] text-zinc-400 hover:text-zinc-600 dark:hover:text-white underline cursor-pointer"
 >
 {copied ? (isRu ? 'Скопировано' : 'Copied') : (isRu ? 'Копировать' : 'Copy')}
 </button>
 </div>
 ) : (
 <span className="text-zinc-400 dark:text-white/40 italic">
 {isRu ? 'Генерируется при подтверждении' : 'Generated upon affirmative consent'}
 </span>
 )}
 </div>

 <div>
 <span className="text-[11px] text-zinc-500 dark:text-white/50 block uppercase tracking-wider mb-1 flex items-center gap-1">
 <Clock className="w-3 h-3" />
 {isRu ? 'Временная метка' : 'Recorded Timestamp'}
 </span>
 <span className="text-zinc-800 dark:text-white/90 truncate block" title={formattedDate}>
 {formattedDate}
 </span>
 </div>

 <div>
 <span className="text-[11px] text-zinc-500 dark:text-white/50 block uppercase tracking-wider mb-1">
 {isRu ? 'Версия спецификации' : 'Policy Version'}
 </span>
 <span className="text-zinc-800 dark:text-white/90">
 {consent?.version || 'v2026.1 (Active)'}
 </span>
 </div>

 <div>
 <span className="text-[11px] text-zinc-500 dark:text-white/50 block uppercase tracking-wider mb-1">
 {isRu ? 'Сигнал GPC (Браузер)' : 'Global Privacy Control'}
 </span>
 <span
 className={`inline-flex items-center gap-1 font-semibold ${
 isGPC ? 'text-zinc-800 dark:text-white' : 'text-zinc-500 dark:text-white/50'
 }`}
 >
 {isGPC ? (isRu ? 'Активен (Переопределяет маркетинг)' : 'Active (Opt-out enforced)') : (isRu ? 'Не обнаружен' : 'Not detected')}
 </span>
 </div>
 </div>

 {/* Granular Active Categories Matrix */}
 <div className="pt-5">
 <h4 className="text-xs font-mono uppercase tracking-wider text-zinc-500 dark:text-white/50 mb-3">
 {isRu ? 'Текущий статус категорий' : 'Category Authorization Status'}
 </h4>
 <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
 {/* Strictly Necessary */}
 <div
 className={`p-3 rounded-none border text-xs ${
 isLight ? 'bg-white border-black/10' : 'bg-black/30 border-white/10'
 }`}
 >
 <div className="flex items-center justify-between mb-1.5">
 <span className="font-semibold text-zinc-900 dark:text-white">
 {isRu ? 'Обязательные' : 'Strictly Necessary'}
 </span>
 <span className="text-zinc-900 dark:text-white flex items-center gap-1 text-[11px] font-mono font-medium">
 <CheckCircle2 className="w-3 h-3" />
 {isRu ? 'Активны' : 'Active'}
 </span>
 </div>
 <p className="text-[11px] text-zinc-500 dark:text-white/50 leading-relaxed">
 {isRu
 ? 'Сессии, CSRF, балансировка нагрузки. Отключение невозможно.'
 : 'Auth sessions, CSRF, load balancing. Cannot be disabled.'}
 </p>
 </div>

 {/* Functional */}
 <div
 className={`p-3 rounded-none border text-xs ${
 isLight ? 'bg-white border-black/10' : 'bg-black/30 border-white/10'
 }`}
 >
 <div className="flex items-center justify-between mb-1.5">
 <span className="font-semibold text-zinc-900 dark:text-white">
 {isRu ? 'Функциональные' : 'Functional'}
 </span>
 {consent?.categories.functional ? (
 <span className="text-zinc-900 dark:text-white flex items-center gap-1 text-[11px] font-mono font-medium">
 <CheckCircle2 className="w-3 h-3" />
 {isRu ? 'Разрешены' : 'Enabled'}
 </span>
 ) : (
 <span className="text-zinc-400 dark:text-white/40 flex items-center gap-1 text-[11px] font-mono">
 <XCircle className="w-3 h-3" />
 {isRu ? 'Заблокированы' : 'Blocked'}
 </span>
 )}
 </div>
 <p className="text-[11px] text-zinc-500 dark:text-white/50 leading-relaxed">
 {isRu
 ? 'Язык интерфейса, тема оформления, локальные фильтры.'
 : 'UI theme, language selection, client-side filters.'}
 </p>
 </div>

 {/* Analytics */}
 <div
 className={`p-3 rounded-none border text-xs ${
 isLight ? 'bg-white border-black/10' : 'bg-black/30 border-white/10'
 }`}
 >
 <div className="flex items-center justify-between mb-1.5">
 <span className="font-semibold text-zinc-900 dark:text-white">
 {isRu ? 'Аналитические' : 'Analytics & Telemetry'}
 </span>
 {consent?.categories.analytics ? (
 <span className="text-zinc-900 dark:text-white flex items-center gap-1 text-[11px] font-mono font-medium">
 <CheckCircle2 className="w-3 h-3" />
 {isRu ? 'Разрешены' : 'Enabled'}
 </span>
 ) : (
 <span className="text-zinc-400 dark:text-white/40 flex items-center gap-1 text-[11px] font-mono">
 <XCircle className="w-3 h-3" />
 {isRu ? 'Заблокированы' : 'Blocked'}
 </span>
 )}
 </div>
 <p className="text-[11px] text-zinc-500 dark:text-white/50 leading-relaxed">
 {isRu
 ? 'Анонимные метрики производительности и мониторинг сбоев.'
 : 'Aggregated error diagnostics and telemetry.'}
 </p>
 </div>

 {/* Marketing */}
 <div
 className={`p-3 rounded-none border text-xs ${
 isLight ? 'bg-white border-black/10' : 'bg-black/30 border-white/10'
 }`}
 >
 <div className="flex items-center justify-between mb-1.5">
 <span className="font-semibold text-zinc-900 dark:text-white">
 {isRu ? 'Маркетинг' : 'Marketing & Tracking'}
 </span>
 {isGPC ? (
 <span className="text-zinc-900 dark:text-white flex items-center gap-1 text-[11px] font-mono font-medium">
 <AlertCircle className="w-3 h-3" />
 GPC Opt-Out
 </span>
 ) : consent?.categories.marketing ? (
 <span className="text-zinc-900 dark:text-white flex items-center gap-1 text-[11px] font-mono font-medium">
 <CheckCircle2 className="w-3 h-3" />
 {isRu ? 'Разрешены' : 'Enabled'}
 </span>
 ) : (
 <span className="text-zinc-400 dark:text-white/40 flex items-center gap-1 text-[11px] font-mono">
 <XCircle className="w-3 h-3" />
 {isRu ? 'Заблокированы' : 'Blocked'}
 </span>
 )}
 </div>
 <p className="text-[11px] text-zinc-500 dark:text-white/50 leading-relaxed">
 {isRu
 ? 'B2B-атрибуция и межсайтовая аналитика кампаний.'
 : 'B2B campaign attribution and cross-site conversion tracking.'}
 </p>
 </div>
 </div>
 </div>
 </div>
 );
}
