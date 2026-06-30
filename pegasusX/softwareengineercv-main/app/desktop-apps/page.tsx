'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import Image from 'next/image';
import Link from 'next/link';
import { FiMonitor, FiCpu, FiHardDrive, FiLayers, FiZap } from 'react-icons/fi';
import StaggeredMenu from '@/components/StaggeredMenu';

gsap.registerPlugin(ScrollTrigger);

export default function DesktopAppsPage() {
  const heroRef = useRef<HTMLDivElement>(null);
  const titleRef = useRef<HTMLHeadingElement>(null);
  const subtitleRef = useRef<HTMLParagraphElement>(null);
  const app1Ref = useRef<HTMLDivElement>(null);
  const app2Ref = useRef<HTMLDivElement>(null);
  const app3Ref = useRef<HTMLDivElement>(null);
  const app4Ref = useRef<HTMLDivElement>(null);
  const app5Ref = useRef<HTMLDivElement>(null);
  const featuresRef = useRef<HTMLDivElement>(null);
  const feature1Ref = useRef<HTMLDivElement>(null);
  const feature2Ref = useRef<HTMLDivElement>(null);
  const feature3Ref = useRef<HTMLDivElement>(null);
  const feature4Ref = useRef<HTMLDivElement>(null);

  const menuItems = [
    { label: 'Home', ariaLabel: 'Go to home page', link: '/' },
    { label: 'About', ariaLabel: 'Learn about me', link: '/#about' },
    { label: 'Projects', ariaLabel: 'View all projects', link: '/#projects' },
    { label: 'Web Apps', ariaLabel: 'View web applications', link: '/web-apps' },
    { label: 'Mobile Apps', ariaLabel: 'View mobile applications', link: '/mobile-apps' },
    { label: 'Desktop Apps', ariaLabel: 'View desktop applications', link: '/desktop-apps' },
    { label: 'Contact', ariaLabel: 'Get in touch', link: '/#contact' }
  ];

  const socialItems = [
    { label: 'GitHub', link: 'https://github.com' },
    { label: 'LinkedIn', link: 'https://linkedin.com' },
    { label: 'Twitter', link: 'https://twitter.com' }
  ];

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

      // Features section animation - individual cards
      const featureRefs = [feature1Ref, feature2Ref, feature3Ref, feature4Ref];
      
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
      {/* StaggeredMenu */}
      <StaggeredMenu
        position="right"
        items={menuItems}
        socialItems={socialItems}
        displaySocials={true}
        displayItemNumbering={true}
        menuButtonColor="#FFFFFF"
        openMenuButtonColor="#000000"
        changeMenuColorOnOpen={true}
        colors={['#0D0D0D', '#1a1a1a', '#000000']}
        accentColor="#A9EBF9"
        isFixed={true}
        onMenuOpen={() => console.log('Menu opened')}
        onMenuClose={() => console.log('Menu closed')}
      />

      {/* Hero Section */}
      <section ref={heroRef} className="min-h-screen flex items-center justify-center px-4 md:px-8 pt-24 md:pt-0">
        <div className="max-w-6xl mx-auto text-center">
          <h1
            ref={titleRef}
            className="text-5xl md:text-7xl lg:text-8xl font-bold mb-8"
          >
            Desktop Applications
          </h1>
          <p
            ref={subtitleRef}
            className="text-xl md:text-2xl text-[#C0C0C0] max-w-3xl mx-auto"
          >
            Powerful native desktop applications built for performance, efficiency, and seamless user experiences
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
                  <h2 className="text-3xl md:text-4xl font-bold">Video Editing Suite</h2>
                </div>
                <p className="text-lg text-[#C0C0C0] mb-6">
                  Professional-grade video editing desktop application with multi-track timeline, 
                  real-time effects, color grading, and advanced rendering capabilities for creators and professionals.
                </p>
                <div className="flex flex-wrap gap-4">
                  <span className="px-4 py-2 bg-black border-2 border-[#A9EBF9] text-[#A9EBF9] rounded-2xl text-sm font-bold">
                    Electron
                  </span>
                  <span className="px-4 py-2 bg-black border-2 border-[#8DDC96] text-[#8DDC96] rounded-2xl text-sm font-bold">
                    C++
                  </span>
                  <span className="px-4 py-2 bg-black border-2 border-[#DABDFF] text-[#DABDFF] rounded-2xl text-sm font-bold">
                    GPU Acceleration
                  </span>
                </div>
              </div>
              <div className="order-1 md:order-2">
                <div className="relative rounded-3xl overflow-hidden border-2 border-white group-hover:border-[#A9EBF9] transition-all duration-300">
                  <Image
                    src="/original-1137dbc252460b3d33f86e46095e1b12.webp"
                    alt="Video Editing Desktop Application"
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
                  <FiCpu size={32} className="text-[#8DDC96]" />
                  <h2 className="text-3xl md:text-4xl font-bold">System Monitor Pro</h2>
                </div>
                <p className="text-lg text-[#C0C0C0] mb-6">
                  Advanced system monitoring tool providing real-time CPU, memory, disk, and network statistics 
                  with customizable alerts and performance optimization recommendations.
                </p>
                <div className="flex flex-wrap gap-4">
                  <span className="px-4 py-2 bg-black border-2 border-[#8DDC96] text-[#8DDC96] rounded-2xl text-sm font-bold">
                    Native
                  </span>
                  <span className="px-4 py-2 bg-black border-2 border-[#FFDA6F] text-[#FFDA6F] rounded-2xl text-sm font-bold">
                    Real-Time Monitoring
                  </span>
                  <span className="px-4 py-2 bg-black border-2 border-[#FE5934] text-[#FE5934] rounded-2xl text-sm font-bold">
                    Cross-Platform
                  </span>
                </div>
              </div>
              <div className="order-1 md:order-1">
                <div className="relative rounded-3xl overflow-hidden border-2 border-white group-hover:border-[#8DDC96] transition-all duration-300">
                  <Image
                    src="/original-2096796ee8410886b847d3c0a93c3d4e.webp"
                    alt="System Monitor Desktop Application"
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
                  <h2 className="text-3xl md:text-4xl font-bold">Code Editor IDE</h2>
                </div>
                <p className="text-lg text-[#C0C0C0] mb-6">
                  Modern integrated development environment with intelligent code completion, 
                  debugging tools, version control integration, and support for multiple programming languages.
                </p>
                <div className="flex flex-wrap gap-4">
                  <span className="px-4 py-2 bg-black border-2 border-[#DABDFF] text-[#DABDFF] rounded-2xl text-sm font-bold">
                    TypeScript
                  </span>
                  <span className="px-4 py-2 bg-black border-2 border-[#BDE7FF] text-[#BDE7FF] rounded-2xl text-sm font-bold">
                    LSP Support
                  </span>
                  <span className="px-4 py-2 bg-black border-2 border-[#A9EBF9] text-[#A9EBF9] rounded-2xl text-sm font-bold">
                    Plugin System
                  </span>
                </div>
              </div>
              <div className="order-1 md:order-2">
                <div className="relative rounded-3xl overflow-hidden border-2 border-white group-hover:border-[#DABDFF] transition-all duration-300">
                  <Image
                    src="/original-50941087b949fb6cd5a51dc535e09879.webp"
                    alt="Code Editor Desktop Application"
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
                  <FiHardDrive size={32} className="text-[#FFDA6F]" />
                  <h2 className="text-3xl md:text-4xl font-bold">File Manager Plus</h2>
                </div>
                <p className="text-lg text-[#C0C0C0] mb-6">
                  Enhanced file management application with dual-pane interface, advanced search, 
                  batch operations, cloud storage integration, and powerful file organization tools.
                </p>
                <div className="flex flex-wrap gap-4">
                  <span className="px-4 py-2 bg-black border-2 border-[#FFDA6F] text-[#FFDA6F] rounded-2xl text-sm font-bold">
                    Native UI
                  </span>
                  <span className="px-4 py-2 bg-black border-2 border-[#FFA500] text-[#FFA500] rounded-2xl text-sm font-bold">
                    Cloud Sync
                  </span>
                  <span className="px-4 py-2 bg-black border-2 border-[#FBFF63] text-[#FBFF63] rounded-2xl text-sm font-bold">
                    Fast Search
                  </span>
                </div>
              </div>
              <div className="order-1 md:order-1">
                <div className="relative rounded-3xl overflow-hidden border-2 border-white group-hover:border-[#FFDA6F] transition-all duration-300">
                  <Image
                    src="/original-b06b1f111a877844e5b3566f18b74306.webp"
                    alt="File Manager Desktop Application"
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
                  <FiZap size={32} className="text-[#FE5934]" />
                  <h2 className="text-3xl md:text-4xl font-bold">Design Studio</h2>
                </div>
                <p className="text-lg text-[#C0C0C0] mb-6">
                  Professional graphic design application with vector editing, raster manipulation, 
                  typography tools, and seamless workflow for creating stunning visual content.
                </p>
                <div className="flex flex-wrap gap-4">
                  <span className="px-4 py-2 bg-black border-2 border-[#FE5934] text-[#FE5934] rounded-2xl text-sm font-bold">
                    Vector Graphics
                  </span>
                  <span className="px-4 py-2 bg-black border-2 border-[#FFA500] text-[#FFA500] rounded-2xl text-sm font-bold">
                    Layers System
                  </span>
                  <span className="px-4 py-2 bg-black border-2 border-[#A9EBF9] text-[#A9EBF9] rounded-2xl text-sm font-bold">
                    Export Multiple Formats
                  </span>
                </div>
              </div>
              <div className="order-1 md:order-2">
                <div className="relative rounded-3xl overflow-hidden border-2 border-white group-hover:border-[#FE5934] transition-all duration-300">
                  <Image
                    src="/original-1ec76e0e43bf8113cf940c5b4cdaed7a.webp"
                    alt="Design Studio Desktop Application"
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
            Why Choose Our Desktop Applications?
          </h2>
          <div
            ref={featuresRef}
            className="grid md:grid-cols-4 gap-[30px]"
          >
            <div
              ref={feature1Ref}
              className="bg-black rounded-3xl p-8 border-2 border-white hover:border-[#A9EBF9] hover:bg-[#0D0D0D] transition-all duration-300"
            >
              <div className="w-16 h-16 bg-[#A9EBF9] rounded-2xl flex items-center justify-center mb-6">
                <FiZap size={32} className="text-black" />
              </div>
              <h3 className="text-2xl font-bold mb-4">Native Performance</h3>
              <p className="text-[#C0C0C0]">
                Built with native technologies for maximum speed, efficiency, and system resource optimization.
              </p>
            </div>

            <div
              ref={feature2Ref}
              className="bg-black rounded-3xl p-8 border-2 border-white hover:border-[#8DDC96] hover:bg-[#0D0D0D] transition-all duration-300"
            >
              <div className="w-16 h-16 bg-[#8DDC96] rounded-2xl flex items-center justify-center mb-6">
                <FiMonitor size={32} className="text-black" />
              </div>
              <h3 className="text-2xl font-bold mb-4">Cross-Platform</h3>
              <p className="text-[#C0C0C0]">
                Works seamlessly across Windows, macOS, and Linux with consistent user experience.
              </p>
            </div>

            <div
              ref={feature3Ref}
              className="bg-black rounded-3xl p-8 border-2 border-white hover:border-[#DABDFF] hover:bg-[#0D0D0D] transition-all duration-300"
            >
              <div className="w-16 h-16 bg-[#DABDFF] rounded-2xl flex items-center justify-center mb-6">
                <FiHardDrive size={32} className="text-black" />
              </div>
              <h3 className="text-2xl font-bold mb-4">Offline First</h3>
              <p className="text-[#C0C0C0]">
                Full functionality without internet connection, with optional cloud sync for collaboration.
              </p>
            </div>

            <div
              ref={feature4Ref}
              className="bg-black rounded-3xl p-8 border-2 border-white hover:border-[#FFDA6F] hover:bg-[#0D0D0D] transition-all duration-300"
            >
              <div className="w-16 h-16 bg-[#FFDA6F] rounded-2xl flex items-center justify-center mb-6">
                <FiCpu size={32} className="text-black" />
              </div>
              <h3 className="text-2xl font-bold mb-4">Hardware Integration</h3>
              <p className="text-[#C0C0C0]">
                Direct access to system hardware and peripherals for professional-grade capabilities.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="py-20 px-4 md:px-8">
        <div className="max-w-4xl mx-auto text-center">
          <h2 className="text-4xl md:text-5xl font-bold mb-8">
            Ready to Build Your Desktop Application?
          </h2>
          <p className="text-xl text-[#C0C0C0] mb-12">
            Let&apos;s create powerful native applications together
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
