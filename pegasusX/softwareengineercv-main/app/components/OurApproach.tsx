import React from 'react';
import { Star, Eye, Zap } from 'lucide-react';
import Digit369 from './Digit369';

export default function OurApproach() {
  return (
    <section className="w-full flex flex-col lg:flex-row min-h-0 lg:min-h-[640px] xl:min-h-[800px] border-t border-white/10 overflow-hidden">
      <div className="flex-1 relative bg-black min-h-[240px] sm:min-h-[300px] md:min-h-[360px] lg:min-h-full">
        <div className="absolute inset-0">
          <Digit369 color="#e8e4e3" backgroundColor="#000000" />
        </div>
      </div>

      <div className="flex-1 flex flex-col bg-[#e6e6e6] text-black relative">
        <div className="absolute inset-0 bg-[url('https://framerusercontent.com/images/6mcf62RlDfRfU61Yg5vb2pefpi4.png')] opacity-[0.04] mix-blend-multiply pointer-events-none" />

        <div className="relative z-10 flex flex-col h-full w-full max-w-[800px]">
          <div className="p-6 sm:p-8 md:p-12 lg:p-16 border-b border-black/10">
            <div className="flex items-center gap-4 mb-6 sm:mb-8">
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 93.865 16.562" className="h-3 sm:h-4 w-auto fill-black">
                <path d="M 7.998 4 L 80.998 4 L 80.998 13 L 7.998 13 Z" fill="transparent" />
                <path d="M 56.561 14.72 L 40.21 0 L 38.553 1.84 L 54.903 16.561 Z M 64.021 14.72 L 47.672 0 L 46.014 1.84 L 62.364 16.561 Z M 71.482 14.72 L 55.133 0 L 53.475 1.84 L 69.825 16.561 Z M 78.943 14.72 L 62.594 0 L 60.936 1.84 L 77.286 16.561 Z M 86.404 14.72 L 70.055 0 L 68.397 1.84 L 84.747 16.561 Z M 93.865 14.72 L 77.516 0 L 75.858 1.84 L 92.208 16.561 Z M 48.85 14.72 L 32.5 0 L 30.842 1.84 L 47.192 16.561 Z M 41.138 14.72 L 24.79 0 L 23.131 1.84 L 39.481 16.561 Z M 33.428 14.72 L 17.078 0 L 15.421 1.84 L 31.771 16.561 Z M 25.717 14.72 L 9.367 0 L 7.711 1.84 L 24.058 16.562 Z M 18.006 14.72 L 1.656 0 L 0 1.84 L 16.348 16.562 Z" />
              </svg>
              <span className="text-xs tracking-widest font-mono text-black/60 uppercase">OUR APPROACH</span>
            </div>

            <h2 className="text-3xl sm:text-4xl md:text-5xl lg:text-6xl font-medium tracking-tight mb-5 sm:mb-8">
              Built for the long term
            </h2>

            <p className="text-base sm:text-lg text-black/70 max-w-xl leading-relaxed">
              We don't just ship code; we architect neural ecosystems. Our approach combines rigorous testing with rapid deployment cycles.
            </p>
          </div>

          <div className="flex-1 grid grid-cols-1 md:grid-cols-2">
            <div className="p-6 sm:p-8 md:p-10 border-b md:border-r border-black/10 flex flex-col gap-4 sm:gap-6 group transition-colors duration-300 hover:bg-black/[0.04]">
              <div className="w-10 h-10 sm:w-12 sm:h-12 flex items-center justify-center md:group-hover:scale-110 transition-transform duration-300">
                <Star strokeWidth={1.5} className="w-8 h-8 sm:w-10 sm:h-10 text-black" />
              </div>
              <div>
                <h3 className="text-lg sm:text-xl font-medium mb-2 sm:mb-3 font-mono tracking-tight">Prime Logic</h3>
                <p className="text-black/60 text-sm leading-relaxed transition-colors duration-300 group-hover:text-black/85">
                  We prioritize high-fidelity model alignment to ensure your agents deliver consistent results.
                </p>
              </div>
            </div>

            <div className="p-6 sm:p-8 md:p-10 border-b border-black/10 flex flex-col gap-4 sm:gap-6 group transition-colors duration-300 hover:bg-black/[0.04]">
              <div className="w-10 h-10 sm:w-12 sm:h-12 flex items-center justify-center md:group-hover:scale-110 transition-transform duration-300">
                <Eye strokeWidth={1.5} className="w-8 h-8 sm:w-10 sm:h-10 text-black" />
              </div>
              <div>
                <h3 className="text-lg sm:text-xl font-medium mb-2 sm:mb-3 font-mono tracking-tight">Total Clarity</h3>
                <p className="text-black/60 text-sm leading-relaxed transition-colors duration-300 group-hover:text-black/85">
                  Gain full observability into how your data is processed, indexed, and retrieved by your AI.
                </p>
              </div>
            </div>

            <div className="p-6 sm:p-8 md:p-10 md:border-r border-black/10 flex flex-col gap-4 sm:gap-6 group transition-colors duration-300 hover:bg-black/[0.04]">
              <div className="w-10 h-10 sm:w-12 sm:h-12 flex items-center justify-center md:group-hover:scale-110 transition-transform duration-300">
                <Zap strokeWidth={1.5} className="w-8 h-8 sm:w-10 sm:h-10 text-black" />
              </div>
              <div>
                <h3 className="text-lg sm:text-xl font-medium mb-2 sm:mb-3 font-mono tracking-tight">Fast Cycles</h3>
                <p className="text-black/60 text-sm leading-relaxed transition-colors duration-300 group-hover:text-black/85">
                  Transition from prototype to production in weeks, not months, with our pre-built frameworks.
                </p>
              </div>
            </div>

            <div className="p-10 hidden md:block" />
          </div>
        </div>
      </div>
    </section>
  );
}
