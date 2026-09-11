'use client';

import React, { useState, useEffect } from 'react';
import DocsHeader from './DocsHeader';
import DocsSidebar from './DocsSidebar';
import DocsSearchModal from './DocsSearchModal';
import { DocArticle } from '@/app/data/docsData';

type DocsLayoutProps = {
  children: React.ReactNode;
  activeArticle?: DocArticle;
};

export default function DocsLayout({ children, activeArticle }: DocsLayoutProps) {
  const [searchOpen, setSearchOpen] = useState(false);
  const [mobileSidebarOpen, setMobileSidebarOpen] = useState(false);

  // Global ⌘K / Ctrl+K keyboard listener
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setSearchOpen((prev) => !prev);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  return (
    <div className="min-h-screen bg-[#07070A] text-white flex flex-col font-sans selection:bg-[#2563EB] selection:text-white">
      {/* Search Modal */}
      <DocsSearchModal isOpen={searchOpen} onClose={() => setSearchOpen(false)} />

      {/* Top Header */}
      <DocsHeader
        onOpenSearch={() => setSearchOpen(true)}
        onToggleMobileSidebar={() => setMobileSidebarOpen(true)}
        activeCategory={activeArticle?.categoryId}
        activeTitle={activeArticle?.shortTitle || activeArticle?.title}
      />

      {/* 2-Column Documentation Canvas */}
      <div className="flex-1 max-w-[1520px] w-full mx-auto flex flex-row">
        {/* Left Sidebar Tree */}
        <DocsSidebar
          mobileOpen={mobileSidebarOpen}
          onCloseMobile={() => setMobileSidebarOpen(false)}
        />

        {/* Main Content Area */}
        <main className="flex-1 min-w-0 px-4 sm:px-8 lg:px-12 py-8 lg:py-10 max-w-5xl">
          {children}
        </main>
      </div>
    </div>
  );
}
