'use client';

import OrderCycleVisualSection from '../OrderCycleVisualSection';

type OrderLifecycleVideoProps = {
  variant?: 'hero' | 'inline';
  showCaption?: boolean;
  hoverLabel?: string;
};

export default function OrderLifecycleVideo(_props: OrderLifecycleVideoProps) {
  return <OrderCycleVisualSection />;
}
