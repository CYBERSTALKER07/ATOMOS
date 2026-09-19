'use client';

import { useLanguage } from '@/app/context/LanguageContext';

type LocalizedLaneDividerProps = {
  index: string;
  labelKey: string;
};

export default function LocalizedLaneDivider({ index, labelKey }: LocalizedLaneDividerProps) {
  const { t } = useLanguage();

  return (
    <div className="lane-divider" role="presentation" aria-hidden>
      <span className="lane-divider__label">
        <strong>{index}</strong>
        <span aria-hidden> · </span>
        {t(labelKey as any)}
      </span>
    </div>
  );
}
