'use client';

import dynamic from 'next/dynamic';
import ChamferButton from './ChamferButton';
import { ORDER_LIFECYCLE_GEMINI_SHARE } from '@/app/lib/lifecycleAssets';

const OrderLifecycleVideo = dynamic(() => import('./lifecycle/OrderLifecycleVideo'), {
  ssr: false,
});

export default function OrderCycleVisualSection() {
  return (
    <>
      <OrderLifecycleVideo variant="hero" />

      <section className="bg-black py-12 text-white">
        <div className="page-shell">
          <div className="flex flex-col gap-6 md:flex-row md:items-center md:justify-between">

            <div className="flex flex-col gap-3 sm:flex-row">
              <ChamferButton href="/platform/order-lifecycle" variant="fill">
                View lifecycle
              </ChamferButton>
              <ChamferButton href={ORDER_LIFECYCLE_GEMINI_SHARE} variant="ghost">
                Gemini source ↗
              </ChamferButton>
            </div>
          </div>
        </div>
      </section>
    </>
  );
}
