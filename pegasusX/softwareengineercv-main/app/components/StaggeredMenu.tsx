'use client';

import React, { useEffect, useRef, useState } from 'react';
import { gsap } from 'gsap';
import Link from 'next/link';
import './StaggeredMenu.css';

interface StaggeredMenuItem {
  label: string;
  link: string;
  ariaLabel?: string;
}

interface StaggeredMenuProps {
  isOpen: boolean;
  onClose: () => void;
  position?: 'left' | 'right' | 'top';
  items?: StaggeredMenuItem[];
  socialItems?: StaggeredMenuItem[];
  displaySocials?: boolean;
  displayItemNumbering?: boolean;
  fontFamily?: string;
}

export default function StaggeredMenu({
  isOpen,
  onClose,
  position = 'top',
  items = [],
  socialItems = [],
  displaySocials = true,
  displayItemNumbering = true,
  fontFamily = 'var(--font-body), sans-serif',
}: StaggeredMenuProps) {
  const [shouldRender, setShouldRender] = useState(isOpen);

  const wrapperRef = useRef<HTMLDivElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const itemsRef = useRef<Array<HTMLAnchorElement | null>>([]);
  const socialsRef = useRef<Array<HTMLAnchorElement | null>>([]);
  const prelayersRef = useRef<Array<HTMLDivElement | null>>([]);
  const tlRef = useRef<gsap.core.Timeline | null>(null);

  // Sync shouldRender with isOpen for unmount/mount animation
  useEffect(() => {
    if (isOpen) {
      setShouldRender(true);
    }
  }, [isOpen]);

  useEffect(() => {
    if (!shouldRender || !panelRef.current) return;

    gsap.set(panelRef.current, {
      opacity: 0,
      xPercent: position === 'left' ? -100 : position === 'right' ? 100 : 0,
      yPercent: position === 'top' ? -100 : 0,
    });
    
    gsap.set(prelayersRef.current, {
      opacity: 0,
      xPercent: position === 'left' ? -100 : position === 'right' ? 100 : 0,
      yPercent: position === 'top' ? -100 : 0,
    });
    
    gsap.set(itemsRef.current, { opacity: 0, y: 40 });
    if (socialsRef.current.length) {
      gsap.set(socialsRef.current, { opacity: 0, y: 20 });
    }

    const tl = gsap.timeline({
      paused: true,
      defaults: { ease: 'power4.inOut' },
      onReverseComplete: () => {
        setShouldRender(false);
      }
    });
    tlRef.current = tl;

    // Prelayers
    if (prelayersRef.current.length > 0) {
      tl.to(prelayersRef.current, {
        opacity: 1,
        xPercent: 0,
        yPercent: 0,
        duration: 0.8,
        stagger: 0.1,
      }, 0);
    }

    // Main panel
    tl.to(panelRef.current, {
      opacity: 1,
      xPercent: 0,
      yPercent: 0,
      duration: 0.8,
    }, 0.1);

    // Nav items
    tl.to(itemsRef.current, {
      opacity: 1,
      y: 0,
      duration: 0.6,
      stagger: 0.05,
      ease: 'back.out(1.5)',
    }, 0.3);

    // Socials
    if (socialsRef.current.length) {
      tl.to(socialsRef.current, {
        opacity: 1,
        y: 0,
        duration: 0.6,
        stagger: 0.05,
      }, 0.5);
    }

    if (isOpen) {
      tl.play();
    }

    return () => {
      tl.kill();
      tlRef.current = null;
    };
  }, [position, shouldRender]);

  useEffect(() => {
    if (tlRef.current) {
      if (isOpen) {
        tlRef.current.play();
      } else {
        tlRef.current.reverse();
      }
    }
  }, [isOpen]);

  if (!shouldRender) return null;

  return (
    <div
      ref={wrapperRef}
      className="staggered-menu-wrapper fixed-wrapper"
      data-open={isOpen ? 'true' : 'false'}
      data-position={position}
      style={{ fontFamily }}
    >
      <div className="sm-prelayers">
        <div ref={(el) => { prelayersRef.current[0] = el; }} className="sm-prelayer" style={{ background: '#000000', opacity: 0.5 }} />
        <div ref={(el) => { prelayersRef.current[1] = el; }} className="sm-prelayer" style={{ background: '#333333', opacity: 0.8 }} />
      </div>

      <div ref={panelRef} className="staggered-menu-panel bg-white">
        <button
          onClick={onClose}
          className="absolute top-8 right-8 w-10 h-10 rounded-full bg-black flex items-center justify-center text-white hover:scale-110 transition-transform"
          aria-label="Close menu"
        >
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M1 1L13 13M1 13L13 1" stroke="currentColor" strokeWidth="2" strokeLinecap="round"/>
          </svg>
        </button>

        <div className="sm-panel-inner">
          <ul className="sm-panel-list" data-numbering={displayItemNumbering ? 'true' : undefined}>
            {items.map((item, i) => (
              <li key={i}>
                <Link
                  href={item.link}
                  ref={(el) => { itemsRef.current[i] = el; }}
                  className="sm-panel-item"
                  aria-label={item.ariaLabel}
                  onClick={onClose}
                >
                  <span className="sm-panel-itemLabel">{item.label}</span>
                </Link>
              </li>
            ))}
          </ul>

          {displaySocials && socialItems.length > 0 && (
            <div className="sm-socials">
              <p className="sm-socials-title text-black">Socials</p>
              <ul className="sm-socials-list">
                {socialItems.map((item, i) => (
                  <li key={i}>
                    <a
                      href={item.link}
                      ref={(el) => { socialsRef.current[i] = el; }}
                      className="sm-socials-link"
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      {item.label}
                    </a>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
