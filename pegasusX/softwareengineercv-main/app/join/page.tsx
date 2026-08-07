'use client';

import { useEffect, useState } from 'react';
import dynamic from 'next/dynamic';
import PixelDualHero from '@/app/components/visuals/PixelDualHero';
import FleekSecondaryLayout from '@/app/components/fleek/FleekSecondaryLayout';
import ContentCard, { EDITORIAL_IMAGES } from '@/app/components/ContentCard';
import ChamferButton from '@/app/components/ChamferButton';
import { usePerfProfile } from '@/app/hooks/useDevice';
import LazyWhenInView from '@/app/components/LazyWhenInView';
import { useLanguage } from '@/app/context/LanguageContext';

const Lanyard = dynamic(() => import('@/app/components/Lanyard'), { ssr: false });

export default function JoinPage() {
  const { allowHeavyFx } = usePerfProfile();
  const { t, language } = useLanguage();
  const [formData, setFormData] = useState({
    name: '',
    email: '',
    position: 'Supplier Operations',
    portfolio: '',
    message: '',
  });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitStatus, setSubmitStatus] = useState<'idle' | 'success' | 'error'>('idle');
  const [errorMessage, setErrorMessage] = useState('');
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    document.title = language === 'ru' ? 'Запросить демо | Pegasus' : 'Request Demo | Pegasus';
  }, [language]);

  const validate = () => {
    const next: Record<string, string> = {};
    if (!formData.name.trim()) next.name = t('join_err_name');
    if (!formData.email.trim()) next.email = t('join_err_email');
    else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) next.email = t('join_err_email_valid');
    if (!formData.position.trim()) next.position = t('join_err_role');
    setFieldErrors(next);
    return Object.keys(next).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!validate()) return;

    setIsSubmitting(true);
    setSubmitStatus('idle');
    setErrorMessage('');

    try {
      const response = await fetch('/api/apply', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(formData),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || 'Failed to submit');

      const stored = localStorage.getItem('team_applications');
      const applications = stored ? JSON.parse(stored) : [];
      applications.unshift(data.application);
      localStorage.setItem('team_applications', JSON.stringify(applications));

      setSubmitStatus('success');
      setFormData({ name: '', email: '', position: 'Supplier Operations', portfolio: '', message: '' });
      setFieldErrors({});
    } catch (error) {
      setSubmitStatus('error');
      setErrorMessage(error instanceof Error ? error.message : 'Something went wrong');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>
  ) => {
    const { name, value } = e.target;
    setFormData((prev) => ({ ...prev, [name]: value }));
    if (fieldErrors[name]) {
      setFieldErrors((prev) => {
        const next = { ...prev };
        delete next[name];
        return next;
      });
    }
  };

  return (
    <FleekSecondaryLayout
      activeHref="/join"
      sectionTitle="REQUEST DEMO"
      title={t('join_title')}
      summary={t('join_subtitle')}
      primaryHref="#demo-form"
      primaryLabel={t('btn_request_demo')}
      secondaryHref="/platform"
      secondaryLabel={t('nav_tour').toUpperCase()}
      showStack={false}
      heroVisual={<PixelDualHero />}
      section06={
        <>
          <section className="docs-section">
            <p className="font-mono text-[10px] uppercase tracking-widest text-white/40">{t('join_what_you_see')}</p>
            <h2 className="mt-3 max-w-3xl text-3xl font-semibold tracking-tight">{t('join_walkthrough')}</h2>
            <div className="mt-10 grid gap-12 lg:grid-cols-2 lg:gap-16">
              <div className="space-y-3">
                {[
                  { tag: t('join_card1_tag'), title: t('join_card1_title'), desc: t('join_card1_desc'), img: 0 },
                  { tag: t('join_card2_tag'), title: t('join_card2_title'), desc: t('join_card2_desc'), img: 1 },
                  { tag: t('join_card3_tag'), title: t('join_card3_title'), desc: t('join_card3_desc'), img: 2 },
                ].map((card) => (
                  <ContentCard
                    key={card.title}
                    variant="split"
                    tone="dark"
                    tag={card.tag}
                    title={card.title}
                    description={card.desc}
                    image={EDITORIAL_IMAGES[card.img]}
                  />
                ))}
              </div>

              <div id="demo-form">
                {allowHeavyFx && (
                  <LazyWhenInView
                    className="docs-surface relative mb-8 hidden h-48 overflow-hidden lg:block lg:h-64"
                    minHeight="256px"
                  >
                    <Lanyard position={[0, 0, 30]} gravity={[0, -40, 0]} fov={20} transparent />
                  </LazyWhenInView>
                )}

                <form onSubmit={handleSubmit} className="docs-surface docs-grain p-6 md:p-8" noValidate>
                <h2 className="text-xl font-semibold">{t('join_book_walkthrough')}</h2>
                <div className="mt-6 space-y-4">
                  {[
                    { name: 'name', label: t('join_full_name_form'), type: 'text', required: true, autoComplete: 'name' },
                    { name: 'email', label: t('join_email_form'), type: 'email', required: true, autoComplete: 'email' },
                    { name: 'portfolio', label: t('join_portfolio_form'), type: 'url', required: false, autoComplete: 'url' },
                  ].map((f) => (
                    <div key={f.name} className="docs-form-field">
                      <label htmlFor={`join-${f.name}`} className="text-xs font-mono uppercase tracking-wider text-white/50">
                        {f.label}
                        {f.required ? ' *' : ''}
                      </label>
                      <input
                        id={`join-${f.name}`}
                        type={f.type}
                        name={f.name}
                        value={formData[f.name as keyof typeof formData]}
                        onChange={handleChange}
                        required={f.required}
                        disabled={isSubmitting}
                        autoComplete={f.autoComplete}
                        aria-invalid={Boolean(fieldErrors[f.name])}
                        aria-describedby={fieldErrors[f.name] ? `join-${f.name}-error` : undefined}
                        className="docs-input disabled:opacity-50"
                      />
                      {fieldErrors[f.name] ? (
                        <p id={`join-${f.name}-error`} className="text-sm text-[#FE5934]" role="alert">
                          {fieldErrors[f.name]}
                        </p>
                      ) : null}
                    </div>
                  ))}
                  <div className="docs-form-field">
                    <label htmlFor="join-position" className="text-xs font-mono uppercase tracking-wider text-white/50">
                      {t('join_role_form')}
                    </label>
                    <select
                      id="join-position"
                      name="position"
                      value={formData.position}
                      onChange={handleChange}
                      disabled={isSubmitting}
                      aria-invalid={Boolean(fieldErrors.position)}
                      className="docs-select disabled:opacity-50"
                    >
                      <option value="Supplier Operations">{t('join_role_supplier', 'Supplier Operations')}</option>
                      <option value="Warehouse Manager">{t('join_role_warehouse', 'Warehouse Manager')}</option>
                      <option value="Fleet / Dispatch Lead">{t('join_role_fleet', 'Fleet / Dispatch Lead')}</option>
                      <option value="IT / Platform Owner">{t('join_role_it', 'IT / Platform Owner')}</option>
                      <option value="Executive / Founder">{t('join_role_executive', 'Executive / Founder')}</option>
                    </select>
                    {fieldErrors.position ? (
                      <p className="text-sm text-[#FE5934]" role="alert">
                        {fieldErrors.position}
                      </p>
                    ) : null}
                  </div>
                  <div className="docs-form-field">
                    <label htmlFor="join-message" className="text-xs font-mono uppercase tracking-wider text-white/50">
                      {t('join_about_network')}
                    </label>
                    <textarea
                      id="join-message"
                      name="message"
                      value={formData.message}
                      onChange={handleChange}
                      rows={4}
                      disabled={isSubmitting}
                      className="docs-textarea disabled:opacity-50"
                      placeholder={t('join_network_placeholder')}
                    />
                  </div>
                </div>
                {submitStatus === 'success' && (
                  <p className="mt-4 border border-[#8DDC96]/40 bg-[#8DDC96]/15 p-3 text-center text-sm font-medium text-[#8DDC96]" role="status">
                    {t('join_success_msg')}
                  </p>
                )}
                {submitStatus === 'error' && (
                  <p className="mt-4 border border-[#FE5934]/40 bg-[#FE5934]/15 p-3 text-center text-sm text-[#FE5934]" role="alert">
                    {errorMessage}
                  </p>
                )}
                <div className="mt-6">
                  <ChamferButton type="submit" variant="fill" className="w-full justify-center" disabled={isSubmitting}>
                    {isSubmitting ? t('join_submitting_btn') : t('join_submit_btn')}
                  </ChamferButton>
                </div>
              </form>
              </div>
            </div>
          </section>
        </>
      }
    />
  );
}
