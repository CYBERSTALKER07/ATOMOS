'use client';

import Image from 'next/image';
import { useLanguage } from '@/app/context/LanguageContext';
import { SITE_IMAGES } from '@/app/lib/siteAssets';

type OrderLifecycleVideoProps = {
  variant?: 'hero' | 'inline';
  showCaption?: boolean;
  hoverLabel?: string;
};

export default function OrderLifecycleVideo({
  variant = 'hero',
}: OrderLifecycleVideoProps) {
  const { t } = useLanguage();

  const frameClass =
    variant === 'hero'
      ? 'order-lifecycle-video order-lifecycle-video--hero'
      : 'order-lifecycle-video order-lifecycle-video--inline';

  return (
    <section className={frameClass} aria-label="Pegasus Logistics Infrastructure">
      <div className="order-lifecycle-video__pin">
        <div className="relative w-full overflow-hidden bg-black">
          <div className="editorial-card__media order-lifecycle-video__media relative aspect-[16/9] w-full overflow-hidden">
            <Image
              src={SITE_IMAGES.geometricTerminal}
              alt={t(
                'geometric_terminal_alt',
                'Pegasus autonomous logistics terminal and geometric distribution hub'
              )}
              fill
              className="object-cover"
              sizes="(max-width: 768px) 100vw, (max-width: 1400px) 95vw, 1600px"
              priority
            />
          </div>
        </div>
      </div>
    </section>
  );
}
