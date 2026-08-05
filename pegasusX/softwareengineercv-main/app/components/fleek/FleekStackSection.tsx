'use client';

import { FLEEK_STACK_FEATURES } from '@/app/data/fleekPageContent';
import FleekSection from './FleekSection';

type FleekStackSectionProps = {
  sectionTitle?: string;
  title?: string;
  features?: string[];
};

export default function FleekStackSection({
  sectionTitle = 'GIVING LOGISTICS AN EDGE',
  title = 'Modern dispatch infrastructure',
  features = [...FLEEK_STACK_FEATURES],
}: FleekStackSectionProps) {
  return (
    <FleekSection id="fleek-section-02" number="02" title={sectionTitle}>
      <div className="fleek-stack">
        <div className="fleek-stack__visual" aria-hidden>
          <div className="fleek-stack__layer fleek-stack__layer--client">
            <span>01 Client layer</span>
            <div className="fleek-stack__devices">
              <span>⌚</span><span>📱</span><span>🖥</span>
            </div>
          </div>
          <div className="fleek-stack__layer fleek-stack__layer--edge">
            <span>02 FAST DELIVERY</span>
            <div className="fleek-stack__bolt">⚡</div>
          </div>
          <div className="fleek-stack__layer fleek-stack__layer--service">
            <span>03 Service layer</span>
            <div className="fleek-stack__grid">
              {['D', 'F', 'T', 'P', 'W', 'R'].map((c, i) => (
                <span key={c} className={i < 2 ? 'is-accent' : ''}>{c}</span>
              ))}
            </div>
          </div>
        </div>
        <div className="fleek-stack__copy">
          <h3 className="fleek-stack__heading">{title}</h3>
          <ul className="fleek-stack__features">
            {features.map((f) => (
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
