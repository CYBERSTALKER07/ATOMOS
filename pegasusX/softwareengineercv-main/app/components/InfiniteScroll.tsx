'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';

interface InfiniteScrollProps {
  items: Array<{ content: React.ReactNode }>;
  isTilted?: boolean;
  tiltDirection?: 'left' | 'right';
  autoplay?: boolean;
  autoplaySpeed?: number;
  autoplayDirection?: 'up' | 'down';
  pauseOnHover?: boolean;
  width?: string;
  maxHeight?: string;
  itemMinHeight?: number;
  negativeMargin?: string;
}

export default function InfiniteScroll({
  items,
  isTilted = false,
  tiltDirection = 'left',
  autoplay = true,
  autoplaySpeed = 0.5,
  autoplayDirection = 'down',
  pauseOnHover = true,
  width = '100%',
  maxHeight = '600px',
  itemMinHeight = 100,
  negativeMargin = '-1rem'
}: InfiniteScrollProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const scrollRef = useRef<HTMLDivElement>(null);
  const animationRef = useRef<gsap.core.Tween | null>(null);

  useEffect(() => {
    if (!containerRef.current || !scrollRef.current || !autoplay) return;

    const scrollElement = scrollRef.current;
    const itemHeight = itemMinHeight;
    const totalHeight = items.length * itemHeight;

    // Clone items for seamless loop
    const scrollContent = scrollElement.children[0] as HTMLElement;
    if (scrollContent) {
      // Set initial position based on direction
      if (autoplayDirection === 'down') {
        gsap.set(scrollContent, { y: 0 });
      } else {
        gsap.set(scrollContent, { y: -totalHeight });
      }

      // Create infinite scroll animation
      const duration = totalHeight / (autoplaySpeed * 100);
      
      if (autoplayDirection === 'down') {
        animationRef.current = gsap.to(scrollContent, {
          y: -totalHeight,
          duration: duration,
          ease: 'none',
          repeat: -1,
          modifiers: {
            y: (y) => {
              const yValue = parseFloat(y);
              return `${yValue % -totalHeight}px`;
            }
          }
        });
      } else {
        animationRef.current = gsap.to(scrollContent, {
          y: 0,
          duration: duration,
          ease: 'none',
          repeat: -1,
          modifiers: {
            y: (y) => {
              const yValue = parseFloat(y);
              return `${((yValue % totalHeight) + totalHeight) % totalHeight - totalHeight}px`;
            }
          }
        });
      }
    }

    // Pause on hover
    if (pauseOnHover && containerRef.current) {
      const container = containerRef.current;
      
      const handleMouseEnter = () => {
        if (animationRef.current) {
          animationRef.current.pause();
        }
      };
      
      const handleMouseLeave = () => {
        if (animationRef.current) {
          animationRef.current.play();
        }
      };

      container.addEventListener('mouseenter', handleMouseEnter);
      container.addEventListener('mouseleave', handleMouseLeave);

      return () => {
        container.removeEventListener('mouseenter', handleMouseEnter);
        container.removeEventListener('mouseleave', handleMouseLeave);
        if (animationRef.current) {
          animationRef.current.kill();
        }
      };
    }

    return () => {
      if (animationRef.current) {
        animationRef.current.kill();
      }
    };
  }, [items, autoplay, autoplaySpeed, autoplayDirection, pauseOnHover, itemMinHeight]);

  // Calculate rotation class
  const getTiltClass = () => {
    if (!isTilted) return '';
    return tiltDirection === 'left' ? '-rotate-[15deg]' : 'rotate-[15deg]';
  };

  // Duplicate items for seamless loop
  const duplicatedItems = [...items, ...items];

  return (
    <div
      ref={containerRef}
      className={`relative overflow-hidden ${getTiltClass()}`}
      style={{
        '--scroll-width': width,
        '--scroll-max-height': maxHeight,
        width: 'var(--scroll-width)',
        maxHeight: 'var(--scroll-max-height)',
      } as React.CSSProperties}
    >
      <div ref={scrollRef} className="relative">
        <div 
          className="flex flex-col"
          style={{
            '--negative-margin': negativeMargin,
            marginTop: 'var(--negative-margin)',
          } as React.CSSProperties}
        >
          {duplicatedItems.map((item, index) => (
            <div
              key={index}
              className="flex items-center justify-center mb-[30px]"
              style={{
                '--item-min-height': `${itemMinHeight}px`,
                minHeight: 'var(--item-min-height)',
              } as React.CSSProperties}
            >
              {item.content}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
