'use client';

import React, { useEffect, useState, useRef } from 'react';
import { usePathname } from 'next/navigation';

export default function NavigationProgressBar() {
  const pathname = usePathname();
  const [progress, setProgress] = useState(0);
  const [visible, setVisible] = useState(false);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Complete progress on pathname change
  useEffect(() => {
    if (visible) {
      setProgress(100);
      const timer = setTimeout(() => {
        setVisible(false);
        setProgress(0);
      }, 200);
      return () => clearTimeout(timer);
    }
  }, [pathname, visible]);

  // Listen to internal link clicks to give instant feedback
  useEffect(() => {
    const handleLinkClick = (e: MouseEvent) => {
      const target = (e.target as Element)?.closest?.('a');
      if (!target) return;

      const href = target.getAttribute('href');
      const targetAttr = target.getAttribute('target');

      // Only handle internal relative links without hash or external target
      if (
        href &&
        href.startsWith('/') &&
        !href.startsWith('//') &&
        targetAttr !== '_blank' &&
        !href.startsWith('/#') &&
        href !== pathname
      ) {
        // Start progress bar immediately
        setVisible(true);
        setProgress(25);

        if (timerRef.current) clearInterval(timerRef.current);
        timerRef.current = setInterval(() => {
          setProgress((prev) => {
            if (prev >= 90) {
              if (timerRef.current) clearInterval(timerRef.current);
              return 90;
            }
            return prev + Math.random() * 15;
          });
        }, 120);
      }
    };

    window.addEventListener('click', handleLinkClick, { capture: true });
    window.addEventListener('popstate', () => {
      setVisible(true);
      setProgress(40);
    });

    return () => {
      window.removeEventListener('click', handleLinkClick, { capture: true });
      if (timerRef.current) clearInterval(timerRef.current);
    };
  }, [pathname]);

  if (!visible && progress === 0) return null;

  return (
    <div
      className="fixed top-0 left-0 right-0 z-[2147483646] pointer-events-none h-[2.5px] bg-transparent"
      aria-hidden="true"
    >
      <div
        className="h-full bg-[#10B981] transition-all duration-200 ease-out shadow-[0_0_10px_#10B981,0_0_5px_#10B981]"
        style={{
          width: `${progress}%`,
          opacity: visible ? 1 : 0,
          transitionProperty: 'width, opacity',
        }}
      />
    </div>
  );
}
