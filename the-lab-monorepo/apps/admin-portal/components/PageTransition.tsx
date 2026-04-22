'use client';

import { usePathname } from 'next/navigation';
import { useEffect, useRef, useState, type ReactNode } from 'react';

export default function PageTransition({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const [displayChildren, setDisplayChildren] = useState(children);
  const [phase, setPhase] = useState<'enter' | 'exit'>('enter');
  const prevPath = useRef(pathname);

  useEffect(() => {
    if (pathname !== prevPath.current) {
      setPhase('exit');
      const timer = setTimeout(() => {
        setDisplayChildren(children);
        setPhase('enter');
        prevPath.current = pathname;
      }, 60);
      return () => clearTimeout(timer);
    } else {
      setDisplayChildren(children);
    }
  }, [pathname, children]);

  return (
    <div
      className="flex-1 min-w-0 w-full"
      style={{
        opacity: phase === 'exit' ? 0 : 1,
        transform: phase === 'exit' ? 'scale(0.998) translateY(2px)' : 'scale(1) translateY(0)',
        transition: phase === 'enter'
          ? 'opacity 120ms ease-out, transform 120ms ease-out'
          : 'opacity 60ms ease-in, transform 60ms ease-in',
        willChange: 'opacity, transform',
      }}
    >
      {displayChildren}
    </div>
  );
}
