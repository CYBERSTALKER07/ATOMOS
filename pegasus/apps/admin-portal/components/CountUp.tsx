'use client';

import { useEffect, useRef } from 'react';
import { useMotionValue, useSpring, useTransform, motion } from 'framer-motion';

export default function CountUp({
  value,
  duration = 2,
  delay = 0,
  className = '',
}: {
  value: string;
  duration?: number;
  delay?: number;
  className?: string;
}) {
  const numericValue = parseFloat(value.replace(/[^0-9.]/g, '')) || 0;
  const prefix = value.match(/^[^0-9]*/)?.[0] || '';
  const suffix = value.match(/[0-9.]*(.*)$/)?.[1] || '';
  
  const count = useMotionValue(0);
  const rounded = useTransform(count, (latest) => {
    // If it was an integer, keep it integer. If it had decimals, keep 2 decimals.
    if (value.includes('.')) {
      return latest.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
    }
    return Math.floor(latest).toLocaleString();
  });
  
  const springValue = useSpring(count, {
    stiffness: 100,
    damping: 30,
    restDelta: 0.001
  });

  useEffect(() => {
    const timer = setTimeout(() => {
      count.set(numericValue);
    }, delay * 1000);
    return () => clearTimeout(timer);
  }, [numericValue, count, delay]);

  return (
    <span className={className}>
      {prefix}
      <motion.span>{rounded}</motion.span>
      {suffix}
    </span>
  );
}
