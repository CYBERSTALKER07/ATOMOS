'use client';

import Link from 'next/link';
import Image from 'next/image';
import { getRolesData } from '@/app/data/rolesData';
import { EDITORIAL_IMAGES } from '@/app/components/ContentCard';
import { FLEET_TRUCK_IMAGES } from '@/app/lib/fleetAssets';
import { O9FleekPageLayout } from '@/app/components/fleek/o9';
import FleekPageShell from '@/app/components/fleek/FleekPageShell';
import { DEFAULT_PROOF } from '@/app/data/topicContent/helpers';
import { useLanguage } from '@/app/context/LanguageContext';

function roleImage(index: number): string {
  return index % 2 === 0
    ? FLEET_TRUCK_IMAGES[index % FLEET_TRUCK_IMAGES.length].src
    : EDITORIAL_IMAGES[index % EDITORIAL_IMAGES.length];
}

export default function RolesPage() {
  const { language, t } = useLanguage();
  const rolesData = getRolesData(language);

  const capabilities = rolesData.map((role, i) => ({
    title: role.name,
    description: role.description,
    href: `/roles/${role.id}`,
    image: roleImage(i),
    tag: t('nav_roles', 'Role'),
  }));

  const differentiators = rolesData.slice(0, 4).map((role) => ({
    title: role.name,
    description: role.description,
  }));

  return (
    <FleekPageShell activeHref="/roles">
      <O9FleekPageLayout
        categoryLabel={t('nav_roles', 'Roles')}
        categoryHref="/roles"
        title={t('roles_hero_title', 'Six roles, one order truth')}
        summary={t('roles_hero_summary', 'Supplier, warehouse, factory, driver, retailer, and payload/gate — features mapped across portal, mobile, and desktop on one shared order record.')}
        heroImageSrc={EDITORIAL_IMAGES[0]}
        proofItems={DEFAULT_PROOF}
        hubId="roles"
        differentiators={differentiators}
        differentiatorsTitle={t('roles_diff_title', 'Tailored logistics for every business role')}
        capabilities={capabilities}
        capabilitiesTitle={t('roles_cap_title', 'Role solutions')}
        showTourCta
        details={
          <div className="space-y-16">
            {rolesData.map((role, roleIndex) => (
              <section key={role.id} id={role.id} className="scroll-mt-28 docs-section">
                <p className="font-mono text-[10px] uppercase tracking-[0.22em] text-white/45">
                  {String(roleIndex + 1).padStart(2, '0')}
                </p>
                <h2 className="mt-3 text-3xl font-semibold tracking-tight md:text-4xl">{role.name}</h2>
                <p className="mt-4 max-w-2xl text-base leading-relaxed text-white/70">{role.description}</p>
                <div className="mt-8 grid gap-4 md:grid-cols-2">
                  {role.subtopics.map((subtopic, i) => (
                    <article key={subtopic.id} className="o9-card overflow-hidden">
                      <div className="relative aspect-[16/9]">
                        <Image
                          src={roleImage(roleIndex + i)}
                          alt={subtopic.title}
                          fill
                          className="object-cover"
                          sizes="(max-width: 768px) 100vw, 50vw"
                        />
                      </div>
                      <div className="p-5">
                        <h3 className="text-lg font-semibold">{subtopic.title}</h3>
                        <p className="mt-2 text-sm leading-relaxed text-white/65">{subtopic.description}</p>
                        <Link href={`/roles/${role.id}`} className="o9-btn o9-btn--fill mt-4">
                          {t('btn_read_more', 'Explore more')}
                        </Link>
                      </div>
                    </article>
                  ))}
                </div>
              </section>
            ))}
          </div>
        }
      />
    </FleekPageShell>
  );
}
