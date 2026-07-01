'use client';

import { useEffect, useRef } from 'react';
import Link from 'next/link';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import { SiGithub, SiLinkedin } from 'react-icons/si';
import { HiOutlineMail } from 'react-icons/hi';
import { FaXTwitter } from 'react-icons/fa6';
import InfiniteScroll from './InfiniteScroll';
import { useIsMobile } from '../hooks/useDevice';
import { useInView } from '../hooks/useInView';

gsap.registerPlugin(ScrollTrigger);

const PLATFORM_LINKS = [
  { name: 'Platform overview', href: '/platform' },
  { name: 'Order lifecycle', href: '/platform/order-lifecycle' },
  { name: 'How Pegasus works', href: '/platform/how-pegasus-works' },
  { name: 'Trust & reliability', href: '/platform/trust-reliability' },
];

const SOLUTION_LINKS = [
  { name: 'Dispatch engine', href: '/solutions/visual-dispatch-engine' },
  { name: 'Fleet visibility', href: '/solutions/fleet-visibility' },
  { name: 'Payment confidence', href: '/capabilities/payment-confidence' },
  { name: 'Live coordination', href: '/capabilities/instant-coordination' },
];

const EXPLORE_LINKS = [
  { name: 'Roles', href: '/roles' },
  { name: 'Technology', href: '/technology' },
  { name: 'Open source stack', href: '/technology/go-backend-platform' },
  { name: 'Apps & deploy', href: '/apps-deploy' },
  { name: 'Contact', href: '/contact' },
  { name: 'Request demo', href: '/join' },
];

const SOCIAL_LINKS = [
  { name: 'GitHub', href: 'https://github.com', Icon: SiGithub },
  { name: 'LinkedIn', href: 'https://linkedin.com/company/pegasus', Icon: SiLinkedin },
  { name: 'X', href: 'https://twitter.com', Icon: FaXTwitter },
  { name: 'Email', href: 'mailto:demo@pegasus.io', Icon: HiOutlineMail },
] as const;

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
  }, [isMobile, footerRef]);

  const scrollItems = [
    { content: <div className="text-white/20 text-[6rem] md:text-[9rem] lg:text-[11rem] font-black leading-none tracking-tighter">DISPATCH</div> },
    { content: <div className="text-white/20 text-[6rem] md:text-[9rem] lg:text-[11rem] font-black leading-none tracking-tighter">TRACK</div> },
    { content: <div className="text-white/20 text-[6rem] md:text-[9rem] lg:text-[11rem] font-black leading-none tracking-tighter">DELIVER</div> },
    { content: <div className="text-white/20 text-[6rem] md:text-[9rem] lg:text-[11rem] font-black leading-none tracking-tighter">PEGASUS</div> },
  ];

  return (
    <footer ref={footerRef} className="relative bg-black text-white overflow-hidden min-h-screen flex items-center">
      {!isMobile && isInView ? (
        <div className="absolute inset-0 pointer-events-none z-0">
          <InfiniteScroll
            items={scrollItems}
            isTilted
            tiltDirection="left"
            autoplay
            autoplaySpeed={0.45}
            autoplayDirection="down"
            pauseOnHover
            width="100%"
            maxHeight="100%"
            itemMinHeight={180}
            negativeMargin="-1.5rem"
          />
        </div>
      ) : null}

      <div ref={contentRef} className="relative z-20 container mx-auto px-4 py-20">
        <div className="max-w-7xl mx-auto">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-12 gap-12 mb-16">
            <div className="lg:col-span-4">
              <h3 className="text-3xl md:text-4xl font-light mb-6 text-white">
                Run your network on Pegasus
              </h3>
              <p className="text-white/70 text-lg mb-6 max-w-md leading-relaxed">
                Dispatch, tracking, payments, and coordination for supplier-led logistics — one platform,
                six roles, every site connected.
              </p>
              <Link href="/join" className="editorial-btn">
                GET A DEMO →
              </Link>
            </div>

            <div className="lg:col-span-2">
              <h4 className="editorial-eyebrow mb-6">Platform</h4>
              <ul className="space-y-3">
                {PLATFORM_LINKS.map((link) => (
                  <li key={link.name}>
                    <Link href={link.href} className="text-white/75 hover:text-white transition-colors text-base">
                      {link.name}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>

            <div className="lg:col-span-2">
              <h4 className="editorial-eyebrow mb-6">Solutions</h4>
              <ul className="space-y-3">
                {SOLUTION_LINKS.map((link) => (
                  <li key={link.name}>
                    <Link href={link.href} className="text-white/75 hover:text-white transition-colors text-base">
                      {link.name}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>

            <div className="lg:col-span-2">
              <h4 className="editorial-eyebrow mb-6">Explore</h4>
              <ul className="space-y-3">
                {EXPLORE_LINKS.map((link) => (
                  <li key={link.name}>
                    <Link href={link.href} className="text-white/75 hover:text-white transition-colors text-base">
                      {link.name}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>

            <div className="lg:col-span-2">
              <h4 className="editorial-eyebrow mb-6">Connect</h4>
              <div className="space-y-4">
                {SOCIAL_LINKS.map(({ name, href, Icon }) => (
                  <a
                    key={name}
                    href={href}
                    target={href.startsWith('http') ? '_blank' : undefined}
                    rel={href.startsWith('http') ? 'noreferrer noopener' : undefined}
                    className="flex items-center gap-3 text-white/75 hover:text-white transition-colors group"
                  >
                    <span className="flex h-10 w-10 items-center justify-center border border-white/20 group-hover:border-white/50 transition-colors">
                      <Icon className="h-4 w-4" aria-hidden />
                    </span>
                    <span className="text-base">{name}</span>
                  </a>
                ))}
              </div>
            </div>
          </div>

          <div className="border-t border-white/15 mb-8" />

          <div className="flex flex-col md:flex-row justify-between items-center gap-6">
            <p className="text-white/50 text-center md:text-left">
              © {new Date().getFullYear()} Pegasus. All rights reserved.
            </p>
            <div className="flex gap-8">
              <Link href="/contact" className="text-white/50 hover:text-white transition-colors">
                Privacy & data requests
              </Link>
              <Link href="/contact" className="text-white/50 hover:text-white transition-colors">
                Terms of use
              </Link>
            </div>
          </div>

          <div className="mt-12 text-center">
            <a href="#hero" className="editorial-btn editorial-btn--sm">
              <span>↑</span>
              <span>BACK TO TOP</span>
            </a>
          </div>
        </div>
      </div>
    </footer>
  );
}
