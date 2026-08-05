'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { ArrowRight, ChevronRight, X } from 'lucide-react';
import { cn } from '@/lib/utils';
import LetterGlitch from './LetterGlitch';
import { gsap } from 'gsap';
import { useReducedMotion, usePerfProfile } from '../hooks/useDevice';
import {
  DEFAULT_MEGA_PROMO,
  MEGA_NAV_CATEGORIES,
  MEGA_NAV_FOOTER_LINKS,
  type MegaNavCategory,
  type MegaNavPromo,
} from '../data/megaNavigation';
import {
  Box,
  Layers,
  Zap,
  Settings,
  Activity,
  Network,
  Truck,
  Navigation,
  Shield,
  Database,
  DollarSign
} from 'lucide-react';

function getIconForFlow(flow?: string) {
  const cls = "w-6 h-6 text-white group-hover:text-black transition-colors duration-300";
  switch (flow) {
    case 'controlPlane': return <Layers className={cls} />;
    case 'orderLifecycle': return <Activity className={cls} />;
    case 'mutatingHandler': return <Settings className={cls} />;
    case 'realtimePipeline': return <Zap className={cls} />;
    case 'topologyMap': return <Network className={cls} />;
    case 'dispatchBoard': return <Truck className={cls} />;
    case 'fleetMap': return <Navigation className={cls} />;
    case 'paymentFlow': return <Shield className={cls} />;
    case 'dataPlane': return <Database className={cls} />;
    case 'financials': return <DollarSign className={cls} />;
    default: return <Box className={cls} />;
  }
}

const FOCUSABLE =
  'a[href], button:not([disabled]), textarea, input, select, [tabindex]:not([tabindex="-1"])';

type MegaMenuOverlayProps = {
  open: boolean;
  onClose: () => void;
  categories?: MegaNavCategory[];
};

function NavLink({
  label,
  description,
  href,
  badge,
  flow,
  onNavigate,
}: {
  label: string;
  description?: string;
  href: string;
  badge?: 'NEW';
  flow?: string;
  onNavigate: () => void;
}) {
  const isExternal = href.startsWith('http');

  const content = (
    <div className="group flex flex-col justify-between p-5 min-h-[140px] bg-[#111] relative w-full h-full overflow-hidden" style={{ clipPath: 'polygon(0 0, calc(100% - 24px) 0, 100% 24px, 100% 100%, 0 100%)' }}>
      <div className="absolute inset-0 w-full h-full bg-white origin-left transform scale-x-0 group-hover:scale-x-100 transition-transform duration-300 ease-[cubic-bezier(0.25,0.1,0.25,1)] z-0" />
      <div className="flex justify-between items-start relative z-10">
        {getIconForFlow(flow)}
        {badge && (
          <span className="text-[10px] font-bold px-2 py-0.5 bg-white text-black group-hover:bg-black group-hover:text-white transition-colors duration-300 rounded-sm tracking-wider">
            {badge}
          </span>
        )}
      </div>
      <div className="mt-6 relative z-10">
        <div className="text-white group-hover:text-black font-medium text-sm transition-colors duration-300">
          {label}
        </div>
        {description && (
          <p className="text-xs text-gray-400 group-hover:text-gray-800 line-clamp-2 mt-1 transition-colors duration-300">
            {description}
          </p>
        )}
      </div>
    </div>
  );

  if (isExternal) {
    return (
      <a
        href={href}
        className="block w-full h-full"
        target="_blank"
        rel="noreferrer noopener"
        onClick={onNavigate}
      >
        {content}
      </a>
    );
  }

  return (
    <Link href={href} className="block w-full h-full" onClick={onNavigate} prefetch={false}>
      {content}
    </Link>
  );
}

function PromoBlock({ promo, onNavigate }: { promo: MegaNavPromo; onNavigate: () => void }) {
  return (
    <div className="mega-menu__promo">
      <h2 className="mega-menu__promo-title">{promo.title}</h2>
      <p className="mega-menu__promo-body">{promo.body}</p>
      <div className="mega-menu__promo-actions">
        <Link
          href={promo.primaryHref}
          className="mega-menu__promo-link"
          onClick={onNavigate}
          prefetch={false}
        >
          {promo.primaryLabel} &gt;
        </Link>
        {promo.secondaryHref && promo.secondaryLabel ? (
          <Link
            href={promo.secondaryHref}
            className="mega-menu__promo-link"
            onClick={onNavigate}
            prefetch={false}
          >
            {promo.secondaryLabel} &gt;
          </Link>
        ) : null}
      </div>
    </div>
  );
}

