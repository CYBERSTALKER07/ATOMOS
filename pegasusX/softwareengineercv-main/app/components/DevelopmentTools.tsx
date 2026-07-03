'use client';

import { useCallback, useEffect, useRef, type ReactNode } from 'react';
import Link from 'next/link';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import LogoLoop, { type LogoItem } from './LogoLoop';
import { useInView } from '../hooks/useInView';
import PageSection from './layout/PageSection';
import SectionHeader from './layout/SectionHeader';
import {
  SiReact,
  SiNextdotjs,
  SiVuedotjs,
  SiSvelte,
  SiAngular,
  SiNodedotjs,
  SiExpress,
  SiGraphql,
  SiSocketdotio,
  SiPostgresql,
  SiMongodb,
  SiRedis,
  SiFirebase,
  SiSupabase,
  SiDocker,
  SiVercel,
  SiGithubactions,
  SiKubernetes,
  SiGit,
  SiFigma,
  SiPostman,
  SiJest,
  SiTypescript,
  SiTailwindcss,
  SiPython,
  SiDjango,
  SiFastapi,
} from 'react-icons/si';
import { VscCode } from 'react-icons/vsc';
import { FaAws } from 'react-icons/fa6';

gsap.registerPlugin(ScrollTrigger);

const SPOTLIGHT_RADIUS = 140;
const MONO_FILTER = 'grayscale(1) brightness(1.85)';

function icon(node: ReactNode, brandColor: string, title: string, href: string): LogoItem {
  return {
    node,
    brandColor,
    title,
    href,
    logoClassName: 'platform-stack-logo',
  };
}

const carouselRows: { logos: LogoItem[]; direction: 'left' | 'right' }[] = [
  {
    direction: 'left',
    logos: [
      icon(<SiReact />, '#61DAFB', 'React', 'https://react.dev'),
      icon(<SiNextdotjs />, '#FFFFFF', 'Next.js', 'https://nextjs.org'),
      icon(<SiVuedotjs />, '#4FC08D', 'Vue.js', 'https://vuejs.org'),
      icon(<SiSvelte />, '#FF3E00', 'Svelte', 'https://svelte.dev'),
      icon(<SiAngular />, '#DD0031', 'Angular', 'https://angular.io'),
      icon(<SiTypescript />, '#3178C6', 'TypeScript', 'https://www.typescriptlang.org'),
      icon(<SiTailwindcss />, '#06B6D4', 'Tailwind CSS', 'https://tailwindcss.com'),
    ],
  },
  {
    direction: 'right',
    logos: [
      icon(<SiNodedotjs />, '#339933', 'Node.js', 'https://nodejs.org'),
      icon(<SiExpress />, '#FFFFFF', 'Express', 'https://expressjs.com'),
      icon(<SiGraphql />, '#E10098', 'GraphQL', 'https://graphql.org'),
      icon(<SiSocketdotio />, '#FFFFFF', 'Socket.io', 'https://socket.io'),
      icon(<SiPython />, '#3776AB', 'Python', 'https://python.org'),
      icon(<SiDjango />, '#44B78B', 'Django', 'https://djangoproject.com'),
      icon(<SiFastapi />, '#009688', 'FastAPI', 'https://fastapi.tiangolo.com'),
    ],
  },
  {
    direction: 'left',
    logos: [
      icon(<SiPostgresql />, '#4169E1', 'PostgreSQL', 'https://postgresql.org'),
      icon(<SiMongodb />, '#47A248', 'MongoDB', 'https://mongodb.com'),
      icon(<SiRedis />, '#DC382D', 'Redis', 'https://redis.io'),
      icon(<SiFirebase />, '#FFCA28', 'Firebase', 'https://firebase.google.com'),
      icon(<SiSupabase />, '#3FCF8E', 'Supabase', 'https://supabase.com'),
    ],
  },
  {
    direction: 'right',
    logos: [
      icon(<SiDocker />, '#2496ED', 'Docker', 'https://docker.com'),
      icon(<FaAws />, '#FF9900', 'AWS', 'https://aws.amazon.com'),
      icon(<SiVercel />, '#FFFFFF', 'Vercel', 'https://vercel.com'),
      icon(<SiGithubactions />, '#2088FF', 'GitHub Actions', 'https://github.com/features/actions'),
      icon(<SiKubernetes />, '#326CE5', 'Kubernetes', 'https://kubernetes.io'),
    ],
  },
  {
    direction: 'left',
    logos: [
      icon(<SiGit />, '#F05032', 'Git', 'https://git-scm.com'),
      icon(<VscCode />, '#007ACC', 'VS Code', 'https://code.visualstudio.com'),
      icon(<SiFigma />, '#F24E1E', 'Figma', 'https://figma.com'),
      icon(<SiPostman />, '#FF6C37', 'Postman', 'https://postman.com'),
      icon(<SiJest />, '#C21325', 'Jest', 'https://jestjs.io'),
    ],
  },
];

