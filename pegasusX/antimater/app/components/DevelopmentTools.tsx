'use client';

import { useCallback, useEffect, useRef, type ReactNode } from 'react';
import { gsap } from '@/app/lib/gsap';
import LogoLoop, { type LogoItem } from './LogoLoop';
import { useInView } from '../hooks/useInView';
import { usePerfProfile } from '../hooks/useDevice';
import PageSection from './layout/PageSection';
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
import { useLanguage } from '../context/LanguageContext';

function AppIconWrapper({ children, brandColor }: { children: ReactNode; brandColor: string }) {
	// If brandColor is very light (like #FFFFFF for Next.js), we use a dark background
	const isLight = brandColor === '#FFFFFF' || brandColor === '#fff';
	const bgColor = isLight ? '#000000' : brandColor;
	const iconColor = isLight ? '#FFFFFF' : '#FFFFFF';

	return (
		<div 
			className="flex items-center justify-center rounded-[2rem] shrink-0" 
			style={{ 
				backgroundColor: bgColor, 
				color: iconColor,
				width: '120px',
				height: '120px'
			}}
		>
			<div style={{ width: '64px', height: '64px' }} className="flex items-center justify-center *:w-full *:h-full">
				{children}
			</div>
		</div>
	);
}

function icon(node: ReactNode, brandColor: string, title: string, href: string): LogoItem {
	return {
		node: <AppIconWrapper brandColor={brandColor}>{node}</AppIconWrapper>,
		brandColor,
		title,
		href,
		logoClassName: 'hover:scale-105 transition-transform duration-300',
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
			icon(<SiFigma />, '#F24E1E', 'Figma', 'https://figma.com'),
			icon(<SiPostman />, '#FF6C37', 'Postman', 'https://postman.com'),
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
			icon(<SiDjango />, '#0C4B33', 'Django', 'https://djangoproject.com'),
			icon(<SiFastapi />, '#009688', 'FastAPI', 'https://fastapi.tiangolo.com'),
			icon(<SiJest />, '#C21325', 'Jest', 'https://jestjs.io'),
			icon(<VscCode />, '#007ACC', 'VS Code', 'https://code.visualstudio.com'),
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
			icon(<SiDocker />, '#2496ED', 'Docker', 'https://docker.com'),
			icon(<FaAws />, '#FF9900', 'AWS', 'https://aws.amazon.com'),
			icon(<SiVercel />, '#FFFFFF', 'Vercel', 'https://vercel.com'),
			icon(<SiGithubactions />, '#2088FF', 'GitHub Actions', 'https://github.com/features/actions'),
			icon(<SiKubernetes />, '#326CE5', 'Kubernetes', 'https://kubernetes.io'),
			icon(<SiGit />, '#F05032', 'Git', 'https://git-scm.com'),
		],
	}
];

export default function DevelopmentTools() {
	const { t } = useLanguage();
	const { isMobile, isLowEnd, prefersReducedMotion } = usePerfProfile();
	const { ref: sectionRef, isInView } = useInView<HTMLElement>({ rootMargin: '0px' });
	const rowsRef = useRef<HTMLDivElement>(null);

	useEffect(() => {
		if (!sectionRef.current) return;

		if (isMobile || isLowEnd || prefersReducedMotion) {
			if (rowsRef.current) gsap.set(rowsRef.current, { opacity: 1, y: 0 });
			return;
		}

		const ctx = gsap.context(() => {
			const timeline = gsap.timeline({
				scrollTrigger: {
					trigger: sectionRef.current,
					start: 'top 80%',
					end: 'bottom 20%',
					toggleActions: 'play none none reverse',
					fastScrollEnd: true,
				},
			});

			timeline.fromTo(
				rowsRef.current,
				{ opacity: 0, y: 40 },
				{ opacity: 1, y: 0, duration: 0.7, ease: 'pegasus' },
				'-=0.45'
			);
		}, sectionRef);

		return () => ctx.revert();
	}, [sectionRef, isMobile, isLowEnd, prefersReducedMotion]);

	return (
		<PageSection ref={sectionRef} id="tools" className="bg-black overflow-hidden py-16 md:py-24">
			<div ref={rowsRef} className="flex flex-col gap-4">
				{carouselRows.map((row, index) => (
					<div key={index} className="relative h-[120px]">
						<LogoLoop
							logos={row.logos}
							speed={40}
							direction={row.direction}
							logoHeight={120}
							gap={16}
							pauseOnHover={false}
							scaleOnHover={false}
							fadeOut={true}
							fadeOutColor="#000000"
							active={isInView}
							ariaLabel={t('tools_aria_stack', 'Platform stack technologies')}
						/>
					</div>
				))}
			</div>
		</PageSection>
	);
}
