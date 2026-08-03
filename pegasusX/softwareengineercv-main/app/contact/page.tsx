'use client';

import { useEffect, useRef, useState } from 'react';
import { gsap } from 'gsap';
import Link from 'next/link';
import { Mail, Zap, Clock, Globe } from 'lucide-react';

export default function ContactPage() {
  const headerRef = useRef<HTMLDivElement>(null);
  const formRef = useRef<HTMLDivElement>(null);
  const infoRef = useRef<HTMLDivElement>(null);

  const [formData, setFormData] = useState({
    name: '',
    email: '',
    subject: '',
    message: ''
  });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitStatus, setSubmitStatus] = useState<'idle' | 'success' | 'error'>('idle');
  const [errorMessage, setErrorMessage] = useState('');

  useEffect(() => {
    document.title = "Let's Connect | Get in Touch";

    const timeline = gsap.timeline({ defaults: { ease: 'power3.out' } });

    timeline
      .fromTo(headerRef.current,
        { opacity: 0, y: -50 },
        { opacity: 1, y: 0, duration: 1 }
      )
      .fromTo(formRef.current,
        { opacity: 0, x: -50 },
        { opacity: 1, x: 0, duration: 1 },
        '-=0.5'
      )
      .fromTo(infoRef.current,
        { opacity: 0, x: 50 },
        { opacity: 1, x: 0, duration: 1 },
        '-=0.8'
      );
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    setSubmitStatus('idle');
    setErrorMessage('');

    try {
      const response = await fetch('/api/contact', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(formData),
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error || 'Failed to send message');
      }

      // Save to localStorage for admin dashboard
      const stored = localStorage.getItem('customer_messages');
      const messages = stored ? JSON.parse(stored) : [];
      messages.unshift(data.message);
      localStorage.setItem('customer_messages', JSON.stringify(messages));

      setSubmitStatus('success');
      setFormData({
        name: '',
        email: '',
        subject: '',
        message: ''
      });

      // Animate success message
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

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
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

      <div className="container mx-auto px-4 py-20">
        {/* Header */}
        <div ref={headerRef} className="text-center max-w-4xl mx-auto mb-16 pt-16">
          <h1 className="text-5xl md:text-6xl lg:text-7xl font-light mb-6">
            Let&apos;s Connect
          </h1>
          <div className="w-24 h-1 bg-white mx-auto mb-8" />
          <p className="text-xl md:text-2xl text-gray-300 leading-relaxed">
            Have a project in mind? Let&apos;s discuss how we can work together to bring your ideas to life.
          </p>
        </div>

        <div className="max-w-7xl mx-auto">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-12">
            {/* Contact Form */}
            <div ref={formRef}>
              <div className="border border-white/10 rounded-none p-8 bg-[#111]">
                <h2 className="text-3xl font-light mb-6">Send a Message</h2>

                <form onSubmit={handleSubmit} className="space-y-6">
                  <div>
                    <label className="block text-sm font-semibold mb-2">Your Name *</label>
                    <input
                      type="text"
                      name="name"
                      value={formData.name}
                      onChange={handleChange}
                      required
                      disabled={isSubmitting}
                      className="w-full px-4 py-3 bg-black border border-white/10 rounded-none text-white focus:outline-none focus:border-white transition-colors disabled:opacity-50"
                      placeholder="John Doe"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-semibold mb-2">Email Address *</label>
                    <input
                      type="email"
                      name="email"
                      value={formData.email}
                      onChange={handleChange}
                      required
                      disabled={isSubmitting}
                      className="w-full px-4 py-3 bg-black border border-white/10 rounded-none text-white focus:outline-none focus:border-white transition-colors disabled:opacity-50"
                      placeholder="john@example.com"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-semibold mb-2">Subject *</label>
                    <input
                      type="text"
                      name="subject"
                      value={formData.subject}
                      onChange={handleChange}
                      required
                      disabled={isSubmitting}
                      className="w-full px-4 py-3 bg-black border border-white/10 rounded-none text-white focus:outline-none focus:border-white transition-colors disabled:opacity-50"
                      placeholder="Project Inquiry"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-semibold mb-2">Message *</label>
                    <textarea
                      name="message"
                      value={formData.message}
                      onChange={handleChange}
                      required
                      rows={6}
                      disabled={isSubmitting}
                      className="w-full px-4 py-3 bg-black border border-white/10 rounded-none text-white focus:outline-none focus:border-white transition-colors resize-none disabled:opacity-50"
                      placeholder="Tell us about your project..."
                    />
                  </div>

                  {submitStatus === 'success' && (
                    <div className="success-message bg-[#8DDC96] text-black p-4 rounded-none font-semibold text-center" role="status">
                      ✓ Message sent successfully! We&apos;ll get back to you soon.
                    </div>
                  )}

                  {submitStatus === 'error' && (
                    <div className="bg-[#FE5934] text-white p-4 rounded-none font-semibold text-center" role="alert" aria-atomic="true">
                      ✗ {errorMessage}
                    </div>
                  )}

                  <button
                    type="submit"
                    disabled={isSubmitting}
                    className="editorial-btn editorial-btn--full"
                  >
                    {isSubmitting ? 'Sending...' : 'Send Message'}
                  </button>
                </form>
              </div>
            </div>

            {/* Contact Info */}
            <div ref={infoRef} className="space-y-6">
              {/* Get in Touch */}
              <div className="group border border-white/10 rounded-none p-8 bg-[#111] hover:bg-[#222] transition-colors">
                <Mail className="w-8 h-8 mb-4 text-white" />
                <h3 className="text-2xl font-light mb-3">Email</h3>
                <div className="space-y-2">
                  <a
                    href="mailto:shsoliyev@aut-edu.uz"
                    className="block text-[#A9EBF9] hover:text-white transition-colors text-lg"
                  >
                  </a>
                  <a
                    href="mailto:cyberstalkerx7@gmail.com"
                    className="block text-[#A9EBF9] hover:text-white transition-colors text-lg"
                  >
                    cyberstalkerx7@gmail.com
                  </a>
                </div>
              </div>

              {/* Response Time */}
              <div className="group border border-white/10 rounded-none p-8 bg-[#111] hover:bg-[#222] transition-colors">
                <Zap className="w-8 h-8 mb-4 text-white" />
                <h3 className="text-2xl font-light mb-3">Response Time</h3>
                <p className="text-gray-300 text-lg">
                  We typically respond within 24-48 hours during business days.
                </p>
              </div>

              {/* Office Hours */}
              <div className="group border border-white/10 rounded-none p-8 bg-[#111] hover:bg-[#222] transition-colors">
                <Clock className="w-8 h-8 mb-4 text-white" />
                <h3 className="text-2xl font-light mb-3">Office Hours</h3>
                <p className="text-gray-300 text-lg">
                  Monday - Friday: 9:00 AM - 6:00 PM<br />
                  Weekend: By appointment
                </p>
              </div>

              {/* Social Links */}
              <div className="group border border-white/10 rounded-none p-8 bg-[#111] hover:bg-[#222] transition-colors">
                <Globe className="w-8 h-8 mb-4 text-white" />
                <h3 className="text-2xl font-light mb-4">Connect Online</h3>
                <div className="flex flex-wrap gap-3">
                  <a href="#" className="editorial-btn editorial-btn--sm">LinkedIn</a>
                  <a href="#" className="editorial-btn editorial-btn--sm">Twitter</a>
                  <a href="#" className="editorial-btn editorial-btn--sm">GitHub</a>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Bottom CTA */}
        <div className="max-w-7xl mx-auto mt-16">
          <div className="border border-white/10 rounded-none p-12 text-center bg-[#111]">
            <h2 className="text-4xl font-light mb-4">Ready to Start Your Project?</h2>
            <p className="text-xl text-gray-300 mb-8 max-w-2xl mx-auto">
              Let&apos;s turn your vision into reality. Reach out today and let&apos;s discuss how we can help.
            </p>
            <div className="flex flex-wrap gap-4 justify-center">
              <Link href="/join" className="editorial-btn">
                Request Demo
              </Link>
              <Link href="/projects" className="editorial-btn">
                View Our Work
              </Link>
            </div>
          </div>
        </div>
      </div>

      {/* Decorative Elements */}
      <div className="absolute top-32 right-20 w-40 h-40 border border-white/10 opacity-50 pointer-events-none rounded-none" />
      <div className="absolute bottom-32 left-20 w-32 h-32 border border-white/10 opacity-50 pointer-events-none rounded-none" />
      <div className="absolute top-1/2 left-10 w-2 h-2 bg-white rounded-full opacity-30" />
      <div className="absolute top-1/3 right-1/4 w-2 h-2 bg-white rounded-full opacity-30" />
      <div className="absolute bottom-1/3 left-1/3 w-2 h-2 bg-white rounded-full opacity-30" />
    </div>
  );
}