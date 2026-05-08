'use client';

import { useRef, type ReactNode } from 'react';
import { motion, useInView } from 'framer-motion';

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
  const isInView = useInView(ref, { once: true, amount: 0.2 });

  return (
    <motion.div
      ref={ref}
      initial={{ opacity: 0, y: 20, scale: 0.97 }}
      animate={isInView ? { opacity: 1, y: 0, scale: 1 } : {}}
      transition={{ 
        duration: 0.5, 
        delay: delay / 1000, 
        ease: [0.21, 0.47, 0.32, 0.98] 
      }}
      className={`bento-card bento-span-${span} ${rowSpan ? 'bento-row-2' : ''} hover-lift ${className}`}
    >
      {children}
    </motion.div>
  );
}
