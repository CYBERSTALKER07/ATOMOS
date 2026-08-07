import React from 'react';
import Image from 'next/image';

export function PegasusTestimonialsSection() {
  return (
    <section className="w-full bg-black text-white py-16 px-4 flex justify-center items-center font-sans antialiased">
      <div className="max-w-6xl w-full mx-auto flex flex-col items-center gap-16">
        
        {/* Top Header & Logo Cloud */}
        <div className="flex flex-col items-center gap-7 w-full text-center">
          <p className="text-sm font-medium text-zinc-500 tracking-tight">
            Trusted by 200+ companies to make smarter decisions
          </p>

          <div className="flex flex-wrap justify-center items-center gap-x-12 gap-y-6 opacity-85 w-full">
            <div className="flex items-center gap-2 text-xl font-bold text-zinc-200 tracking-tight cursor-pointer hover:opacity-100 hover:-translate-y-0.5 transition-all">
              <svg className="h-5.5 fill-current" viewBox="0 0 24 24"><path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/></svg>
              Tiger Data
            </div>
            <div className="flex items-center gap-2 text-xl font-bold text-zinc-200 tracking-tight cursor-pointer hover:opacity-100 hover:-translate-y-0.5 transition-all">
              <svg className="h-5.5 fill-current" viewBox="0 0 24 24"><circle cx="12" cy="12" r="9" stroke="currentColor" strokeWidth="2" fill="none"/><path d="M12 7v10M7 12h10"/></svg>
              Gumloop
            </div>
            <div className="flex items-center gap-2 text-xl font-bold text-zinc-200 tracking-tight cursor-pointer hover:opacity-100 hover:-translate-y-0.5 transition-all">
              <svg className="h-5.5 fill-current" viewBox="0 0 24 24"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/></svg>
              Composio
            </div>
            <div className="flex items-center gap-2 text-xl font-bold text-zinc-200 tracking-tight cursor-pointer hover:opacity-100 hover:-translate-y-0.5 transition-all">
              <svg className="h-5.5 fill-current" viewBox="0 0 24 24"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8z"/></svg>
              Fullscript
            </div>
            <div className="flex items-center gap-2 text-xl font-bold text-zinc-200 tracking-tight cursor-pointer hover:opacity-100 hover:-translate-y-0.5 transition-all">
              <svg className="h-5.5 fill-current" viewBox="0 0 24 24"><path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"/></svg>
              ShiftRx
            </div>
            <div className="flex items-center gap-2 text-xl font-black text-zinc-200 tracking-widest cursor-pointer hover:opacity-100 hover:-translate-y-0.5 transition-all">
              DRATA
            </div>
          </div>
        </div>

        {/* 2-Column Testimonials Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-12 w-full">
          
          {/* Card 1: CTO of Pegasus */}
          <div className="flex flex-col gap-6 p-4 rounded-xl hover:bg-white/[0.02] transition-colors">
            <div className="flex gap-6 items-start">
              <div className="w-28 h-28 flex-shrink-0 rounded-lg overflow-hidden bg-zinc-950 relative border border-zinc-800/50">
                <Image 
                  src="/Gemini_Generated_Image_e86uare86uare86u.png" 
                  alt="CTO of Pegasus"
                  width={112}
                  height={112}
                  className="w-full h-full object-cover filter contrast-105"
                />
              </div>
              <blockquote className="font-serif text-xl md:text-2xl leading-snug text-zinc-100 tracking-tight">
                “We evaluated Omni and other BI tools, but the speed to insight with Pegasus is unmatched.”
              </blockquote>
            </div>
            
            <div className="flex flex-col gap-4 pl-0 sm:pl-[136px]">
              <div className="flex items-center gap-3">
                <div className="flex items-center gap-1.5 text-xs font-bold tracking-widest uppercase text-white">
                  <svg className="h-4.5 fill-current" viewBox="0 0 24 24"><path d="M12 2L2 19h20L12 2zm0 4l6.5 11h-13L12 6z"/></svg>
                  PEGASUS
                </div>
                <div className="flex flex-col">
                  <span className="text-sm font-semibold text-zinc-100 leading-tight">Greg Demoge</span>
                  <span className="text-xs text-zinc-500 leading-tight">CTO · Pegasus</span>
                </div>
              </div>
              
              <a href="#" className="inline-flex items-center gap-1.5 text-xs font-medium text-zinc-400 hover:text-white transition-colors group">
                Read case study <span className="group-hover:translate-x-1 transition-transform">→</span>
              </a>
            </div>
          </div>

          {/* Card 2: Taxfyle Lead */}
          <div className="flex flex-col gap-6 p-4 rounded-xl hover:bg-white/[0.02] transition-colors">
            <div className="flex gap-6 items-start">
              <div className="w-28 h-28 flex-shrink-0 rounded-lg overflow-hidden bg-zinc-950 relative border border-zinc-800/50">
                <Image 
                  src="/partner_dithered_portrait_1786077142245.png" 
                  alt="Claudio Godoy"
                  width={112}
                  height={112}
                  className="w-full h-full object-cover filter contrast-105"
                />
              </div>
              <blockquote className="font-serif text-xl md:text-2xl leading-snug text-zinc-100 tracking-tight">
                “For a security-conscious company like ours, Basedash instantly clicked. Reports that took weeks are ready in hours.”
              </blockquote>
            </div>
            
            <div className="flex flex-col gap-4 pl-0 sm:pl-[136px]">
              <div className="flex items-center gap-3">
                <div className="flex items-center gap-1.5 text-sm font-extrabold lowercase text-white">
                  taxfyle
                  <svg className="h-3.5 fill-current ml-0.5" viewBox="0 0 24 24"><path d="M5 3l14 9-14 9V3z"/></svg>
                </div>
                <div className="flex flex-col">
                  <span className="text-sm font-semibold text-zinc-100 leading-tight">Claudio Godoy</span>
                  <span className="text-xs text-zinc-500 leading-tight">AI Agents Lead · Taxfyle</span>
                </div>
              </div>
              
              <a href="#" className="inline-flex items-center gap-1.5 text-xs font-medium text-zinc-400 hover:text-white transition-colors group">
                Read case study <span className="group-hover:translate-x-1 transition-transform">→</span>
              </a>
            </div>
          </div>

        </div>

      </div>
    </section>
  );
}
