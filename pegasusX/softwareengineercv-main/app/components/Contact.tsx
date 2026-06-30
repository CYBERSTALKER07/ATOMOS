'use client';

import { useEffect, useRef, useState } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import CurvedLoop from './CurvedLoop';
import ContentCard from './ContentCard';
import { useIsMobile } from '../hooks/useDevice';

gsap.registerPlugin(ScrollTrigger);

type InquiryType = 'general' | 'client' | 'sponsor';

export default function Contact() {
  const { isMobile } = useIsMobile();
  const sectionRef = useRef<HTMLDivElement>(null);
  const titleRef = useRef<HTMLDivElement>(null);
  const tabsRef = useRef<HTMLDivElement>(null);
  const formRef = useRef<HTMLFormElement>(null);
  const infoRef = useRef<HTMLDivElement>(null);
  
  const [inquiryType, setInquiryType] = useState<InquiryType>('general');
  const tabsRefList = useRef<(HTMLButtonElement | null)[]>([]);
  const [formData, setFormData] = useState({
    name: '',
    email: '',
    company: '',
    budget: '',
    timeline: '',
    category: '',
    message: ''
  });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitStatus, setSubmitStatus] = useState<'idle' | 'success' | 'error'>('idle');

  useEffect(() => {
    if (!sectionRef.current) return;

    // Mobile: Simple fade-in
    if (isMobile) {
      gsap.set([titleRef.current, tabsRef.current, formRef.current, infoRef.current], {
        opacity: 1,
        y: 0,
        clearProps: 'transform',
      });
      return;
    }

    // Desktop: Scroll-triggered animations
    const timeline = gsap.timeline({
      scrollTrigger: {
        trigger: sectionRef.current,
        start: 'top 80%',
        end: 'bottom 20%',
        toggleActions: 'play none none reverse',
      },
      onComplete: () => {
        gsap.set([formRef.current, infoRef.current], { clearProps: 'transform' });
      },
    });

    timeline
      .fromTo(titleRef.current, { opacity: 0, y: 30 }, { opacity: 1, y: 0, duration: 0.8, clearProps: 'transform' })
      .fromTo(tabsRef.current, { opacity: 0, y: 20 }, { opacity: 1, y: 0, duration: 0.6, clearProps: 'transform' }, '-=0.4')
      .fromTo(
        [formRef.current, infoRef.current],
        { opacity: 0 },
        { opacity: 1, duration: 0.8, stagger: 0.2 },
        '-=0.4'
      );
  }, [isMobile]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
    setFormData({ ...formData, [e.target.name]: e.target.value });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    setSubmitStatus('idle');

    try {
      const response = await fetch('/api/contact', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...formData, inquiryType }),
      });
      
      const data = await response.json();
      
      if (response.ok) {
        setSubmitStatus('success');
        setFormData({ name: '', email: '', company: '', budget: '', timeline: '', category: '', message: '' });
        
        // Store in localStorage
        const stored = localStorage.getItem(`${inquiryType}_inquiries`);
        const inquiries = stored ? JSON.parse(stored) : [];
        inquiries.unshift(data);
        localStorage.setItem(`${inquiryType}_inquiries`, JSON.stringify(inquiries));
        
        // Success animation
        gsap.fromTo('.success-message', 
          { scale: 0, opacity: 0 },
          { scale: 1, opacity: 1, duration: 0.5, ease: 'back.out' }
        );
        
        setTimeout(() => setSubmitStatus('idle'), 5000);
      } else {
        setSubmitStatus('error');
      }
    } catch {
      setSubmitStatus('error');
    } finally {
      setIsSubmitting(false);
    }
  };

  const contactInfo = [
    { title: 'Email', detail: 'demo@pegasus.io', link: 'mailto:demo@pegasus.io' },
    { title: 'LinkedIn', detail: 'Follow Pegasus', link: 'https://linkedin.com' },
    { title: 'Platform', detail: 'Explore modules', link: '#projects' },
    { title: 'Sales', detail: 'Enterprise inquiries', link: 'mailto:sales@pegasus.io' }
  ];

  const tabs = [
    { id: 'general' as InquiryType, label: 'General Inquiry' },
    { id: 'client' as InquiryType, label: 'Request Demo' },
    { id: 'sponsor' as InquiryType, label: 'Partner With Us' }
  ];

  const handleKeyDown = (e: React.KeyboardEvent, index: number) => {
    const isVerticalTabList = isMobile;
    let newIndex = -1;

    if (e.key === 'Home') {
      newIndex = 0;
    } else if (e.key === 'End') {
      newIndex = tabs.length - 1;
    } else if (isVerticalTabList && e.key === 'ArrowDown') {
      newIndex = (index + 1) % tabs.length;
    } else if (isVerticalTabList && e.key === 'ArrowUp') {
      newIndex = (index - 1 + tabs.length) % tabs.length;
    } else if (!isVerticalTabList && e.key === 'ArrowRight') {
      newIndex = (index + 1) % tabs.length;
    } else if (!isVerticalTabList && e.key === 'ArrowLeft') {
      newIndex = (index - 1 + tabs.length) % tabs.length;
    }

    if (newIndex !== -1) {
      e.preventDefault();
      setInquiryType(tabs[newIndex].id);
      tabsRefList.current[newIndex]?.focus();
    }
  };

  return (
    <section 
      ref={sectionRef} 
      id="contact" 
      className="py-20 md:py-28 bg-white text-black relative overflow-x-hidden"
    >
      {/* Corner Curved Loops - Hidden on mobile */}
      {!isMobile && (
        <>
          <div className="absolute top-0 left-0 w-64 md:w-80 h-20 md:h-24 pointer-events-none opacity-20 z-10">
            <CurvedLoop marqueeText="CONNECT ✦ " speed={1.8} curveAmount={200} direction="right" interactive={false} className="fill-black/30" />
          </div>
          <div className="absolute top-0 right-0 w-64 md:w-80 h-20 md:h-24 pointer-events-none opacity-20 z-10 scale-x-[-1]">
            <CurvedLoop marqueeText="GET IN TOUCH ✦ " speed={1.8} curveAmount={200} direction="left" interactive={false} className="fill-black/30" />
          </div>
        </>
      )}

      <div className="container mx-auto px-4 relative z-20">
        <div ref={titleRef} className="text-center mb-16">
          <h2 className="text-4xl md:text-5xl lg:text-6xl font-light mb-4 text-black">
            Let&apos;s Talk Logistics
          </h2>
          <div className="w-20 h-1 bg-black rounded-full mx-auto mb-6" />
          <p className="text-lg md:text-xl text-black max-w-2xl mx-auto">
            Tell us how you run dispatch today — we&apos;ll show you Pegasus
          </p>
        </div>

        {/* Tabs */}
        <div ref={tabsRef} className="max-w-4xl mx-auto mb-12">
          <div className="flex flex-col sm:flex-row gap-4 justify-center" role="tablist" aria-label="Inquiry types">
            {tabs.map((tab, index) => (
              <button
                key={tab.id}
                ref={(el) => { tabsRefList.current[index] = el; }}
                role="tab"
                aria-selected={inquiryType === tab.id}
                aria-controls="contact-form"
                id={`tab-${tab.id}`}
                tabIndex={inquiryType === tab.id ? 0 : -1}
                onClick={() => setInquiryType(tab.id)}
                onKeyDown={(e) => handleKeyDown(e, index)}
                className={`editorial-btn editorial-btn--sm editorial-btn--on-light ${
                  inquiryType === tab.id ? 'editorial-btn--active' : ''
                }`}
              >
                {tab.label}
              </button>
            ))}
          </div>
        </div>

        <div className="max-w-7xl mx-auto">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 lg:gap-10 lg:items-start">
            {/* Left: Contact Info */}
            <div ref={infoRef} className="space-y-8 min-w-0">
              <div className="space-y-6">
                <h3 className="text-2xl md:text-3xl font-light text-black">
                  Get in Touch
                </h3>
                
                <div className="editorial-grid editorial-grid--light grid grid-cols-1 sm:grid-cols-2">
                  {contactInfo.map((info, index) => (
                    <ContentCard
                      key={index}
                      variant="vertical"
                      tone={index % 2 === 0 ? 'dark' : 'light'}
                      tag={info.title}
                      title={info.detail}
                      href={info.link}
                      ctaLabel="CONTACT"
                      ctaStyle="link"
                      className="min-h-[14rem]"
                    />
                  ))}
                </div>
              </div>

              {/* Benefits Card */}
              <div className="editorial-card editorial-card--dark border border-black p-8 text-white relative">
                <h4 className="text-xl font-light mb-4">Why Pegasus?</h4>
                <ul className="space-y-3">
                  <li className="flex items-start gap-3">
                    <span className="mt-1">✓</span>
                    <span>Built for supplier-led logistics networks</span>
                  </li>
                  <li className="flex items-start gap-3">
                    <span className="mt-1">✓</span>
                    <span>Response within one business day</span>
                  </li>
                  <li className="flex items-start gap-3">
                    <span className="mt-1">✓</span>
                    <span>Six role apps on one platform</span>
                  </li>
                  <li className="flex items-start gap-3">
                    <span className="mt-1">✓</span>
                    <span>Guided rollout for your network</span>
                  </li>
                </ul>
              </div>
            </div>

            {/* Right: Dynamic Form */}
            <div className="min-w-0">
              <form 
                ref={formRef}
                id="contact-form"
                role="tabpanel"
                aria-labelledby={`tab-${inquiryType}`}
                onSubmit={handleSubmit} 
                className="border border-black p-8 md:p-10 bg-white"
              >
                <h3 className="text-2xl md:text-3xl font-light text-black mb-8">
                  {inquiryType === 'general' && 'Send a Message'}
                  {inquiryType === 'client' && 'Request a Demo'}
                  {inquiryType === 'sponsor' && 'Partnership Inquiry'}
                </h3>

                <div className="space-y-6">
                  {/* Name Field */}
                  <div>
                    <label htmlFor="name" className="block text-sm font-light mb-3 text-black">
                      YOUR NAME *
                    </label>
                    <input
                      type="text"
                      id="name"
                      name="name"
                      value={formData.name}
                      onChange={handleChange}
                      className="w-full px-4 py-3 border-2 border-black bg-white text-black focus:outline-none focus:border-black transition-colors rounded-lg"
                      placeholder="John Doe"
                      required
                      disabled={isSubmitting}
                    />
                  </div>

                  {/* Email Field */}
                  <div>
                    <label htmlFor="email" className="block text-sm font-light mb-3 text-black">
                      YOUR EMAIL *
                    </label>
                    <input
                      type="email"
                      id="email"
                      name="email"
                      value={formData.email}
                      onChange={handleChange}
                      className="w-full px-4 py-3 border-2 border-black bg-white text-black focus:outline-none focus:border-black transition-colors rounded-lg"
                      placeholder="john@example.com"
                      required
                      disabled={isSubmitting}
                    />
                  </div>

                  {/* Client/Sponsor Specific Fields */}
                  {(inquiryType === 'client' || inquiryType === 'sponsor') && (
                    <>
                      <div>
                        <label htmlFor="company" className="block text-sm font-light mb-3 text-black">
                          COMPANY NAME
                        </label>
                        <input
                          type="text"
                          id="company"
                          name="company"
                          value={formData.company}
                          onChange={handleChange}
                          className="w-full px-4 py-3 border-2 border-black bg-white text-black focus:outline-none focus:border-black transition-colors rounded-lg"
                          placeholder="Your Company"
                          disabled={isSubmitting}
                        />
                      </div>

                      {inquiryType === 'client' && (
                        <>
                          <div>
                            <label htmlFor="budget" className="block text-sm font-light mb-3 text-black">
                              NETWORK SIZE
                            </label>
                            <select
                              id="budget"
                              name="budget"
                              value={formData.budget}
                              onChange={handleChange}
                              className="w-full px-4 py-3 border-2 border-black bg-white text-black focus:outline-none focus:border-black transition-colors rounded-lg"
                              disabled={isSubmitting}
                            >
                              <option value="">Select network size</option>
                              <option value="5k-10k">1–3 sites</option>
                              <option value="10k-25k">4–10 sites</option>
                              <option value="25k-50k">11–25 sites</option>
                              <option value="50k+">25+ sites</option>
                            </select>
                          </div>

                          <div>
                            <label htmlFor="timeline" className="block text-sm font-light mb-3 text-black">
                              ROLLOUT TIMELINE
                            </label>
                            <select
                              id="timeline"
                              name="timeline"
                              value={formData.timeline}
                              onChange={handleChange}
                              className="w-full px-4 py-3 border-2 border-black bg-white text-black focus:outline-none focus:border-black transition-colors rounded-lg"
                              disabled={isSubmitting}
                            >
                              <option value="">Select timeline</option>
                              <option value="urgent">Pilot (2–4 weeks)</option>
                              <option value="short">Phase 1 (1–3 months)</option>
                              <option value="medium">Full rollout (3–6 months)</option>
                              <option value="long">Enterprise (6+ months)</option>
                            </select>
                          </div>
                        </>
                      )}

                      {inquiryType === 'sponsor' && (
                        <div>
                          <label htmlFor="category" className="block text-sm font-light mb-3 text-black">
                            PARTNERSHIP TYPE
                          </label>
                          <select
                            id="category"
                            name="category"
                            value={formData.category}
                            onChange={handleChange}
                            className="w-full px-4 py-3 border-2 border-black bg-white text-black focus:outline-none focus:border-black transition-colors rounded-lg"
                            disabled={isSubmitting}
                          >
                            <option value="">Select partnership type</option>
                            <option value="project">Technology Partner</option>
                            <option value="content">Integration Partner</option>
                            <option value="event">Regional Operator</option>
                            <option value="longterm">Strategic Alliance</option>
                          </select>
                        </div>
                      )}
                    </>
                  )}

                  {/* Message Field */}
                  <div>
                    <label htmlFor="message" className="block text-sm font-light mb-3 text-black">
                      YOUR MESSAGE *
                    </label>
                    <textarea
                      id="message"
                      name="message"
                      value={formData.message}
                      onChange={handleChange}
                      rows={6}
                      className="w-full px-4 py-3 border-2 border-black bg-white text-black focus:outline-none focus:border-black transition-colors resize-none rounded-lg"
                      placeholder={
                        inquiryType === 'general' ? 'Tell us about your logistics network...' :
                        inquiryType === 'client' ? 'Describe your dispatch and fleet operations...' :
                        'Tell us about your partnership goals...'
                      }
                      required
                      disabled={isSubmitting}
                    />
                  </div>

                  {/* Submit Button */}
                  <button
                    type="submit"
                    disabled={isSubmitting}
                    className="editorial-btn editorial-btn--full editorial-btn--on-light"
                  >
                    {isSubmitting ? 'SUBMITTING...' : 'SUBMIT INQUIRY →'}
                  </button>

                  {/* Status Messages */}
                  {submitStatus === 'success' && (
                    <div className="success-message p-4 border-2 border-black bg-[#8DDC96] text-black rounded-lg" role="status">
                      <p className="font-light">✓ Inquiry submitted successfully!</p>
                      <p className="text-sm mt-1">Our team will respond within one business day.</p>
                    </div>
                  )}

                  {submitStatus === 'error' && (
                    <div className="p-4 border-2 border-black bg-[#FE5934] text-white rounded-lg" role="status">
                      <p className="font-light">✗ Failed to submit inquiry</p>
                      <p className="text-sm mt-1">Please try again or email demo@pegasus.io directly.</p>
                    </div>
                  )}
                </div>
              </form>
            </div>
          </div>
        </div>

        {/* Bottom CTA */}
        <div className="relative z-10 text-center mt-16 pt-10 border-t-2 border-black">
          <p className="text-lg md:text-xl text-black mb-6">
            Prefer a live walkthrough? Book a demo call
          </p>
          <a
            href="https://calendly.com"
            target="_blank"
            rel="noreferrer noopener"
            className="editorial-btn editorial-btn--on-light"
          >
            BOOK A DEMO
          </a>
        </div>
      </div>
    </section>
  );
}
