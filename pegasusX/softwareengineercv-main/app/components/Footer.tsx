'use client';

import Link from 'next/link';
import { Linkedin, Youtube, Instagram } from 'lucide-react';

const PLATFORM_LINKS = [
  { name: 'Platform overview', href: '/platform' },
  { name: 'Order lifecycle', href: '/platform/order-lifecycle' },
  { name: 'How Pegasus works', href: '/platform/how-pegasus-works' },
  { name: 'Trust & reliability', href: '/platform/trust-reliability' },
];

const COMPANY_LINKS = [
  { name: 'Request demo', href: '/join' },
  { name: 'Contact Us', href: '/contact' },
  { name: 'Roles', href: '/roles' },
  { name: 'Modules', href: '/projects' },
];

const POLICIES_LINKS = [
  { name: 'Platform tour', href: '/platform' },
  { name: 'Apps & Deploy', href: '/apps-deploy' },
];

function XIcon({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" className={className} aria-hidden>
      <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z" />
    </svg>
  );
}

const SOCIAL_LINKS = [
  { name: 'X', href: 'https://twitter.com', Icon: XIcon },
  { name: 'LinkedIn', href: 'https://linkedin.com', Icon: Linkedin },
  { name: 'YouTube', href: 'https://youtube.com', Icon: Youtube },
  { name: 'Instagram', href: 'https://instagram.com', Icon: Instagram },
];

export default function Footer() {
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
            placeholder="jane@framer.com"
            className="bg-transparent text-white/80 placeholder:text-white/40 px-4 py-3 outline-none flex-1 text-sm font-mono"
          />
          <button className="bg-[#333] hover:bg-[#444] text-white px-6 py-3 transition-colors flex items-center gap-2 text-sm font-medium border-l border-white/10">
            <span className="opacity-80">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path d="M5 12h14M12 5l7 7-7 7" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
            </span>
            Subscribe
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
            {PLATFORM_LINKS.map(link => (
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
          <h4 className="text-[11px] tracking-[0.2em] text-white/40 mb-8 font-mono uppercase">Company</h4>
          <ul className="space-y-4">
            {COMPANY_LINKS.map(link => (
              <li key={link.name}>
                <Link href={link.href} className="text-white/70 hover:text-white text-sm transition-colors">
                  {link.name}
                </Link>
              </li>
            ))}
          </ul>
        </div>

        {/* Policies col */}
        <div className="p-12">
          <h4 className="text-[11px] tracking-[0.2em] text-white/40 mb-8 font-mono uppercase">Policies</h4>
          <ul className="space-y-4 mb-10">
            {POLICIES_LINKS.map(link => (
              <li key={link.name}>
                <Link href={link.href} className="text-white/70 hover:text-white text-sm transition-colors">
                  {link.name}
                </Link>
              </li>
            ))}
          </ul>

          <div className="flex gap-2">
            {SOCIAL_LINKS.map(({ name, href, Icon }) => (
              <a key={name} href={href} target="_blank" rel="noopener noreferrer" className="w-9 h-9 flex items-center justify-center border border-white/10 bg-white/5 hover:bg-white/10 transition-colors rounded-sm group">
                <Icon className="w-3.5 h-3.5 text-white/60 group-hover:text-white transition-colors" />
              </a>
            ))}
          </div>
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
        ©2026 Pegasus. All rights reserved.
      </div>
    </footer>
  );
}
