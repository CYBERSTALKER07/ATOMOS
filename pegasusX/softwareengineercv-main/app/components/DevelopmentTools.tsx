'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import LogoLoop from './LogoLoop';
import { useInView } from '../hooks/useInView';
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
  SiFastapi
} from 'react-icons/si';
import { VscCode } from 'react-icons/vsc';
import { FaAws } from 'react-icons/fa6';

gsap.registerPlugin(ScrollTrigger);

export default function DevelopmentTools() {
  const { ref: sectionRef, isInView } = useInView<HTMLElement>({ rootMargin: '0px' });
  const titleRef = useRef<HTMLDivElement>(null);
  const row1Ref = useRef<HTMLDivElement>(null);
  const row2Ref = useRef<HTMLDivElement>(null);
  const row3Ref = useRef<HTMLDivElement>(null);
  const row4Ref = useRef<HTMLDivElement>(null);
  const row5Ref = useRef<HTMLDivElement>(null);

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

      const rows = [row1Ref.current, row2Ref.current, row3Ref.current, row4Ref.current, row5Ref.current];
      timeline.fromTo(
        rows,
        { opacity: 0, y: 50 },
        { opacity: 1, y: 0, duration: 0.6, stagger: 0.15 },
        '-=0.4'
      );
    }, sectionRef);

    return () => ctx.revert();
  }, []);

  const frontendLogos = [
    { node: <SiReact className="text-white" />, title: "React", href: "https://react.dev" },
    { node: <SiNextdotjs className="text-white" />, title: "Next.js", href: "https://nextjs.org" },
    { node: <SiVuedotjs className="text-white" />, title: "Vue.js", href: "https://vuejs.org" },
    { node: <SiSvelte className="text-white" />, title: "Svelte", href: "https://svelte.dev" },
    { node: <SiAngular className="text-white" />, title: "Angular", href: "https://angular.io" },
    { node: <SiTypescript className="text-white" />, title: "TypeScript", href: "https://www.typescriptlang.org" },
    { node: <SiTailwindcss className="text-white" />, title: "Tailwind CSS", href: "https://tailwindcss.com" },
  ];

  const backendLogos = [
    { node: <SiNodedotjs className="text-white" />, title: "Node.js", href: "https://nodejs.org" },
    { node: <SiExpress className="text-white" />, title: "Express", href: "https://expressjs.com" },
    { node: <SiGraphql className="text-white" />, title: "GraphQL", href: "https://graphql.org" },
    { node: <SiSocketdotio className="text-white" />, title: "Socket.io", href: "https://socket.io" },
    { node: <SiPython className="text-white" />, title: "Python", href: "https://python.org" },
    { node: <SiDjango className="text-white" />, title: "Django", href: "https://djangoproject.com" },
    { node: <SiFastapi className="text-white" />, title: "FastAPI", href: "https://fastapi.tiangolo.com" },
  ];

  const databaseLogos = [
    { node: <SiPostgresql className="text-white" />, title: "PostgreSQL", href: "https://postgresql.org" },
    { node: <SiMongodb className="text-white" />, title: "MongoDB", href: "https://mongodb.com" },
    { node: <SiRedis className="text-white" />, title: "Redis", href: "https://redis.io" },
    { node: <SiFirebase className="text-white" />, title: "Firebase", href: "https://firebase.google.com" },
    { node: <SiSupabase className="text-white" />, title: "Supabase", href: "https://supabase.com" },
  ];

  const devopsLogos = [
    { node: <SiDocker className="text-white" />, title: "Docker", href: "https://docker.com" },
    { node: <FaAws className="text-white" />, title: "AWS", href: "https://aws.amazon.com" },
    { node: <SiVercel className="text-white" />, title: "Vercel", href: "https://vercel.com" },
    { node: <SiGithubactions className="text-white" />, title: "GitHub Actions", href: "https://github.com/features/actions" },
    { node: <SiKubernetes className="text-white" />, title: "Kubernetes", href: "https://kubernetes.io" },
  ];

  const toolsLogos = [
    { node: <SiGit className="text-white" />, title: "Git", href: "https://git-scm.com" },
    { node: <VscCode className="text-white" />, title: "VS Code", href: "https://code.visualstudio.com" },
    { node: <SiFigma className="text-white" />, title: "Figma", href: "https://figma.com" },
    { node: <SiPostman className="text-white" />, title: "Postman", href: "https://postman.com" },
    { node: <SiJest className="text-white" />, title: "Jest", href: "https://jestjs.io" },
  ];

  const toolCategories = [
    { title: 'Client Applications', logos: frontendLogos, direction: 'left' as const },
    { title: 'Platform Services', logos: backendLogos, direction: 'right' as const },
    { title: 'Data & Events', logos: databaseLogos, direction: 'left' as const },
    { title: 'Infrastructure', logos: devopsLogos, direction: 'right' as const },
    { title: 'Operations Toolkit', logos: toolsLogos, direction: 'left' as const },
  ];

  const rowRefs = [row1Ref, row2Ref, row3Ref, row4Ref, row5Ref];

  return (
    <section ref={sectionRef} className="py-20 bg-white text-black" id="tools">
      <div className="container mx-auto px-4">
        <div ref={titleRef} className="text-center mb-16">
          <h2 className="text-4xl md:text-5xl lg:text-6xl font-bold mb-4 text-black">
            Platform Stack
          </h2>
          <div className="w-20 h-1 bg-black rounded-full mx-auto mb-6" />
          <p className="text-lg md:text-xl text-black max-w-2xl mx-auto">
            The technologies powering Pegasus across web, mobile, and backend services
          </p>
        </div>

        <div className="max-w-7xl mx-auto space-y-12">
          {toolCategories.map((category, index) => (
            <div
              key={index}
              ref={rowRefs[index]}
              className="group"
            >
              <div className="bg-black border border-black overflow-hidden shadow-lg p-8 md:p-10">
                <h3 className="text-2xl md:text-3xl font-bold mb-8 text-center text-white">
                  {category.title}
                </h3>
                
                <div className="relative h-[120px] md:h-[140px]">
                  <LogoLoop
                    logos={category.logos}
                    speed={50}
                    direction={category.direction}
                    logoHeight={64}
                    gap={60}
                    pauseOnHover
                    scaleOnHover
                    fadeOut
                    fadeOutColor="#000000"
                    active={isInView}
                    ariaLabel={`${category.title} logos`}
                  />
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
