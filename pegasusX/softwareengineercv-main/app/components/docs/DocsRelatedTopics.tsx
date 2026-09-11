'use client';

import React from 'react';
import Link from 'next/link';
import { ArrowRight, ArrowLeft, Compass, BookmarkCheck, GitFork } from 'lucide-react';
import { DocRelatedTopic, getAllDocSlugs, getDocArticle } from '@/app/data/docsData';

type DocsRelatedTopicsProps = {
  currentCategory: string;
  currentSlug: string;
  relatedTopics: DocRelatedTopic[];
};

export default function DocsRelatedTopics({
  currentCategory,
  currentSlug,
  relatedTopics,
}: DocsRelatedTopicsProps) {
  // Compute previous and next articles in the global sequence
  const allSlugs = getAllDocSlugs();
  const currentIndex = allSlugs.findIndex(
    (s) => s.category === currentCategory && s.slug === currentSlug
  );

  const prevItem = currentIndex > 0 ? allSlugs[currentIndex - 1] : null;
  const nextItem = currentIndex >= 0 && currentIndex < allSlugs.length - 1 ? allSlugs[currentIndex + 1] : null;

  const prevArticle = prevItem ? getDocArticle(prevItem.category, prevItem.slug) : null;
  const nextArticle = nextItem ? getDocArticle(nextItem.category, nextItem.slug) : null;

  const badgeStyles = {
    'PREREQUISITE': 'bg-[#1F1633] text-[#C084FC] border-[#7C3AED]/40',
    'NEXT STEP': 'bg-[#13203E] text-[#60A5FA] border-[#2563EB]/40',
    'RELATED ENGINE': 'bg-[#0E2838] text-[#38BDF8] border-[#0284C7]/40',
    'EDGE CASE': 'bg-[#2E1810] text-[#FB923C] border-[#EA580C]/40',
    'PLAYBOOK': 'bg-[#2B121A] text-[#FB7185] border-[#E11D48]/40',
  };

  return (
    <div className="mt-16 pt-10 border-t border-[#1F1F2C] space-y-8">
      {/* Header */}
      <div className="space-y-1.5">
        <div className="flex items-center space-x-2 text-xs font-mono uppercase tracking-wider text-[#7E7E94]">
          <Compass className="w-4 h-4 text-[#3B82F6]" />
          <span>Recommended Next Steps & Contextual Topics</span>
        </div>
        <h3 className="text-xl sm:text-2xl font-bold text-white tracking-tight">
          Keep Exploring Pegasus Architecture & Operations
        </h3>
        <p className="text-xs sm:text-sm text-[#8E8EA0] max-w-2xl">
          Based on the invariants, physical actors, and data flows of this module, explore these recommended companion guides and operational playbooks.
        </p>
      </div>

      {/* Recommendations Cards Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {relatedTopics.map((topic, idx) => {
          const badgeCls = badgeStyles[topic.badge] || badgeStyles['NEXT STEP'];

          return (
            <Link
              key={idx}
              href={`/docs/${topic.categoryId}/${topic.slug}`}
              className="group relative p-5 rounded-2xl bg-[#0E0E15] border border-[#222234] hover:border-[#3B82F6]/60 hover:bg-[#11111C] transition-all duration-200 flex flex-col justify-between shadow-sm hover:shadow-[0_4px_20px_rgba(37,99,235,0.15)]"
            >
              <div>
                {/* Category & Badge */}
                <div className="flex items-center justify-between gap-2 mb-3">
                  <span className="text-[10px] font-mono uppercase tracking-wider text-[#717185]">
                    {topic.category}
                  </span>
                  <span className={`text-[9px] font-mono px-2 py-0.5 rounded-full border font-semibold ${badgeCls}`}>
                    {topic.badge}
                  </span>
                </div>

                {/* Title */}
                <h4 className="text-sm font-bold text-white group-hover:text-[#60A5FA] transition-colors leading-snug">
                  {topic.title}
                </h4>

                {/* Rationale / Reason */}
                <p className="mt-2 text-xs text-[#8E8EA0] leading-relaxed line-clamp-3">
                  {topic.reason}
                </p>
              </div>

              {/* Action Bottom */}
              <div className="mt-5 pt-3 border-t border-[#1C1C2A] flex items-center justify-between text-xs font-medium text-[#7E7E94] group-hover:text-white transition-colors">
                <span className="font-mono text-[11px]">View specification</span>
                <ArrowRight className="w-4 h-4 text-[#4E4E64] group-hover:text-[#3B82F6] group-hover:translate-x-1 transition-all" />
              </div>
            </Link>
          );
        })}
      </div>

      {/* Prev / Next Pagination Bar */}
      <div className="pt-6 grid grid-cols-1 sm:grid-cols-2 gap-4">
        {prevArticle ? (
          <Link
            href={`/docs/${prevItem!.category}/${prevItem!.slug}`}
            className="flex items-center space-x-3 p-4 rounded-xl border border-[#20202E] bg-[#0C0C12] hover:bg-[#12121C] hover:border-[#3A3A4E] transition-all group"
          >
            <ArrowLeft className="w-4 h-4 text-[#7E7E94] group-hover:text-white group-hover:-translate-x-1 transition-all shrink-0" />
            <div className="min-w-0">
              <span className="text-[10px] font-mono uppercase tracking-wider text-[#68687A] block">
                Previous Article
              </span>
              <span className="text-xs font-semibold text-white truncate block mt-0.5">
                {prevArticle.title}
              </span>
            </div>
          </Link>
        ) : (
          <div />
        )}

        {nextArticle && (
          <Link
            href={`/docs/${nextItem!.category}/${nextItem!.slug}`}
            className="flex items-center justify-end text-right space-x-3 p-4 rounded-xl border border-[#20202E] bg-[#0C0C12] hover:bg-[#12121C] hover:border-[#3A3A4E] transition-all group"
          >
            <div className="min-w-0">
              <span className="text-[10px] font-mono uppercase tracking-wider text-[#68687A] block">
                Next Article
              </span>
              <span className="text-xs font-semibold text-white truncate block mt-0.5">
                {nextArticle.title}
              </span>
            </div>
            <ArrowRight className="w-4 h-4 text-[#7E7E94] group-hover:text-white group-hover:translate-x-1 transition-all shrink-0" />
          </Link>
        )}
      </div>
    </div>
  );
}
