'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import Image from 'next/image';
import Link from 'next/link';
import { FiSmartphone, FiCode, FiZap } from 'react-icons/fi';
import { StaggeredMenu } from '@/components/StaggeredMenu';

gsap.registerPlugin(ScrollTrigger);

export default function MobileAppsPage() {
  const heroRef = useRef<HTMLDivElement>(null);
  const titleRef = useRef<HTMLHeadingElement>(null);
  const subtitleRef = useRef<HTMLParagraphElement>(null);
  const app1Ref = useRef<HTMLDivElement>(null);
  const app2Ref = useRef<HTMLDivElement>(null);
  const featuresRef = useRef<HTMLDivElement>(null);
  const feature1Ref = useRef<HTMLDivElement>(null);
  const feature2Ref = useRef<HTMLDivElement>(null);
  const feature3Ref = useRef<HTMLDivElement>(null);
  const ctaButtonRef = useRef<HTMLAnchorElement>(null);

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

      // CTA Button animation
      gsap.from(ctaButtonRef.current, {
        y: -50,
        opacity: 0,
        duration: 0.8,
        delay: 0.3,
        ease: 'power4.out',
      });

      // App card animations
      gsap.from(app1Ref.current, {
        scrollTrigger: {
          trigger: app1Ref.current,
          start: 'top 80%',
          toggleActions: 'play none none reverse',
        },
        x: -100,
        opacity: 0,
        duration: 1,
        ease: 'power3.out',
      });

      gsap.from(app2Ref.current, {
        scrollTrigger: {
          trigger: app2Ref.current,
          start: 'top 80%',
          toggleActions: 'play none none reverse',
        },
        x: 100,
        opacity: 0,
        duration: 1,
        ease: 'power3.out',
      });

      // Features section animation - individual cards
      const featureRefs = [feature1Ref, feature2Ref, feature3Ref];
      
      featureRefs.forEach((ref, index) => {
        if (ref.current) {
          gsap.from(ref.current, {
            scrollTrigger: {
              trigger: ref.current,
              start: 'top 85%',
              toggleActions: 'play none none reverse',
            },
            y: 50,
            opacity: 0,
            duration: 0.8,
            delay: index * 0.15,
            ease: 'power3.out',
          });
        }
      });
    });

    return () => ctx.revert();
  }, []);

  return (
    <div className="min-h-screen bg-black text-white">
      {/* Fixed Header with CTA Button - positioned to not overlap with menu */}
      <div className="fixed top-8 right-24 z-40 hidden md:block">
        <Link
          ref={ctaButtonRef}
          href="/#contact"
          className="px-8 py-4 bg-white text-black border-2 border-white hover:bg-[#FFA500] hover:border-[#FFA500] hover:text-white transition-all duration-300 font-bold text-base rounded-2xl shadow-lg"
        >
          Hire Me
        </Link>
      </div>

      {/* StaggeredMenu */}
      <StaggeredMenu
        position="right"
        colors={['#0D0D0D', '#000000']}
        items={[
          { label: 'Home', ariaLabel: 'Go to home page', link: '/' },
          { label: 'Projects', ariaLabel: 'View projects', link: '/#projects' },
          { label: 'About', ariaLabel: 'About me', link: '/#about' },
          { label: 'Contact', ariaLabel: 'Contact me', link: '/#contact' },
        ]}
        socialItems={[
          { label: 'GitHub', link: 'https://github.com' },
          { label: 'LinkedIn', link: 'https://linkedin.com' },
          { label: 'Twitter', link: 'https://twitter.com' },
        ]}
        displaySocials={true}
        displayItemNumbering={true}
        menuButtonColor="#FFFFFF"
        openMenuButtonColor="#000000"
        accentColor="#FFA500"
        isFixed={true}
        changeMenuColorOnOpen={true}
      />

      {/* Hero Section */}
      <section ref={heroRef} className="min-h-screen flex items-center justify-center px-4 md:px-8 pt-24 md:pt-0">
        <div className="max-w-6xl mx-auto text-center">
          <h1
            ref={titleRef}
            className="text-5xl md:text-7xl lg:text-8xl xl:text-9xl font-bold mb-8"
          >
            Mobile Apps
          </h1>
          <p
            ref={subtitleRef}
            className="text-xl md:text-2xl lg:text-3xl text-[#C0C0C0] max-w-3xl mx-auto"
          >
            Crafting beautiful and performant mobile experiences with cutting-edge technology
          </p>
        </div>
      </section>

      {/* Apps Showcase */}
      <section className="py-20 px-4 md:px-8 lg:px-12">
        <div className="max-w-7xl mx-auto space-y-[30px]">
          
          {/* App 1 */}
          <div
            ref={app1Ref}
            className="bg-[#0D0D0D] rounded-3xl p-8 md:p-12 lg:p-16 border-2 border-white hover:border-[#FFA500] transition-all duration-300 group"
          >
            <div className="grid lg:grid-cols-2 gap-12 lg:gap-16 items-center">
              <div className="order-2 lg:order-1">
                <div className="flex items-center gap-4 mb-6">
                  <FiSmartphone size={40} className="text-[#8DDC96]" />
                  <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold">Modern UI Experience</h2>
                </div>
                <p className="text-lg md:text-xl lg:text-2xl text-[#C0C0C0] mb-8">
                  A sleek and intuitive mobile application featuring modern design patterns,
                  smooth animations, and seamless user interactions. Built with performance
                  and user experience at the forefront.
                </p>
                <div className="flex flex-wrap gap-4">
                  <span className="px-6 py-3 bg-black border-2 border-[#A9EBF9] text-[#A9EBF9] rounded-2xl text-sm md:text-base font-bold">
                    React Native
                  </span>
                  <span className="px-6 py-3 bg-black border-2 border-[#DABDFF] text-[#DABDFF] rounded-2xl text-sm md:text-base font-bold">
                    TypeScript
                  </span>
                  <span className="px-6 py-3 bg-black border-2 border-[#FFDA6F] text-[#FFDA6F] rounded-2xl text-sm md:text-base font-bold">
                    Native Animations
                  </span>
                </div>
              </div>
              <div className="order-1 lg:order-2">
                <div className="relative rounded-3xl overflow-hidden border-2 border-white group-hover:border-[#FFA500] transition-all duration-300 shadow-2xl">
                  <Image
                    src="/d43ad1fb95964cffaa808bf7a0364307.webp"
                    alt="Mobile App Screenshot 1"
                    width={600}
                    height={800}
                    className="w-full h-auto object-cover"
                  />
                </div>
              </div>
            </div>
          </div>

          {/* App 2 */}
          <div
            ref={app2Ref}
            className="bg-[#0D0D0D] rounded-3xl p-8 md:p-12 lg:p-16 border-2 border-white hover:border-[#FBFF63] transition-all duration-300 group"
          >
            <div className="grid lg:grid-cols-2 gap-12 lg:gap-16 items-center">
              <div className="order-2 lg:order-2">
                <div className="flex items-center gap-4 mb-6">
                  <FiCode size={40} className="text-[#BDE7FF]" />
                  <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold">Feature-Rich Platform</h2>
                </div>
                <p className="text-lg md:text-xl lg:text-2xl text-[#C0C0C0] mb-8">
                  Advanced mobile application with comprehensive features including real-time
                  updates, cloud integration, and robust state management. Optimized for
                  both iOS and Android platforms.
                </p>
                <div className="flex flex-wrap gap-4">
                  <span className="px-6 py-3 bg-black border-2 border-[#8DDC96] text-[#8DDC96] rounded-2xl text-sm md:text-base font-bold">
                    Cross-Platform
                  </span>
                  <span className="px-6 py-3 bg-black border-2 border-[#FE5934] text-[#FE5934] rounded-2xl text-sm md:text-base font-bold">
                    Real-Time
                  </span>
                  <span className="px-6 py-3 bg-black border-2 border-[#FBFF63] text-[#FBFF63] rounded-2xl text-sm md:text-base font-bold">
                    Cloud Sync
                  </span>
                </div>
              </div>
              <div className="order-1 lg:order-1">
                <div className="relative rounded-3xl overflow-hidden border-2 border-white group-hover:border-[#FBFF63] transition-all duration-300 shadow-2xl">
                  <Image
                    src="/original-66e7443f7182936b9a9732295fbfc121.webp"
                    alt="Mobile App Screenshot 2"
                    width={600}
                    height={800}
                    className="w-full h-auto object-cover"
                  />
                </div>
              </div>
            </div>
          </div>

        </div>
      </section>

      {/* Features Section */}
      <section className="py-20 px-4 md:px-8 lg:px-12 bg-[#0D0D0D]">
        <div className="max-w-7xl mx-auto">
          <h2 className="text-4xl md:text-5xl lg:text-6xl font-bold text-center mb-16">
            Why Choose Our Mobile Apps?
          </h2>
          <div
            ref={featuresRef}
            className="grid md:grid-cols-2 lg:grid-cols-3 gap-[30px]"
          >
            <div
              ref={feature1Ref}
              className="bg-black rounded-3xl p-8 md:p-10 lg:p-12 border-2 border-white hover:border-[#A9EBF9] hover:bg-[#0D0D0D] transition-all duration-300 cursor-pointer"
            >
              <div className="w-20 h-20 bg-[#A9EBF9] rounded-2xl flex items-center justify-center mb-6">
                <FiZap size={40} className="text-black" />
              </div>
              <h3 className="text-2xl md:text-3xl font-bold mb-4">Lightning Fast</h3>
              <p className="text-lg md:text-xl text-[#C0C0C0]">
                Optimized performance with smooth 60fps animations and instant response times
                for the best user experience.
              </p>
            </div>

            <div
              ref={feature2Ref}
              className="bg-black rounded-3xl p-8 md:p-10 lg:p-12 border-2 border-white hover:border-[#8DDC96] hover:bg-[#0D0D0D] transition-all duration-300 cursor-pointer"
            >
              <div className="w-20 h-20 bg-[#8DDC96] rounded-2xl flex items-center justify-center mb-6">
                <FiSmartphone size={40} className="text-black" />
              </div>
              <h3 className="text-2xl md:text-3xl font-bold mb-4">Responsive Design</h3>
              <p className="text-lg md:text-xl text-[#C0C0C0]">
                Perfectly adapted for all screen sizes and devices, ensuring consistency
                across iOS and Android platforms.
              </p>
            </div>

            <div
              ref={feature3Ref}
              className="bg-black rounded-3xl p-8 md:p-10 lg:p-12 border-2 border-white hover:border-[#DABDFF] hover:bg-[#0D0D0D] transition-all duration-300 cursor-pointer"
            >
              <div className="w-20 h-20 bg-[#DABDFF] rounded-2xl flex items-center justify-center mb-6">
                <FiCode size={40} className="text-black" />
              </div>
              <h3 className="text-2xl md:text-3xl font-bold mb-4">Clean Code</h3>
              <p className="text-lg md:text-xl text-[#C0C0C0]">
                Written with best practices, following industry standards for maintainability,
                scalability, and performance.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="py-20 px-4 md:px-8 lg:px-12">
        <div className="max-w-4xl mx-auto text-center">
          <h2 className="text-4xl md:text-5xl lg:text-6xl font-bold mb-8">
            Ready to Build Something Amazing?
          </h2>
          <p className="text-xl md:text-2xl lg:text-3xl text-[#C0C0C0] mb-12">
            Let&apos;s create exceptional mobile experiences together
          </p>
          <Link
            href="/#contact"
            className="inline-block px-12 py-6 bg-white text-black border-2 border-white hover:bg-black hover:text-white transition-all duration-300 font-bold text-lg md:text-xl rounded-3xl shadow-lg"
          >
            Get In Touch
          </Link>
        </div>
      </section>

      {/* Footer */}
      <footer className="py-12 px-4 md:px-8 lg:px-12 border-t-2 border-white">
        <div className="max-w-7xl mx-auto text-center">
          <p className="text-base md:text-lg text-[#C0C0C0]">
            © 2025 Software Engineer Portfolio. All rights reserved.
          </p>
        </div>
      </footer>
    </div>
  );
}
