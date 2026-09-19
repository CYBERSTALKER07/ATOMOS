'use client';

import dynamic from 'next/dynamic';

const OrderLifecycleVideo = dynamic(() => import('./lifecycle/OrderLifecycleVideo'), {
  ssr: false,
});

export default function OrderCycleVisualSection() {
  return (
    <>
      <OrderLifecycleVideo variant="hero" />
    </>
  );
}
