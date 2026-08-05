'use client';

import { useEffect, useState } from 'react';
import dynamic from 'next/dynamic';
import PixelDualHero from '@/app/components/visuals/PixelDualHero';
import FleekSecondaryLayout from '@/app/components/fleek/FleekSecondaryLayout';
import ContentCard, { EDITORIAL_IMAGES } from '@/app/components/ContentCard';
import ChamferButton from '@/app/components/ChamferButton';
import { usePerfProfile } from '@/app/hooks/useDevice';
import LazyWhenInView from '@/app/components/LazyWhenInView';
import { O9TourCTA } from '@/app/components/page-sections/o9/O9PageChrome';

const Lanyard = dynamic(() => import('@/app/components/Lanyard'), { ssr: false });

export default function JoinPage() {
  const { allowHeavyFx } = usePerfProfile();
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
    document.title = 'Request Demo | Pegasus';
  }, []);

  const validate = () => {
    const next: Record<string, string> = {};
    if (!formData.name.trim()) next.name = 'Enter your full name.';
    if (!formData.email.trim()) next.email = 'Enter a work email.';
    else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) next.email = 'Enter a valid email.';
    if (!formData.position.trim()) next.position = 'Select your role.';
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
      title="Request a Demo"
      summary="See dispatch, tracking, and payments for supplier-led logistics networks — Spanner truth, six roles, portal and native."
      primaryHref="#demo-form"
      primaryLabel="BOOK WALKTHROUGH"
      secondaryHref="/platform"
      secondaryLabel="PLATFORM TOUR"
      showStack={false}
      dataExtra={<PixelDualHero />}
      section06={
        <>
          <section className="docs-section">
            <p className="font-mono text-[10px] uppercase tracking-widest text-white/40">what you will see</p>
            <h2 className="mt-3 max-w-3xl text-3xl font-semibold tracking-tight">Walkthrough coverage</h2>
            <div className="mt-10 grid gap-12 lg:grid-cols-2 lg:gap-16">
              <div className="space-y-3">
                {[
                  { tag: 'Dispatch', title: 'Dispatch Accuracy', desc: 'Visual warehouse boards with smart truck matching.', img: 0 },
                  { tag: 'Visibility', title: 'Fleet Visibility', desc: 'Live maps with planned-vs-actual routes.', img: 1 },
                  { tag: 'Finance', title: 'Payment Confidence', desc: 'One reconciled flow from checkout to treasury.', img: 2 },
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
                <h2 className="text-xl font-semibold">Book your walkthrough</h2>
                <div className="mt-6 space-y-4">
                  {[
                    { name: 'name', label: 'Full name', type: 'text', required: true, autoComplete: 'name' },
                    { name: 'email', label: 'Email', type: 'email', required: true, autoComplete: 'email' },
                    { name: 'portfolio', label: 'Company website', type: 'url', required: false, autoComplete: 'url' },
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
                      Your role *
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
                      <option>Supplier Operations</option>
                      <option>Warehouse Manager</option>
                      <option>Fleet / Dispatch Lead</option>
                      <option>IT / Platform Owner</option>
                      <option>Executive / Founder</option>
                    </select>
                    {fieldErrors.position ? (
                      <p className="text-sm text-[#FE5934]" role="alert">
                        {fieldErrors.position}
                      </p>
                    ) : null}
                  </div>
                  <div className="docs-form-field">
                    <label htmlFor="join-message" className="text-xs font-mono uppercase tracking-wider text-white/50">
                      Tell us about your network
                    </label>
                    <textarea
                      id="join-message"
                      name="message"
                      value={formData.message}
                      onChange={handleChange}
                      rows={4}
                      disabled={isSubmitting}
                      className="docs-textarea disabled:opacity-50"
                      placeholder="Sites, fleet size, dispatch volume..."
                    />
                  </div>
                </div>
                {submitStatus === 'success' && (
                  <p className="mt-4 border border-[#8DDC96]/40 bg-[#8DDC96]/15 p-3 text-center text-sm font-medium text-[#8DDC96]" role="status">
                    Demo request submitted — we&apos;ll reach out within one business day.
                  </p>
                )}
                {submitStatus === 'error' && (
                  <p className="mt-4 border border-[#FE5934]/40 bg-[#FE5934]/15 p-3 text-center text-sm text-[#FE5934]" role="alert">
                    {errorMessage}
                  </p>
                )}
                <div className="mt-6">
                  <ChamferButton type="submit" variant="fill" className="w-full justify-center" disabled={isSubmitting}>
                    {isSubmitting ? 'Submitting...' : 'Request demo'}
                  </ChamferButton>
                </div>
              </form>
              </div>
            </div>
          </section>
        <O9TourCTA />
        </>
      }
    />
  );
}
