'use client';

import Link from 'next/link';
import type { TopicPage } from '@/app/data/topicTypes';
import { useLanguage } from '@/app/context/LanguageContext';
import { ArrowRight } from 'lucide-react';

type HubTopicGridProps = {
  hubId: string;
  hubLabel: string;
  topics: TopicPage[];
  layout?: 'uniform' | 'masonry' | 'featured';
};

export default function HubTopicGrid({
  hubId,
  hubLabel,
  topics,
}: HubTopicGridProps) {
  const { language, t } = useLanguage();
  const localizedTag = t(`nav_${hubId}`, hubLabel);

  return (
    <div className="mt-16 sm:mt-24 w-full flex flex-col border-t border-zinc-200 dark:border-white/20">
      {topics.map((topic) => {
        const content = topic.content[language] || topic.content.en;
        return (
          <Link
            key={topic.slug}
            href={topic.href}
            className="group relative flex flex-col md:flex-row md:items-center justify-between p-6 sm:p-10 border-b border-zinc-200 dark:border-white/20 hover:bg-zinc-100 dark:hover:bg-white transition-colors duration-300 overflow-hidden"
          >
            {/* The fill transition from left effect */}
            <div className="absolute inset-0 bg-zinc-200 dark:bg-white translate-x-[-101%] group-hover:translate-x-0 transition-transform duration-500 ease-[cubic-bezier(0.19,1,0.22,1)] z-0" />

            <div className="relative z-10 flex flex-col max-w-3xl">
              <p className="font-mono text-xs uppercase tracking-[0.2em] text-zinc-500 dark:text-white/40 group-hover:text-black/60 dark:group-hover:text-black/50 transition-colors duration-300 mb-3 sm:mb-4">
                [ {localizedTag} ]
              </p>
              <h3 className="text-3xl sm:text-5xl lg:text-6xl font-bold tracking-tighter text-zinc-900 dark:text-white group-hover:text-black transition-colors duration-300">
                {content.title}
              </h3>
              <p className="mt-4 text-sm sm:text-base text-zinc-600 dark:text-white/60 group-hover:text-black/80 transition-colors duration-300 font-mono leading-relaxed max-w-2xl">
                {content.summary}
              </p>
            </div>
            
            <div className="relative z-10 mt-8 md:mt-0 opacity-0 group-hover:opacity-100 transform translate-x-8 group-hover:translate-x-0 transition-all duration-500 hidden md:flex items-center justify-center p-4 bg-black text-white rounded-full">
              <ArrowRight className="w-8 h-8" />
            </div>
          </Link>
        );
      })}
    </div>
  );
}
