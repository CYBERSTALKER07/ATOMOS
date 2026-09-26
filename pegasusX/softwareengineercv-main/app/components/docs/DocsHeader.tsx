'use client';

import React from 'react';
import Link from 'next/link';
import { Globe } from 'lucide-react';
import { Search, ChevronRight, Terminal, ArrowLeft, Menu } from '@/components/icons';
import { useLanguage } from '@/app/context/LanguageContext';

type DocsHeaderProps = {
 onOpenSearch: () => void;
 onToggleMobileSidebar: () => void;
 activeCategory?: string;
 activeTitle?: string;
};

export default function DocsHeader({
 onOpenSearch,
 onToggleMobileSidebar,
 activeCategory,
 activeTitle,
}: DocsHeaderProps) {
 const { language, setLanguage } = useLanguage();

 return (
 <header className="sticky top-0 z-40 w-full border-b border-white/10 bg-black/90 backdrop-blur-md">
 <div className="max-w-[1520px] mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
 {/* Left Brand and Breadcrumb */}
 <div className="flex items-center space-x-3 sm:space-x-4">
 <button
 onClick={onToggleMobileSidebar}
 className="lg:hidden p-2 -ml-2 rounded-none text-zinc-400 hover:text-white hover:bg-black transition-colors"
 aria-label="Toggle Navigation Sidebar"
 >
 <Menu size={20} />
 </button>

 <Link href="/docs" className="flex items-center space-x-2.5 group">
 <div className="w-8 h-8 rounded-none bg-white text-black flex items-center justify-center border border-white group-hover:scale-105 transition-transform">
 <Terminal size={16} className="text-black" />
 </div>
 <div className="flex items-center space-x-1.5">
 <span className="font-bold text-white text-base tracking-tight group-hover:text-white transition-colors">
 Pegasus
 </span>
 <span className="text-zinc-400 text-base font-normal">Docs</span>
 </div>
 </Link>

 {/* Breadcrumb path if available */}
 {activeCategory && activeTitle && (
 <div className="hidden md:flex items-center space-x-1.5 pl-3 border-l border-white/10 text-xs font-mono text-zinc-400">
 <span className="capitalize">{activeCategory}</span>
 <ChevronRight size={14} className="text-zinc-600" />
 <span className="text-white truncate max-w-[200px]">{activeTitle}</span>
 </div>
 )}
 </div>

 {/* Center / Right Controls */}
 <div className="flex items-center space-x-3 sm:space-x-4">
 {/* Quick Category Nav Links (Desktop) */}
 <nav className="hidden xl:flex items-center space-x-5 text-xs font-medium text-zinc-400">
 <Link href="/docs/guides/introduction" className="hover:text-white transition-colors">
 Guides
 </Link>
 <Link href="/docs/roles/supplier-control-plane" className="hover:text-white transition-colors">
 Roles
 </Link>
 <Link href="/docs/protocols/cvrp-route-optimizer" className="hover:text-white transition-colors">
 Protocols
 </Link>
 <Link href="/docs/playbooks/stockout-rejection" className="hover:text-white transition-colors">
 Playbooks
 </Link>
 <Link href="/docs/api/rest-api-reference" className="hover:text-white transition-colors">
 API
 </Link>
 </nav>

 {/* Search Trigger Bar (styled like Primer search) */}
 <button
 onClick={onOpenSearch}
 className="flex items-center justify-between w-40 sm:w-64 h-9 px-3 rounded-none bg-black border border-white/15 text-xs text-zinc-400 hover:border-white/40 hover:text-white transition-all group cursor-pointer"
 >
 <div className="flex items-center space-x-2 truncate">
 <Search size={14} className="text-zinc-500 group-hover:text-white" />
 <span className="truncate">Search Docs...</span>
 </div>
 <kbd className="hidden sm:inline-flex items-center px-1.5 py-0.5 text-[10px] font-mono text-zinc-400 bg-black border border-white/10 rounded-none">
 ⌘K
 </kbd>
 </button>

 {/* Language Switcher */}
 <div className="flex items-center rounded-none bg-black border border-white/15 p-0.5 text-xs font-mono">
 <button
 onClick={() => setLanguage('en')}
 className={`px-2 py-1 rounded-none transition-colors ${
 language === 'en' ? 'bg-white text-black font-semibold' : 'text-zinc-400 hover:text-white'
 }`}
 >
 EN
 </button>
 <button
 onClick={() => setLanguage('ru')}
 className={`px-2 py-1 rounded-none transition-colors ${
 language === 'ru' ? 'bg-white text-black font-semibold' : 'text-zinc-400 hover:text-white'
 }`}
 >
 RU
 </button>
 </div>

 {/* Return to Platform Link */}
 <Link
 href="/platform"
 className="hidden sm:inline-flex items-center space-x-1.5 text-xs font-medium text-zinc-400 hover:text-white transition-colors pl-2 group"
 >
 <ArrowLeft size={14} />
 <span>Platform</span>
 </Link>
 </div>
 </div>
 </header>
 );
}
