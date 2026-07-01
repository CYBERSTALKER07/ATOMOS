'use client';

import React, { useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import { gsap } from 'gsap';

import GigaMenuDropdown from './GigaMenuDropdown';
import MegaMenuOverlay from './MegaMenuOverlay';
import { MEGA_NAV_CATEGORIES, MEGA_NAV_FOOTER_LINKS, type MegaNavCategory } from '../data/megaNavigation';

export type PillNavItem = {
  label: string;
  href: string;
  ariaLabel?: string;
};

export interface PillNavProps {
  logo: string;
  logoAlt?: string;
  items: PillNavItem[];
  activeHref?: string;
  className?: string;
  ease?: string;
  baseColor?: string;
  pillColor?: string;
  hoveredPillTextColor?: string;
  pillTextColor?: string;
  onMobileMenuClick?: () => void;
  initialLoadAnimation?: boolean;
  showMenuButton?: boolean;
  categories?: MegaNavCategory[];
}

const PillNav: React.FC<PillNavProps> = ({
  logo,
  logoAlt = 'Logo',
  items,
  activeHref,
  className = '',
  ease = 'power3.easeOut',
  baseColor = '#000000',
  pillColor = '#ffffff',
  hoveredPillTextColor = '#000000',
  pillTextColor,
  onMobileMenuClick,
  initialLoadAnimation = true,
  showMenuButton = false,
  categories,
}) => {
  const resolvedPillTextColor = pillTextColor ?? baseColor;
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
  const [megaMenuOpen, setMegaMenuOpen] = useState(false);
  const [activeCategory, setActiveCategory] = useState<MegaNavCategory | null>(null);

  const displayItems = categories ? categories.map(c => ({ label: c.label, href: c.viewAllHref || '#', id: c.id })) : items;

  const circleRefs = useRef<Array<HTMLSpanElement | null>>([]);
  const tlRefs = useRef<Array<gsap.core.Timeline | null>>([]);
  const activeTweenRefs = useRef<Array<gsap.core.Tween | null>>([]);
  const logoImgRef = useRef<HTMLImageElement | null>(null);
  const hamburgerRef = useRef<HTMLButtonElement | null>(null);
  const mobileMenuRef = useRef<HTMLDivElement | null>(null);
  const navItemsRef = useRef<HTMLDivElement | null>(null);
  const logoRef = useRef<HTMLAnchorElement | HTMLElement | null>(null);
  const wrapperRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (
        wrapperRef.current &&
        !wrapperRef.current.contains(event.target as Node)
      ) {
        setActiveCategory(null);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  useEffect(() => {
    const hamburger = hamburgerRef.current;
    if (hamburger) {
      const lines = hamburger.querySelectorAll('.hamburger-line');
      const isOpen = showMenuButton ? megaMenuOpen : isMobileMenuOpen;
      if (isOpen) {
        gsap.to(lines[0], { rotation: 45, y: 6, duration: 0.3, ease });
        gsap.to(lines[1], { opacity: 0, duration: 0.3, ease });
        if (lines[2]) gsap.to(lines[2], { rotation: -45, y: -6, duration: 0.3, ease });
      } else {
        gsap.to(lines[0], { rotation: 0, y: 0, duration: 0.3, ease });
        gsap.to(lines[1], { opacity: 1, duration: 0.3, ease });
        if (lines[2]) gsap.to(lines[2], { rotation: 0, y: 0, duration: 0.3, ease });
      }
    }
  }, [megaMenuOpen, isMobileMenuOpen, showMenuButton, ease]);

  useEffect(() => {
    const layout = () => {
      circleRefs.current.forEach(circle => {
        if (!circle?.parentElement) return;

        const pill = circle.parentElement as HTMLElement;
        const rect = pill.getBoundingClientRect();
        const { width: w, height: h } = rect;
        const R = ((w * w) / 4 + h * h) / (2 * h);
        const D = Math.ceil(2 * R) + 2;
        const delta = Math.ceil(R - Math.sqrt(Math.max(0, R * R - (w * w) / 4))) + 1;
        const originY = D - delta;

        circle.style.width = `${D}px`;
        circle.style.height = `${D}px`;
        circle.style.bottom = `-${delta}px`;

        gsap.set(circle, {
          xPercent: -50,
          scale: 0,
          transformOrigin: `50% ${originY}px`
        });

        const label = pill.querySelector<HTMLElement>('.pill-label');
        const white = pill.querySelector<HTMLElement>('.pill-label-hover');

        if (label) gsap.set(label, { y: 0 });
        if (white) gsap.set(white, { y: h + 12, opacity: 0 });

        const index = circleRefs.current.indexOf(circle);
        if (index === -1) return;

        tlRefs.current[index]?.kill();
        const tl = gsap.timeline({ paused: true });

        tl.to(circle, { scale: 1.2, xPercent: -50, duration: 2, ease, overwrite: 'auto' }, 0);

        if (label) {
          tl.to(label, { y: -(h + 8), duration: 2, ease, overwrite: 'auto' }, 0);
        }

        if (white) {
          gsap.set(white, { y: Math.ceil(h + 100), opacity: 0 });
          tl.to(white, { y: 0, opacity: 1, duration: 2, ease, overwrite: 'auto' }, 0);
        }

        tlRefs.current[index] = tl;
      });
    };

    layout();

    const onResize = () => layout();
    window.addEventListener('resize', onResize);

    if (document.fonts) {
      document.fonts.ready.then(layout).catch(() => { });
    }

    const menu = mobileMenuRef.current;
    if (menu) {
      gsap.set(menu, { visibility: 'hidden', opacity: 0, scaleY: 1, y: 0 });
    }

    if (initialLoadAnimation) {
      const logo = logoRef.current;
      const navItems = navItemsRef.current;

      if (logo) {
        gsap.set(logo, { scale: 0 });
        gsap.to(logo, {
          scale: 1,
          duration: 0.6,
          ease
        });
      }

      if (navItems) {
        gsap.set(navItems, { opacity: 0, x: -8 });
        gsap.to(navItems, {
          opacity: 1,
          x: 0,
          duration: 0.6,
          ease
        });
      }
    }

    return () => window.removeEventListener('resize', onResize);
  }, [items, ease, initialLoadAnimation, showMenuButton]);

  const handleEnter = (i: number) => {
    const tl = tlRefs.current[i];
    if (!tl) return;
    activeTweenRefs.current[i]?.kill();
    activeTweenRefs.current[i] = tl.tweenTo(tl.duration(), {
      duration: 0.3,
      ease,
      overwrite: 'auto'
    });
  };

  const handleLeave = (i: number) => {
    const tl = tlRefs.current[i];
    if (!tl) return;
    activeTweenRefs.current[i]?.kill();
    activeTweenRefs.current[i] = tl.tweenTo(0, {
      duration: 0.2,
      ease,
      overwrite: 'auto'
    });
  };



  const openMegaMenu = () => {
    setMegaMenuOpen(true);
    onMobileMenuClick?.();
  };

  const toggleMobileMenu = () => {
    if (showMenuButton) {
      setMegaMenuOpen(!megaMenuOpen);
      if (!megaMenuOpen) onMobileMenuClick?.();
      return;
    }

    const newState = !isMobileMenuOpen;
    setIsMobileMenuOpen(newState);

    const menu = mobileMenuRef.current;

    if (menu) {
      if (newState) {
        gsap.set(menu, { visibility: 'visible' });
        gsap.fromTo(
          menu,
          { opacity: 0, y: 10, scaleY: 1 },
          {
            opacity: 1,
            y: 0,
            scaleY: 1,
            duration: 0.3,
            ease,
            transformOrigin: 'top center'
          }
        );
      } else {
        gsap.to(menu, {
          opacity: 0,
          y: 10,
          scaleY: 1,
          duration: 0.2,
          ease,
          transformOrigin: 'top center',
          onComplete: () => {
            gsap.set(menu, { visibility: 'hidden' });
          }
        });
      }
    }

    onMobileMenuClick?.();
  };

  const isExternalLink = (href: string) =>
    href.startsWith('http://') ||
    href.startsWith('https://') ||
    href.startsWith('//') ||
    href.startsWith('mailto:') ||
    href.startsWith('tel:') ||
    href.startsWith('#');

  const cssVars = {
    ['--base']: baseColor,
    ['--pill-bg']: pillColor,
    ['--hover-text']: hoveredPillTextColor,
    ['--pill-text']: resolvedPillTextColor,
    ['--nav-h']: '42px',
    ['--logo']: '36px',
    ['--pill-pad-x']: '12px',
    ['--pill-gap']: '2px'
  } as React.CSSProperties;

  const basePillClasses =
    'relative overflow-hidden inline-flex items-center justify-center h-full no-underline rounded-full box-border font-semibold text-[12px] leading-[0] uppercase tracking-[0.2px] whitespace-nowrap cursor-pointer px-0 focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-white outline-none';

  const pillStyleBase: React.CSSProperties = {
    background: 'var(--pill-bg, #fff)',
    color: 'var(--pill-text, var(--base, #000))',
    paddingLeft: 'var(--pill-pad-x)',
    paddingRight: 'var(--pill-pad-x)',
  };

  return (
    <div
      ref={wrapperRef}
      className="fixed top-0 left-0 right-0 z-[10002] bg-black border-b border-white/10"
      onBlur={(e) => {
        if (!e.currentTarget.contains(e.relatedTarget as Node)) {
          setActiveCategory(null);
        }
      }}
    >
      <div className="relative pointer-events-none px-4 py-3">
        <nav
          className={`pill-nav pointer-events-auto w-full flex items-center gap-2 min-w-0 max-w-7xl mx-auto ${className}`}
          aria-label="Primary"
          style={cssVars}
        >
          <Link
            href="/"
            aria-label="Home"
            ref={el => {
              logoRef.current = el;
            }}
            className="shrink-0 inline-flex items-center justify-center overflow-hidden focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-white outline-none"
            style={{
              width: '64px',
              height: '64px',
              background: 'var(--base, #000)'
            }}
          >
            <img
              src="/pegasus.jpg"
              alt={logoAlt}
              ref={logoImgRef}
              className="w-full h-full object-contain"
            />
          </Link>

          <div
            ref={navItemsRef}
            className="relative hidden md:flex min-w-0 flex-1 items-center rounded-full overflow-hidden"
            style={{
              height: 'var(--nav-h)',
              background: 'var(--base, #000)'
            }}
          >
            <ul
              role="menubar"
              className="list-none flex items-stretch m-0 p-[3px] h-full w-full min-w-0 overflow-x-auto [scrollbar-width:none] [-ms-overflow-style:none] [&::-webkit-scrollbar]:hidden"
              style={{ gap: 'var(--pill-gap)' }}
            >
              {displayItems.map((item, i) => {
                const isActive = activeHref === item.href;

                const pillStyle: React.CSSProperties = { ...pillStyleBase };

                const PillContent = (
                  <>
                    <span
                      className="hover-circle absolute left-1/2 bottom-0 rounded- z-[1] block pointer-events-none"
                      style={{
                        background: 'var(--base, #000)',
                        willChange: 'transform'
                      }}
                      aria-hidden="true"
                      ref={el => {
                        circleRefs.current[i] = el;
                      }}
                    />
                    <span className="label-stack relative inline-block leading-[1] z-[2]">
                      <span
                        className="pill-label relative z-[2] inline-block leading-[1]"
                        style={{ willChange: 'transform' }}
                      >
                        {item.label}
                      </span>
                      <span
                        className="pill-label-hover absolute left-0 top-0 z-[3] inline-block"
                        style={{
                          color: 'var(--hover-text, #fff)',
                          willChange: 'transform, opacity'
                        }}
                        aria-hidden="true"
                      >
                        {item.label}
                      </span>
                    </span>
                    {isActive && (
                      <span
                        className="absolute left-1/2 -bottom-[6px] -translate-x-1/2 w-3 h-3 rounded-full z-[4]"
                        style={{ background: 'var(--base, #000)' }}
                        aria-hidden="true"
                      />
                    )}
                  </>
                );

                return (
                  <li key={item.href} role="none" className="flex h-full">
                    {isExternalLink(item.href) ? (
                      <a
                        role="menuitem"
                        href={item.href}
                        aria-current={isActive ? 'page' : undefined}
                        className={basePillClasses}
                        style={pillStyle}
                        onMouseEnter={() => handleEnter(i)}
                        onMouseLeave={() => handleLeave(i)}
                        onFocus={() => handleEnter(i)}
                        onBlur={() => handleLeave(i)}
                        onClick={(e) => {
                          if (categories) {
                            e.preventDefault();
                            setActiveCategory(activeCategory === categories[i] ? null : categories[i]);
                          }
                        }}
                      >
                        {PillContent}
                      </a>
                    ) : (
                      <Link
                        role="menuitem"
                        href={item.href}
                        aria-current={isActive ? 'page' : undefined}
                        className={basePillClasses}
                        style={pillStyle}
                        onMouseEnter={() => handleEnter(i)}
                        onMouseLeave={() => handleLeave(i)}
                        onFocus={() => handleEnter(i)}
                        onBlur={() => handleLeave(i)}
                      >
                        {PillContent}
                      </Link>
                    )}
                  </li>
                );
              })}
              {/* Text menu button removed in favor of hamburger */}
            </ul>
          </div>

          <div className="shrink-0 ml-auto flex items-center gap-0 pointer-events-auto">
            <button
              ref={hamburgerRef}
              onClick={toggleMobileMenu}
              aria-label={showMenuButton ? 'Toggle site menu' : 'Toggle navigation menu'}
              aria-expanded={showMenuButton ? megaMenuOpen : isMobileMenuOpen}
              className={`${showMenuButton ? '' : 'md:hidden'} flex items-center gap-3 px-4 py-2 border border-white text-white hover:bg-white hover:text-black transition-colors group outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-white`}
            >
              <span className="text-sm font-medium tracking-wider">MENU</span>
              <div className="flex flex-col items-center justify-center gap-[4px] w-5">
                <span
                  className={`hamburger-line w-5 h-[2px] origin-center transition-all duration-300 ${(showMenuButton ? megaMenuOpen : isMobileMenuOpen) ? 'rotate-45 translate-y-[6px]' : ''}`}
                  style={{ background: 'currentColor' }}
                />
                <span
                  className={`hamburger-line w-5 h-[2px] origin-center transition-all duration-300 ${(showMenuButton ? megaMenuOpen : isMobileMenuOpen) ? 'opacity-0' : ''}`}
                  style={{ background: 'currentColor' }}
                />
                <span
                  className={`hamburger-line w-5 h-[2px] origin-center transition-all duration-300 ${(showMenuButton ? megaMenuOpen : isMobileMenuOpen) ? '-rotate-45 -translate-y-[6px]' : ''}`}
                  style={{ background: 'currentColor' }}
                />
              </div>
            </button>
            <Link
              href="/contact"
              className="hidden sm:block px-4 py-2 bg-white text-black border border-white text-sm font-medium tracking-wider hover:bg-gray-200 transition-colors outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-white"
            >
              REQUEST DEMO
            </Link>
          </div>
        </nav>

        {categories && (
          <GigaMenuDropdown
            activeCategory={activeCategory}
            onMouseEnter={() => { }}
            onMouseLeave={() => { }}
          />
        )}

        {showMenuButton ? (
          <MegaMenuOverlay
            open={megaMenuOpen}
            onClose={() => setMegaMenuOpen(false)}
            categories={categories || MEGA_NAV_CATEGORIES}
          />
        ) : null}

        {(!showMenuButton && !categories) ? (
          <div
            ref={mobileMenuRef}
            className="md:hidden pointer-events-auto absolute top-[calc(var(--nav-h)+0.75rem)] left-0 right-0 rounded-[27px] shadow-[0_8px_32px_rgba(0,0,0,0.12)] z-[998] origin-top max-h-[70vh] overflow-y-auto"
            style={{
              ...cssVars,
              background: 'var(--base, #000)'
            }}
          >
            <ul className="list-none m-0 p-[3px] flex flex-col gap-[3px]">
              {displayItems.map(item => {
                const defaultStyle: React.CSSProperties = {
                  background: 'var(--pill-bg, #fff)',
                  color: 'var(--pill-text, #000)'
                };
                const hoverIn = (e: React.MouseEvent<HTMLAnchorElement> | React.FocusEvent<HTMLAnchorElement>) => {
                  e.currentTarget.style.background = 'var(--base)';
                  e.currentTarget.style.color = 'var(--hover-text, #fff)';
                };
                const hoverOut = (e: React.MouseEvent<HTMLAnchorElement> | React.FocusEvent<HTMLAnchorElement>) => {
                  e.currentTarget.style.background = 'var(--pill-bg, #fff)';
                  e.currentTarget.style.color = 'var(--pill-text, #000)';
                };

                const linkClasses =
                  'block py-3 px-4 text-[16px] font-medium rounded-[50px] transition-all duration-200 ease-[cubic-bezier(0.25,0.1,0.25,1)] focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-white outline-none';

                return (
                  <li key={item.href}>
                    {isExternalLink(item.href) ? (
                      <a
                        href={item.href}
                        className={linkClasses}
                        style={defaultStyle}
                        onMouseEnter={hoverIn}
                        onMouseLeave={hoverOut}
                        onFocus={hoverIn}
                        onBlur={hoverOut}
                        onClick={() => setIsMobileMenuOpen(false)}
                      >
                        {item.label}
                      </a>
                    ) : (
                      <Link
                        href={item.href}
                        className={linkClasses}
                        style={defaultStyle}
                        onMouseEnter={hoverIn}
                        onMouseLeave={hoverOut}
                        onFocus={hoverIn}
                        onBlur={hoverOut}
                        onClick={() => setIsMobileMenuOpen(false)}
                      >
                        {item.label}
                      </Link>
                    )}
                  </li>
                );
              })}
            </ul>
          </div>
        ) : null}
      </div>
    </div>
  );
};

export default PillNav;
