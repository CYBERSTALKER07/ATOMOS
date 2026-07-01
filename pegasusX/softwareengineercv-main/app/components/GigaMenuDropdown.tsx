'use client';

import { useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import { gsap } from 'gsap';
import { 
  Box, 
  Layers, 
  Zap, 
  Settings, 
  Activity, 
  Network, 
  Truck, 
  Navigation, 
  Shield 
} from 'lucide-react';
import type { MegaNavCategory, MegaNavPromo } from '../data/megaNavigation';

function getIconForFlow(flow?: string) {
  switch (flow) {
    case 'controlPlane': return <Layers className="w-6 h-6 text-blue-500" />;
    case 'orderLifecycle': return <Activity className="w-6 h-6 text-blue-500" />;
    case 'mutatingHandler': return <Settings className="w-6 h-6 text-blue-500" />;
    case 'realtimePipeline': return <Zap className="w-6 h-6 text-blue-500" />;
    case 'topologyMap': return <Network className="w-6 h-6 text-blue-500" />;
    case 'dispatchBoard': return <Truck className="w-6 h-6 text-blue-500" />;
    case 'fleetMap': return <Navigation className="w-6 h-6 text-blue-500" />;
    case 'paymentFlow': return <Shield className="w-6 h-6 text-blue-500" />;
    default: return <Box className="w-6 h-6 text-blue-500" />;
  }
}

type GigaMenuDropdownProps = {
  activeCategory: MegaNavCategory | null;
  onMouseEnter: () => void;
  onMouseLeave: () => void;
};

export default function GigaMenuDropdown({
  activeCategory,
  onMouseEnter,
  onMouseLeave,
}: GigaMenuDropdownProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const [isOpen, setIsOpen] = useState(false);
  const [renderedCategory, setRenderedCategory] = useState<MegaNavCategory | null>(null);

  // We use a small delay before closing to allow moving the mouse from the nav to the dropdown
  useEffect(() => {
    if (activeCategory) {
      setRenderedCategory(activeCategory);
      setIsOpen(true);
    } else {
      setIsOpen(false);
    }
  }, [activeCategory]);

  useEffect(() => {
    if (!containerRef.current || !contentRef.current) return;

    if (isOpen) {
      gsap.to(containerRef.current, {
        height: 'auto',
        duration: 0.4,
        ease: 'power3.out',
      });
      gsap.to(contentRef.current, {
        opacity: 1,
        y: 0,
        duration: 0.3,
        delay: 0.1,
        ease: 'power2.out',
      });
    } else {
      gsap.to(contentRef.current, {
        opacity: 0,
        y: -10,
        duration: 0.2,
        ease: 'power2.in',
      });
      gsap.to(containerRef.current, {
        height: 0,
        duration: 0.3,
        delay: 0.1,
        ease: 'power3.inOut',
      });
    }
  }, [isOpen]);

  if (!renderedCategory && !isOpen) return null;

  return (
    <div
      ref={containerRef}
      onMouseEnter={onMouseEnter}
      onMouseLeave={onMouseLeave}
      className="absolute top-full left-0 right-0 z-50 bg-black border-b border-white/10 overflow-hidden"
      style={{ height: 0 }}
    >
      <div 
        ref={contentRef}
        className="max-w-7xl mx-auto px-4 py-12 flex flex-col lg:flex-row gap-12 opacity-0 -translate-y-2"
      >
        {renderedCategory && (
          <>
            {/* Left Column: Grid of Links */}
            <div className="flex-1">
              <h2 className="text-2xl font-light text-white mb-8 tracking-wide">
                {renderedCategory.label}
              </h2>
              
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {renderedCategory.links.map((link, idx) => (
                  <Link
                    key={`${link.label}-${idx}`}
                    href={link.href}
                    className="group flex flex-col justify-between p-5 min-h-[140px] bg-[#111] hover:bg-[#222] transition-colors relative"
                    style={{
                      clipPath: 'polygon(0 0, calc(100% - 24px) 0, 100% 24px, 100% 100%, 0 100%)'
                    }}
                  >
                    <div className="flex justify-between items-start">
                      {getIconForFlow(link.flow)}
                      {link.badge && (
                        <span className="text-[10px] font-bold px-2 py-0.5 bg-white text-black rounded-sm tracking-wider">
                          {link.badge}
                        </span>
                      )}
                    </div>
                    <div className="mt-6">
                      <div className="text-white font-medium text-sm group-hover:text-blue-400 transition-colors">
                        {link.label}
                      </div>
                      {link.description && (
                        <p className="text-xs text-gray-400 line-clamp-2 mt-1">
                          {link.description}
                        </p>
                      )}
                    </div>
                  </Link>
                ))}
              </div>

              {renderedCategory.viewAllHref && (
                <div className="mt-6 pt-6 border-t border-white/10 flex justify-between items-center">
                  <div className="text-white font-bold tracking-widest uppercase">
                    Distributors & Resellers
                  </div>
                  <Link
                    href={renderedCategory.viewAllHref}
                    className="inline-flex items-center justify-between px-6 py-4 bg-[#111] hover:bg-[#222] text-white text-sm font-bold tracking-widest uppercase transition-colors rounded-lg"
                  >
                    {renderedCategory.viewAllLabel || 'Learn More'}
                    <span className="ml-4">▼</span>
                  </Link>
                </div>
              )}
            </div>

            {/* Right Column: Featured Promo */}
            {renderedCategory.promo && (
              <div className="w-full lg:w-[400px] shrink-0">
                <h2 className="text-2xl font-light text-white mb-8 tracking-wide">
                  Customer Stories
                </h2>
                <Link
                  href={renderedCategory.promo.primaryHref}
                  className="relative h-[400px] bg-gradient-to-br from-[#222] to-[#0a0a0a] rounded-lg border border-white/10 p-8 flex flex-col justify-end overflow-hidden group block hover:border-white/30 transition-colors"
                >
                  {/* Subtle background abstract pattern/gradient */}
                  <div className="absolute inset-0 opacity-20 bg-[radial-gradient(circle_at_top_right,_var(--tw-gradient-stops))] from-white/20 via-transparent to-transparent mix-blend-overlay"></div>
                  
                  <div className="relative z-10">
                    <h3 className="text-2xl font-medium text-white mb-4 leading-tight">
                      {renderedCategory.promo.title}
                    </h3>
                    <p className="text-gray-400 mb-8">
                      {renderedCategory.promo.body}
                    </p>
                    <div className="flex gap-4">
                      <span
                        className="px-6 py-3 bg-white text-black font-semibold text-sm hover:bg-gray-200 transition-colors inline-block"
                      >
                        {renderedCategory.promo.primaryLabel}
                      </span>
                      {renderedCategory.promo.secondaryHref && (
                        <span
                          className="px-6 py-3 bg-transparent border border-white/20 text-white font-semibold text-sm hover:bg-white/10 transition-colors inline-block"
                        >
                          {renderedCategory.promo.secondaryLabel}
                        </span>
                      )}
                    </div>
                  </div>
                </Link>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