function setLogoMono(el: HTMLElement) {
  el.style.color = '#ffffff';
  el.style.filter = MONO_FILTER;
}

function revealLogo(el: HTMLElement) {
  const brand = el.dataset.brandColor;
  el.style.color = brand ?? '#ffffff';
  el.style.filter = 'none';
}

export default function DevelopmentTools() {
  const { ref: sectionRef, isInView } = useInView<HTMLElement>({ rootMargin: '0px' });
  const titleRef = useRef<HTMLDivElement>(null);
  const stackRef = useRef<HTMLDivElement>(null);
  const rowsRef = useRef<HTMLDivElement>(null);
  const rafRef = useRef<number | null>(null);
  const pointerRef = useRef<{ x: number; y: number; active: boolean }>({
    x: 0,
    y: 0,
    active: false,
  });

  useEffect(() => {
    if (!sectionRef.current) return;

    const ctx = gsap.context(() => {
      const timeline = gsap.timeline({
        scrollTrigger: {
          trigger: sectionRef.current,
          start: 'top 80%',
          end: 'bottom 20%',
          toggleActions: 'play none none reverse',
        },
      });

      timeline.fromTo(titleRef.current, { opacity: 0, y: 30 }, { opacity: 1, y: 0, duration: 0.8 });
      timeline.fromTo(
        rowsRef.current,
        { opacity: 0, y: 40 },
        { opacity: 1, y: 0, duration: 0.7 },
        '-=0.45'
      );
    }, sectionRef);

    return () => ctx.revert();
  }, [sectionRef]);

  useEffect(() => {
    const stack = stackRef.current;
    if (!stack) return;

    stack.querySelectorAll<HTMLElement>('.platform-stack-logo').forEach(setLogoMono);

    return () => {
      if (rafRef.current !== null) cancelAnimationFrame(rafRef.current);
    };
  }, []);

  const applySpotlight = useCallback(() => {
    const stack = stackRef.current;
    if (!stack) return;

    const { x, y, active } = pointerRef.current;
    const logos = stack.querySelectorAll<HTMLElement>('.platform-stack-logo');
    const radiusSq = SPOTLIGHT_RADIUS * SPOTLIGHT_RADIUS;

    logos.forEach((el) => {
      if (!active) {
        setLogoMono(el);
        return;
      }

      const rect = el.getBoundingClientRect();
      const cx = rect.left + rect.width / 2;
      const cy = rect.top + rect.height / 2;
      const dx = x - cx;
      const dy = y - cy;

      if (dx * dx + dy * dy <= radiusSq) {
        revealLogo(el);
      } else {
        setLogoMono(el);
      }
    });
  }, []);

  const handlePointerMove = useCallback(
    (event: React.PointerEvent<HTMLDivElement>) => {
      pointerRef.current = { x: event.clientX, y: event.clientY, active: true };

      if (rafRef.current !== null) return;
      rafRef.current = requestAnimationFrame(() => {
        rafRef.current = null;
        applySpotlight();
      });
    },
    [applySpotlight]
  );

  const handlePointerLeave = useCallback(() => {
    pointerRef.current.active = false;
    applySpotlight();
  }, [applySpotlight]);

  return (
    <PageSection ref={sectionRef} id="tools">
      <div ref={titleRef}>
        <SectionHeader
          align="center"
          eyebrow="Under the hood"
          title="Built to run at network scale"
          description="Production-grade infrastructure keeps your operation reliable — explore the full technology stack, open-source components, and architecture on our technology pages."
        />
        <div className="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
          <Link href="/technology" className="editorial-btn">
            VIEW TECHNOLOGY
          </Link>
          <Link href="/technology/go-backend-platform" className="editorial-btn editorial-btn--sm">
            OPEN SOURCE STACK →
          </Link>
        </div>
      </div>

      <div
        ref={stackRef}
        className="border-none bg-black overflow-hidden"
        onPointerMove={handlePointerMove}
        onPointerLeave={handlePointerLeave}
      >
        <div ref={rowsRef} className="flex flex-col gap-2 py-6 md:py-8">
          {carouselRows.map((row, index) => (
            <div key={index} className="relative h-[96px] md:h-[108px]">
              <LogoLoop
                logos={row.logos}
                speed={50}
                direction={row.direction}
                logoHeight={56}
                gap={28}
                pauseOnHover={false}
                scaleOnHover
                fadeOut
                fadeOutColor="#000000"
                active={isInView}
                ariaLabel="Platform stack technologies"
              />
            </div>
          ))}
        </div>
      </div>
    </PageSection>
  );
}
