'use client';

import type { O9Testimonial } from '@/app/data/o9FleekDefaults';
import { getTestimonials } from '@/app/data/o9FleekDefaults';
import { O9SectionLabel } from '@/app/components/page-sections/o9/O9Sections';
import { useLanguage } from '@/app/context/LanguageContext';

type O9TestimonialRowProps = {
  items?: O9Testimonial[];
};

export default function O9TestimonialRow({ items }: O9TestimonialRowProps) {
  const { language } = useLanguage();
  const list = items ?? getTestimonials(language);
  if (!list.length) return null;

  return (
    <section className="o9-section">
      <O9SectionLabel>{language === 'ru' ? 'ЦЕННОСТЬ ДЛЯ КОРПОРАТИВНЫХ КЛИЕНТОВ' : 'ENTERPRISE VALUE'}</O9SectionLabel>
      <h2 className="o9-section__title">{language === 'ru' ? 'Реальная ценность для логистических сетей' : 'Driving real network value'}</h2>
      <div className="o9-testimonial-grid">
        {list.slice(0, 3).map((item) => (
          <blockquote key={item.company} className="o9-card o9-testimonial-card">
            <p className="o9-testimonial-card__company">{item.company}</p>
            <p className="o9-testimonial-card__quote">&ldquo;{item.quote}&rdquo;</p>
            <footer className="o9-testimonial-card__meta">
              <span className="o9-testimonial-card__name">{item.name}</span>
              <span className="o9-testimonial-card__title">{item.title}</span>
            </footer>
          </blockquote>
        ))}
      </div>
    </section>
  );
}
