'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import Image from 'next/image';
import Link from 'next/link';
import { FiArrowLeft, FiMonitor, FiCode, FiZap, FiLayers } from 'react-icons/fi';

gsap.registerPlugin(ScrollTrigger);

export default function WebAppsPage() {
  const heroRef = useRef<HTMLDivElement>(null);
  const titleRef = useRef<HTMLHeadingElement>(null);
  const subtitleRef = useRef<HTMLParagraphElement>(null);
  const app1Ref = useRef<HTMLDivElement>(null);
  const app2Ref = useRef<HTMLDivElement>(null);
  const app3Ref = useRef<HTMLDivElement>(null);
  const app4Ref = useRef<HTMLDivElement>(null);
  const app5Ref = useRef<HTMLDivElement>(null);
  const featuresRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const ctx = gsap.context(() => {
      // Hero animations
      gsap.from(titleRef.current, {
        y: 100,
        opacity: 0,
        duration: 1,
        ease: 'power4.out',
      });

      gsap.from(subtitleRef.current, {
        y: 50,
        opacity: 0,
        duration: 1,
        delay: 0.2,
        ease: 'power4.out',
      });

      // App card animations
      const appRefs = [app1Ref, app2Ref, app3Ref, app4Ref, app5Ref];
      
      appRefs.forEach((ref, index) => {
        gsap.from(ref.current, {
          scrollTrigger: {
            trigger: ref.current,
            start: 'top 80%',
            toggleActions: 'play none none reverse',
          },
          x: index % 2 === 0 ? -100 : 100,
          opacity: 0,
          duration: 1,
          ease: 'power3.out',
        });
      });

      // Features section animation
      gsap.from(featuresRef.current?.children || [], {
        scrollTrigger: {
          trigger: featuresRef.current,
          start: 'top 80%',
          toggleActions: 'play none none reverse',
        },
        y: 50,
        opacity: 0,
        duration: 0.8,
        stagger: 0.2,
        ease: 'power3.out',
      });
    });

    return () => ctx.revert();
  }, []);

  return (
    <div className="min-h-screen bg-black text-white">
      {/* Back Button */}
      <Link
        href="/"
        className="fixed top-8 left-8 z-50 px-6 py-3 bg-black text-white border-2 border-white hover:bg-white hover:text-black transition-all duration-300 font-bold rounded-3xl flex items-center gap-2"
      >
        <FiArrowLeft size={20} />
        Back
      </Link>

      {/* Hero Section */}
      <section ref={heroRef} className="min-h-screen flex items-center justify-center px-4 md:px-8 pt-24 md:pt-0">
        <div className="max-w-6xl mx-auto text-center">
          <h1
            ref={titleRef}
            className="text-5xl md:text-7xl lg:text-8xl font-bold mb-8"
          >
            Web Applications
          </h1>
          <p
            ref={subtitleRef}
            className="text-xl md:text-2xl text-[#C0C0C0] max-w-3xl mx-auto"
          >
            Building powerful and scalable web applications with modern technologies and best practices
          </p>
        </div>
      </section>

      {/* Apps Showcase */}
      <section className="py-20 px-4 md:px-8">
        <div className="max-w-7xl mx-auto space-y-[30px]">
          
          {/* App 1 */}
          <div
            ref={app1Ref}
            className="bg-[#0D0D0D] rounded-3xl p-8 md:p-12 border-2 border-white hover:border-[#A9EBF9] transition-all duration-300 group"
          >
            <div className="grid md:grid-cols-2 gap-12 items-center">
              <div className="order-2 md:order-1">
                <div className="flex items-center gap-4 mb-6">
                  <FiMonitor size={32} className="text-[#A9EBF9]" />
                  <h2 className="text-3xl md:text-4xl font-bold">E-Commerce Platform</h2>
                </div>
                <p className="text-lg text-[#C0C0C0] mb-6">
                  A fully-featured e-commerce web application with shopping cart, payment integration,
                  user authentication, and real-time inventory management. Built for scalability and performance.
                </p>
                <div className="flex flex-wrap gap-4">
                  <span className="px-4 py-2 bg-black border-2 border-[#A9EBF9] text-[#A9EBF9] rounded-2xl text-sm font-bold">
                    Next.js
                  </span>
                  <span className="px-4 py-2 bg-black border-2 border-[#8DDC96] text-[#8DDC96] rounded-2xl text-sm font-bold">
                    React
                  </span>
                  <span className="px-4 py-2 bg-black border-2 border-[#DABDFF] text-[#DABDFF] rounded-2xl text-sm font-bold">
                    TypeScript
                  </span>
                </div>
              </div>
              <div className="order-1 md:order-2">
                <div className="relative rounded-3xl overflow-hidden border-2 border-white group-hover:border-[#A9EBF9] transition-all duration-300">
                  <Image
                    src="/original-1ec76e0e43bf8113cf940c5b4cdaed7a.webp"
                    alt="E-Commerce Web Application"
                    width={800}
                    height={600}
                    className="w-full h-auto object-cover"
                  />
                </div>
              </div>
            </div>
          </div>

          {/* App 2 */}
          <div
            ref={app2Ref}
            className="bg-[#0D0D0D] rounded-3xl p-8 md:p-12 border-2 border-white hover:border-[#8DDC96] transition-all duration-300 group"
          >
            <div className="grid md:grid-cols-2 gap-12 items-center">
              <div className="order-2 md:order-2">
                <div className="flex items-center gap-4 mb-6">
                  <FiCode size={32} className="text-[#8DDC96]" />
                  <h2 className="text-3xl md:text-4xl font-bold">Dashboard Analytics</h2>
                </div>
                <p className="text-lg text-[#C0C0C0] mb-6">
                  Advanced analytics dashboard with real-time data visualization, interactive charts,
                  and comprehensive reporting tools. Designed for data-driven decision making.
                </p>
                <div className="flex flex-wrap gap-4">
                  <span className="px-4 py-2 bg-black border-2 border-[#8DDC96] text-[#8DDC96] rounded-2xl text-sm font-bold">
                    Data Visualization
                  </span>
                  <span className="px-4 py-2 bg-black border-2 border-[#FFDA6F] text-[#FFDA6F] rounded-2xl text-sm font-bold">
                    Real-Time
                  </span>
                  <span className="px-4 py-2 bg-black border-2 border-[#FE5934] text-[#FE5934] rounded-2xl text-sm font-bold">
                    API Integration
                  </span>
                </div>
              </div>
              <div className="order-1 md:order-1">
                <div className="relative rounded-3xl overflow-hidden border-2 border-white group-hover:border-[#8DDC96] transition-all duration-300">
                  <Image
                    src="/original-6ba13d16f8042b23a4c50a692bb23176.webp"
                    alt="Dashboard Analytics Application"
                    width={800}
                    height={600}
                    className="w-full h-auto object-cover"
                  />
                </div>
              </div>
            </div>
          </div>

          {/* App 3 */}
          <div
            ref={app3Ref}
            className="bg-[#0D0D0D] rounded-3xl p-8 md:p-12 border-2 border-white hover:border-[#DABDFF] transition-all duration-300 group"
          >
            <div className="grid md:grid-cols-2 gap-12 items-center">
              <div className="order-2 md:order-1">
                <div className="flex items-center gap-4 mb-6">
                  <FiLayers size={32} className="text-[#DABDFF]" />
                  <h2 className="text-3xl md:text-4xl font-bold">Content Management System</h2>
                </div>
                <p className="text-lg text-[#C0C0C0] mb-6">
                  Powerful CMS with intuitive interface for content creation, media management,
                  and multi-user collaboration. Built with flexibility and ease of use in mind.
                </p>
                <div className="flex flex-wrap gap-4">
                  <span className="px-4 py-2 bg-black border-2 border-[#DABDFF] text-[#DABDFF] rounded-2xl text-sm font-bold">
                    CMS
                  </span>
                  <span className="px-4 py-2 bg-black border-2 border-[#BDE7FF] text-[#BDE7FF] rounded-2xl text-sm font-bold">
                    Headless
                  </span>
                  <span className="px-4 py-2 bg-black border-2 border-[#A9EBF9] text-[#A9EBF9] rounded-2xl text-sm font-bold">
                    Cloud Storage
                  </span>
                </div>
              </div>
              <div className="order-1 md:order-2">
                <div className="relative rounded-3xl overflow-hidden border-2 border-white group-hover:border-[#DABDFF] transition-all duration-300">
                  <Image
                    src="/original-7b36f739e822f2c737563ad11d6c58df.webp"
                    alt="Content Management System"
                    width={800}
                    height={600}
                    className="w-full h-auto object-cover"
                  />
                </div>
              </div>
            </div>
          </div>

          {/* App 4 */}
          <div
            ref={app4Ref}
            className="bg-[#0D0D0D] rounded-3xl p-8 md:p-12 border-2 border-white hover:border-[#FFDA6F] transition-all duration-300 group"
          >
            <div className="grid md:grid-cols-2 gap-12 items-center">
              <div className="order-2 md:order-2">
                <div className="flex items-center gap-4 mb-6">
                  <FiZap size={32} className="text-[#FFDA6F]" />
                  <h2 className="text-3xl md:text-4xl font-bold">Project Management Tool</h2>
                </div>
                <p className="text-lg text-[#C0C0C0] mb-6">
                  Comprehensive project management application with task tracking, team collaboration,
                  time management, and progress reporting features for enhanced productivity.
                </p>
                <div className="flex flex-wrap gap-4">
                  <span className="px-4 py-2 bg-black border-2 border-[#FFDA6F] text-[#FFDA6F] rounded-2xl text-sm font-bold">
                    Collaboration
                  </span>
                  <span className="px-4 py-2 bg-black border-2 border-[#FFA500] text-[#FFA500] rounded-2xl text-sm font-bold">
                    Task Management
                  </span>
                  <span className="px-4 py-2 bg-black border-2 border-[#FBFF63] text-[#FBFF63] rounded-2xl text-sm font-bold">
                    Agile
                  </span>
                </div>
              </div>
              <div className="order-1 md:order-1">
                <div className="relative rounded-3xl overflow-hidden border-2 border-white group-hover:border-[#FFDA6F] transition-all duration-300">
                  <Image
                    src="/original-7b36f739e822f2c737563ad11d6c58df.webp"
                    alt="Project Management Application"
                    width={800}
                    height={600}
                    className="w-full h-auto object-cover"
                  />
                </div>
              </div>
            </div>
          </div>

          {/* App 5 */}
          <div
            ref={app5Ref}
            className="bg-[#0D0D0D] rounded-3xl p-8 md:p-12 border-2 border-white hover:border-[#FE5934] transition-all duration-300 group"
          >
            <div className="grid md:grid-cols-2 gap-12 items-center">
              <div className="order-2 md:order-1">
                <div className="flex items-center gap-4 mb-6">
                  <FiMonitor size={32} className="text-[#FE5934]" />
                  <h2 className="text-3xl md:text-4xl font-bold">Social Network Platform</h2>
                </div>
                <p className="text-lg text-[#C0C0C0] mb-6">
                  Modern social networking platform with user profiles, real-time messaging,
                  feed algorithms, and engagement features for building connected communities.
                </p>
                <div className="flex flex-wrap gap-4">
                  <span className="px-4 py-2 bg-black border-2 border-[#FE5934] text-[#FE5934] rounded-2xl text-sm font-bold">
                    Social Media
                  </span>
                  <span className="px-4 py-2 bg-black border-2 border-[#FFA500] text-[#FFA500] rounded-2xl text-sm font-bold">
                    WebSocket
                  </span>
                  <span className="px-4 py-2 bg-black border-2 border-[#A9EBF9] text-[#A9EBF9] rounded-2xl text-sm font-bold">
                    Microservices
                  </span>
                </div>
              </div>
              <div className="order-1 md:order-2">
                <div className="relative rounded-3xl overflow-hidden border-2 border-white group-hover:border-[#FE5934] transition-all duration-300">
                  <Image
                    src="/original-822ee72523e5fd7b7d2d1e968054b218.webp"
                    alt="Social Network Platform"
                    width={800}
                    height={600}
                    className="w-full h-auto object-cover"
                  />
                </div>
              </div>
            </div>
          </div>

        </div>
      </section>

      {/* Features Section */}
      <section className="py-20 px-4 md:px-8 bg-[#0D0D0D]">
        <div className="max-w-7xl mx-auto">
          <h2 className="text-4xl md:text-5xl font-bold text-center mb-16">
            Why Choose Our Web Applications?
          </h2>
          <div
            ref={featuresRef}
            className="grid md:grid-cols-4 gap-[30px]"
          >
            <div className="bg-black rounded-3xl p-8 border-2 border-white hover:border-[#A9EBF9] hover:bg-[#0D0D0D] transition-all duration-300">
              <div className="w-16 h-16 bg-[#A9EBF9] rounded-2xl flex items-center justify-center mb-6">
                <FiZap size={32} className="text-black" />
              </div>
              <h3 className="text-2xl font-bold mb-4">High Performance</h3>
              <p className="text-[#C0C0C0]">
                Optimized for speed with server-side rendering, code splitting, and efficient caching strategies.
              </p>
            </div>

            <div className="bg-black rounded-3xl p-8 border-2 border-white hover:border-[#8DDC96] hover:bg-[#0D0D0D] transition-all duration-300">
              <div className="w-16 h-16 bg-[#8DDC96] rounded-2xl flex items-center justify-center mb-6">
                <FiMonitor size={32} className="text-black" />
              </div>
              <h3 className="text-2xl font-bold mb-4">Responsive Design</h3>
              <p className="text-[#C0C0C0]">
                Fully responsive interfaces that work seamlessly across all devices and screen sizes.
              </p>
            </div>

            <div className="bg-black rounded-3xl p-8 border-2 border-white hover:border-[#DABDFF] hover:bg-[#0D0D0D] transition-all duration-300">
              <div className="w-16 h-16 bg-[#DABDFF] rounded-2xl flex items-center justify-center mb-6">
                <FiCode size={32} className="text-black" />
              </div>
              <h3 className="text-2xl font-bold mb-4">Clean Architecture</h3>
              <p className="text-[#C0C0C0]">
                Built with maintainable code following industry best practices and design patterns.
              </p>
            </div>

            <div className="bg-black rounded-3xl p-8 border-2 border-white hover:border-[#FFDA6F] hover:bg-[#0D0D0D] transition-all duration-300">
              <div className="w-16 h-16 bg-[#FFDA6F] rounded-2xl flex items-center justify-center mb-6">
                <FiLayers size={32} className="text-black" />
              </div>
              <h3 className="text-2xl font-bold mb-4">Scalable Solutions</h3>
              <p className="text-[#C0C0C0]">
                Architected to handle growth with microservices and cloud-native deployment strategies.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="py-20 px-4 md:px-8">
        <div className="max-w-4xl mx-auto text-center">
          <h2 className="text-4xl md:text-5xl font-bold mb-8">
            Ready to Build Your Next Web Application?
          </h2>
          <p className="text-xl text-[#C0C0C0] mb-12">
            Let&apos;s create powerful web solutions together
          </p>
          <Link
            href="/#contact"
            className="inline-block px-12 py-5 bg-white text-black border-2 border-white hover:bg-black hover:text-white transition-all duration-300 font-bold text-lg rounded-3xl"
          >
            Get In Touch
          </Link>
        </div>
      </section>

      {/* Footer */}
      <footer className="py-12 px-4 md:px-8 border-t-2 border-white">
        <div className="max-w-7xl mx-auto text-center">
          <p className="text-[#C0C0C0]">
            © 2025 Software Engineer Portfolio. All rights reserved.
          </p>
        </div>
      </footer>
    </div>
  );
}
