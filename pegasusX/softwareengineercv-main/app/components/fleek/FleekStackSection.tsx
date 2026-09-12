'use client';

import { getStackFeatures } from '@/app/data/fleekPageContent';
import FleekSection from './FleekSection';
import { useLanguage } from '@/app/context/LanguageContext';

type FleekStackSectionProps = {
  sectionTitle?: string;
  title?: string;
  features?: string[];
};

export default function FleekStackSection({
  sectionTitle,
  title,
  features,
}: FleekStackSectionProps) {
  const { language } = useLanguage();
  const resolvedFeatures = features ?? getStackFeatures(language);
  const resolvedSectionTitle =
    sectionTitle ??
    (language === 'ru' ? 'ЛОГИСТИКЕ — ПРЕИМУЩЕСТВО' : 'GIVING LOGISTICS AN EDGE');
  const resolvedTitle =
    title ??
    (language === 'ru'
      ? 'Современная инфраструктура диспетчеризации'
      : 'Modern dispatch infrastructure');
  const layers =
    language === 'ru'
      ? ['01 Клиентский слой', '02 БЫСТРАЯ ДОСТАВКА', '03 Сервисный слой']
      : ['01 Client layer', '02 FAST DELIVERY', '03 Service layer'];

  return (
    <FleekSection id="fleek-section-02" number="02" title={resolvedSectionTitle}>
      <div className="fleek-stack">
        <div className="fleek-stack__visual" aria-hidden>
          <div className="fleek-stack__layer fleek-stack__layer--client">
            <span>{layers[0]}</span>
            <div className="fleek-stack__devices">
              <span>⌚</span><span>📱</span><span>🖥</span>
            </div>
          </div>
          <div className="fleek-stack__layer fleek-stack__layer--edge">
            <span>{layers[1]}</span>
            <div className="fleek-stack__bolt">⚡</div>
          </div>
          <div className="fleek-stack__layer fleek-stack__layer--service">
            <span>{layers[2]}</span>
            <div className="fleek-stack__grid">
              {['D', 'F', 'T', 'P', 'W', 'R'].map((c, i) => (
                <span key={c} className={i < 2 ? 'is-accent' : ''}>{c}</span>
              ))}
            </div>
          </div>
        </div>
        <div className="fleek-stack__copy">
          <h3 className="fleek-stack__heading">{resolvedTitle}</h3>
          <ul className="fleek-stack__features">
            {resolvedFeatures.map((f) => (
              <li key={f}>
                <span className="fleek-stack__bolt-icon" aria-hidden>⚡</span>
                {f}
              </li>
            ))}
          </ul>
        </div>
      </div>
    </FleekSection>
  );
}
