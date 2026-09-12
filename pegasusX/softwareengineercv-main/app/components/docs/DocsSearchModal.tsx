'use client';

import React, { useEffect, useState, useRef } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { Search, X, ArrowRight, FileText, Sparkles, Hash } from 'lucide-react';
import { searchDocs, DocArticle, DocCategory } from '@/app/data/docsData';

type DocsSearchModalProps = {
  isOpen: boolean;
  onClose: () => void;
};

export default function DocsSearchModal({ isOpen, onClose }: DocsSearchModalProps) {
  const [query, setQuery] = useState('');
  const [selectedIndex, setSelectedIndex] = useState(0);
  const router = useRouter();
  const inputRef = useRef<HTMLInputElement | null>(null);

  const results = searchDocs(query);

  useEffect(() => {
    if (isOpen) {
      setTimeout(() => inputRef.current?.focus(), 50);
      setSelectedIndex(0);
    } else {
      setQuery('');
    }
  }, [isOpen]);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!isOpen) return;

      if (e.key === 'Escape') {
        onClose();
      } else if (e.key === 'ArrowDown') {
        e.preventDefault();
        setSelectedIndex((prev) => (results.length > 0 ? (prev + 1) % results.length : 0));
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        setSelectedIndex((prev) => (results.length > 0 ? (prev - 1 + results.length) % results.length : 0));
      } else if (e.key === 'Enter' && results.length > 0) {
        e.preventDefault();
        const selected = results[selectedIndex];
        if (selected) {
          router.push(`/docs/${selected.category.id}/${selected.article.slug}`);
          onClose();
        }
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, results, selectedIndex, router, onClose]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-20 px-4 bg-black/80 backdrop-blur-md transition-all">
      <div
        className="w-full max-w-2xl bg-[#0F0F14] border border-[#262633] rounded-2xl shadow-2xl overflow-hidden flex flex-col max-h-[80vh] animate-in fade-in zoom-in-95 duration-150"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Search Input Bar */}
        <div className="flex items-center px-4 py-3.5 border-b border-[#22222C] bg-[#14141C]">
          <Search className="w-5 h-5 text-[#8E8EA0] mr-3 shrink-0" />
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setSelectedIndex(0);
            }}
            placeholder="Search documentation, operators, protocols, APIs... (Type to filter)"
            className="w-full bg-transparent text-white placeholder-[#6E6E80] text-sm focus:outline-none"
          />
          {query && (
            <button
              onClick={() => setQuery('')}
              className="p-1 text-[#8E8EA0] hover:text-white transition-colors"
            >
              <X className="w-4 h-4" />
            </button>
          )}
          <kbd className="hidden sm:inline-flex items-center px-2 py-0.5 ml-2 text-[10px] font-mono text-[#8E8EA0] bg-[#1F1F2B] border border-[#2D2D3D] rounded">
            ESC
          </kbd>
        </div>

        {/* Search Results List */}
        <div className="flex-1 overflow-y-auto p-2 divide-y divide-[#1D1D28]/60">
          {query.trim() === '' ? (
            <div className="py-12 px-6 text-center">
              <div className="w-12 h-12 mx-auto mb-3 rounded-full bg-[#181822] border border-[#2A2A3A] flex items-center justify-center text-[#3B82F6]">
                <Sparkles className="w-5 h-5" />
              </div>
              <h4 className="text-white font-medium text-sm">Instant Documentation Search</h4>
              <p className="text-xs text-[#8E8EA0] mt-1 max-w-md mx-auto">
                Quickly locate architecture guides, operator manuals, mathematical CVRP solver docs, or operational edge-case playbooks.
              </p>
              <div className="mt-4 flex flex-wrap justify-center gap-2">
                {['CVRP Optimizer', 'Warehouse Hub', 'Double-Entry Ledger', 'Tamper Seal', 'REST API'].map((tag) => (
                  <button
                    key={tag}
                    onClick={() => setQuery(tag)}
                    className="text-xs px-2.5 py-1 rounded-full bg-[#191924] border border-[#282838] text-[#B5B5C5] hover:text-white hover:border-[#3B82F6] transition-colors"
                  >
                    {tag}
                  </button>
                ))}
              </div>
            </div>
          ) : results.length === 0 ? (
            <div className="py-12 px-6 text-center">
              <p className="text-sm text-[#8E8EA0]">No documentation articles found matching &quot;{query}&quot;.</p>
              <p className="text-xs text-[#5E5E70] mt-1">Try searching for &quot;dispatch&quot;, &quot;ledger&quot;, &quot;driver&quot;, or &quot;outbox&quot;.</p>
            </div>
          ) : (
            results.map((res, idx) => {
              const isSelected = idx === selectedIndex;
              return (
                <Link
                  key={`${res.category.id}-${res.article.slug}`}
                  href={`/docs/${res.category.id}/${res.article.slug}`}
                  onClick={onClose}
                  onMouseEnter={() => setSelectedIndex(idx)}
                  className={`flex items-start justify-between p-3 rounded-xl transition-colors ${
                    isSelected ? 'bg-[#1D1D2C] border border-[#3A3A52]' : 'hover:bg-[#161622] border border-transparent'
                  }`}
                >
                  <div className="flex items-start space-x-3 min-w-0 pr-3">
                    <div className="w-8 h-8 rounded-lg bg-[#14141E] border border-[#2A2A3A] flex items-center justify-center text-[#3B82F6] shrink-0 mt-0.5">
                      <FileText className="w-4 h-4" />
                    </div>
                    <div className="min-w-0">
                      <div className="flex items-center space-x-2">
                        <span className="text-xs font-mono text-[#3B82F6] uppercase tracking-wider">
                          {res.category.title}
                        </span>
                        <span className="text-[#4E4E60] text-xs">/</span>
                        <span className="text-xs text-[#8E8EA0] font-mono">{res.article.version}</span>
                        {res.article.badge && (
                          <span className="text-[10px] px-1.5 py-0.2 rounded bg-[#202030] text-[#9E9EB0] border border-[#303044]">
                            {res.article.badge}
                          </span>
                        )}
                      </div>
                      <h4 className="text-sm font-semibold text-white truncate mt-0.5">
                        {res.article.title}
                      </h4>
                      <p className="text-xs text-[#8E8EA0] line-clamp-1 mt-0.5">
                        {res.article.leadSentence}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center space-x-1 shrink-0 pt-2 text-[#5E5E70]">
                    <span className="text-[10px] font-mono text-[#6E6E80] mr-1 hidden sm:inline">Jump to</span>
                    <ArrowRight className={`w-4 h-4 ${isSelected ? 'text-[#3B82F6]' : 'text-[#4E4E60]'}`} />
                  </div>
                </Link>
              );
            })
          )}
        </div>

        {/* Footer info */}
        <div className="px-4 py-2.5 bg-[#12121A] border-t border-[#20202C] flex items-center justify-between text-[11px] text-[#6E6E80] font-mono">
          <div className="flex items-center space-x-3">
            <span><kbd className="px-1.5 py-0.5 bg-[#1C1C26] rounded border border-[#2D2D3E]">↑</kbd> <kbd className="px-1.5 py-0.5 bg-[#1C1C26] rounded border border-[#2D2D3E]">↓</kbd> Navigate</span>
            <span><kbd className="px-1.5 py-0.5 bg-[#1C1C26] rounded border border-[#2D2D3E]">↵</kbd> Select</span>
          </div>
          <span>Pegasus OS Documentation v4.2</span>
        </div>
      </div>
    </div>
  );
}
