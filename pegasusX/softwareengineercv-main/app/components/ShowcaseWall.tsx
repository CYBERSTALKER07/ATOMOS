'use client';

import DriftWall, { DriftWallItem } from './DriftWall';
import { useLanguage } from '../context/LanguageContext';
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
  const { t } = useLanguage();

  return (
    <PageSection className="bg-[#030303] py-20 border-t border-white/5 relative overflow-hidden">
      <div className="text-center mb-12 relative z-10 px-4">
        <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full border border-emerald-500/20 bg-emerald-950/30 text-emerald-400 text-xs font-mono mb-4">
          <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
          ECOSYSTEM VISUAL MATRIX
        </div>
        <h2 className="text-4xl md:text-6xl font-medium tracking-tight text-white mb-4">
          {t('showcase_title', '3D Interactive Ecosystem Wall')}
        </h2>
        <p className="text-white/60 text-base md:text-lg max-w-2xl mx-auto font-light">
          {t('showcase_subtitle', 'Explore real-time telemetry, dispatch boards, and multi-role operations floating in 3D parallax.')}
        </p>
      </div>

      <div className="w-full h-[540px] md:h-[640px] relative z-10 rounded-3xl border border-white/10 overflow-hidden bg-black/60 shadow-[0_30px_100px_rgba(0,0,0,0.9)]">
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
