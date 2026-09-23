'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import Link from 'next/link';
import { Layers, Activity, Settings } from '@/components/icons';
import LetterGlitch from './LetterGlitch';
import { gsap } from 'gsap';
import { useReducedMotion, usePerfProfile } from '../hooks/useDevice';
import {
  MEGA_NAV_CATEGORIES,
  MEGA_NAV_FOOTER_LINKS,
  type MegaNavCategory,
  type MegaNavPromo,
  type MegaNavLink,
} from '../data/megaNavigation';
import {
  Box,
  Zap,
  Network,
  Truck,
  Navigation,
  Shield,
  Database,
  DollarSign,
  Search,
  X,
  Sparkles,
} from 'lucide-react';

function getIconForFlow(flow?: string) {
  const cls = 'w-5 h-5 text-white/90 group-hover:text-emerald-400 transition-colors duration-200';
  switch (flow) {
    case 'controlPlane':
      return <Layers className={cls} />;
    case 'orderLifecycle':
      return <Activity className={cls} />;
    case 'mutatingHandler':
      return <Settings className={cls} />;
    case 'realtimePipeline':
      return <Zap className={cls} />;
    case 'topologyMap':
      return <Network className={cls} />;
    case 'dispatchBoard':
      return <Truck className={cls} />;
    case 'fleetMap':
      return <Navigation className={cls} />;
    case 'paymentFlow':
      return <Shield className={cls} />;
    case 'dataPlane':
      return <Database className={cls} />;
    case 'financials':
      return <DollarSign className={cls} />;
    case 'aiAssist':
      return <Sparkles className={cls} />;
    default:
      return <Box className={cls} />;
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
  categoryLabel,
  onNavigate,
}: {
  label: string;
  description?: string;
  href: string;
  badge?: 'NEW';
  flow?: string;
  categoryLabel?: string;
  onNavigate: () => void;
}) {
  const isExternal = href.startsWith('http');

  const content = (
    <div
      className="group flex flex-col justify-between p-4 sm:p-5 min-h-[130px] sm:min-h-[140px] bg-[#0c0c0e] hover:bg-[#151518] border border-white/10 hover:border-white/30 relative w-full h-full overflow-hidden transition-all duration-200"
      style={{
        clipPath: 'polygon(0 0, calc(100% - 20px) 0, 100% 20px, 100% 100%, 0 100%)',
      }}
    >
      <div className="flex justify-between items-start relative z-10 gap-2">
        <div className="flex items-center gap-2">
          <div className="p-2 bg-white/5 border border-white/10 group-hover:border-white/20 transition-colors">
            {getIconForFlow(flow)}
          </div>
          {categoryLabel && (
            <span className="font-mono text-[9px] uppercase tracking-wider text-zinc-400">
              {categoryLabel}
            </span>
          )}
        </div>
        <div className="flex items-center gap-1.5">
          {badge && (
            <span className="text-[9px] font-mono font-bold px-2 py-0.5 bg-emerald-400 text-black tracking-wider uppercase">
              {badge}
            </span>
          )}
          <span className="text-zinc-500 group-hover:text-white transition-colors text-xs opacity-0 group-hover:opacity-100 transform translate-x-[-4px] group-hover:translate-x-0 duration-200">
            →
          </span>
        </div>
      </div>
      <div className="mt-4 relative z-10">
        <div className="text-white font-medium text-xs sm:text-sm tracking-wide group-hover:text-white transition-colors flex items-center justify-between">
          <span>{label}</span>
        </div>
        {description && (
          <p className="text-[11px] sm:text-xs text-zinc-400 group-hover:text-zinc-300 line-clamp-2 mt-1 leading-relaxed transition-colors">
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
        className="block w-full h-full outline-none focus-visible:ring-1 focus-visible:ring-white"
        target="_blank"
        rel="noreferrer noopener"
        onClick={onNavigate}
      >
        {content}
      </a>
    );
  }

  return (
    <Link
      href={href}
      className="block w-full h-full outline-none focus-visible:ring-1 focus-visible:ring-white"
      onClick={onNavigate}
      prefetch={false}
    >
      {content}
    </Link>
  );
}

function PromoBlock({ promo, onNavigate }: { promo: MegaNavPromo; onNavigate: () => void }) {
  return (
    <div className="bg-gradient-to-br from-[#141416] to-[#09090b] border border-white/10 p-5 rounded-none relative overflow-hidden group">
      <div className="flex items-center justify-between mb-2">
        <span className="font-mono text-[9px] uppercase tracking-widest text-emerald-400 font-bold">
          SPOTLIGHT
        </span>
        <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
      </div>
      <h2 className="text-sm font-semibold text-white tracking-wide mb-1.5 leading-snug">
        {promo.title}
      </h2>
      <p className="text-xs text-zinc-400 leading-relaxed mb-4 line-clamp-3">
        {promo.body}
      </p>
      <div className="flex flex-wrap gap-2">
        <Link
          href={promo.primaryHref}
          className="px-3 py-1.5 bg-white text-black font-mono text-[11px] font-semibold uppercase tracking-wider hover:bg-zinc-200 transition-colors inline-block"
          onClick={onNavigate}
          prefetch={false}
        >
          {promo.primaryLabel}
        </Link>
        {promo.secondaryHref && promo.secondaryLabel ? (
          <Link
            href={promo.secondaryHref}
            className="px-3 py-1.5 bg-transparent border border-white/20 text-white font-mono text-[11px] font-semibold uppercase tracking-wider hover:bg-white/10 transition-colors inline-block"
            onClick={onNavigate}
            prefetch={false}
          >
            {promo.secondaryLabel}
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
  const [searchQuery, setSearchQuery] = useState('');

  useEffect(() => {
    setPortalReady(true);
  }, []);

  const overlayRef = useRef<HTMLDivElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const railRef = useRef<HTMLUListElement>(null);
  const closeBtnRef = useRef<HTMLButtonElement>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const previousFocusRef = useRef<HTMLElement | null>(null);

  const activeCategory = categories.find((c) => c.id === activeId) ?? categories[0];

  const handleNavigate = useCallback(() => {
    onClose();
  }, [onClose]);

  // Flatten all links across all categories for instant global search
  const allSearchableLinks = useMemo(() => {
    const list: Array<MegaNavLink & { categoryLabel: string; categoryId: string }> = [];
    categories.forEach((cat) => {
      cat.links.forEach((l) => {
        list.push({
          ...l,
          categoryLabel: cat.label,
          categoryId: cat.id,
        });
      });
    });
    return list;
  }, [categories]);

  const filteredLinks = useMemo(() => {
    const q = searchQuery.trim().toLowerCase();
    if (!q) return [];
    return allSearchableLinks.filter(
      (item) =>
        item.label.toLowerCase().includes(q) ||
        item.description?.toLowerCase().includes(q) ||
        item.categoryLabel.toLowerCase().includes(q) ||
        item.slug.toLowerCase().includes(q)
    );
  }, [allSearchableLinks, searchQuery]);

  useEffect(() => {
    if (open) {
      setMounted(true);
      setSearchQuery('');
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

  // Keyboard navigation & slash-to-search shortcut
  useEffect(() => {
    if (!open) return;

    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        if (searchQuery) {
          setSearchQuery('');
        } else {
          onClose();
        }
        return;
      }

      if (e.key === '/' && document.activeElement !== searchInputRef.current) {
        e.preventDefault();
        searchInputRef.current?.focus();
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
  }, [open, onClose, searchQuery]);

  // Entrance animations
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
          { opacity: 1, duration: 0.3, ease: 'power2.out' }
        );
        if (railRef.current) {
          gsap.fromTo(
            railRef.current.children,
            { opacity: 0, x: -8 },
            { opacity: 1, x: 0, duration: 0.35, stagger: 0.03, ease: 'power3.out', delay: 0.05 }
          );
        }
        if (panelRef.current) {
          gsap.fromTo(
            panelRef.current,
            { opacity: 0, y: 10 },
            { opacity: 1, y: 0, duration: 0.35, ease: 'power2.out', delay: 0.08 }
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
        duration: 0.2,
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
      { opacity: 0, y: 6 },
      { opacity: 1, y: 0, duration: 0.25, ease: 'power2.out' }
    );
  }, [activeId, prefersReducedMotion]);

  if (!mounted || !portalReady) return null;

  const isSearching = searchQuery.trim().length > 0;

  const menu = (
    <div
      ref={overlayRef}
      className="mega-menu"
      role="dialog"
      aria-modal="true"
      aria-label="Site navigation"
      aria-hidden={!open}
    >
      {/* Background glow overlay */}
      <div className="absolute inset-0 pointer-events-none z-0 opacity-40 bg-[radial-gradient(ellipse_at_top,rgba(255,255,255,0.12)_0%,transparent_55%)]" />

      <div className="mega-menu__inner relative z-10 flex flex-col h-full max-h-screen pt-4 sm:pt-6 pb-4 px-4 sm:px-8 max-w-[1600px] mx-auto w-full">
        {/* ── Top Console Control Bar ── */}
        <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-4 pb-4 border-b border-white/10 flex-shrink-0">
          {/* Breadcrumb & Scope */}
          <div className="flex items-center gap-2.5">
            <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
            <span className="font-mono text-xs font-bold tracking-[0.18em] text-white uppercase">
              PEGASUS // SYSTEM DIRECTORY
            </span>
            <span className="hidden md:inline font-mono text-[10px] text-zinc-500 uppercase tracking-widest">
              · {categories.length} DOMAINS · 60+ MODULES
            </span>
          </div>

          {/* Quick Search & Close Action */}
          <div className="flex items-center gap-3 flex-1 sm:max-w-md ml-auto">
            <div className="relative flex-1">
              <Search className="w-3.5 h-3.5 text-zinc-500 absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none" />
              <input
                ref={searchInputRef}
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="Filter 60+ modules... (press / to focus)"
                className="w-full bg-[#111113] border border-white/15 focus:border-white text-xs font-mono text-white placeholder:text-zinc-600 pl-8 pr-8 py-2 rounded-none outline-none transition-colors"
              />
              {searchQuery && (
                <button
                  type="button"
                  onClick={() => setSearchQuery('')}
                  className="absolute right-2.5 top-1/2 -translate-y-1/2 text-zinc-500 hover:text-white text-xs font-mono"
                  aria-label="Clear search"
                >
                  ✕
                </button>
              )}
            </div>

            <button
              ref={closeBtnRef}
              type="button"
              onClick={onClose}
              className="shrink-0 flex items-center gap-2 px-3 py-2 border border-white/20 hover:border-white hover:bg-white hover:text-black text-white font-mono text-xs uppercase tracking-wider transition-all rounded-none outline-none focus-visible:ring-1 focus-visible:ring-white"
              aria-label="Close menu"
            >
              <span className="hidden sm:inline">ESC</span>
              <X className="w-3.5 h-3.5" />
            </button>
          </div>
        </div>

        {/* ── Main Directory View ── */}
        <div className="flex flex-1 min-h-0 flex-col lg:flex-row gap-6 lg:gap-8 pt-4 pb-2">
          {/* Left Rail: Categories & Spotlight (Hidden during active search) */}
          {!isSearching ? (
            <div className="w-full lg:w-[260px] xl:w-[280px] shrink-0 flex flex-col justify-between min-h-0 overflow-y-auto pr-1 [scrollbar-width:none]">
              <ul
                ref={railRef}
                className="flex flex-row lg:flex-col gap-1 overflow-x-auto lg:overflow-x-visible pb-2 lg:pb-0 list-none m-0 p-0"
                role="tablist"
                aria-label="Navigation categories"
              >
                {categories.map((category, idx) => {
                  const isActive = category.id === activeId;
                  return (
                    <li key={category.id} className="mega-menu__rail-item shrink-0 lg:shrink" role="presentation">
                      <button
                        type="button"
                        role="tab"
                        aria-selected={isActive}
                        className={`w-full flex items-center justify-between px-3 sm:px-4 py-2 sm:py-2.5 font-mono text-xs sm:text-sm uppercase tracking-wider text-left transition-all rounded-none outline-none focus-visible:ring-1 focus-visible:ring-white ${
                          isActive
                            ? 'bg-white text-black font-bold'
                            : 'text-zinc-400 hover:text-white hover:bg-white/5'
                        }`}
                        onMouseEnter={() => setActiveId(category.id)}
                        onFocus={() => setActiveId(category.id)}
                        onClick={() => setActiveId(category.id)}
                      >
                        <span className="flex items-center gap-2.5">
                          <span
                            className={`text-[10px] font-mono ${
                              isActive ? 'text-black/60' : 'text-zinc-600'
                            }`}
                          >
                            0{idx + 1}
                          </span>
                          <span>{category.label}</span>
                        </span>
                        <span
                          className={`text-[10px] font-mono ${
                            isActive ? 'text-black/70' : 'text-zinc-500'
                          }`}
                        >
                          ({category.links.length})
                        </span>
                      </button>
                    </li>
                  );
                })}
              </ul>

              {/* Spotlight Promo under categories on desktop */}
              {activeCategory?.promo && (
                <div className="mt-4 hidden lg:block">
                  <PromoBlock promo={activeCategory.promo} onNavigate={handleNavigate} />
                </div>
              )}
            </div>
          ) : null}

          {/* Right Panels: Cards Grid (or Search Results) */}
          <div ref={panelRef} className="flex-1 min-h-0 overflow-y-auto pr-2 pb-6" role="tabpanel">
            {isSearching ? (
              <div>
                <div className="flex items-center justify-between pb-3 border-b border-white/10 mb-4">
                  <span className="font-mono text-xs text-zinc-400 uppercase tracking-wider">
                    FOUND {filteredLinks.length} MODULES MATCHING &quot;{searchQuery.toUpperCase()}&quot;
                  </span>
                  <button
                    onClick={() => setSearchQuery('')}
                    className="font-mono text-xs text-zinc-500 hover:text-white transition-colors"
                  >
                    CLEAR [ESC]
                  </button>
                </div>

                {filteredLinks.length > 0 ? (
                  <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-3.5 w-full auto-rows-fr">
                    {filteredLinks.map((link) => (
                      <div key={`search-${link.categoryId}-${link.label}`} className="h-full">
                        <NavLink {...link} onNavigate={handleNavigate} />
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="p-12 text-center border border-dashed border-white/10">
                    <p className="text-zinc-400 font-mono text-sm uppercase tracking-wider">
                      NO MODULES MATCHING &quot;{searchQuery}&quot;
                    </p>
                    <p className="text-zinc-600 text-xs mt-2 font-mono">
                      Try searching for keywords like &quot;dispatch&quot;, &quot;spanner&quot;, &quot;driver&quot;, &quot;telemetry&quot;, or &quot;kafka&quot;
                    </p>
                  </div>
                )}
              </div>
            ) : (
              <div>
                {/* Section Header */}
                <div className="flex items-center justify-between pb-3 border-b border-white/10 mb-4">
                  <h3 className="font-mono text-xs text-zinc-400 uppercase tracking-widest font-semibold">
                    {activeCategory?.label} Modules
                  </h3>
                  {activeCategory?.viewAllHref && (
                    <Link
                      href={activeCategory.viewAllHref}
                      className="font-mono text-[11px] uppercase tracking-wider text-zinc-500 hover:text-white transition-colors inline-flex items-center gap-1"
                      onClick={handleNavigate}
                    >
                      <span>Explore Overview</span>
                      <span>→</span>
                    </Link>
                  )}
                </div>

                {/* Module Cards Grid */}
                <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-3.5 w-full auto-rows-fr">
                  {activeCategory?.links.map((link) => (
                    <div key={`${activeId}-${link.label}`} className="h-full">
                      <NavLink {...link} onNavigate={handleNavigate} />
                    </div>
                  ))}

                  {/* View All / Deep Dive Tile */}
                  <Link
                    href={activeCategory?.viewAllHref ?? '/projects'}
                    className="group relative h-full flex items-center justify-center p-5 min-h-[130px] border border-white/10 rounded-none bg-[#0c0c0e] hover:bg-[#151518] hover:border-white/30 overflow-hidden transition-all duration-200"
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
                    <div className="relative z-10 flex items-center gap-2 text-white font-mono text-xs uppercase tracking-widest font-bold">
                      <span>{activeCategory?.viewAllLabel ?? `ALL ${activeCategory?.label.toUpperCase()}`}</span>
                      <span className="text-zinc-400 group-hover:text-white transition-colors">→</span>
                    </div>
                  </Link>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* ── Footer ── */}
        <footer className="mega-menu__footer pt-3 border-t border-white/10 flex flex-col sm:flex-row items-center justify-between gap-3 text-xs font-mono text-zinc-500 shrink-0">
          <ul className="flex flex-wrap items-center gap-4 sm:gap-6 list-none m-0 p-0">
            {MEGA_NAV_FOOTER_LINKS.map((link) => (
              <li key={link.href}>
                <Link
                  href={link.href}
                  className="text-zinc-400 hover:text-white transition-colors uppercase tracking-wider text-[11px]"
                  onClick={handleNavigate}
                  prefetch={false}
                >
                  {link.label}
                </Link>
              </li>
            ))}
          </ul>
          <div className="flex items-center gap-4 text-[10px] text-zinc-500 tracking-wider uppercase">
            <span className="hidden md:inline">[ESC] Close · [/] Search · [TAB] Navigate</span>
            <span>© {new Date().getFullYear()} PEGASUS OS</span>
          </div>
        </footer>
      </div>
    </div>
  );

  return createPortal(menu, document.body);
}
