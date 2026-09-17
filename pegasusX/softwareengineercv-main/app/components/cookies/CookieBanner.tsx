'use client';

import React from 'react';
import Link from 'next/link';
import { Shield, Sliders, Check, X, ShieldAlert } from 'lucide-react';
import { useCookieConsent } from '@/app/context/CookieConsentContext';
import { useLanguage } from '@/app/context/LanguageContext';
import { useTheme } from '@/app/context/ThemeContext';

export default function CookieBanner() {
  const { isBannerOpen, acceptAll, rejectNonEssential, openModal, isGPC } = useCookieConsent();
  const { language } = useLanguage();
  const { resolvedTheme } = useTheme();
  const isLight = resolvedTheme === 'light';
  const isRu = language === 'ru';

  if (!isBannerOpen) return null;

  return (
    <aside
      aria-label={isRu ? 'Согласие на использование файлов cookie' : 'Cookie Consent Notice'}
      role="region"
      className="fixed bottom-0 left-0 right-0 z-[10005] p-3 sm:p-5 pointer-events-none transition-all duration-300 animate-in fade-in slide-in-from-bottom-5"
    >
      <div className="max-w-5xl mx-auto pointer-events-auto">
        <div
          className={`border p-4 sm:p-6 shadow-2xl rounded-none transition-colors duration-200 ${
            isLight
              ? 'bg-white/98 border-black/10 text-zinc-900 shadow-[0_12px_40px_rgba(0,0,0,0.12)] backdrop-blur-xl'
              : 'bg-[#0B0B10]/98 border-white/15 text-white shadow-[0_16px_48px_rgba(0,0,0,0.8)] backdrop-blur-xl'
          }`}
        >
          <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-5">
            {/* Left Content Area */}
            <div className="flex items-start gap-3.5 flex-1 min-w-0">
              <div
                className={`w-9 h-9 rounded-none flex items-center justify-center shrink-0 mt-0.5 ${
                  isLight ? 'bg-black/5 text-zinc-900 border border-black/10' : 'bg-white/5 text-white border border-white/15'
                }`}
              >
                <Shield className="w-4 h-4 text-emerald-600 dark:text-[#8DDC96]" />
              </div>

              <div className="space-y-1.5 min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="font-mono text-[10px] uppercase tracking-wider font-semibold text-emerald-600 dark:text-[#8DDC96]">
                    {isRu ? 'КОНФИДЕНЦИАЛЬНОСТЬ & COOKIES' : 'PRIVACY & COOKIE CONSENT'}
                  </span>
                  <span className="font-mono text-[10px] text-zinc-400 dark:text-white/40 uppercase">
                    [GDPR / ePrivacy / CCPA]
                  </span>
                  {isGPC && (
                    <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-none bg-amber-500/10 border border-amber-500/30 text-amber-600 dark:text-amber-400 font-mono text-[9px] uppercase tracking-wider">
                      <ShieldAlert className="w-2.5 h-2.5" />
                      GPC ACTIVE
                    </span>
                  )}
                </div>

                <p className="text-xs sm:text-[13px] leading-relaxed text-zinc-600 dark:text-white/70">
                  {isRu ? (
                    <>
                      Мы используем файлы cookie для аутентификации сессий, обеспечения безопасности и измерения производительности согласно директиве ePrivacy и ст. 6 GDPR. Необязательные файлы cookie загружаются только с вашего предварительного согласия.{' '}
                      <Link
                        href="/cookie-policy"
                        className="underline hover:text-black dark:hover:text-white font-medium transition-colors"
                      >
                        Подробнее в Политике cookie
                      </Link>
                      .
                    </>
                  ) : (
                    <>
                      We deploy cookies for authenticated sessions, security, and telemetry under EU ePrivacy Directive & GDPR Art. 6. Non-essential cookies run only with prior affirmative consent.{' '}
                      <Link
                        href="/cookie-policy"
                        className="underline hover:text-black dark:hover:text-white font-medium transition-colors"
                      >
                        Read our Cookie Policy
                      </Link>
                      .
                    </>
                  )}
                </p>
              </div>
            </div>

            {/* Right Action Buttons */}
            <div className="flex flex-wrap items-center gap-2 sm:gap-2.5 shrink-0 pt-2 lg:pt-0">
              <button
                type="button"
                onClick={openModal}
                className={`inline-flex items-center gap-1.5 px-3.5 py-2 rounded-none text-xs font-mono uppercase tracking-wider border transition-all ${
                  isLight
                    ? 'border-black/15 bg-black/5 hover:bg-black/10 text-zinc-800'
                    : 'border-white/15 bg-white/5 hover:bg-white/10 text-white/80 hover:text-white'
                }`}
              >
                <Sliders className="w-3 h-3" />
                <span>{isRu ? 'Настроить' : 'Preferences'}</span>
              </button>

              <button
                type="button"
                onClick={rejectNonEssential}
                className={`inline-flex items-center gap-1.5 px-3.5 py-2 rounded-none text-xs font-mono uppercase tracking-wider border transition-all ${
                  isLight
                    ? 'border-black/20 bg-white hover:bg-zinc-100 text-zinc-800'
                    : 'border-white/20 bg-transparent hover:bg-white/10 text-white/90 hover:text-white'
                }`}
              >
                <X className="w-3 h-3 text-rose-500" />
                <span>{isRu ? 'Только обязательные' : 'Reject Optional'}</span>
              </button>

              <button
                type="button"
                onClick={acceptAll}
                className={`inline-flex items-center gap-1.5 px-4 py-2 rounded-none text-xs font-mono uppercase tracking-wider font-semibold transition-all shadow-md ${
                  isLight
                    ? 'bg-black text-white hover:bg-zinc-800 border border-black'
                    : 'bg-white text-black hover:bg-zinc-200 border border-white'
                }`}
              >
                <Check className="w-3.5 h-3.5" />
                <span>{isRu ? 'Принять все' : 'Accept All'}</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </aside>
  );
}
