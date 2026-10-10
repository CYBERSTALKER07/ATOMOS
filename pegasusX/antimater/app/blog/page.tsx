import React from 'react';
import Link from 'next/link';
import { Rss } from 'lucide-react';

export const metadata = {
  title: 'Blog | Pegasus',
  description: 'Stay up to date with the latest from the Pegasus team.',
};

export default function BlogPage() {
  return (
    <div className="min-h-screen bg-white text-black pt-32 pb-24 font-sans">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        
        {/* Featured Section */}
        <div className="flex flex-col lg:flex-row gap-16 items-center mb-16">
          <div className="flex-1">
            <h1 className="text-[3.5rem] lg:text-[4rem] leading-none tracking-tight text-black mb-6 font-normal">
              Featured
            </h1>
            <h2 className="text-[2rem] lg:text-[2.5rem] leading-[1.1] tracking-tight text-black mb-8 font-normal max-w-lg">
              Pegasus 3.8 Flash in Global Logistics
            </h2>
            <div className="flex items-center gap-3 text-[15px] text-black/70 mb-8 font-medium">
              <span>1 Sept 2026</span>
              <span>·</span>
              <span>1 min read</span>
              <span className="px-3 py-0.5 bg-black/[0.06] rounded-full text-black/80">Model</span>
            </div>
            <Link href="/blog/pegasus-3-8-flash" className="inline-flex items-center justify-center px-6 py-2 border border-black/20 rounded-full hover:bg-black/[0.04] transition-colors font-medium text-[15px]">
              Read blog
            </Link>
          </div>
          <div className="flex-1 w-full">
            <div className="aspect-[16/10] bg-[#000000] rounded-3xl overflow-hidden flex items-center justify-center relative shadow-xl">
              {/* Abstract shape representing the Pegasus flash announcement */}
              <div className="absolute w-[200px] h-[300px] bg-gradient-to-t from-[#1a73e8] via-purple-400 to-[#ff7a00] rounded-[100%] blur-[40px] opacity-70 mix-blend-screen mix-blend-lighten transform rotate-12 scale-150" />
            </div>
          </div>
        </div>

        {/* Navigation Tabs */}
        <div className="border-b border-black/10 flex flex-wrap items-center justify-between mt-24 gap-y-6">
          <div className="flex items-center gap-8 text-[15px] font-medium overflow-x-auto w-full md:w-auto pb-[-2px]">
            <button className="py-4 border-b-[3px] border-[#1a73e8] text-black whitespace-nowrap">
              All
            </button>
            <button className="py-4 border-b-[3px] border-transparent text-black/60 hover:text-black transition-colors whitespace-nowrap">
              Product
            </button>
            <button className="py-4 border-b-[3px] border-transparent text-black/60 hover:text-black transition-colors whitespace-nowrap">
              Model
            </button>
            <button className="py-4 border-b-[3px] border-transparent text-black/60 hover:text-black transition-colors whitespace-nowrap">
              Enterprise
            </button>
            <button className="py-4 border-b-[3px] border-transparent text-black/60 hover:text-black transition-colors whitespace-nowrap">
              Research
            </button>
          </div>
          <div className="ml-auto">
            <a href="/blog/rss.xml" className="inline-flex items-center gap-2 px-4 py-2 bg-black/[0.06] rounded-full hover:bg-black/[0.1] transition-colors text-sm font-medium text-black/80">
              <Rss className="w-[18px] h-[18px]" />
              RSS Feed
            </a>
          </div>
        </div>
        
        {/* Placeholder for blog posts grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8 mt-12">
          {/* Post 1 */}
          <div className="group cursor-pointer flex flex-col gap-4">
            <div className="aspect-[16/10] bg-[#f2f2f2] rounded-2xl overflow-hidden relative">
               <div className="absolute inset-0 bg-gradient-to-br from-black/5 to-black/10 group-hover:scale-105 transition-transform duration-500" />
            </div>
            <div>
              <div className="flex items-center gap-3 text-sm text-black/60 mb-2">
                <span>15 Aug 2026</span>
                <span className="px-2 py-0.5 bg-black/[0.06] rounded-full text-xs">Enterprise</span>
              </div>
              <h3 className="text-xl font-medium text-black group-hover:text-[#1a73e8] transition-colors line-clamp-2">
                Scaling your supply chain operations with Pegasus
              </h3>
            </div>
          </div>
          {/* Post 2 */}
          <div className="group cursor-pointer flex flex-col gap-4">
            <div className="aspect-[16/10] bg-[#f2f2f2] rounded-2xl overflow-hidden relative">
               <div className="absolute inset-0 bg-gradient-to-tr from-blue-500/10 to-transparent group-hover:scale-105 transition-transform duration-500" />
            </div>
            <div>
              <div className="flex items-center gap-3 text-sm text-black/60 mb-2">
                <span>2 Aug 2026</span>
                <span className="px-2 py-0.5 bg-black/[0.06] rounded-full text-xs">Product</span>
              </div>
              <h3 className="text-xl font-medium text-black group-hover:text-[#1a73e8] transition-colors line-clamp-2">
                New features in the Pegasus Logistics Dashboard
              </h3>
            </div>
          </div>
          {/* Post 3 */}
          <div className="group cursor-pointer flex flex-col gap-4">
            <div className="aspect-[16/10] bg-[#f2f2f2] rounded-2xl overflow-hidden relative">
               <div className="absolute inset-0 bg-gradient-to-b from-black/5 to-black/10 group-hover:scale-105 transition-transform duration-500" />
            </div>
            <div>
              <div className="flex items-center gap-3 text-sm text-black/60 mb-2">
                <span>28 Jul 2026</span>
                <span className="px-2 py-0.5 bg-black/[0.06] rounded-full text-xs">Research</span>
              </div>
              <h3 className="text-xl font-medium text-black group-hover:text-[#1a73e8] transition-colors line-clamp-2">
                The future of automated freight routing
              </h3>
            </div>
          </div>
        </div>

      </div>
    </div>
  );
}
