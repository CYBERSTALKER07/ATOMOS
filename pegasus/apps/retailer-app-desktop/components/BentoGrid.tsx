'use client';

import { useRef, useEffect, useState, type ReactNode } from 'react';

interface BentoGridProps {
  children: ReactNode;
  className?: string;
}

export function BentoGrid({ children, className = '' }: BentoGridProps) {
  return <div className={`bento-grid ${className}`}>{children}</div>;
}

interface BentoCardProps {
  children: ReactNode;
  span?: 1 | 2 | 3 | 4;
  rowSpan?: boolean;
  className?: string;
  delay?: number;
}

export function BentoCard({ children, span = 1, rowSpan = false, className = '', delay = 0 }: BentoCardProps) {
  const ref = useRef<HTMLDivElement>(null);
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const observer = new IntersectionObserver(
      ([entry]) => { if (entry.isIntersecting) { setVisible(true); observer.disconnect(); } },
      { threshold: 0.1 }
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  return (
    <div
      ref={ref}
      className={`bento-card bento-span-${span} ${rowSpan ? 'bento-row-2' : ''} ${className}`}
      style={{
        opacity: visible ? 1 : 0,
        transform: visible ? 'translateY(0) scale(1)' : 'translateY(20px) scale(0.97)',
        transition: `opacity 0.5s cubic-bezier(0.05, 0.7, 0.1, 1) ${delay}ms, transform 0.5s cubic-bezier(0.05, 0.7, 0.1, 1) ${delay}ms`,
      }}
    >
      {children}
    </div>
  );
}
