'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import InfiniteScroll from './InfiniteScroll';
import { useIsMobile } from '../hooks/useDevice';
import { useInView } from '../hooks/useInView';

gsap.registerPlugin(ScrollTrigger);

export default function Footer() {
  const { isMobile } = useIsMobile();
  const { ref: footerRef, isInView } = useInView<HTMLElement>({ exit: true, rootMargin: '0px' });
  const contentRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!footerRef.current || !contentRef.current) return;

    if (isMobile) {
      gsap.set(contentRef.current, { opacity: 1, y: 0 });
      return;
    }

    const ctx = gsap.context(() => {
      gsap.timeline({
        scrollTrigger: {
          trigger: footerRef.current,
          start: 'top 90%',
          end: 'bottom bottom',
          toggleActions: 'play none none reverse',
        },
      }).fromTo(contentRef.current, { opacity: 0, y: 50 }, { opacity: 1, y: 0, duration: 1 });
    }, footerRef);

    return () => ctx.revert();
  }, [isMobile]);

  // Large text items for full-width background coverage
  const scrollItems = [
    { content: <div className="text-white/25 text-[6rem] md:text-[9rem] lg:text-[11rem] font-black leading-none tracking-tighter">DISPATCH</div> },
    { content: <div className="text-white/25 text-[6rem] md:text-[9rem] lg:text-[11rem] font-black leading-none tracking-tighter">TRACK</div> },
    { content: <div className="text-white/25 text-[6rem] md:text-[9rem] lg:text-[11rem] font-black leading-none tracking-tighter">DELIVER</div> },
    { content: <div className="text-white/25 text-[6rem] md:text-[9rem] lg:text-[11rem] font-black leading-none tracking-tighter">PEGASUS</div> },
  ];

  const quickLinks = [
    { name: 'Home', href: '#hero' },
    { name: 'About', href: '#about' },
    { name: 'Skills', href: '#skills' },
    { name: 'Projects', href: '#projects' },
    { name: 'Demo', href: '/join' }
  ];

  const socialLinks = [
    { name: 'GitHub', href: 'https://github.com', icon: '💻' },
    { name: 'LinkedIn', href: 'https://linkedin.com', icon: '💼' },
    { name: 'Twitter', href: 'https://twitter.com', icon: '🐦' },
    { name: 'Email', href: 'mailto:demo@pegasus.io', icon: '📧' }
  ];

  return (
    <footer ref={footerRef} className="relative bg-black text-white overflow-hidden min-h-screen flex items-center">
      
      {/* Infinite Scroll Background - Hidden on mobile */}
      {!isMobile && isInView ? (
        <div className="absolute inset-0 pointer-events-none z-0 flex">
          <div className="flex-1 h-full">
            <InfiniteScroll
              items={scrollItems}
              isTilted={true}
              tiltDirection="left"
              autoplay={true}
              autoplaySpeed={0.45}
              autoplayDirection="down"
              pauseOnHover={true}
              width="100%"
              maxHeight="100%"
              itemMinHeight={180}
              negativeMargin="-1.5rem"
            />
          </div>

          <div className="flex-1 h-full hidden md:block">
            <InfiniteScroll
              items={scrollItems}
              isTilted={true}
              tiltDirection="right"
              autoplay={true}
              autoplaySpeed={0.55}
              autoplayDirection="up"
              pauseOnHover={true}
              width="100%"
              maxHeight="100%"
              itemMinHeight={180}
              negativeMargin="-1.5rem"
            />
          </div>
        </div>
      ) : null}

      {/* Footer Content */}
      <div ref={contentRef} className="relative z-20 container mx-auto px-4 py-20">
        <div className="max-w-7xl mx-auto">
          {/* Main Footer Content */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-12 mb-16">
            
            {/* Brand Section */}
            <div className="lg:col-span-2">
              <h3 className="text-3xl md:text-4xl font-bold mb-6 text-white">
                Run Your Network on Pegasus
              </h3>
              <p className="text-white text-lg mb-6 max-w-md leading-relaxed">
                Dispatch, tracking, payments, and coordination for supplier-led logistics — one platform, six roles, every site connected.
              </p>
              <a
                href="/join"
                className="inline-block px-8 py-4 bg-white text-black border-2 border-white hover:bg-black hover:text-white transition-all duration-300 font-bold rounded-2xl"
              >
                GET A DEMO →
              </a>
            </div>

            {/* Quick Links */}
            <div>
              <h4 className="text-xl font-bold mb-6 text-white">Quick Links</h4>
              <ul className="space-y-3">
                {quickLinks.map((link, index) => (
                  <li key={index}>
                    <a
                      href={link.href}
                      className="text-white hover:text-gray-300 transition-colors duration-300 text-lg"
                    >
                      {link.name}
                    </a>
                  </li>
                ))}
              </ul>
            </div>

            {/* Social Links */}
            <div>
              <h4 className="text-xl font-bold mb-6 text-white">Connect</h4>
              <div className="space-y-4">
                {socialLinks.map((social, index) => (
                  <a
                    key={index}
                    href={social.href}
                    target={social.href.startsWith('http') ? '_blank' : undefined}
                    rel={social.href.startsWith('http') ? 'noreferrer noopener' : undefined}
                    className="flex items-center gap-3 text-white hover:text-gray-300 transition-colors duration-300 group"
                  >
                    <span className="text-2xl group-hover:scale-110 transition-transform duration-300">
                      {social.icon}
                    </span>
                    <span className="text-lg">{social.name}</span>
                  </a>
                ))}
              </div>
            </div>
          </div>

          {/* Divider */}
          <div className="border-t-2 border-white/20 mb-8"></div>

          {/* Bottom Footer */}
          <div className="flex flex-col md:flex-row justify-between items-center gap-6">
            <div className="text-white/60 text-center md:text-left">
              <p className="text-lg">
                © {new Date().getFullYear()} Pegasus. All rights reserved.
              </p>
            </div>

            <div className="flex gap-8">
              <a
                href="#"
                className="text-white/60 hover:text-white transition-colors duration-300 text-lg"
              >
                Privacy Policy
              </a>
              <a
                href="#"
                className="text-white/60 hover:text-white transition-colors duration-300 text-lg"
              >
                Terms of Service
              </a>
            </div>
          </div>

          {/* Back to Top Button */}
          <div className="mt-12 text-center">
            <a
              href="#hero"
              className="inline-flex items-center gap-2 px-6 py-3 border-2 border-white text-white hover:bg-white hover:text-black transition-all duration-300 rounded-2xl font-bold"
            >
              <span>↑</span>
              <span>BACK TO TOP</span>
            </a>
          </div>
          
        </div>
      </div>
    </footer>
  );
}
