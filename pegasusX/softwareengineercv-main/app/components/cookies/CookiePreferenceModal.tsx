'use client';

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import {
  X,
  Shield,
  Check,
  Lock,
  ChevronDown,
  ChevronUp,
  ShieldAlert,
  Database,
  Sliders,
  Layers,
} from 'lucide-react';
import { useCookieConsent } from '@/app/context/CookieConsentContext';
import { useLanguage } from '@/app/context/LanguageContext';
import { useTheme } from '@/app/context/ThemeContext';
import { ConsentCategories, CookieCategory } from '@/app/lib/cookies/cookieTypes';
import { COOKIE_INVENTORY } from '@/app/lib/cookies/cookieRegistry';

export default function CookiePreferenceModal() {
  const {
    isModalOpen,
    closeModal,
    consent,
    savePreferences,
    acceptAll,
    rejectNonEssential,
    isGPC,
  } = useCookieConsent();
  const { language } = useLanguage();
  const { resolvedTheme } = useTheme();
  const isLight = resolvedTheme === 'light';
  const isRu = language === 'ru';

  const [categories, setCategories] = useState<ConsentCategories>({
    necessary: true,
    functional: false,
    analytics: false,
    marketing: false,
  });

  const [expandedCategory, setExpandedCategory] = useState<CookieCategory | null>(null);

  // Sync state when modal opens
  useEffect(() => {
    if (isModalOpen) {
      if (consent) {
        setCategories(consent.categories);
      } else {
        setCategories({
          necessary: true,
          functional: false,
          analytics: false,
          marketing: false,
        });
      }
    }
  }, [isModalOpen, consent]);

  if (!isModalOpen) return null;

  const handleToggle = (key: keyof ConsentCategories) => {
    if (key === 'necessary') return; // Cannot toggle strictly necessary
    if (key === 'marketing' && isGPC) return; // Locked by GPC

    setCategories((prev) => ({
      ...prev,
      [key]: !prev[key],
    }));
  };

  const handleSave = () => {
    savePreferences(categories);
  };

  const categoryCards: {
    key: CookieCategory;
    titleEn: string;
    titleRu: string;
    descEn: string;
    descRu: string;
    legalBasisEn: string;
    legalBasisRu: string;
    isMandatory?: boolean;
    icon: React.ComponentType<{ className?: string }>;
  }[] = [
    {
      key: 'necessary',
      titleEn: 'Strictly Necessary',
      titleRu: 'Строго обязательные',
      descEn:
        'Essential for core platform operations, cryptographic authentication, CSRF mutation guards, session maintenance, and recording compliance consent audit records.',
      descRu:
        'Необходимы для работы платформы, криптографической аутентификации, защиты от CSRF, сессий пользователей и фиксации аудита согласий.',
      legalBasisEn: 'GDPR Art. 6(1)(f) Legitimate Interest / ePrivacy Directive Art. 5(3)',
      legalBasisRu: 'Ст. 6(1)(f) GDPR (Законный интерес) / Ст. 5(3) Директивы ePrivacy',
      isMandatory: true,
      icon: Lock,
    },
    {
      key: 'functional',
      titleEn: 'Functional & Preferences',
      titleRu: 'Функциональные и настройки',
      descEn:
        'Remembers role navigation layout, density state, regional freight corridor selection, and tactical map viewport preferences.',
      descRu:
        'Сохраняют конфигурацию навигационных панелей, плотность интерфейса, фильтры региональных коридоров и координаты карты.',
      legalBasisEn: 'GDPR Art. 6(1)(a) Consent',
      legalBasisRu: 'Ст. 6(1)(a) GDPR (Согласие)',
      icon: Sliders,
    },
    {
      key: 'analytics',
      titleEn: 'Analytics & Latency Telemetry',
      titleRu: 'Аналитика и телеметрия производительности',
      descEn:
        'Self-hosted pseudonymized telemetry measuring API throughput, platform roundtrip latency, and diagnostic crash traces without user profiling.',
      descRu:
        'Обезличенная телеметрия для оценки пропускной способности API, задержек системы и диагностики сбоев без профилирования пользователей.',
      legalBasisEn: 'GDPR Art. 6(1)(a) Consent',
      legalBasisRu: 'Ст. 6(1)(a) GDPR (Согласие)',
      icon: Database,
    },
    {
      key: 'marketing',
      titleEn: 'Marketing & Partner Attribution',
      titleRu: 'Маркетинг и партнерская атрибуция',
      descEn:
        'Measures conversion attribution for B2B wholesale freight logistics inquiries and carrier network recruitment.',
      descRu:
        'Оценивает конверсии корпоративных B2B-запросов и привлечения перевозчиков в сеть Pegasus.',
      legalBasisEn: 'GDPR Art. 6(1)(a) Consent / CCPA Opt-Out',
      legalBasisRu: 'Ст. 6(1)(a) GDPR (Согласие) / CCPA Отказ',
      icon: Layers,
    },
  ];

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="cookie-modal-title"
      className="fixed inset-0 z-[10006] flex items-center justify-center p-3 sm:p-6 overflow-y-auto bg-black/80 backdrop-blur-md animate-in fade-in duration-200"
    >
      <div
        className={`w-full max-w-3xl my-auto rounded-none border shadow-2xl overflow-hidden flex flex-col max-h-[90vh] transition-colors duration-200 ${
          isLight
            ? 'bg-white border-black/10 text-zinc-900 shadow-[0_24px_64px_rgba(0,0,0,0.18)]'
            : 'bg-black border-white/15 text-white shadow-[0_24px_64px_rgba(0,0,0,0.9)]'
        }`}
      >
        {/* Modal Header */}
        <div
          className={`px-6 py-5 border-b flex items-center justify-between shrink-0 ${
            isLight ? 'border-black/10 bg-zinc-50' : 'border-white/10 bg-zinc-950'
          }`}
        >
          <div className="flex items-center gap-3">
            <div
              className={`w-9 h-9 rounded-none flex items-center justify-center border ${
                isLight ? 'bg-white border-black/10 text-zinc-800' : 'bg-white/5 border-white/15 text-white'
              }`}
            >
              <Shield className="w-4 h-4 text-zinc-900 dark:text-white" />
            </div>
            <div>
              <h2 id="cookie-modal-title" className="text-base sm:text-lg font-semibold tracking-tight">
                {isRu ? 'Центр управления файлами cookie' : 'Cookie & Telemetry Preference Center'}
              </h2>
              <div className="flex items-center gap-2 mt-0.5">
                <span className="font-mono text-[10px] uppercase text-zinc-500 dark:text-white/50">
                  PEGASUS COMPLIANCE ENGINE
                </span>
                <span className="font-mono text-[10px] text-zinc-600 dark:text-white/70">
                  [v2026.1]
                </span>
              </div>
            </div>
          </div>

          <button
            type="button"
            onClick={closeModal}
            aria-label={isRu ? 'Закрыть' : 'Close'}
            className={`p-2 rounded-none border transition-colors ${
              isLight
                ? 'border-black/10 hover:bg-black/5 text-zinc-700'
                : 'border-white/10 hover:bg-white/10 text-white/70 hover:text-white'
            }`}
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Modal Body: Scrollable */}
        <div className="p-6 overflow-y-auto space-y-4 flex-1">
          {/* GPC Alert if active */}
          {isGPC && (
            <div className="p-3.5 rounded-none bg-zinc-900/50 border border-white/20 flex items-start gap-3">
              <ShieldAlert className="w-4 h-4 text-zinc-900 dark:text-white shrink-0 mt-0.5" />
              <div className="text-xs leading-relaxed text-zinc-800 dark:text-zinc-200">
                <strong>{isRu ? 'Обнаружен сигнал GPC' : 'Global Privacy Control (GPC) Active'}:</strong>{' '}
                {isRu
                  ? 'Ваш браузер передаёт сигнал отказа от продажи и передачи данных. Маркетинговые cookie принудительно отключены согласно закону CCPA/CPRA.'
                  : 'Your browser transmits a Global Privacy Control signal. Marketing and targeting cookies have been permanently disabled to honor your opt-out rights under CCPA/CPRA.'}
              </div>
            </div>
          )}

          {/* Legal overview text */}
          <p className="text-xs sm:text-[13px] leading-relaxed text-zinc-600 dark:text-white/70">
            {isRu
              ? 'Выберите категории файлов cookie, которые вы разрешаете использовать платформе Pegasus. Обязательные cookie необходимы для функционирования системы и не могут быть отключены.'
              : 'Configure which cookie categories you permit Pegasus to process. Strictly Necessary cookies remain active to preserve cryptographic security, session authentication, and transactional ledger routing.'}
          </p>

          {/* Category Cards */}
          <div className="space-y-3 pt-2">
            {categoryCards.map((card) => {
              const Icon = card.icon;
              const isChecked = categories[card.key];
              const isExpanded = expandedCategory === card.key;
              const matchingCookies = COOKIE_INVENTORY.filter((c) => c.category === card.key);

              return (
                <div
                  key={card.key}
                  className={`border rounded-none p-4 transition-all duration-200 ${
                    isLight
                      ? 'border-black/10 bg-zinc-50/70 hover:border-black/20'
                      : 'border-white/10 bg-zinc-950 hover:border-white/20'
                  }`}
                >
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex items-start gap-3 flex-1 min-w-0">
                      <div
                        className={`w-7 h-7 rounded-none flex items-center justify-center shrink-0 mt-0.5 ${
                          isLight ? 'bg-black/5 text-zinc-700' : 'bg-white/5 text-white/80'
                        }`}
                      >
                        <Icon className="w-3.5 h-3.5" />
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="text-sm font-semibold tracking-tight">
                            {isRu ? card.titleRu : card.titleEn}
                          </span>
                          {card.isMandatory && (
                            <span className="px-2 py-0.5 rounded-none text-[9px] font-mono uppercase tracking-wider font-bold bg-white/10 border border-white/20 text-zinc-800 dark:text-white">
                              {isRu ? 'ОБЯЗАТЕЛЬНО' : 'ALWAYS ACTIVE'}
                            </span>
                          )}
                          {card.key === 'marketing' && isGPC && (
                            <span className="px-2 py-0.5 rounded-none text-[9px] font-mono uppercase tracking-wider font-bold bg-white/10 border border-white/20 text-zinc-800 dark:text-white">
                              LOCKED BY GPC
                            </span>
                          )}
                        </div>
                        <p className="text-xs leading-relaxed text-zinc-600 dark:text-white/60 mt-1">
                          {isRu ? card.descRu : card.descEn}
                        </p>
                        <span className="inline-block text-[10px] font-mono text-zinc-400 dark:text-white/40 mt-1.5">
                          {isRu ? card.legalBasisRu : card.legalBasisEn}
                        </span>
                      </div>
                    </div>

                    {/* Switch Toggle */}
                    <div className="shrink-0 flex items-center pt-1">
                      {card.isMandatory ? (
                        <div className="w-11 h-6 rounded-none bg-black/20 dark:bg-white/20 border border-black/30 dark:border-white/30 flex items-center justify-end px-1 cursor-not-allowed">
                          <div className="w-4 h-4 rounded-none bg-black dark:bg-white" />
                        </div>
                      ) : (
                        <button
                          type="button"
                          role="switch"
                          aria-checked={isChecked}
                          disabled={card.key === 'marketing' && isGPC}
                          onClick={() => handleToggle(card.key)}
                          className={`w-11 h-6 rounded-none border transition-colors duration-200 p-0.5 flex items-center outline-none focus-visible:ring-2 focus-visible:ring-offset-2 cursor-pointer ${
                            isChecked
                              ? 'bg-black dark:bg-white border-black dark:border-white justify-end hover:bg-zinc-800 dark:hover:bg-zinc-200'
                              : isLight
                              ? 'bg-zinc-200 border-zinc-300 justify-start hover:border-black'
                              : 'bg-zinc-800 border-zinc-700 justify-start hover:border-white'
                          }`}
                        >
                          <div className={`w-4 h-4 rounded-none shadow-sm ${isChecked ? 'bg-white dark:bg-black' : 'bg-zinc-500 dark:bg-zinc-400'}`} />
                        </button>
                      )}
                    </div>
                  </div>

                  {/* Cookie Details Dropdown Trigger */}
                  <div className="mt-3 pt-2.5 border-t border-black/5 dark:border-white/5 flex items-center justify-between">
                    <button
                      type="button"
                      onClick={() => setExpandedCategory(isExpanded ? null : card.key)}
                      className="inline-flex items-center gap-1.5 text-[11px] font-mono text-zinc-500 hover:text-black dark:text-white/50 dark:hover:text-white transition-colors"
                    >
                      <span>
                        {isRu
                          ? `${matchingCookies.length} файлов cookie в этой категории`
                          : `${matchingCookies.length} cookies declared`}
                      </span>
                      {isExpanded ? <ChevronUp className="w-3 h-3" /> : <ChevronDown className="w-3 h-3" />}
                    </button>
                  </div>

                  {/* Expanded Cookies Inventory Table */}
                  {isExpanded && (
                    <div className="mt-3 pt-3 border-t border-black/5 dark:border-white/5 space-y-2">
                      {matchingCookies.map((c) => (
                        <div
                          key={c.name}
                          className={`p-2.5 rounded-none text-xs font-mono border ${
                            isLight
                              ? 'bg-white border-black/5 text-zinc-800'
                              : 'bg-black/40 border-white/5 text-white/80'
                          }`}
                        >
                          <div className="flex items-center justify-between font-bold text-zinc-900 dark:text-white mb-1">
                            <span>{c.name}</span>
                            <span className="text-[10px] text-zinc-400 dark:text-white/40 font-normal">
                              {c.party} · {isRu ? c.expiryRu : c.expiryEn}
                            </span>
                          </div>
                          <p className="text-[11px] font-sans text-zinc-600 dark:text-white/60 leading-normal">
                            {isRu ? c.purposeRu : c.purposeEn}
                          </p>
                          <div className="mt-1 text-[10px] text-zinc-400 dark:text-white/40">
                            Host: <span className="underline">{c.domain}</span> ({c.provider})
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>

        {/* Modal Footer: Action Buttons */}
        <div
          className={`p-5 sm:p-6 border-t flex flex-col sm:flex-row items-center justify-between gap-3 shrink-0 ${
            isLight ? 'border-black/10 bg-zinc-50' : 'border-white/10 bg-zinc-950'
          }`}
        >
          <div className="text-[11px] text-zinc-500 dark:text-white/50">
            <Link
              href="/cookie-policy"
              onClick={closeModal}
              className="underline hover:text-black dark:hover:text-white transition-colors"
            >
              {isRu ? 'Полный текст Политики cookie' : 'View Full Cookie Policy'}
            </Link>
          </div>

          <div className="flex flex-wrap items-center gap-2.5 w-full sm:w-auto justify-end">
            <button
              type="button"
              onClick={rejectNonEssential}
              className={`px-4 py-2 rounded-none text-xs font-mono uppercase tracking-wider border transition-all ${
                isLight
                  ? 'border-black/20 bg-white hover:bg-zinc-100 text-zinc-800'
                  : 'border-white/20 bg-transparent hover:bg-white/10 text-white/80 hover:text-white'
              }`}
            >
              {isRu ? 'Отклонить необязательные' : 'Reject Non-Essential'}
            </button>

            <button
              type="button"
              onClick={handleSave}
              className={`px-4 py-2 rounded-none text-xs font-mono uppercase tracking-wider font-semibold border transition-all ${
                isLight
                  ? 'bg-zinc-900 text-white hover:bg-black border-zinc-900'
                  : 'bg-white/15 hover:bg-white/25 text-white border-white/20'
              }`}
            >
              {isRu ? 'Сохранить выбор' : 'Save Preferences'}
            </button>

            <button
              type="button"
              onClick={acceptAll}
              className={`px-4 py-2 rounded-none text-xs font-mono uppercase tracking-wider font-semibold transition-all shadow-md ${
                isLight
                  ? 'bg-black text-white hover:bg-zinc-800 border border-black'
                  : 'bg-white text-black hover:bg-zinc-200 border border-white'
              }`}
            >
              <Check className="w-3.5 h-3.5 inline mr-1" />
              {isRu ? 'Принять все' : 'Accept All'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
