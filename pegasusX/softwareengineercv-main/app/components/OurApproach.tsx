import React from 'react';
import { Brain, Eye, Zap } from 'lucide-react';

export default function OurApproach() {
  return (
    <section className="w-full bg-black text-white py-24 md:py-32 flex justify-center border-t border-white/10">
      <div className="w-[90%] max-w-[1400px] flex flex-col lg:flex-row gap-16 lg:gap-24">
        
        {/* Left Content */}
        <div className="flex-1 flex flex-col justify-start">
          <div className="flex items-center gap-4 mb-8">
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 93.865 16.562" className="h-4 w-auto fill-white">
              <path d="M 7.998 4 L 80.998 4 L 80.998 13 L 7.998 13 Z" fill="transparent" />
              <path d="M 56.561 14.72 L 40.21 0 L 38.553 1.84 L 54.903 16.561 Z M 64.021 14.72 L 47.672 0 L 46.014 1.84 L 62.364 16.561 Z M 71.482 14.72 L 55.133 0 L 53.475 1.84 L 69.825 16.561 Z M 78.943 14.72 L 62.594 0 L 60.936 1.84 L 77.286 16.561 Z M 86.404 14.72 L 70.055 0 L 68.397 1.84 L 84.747 16.561 Z M 93.865 14.72 L 77.516 0 L 75.858 1.84 L 92.208 16.561 Z M 48.85 14.72 L 32.5 0 L 30.842 1.84 L 47.192 16.561 Z M 41.138 14.72 L 24.79 0 L 23.131 1.84 L 39.481 16.561 Z M 33.428 14.72 L 17.078 0 L 15.421 1.84 L 31.771 16.561 Z M 25.717 14.72 L 9.367 0 L 7.711 1.84 L 24.058 16.562 Z M 18.006 14.72 L 1.656 0 L 0 1.84 L 16.348 16.562 Z" />
            </svg>
            <span className="text-sm tracking-widest font-mono text-white/80 uppercase">OUR APPROACH</span>
          </div>
          
          <h2 className="text-5xl md:text-6xl lg:text-7xl font-medium tracking-tight mb-8">
            Built for the long term
          </h2>
          
          <p className="text-lg md:text-xl text-white/50 max-w-xl leading-relaxed font-mono">
            We don't just ship code; we architect neural ecosystems. Our approach combines rigorous testing with rapid deployment cycles.
          </p>
        </div>

        {/* Right Content - Cards */}
        <div className="flex-1 flex flex-col gap-6">
          
          {/* Card 1 */}
          <div className="group relative bg-[#0a0a0a] border border-white/5 rounded-xl p-8 overflow-hidden hover:border-white/20 transition-all duration-300">
             <div className="absolute inset-0 bg-[url('https://framerusercontent.com/images/6mcf62RlDfRfU61Yg5vb2pefpi4.png')] opacity-[0.03] mix-blend-overlay"></div>
             <div className="absolute inset-0 bg-gradient-to-r from-transparent via-white/5 to-transparent -translate-x-[100%] group-hover:animate-[shimmer_2s_infinite]"></div>
             
             <div className="relative z-10 flex flex-col sm:flex-row items-start sm:items-center gap-6">
                <div className="p-4 bg-white/5 border border-white/10 rounded-lg group-hover:scale-110 transition-transform duration-300">
                   <Brain className="w-8 h-8 text-white/80" />
                </div>
                <div>
                   <h3 className="text-2xl font-medium mb-2">Prime Logic</h3>
                   <p className="text-white/50 leading-relaxed text-sm md:text-base">We prioritize high-fidelity model alignment to ensure your agents deliver consistent results.</p>
                </div>
             </div>
          </div>

          {/* Card 2 */}
          <div className="group relative bg-[#0a0a0a] border border-white/5 rounded-xl p-8 overflow-hidden hover:border-white/20 transition-all duration-300">
             <div className="absolute inset-0 bg-[url('https://framerusercontent.com/images/6mcf62RlDfRfU61Yg5vb2pefpi4.png')] opacity-[0.03] mix-blend-overlay"></div>
             <div className="absolute inset-0 bg-gradient-to-r from-transparent via-white/5 to-transparent -translate-x-[100%] group-hover:animate-[shimmer_2s_infinite]"></div>

             <div className="relative z-10 flex flex-col sm:flex-row items-start sm:items-center gap-6">
                <div className="p-4 bg-white/5 border border-white/10 rounded-lg group-hover:scale-110 transition-transform duration-300">
                   <Eye className="w-8 h-8 text-white/80" />
                </div>
                <div>
                   <h3 className="text-2xl font-medium mb-2">Total Clarity</h3>
                   <p className="text-white/50 leading-relaxed text-sm md:text-base">Gain full observability into how your data is processed, indexed, and retrieved by your AI.</p>
                </div>
             </div>
          </div>

          {/* Card 3 */}
          <div className="group relative bg-[#0a0a0a] border border-white/5 rounded-xl p-8 overflow-hidden hover:border-white/20 transition-all duration-300">
             <div className="absolute inset-0 bg-[url('https://framerusercontent.com/images/6mcf62RlDfRfU61Yg5vb2pefpi4.png')] opacity-[0.03] mix-blend-overlay"></div>
             <div className="absolute inset-0 bg-gradient-to-r from-transparent via-white/5 to-transparent -translate-x-[100%] group-hover:animate-[shimmer_2s_infinite]"></div>

             <div className="relative z-10 flex flex-col sm:flex-row items-start sm:items-center gap-6">
                <div className="p-4 bg-white/5 border border-white/10 rounded-lg group-hover:scale-110 transition-transform duration-300">
                   <Zap className="w-8 h-8 text-white/80" />
                </div>
                <div>
                   <h3 className="text-2xl font-medium mb-2">Fast Cycles</h3>
                   <p className="text-white/50 leading-relaxed font-mono text-xs md:text-sm">
                      Transition from prototype to production in weeks, not months, with our pre-built frameworks.
                   </p>
                </div>
             </div>
          </div>

        </div>
      </div>
    </section>
  );
}