export default function MegaMenuOverlay({
  open,
  onClose,
  categories = MEGA_NAV_CATEGORIES,
}: MegaMenuOverlayProps) {
  const prefersReducedMotion = useReducedMotion();
  const { allowHoverFx } = usePerfProfile();
  const [activeId, setActiveId] = useState(categories[0]?.id ?? 'platform');
  const [mounted, setMounted] = useState(open);
  const [portalReady, setPortalReady] = useState(false);

  useEffect(() => {
    setPortalReady(true);
  }, []);

  const overlayRef = useRef<HTMLDivElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const railRef = useRef<HTMLUListElement>(null);
  const closeBtnRef = useRef<HTMLButtonElement>(null);
  const previousFocusRef = useRef<HTMLElement | null>(null);

  const activeCategory = categories.find((c) => c.id === activeId) ?? categories[0];
  const promo = activeCategory?.promo ?? DEFAULT_MEGA_PROMO;

  const handleNavigate = useCallback(() => {
    onClose();
  }, [onClose]);

  useEffect(() => {
    if (open) {
      setMounted(true);
      previousFocusRef.current = document.activeElement as HTMLElement | null;
      document.body.style.overflow = 'hidden';
    } else if (mounted) {
      document.body.style.overflow = '';
      previousFocusRef.current?.focus?.();
    }
    return () => {
      document.body.style.overflow = '';
    };
  }, [open, mounted]);

  useEffect(() => {
    if (!open) return;
    const timer = window.setTimeout(() => closeBtnRef.current?.focus(), 50);
    return () => window.clearTimeout(timer);
  }, [open]);

  useEffect(() => {
    if (!open) return;

    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        onClose();
        return;
      }

      if (e.key !== 'Tab' || !overlayRef.current) return;

      const focusable = Array.from(
        overlayRef.current.querySelectorAll<HTMLElement>(FOCUSABLE)
      ).filter((el) => !el.hasAttribute('disabled'));

      if (focusable.length === 0) return;

      const first = focusable[0];
      const last = focusable[focusable.length - 1];

      if (e.shiftKey && document.activeElement === first) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && document.activeElement === last) {
        e.preventDefault();
        first.focus();
      }
    };

    document.addEventListener('keydown', onKeyDown);
    return () => document.removeEventListener('keydown', onKeyDown);
  }, [open, onClose]);

  useEffect(() => {
    if (!mounted || !overlayRef.current) return;

    const ctx = gsap.context(() => {
      if (open) {
        if (prefersReducedMotion) {
          gsap.set(overlayRef.current, { opacity: 1, visibility: 'visible' });
          return;
        }
        gsap.fromTo(
          overlayRef.current,
          { opacity: 0 },
          { opacity: 1, duration: 0.35, ease: 'power2.out' }
        );
        if (railRef.current) {
          gsap.fromTo(
            railRef.current.children,
            { opacity: 0, y: 12 },
            { opacity: 1, y: 0, duration: 0.4, stagger: 0.04, ease: 'power3.out', delay: 0.05 }
          );
        }
        if (panelRef.current) {
          gsap.fromTo(
            panelRef.current,
            { opacity: 0 },
            { opacity: 1, duration: 0.35, ease: 'power2.out', delay: 0.1 }
          );
        }
      }
    }, overlayRef);

    return () => ctx.revert();
  }, [open, mounted, prefersReducedMotion]);

  useEffect(() => {
    if (!open && mounted && overlayRef.current) {
      if (prefersReducedMotion) {
        setMounted(false);
        return;
      }
      gsap.to(overlayRef.current, {
        opacity: 0,
        duration: 0.25,
        ease: 'power2.in',
        onComplete: () => setMounted(false),
      });
      return;
    }

    if (open && overlayRef.current) {
      gsap.set(overlayRef.current, { opacity: 1, pointerEvents: 'auto' });
    }
  }, [open, mounted, prefersReducedMotion]);

  useEffect(() => {
    if (!panelRef.current || prefersReducedMotion) return;
    gsap.fromTo(
      panelRef.current,
      { opacity: 0, y: 8 },
      { opacity: 1, y: 0, duration: 0.3, ease: 'power2.out' }
    );
  }, [activeId, prefersReducedMotion]);

  if (!mounted || !portalReady) return null;

  const menu = (
    <div
      ref={overlayRef}
      className="mega-menu"
      role="dialog"
      aria-modal="true"
      aria-label="Site navigation"
      aria-hidden={!open}
    >
      <div className="absolute inset-0 pointer-events-none z-0 opacity-40 bg-[radial-gradient(ellipse_at_top,rgba(255,255,255,0.12)_0%,transparent_55%)]" />
      <div className="mega-menu__inner relative z-10 !pt-[80px]">

        <div className="flex flex-1 min-h-0 flex-col lg:flex-row gap-8 lg:gap-16 w-full max-w-7xl mx-auto pt-8">
          {/* Left Column: Rail + Promo */}
          <div className="w-full lg:w-[20rem] flex-shrink-0 flex flex-col justify-between min-h-0">
            <ul ref={railRef} className="mega-menu__rail flex-1" role="tablist" aria-label="Navigation categories">
              {categories.map((category) => {
                const isActive = category.id === activeId;
                return (
                  <li key={category.id} className="mega-menu__rail-item" role="presentation">
                    <button
                      type="button"
                      role="tab"
                      aria-selected={isActive}
                      className={`mega-menu__rail-btn${isActive ? ' mega-menu__rail-btn--active' : ''}`}
                      onMouseEnter={() => setActiveId(category.id)}
                      onFocus={() => setActiveId(category.id)}
                      onClick={() => setActiveId(category.id)}
                    >
                      <span className="mega-menu__rail-chevron" aria-hidden="true">
                        ›
                      </span>
                      {category.label}
                    </button>
                  </li>
                );
              })}
            </ul>
            <div className="hidden lg:block mt-auto pt-8">
              <PromoBlock promo={promo} onNavigate={handleNavigate} />
            </div>
          </div>

          {/* Right Column: Panels */}
          <div ref={panelRef} className="flex-1 overflow-y-auto pb-12 pr-4 min-h-0" role="tabpanel">
            <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4 w-full auto-rows-fr h-fit">
              {activeCategory?.links.map((link) => (
                <div key={`${activeId}-${link.label}`} className="h-full">
                  <NavLink {...link} onNavigate={handleNavigate} />
                </div>
              ))}
              <Link
                href={activeCategory?.viewAllHref ?? '/projects'}
                className="group relative h-full flex items-center justify-center p-5 min-h-[140px] border border-white/10 rounded-lg bg-[#111] hover:bg-[#222] overflow-hidden transition-colors"
                onClick={handleNavigate}
                prefetch={false}
              >
                <div className="absolute inset-0 opacity-0 group-hover:opacity-100 transition-opacity duration-500 pointer-events-none z-0">
                  {allowHoverFx ? (
                    <LetterGlitch
                      glitchSpeed={50}
                      centerVignette={true}
                      outerVignette={true}
                      smooth={true}
                    />
                  ) : (
                    <div className="absolute inset-0 bg-white/5" />
                  )}
                </div>
                <span className="text-white font-bold tracking-widest uppercase text-sm relative z-10">
                  {activeCategory?.viewAllLabel ?? 'VIEW ALL'} &gt;
                </span>
              </Link>
            </div>
            {/* Mobile Promo */}
            <div className="block lg:hidden mt-12 shrink-0">
              <PromoBlock promo={promo} onNavigate={handleNavigate} />
            </div>
          </div>
        </div>
      </div>

      <footer className="mega-menu__footer">
        <ul className="mega-menu__footer-links">
          {MEGA_NAV_FOOTER_LINKS.map((link) => (
            <li key={link.href}>
              <Link
                href={link.href}
                className="mega-menu__footer-link"
                onClick={handleNavigate}
                prefetch={false}
              >
                {link.label}
              </Link>
            </li>
          ))}
        </ul>
        <span>© {new Date().getFullYear()} Pegasus</span>
      </footer>
    </div>
  );

  return createPortal(menu, document.body);
}
