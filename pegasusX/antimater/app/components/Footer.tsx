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

  const companyLinks = [
    { name: 'Home', href: '/' },
    { name: t('footer_about', 'About us'), href: '/#about' },
    { name: t('nav_contact', 'Contact us'), href: '/contact' },
    { name: t('nav_demo', 'Request Demo'), href: '/join' },
    { name: t('nav_roles', 'Roles'), href: '/roles' },
    { name: t('nav_solutions', 'Solutions'), href: '/solutions' },
    { name: t('cloud_eco_nav', 'Cloud Ecosystem'), href: '/cloud-ecosystem' },
  ];

  const platformLinks = [
    { name: 'Platform OS', href: '/platform' },
    { name: 'Documentation', href: '/docs' },
    { name: 'Order Lifecycle', href: '/platform/order-lifecycle' },
    { name: 'Logistics Automation', href: '/logistics-automation' },
    { name: 'Trust & Reliability', href: '/platform/trust-reliability' },
    { name: 'Global Logistics', href: '/global-logistics' },
    { name: 'Supply Chain Software', href: '/supply-chain-software' },
  ];

  const resourceLinks = [
    { name: 'Insights & Overview', href: '/platform' },
    { name: 'AI Practices', href: '/platform/how-pegasus-works' },
    { name: 'Design Workflows', href: '/apps-deploy' },
    { name: 'TMS Alternatives', href: '/alternatives' },
    { name: 'Global Markets', href: '/markets' },
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

      {/* ARCHITECTURAL GRID: 2-TIER SWISS GRID (4 COLS X 2 ROWS + DEDICATED MAP COLUMN) */}
      <div className="border-b border-white/10 relative z-10">
        <div className="grid grid-cols-1 lg:grid-cols-5 items-stretch">
          {/* LEFT 4 COLUMNS: ROW 1 (LOGO + NAV) & ROW 2 (SUBSCRIBE + CONTACT) */}
          <div className="lg:col-span-4 flex flex-col border-b lg:border-b-0 lg:border-r border-white/10">
            {/* ROW 1: LOGO + COMPANY + PRODUCT + RESOURCES */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 border-b border-white/10 items-stretch">
              {/* Col 1: PEGASUS LOGO */}
              <div className="p-6 sm:p-8 flex items-center justify-center border-b md:border-b-0 md:border-r border-white/10">
                <Link href="/" prefetch={true} className="inline-flex items-center justify-center group">
                  <Image
                    src="/pegasus-nav.png"
                    width={80}
                    height={66}
                    alt="Pegasus Logo"
                    className="object-contain transition-transform duration-300 group-hover:scale-105"
                    priority
                  />
                </Link>
              </div>

              {/* Col 2: COMPANY */}
              <div className="p-6 sm:p-8 border-b md:border-b-0 md:border-r border-white/10">
                <span className="text-[11px] font-mono tracking-wider text-white/50 uppercase block mb-4">
                  {t('footer_company', 'COMPANY')}
                </span>
                <div className="flex flex-col space-y-2.5 text-xs">
                  {companyLinks.map((link, idx) => (
                    <Link
                      key={link.name}
                      prefetch={true}
                      href={link.href}
                      className={`transition-colors block ${
                        idx === 0
                          ? 'text-[#ff4d2e] hover:text-[#ff6b4a] font-medium'
                          : 'text-white/70 hover:text-white'
                      }`}
                    >
                      {link.name}
                    </Link>
                  ))}
                </div>
              </div>

              {/* Col 3: PRODUCT */}
              <div className="p-6 sm:p-8 border-b md:border-b-0 md:border-r border-white/10">
                <span className="text-[11px] font-mono tracking-wider text-white/50 uppercase block mb-4">
                  {t('footer_platform_title', 'PRODUCT')}
                </span>
                <div className="flex flex-col space-y-2.5 text-xs">
                  {platformLinks.map((link) => (
                    <Link
                      key={link.name}
                      prefetch={true}
                      href={link.href}
                      className="text-white/70 hover:text-white transition-colors block"
                    >
                      {link.name}
                    </Link>
                  ))}
                </div>
              </div>

              {/* Col 4: RESOURCES */}
              <div className="p-6 sm:p-8 border-b md:border-b-0 border-white/10">
                <span className="text-[11px] font-mono tracking-wider text-white/50 uppercase block mb-4">
                  {t('footer_policies', 'RESOURCES')}
                </span>
                <div className="flex flex-col space-y-2.5 text-xs">
                  {resourceLinks.map((link) => (
                    <Link
                      key={link.name}
                      prefetch={true}
                      href={link.href}
                      className="text-white/70 hover:text-white transition-colors block"
                    >
                      {link.name}
                    </Link>
                  ))}
                  <CookieSettingsTrigger className="text-white/70 hover:text-white transition-colors cursor-pointer text-xs text-left whitespace-nowrap block" />
                </div>
              </div>
            </div>

            {/* ROW 2: SUBSCRIBE + ADDRESS + INQUIRIES + SOCIAL */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 items-stretch flex-1">
              {/* Col 1: SUBSCRIBE */}
              <div className="p-6 sm:p-8 flex flex-col justify-between border-b md:border-b-0 md:border-r border-white/10">
                <div>
                  <span className="text-[11px] font-mono tracking-wider text-white/50 uppercase block mb-2">
                    SUBSCRIBE
                  </span>
                  <p className="text-xs text-white/60 leading-relaxed font-sans">
                    Stay ahead with weekly platform updates, logistics engineering, and network releases.
                  </p>
                </div>

                <form onSubmit={handleSubmit} className="mt-4 flex flex-col gap-2">
                  <div className="flex items-stretch border border-white/20 focus-within:border-white/50 transition-colors bg-[#000000]">
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
                      placeholder="Enter your email"
                      disabled={status === 'loading'}
                      required
                      aria-label="Email Address"
                      className="bg-transparent text-white placeholder:text-zinc-500 px-3 py-2 text-[11px] font-mono outline-none flex-1 min-w-0 disabled:opacity-50"
                    />
                    <button
                      type="submit"
                      disabled={status === 'loading'}
                      aria-label="Subscribe"
                      className="px-3 py-2 text-white hover:text-white/80 transition-colors flex items-center justify-center text-sm font-mono border-l border-white/20 disabled:opacity-50 cursor-pointer disabled:cursor-not-allowed shrink-0 hover:bg-white/5"
                    >
                      {status === 'loading' ? (
                        <Loader2 className="w-3.5 h-3.5 animate-spin text-white" />
                      ) : (
                        <span>→</span>
                      )}
                    </button>
                  </div>

                  {status === 'success' && (
                    <div className="flex items-center gap-1.5 text-[11px] font-mono text-emerald-300">
                      <CheckCircle2 className="w-3 h-3 text-emerald-400 shrink-0" />
                      <span>{responseMsg || t('subscribe_success', 'Subscribed')}</span>
                    </div>
                  )}
                  {status === 'error' && (
                    <div className="flex items-center gap-1.5 text-[11px] font-mono text-rose-300">
                      <AlertCircle className="w-3 h-3 text-rose-400 shrink-0" />
                      <span>{responseMsg || t('subscribe_invalid', 'Invalid email')}</span>
                    </div>
                  )}
                </form>
              </div>

              {/* Col 2: ADDRESS */}
              <div className="p-6 sm:p-8 flex flex-col justify-between border-b md:border-b-0 md:border-r border-white/10">
                <div>
                  <span className="text-[11px] font-mono tracking-wider text-white/50 uppercase block mb-3">
                    ADDRESS
                  </span>
                  <div className="text-xs text-white/70 leading-relaxed font-sans">
                    60 Martin Pl, Sydney,<br />
                    NSW 2000, Australia
                  </div>
                </div>

                <div className="pt-4 mt-auto">
                  <a
                    href="https://maps.google.com"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-xs text-white/80 hover:text-white transition-colors inline-flex items-center gap-1 group font-sans"
                  >
                    Direction
                  </a>
                </div>
              </div>

              {/* Col 3: NEW BUSINESS & GENERAL INQUIRIES */}
              <div className="p-6 sm:p-8 flex flex-col justify-between border-b md:border-b-0 md:border-r border-white/10">
                <div>
                  <span className="text-[11px] font-mono tracking-wider text-white/50 uppercase block mb-1.5">
                    NEW BUSINESS
                  </span>
                  <a
                    href="mailto:business@pegasus.io"
                    className="text-xs text-white/80 hover:text-white transition-colors font-sans block"
                  >
                    business@pegasus.io
                  </a>
                </div>

                <div className="pt-4 border-t border-white/10 mt-4">
                  <span className="text-[11px] font-mono tracking-wider text-white/50 uppercase block mb-1.5">
                    GENERAL INQUIRIES
                  </span>
                  <a
                    href="mailto:hello@pegasus.io"
                    className="text-xs text-white/80 hover:text-white transition-colors font-sans block"
                  >
                    hello@pegasus.io
                  </a>
                </div>
              </div>

              {/* Col 4: SOCIAL */}
              <div className="p-6 sm:p-8 flex flex-col justify-between border-b md:border-b-0 border-white/10">
                <div>
                  <span className="text-[11px] font-mono tracking-wider text-white/50 uppercase block mb-3">
                    SOCIAL
                  </span>
                  <div className="grid grid-cols-2 gap-x-4 gap-y-2.5 text-xs font-sans">
                    <div className="flex flex-col space-y-2.5">
                      <Link href="/" className="text-white/70 hover:text-white transition-colors">Website</Link>
                      <a href="https://github.com/pegasus-os" target="_blank" rel="noopener noreferrer" className="text-white/70 hover:text-white transition-colors">GitHub</a>
                      <a href="https://t.me/pegasus" target="_blank" rel="noopener noreferrer" className="text-white/70 hover:text-white transition-colors">Telegram</a>
                    </div>
                    <div className="flex flex-col space-y-2.5">
                      <a href="https://linkedin.com" target="_blank" rel="noopener noreferrer" className="text-white/70 hover:text-white transition-colors">LinkedIn</a>
                      <a href="https://x.com" target="_blank" rel="noopener noreferrer" className="text-white/70 hover:text-white transition-colors">X / Twitter</a>
                      <Link href="/docs" className="text-white/70 hover:text-white transition-colors">API Docs</Link>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* RIGHT 5TH COLUMN: INTERACTIVE WORLD MAP (SPANS FULL 2-ROW HEIGHT) */}
          <div className="lg:col-span-1 p-6 sm:p-8 flex flex-col items-center justify-center relative overflow-hidden bg-[#000000]">
            <div className="w-full flex items-center justify-center max-w-[280px]">
              <ParticleWorldMap variant="dark" showHubs={false} className="w-full" />
            </div>
          </div>
        </div>
      </div>

      {/* 3. THIRD ROW: Full-Width Interactive Particle Text with CAD Blueprint Crosshairs */}
      <div className="py-4 sm:py-6 md:py-8 px-0 flex justify-center items-center overflow-hidden relative z-10 w-full border-b border-white/10 bg-[#000000]">
        {/* CAD Blueprint corner markers */}
        <span className="absolute top-2 left-3 font-mono text-xs text-white/30 select-none">+</span>
        <span className="absolute top-2 right-3 font-mono text-xs text-white/30 select-none">+</span>
        <span className="absolute bottom-2 left-3 font-mono text-xs text-white/30 select-none">+</span>
        <span className="absolute bottom-2 right-3 font-mono text-xs text-white/30 select-none">+</span>

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

      {/* 4. FOURTH ROW: Legal & Policies Bar with Boxed Cells */}
      <div className="flex flex-col sm:flex-row items-stretch justify-between text-[10px] font-mono text-white/50 uppercase relative z-10">
        <div className="px-6 sm:px-8 py-3.5 flex items-center">
          © 2026 PEGASUS LOGISTICS OS. ALL RIGHTS RESERVED. CONCEPT CRAFTED FOR GLOBAL SUPPLY CHAINS.
        </div>

        <div className="flex flex-wrap items-stretch border-t sm:border-t-0 sm:border-l border-white/10">
          <Link
            prefetch={true}
            href="/cookie-policy"
            className="px-5 py-3.5 border-r border-white/10 hover:text-white hover:bg-white/5 transition-colors"
          >
            Privacy Policy
          </Link>
          <Link
            prefetch={true}
            href="/cookie-policy"
            className="px-5 py-3.5 border-r border-white/10 hover:text-white hover:bg-white/5 transition-colors"
          >
            Terms & Conditions
          </Link>
          <div className="px-5 py-3.5 hover:text-white hover:bg-white/5 transition-colors flex items-center">
            <CookieSettingsTrigger className="hover:text-white cursor-pointer uppercase text-[10px] font-mono" />
          </div>
        </div>
      </div>
    </footer>
  );
}
