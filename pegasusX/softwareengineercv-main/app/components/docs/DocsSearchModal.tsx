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
 className="w-full max-w-2xl bg-black border border-white/20 rounded-none overflow-hidden flex flex-col max-h-[80vh] animate-in fade-in zoom-in-95 duration-150"
 onClick={(e) => e.stopPropagation()}
 >
 {/* Search Input Bar */}
 <div className="flex items-center px-4 py-3.5 border-b border-white/10 bg-black">
 <Search className="w-5 h-5 text-zinc-400 mr-3 shrink-0" />
 <input
 ref={inputRef}
 type="text"
 value={query}
 onChange={(e) => {
 setQuery(e.target.value);
 setSelectedIndex(0);
 }}
 placeholder="Search documentation, operators, protocols, APIs... (Type to filter)"
 className="w-full bg-transparent text-white placeholder-zinc-500 text-sm focus:outline-none"
 />
 {query && (
 <button
 onClick={() => setQuery('')}
 className="p-1 text-zinc-400 hover:text-white transition-colors"
 >
 <X className="w-4 h-4" />
 </button>
 )}
 <kbd className="hidden sm:inline-flex items-center px-2 py-0.5 ml-2 text-[10px] font-mono text-zinc-400 bg-black border border-white/10 rounded-none">
 ESC
 </kbd>
 </div>

 {/* Search Results List */}
 <div className="flex-1 overflow-y-auto p-2 divide-y divide-white/5">
 {query.trim() === '' ? (
 <div className="py-12 px-6 text-center">
 <div className="w-12 h-12 mx-auto mb-3 rounded-none bg-black border border-white/10 flex items-center justify-center text-white">
 <Sparkles className="w-5 h-5" />
 </div>
 <h4 className="text-white font-medium text-sm">Instant Documentation Search</h4>
 <p className="text-xs text-zinc-400 mt-1 max-w-md mx-auto">
 Quickly locate architecture guides, operator manuals, mathematical CVRP solver docs, or operational edge-case playbooks.
 </p>
 <div className="mt-4 flex flex-wrap justify-center gap-2">
 {['CVRP Optimizer', 'Warehouse Hub', 'Double-Entry Ledger', 'Tamper Seal', 'REST API'].map((tag) => (
 <button
 key={tag}
 onClick={() => setQuery(tag)}
 className="text-xs px-2.5 py-1 rounded-none bg-black border border-white/10 text-zinc-300 hover:text-white hover:border-white transition-colors"
 >
 {tag}
 </button>
 ))}
 </div>
 </div>
 ) : results.length === 0 ? (
 <div className="py-12 px-6 text-center">
 <p className="text-sm text-zinc-400">No documentation articles found matching &quot;{query}&quot;.</p>
 <p className="text-xs text-zinc-500 mt-1">Try searching for &quot;dispatch&quot;, &quot;ledger&quot;, &quot;driver&quot;, or &quot;outbox&quot;.</p>
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
 className={`flex items-start justify-between p-3 rounded-none transition-colors ${
 isSelected ? 'bg-black border border-white/30' : 'hover:bg-black border border-transparent'
 }`}
 >
 <div className="flex items-start space-x-3 min-w-0 pr-3">
 <div className="w-8 h-8 rounded-none bg-black border border-white/10 flex items-center justify-center text-white shrink-0 mt-0.5">
 <FileText className="w-4 h-4" />
 </div>
 <div className="min-w-0">
 <div className="flex items-center space-x-2">
 <span className="text-xs font-mono text-white font-semibold uppercase tracking-wider">
 {res.category.title}
 </span>
 <span className="text-zinc-600 text-xs">/</span>
 <span className="text-xs text-zinc-400 font-mono">{res.article.version}</span>
 {res.article.badge && (
 <span className="text-[10px] px-1.5 py-0.2 rounded-none bg-white/5 text-zinc-400 border border-white/10">
 {res.article.badge}
 </span>
 )}
 </div>
 <h4 className="text-sm font-semibold text-white truncate mt-0.5">
 {res.article.title}
 </h4>
 <p className="text-xs text-zinc-400 line-clamp-1 mt-0.5">
 {res.article.leadSentence}
 </p>
 </div>
 </div>
 <div className="flex items-center space-x-1 shrink-0 pt-2 text-zinc-500">
 <span className="text-[10px] font-mono text-zinc-500 mr-1 hidden sm:inline">Jump to</span>
 <ArrowRight className={`w-4 h-4 ${isSelected ? 'text-white' : 'text-zinc-600'}`} />
 </div>
 </Link>
 );
 })
 )}
 </div>

 {/* Footer info */}
 <div className="px-4 py-2.5 bg-black border-t border-white/10 flex items-center justify-between text-[11px] text-zinc-400 font-mono">
 <div className="flex items-center space-x-3">
 <span><kbd className="px-1.5 py-0.5 bg-black rounded-none border border-white/10">↑</kbd> <kbd className="px-1.5 py-0.5 bg-black rounded-none border border-white/10">↓</kbd> Navigate</span>
 <span><kbd className="px-1.5 py-0.5 bg-black rounded-none border border-white/10">↵</kbd> Select</span>
 </div>
 <span>Pegasus OS Documentation v4.2</span>
 </div>
 </div>
 </div>
 );
}
