'use client';

import { useMemo, useRef } from 'react';
import Image from 'next/image';
import Link from 'next/link';
import { EDITORIAL_IMAGES } from './ContentCard';
import PageSection from './layout/PageSection';
import { useLanguage } from '../context/LanguageContext';

type HomeProjectCard = {
	title: string;
	description: string;
	tag: string;
	href: string;
};

const HOME_PROJECTS_EN: HomeProjectCard[] = [
	{
		title: 'Dispatch Engine',
		description:
			'Visual warehouse dispatching with smart truck-to-order matching, gate seals, and a live board for morning load-outs.',
		tag: 'Operations',
		href: '/projects/dispatch-engine',
	},
	{
		title: 'Supplier Control Plane',
		description:
			'Network oversight for suppliers — order vetting, dispatch preview, topology, and treasury across warehouses and retailers.',
		tag: 'Platform',
		href: '/projects/supplier-control-plane',
	},
	{
		title: 'Driver Execution App',
		description:
			'Native route execution with sealed manifests, stop-by-stop delivery, cash collection, and live progress reporting.',
		tag: 'Mobile',
		href: '/projects/driver-execution-app',
	},
	{
		title: 'Retailer Commerce',
		description:
			'Catalog, checkout, scheduling, and live order tracking — desktop and mobile parity for retailer teams.',
		tag: 'Commerce',
		href: '/projects/retailer-commerce',
	},
];

const HOME_PROJECTS_RU: HomeProjectCard[] = [
	{
		title: 'Движок диспетчеризации',
		description:
			'Визуальная диспетчеризация склада с умным подбором грузовиков и заказов, пломбами на воротах и живой доской для пиковых утренних загрузок.',
		tag: 'Операции',
		href: '/projects/dispatch-engine',
	},
	{
		title: 'Панель управления поставщика',
		description:
			'Контроль сети для поставщиков — проверка заказов, превью диспетчеризации, топология и казначейство по складам и ритейлерам.',
		tag: 'Платформа',
		href: '/projects/supplier-control-plane',
	},
	{
		title: 'Приложение водителя',
		description:
			'Нативное исполнение маршрута с пломбированными манифестами, доставкой по остановкам, сбором наличных и живым отчётом о прогрессе.',
		tag: 'Мобильные',
		href: '/projects/driver-execution-app',
	},
	{
		title: 'Коммерция для ритейлера',
		description:
			'Каталог, оформление, планирование и живое отслеживание заказов — паритет desktop и mobile для команд ритейлера.',
		tag: 'Коммерция',
		href: '/projects/retailer-commerce',
	},
];

export default function Projects() {
	const { language } = useLanguage();
	const projects = useMemo(
		() => (language === 'ru' ? HOME_PROJECTS_RU : HOME_PROJECTS_EN),
		[language]
	);
	const scrollRef = useRef<HTMLDivElement>(null);

	const scrollLeft = () => {
		if (scrollRef.current) {
			const card = scrollRef.current.querySelector('.module-card') as HTMLElement;
			const cardWidth = card?.clientWidth || 0;
			scrollRef.current.scrollBy({ left: -(cardWidth + 24), behavior: 'smooth' });
		}
	};

	const scrollRight = () => {
		if (scrollRef.current) {
			const card = scrollRef.current.querySelector('.module-card') as HTMLElement;
			const cardWidth = card?.clientWidth || 0;
			scrollRef.current.scrollBy({ left: cardWidth + 24, behavior: 'smooth' });
		}
	};

	return (
		<PageSection id="projects" className="py-24 bg-black" bleed={true}>
			<div className="relative w-full group">
				{/* Scroll Navigation Buttons */}
				<button
					onClick={scrollLeft}
					className="absolute left-2 sm:left-4 top-1/2 -translate-y-1/2 z-20 w-10 h-10 flex items-center justify-center rounded-full bg-black/40 hover:bg-black/80 border border-white/10 text-white backdrop-blur-sm transition-all opacity-0 group-hover:opacity-100 disabled:opacity-0"
					aria-label="Scroll left"
				>
					<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
						<path d="M15 18l-6-6 6-6" />
					</svg>
				</button>
				
				<button
					onClick={scrollRight}
					className="absolute right-2 sm:right-4 top-1/2 -translate-y-1/2 z-20 w-10 h-10 flex items-center justify-center rounded-full bg-black/40 hover:bg-black/80 border border-white/10 text-white backdrop-blur-sm transition-all opacity-0 group-hover:opacity-100 disabled:opacity-0"
					aria-label="Scroll right"
				>
					<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
						<path d="M9 18l6-6-6-6" />
					</svg>
				</button>

				{/* Scrollable Container */}
				<div 
					ref={scrollRef}
					className="flex flex-row overflow-x-auto scroll-smooth overscroll-x-contain snap-x snap-proximity gap-4 md:gap-6 pl-6 pr-12 md:pl-[100px] md:pr-24 w-full h-[650px] md:h-[750px] lg:h-[800px] [scrollbar-width:none] [-ms-overflow-style:none] [&::-webkit-scrollbar]:hidden"
				>
					{projects.map((project, index) => (
						<Link
							key={project.href}
							href={project.href}
							className="module-card relative rounded-[32px] overflow-hidden flex-shrink-0 snap-center bg-zinc-900 
										w-[90vw] md:w-[80vw] lg:w-[960px] block text-left"
						>
							{/* Blurred Image Background with Hover Reveal */}
							<Image
								src={EDITORIAL_IMAGES[index % EDITORIAL_IMAGES.length]}
								alt={project.title}
								fill
								className="module-card__image"
								sizes="(max-width: 768px) 90vw, (max-width: 1024px) 80vw, 960px"
							/>
							{/* Overlay gradient for text readability */}
							<div className="module-card__overlay" />

							{/* Content */}
							<div className="absolute inset-0 p-8 md:p-16 lg:p-20 flex flex-col justify-between">
								<div>
									<p className="text-white/80 font-mono text-xs md:text-sm uppercase tracking-[0.2em] mb-6">
										{project.tag}
									</p>
									<h3 className="text-white text-4xl md:text-5xl lg:text-[56px] font-medium leading-[1.1] max-w-2xl mb-8 tracking-tight">
										{project.title}
									</h3>
									<p className="text-white/90 text-lg md:text-[22px] max-w-2xl leading-[1.4]">
										{project.description}
									</p>
								</div>
								<div className="flex justify-center pb-4 md:pb-8">
									<span
										className="module-card__btn px-12 py-4 text-xs font-bold tracking-[0.15em] uppercase rounded-md inline-block"
									>
										MORE
									</span>
								</div>
							</div>
						</Link>
					))}
				</div>
			</div>
		</PageSection>
	);
}
