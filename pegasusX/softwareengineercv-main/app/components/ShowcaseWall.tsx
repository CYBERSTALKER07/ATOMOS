'use client';

import DriftWall, { DriftWallItem } from './DriftWall';
import { useLanguage } from '../context/LanguageContext';
import PageSection from './layout/PageSection';

const DRIFT_ITEMS: DriftWallItem[] = [
  { image: 'https://images.unsplash.com/photo-1586528116311-ad8dd3c8310d?auto=format&fit=crop&w=800&q=80', title: 'Smart Dispatch Hub', href: '/platform' },
  { image: 'https://images.unsplash.com/photo-1618005182384-a83a8bd57fbe?auto=format&fit=crop&w=800&q=80', title: 'Telemetry Engine', href: '/capabilities' },
  { image: 'https://images.unsplash.com/photo-1578575437130-527eed3abbec?auto=format&fit=crop&w=800&q=80', title: 'Fleet Tracking', href: '/capabilities/live-fleet-tracking' },
  { image: 'https://images.unsplash.com/photo-1565891741441-64926e441838?auto=format&fit=crop&w=800&q=80', title: 'Warehouse Controls', href: '/roles/warehouse' },
  { image: 'https://images.unsplash.com/photo-1504384308090-c894fdcc538d?auto=format&fit=crop&w=800&q=80', title: 'Supplier Network', href: '/roles/supplier' },
  { image: 'https://images.unsplash.com/photo-1551288049-bebda4e38f71?auto=format&fit=crop&w=800&q=80', title: 'Real-time Analytics', href: '/operations' },
  { image: 'https://images.unsplash.com/photo-1600880292203-757bb62b4baf?auto=format&fit=crop&w=800&q=80', title: 'Fulfillment Control', href: '/platform' },
  { image: 'https://images.unsplash.com/photo-1519389950473-47ba0277781c?auto=format&fit=crop&w=800&q=80', title: 'Payment Confidence', href: '/capabilities/payment-confidence' },
  { image: 'https://images.unsplash.com/photo-1451187580459-43490279c0fa?auto=format&fit=crop&w=800&q=80', title: 'Global Operations', href: '/projects' },
  { image: 'https://images.unsplash.com/photo-1526374965328-7f61d4dc18c5?auto=format&fit=crop&w=800&q=80', title: 'Encrypted State Machine', href: '/technology' },
  { image: 'https://images.unsplash.com/photo-1581091226825-a6a2a5aee158?auto=format&fit=crop&w=800&q=80', title: 'Factory Dispatch Gate', href: '/roles/payload-gate' },
  { image: 'https://images.unsplash.com/photo-1512758017271-d7b84c2113f1?auto=format&fit=crop&w=800&q=80', title: 'Retailer Dashboard', href: '/demo/retailer' },
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
