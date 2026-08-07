'use client';

import type { ReactNode } from 'react';
import dynamic from 'next/dynamic';
import FleekPageShell from '@/app/components/fleek/FleekPageShell';
import { usePerfProfile } from '@/app/hooks/useDevice';

const Lanyard = dynamic(() => import('@/app/components/Lanyard'), { ssr: false });

type FormLanyardPageProps = {
  activeHref: string;
  title: string;
  subtitle?: string;
  children: ReactNode;
};

export default function FormLanyardPage({
  activeHref,
  title,
  subtitle,
  children,
}: FormLanyardPageProps) {
  const { allowHeavyFx } = usePerfProfile();

  return (
    <FleekPageShell activeHref={activeHref}>
      <div className="relative grid min-h-[calc(100svh-4.5rem)] lg:min-h-[calc(100svh-5rem)] lg:grid-cols-2">
        <aside
          className="relative h-[42vh] min-h-[280px] overflow-hidden border-b border-white/10 bg-black lg:h-auto lg:min-h-full lg:border-b-0 lg:border-r"
          aria-hidden={!allowHeavyFx}
        >
          {allowHeavyFx ? (
            <div className="absolute inset-0">
              <Lanyard position={[0, 0, 22]} gravity={[0, -40, 0]} fov={18} transparent />
            </div>
          ) : (
            <div className="flex h-full items-center justify-center p-8">
              <img
                src="/pegasus.jpg"
                alt=""
                className="max-h-[70%] max-w-[min(280px,70%)] object-contain opacity-90"
              />
            </div>
          )}
        </aside>

        <section className="flex items-center px-5 py-10 sm:px-8 md:px-12 lg:py-16">
          <div className="mx-auto w-full max-w-lg">
            <h1 className="font-title text-3xl font-semibold tracking-tight sm:text-4xl">{title}</h1>
            {subtitle ? <p className="mt-3 text-sm leading-relaxed text-white/55 sm:text-base">{subtitle}</p> : null}
            <div className="mt-8">{children}</div>
          </div>
        </section>
      </div>
    </FleekPageShell>
  );
}
