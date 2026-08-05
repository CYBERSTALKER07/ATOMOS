'use client';

import ChamferButton from '@/app/components/ChamferButton';

type CTABandProps = {
  relatedProjectSlug?: string;
  primaryLabel?: string;
  primaryHref?: string;
};

export default function CTABand({
  relatedProjectSlug,
  primaryLabel = 'Request demo',
  primaryHref = '/join',
}: CTABandProps) {
  return (
    <div className="mt-16 flex flex-col gap-3 border-t border-white/10 pt-12 sm:flex-row">
      {relatedProjectSlug ? (
        <ChamferButton href={`/projects/${relatedProjectSlug}`} variant="fill">
          View module
        </ChamferButton>
      ) : null}
      <ChamferButton href={primaryHref} variant="ghost">
        {primaryLabel}
      </ChamferButton>
    </div>
  );
}
