'use client';

import { useInView } from '@/app/hooks/useInView';

type LazyWhenInViewProps = {
  children: React.ReactNode;
  /** Unmount when scrolled away — frees WebGL / iframes */
  unmountOnExit?: boolean;
  rootMargin?: string;
  className?: string;
  minHeight?: string;
};

/**
 * Renders children only when near the viewport. Use for FleetScrollShowcase,
 * Lanyard, FlowSlot, and other heavy client-only blocks.
 */
export default function LazyWhenInView({
  children,
  unmountOnExit = true,
  rootMargin = '200px',
  className,
  minHeight = '1px',
}: LazyWhenInViewProps) {
  const { ref, isInView } = useInView<HTMLDivElement>({
    rootMargin,
    exit: unmountOnExit,
  });

  return (
    <div ref={ref} className={className} style={{ minHeight: isInView ? undefined : minHeight }}>
      {isInView ? children : null}
    </div>
  );
}
