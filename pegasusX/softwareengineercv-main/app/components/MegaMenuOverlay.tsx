'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import Link from 'next/link';
import { gsap } from 'gsap';
import { useReducedMotion } from '../hooks/useDevice';
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
  Shield 
} from 'lucide-react';

function getIconForFlow(flow?: string) {
  switch (flow) {
    case 'controlPlane': return <Layers className="w-6 h-6 text-blue-500" />;
    case 'orderLifecycle': return <Activity className="w-6 h-6 text-blue-500" />;
    case 'mutatingHandler': return <Settings className="w-6 h-6 text-blue-500" />;
    case 'realtimePipeline': return <Zap className="w-6 h-6 text-blue-500" />;
    case 'topologyMap': return <Network className="w-6 h-6 text-blue-500" />;
    case 'dispatchBoard': return <Truck className="w-6 h-6 text-blue-500" />;
    case 'fleetMap': return <Navigation className="w-6 h-6 text-blue-500" />;
    case 'paymentFlow': return <Shield className="w-6 h-6 text-blue-500" />;
    default: return <Box className="w-6 h-6 text-blue-500" />;
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
    <div className="group flex flex-col justify-between p-5 min-h-[140px] bg-[#111] hover:bg-[#222] transition-colors relative w-full h-full" style={{ clipPath: 'polygon(0 0, calc(100% - 24px) 0, 100% 24px, 100% 100%, 0 100%)' }}>
      <div className="flex justify-between items-start">
        {getIconForFlow(flow)}
        {badge && (
          <span className="text-[10px] font-bold px-2 py-0.5 bg-white text-black rounded-sm tracking-wider">
            {badge}
          </span>
        )}
      </div>
      <div className="mt-6">
        <div className="text-white font-medium text-sm group-hover:text-blue-400 transition-colors">
          {label}
        </div>
        {description && (
          <p className="text-xs text-gray-400 line-clamp-2 mt-1">
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
      <div className="mega-menu__inner">
        <header className="mega-menu__header">
          <span className="mega-menu__label">[NAVIGATION]</span>
          <div className="flex items-center gap-4">
            <span className="mega-menu__search-placeholder" aria-hidden="true" title="Search coming soon">
              ⌕
            </span>
            <button
              ref={closeBtnRef}
              type="button"
              className="mega-menu__close"
              onClick={onClose}
              aria-label="Close navigation menu"
            >
              Close
            </button>
          </div>
        </header>

        <div className="mega-menu__content-wrap">
          <ul ref={railRef} className="mega-menu__rail" role="tablist" aria-label="Navigation categories">
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

          <div ref={panelRef} className="mega-menu__panels" role="tabpanel">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 w-full auto-rows-fr h-fit max-h-[100%] overflow-y-auto pb-8 pr-4">
              {activeCategory?.links.map((link) => (
                <div key={`${activeId}-${link.label}`} className="h-full">
                  <NavLink {...link} onNavigate={handleNavigate} />
                </div>
              ))}
              <div className="h-full flex items-center justify-center p-5 min-h-[140px] border border-white/10 rounded-lg hover:bg-white/5 transition-colors">
                <Link
                  href={activeCategory?.viewAllHref ?? '/projects'}
                  className="text-white font-bold tracking-widest uppercase text-sm"
                  onClick={handleNavigate}
                  prefetch={false}
                >
                  {activeCategory?.viewAllLabel ?? 'VIEW ALL'} &gt;
                </Link>
              </div>
            </div>
          </div>
        </div>

        <PromoBlock promo={promo} onNavigate={handleNavigate} />
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
