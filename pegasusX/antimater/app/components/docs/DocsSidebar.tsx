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
 <aside className="w-full lg:w-72 xl:w-80 shrink-0 border-r border-white/10 bg-black h-full overflow-y-auto py-6 px-4 space-y-7">
 {/* Mobile Close Button */}
 <div className="lg:hidden flex items-center justify-between pb-4 border-b border-white/10">
 <div className="flex items-center space-x-2">
 <Sparkles className="w-4 h-4 text-white" />
 <span className="font-semibold text-white text-sm">Documentation Tree</span>
 </div>
 <button
 onClick={onCloseMobile}
 className="p-1 rounded-none text-zinc-400 hover:text-white hover:bg-black"
 >
 <X className="w-5 h-5" />
 </button>
 </div>

 {/* System Status Pill */}
 <div className="px-3 py-2 rounded-none bg-black border border-white/10 flex items-center justify-between">
 <div className="flex items-center space-x-2">
 <span className="relative flex h-2 w-2">
 <span className="relative inline-flex rounded-none h-2 w-2 bg-white"></span>
 </span>
 <span className="text-[11px] font-mono font-medium text-zinc-300">Pegasus OS v4.2</span>
 </div>
 <span className="text-[10px] font-mono px-1.5 py-0.5 rounded-none bg-white/10 text-white border border-white/20">
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
 className="w-full flex items-center justify-between px-2 py-1.5 text-xs font-semibold uppercase tracking-wider text-zinc-400 hover:text-white transition-colors group"
 >
 <div className="flex items-center space-x-2">
 <span className="text-zinc-500 group-hover:text-white transition-colors">
 {CATEGORY_ICONS[category.id] || <BookOpen className="w-4 h-4" />}
 </span>
 <span>{category.title}</span>
 </div>
 <div className="flex items-center space-x-1.5">
 {category.badge && (
 <span className="text-[9px] font-mono px-1.5 py-0.5 rounded-none bg-white/5 text-zinc-400 border border-white/10">
 {category.badge}
 </span>
 )}
 <ChevronDown
 className={`w-3.5 h-3.5 text-zinc-500 transition-transform duration-200 ${
 isCollapsed ? '-rotate-90' : 'rotate-0'
 }`}
 />
 </div>
 </button>

 {/* Article Links */}
 {!isCollapsed && (
 <div className="space-y-0.5 pl-3 border-l border-white/10 ml-3 mt-1">
 {category.articles.map((article) => {
 const href = `/docs/${category.id}/${article.slug}`;
 const isActive = pathname === href || (pathname === '/docs' && article.slug === 'introduction');

 return (
 <Link
 key={article.slug}
 href={href}
 onClick={onCloseMobile}
 className={`group flex items-center justify-between px-3 py-2 text-xs rounded-none transition-all ${
 isActive
 ? 'bg-white text-black font-semibold '
 : 'text-zinc-400 hover:text-white hover:bg-black'
 }`}
 >
 <span className="truncate">{article.title}</span>
 {article.badge && (
 <span
 className={`text-[9px] font-mono px-1.5 py-0.2 rounded-none shrink-0 ml-2 transition-colors ${
 isActive
 ? 'bg-black text-white'
 : 'bg-white/5 text-zinc-400 border border-white/10 group-hover:text-white'
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
 <div className="relative w-80 max-w-[85vw] bg-black border-r border-white/15 h-full z-10">
 {content}
 </div>
 </div>
 )}
 </>
 );
}
