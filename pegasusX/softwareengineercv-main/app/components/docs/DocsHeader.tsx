'use client';

import React from 'react';
import Link from 'next/link';
import { Search, ChevronRight, Terminal, Globe, ArrowLeft, Menu } from 'lucide-react';
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
    <header className="sticky top-0 z-40 w-full border-b border-[#1F1F2B] bg-[#09090D]/90 backdrop-blur-md">
      <div className="max-w-[1520px] mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
        {/* Left Brand and Breadcrumb */}
        <div className="flex items-center space-x-3 sm:space-x-4">
          <button
            onClick={onToggleMobileSidebar}
            className="lg:hidden p-2 -ml-2 rounded-lg text-[#8E8EA0] hover:text-white hover:bg-[#161622] transition-colors"
            aria-label="Toggle Navigation Sidebar"
          >
            <Menu className="w-5 h-5" />
          </button>

          <Link href="/docs" className="flex items-center space-x-2.5 group">
            <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-[#1E3A8A] to-[#2563EB] flex items-center justify-center shadow-[0_0_15px_rgba(37,99,235,0.4)] border border-[#3B82F6]/40 group-hover:scale-105 transition-transform">
              <Terminal className="w-4 h-4 text-white" />
            </div>
            <div className="flex items-center space-x-1.5">
              <span className="font-bold text-white text-base tracking-tight group-hover:text-[#60A5FA] transition-colors">
                Pegasus
              </span>
              <span className="text-[#8E8EA0] text-base font-normal">Docs</span>
            </div>
          </Link>

          {/* Breadcrumb path if available */}
          {activeCategory && activeTitle && (
            <div className="hidden md:flex items-center space-x-1.5 pl-3 border-l border-[#242434] text-xs font-mono text-[#8E8EA0]">
              <span className="capitalize">{activeCategory}</span>
              <ChevronRight className="w-3.5 h-3.5 text-[#525266]" />
              <span className="text-white truncate max-w-[200px]">{activeTitle}</span>
            </div>
          )}
        </div>

        {/* Center / Right Controls */}
        <div className="flex items-center space-x-3 sm:space-x-4">
          {/* Quick Category Nav Links (Desktop) */}
          <nav className="hidden xl:flex items-center space-x-5 text-xs font-medium text-[#8E8EA0]">
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
            className="flex items-center justify-between w-40 sm:w-64 h-9 px-3 rounded-lg bg-[#12121A] border border-[#262638] text-xs text-[#8E8EA0] hover:border-[#3B82F6]/60 hover:text-white transition-all shadow-inner group"
          >
            <div className="flex items-center space-x-2 truncate">
              <Search className="w-3.5 h-3.5 text-[#6E6E82] group-hover:text-[#3B82F6] transition-colors" />
              <span className="truncate">Search Docs...</span>
            </div>
            <kbd className="hidden sm:inline-flex items-center px-1.5 py-0.5 text-[10px] font-mono text-[#7E7E94] bg-[#1A1A26] border border-[#2B2B3D] rounded">
              ⌘K
            </kbd>
          </button>

          {/* Language Switcher */}
          <div className="flex items-center rounded-lg bg-[#12121A] border border-[#262638] p-0.5 text-xs font-mono">
            <button
              onClick={() => setLanguage('en')}
              className={`px-2 py-1 rounded transition-colors ${
                language === 'en' ? 'bg-[#2563EB] text-white font-semibold' : 'text-[#8E8EA0] hover:text-white'
              }`}
            >
              EN
            </button>
            <button
              onClick={() => setLanguage('ru')}
              className={`px-2 py-1 rounded transition-colors ${
                language === 'ru' ? 'bg-[#2563EB] text-white font-semibold' : 'text-[#8E8EA0] hover:text-white'
              }`}
            >
              RU
            </button>
          </div>

          {/* Return to Platform Link */}
          <Link
            href="/platform"
            className="hidden sm:inline-flex items-center space-x-1.5 text-xs font-medium text-[#8E8EA0] hover:text-white transition-colors pl-2"
          >
            <ArrowLeft className="w-3.5 h-3.5" />
            <span>Platform</span>
          </Link>
        </div>
      </div>
    </header>
  );
}
