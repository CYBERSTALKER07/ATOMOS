'use client';

import Link from 'next/link';
import Image from 'next/image';
import { ROLES_DATA } from '@/app/data/rolesData';
import { EDITORIAL_IMAGES } from '@/app/components/ContentCard';
import { FLEET_TRUCK_IMAGES } from '@/app/lib/fleetAssets';
import { AxionPageLayout } from '@/app/components/fleek/axion';
import FleekPageShell from '@/app/components/fleek/FleekPageShell';
import FleekDataSection from '@/app/components/fleek/FleekDataSection';
import { mapTopicsToSolutions } from '@/app/data/axionSectionContent';
import { O9TourCTA } from '@/app/components/page-sections/o9/O9PageChrome';

function roleImage(index: number): string {
  return index % 2 === 0
    ? FLEET_TRUCK_IMAGES[index % FLEET_TRUCK_IMAGES.length].src
    : EDITORIAL_IMAGES[index % EDITORIAL_IMAGES.length];
}

export default function RolesPage() {
  const roleLinks = ROLES_DATA.map((r) => ({
    label: r.name,
    href: `/roles/${r.id}`,
    description: r.description,
  }));

  const industryItems = ROLES_DATA.map((role, i) => ({
    title: role.name,
    description: role.description,
    icon: (['retail', 'warehouse', 'manufacturing', 'fleet', 'health', 'tech'] as const)[i % 6],
    href: `/roles/${role.id}`,
    highlight: i === 3,
  }));

  return (
    <FleekPageShell activeHref="/roles">
      <AxionPageLayout
        hero={{
          title: "Six roles,\none order truth",
          summary:
            'Supplier, warehouse, factory, driver, retailer, and payload/gate — features mapped across portal, mobile, and desktop on one Spanner spine.',
          primaryHref: '/roles/supplier',
          primaryLabel: 'Learn More',
          imageSrc: EDITORIAL_IMAGES[0],
        }}
        solutions={{
          title: 'Role solutions',
          items: mapTopicsToSolutions(roleLinks, [...EDITORIAL_IMAGES]),
          seeAllHref: '/roles',
        }}
        industries={{
          eyebrow: '/ ROLES',
          title: 'Tailored logistics for every business role',
          items: industryItems,
        }}
        technology={{
          extra: <FleekDataSection hubId="roles" />,
        }}
        details={
          <div className="space-y-24">
            {ROLES_DATA.map((role, roleIndex) => (
              <section key={role.id} id={role.id} className="scroll-mt-28">
                <div className="mb-10 flex flex-col gap-4 border-b border-black/10 pb-8 md:flex-row md:items-end md:justify-between">
                  <div>
                    <p className="axion-eyebrow">{String(roleIndex + 1).padStart(2, '0')}</p>
                    <h2 className="axion-section__title mt-2">{role.name}</h2>
                    <p className="axion-section__subtitle mt-3 max-w-2xl">{role.description}</p>
                  </div>
                </div>
                <div className="grid gap-6 lg:grid-cols-2">
                  {role.subtopics.map((subtopic, i) => (
                    <article key={subtopic.id} className="axion-industry-card">
                      <div className="relative aspect-[16/9] overflow-hidden rounded-2xl mb-4">
                        <Image
                          src={roleImage(roleIndex + i)}
                          alt={subtopic.title}
                          fill
                          className="object-cover"
                          sizes="(max-width: 768px) 100vw, 50vw"
                        />
                      </div>
                      <h3 className="axion-industry-card__title">{subtopic.title}</h3>
                      <p className="axion-industry-card__desc">{subtopic.description}</p>
                      <Link href={`/roles/${role.id}`} className="axion-tech-feature__link mt-4">
                        Explore more
                        <span className="axion-tech-feature__arrow" aria-hidden>→</span>
                      </Link>
                    </article>
                  ))}
                </div>
              </section>
            ))}
            <O9TourCTA />
          </div>
        }
      />
    </FleekPageShell>
  );
}
