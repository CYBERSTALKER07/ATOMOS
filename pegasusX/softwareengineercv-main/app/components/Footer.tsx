'use client';

import { useState, FormEvent } from 'react';
import Link from 'next/link';
import { Linkedin, Youtube, Instagram, CheckCircle2, AlertCircle, Loader2 } from 'lucide-react';
import { ArrowRight } from '@/components/icons';
import { useLanguage } from '../context/LanguageContext';
import CookieSettingsTrigger from './cookies/CookieSettingsTrigger';
import ParticleText from './ParticleText';
import TextType from './TextType';

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
      setResponseMsg(t('subscribe_invalid'));
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
        setResponseMsg(t('subscribe_success'));
        setEmail('');

        // Store subscription record in local storage history
        try {
          const stored = localStorage.getItem('pegasus_subscriptions');
          const list = stored ? JSON.parse(stored) : [];
          list.unshift({ email: email.trim(), date: new Date().toISOString() });
          localStorage.setItem('pegasus_subscriptions', JSON.stringify(list));
        } catch (e) {
          console.error('Failed to store subscription locally', e);
        }

        setTimeout(() => {
          setStatus('idle');
          setResponseMsg('');
        }, 5000);
      } else {
        setStatus('error');
        setResponseMsg(data.error || t('subscribe_error'));
      }
    } catch (err) {
      console.error('[Footer] Subscription request failed:', err);
      setStatus('error');
      setResponseMsg(t('subscribe_error'));
    }
  };

  const platformLinks = [
    { name: t('nav_docs', 'Documentation'), href: '/docs' },
    { name: t('nav_platform'), href: '/platform' },
    { name: t('nav_global_logistics', 'Global Logistics'), href: '/global-logistics' },
    { name: t('nav_supply_chain_software', 'Supply Chain Software'), href: '/supply-chain-software' },
    { name: t('nav_logistics_automation', 'Logistics Automation'), href: '/logistics-automation' },
    { name: t('footer_order_lifecycle'), href: '/platform/order-lifecycle' },
    { name: t('footer_how_it_works'), href: '/platform/how-pegasus-works' },
    { name: t('footer_trust'), href: '/platform/trust-reliability' },
  ];

  const companyLinks = [
    { name: t('nav_demo'), href: '/join' },
    { name: t('nav_contact'), href: '/contact' },
    { name: t('nav_roles'), href: '/roles' },
    { name: t('nav_modules'), href: '/projects' },
  ];

  const policiesLinks = [
    { name: t('nav_tour'), href: '/platform' },
    { name: t('cloud_eco_nav', 'Cloud ecosystem'), href: '/cloud-ecosystem' },
    { name: t('footer_apps_deploy'), href: '/apps-deploy' },
    { name: t('nav_alternatives', 'TMS Alternatives'), href: '/alternatives' },
    { name: t('nav_compare', 'TMS Comparisons'), href: '/compare' },
    { name: t('nav_markets', 'Global Markets'), href: '/markets' },
    { name: t('nav_cookie_policy', 'Cookie Policy'), href: '/cookie-policy' },
  ];

  const typedPhrases = [
    t('hero_subtitle', 'Global Enterprise Logistics Architecture'),
    t('hero_phrases_hyperlocal', 'Autonomous Logistics Cloud'),
    t('hero_phrases_b2b', 'Spanner Distributed Core'),
    t('hero_phrases_execution', 'Real-time Telemetry & Dispatch'),
  ];

  return (
    <footer className="bg-[#000000] text-white border-t border-white/5 overflow-hidden font-sans relative">

      {/* Background grain / grid effect */}
      <div className="absolute inset-0 pointer-events-none opacity-20 bg-[linear-gradient(to_right,#80808012_1px,transparent_1px),linear-gradient(to_bottom,#80808012_1px,transparent_1px)] bg-[size:24px_24px]" />
      <div className="absolute inset-0 pointer-events-none opacity-20 bg-[radial-gradient(ellipse_at_center,rgba(255,255,255,0.1)_0%,transparent_70%)]" />

      {/* Top section with input */}
      <div className="border-b border-white/5 flex flex-col items-center justify-center py-16 px-4 relative z-10">
        <form onSubmit={handleSubmit} className="w-full max-w-[440px] flex flex-col gap-3">
          <div className="flex bg-black border border-white/15 focus-within:border-white/40 overflow-hidden w-full transition-colors">
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
              placeholder={t('footer_email_placeholder')}
              disabled={status === 'loading'}
              required
              aria-label={t('footer_subscribe')}
              className="bg-transparent text-white placeholder:text-white/40 px-4 py-3 outline-none flex-1 text-xs sm:text-sm font-mono disabled:opacity-50"
            />
            <button
              type="submit"
              disabled={status === 'loading'}
              className="bg-[#E2FD52] hover:bg-[#CEFF00] text-black px-6 py-3 transition-colors flex items-center justify-center gap-2 text-xs font-bold uppercase tracking-widest border-l border-white/10 disabled:opacity-50 cursor-pointer disabled:cursor-not-allowed min-w-[130px]"
            >
              {status === 'loading' ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin text-black" />
                  <span>{t('subscribe_submitting')}</span>
                </>
              ) : (
                <>
                  <ArrowRight size={14} className="text-black" />
                  <span>{t('footer_subscribe_btn')}</span>
                </>
              )}
            </button>
          </div>

          {/* Feedback Badges */}
          {status === 'success' && (
            <div className="flex items-center gap-2 text-xs font-mono text-emerald-400 bg-emerald-950/50 border border-emerald-500/30 px-3.5 py-2.5 rounded-sm animate-in fade-in slide-in-from-top-1">
              <CheckCircle2 className="w-4 h-4 shrink-0 text-emerald-400" />
              <span>{responseMsg || t('subscribe_success')}</span>
            </div>
          )}

          {status === 'error' && (
            <div className="flex items-center gap-2 text-xs font-mono text-rose-400 bg-rose-950/50 border border-rose-500/30 px-3.5 py-2.5 rounded-sm animate-in fade-in slide-in-from-top-1">
              <AlertCircle className="w-4 h-4 shrink-0 text-rose-400" />
              <span>{responseMsg || t('subscribe_invalid')}</span>
            </div>
          )}
        </form>
      </div>

      {/* Main footer grid */}
      <div className="grid grid-cols-1 md:grid-cols-4 border-b border-white/5 relative z-10 max-w-[1600px] mx-auto">
        <div className="absolute inset-y-0 left-1/4 border-l border-white/5 hidden md:block" />
        <div className="absolute inset-y-0 left-2/4 border-l border-white/5 hidden md:block" />
        <div className="absolute inset-y-0 left-3/4 border-l border-white/5 hidden md:block" />

        {/* Logo col */}
        <div className="p-16 flex flex-col items-center justify-center max-md:border-b border-white/5">
          <img src="/pegasus.jpg" width={100} height={100} alt="Pegasus Logistics Platform Logo" loading="lazy" />
          <span className="mt-6 text-xl font-title font-light tracking-[0.2em] text-white uppercase">Pegasus</span>
        </div>

        {/* Platform Links col */}
        <div className="p-12 max-md:border-b border-white/5">
          <h4 className="text-[10px] sm:text-[11px] tracking-[0.2em] text-white/45 mb-8 font-mono uppercase">{t('footer_platform_title')}</h4>
          <ul className="space-y-4">
            {platformLinks.map(link => (
              <li key={link.name}>
                <Link href={link.href} className="text-white/60 hover:text-white text-sm font-light leading-relaxed transition-colors">
                  {link.name}
                </Link>
              </li>
            ))}
          </ul>
        </div>

        {/* Company col */}
        <div className="p-12 max-md:border-b border-white/5">
          <h4 className="text-[10px] sm:text-[11px] tracking-[0.2em] text-white/45 mb-8 font-mono uppercase">{t('footer_company')}</h4>
          <ul className="space-y-4">
            {companyLinks.map(link => (
              <li key={link.name}>
                <Link href={link.href} className="text-white/60 hover:text-white text-sm font-light leading-relaxed transition-colors">
                  {link.name}
                </Link>
              </li>
            ))}
          </ul>
        </div>

        {/* Resources col */}
        <div className="p-12">
          <h4 className="text-[10px] sm:text-[11px] tracking-[0.2em] text-white/45 mb-8 font-mono uppercase">{t('footer_policies')}</h4>
          <ul className="space-y-4 mb-10">
            {policiesLinks.map(link => (
              <li key={link.name}>
                <Link href={link.href} className="text-white/60 hover:text-white text-sm font-light leading-relaxed transition-colors">
                  {link.name}
                </Link>
              </li>
            ))}
            <li>
              <CookieSettingsTrigger className="text-white/60 hover:text-white text-sm font-light" />
            </li>
          </ul>
        </div>
      </div>

      {/* Animated text styled like hero section */}
      <div className="pt-16 pb-12 px-4 flex flex-col justify-center items-center overflow-hidden border-b border-white/5 relative z-10">
        <div className="w-full max-w-[1600px] h-44 sm:h-56 md:h-72 lg:h-80 xl:h-96 relative flex items-center justify-center">
          <ParticleText
            text="Pegasus"
            particleSize={2.4}
            density={4}
            color="#f8fafc"
            highlightColor="#10B981"
            scatter={160}
            gatherDuration={1500}
            stagger={350}
            pointerRepel={42}
            repelRadius={120}
            idleDrift={0.6}
            trigger="view"
            fontSize="clamp(3.5rem, 15vw, 12rem)"
            fontWeight={800}
            textAlign="center"
            glow={true}
          />
        </div>

        {/* Hero animated typewriter phrase */}
        <div className="mt-3 sm:mt-5 text-center">
          <div className="text-sm sm:text-base md:text-xl font-light tracking-tight text-white/80 min-h-[1.75em] flex items-center justify-center">
            <TextType
              text={typedPhrases}
              typingSpeed={70}
              pauseDuration={1800}
              deletingSpeed={45}
              showCursor={true}
              cursorCharacter="|"
              loop={true}
              startOnVisible={true}
              textColors={['#FFFFFF', '#10B981', '#C0C0C0']}
              className="font-light"
              cursorClassName="text-white font-light"
            />
          </div>
        </div>
      </div>

      {/* Copyright & Legal Row */}
      <div className="py-6 px-4 flex flex-wrap items-center justify-center gap-x-4 gap-y-2 text-center text-white/45 text-[10px] sm:text-[11px] font-mono tracking-wider relative z-10">
        <span>©2026 Pegasus. {t('footer_rights')}</span>
        <span>·</span>
        <Link href="/cookie-policy" className="hover:text-white transition-colors">
          {t('nav_cookie_policy', 'Cookie Policy')}
        </Link>
        <span>·</span>
        <CookieSettingsTrigger className="text-[10px] sm:text-[11px] font-mono text-white/45 hover:text-white" />
      </div>
    </footer>
  );
}
