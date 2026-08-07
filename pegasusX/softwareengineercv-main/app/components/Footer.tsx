'use client';

import Link from 'next/link';
import { Linkedin, Youtube, Instagram } from 'lucide-react';
import { useLanguage } from '../context/LanguageContext';

export default function Footer() {
  const { t } = useLanguage();

  const platformLinks = [
    { name: t('nav_platform'), href: '/platform' },
    { name: 'Order lifecycle', href: '/platform/order-lifecycle' },
    { name: 'How Pegasus works', href: '/platform/how-pegasus-works' },
    { name: 'Trust & reliability', href: '/platform/trust-reliability' },
  ];

  const companyLinks = [
    { name: t('nav_demo'), href: '/join' },
    { name: t('nav_contact'), href: '/contact' },
    { name: t('nav_roles'), href: '/roles' },
    { name: t('nav_modules'), href: '/projects' },
  ];

  const policiesLinks = [
    { name: t('nav_tour'), href: '/platform' },
    { name: 'Apps & Deploy', href: '/apps-deploy' },
  ];

  return (
    <footer className="bg-[#000000] text-white border-t border-white/5 overflow-hidden font-sans relative">

      {/* Background grain / grid effect */}
      <div className="absolute inset-0 pointer-events-none opacity-20 bg-[linear-gradient(to_right,#80808012_1px,transparent_1px),linear-gradient(to_bottom,#80808012_1px,transparent_1px)] bg-[size:24px_24px]" />
      <div className="absolute inset-0 pointer-events-none opacity-20 bg-[radial-gradient(ellipse_at_center,rgba(255,255,255,0.1)_0%,transparent_70%)]" />

      {/* Top section with input */}
      <div className="border-b border-white/5 flex justify-center py-20 relative z-10">
        <div className="flex bg-[#1a1a1a] border border-white/10 rounded-sm overflow-hidden w-full max-w-[400px]">
          <input
            type="email"
            placeholder={t('footer_email_placeholder')}
            className="bg-transparent text-white/80 placeholder:text-white/40 px-4 py-3 outline-none flex-1 text-sm font-mono"
          />
          <button className="bg-[#333] hover:bg-[#444] text-white px-6 py-3 transition-colors flex items-center gap-2 text-sm font-medium border-l border-white/10">
            <span className="opacity-80">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path d="M5 12h14M12 5l7 7-7 7" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
            </span>
            {t('footer_subscribe_btn')}
          </button>
        </div>
      </div>

      {/* Main footer grid */}
      <div className="grid grid-cols-1 md:grid-cols-4 border-b border-white/5 relative z-10 max-w-[1600px] mx-auto">
        <div className="absolute inset-y-0 left-1/4 border-l border-white/5 hidden md:block" />
        <div className="absolute inset-y-0 left-2/4 border-l border-white/5 hidden md:block" />
        <div className="absolute inset-y-0 left-3/4 border-l border-white/5 hidden md:block" />

        {/* Logo col */}
        <div className="p-16 flex flex-col items-center justify-center max-md:border-b border-white/5">
          <img src="/pegasus.jpg" width={100} height={100} alt="" />
          <span className="mt-6 text-xl font-black tracking-widest text-white uppercase">Pegasus</span>
        </div>

        {/* Platform Links col */}
        <div className="p-12 max-md:border-b border-white/5">
          <h4 className="text-[11px] tracking-[0.2em] text-white/40 mb-8 font-mono uppercase">Platform</h4>
          <ul className="space-y-4">
            {platformLinks.map(link => (
              <li key={link.name}>
                <Link href={link.href} className="text-white/70 hover:text-white text-sm transition-colors">
                  {link.name}
                </Link>
              </li>
            ))}
          </ul>
        </div>

        {/* Company col */}
        <div className="p-12 max-md:border-b border-white/5">
          <h4 className="text-[11px] tracking-[0.2em] text-white/40 mb-8 font-mono uppercase">{t('footer_company')}</h4>
          <ul className="space-y-4">
            {companyLinks.map(link => (
              <li key={link.name}>
                <Link href={link.href} className="text-white/70 hover:text-white text-sm transition-colors">
                  {link.name}
                </Link>
              </li>
            ))}
          </ul>
        </div>

        {/* Resources col */}
        <div className="p-12">
          <h4 className="text-[11px] tracking-[0.2em] text-white/40 mb-8 font-mono uppercase">{t('footer_policies')}</h4>
          <ul className="space-y-4 mb-10">
            {policiesLinks.map(link => (
              <li key={link.name}>
                <Link href={link.href} className="text-white/70 hover:text-white text-sm transition-colors">
                  {link.name}
                </Link>
              </li>
            ))}
          </ul>
        </div>
      </div>

      {/* Huge text */}
      <div className="pt-24 pb-8 px-4 flex justify-center items-center overflow-hidden border-b border-white/5 relative z-10">
        <h1 className="text-[25vw] font-black tracking-tighter leading-[0.75] text-[#e5e5e5] select-none lowercase">
          pegasus
        </h1>
      </div>

      {/* Copyright */}
      <div className="py-6 text-center text-white/40 text-[11px] font-mono relative z-10">
        ©2026 Pegasus. {t('footer_rights')}
      </div>
    </footer>
  );
}
