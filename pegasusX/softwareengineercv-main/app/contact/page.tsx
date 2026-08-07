'use client';

import { useEffect, useState } from 'react';
import { Mail, Zap, Clock, Send } from 'lucide-react';
import ChamferButton from '@/app/components/ChamferButton';
import FleekSecondaryLayout from '@/app/components/fleek/FleekSecondaryLayout';

export default function ContactPage() {
  const [formData, setFormData] = useState({
    name: '',
    email: '',
    subject: '',
    message: '',
  });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitStatus, setSubmitStatus] = useState<'idle' | 'success' | 'error'>('idle');
  const [errorMessage, setErrorMessage] = useState('');
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    document.title = "Let's Connect | Pegasus";
  }, []);

  const validate = () => {
    const next: Record<string, string> = {};
    if (!formData.name.trim()) next.name = 'Enter your name.';
    if (!formData.email.trim()) next.email = 'Enter your email.';
    else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) next.email = 'Enter a valid email.';
    if (!formData.subject.trim()) next.subject = 'Add a subject.';
    if (!formData.message.trim()) next.message = 'Write a short message.';
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
      const response = await fetch('/api/contact', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(formData),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || 'Failed to send message');

      const stored = localStorage.getItem('customer_messages');
      const messages = stored ? JSON.parse(stored) : [];
      messages.unshift(data.message);
      localStorage.setItem('customer_messages', JSON.stringify(messages));

      setSubmitStatus('success');
      setFormData({ name: '', email: '', subject: '', message: '' });
      setFieldErrors({});
    } catch (error) {
      setSubmitStatus('error');
      setErrorMessage(error instanceof Error ? error.message : 'Something went wrong');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
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

  const fieldMeta = {
    name: { label: 'Your name', type: 'text' as const, autoComplete: 'name' },
    email: { label: 'Email', type: 'email' as const, autoComplete: 'email' },
    subject: { label: 'Subject', type: 'text' as const, autoComplete: 'off' },
  };

  return (
    <FleekSecondaryLayout
      activeHref="/contact"
      sectionTitle="CONTACT"
      title="Let's Connect"
      summary="Questions about Pegasus, partnerships, or your logistics network — message us via email or Telegram @DominusMunerum."
      primaryHref="https://t.me/DominusMunerum"
      primaryLabel="CHAT ON TELEGRAM"
      secondaryHref="/platform"
      secondaryLabel="EXPLORE PLATFORM"
      section06={
        <>
          <section className="docs-section">
            <h2 className="text-3xl font-semibold tracking-tight">Send a message</h2>
            <div className="mt-10 grid gap-10 lg:grid-cols-5">
              <div className="lg:col-span-3">
                <form onSubmit={handleSubmit} className="docs-surface docs-grain space-y-5 p-6 md:p-8" noValidate>
                  {(['name', 'email', 'subject'] as const).map((field) => (
                    <div key={field} className="docs-form-field">
                      <label
                        htmlFor={`contact-${field}`}
                        className="text-xs font-mono uppercase tracking-wider text-white/50"
                      >
                        {fieldMeta[field].label} *
                      </label>
                      <input
                        id={`contact-${field}`}
                        type={fieldMeta[field].type}
                        name={field}
                        value={formData[field]}
                        onChange={handleChange}
                        required
                        disabled={isSubmitting}
                        autoComplete={fieldMeta[field].autoComplete}
                        aria-invalid={Boolean(fieldErrors[field])}
                        className="docs-input disabled:opacity-50"
                      />
                      {fieldErrors[field] ? (
                        <p className="text-sm text-[#FE5934]" role="alert">{fieldErrors[field]}</p>
                      ) : null}
                    </div>
                  ))}
                  <div className="docs-form-field">
                    <label htmlFor="contact-message" className="text-xs font-mono uppercase tracking-wider text-white/50">
                      Message *
                    </label>
                    <textarea
                      id="contact-message"
                      name="message"
                      value={formData.message}
                      onChange={handleChange}
                      required
                      rows={5}
                      disabled={isSubmitting}
                      className="docs-textarea disabled:opacity-50"
                    />
                  </div>
                  {submitStatus === 'success' && (
                    <p className="border border-[#8DDC96]/40 bg-[#8DDC96]/15 p-3 text-center text-sm text-[#8DDC96]" role="status">
                      Message sent via Resend API — we&apos;ll be in touch soon.
                    </p>
                  )}
                  {submitStatus === 'error' && (
                    <p className="border border-[#FE5934]/40 bg-[#FE5934]/15 p-3 text-center text-sm text-[#FE5934]" role="alert">
                      {errorMessage}
                    </p>
                  )}
                  <ChamferButton type="submit" variant="fill" className="w-full justify-center" disabled={isSubmitting}>
                    {isSubmitting ? 'Sending via Resend...' : 'Send Message'}
                  </ChamferButton>
                </form>
              </div>
              <div className="space-y-4 lg:col-span-2">
                {[
                  { icon: Send, title: 'Telegram', body: '@DominusMunerum', href: 'https://t.me/DominusMunerum', external: true },
                  { icon: Mail, title: 'Email', body: 'cyberstalkerx7@gmail.com', href: 'mailto:cyberstalkerx7@gmail.com' },
                  { icon: Zap, title: 'Response time', body: 'Instant on Telegram, < 24h on Email' },
                  { icon: Clock, title: 'Office hours', body: 'Mon–Fri 9:00–18:00' },
                ].map(({ icon: Icon, title, body, href, external }) => (
                  <div key={title} className="docs-card p-6">
                    <Icon className="mb-3 h-6 w-6 text-white/80" />
                    <h3 className="font-semibold">{title}</h3>
                    {href ? (
                      <a 
                        href={href} 
                        target={external ? '_blank' : undefined} 
                        rel={external ? 'noopener noreferrer' : undefined} 
                        className="mt-2 block text-sm text-white/60 hover:text-white"
                      >
                        {body} {external ? '↗' : ''}
                      </a>
                    ) : (
                      <p className="mt-2 text-sm text-white/60">{body}</p>
                    )}
                  </div>
                ))}
              </div>
            </div>
          </section>
        </>
      }
    />
  );
}
