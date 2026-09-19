'use client';

import React, { ReactNode } from 'react';
import Link from 'next/link';

export type SubpageHeroProps = {
 categoryLabel?: string;
 categoryHref?: string;
 badge?: string;
 badgeIcon?: ReactNode;
 title: string;
 summary: string;
 primaryCta?: {
 label: string;
 href: string;
 };
 secondaryCta?: {
 label: string;
 href: string;
 };
 widget?: any;
 nodes?: any;
 breadcrumb?: any;
};

export default function SubpageHero({
 categoryLabel,
 categoryHref,
 badge,
 title,
 summary,
 primaryCta,
 secondaryCta,
}: SubpageHeroProps) {
 const eyebrowBadge = badge || (categoryLabel ? `PEGASUS // ${categoryLabel.toUpperCase()}` : 'PEGASUS OS');
 
 const primaryButton = primaryCta || {
 label: 'Request Demo',
 href: '/join',
 };

 const secondaryButton = secondaryCta || {
 label: 'Explore Stack',
 href: categoryHref || '/platform',
 };

 return (
 <section className="relative w-full bg-[#000000] text-white pt-40 pb-20 px-4 sm:px-6 lg:px-8 border-b border-white/10 select-none overflow-hidden">
 <div className="w-full max-w-[1560px] mx-auto relative z-10 flex flex-col justify-end min-h-[400px]">
 {/* Eyebrow */}
 <div className="mb-6 flex items-center">
 <div className="h-1.5 w-1.5 bg-white mr-3"></div>
 <span className="font-mono text-xs uppercase tracking-widest text-zinc-400">
 {eyebrowBadge}
 </span>
 </div>

 {/* Title */}
 <h1 className="text-4xl sm:text-5xl md:text-6xl lg:text-[5rem] font-medium tracking-tight text-white max-w-5xl leading-[1.05] mb-8">
 {title}
 </h1>

 {/* Summary */}
 <p className="text-lg sm:text-xl font-light leading-relaxed max-w-2xl text-zinc-400 mb-12">
 {summary}
 </p>

 {/* CTA Buttons - Using the global sweep classes */}
 <div className="flex flex-wrap items-center gap-4">
 <Link
 href={primaryButton.href}
 className="editorial-btn"
 >
 <span>{primaryButton.label}</span>
 <span className="text-lg leading-none mt-[-2px]">›</span>
 </Link>
 <Link
 href={secondaryButton.href}
 className="editorial-btn editorial-btn--inverted"
 >
 <span>{secondaryButton.label}</span>
 <span className="text-lg leading-none mt-[-2px]">›</span>
 </Link>
 </div>
 </div>
 </section>
 );
}
