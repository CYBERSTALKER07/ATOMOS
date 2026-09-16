'use client';

import DriftWall, { DriftWallItem } from './DriftWall';

import PageSection from './layout/PageSection';
import { SITE_IMAGES } from '@/app/lib/siteAssets';

const DRIFT_ITEMS: DriftWallItem[] = [
  { image: SITE_IMAGES.truckTerminal, title: 'Smart Dispatch Hub', href: '/platform' },
  { image: SITE_IMAGES.logisticsPlatformUi, title: 'Operations Board', href: '/capabilities' },
  { image: SITE_IMAGES.multimodalHub, title: 'Fleet Tracking', href: '/capabilities/live-fleet-tracking' },
  { image: SITE_IMAGES.warehouseAutomation, title: 'Warehouse Controls', href: '/roles/warehouse' },
  { image: SITE_IMAGES.pegasusContainer, title: 'Supplier Network', href: '/roles/supplier' },
  { image: SITE_IMAGES.operationsTeam, title: 'Live Analytics', href: '/operations' },
  { image: SITE_IMAGES.warehouseWireframe, title: 'Fulfillment Control', href: '/platform' },
  { image: SITE_IMAGES.deliveryDrone, title: 'Payment Confidence', href: '/capabilities/payment-confidence' },
  { image: SITE_IMAGES.containerShip, title: 'Global Operations', href: '/projects' },
  { image: SITE_IMAGES.terminalArchitecture, title: 'Secure Platform', href: '/technology' },
  { image: SITE_IMAGES.portCraneScene, title: 'Factory Dispatch Gate', href: '/roles/payload-gate' },
  { image: SITE_IMAGES.lastMileDelivery, title: 'Retailer Dashboard', href: '/demo/retailer' },
  { image: SITE_IMAGES.fleekHeroNew, title: 'Network Overview', href: '/platform' },
  { image: SITE_IMAGES.truckTerminal, title: 'Gate-ready Fleet', href: '/capabilities/live-fleet-tracking' },
  { image: SITE_IMAGES.warehouseAutomation, title: 'Yard Flow', href: '/roles/warehouse' },
];

export default function ShowcaseWall() {
  return (
    <PageSection bleed className="bg-[#030303] py-12 md:py-16 relative overflow-hidden border-none">
      <div className="w-full h-[320px] sm:h-[540px] md:h-[640px] relative z-10 overflow-hidden">
        <DriftWall
          items={DRIFT_ITEMS}
          columns={5}
          tileWidth={230}
          tileHeight={145}
          gap={20}
          radius={14}
          tilt={18}
          turn={-12}
          perspective={1100}
          depth={100}
          speed={38}
          direction="up"
          variance={0.4}
          parallax={0.7}
          lift={70}
          fade={0.65}
          dim={0.55}
          overlayColor="#020202"
        />
      </div>
    </PageSection>
  );
}
