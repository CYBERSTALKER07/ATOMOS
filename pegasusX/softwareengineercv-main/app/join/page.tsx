'use client';

import { useEffect, useRef, useState } from 'react';
import { gsap } from 'gsap';
import Link from 'next/link';
import ContentCard, { EDITORIAL_IMAGES } from '../components/ContentCard';
import { useIsMobile } from '../hooks/useDevice';
import Lanyard from '../components/Lanyard';

export default function JoinPage() {
  const { isMobile } = useIsMobile();
  const titleRef = useRef<HTMLDivElement>(null);
  const subtitleRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const formRef = useRef<HTMLDivElement>(null);
  const lanyardRef = useRef<HTMLDivElement>(null);

  const [formData, setFormData] = useState({
    name: '',
    email: '',
    position: 'Supplier Operations',
    portfolio: '',
    message: ''
  });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitStatus, setSubmitStatus] = useState<'idle' | 'success' | 'error'>('idle');
  const [errorMessage, setErrorMessage] = useState('');

  useEffect(() => {
    document.title = 'Request Demo | Pegasus';
    
    // Mobile: Skip all GSAP animations, use simple CSS fade-in
    if (isMobile) {
      gsap.set([titleRef.current, subtitleRef.current, contentRef.current, formRef.current, lanyardRef.current], {
        opacity: 1,
        x: 0,
        y: 0,
        scale: 1
      });
      return;
    }

    // Desktop: Use GSAP animations
    const timeline = gsap.timeline({ defaults: { ease: 'power3.out' } });

    timeline
      .fromTo(titleRef.current,
        { opacity: 0, y: -50 },
        { opacity: 1, y: 0, duration: 1 }
      )
      .fromTo(subtitleRef.current,
        { opacity: 0, y: 30 },
        { opacity: 1, y: 0, duration: 0.8 },
        '-=0.5'
      )
      .fromTo(contentRef.current,
        { opacity: 0, x: -50 },
        { opacity: 1, x: 0, duration: 1 },
        '-=0.4'
      )
      .fromTo(formRef.current,
        { opacity: 0, x: -50 },
        { opacity: 1, x: 0, duration: 1 },
        '-=0.6'
      )
      .fromTo(lanyardRef.current,
        { opacity: 0, scale: 0.8 },
        { opacity: 1, scale: 1, duration: 1.2 },
        '-=0.8'
      );
  }, [isMobile]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    setSubmitStatus('idle');
    setErrorMessage('');

    try {
      const response = await fetch('/api/apply', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(formData),
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error || 'Failed to submit application');
      }

      const stored = localStorage.getItem('team_applications');
      const applications = stored ? JSON.parse(stored) : [];
      applications.unshift(data.application);
      localStorage.setItem('team_applications', JSON.stringify(applications));

      setSubmitStatus('success');
      setFormData({
        name: '',
        email: '',
        position: 'Frontend Developer',
        portfolio: '',
        message: ''
      });

      // Success animation works on all devices
      gsap.fromTo('.success-message', 
        { scale: 0, opacity: 0 },
        { scale: 1, opacity: 1, duration: 0.5, ease: 'back.out' }
      );
    } catch (error) {
      setSubmitStatus('error');
      setErrorMessage(error instanceof Error ? error.message : 'Something went wrong');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) => {
    setFormData(prev => ({
      ...prev,
      [e.target.name]: e.target.value
    }));
  };

  return (
    <div className="min-h-screen bg-black text-white relative overflow-hidden">
      {/* Navigation */}
      <nav className="fixed top-8 left-8 z-50">
        <Link href="/" className="editorial-btn editorial-btn--sm">
          <span>←</span>
          <span>Back to Home</span>
        </Link>
      </nav>

      <div className="container mx-auto px-4 py-20 min-h-screen flex items-center">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-12 items-center max-w-7xl mx-auto w-full">
          {/* Left Content */}
          <div className="space-y-8 relative z-10">
            <div ref={titleRef}>
              <h1 className="text-5xl md:text-6xl lg:text-7xl font-light mb-6 text-white">
                Request a Demo
              </h1>
              <div className="w-24 h-1 bg-white mb-8" />
            </div>

            <div ref={subtitleRef}>
              <p className="text-xl md:text-2xl text-gray-300 mb-8 leading-relaxed">
                See how Pegasus runs dispatch, tracking, and payments for supplier-led logistics networks.
              </p>
            </div>

            <div ref={contentRef} className="editorial-grid grid grid-cols-1">
              <ContentCard
                variant="split"
                tone="dark"
                tag="Dispatch"
                title="Dispatch Accuracy"
                description="Visual warehouse boards with smart truck-and-order matching at peak hours."
                image={EDITORIAL_IMAGES[0]}
              />
              <ContentCard
                variant="split"
                tone="light"
                tag="Visibility"
                title="Fleet Visibility"
                description="Live maps with planned-vs-actual routes and deviation alerts before complaints."
                image={EDITORIAL_IMAGES[1]}
              />
              <ContentCard
                variant="split"
                tone="dark"
                tag="Finance"
                title="Payment Confidence"
                description="Checkout through driver collection to supplier treasury — one reconciled flow."
                image={EDITORIAL_IMAGES[2]}
              />
            </div>

            {/* Application Form */}
            <div ref={formRef} className="editorial-card editorial-card--dark border border-white/20 p-8">
              <h3 className="text-2xl font-light mb-6">Book Your Walkthrough</h3>
              
              <form onSubmit={handleSubmit} className="space-y-6">
                <div>
                  <label className="block text-sm font-semibold mb-2">Full Name *</label>
                  <input 
                    type="text"
                    name="name"
                    value={formData.name}
                    onChange={handleChange}
                    required
                    disabled={isSubmitting}
                    className="w-full px-4 py-3 bg-black border-2 border-white rounded-lg text-white focus:outline-none focus:border-[#FBFF63] transition-colors disabled:opacity-50"
                    placeholder="John Doe"
                  />
                </div>

                <div>
                  <label className="block text-sm font-semibold mb-2">Email *</label>
                  <input 
                    type="email"
                    name="email"
                    value={formData.email}
                    onChange={handleChange}
                    required
                    disabled={isSubmitting}
                    className="w-full px-4 py-3 bg-black border-2 border-white rounded-lg text-white focus:outline-none focus:border-[#FBFF63] transition-colors disabled:opacity-50"
                    placeholder="john@example.com"
                  />
                </div>

                <div>
                  <label className="block text-sm font-semibold mb-2">Your Role *</label>
                  <select 
                    name="position"
                    value={formData.position}
                    onChange={handleChange}
                    disabled={isSubmitting}
                    className="w-full px-4 py-3 bg-black border-2 border-white rounded-lg text-white focus:outline-none focus:border-[#FBFF63] transition-colors disabled:opacity-50"
                  >
                    <option>Supplier Operations</option>
                    <option>Warehouse Manager</option>
                    <option>Fleet / Dispatch Lead</option>
                    <option>IT / Platform Owner</option>
                    <option>Executive / Founder</option>
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-semibold mb-2">Company Website</label>
                  <input 
                    type="url"
                    name="portfolio"
                    value={formData.portfolio}
                    onChange={handleChange}
                    disabled={isSubmitting}
                    className="w-full px-4 py-3 bg-black border-2 border-white rounded-lg text-white focus:outline-none focus:border-[#FBFF63] transition-colors disabled:opacity-50"
                    placeholder="https://yourcompany.com"
                  />
                </div>

                <div>
                  <label className="block text-sm font-semibold mb-2">Tell us about your network</label>
                  <textarea 
                    name="message"
                    value={formData.message}
                    onChange={handleChange}
                    rows={4}
                    disabled={isSubmitting}
                    className="w-full px-4 py-3 bg-black border-2 border-white rounded-lg text-white focus:outline-none focus:border-[#FBFF63] transition-colors resize-none disabled:opacity-50"
                    placeholder="Sites, fleet size, dispatch volume..."
                  />
                </div>

                {submitStatus === 'success' && (
                  <div className="success-message bg-[#8DDC96] text-black p-4 rounded-lg font-semibold text-center">
                    ✓ Demo request submitted! Our team will reach out within one business day.
                  </div>
                )}

                {submitStatus === 'error' && (
                  <div className="bg-[#FE5934] text-white p-4 rounded-lg font-semibold text-center">
                    ✗ {errorMessage}
                  </div>
                )}

                <button
                  type="submit"
                  disabled={isSubmitting}
                  className="editorial-btn editorial-btn--full"
                >
                  {isSubmitting ? 'Submitting...' : 'Request Demo'}
                </button>
              </form>
            </div>
          </div>

          {/* Right Side - Lanyard 3D - Hidden on mobile */}
          {!isMobile && (
            <div ref={lanyardRef} className="relative h-[600px] lg:h-screen hidden lg:block">
              <div className="absolute inset-0">
                <Lanyard
                  position={[0, 0, 30]}
                  gravity={[0, -40, 0]}
                  fov={20}
                  transparent={true}
                />
              </div>
              
              {/* Decorative frame */}
              <div className="absolute inset-0 border-2 border-white rounded-2xl opacity-20 pointer-events-none" />
              <div className="absolute top-0 left-0 w-20 h-20 border-t-2 border-l-2 border-white rounded-tl-2xl" />
              <div className="absolute top-0 right-0 w-20 h-20 border-t-2 border-r-2 border-white rounded-tr-2xl" />
              <div className="absolute bottom-0 left-0 w-20 h-20 border-b-2 border-l-2 border-white rounded-bl-2xl" />
              <div className="absolute bottom-0 right-0 w-20 h-20 border-b-2 border-r-2 border-white rounded-br-2xl" />
            </div>
          )}
        </div>
      </div>

      {/* Decorative Background Elements */}
      <div className="absolute top-20 right-20 w-40 h-40 border-2 border-white rounded-2xl opacity-10 pointer-events-none" />
      <div className="absolute bottom-20 left-20 w-32 h-32 border-2 border-white rounded-2xl opacity-10 pointer-events-none" />
      <div className="absolute top-1/2 left-10 w-2 h-2 bg-white rounded-full opacity-30" />
      <div className="absolute top-1/3 right-1/4 w-2 h-2 bg-white rounded-full opacity-30" />
      <div className="absolute bottom-1/3 left-1/3 w-2 h-2 bg-white rounded-full opacity-30" />
    </div>
  );
}
