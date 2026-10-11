'use client';

import { useState, FormEvent } from 'react';
import Link from 'next/link';
import Image from 'next/image';
import { CheckCircle2, AlertCircle, Loader2 } from 'lucide-react';
import { useLanguage } from '../context/LanguageContext';
import CookieSettingsTrigger from './cookies/CookieSettingsTrigger';
import ParticleText from './ParticleText';
import ParticleWorldMap from './ParticleWorldMap';

const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export default function Footer() {
  const { t } = useLanguage();
  const [email, setEmail] = useState('');
  const [status, setStatus] = useState<'idle' | 'loading' | 'success' | 'error'>('idle');
  const [responseMsg, setResponseMsg] = useState('');

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setResponseMsg('');

    if (!email || !EMAIL_REGEX.test(email.trim())) {
      setStatus('error');
      setResponseMsg(t('subscribe_invalid', 'Please enter a valid email address'));
      return;
    }

    setStatus('loading');

    try {
      const res = await fetch('/api/subscribe', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: email.trim() }),
      });

      const data = await res.json();

      if (res.ok && data.success) {
        setStatus('success');
        setResponseMsg(t('subscribe_success', 'Subscribed successfully'));
        setEmail('');

        try {
          const stored = localStorage.getItem('pegasus_subscriptions');
          const list = stored ? JSON.parse(stored) : [];
          list.unshift({ email: email.trim(), date: new Date().toISOString() });
          localStorage.setItem('pegasus_subscriptions', JSON.stringify(list));
        } catch {}

        setTimeout(() => {
          setStatus('idle');
          setResponseMsg('');
        }, 5000);
      } else {
        setStatus('error');
        setResponseMsg(data.error || t('subscribe_error', 'Subscription failed'));
      }
    } catch (err) {
      console.error('[Footer] Subscription request failed:', err);
      setStatus('error');
      setResponseMsg(t('subscribe_error', 'Subscription failed'));
    }
  };

  const platformLinks = [
    { name: 'Documentation', href: '/docs' },
    { name: t('nav_platform', 'Платформа'), href: '/platform' },
    { name: 'Global Logistics', href: '/global-logistics' },
    { name: 'Supply Chain Software', href: '/supply-chain-software' },
    { name: 'Logistics Automation', href: '/logistics-automation' },
    { name: t('footer_order_lifecycle', 'Жизненный цикл заказа'), href: '/platform/order-lifecycle' },
    { name: t('footer_how_it_works', 'Как работает Pegasus'), href: '/platform/how-pegasus-works' },
    { name: t('footer_trust', 'Доверие и надежность'), href: '/platform/trust-reliability' },
  ];

  const companyLinks = [
    { name: t('footer_about', 'О нас'), href: '/#about' },
    { name: t('nav_demo', 'Запросить демо'), href: '/join' },
    { name: t('nav_contact', 'Связаться с нами'), href: '/contact' },
    { name: t('nav_roles', 'Роли'), href: '/roles' },
    { name: t('nav_modules', 'Модули'), href: '/projects' },
    { name: t('nav_solutions', 'Решения'), href: '/solutions' },
    { name: t('cloud_eco_nav', 'Облачная экосистема'), href: '/cloud-ecosystem' },
    { name: t('footer_trust', 'Доверие и надежность'), href: '/platform/trust-reliability' },
  ];

  const resourceLinks = [
    { name: t('footer_platform', 'Обзор платформы'), href: '/platform' },
    { name: t('cloud_eco_nav', 'Облачная экосистема'), href: '/cloud-ecosystem' },
    { name: t('footer_apps_deploy', 'Приложения и развертывание'), href: '/apps-deploy' },
    { name: 'TMS Alternatives', href: '/alternatives' },
    { name: 'TMS Comparisons', href: '/compare' },
    { name: 'Global Markets', href: '/markets' },
    { name: t('nav_cookie_policy', 'Политика файлов cookie'), href: '/cookie-policy' },
  ];

  return (
    <footer
      className="bg-[#000000] text-white border-t border-white/10 overflow-hidden font-sans relative keep-dark"
      data-keep-dark
      data-keep-white
    >
      {/* Background grain / grid effect */}
      <div className="absolute inset-0 pointer-events-none opacity-20 bg-[linear-gradient(to_right,#80808012_1px,transparent_1px),linear-gradient(to_bottom,#80808012_1px,transparent_1px)] bg-[size:24px_24px]" />
      <div className="absolute inset-0 pointer-events-none opacity-15 bg-[radial-gradient(ellipse_at_center,rgba(16,185,129,0.12)_0%,transparent_70%)]" />

      {/* 1. TOP ROW: Centered Email Subscription */}
      <div className="py-8 px-6 sm:px-10 lg:px-16 border-b border-white/10 relative z-10 flex justify-center">
        <div className="w-full max-w-md">
          <form onSubmit={handleSubmit} className="flex flex-col items-center gap-2 w-full">
            <div className="flex w-full bg-[#050505] border border-white/20 focus-within:border-white/50 rounded-none overflow-hidden transition-colors">
              <input
                type="email"
                value={email}
                onChange={(e) => {
                  setEmail(e.target.value);
                  if (status === 'error') {
                    setStatus('idle');
                    setResponseMsg('');
                  }
                }}
                placeholder={t('footer_email_placeholder', 'your.name@company.com')}
                disabled={status === 'loading'}
                required
                aria-label={t('footer_subscribe', 'Email Address')}
                className="bg-transparent text-white placeholder:text-zinc-500 px-4 py-2.5 outline-none flex-1 text-xs sm:text-sm font-mono disabled:opacity-50"
              />
              <button
                type="submit"
                disabled={status === 'loading'}
                className="bg-white hover:bg-zinc-200 text-black font-semibold px-5 py-2.5 transition-colors flex items-center justify-center gap-2 text-xs sm:text-sm shrink-0 border-l border-white/10 disabled:opacity-50 cursor-pointer disabled:cursor-not-allowed min-w-[130px]"
              >
                {status === 'loading' ? (
                  <>
                    <Loader2 className="w-3.5 h-3.5 animate-spin text-black" />
                    <span>{t('subscribe_submitting', 'Submitting')}</span>
                  </>
                ) : (
                  <span>→ {t('footer_subscribe_btn', 'Подписаться')}</span>
                )}
              </button>
            </div>

            {/* Status Badges */}
            {status === 'success' && (
              <div className="flex items-center gap-2 text-xs font-mono text-emerald-300 bg-emerald-950/40 border border-emerald-500/30 px-3 py-1.5 rounded-none animate-in fade-in w-full justify-center">
                <CheckCircle2 className="w-3.5 h-3.5 shrink-0 text-emerald-400" />
                <span>{responseMsg || t('subscribe_success', 'Subscribed successfully')}</span>
              </div>
            )}

            {status === 'error' && (
              <div className="flex items-center gap-2 text-xs font-mono text-rose-300 bg-rose-950/40 border border-rose-500/30 px-3 py-1.5 rounded-none animate-in fade-in w-full justify-center">
                <AlertCircle className="w-3.5 h-3.5 shrink-0 text-rose-400" />
                <span>{responseMsg || t('subscribe_invalid', 'Invalid email')}</span>
              </div>
            )}
          </form>
        </div>
      </div>

      {/* 2. SECOND ROW: Logo + 3 Navigation Columns + World Map with Vertical Dividers */}
      <div className="flex flex-col md:grid md:grid-cols-2 lg:flex lg:flex-row border-b border-white/10 relative z-10 items-stretch">
        {/* Column 1: Pegasus Logo */}
        <div className="lg:w-[190px] xl:w-[220px] shrink-0 flex flex-col items-center justify-center p-8 border-b md:border-r lg:border-b-0 lg:border-r border-white/10 text-center">
          <div className="flex flex-col items-center justify-center">
            <Image
              src="/pegasus-nav.png"
              width={54}
              height={45}
              alt="Pegasus Logo"
              className="object-contain"
              priority
            />
            <span className="text-white font-bold tracking-[0.25em] text-sm uppercase mt-4 block">
              PEGASUS
            </span>
          </div>
        </div>

        {/* Column 2: ПЛАТФОРМА */}
        <div className="flex-1 min-w-0 p-8 border-b lg:border-b-0 lg:border-r border-white/10">
          <span className="text-[11px] font-mono tracking-wider text-white/50 uppercase block mb-5">
            {t('footer_platform_title', 'ПЛАТФОРМА')}
          </span>
          <div className="flex flex-col space-y-3.5 text-xs">
            {platformLinks.map((link) => (
              <Link
                key={link.name}
                prefetch={true}
                href={link.href}
                className="text-white/70 hover:text-white transition-colors block whitespace-nowrap"
              >
                {link.name}
              </Link>
            ))}
          </div>
        </div>

        {/* Column 3: О КОМПАНИИ */}
        <div className="flex-1 min-w-0 p-8 border-b md:border-r lg:border-b-0 lg:border-r border-white/10">
          <span className="text-[11px] font-mono tracking-wider text-white/50 uppercase block mb-5">
            {t('footer_company', 'О КОМПАНИИ')}
          </span>
          <div className="flex flex-col space-y-3.5 text-xs">
            {companyLinks.map((link) => (
              <Link
                key={link.name}
                prefetch={true}
                href={link.href}
                className="text-white/70 hover:text-white transition-colors block whitespace-nowrap"
              >
                {link.name}
              </Link>
            ))}
          </div>
        </div>

        {/* Column 4: РЕСУРСЫ */}
        <div className="flex-1 min-w-0 p-8 border-b md:border-b lg:border-b-0 lg:border-r border-white/10">
          <span className="text-[11px] font-mono tracking-wider text-white/50 uppercase block mb-5">
            {t('footer_policies', 'РЕСУРСЫ')}
          </span>
          <div className="flex flex-col space-y-3.5 text-xs">
            {resourceLinks.map((link) => (
              <Link
                key={link.name}
                prefetch={true}
                href={link.href}
                className="text-white/70 hover:text-white transition-colors block whitespace-nowrap"
              >
                {link.name}
              </Link>
            ))}
            <CookieSettingsTrigger className="text-white/70 hover:text-white transition-colors cursor-pointer text-xs text-left whitespace-nowrap block" />
          </div>
        </div>

        {/* Column 5: Interactive World Map on the Right Side */}
        <div className="md:col-span-2 lg:w-[320px] xl:w-[360px] shrink-0 p-6 lg:p-8 flex flex-col items-center justify-center">
          <div className="w-full flex items-center justify-center max-w-[340px]">
            <ParticleWorldMap variant="dark" showHubs={false} className="w-full" />
          </div>
        </div>
      </div>

      {/* 3. THIRD ROW: Full-Width Interactive Pegasus Particle Text */}
      <div className="py-4 sm:py-6 md:py-8 px-0 flex justify-center items-center overflow-hidden relative z-10 w-full border-b border-white/10">
        <div className="w-full h-[32vw] sm:h-[30vw] md:h-[28vw] min-h-[160px] max-h-[520px] relative flex items-center justify-center">
          <ParticleText
            text="pegasus"
            fontWeight={900}
            letterSpacing="-0.04em"
            fitWidth={true}
            color="#ffffff"
            highlightColor="#ffffff"
            particleSize={2.4}
            density={4}
            scatter={0}
            gatherDuration={0}
            stagger={0}
            pointerRepel={42}
            repelRadius={120}
            idleDrift={0.6}
            trigger="none"
            textAlign="center"
            glow={false}
          />
        </div>
      </div>

      {/* 4. FOURTH ROW: Legal & Policies Bar at the Bottom */}
      <div className="py-4 px-6 sm:px-10 lg:px-16 flex flex-wrap items-center justify-between gap-y-3 gap-x-6 text-[10px] font-mono text-white/50 uppercase relative z-10">
        <div>
          © 2026 PEGASUS LOGISTICS OS. ALL RIGHTS RESERVED.
        </div>

        <div className="flex flex-wrap items-center gap-x-4 gap-y-1.5">
          <Link prefetch={true} href="/docs" className="hover:text-white transition-colors">
            API DOCS
          </Link>
          <Link prefetch={true} href="/platform/trust-reliability" className="hover:text-white transition-colors">
            SECURITY & TRUST
          </Link>
          <Link prefetch={true} href="/cookie-policy" className="hover:text-white transition-colors">
            PRIVACY POLICY
          </Link>
          <Link prefetch={true} href="/cookie-policy" className="hover:text-white transition-colors">
            TERMS OF SERVICE
          </Link>
          <CookieSettingsTrigger className="hover:text-white cursor-pointer uppercase text-[10px] font-mono" />
        </div>
      </div>
    </footer>
  );
}
