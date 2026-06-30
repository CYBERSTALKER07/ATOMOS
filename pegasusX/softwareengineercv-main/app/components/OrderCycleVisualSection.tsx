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

      <section className="border-t border-white/10 bg-black py-16 text-white">
        <div className="container mx-auto max-w-7xl px-4">
          <div className="flex flex-col gap-6 md:flex-row md:items-center md:justify-between">
            <div>
              <p className="editorial-eyebrow">Order cycle</p>
              <h2 className="text-2xl font-semibold tracking-tight md:text-3xl">
                Retailer → warehouse → gate → driver → payment
              </h2>
              <p className="mt-3 max-w-xl text-sm text-white/55">
                Line-art animation made with Gemini — hover to play with sound.
              </p>
            </div>
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
