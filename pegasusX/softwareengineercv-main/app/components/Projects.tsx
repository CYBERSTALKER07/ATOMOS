'use client';

import { useEffect, useMemo, useRef } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import Link from 'next/link';
import ContentCard, { EDITORIAL_IMAGES } from './ContentCard';
import { useIsMobile } from '../hooks/useDevice';
import PageSection from './layout/PageSection';
import SectionHeader from './layout/SectionHeader';
import { useLanguage } from '../context/LanguageContext';

gsap.registerPlugin(ScrollTrigger);

type HomeProjectCard = {
  title: string;
  description: string;
  tag: string;
  href: string;
  variant: 'featured' | 'vertical' | 'split';
  bento: string;
  tone?: 'light' | 'dark';
};

const HOME_PROJECTS_EN: HomeProjectCard[] = [
  {
    title: 'Dispatch Engine',
    description:
      'Visual warehouse dispatch with smart truck-and-order matching, gate seals, and live board updates for peak morning loads.',
    tag: 'Operations',
    href: '/projects/dispatch-engine',
    variant: 'featured',
    bento: 'editorial-bento__4-2',
  },
  {
    title: 'Supplier Control Plane',
    description:
      'Network oversight for suppliers — order vetting, dispatch preview, topology, and treasury across warehouses and retailers.',
    tag: 'Platform',
    href: '/projects/supplier-control-plane',
    variant: 'vertical',
    bento: 'editorial-bento__2-1',
  },
  {
    title: 'Driver Execution App',
    description:
      'Native route execution with sealed manifests, stop-by-stop delivery, cash collection, and live progress reporting.',
    tag: 'Mobile',
    href: '/projects/driver-execution-app',
    variant: 'split',
    bento: 'editorial-bento__2-2',
  },
  {
    title: 'Retailer Commerce',
    description:
      'Catalog, checkout, scheduling, and live order tracking — desktop and mobile parity for retailer teams.',
    tag: 'Commerce',
    href: '/projects/retailer-commerce',
    variant: 'split',
    bento: 'editorial-bento__4-1',
  },
  {
    title: 'Fleet Telemetry',
    description:
      'Live fleet map with planned-vs-actual routes, deviation alerts, and retailer self-serve tracking.',
    tag: 'Visibility',
    href: '/projects/fleet-telemetry',
    variant: 'vertical',
    bento: 'editorial-bento__2-1',
  },
  {
    title: 'Payment Integrity',
    description:
      'Checkout through driver collection to supplier treasury — duplicate protection and a clear audit trail.',
    tag: 'Finance',
    href: '/projects/payment-integrity',
    variant: 'vertical',
    tone: 'light',
    bento: 'editorial-bento__2-1',
  },
];

const HOME_PROJECTS_RU: HomeProjectCard[] = [
  {
    title: 'Движок диспетчеризации',
    description:
      'Визуальная диспетчеризация склада с умным подбором грузовиков и заказов, пломбами на воротах и живой доской для пиковых утренних загрузок.',
    tag: 'Операции',
    href: '/projects/dispatch-engine',
    variant: 'featured',
    bento: 'editorial-bento__4-2',
  },
  {
    title: 'Панель управления поставщика',
    description:
      'Контроль сети для поставщиков — проверка заказов, превью диспетчеризации, топология и казначейство по складам и ритейлерам.',
    tag: 'Платформа',
    href: '/projects/supplier-control-plane',
    variant: 'vertical',
    bento: 'editorial-bento__2-1',
  },
  {
    title: 'Приложение водителя',
    description:
      'Нативное исполнение маршрута с пломбированными манифестами, доставкой по остановкам, сбором наличных и живым отчётом о прогрессе.',
    tag: 'Мобильные',
    href: '/projects/driver-execution-app',
    variant: 'split',
    bento: 'editorial-bento__2-2',
  },
  {
    title: 'Коммерция для ритейлера',
    description:
      'Каталог, оформление, планирование и живое отслеживание заказов — паритет desktop и mobile для команд ритейлера.',
    tag: 'Коммерция',
    href: '/projects/retailer-commerce',
    variant: 'split',
    bento: 'editorial-bento__4-1',
  },
  {
    title: 'Телеметрия автопарка',
    description:
      'Живая карта автопарка с маршрутами «план vs факт», алертами по отклонениям и самообслуживанием отслеживания для ритейлера.',
    tag: 'Видимость',
    href: '/projects/fleet-telemetry',
    variant: 'vertical',
    bento: 'editorial-bento__2-1',
  },
  {
    title: 'Целостность платежей',
    description:
      'От оформления через сбор водителем до казначейства поставщика — защита от дублей и понятный аудит-след.',
    tag: 'Финансы',
    href: '/projects/payment-integrity',
    variant: 'vertical',
    tone: 'light',
    bento: 'editorial-bento__2-1',
  },
];

export default function Projects() {
  const { isMobile } = useIsMobile();
  const { t, language } = useLanguage();
  const sectionRef = useRef<HTMLElement>(null);
  const titleRef = useRef<HTMLDivElement>(null);
  const gridRef = useRef<HTMLDivElement>(null);
  const projects = useMemo(
    () => (language === 'ru' ? HOME_PROJECTS_RU : HOME_PROJECTS_EN),
    [language]
  );

  useEffect(() => {
    if (!sectionRef.current) return;

    if (isMobile) {
      gsap.set([titleRef.current, gridRef.current], { opacity: 1, y: 0 });
      return;
    }

    const timeline = gsap.timeline({
      scrollTrigger: {
        trigger: sectionRef.current,
        start: 'top 80%',
        end: 'bottom 20%',
        toggleActions: 'play none none reverse',
      },
    });

    timeline
      .fromTo(titleRef.current, { opacity: 0, y: 30 }, { opacity: 1, y: 0, duration: 0.8 })
      .fromTo(
        gridRef.current?.children ? Array.from(gridRef.current.children) : [],
        { opacity: 0, y: 40 },
        { opacity: 1, y: 0, duration: 0.55, stagger: 0.08 },
        '-=0.35'
      );
  }, [isMobile]);

  return (
    <PageSection ref={sectionRef} id="projects">
      <div ref={titleRef}>
        <SectionHeader
          align="center"
          title={t('home_modules_title', 'Platform Modules')}
          description={t(
            'home_modules_desc',
            'Core modules that power supplier-led logistics from dispatch to delivery'
          )}
        />
      </div>

      <div ref={gridRef} className="editorial-bento">
        {projects.map((project, index) => (
          <ContentCard
            key={project.href}
            variant={project.variant}
            tone={project.tone ?? 'dark'}
            tag={project.tag}
            title={project.title}
            description={project.description}
            image={EDITORIAL_IMAGES[index % EDITORIAL_IMAGES.length]}
            href={project.href}
            ctaLabel={t('btn_read_more', 'READ MORE')}
            ctaStyle="link"
            className={project.bento}
            imagePriority={index === 0}
          />
        ))}
      </div>

      <div className="text-center mt-12">
        <Link href="/projects" className="editorial-btn">
          {t('btn_view_all_modules', 'VIEW ALL MODULES')}
        </Link>
      </div>
    </PageSection>
  );
}
