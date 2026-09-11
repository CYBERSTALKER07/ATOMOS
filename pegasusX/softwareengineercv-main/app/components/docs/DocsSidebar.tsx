'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import {
  BookOpen,
  ChevronDown,
  Layers,
  Cpu,
  ShieldAlert,
  Code2,
  CheckCircle2,
  X,
  Sparkles,
} from 'lucide-react';
import { DOC_CATEGORIES, DocCategory } from '@/app/data/docsData';

type DocsSidebarProps = {
  mobileOpen: boolean;
  onCloseMobile: () => void;
};

const CATEGORY_ICONS: Record<string, React.ReactNode> = {
  guides: <BookOpen className="w-4 h-4" />,
  roles: <Layers className="w-4 h-4" />,
  protocols: <Cpu className="w-4 h-4" />,
  playbooks: <ShieldAlert className="w-4 h-4" />,
  api: <Code2 className="w-4 h-4" />,
};

export default function DocsSidebar({ mobileOpen, onCloseMobile }: DocsSidebarProps) {
  const pathname = usePathname();
  // Keep all categories expanded by default for easy browsing, allow toggling
  const [collapsedCategories, setCollapsedCategories] = useState<Record<string, boolean>>({});

  const toggleCategory = (catId: string) => {
    setCollapsedCategories((prev) => ({
      ...prev,
      [catId]: !prev[catId],
    }));
  };

  const content = (
    <aside className="w-full lg:w-72 xl:w-80 shrink-0 border-r border-[#1F1F2B] bg-[#09090D] h-full overflow-y-auto py-6 px-4 space-y-7">
      {/* Mobile Close Button */}
      <div className="lg:hidden flex items-center justify-between pb-4 border-b border-[#1F1F2B]">
        <div className="flex items-center space-x-2">
          <Sparkles className="w-4 h-4 text-[#3B82F6]" />
          <span className="font-semibold text-white text-sm">Documentation Tree</span>
        </div>
        <button
          onClick={onCloseMobile}
          className="p-1 rounded-lg text-[#8E8EA0] hover:text-white hover:bg-[#161622]"
        >
          <X className="w-5 h-5" />
        </button>
      </div>

      {/* System Status Pill */}
      <div className="px-3 py-2 rounded-xl bg-[#111118] border border-[#22222E] flex items-center justify-between">
        <div className="flex items-center space-x-2">
          <span className="relative flex h-2 w-2">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
            <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
          </span>
          <span className="text-[11px] font-mono font-medium text-[#B0B0C2]">Pegasus OS v4.2</span>
        </div>
        <span className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-[#1D1D28] text-emerald-400 border border-emerald-500/20">
          OPERATIONAL
        </span>
      </div>

      {/* Categories Tree */}
      <div className="space-y-6">
        {DOC_CATEGORIES.map((category) => {
          const isCollapsed = !!collapsedCategories[category.id];
          return (
            <div key={category.id} className="space-y-1.5">
              {/* Category Header (Clickable Accordion) */}
              <button
                onClick={() => toggleCategory(category.id)}
                className="w-full flex items-center justify-between px-2 py-1.5 text-xs font-semibold uppercase tracking-wider text-[#7E7E94] hover:text-white transition-colors group"
              >
                <div className="flex items-center space-x-2">
                  <span className="text-[#5A5A70] group-hover:text-[#3B82F6] transition-colors">
                    {CATEGORY_ICONS[category.id] || <BookOpen className="w-4 h-4" />}
                  </span>
                  <span>{category.title}</span>
                </div>
                <div className="flex items-center space-x-1.5">
                  {category.badge && (
                    <span className="text-[9px] font-mono px-1.5 py-0.5 rounded bg-[#161622] text-[#8E8EA0] border border-[#252535]">
                      {category.badge}
                    </span>
                  )}
                  <ChevronDown
                    className={`w-3.5 h-3.5 text-[#5A5A70] transition-transform duration-200 ${
                      isCollapsed ? '-rotate-90' : 'rotate-0'
                    }`}
                  />
                </div>
              </button>

              {/* Article Links */}
              {!isCollapsed && (
                <div className="space-y-0.5 pl-3 border-l border-[#1F1F2C] ml-3 mt-1">
                  {category.articles.map((article) => {
                    const href = `/docs/${category.id}/${article.slug}`;
                    const isActive = pathname === href || (pathname === '/docs' && article.slug === 'introduction');

                    return (
                      <Link
                        key={article.slug}
                        href={href}
                        onClick={onCloseMobile}
                        className={`group flex items-center justify-between px-3 py-2 text-xs rounded-lg transition-all ${
                          isActive
                            ? 'bg-[#182038] text-[#60A5FA] font-medium border border-[#2563EB]/40 shadow-sm'
                            : 'text-[#9A9AA8] hover:text-white hover:bg-[#12121A]'
                        }`}
                      >
                        <span className="truncate">{article.title}</span>
                        {article.badge && (
                          <span
                            className={`text-[9px] font-mono px-1.5 py-0.2 rounded shrink-0 ml-2 transition-colors ${
                              isActive
                                ? 'bg-[#2563EB]/30 text-[#93C5FD] border border-[#3B82F6]/50'
                                : 'bg-[#151520] text-[#717182] border border-[#242434] group-hover:text-[#9A9AA8]'
                            }`}
                          >
                            {article.badge}
                          </span>
                        )}
                      </Link>
                    );
                  })}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </aside>
  );

  return (
    <>
      {/* Desktop Persistent Sidebar */}
      <div className="hidden lg:block shrink-0 sticky top-16 h-[calc(100vh-4rem)]">
        {content}
      </div>

      {/* Mobile Drawer Overlay */}
      {mobileOpen && (
        <div className="fixed inset-0 z-50 lg:hidden flex">
          <div
            className="fixed inset-0 bg-black/80 backdrop-blur-sm transition-opacity"
            onClick={onCloseMobile}
          />
          <div className="relative w-80 max-w-[85vw] bg-[#09090D] shadow-2xl h-full z-10">
            {content}
          </div>
        </div>
      )}
    </>
  );
}
