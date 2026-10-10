'use client';

import * as React from 'react';
import { motion } from 'framer-motion';

export interface AnimatedIconProps {
  icon?: React.ComponentType<any>;
  children?: React.ReactNode;
  size?: number | string;
  className?: string;
  animateOnHover?: boolean;
  [key: string]: any;
}

/**
 * Universal hover-animated icon wrapper.
 * Provides micro-interaction bounce, scale, and subtle rotation on hover.
 */
export default function AnimatedIcon({
  icon: IconComponent,
  children,
  size = 20,
  className = '',
  animateOnHover = true,
  ...props
}: AnimatedIconProps) {
  return (
    <motion.span
      className={`inline-flex items-center justify-center ${className}`}
      whileHover={animateOnHover ? { scale: 1.14, rotate: 4 } : undefined}
      whileTap={{ scale: 0.95 }}
      transition={{ type: 'spring', stiffness: 420, damping: 18 }}
      {...props}
    >
      {IconComponent ? <IconComponent size={size} /> : children}
    </motion.span>
  );
}
