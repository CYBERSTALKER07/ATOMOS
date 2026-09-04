'use client';

import { useLanguage } from '@/app/context/LanguageContext';
import type { FleekTickerItem } from '@/app/data/fleekPageContent';

type FleekTickerProps = {
  items: FleekTickerItem[];
};

export default function FleekTicker({ items }: FleekTickerProps) {
  const { language } = useLanguage();
  const text = items.map((i) => i.text).join(' //////// ');

  return (
    <div className="fleek-ticker" aria-label={language === 'ru' ? 'Живые обновления сети' : 'Live network updates'}>
      <div className="fleek-ticker__track">
        <span>{text}</span>
        <span aria-hidden>{text}</span>
      </div>
    </div>
  );
}
