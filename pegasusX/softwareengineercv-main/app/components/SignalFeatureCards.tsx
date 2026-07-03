'use client';

import Link from 'next/link';
import ChamferButton, { ChamferArrowIcon } from './ChamferButton';
import PageSection from './layout/PageSection';
import SectionHeader from './layout/SectionHeader';
import MapPinIcon from './icons/MapPinIcon';

const CARDS = [
  {
    title: 'Unified signal, zero noise',
    description:
      'One order lifecycle every role agrees on — from retailer checkout through vetting, dispatch, and delivery.',
    meta: 'Platform · Order Lifecycle',
    href: '/platform/order-lifecycle',
    ascii: '010\n101\n001',
    icon: 'cycle',
    from: 'Retailer checkout',
    to: 'Delivery stop',
  },
  {
    title: 'Dispatch you can trust',
    description:
      'Visual boards, gate seals, and capacity checks — warehouse leads confirm every load before wheels roll.',
    meta: 'Solutions · Dispatch Engine',
    href: '/solutions/visual-dispatch-engine',
    ascii: '110\n011\n100',
    icon: 'dispatch',
    from: 'Warehouse gate',
    to: 'Route manifest',
  },
  {
    title: 'Fleet truth on the map',
    description:
      'Loss-tolerant telemetry, and honest ETAs — ops and retailers see the same picture.',
    meta: 'Solutions · Fleet Telemetry',
    href: '/solutions/fleet-visibility',
    ascii: '001\n110\n010',
    icon: 'fleet',
    from: 'Depot yard',
    to: 'Live ETA',
  },
] as const;

function RouteEndpoints({ from, to }: { from: string; to: string }) {
  return (
    <div className="chamfer-card__route" aria-label={`Route from ${from} to ${to}`}>
      <div className="chamfer-card__route-end">
        <MapPinIcon size={18} className="chamfer-card__route-pin chamfer-card__route-pin--from" title="From" />
        <div className="chamfer-card__route-copy">
          <span className="chamfer-card__route-label">From</span>
          <span className="chamfer-card__route-place">{from}</span>
        </div>
      </div>
      <div className="chamfer-card__route-track" aria-hidden />
      <div className="chamfer-card__route-end chamfer-card__route-end--to">
        <MapPinIcon size={18} className="chamfer-card__route-pin chamfer-card__route-pin--to" title="To" />
        <div className="chamfer-card__route-copy">
          <span className="chamfer-card__route-label">To</span>
          <span className="chamfer-card__route-place">{to}</span>
        </div>
      </div>
    </div>
  );
}

function LineIcon({ type }: { type: string }) {
  if (type === 'cycle') {
    return (
      <svg width="72" height="72" viewBox="0 0 72 72" fill="none" aria-hidden>
        <circle cx="36" cy="36" r="28" stroke="currentColor" strokeWidth="1" />
        <path d="M36 14v8M36 50v8M14 36h8M50 36h8" stroke="currentColor" strokeWidth="1" />
        <path d="M28 28l16 16M44 28L28 44" stroke="currentColor" strokeWidth="1" opacity="0.5" />
      </svg>
    );
  }
  if (type === 'dispatch') {
    return (
      <svg width="72" height="72" viewBox="0 0 72 72" fill="none" aria-hidden>
        <rect x="12" y="28" width="48" height="24" stroke="currentColor" strokeWidth="1" />
        <path d="M18 52h36M22 28V20h28v8" stroke="currentColor" strokeWidth="1" />
        <rect x="20" y="32" width="10" height="8" stroke="currentColor" strokeWidth="1" />
        <rect x="34" y="32" width="10" height="8" stroke="currentColor" strokeWidth="1" />
      </svg>
    );
  }
  return (
    <svg width="72" height="72" viewBox="0 0 72 72" fill="none" aria-hidden>
      <path d="M12 48 L36 20 L60 48" stroke="currentColor" strokeWidth="1" fill="none" />
      <circle cx="36" cy="44" r="4" stroke="currentColor" strokeWidth="1" />
      <path d="M20 52h32" stroke="currentColor" strokeWidth="1" strokeDasharray="4 3" />
    </svg>
  );
}

export default function SignalFeatureCards() {
  return (
    <PageSection aria-labelledby="signal-features-heading">
      <div className="mb-12 flex flex-col gap-8 lg:flex-row lg:items-end lg:justify-between">
        <SectionHeader
          eyebrow="Capabilities"
          title="Signal over noise"
          titleId="signal-features-heading"
          description="Three outcomes every network needs — clearer operations, fewer surprises, faster decisions."
          className="mb-0"
        />
        <div className="flex shrink-0 flex-col gap-3 sm:flex-row lg:pb-1">
          <ChamferButton href="/join" variant="fill">
            Talk to us
          </ChamferButton>
          <ChamferButton href="/platform" variant="ghost">
            Get started
          </ChamferButton>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
          {CARDS.map((card) => (
            <Link key={card.href} href={card.href} className="chamfer-card p-6 md:p-8">
              <div className="flex items-start justify-between gap-4">
                <h3 className="text-lg font-semibold leading-snug md:text-xl">{card.title}</h3>
                <span className="chamfer-card__ascii hidden sm:block" aria-hidden>
                  {card.ascii}
                </span>
              </div>
              <div className="chamfer-card__icon my-6">
                <LineIcon type={card.icon} />
              </div>
              <p className="text-sm leading-relaxed text-white/55">{card.description}</p>
              <div className="chamfer-card__bottom">
                <RouteEndpoints from={card.from} to={card.to} />
                <div className="chamfer-card__footer chamfer-card__footer--route">
                  <span className="chamfer-card__meta">{card.meta}</span>
                  <span className="chamfer-btn chamfer-btn--ghost chamfer-btn--icon" aria-hidden>
                    <ChamferArrowIcon />
                  </span>
                </div>
              </div>
            </Link>
          ))}
        </div>
    </PageSection>
  );
}
