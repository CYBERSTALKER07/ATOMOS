import AppsFamilyPage from '@/app/components/apps/AppsFamilyPage';
import { EDITORIAL_IMAGES } from '@/app/components/apps/AppsFamilyPage';
import { SITE_IMAGES } from '@/app/lib/siteAssets';

export default function WebAppsPage() {
  return (
    <AppsFamilyPage
      config={{
        surface: 'web',
        title: 'Operations Portals',
        subtitle:
          'Web portals for suppliers, warehouses, factories, and retailers — dispatch boards, treasury, and live ops.',
        laneLabel: 'Portals',
        deviceVisual: (
          <div className="w-full max-w-md border border-white/20 bg-black p-4">
            <div className="mb-3 flex gap-1.5">
              <span className="h-2 w-2 rounded-full bg-white/30" />
              <span className="h-2 w-2 rounded-full bg-white/30" />
              <span className="h-2 w-2 rounded-full bg-white/30" />
            </div>
            <div className="grid grid-cols-3 gap-2">
              {['Dispatch', 'Fleet', 'Treasury'].map((l) => (
                <div key={l} className="border border-white/15 p-3 text-center text-[10px] font-mono uppercase text-white/50">
                  {l}
                </div>
              ))}
            </div>
          </div>
        ),
        featured: {
          tag: 'Platform',
          title: 'Supplier Control Plane',
          description:
            'Network oversight for suppliers — order vetting, dispatch preview, topology management, and treasury views.',
          image: SITE_IMAGES.logisticsPlatformUi,
          href: '/join',
          ctaLabel: 'REQUEST DEMO',
        },
        apps: [
          {
            tag: 'Warehouse',
            title: 'Warehouse Dispatch Board',
            description:
              'Visual morning dispatch with truck-and-order matching, capacity planning, and gate seal workflow.',
            image: SITE_IMAGES.warehouseAutomation,
            href: '/join',
          },
          {
            tag: 'Retailer',
            title: 'Retailer Commerce Portal',
            description: 'Catalog browsing, checkout, delivery scheduling, and live order tracking.',
            image: SITE_IMAGES.multimodalHub,
            href: '/join',
            tone: 'light',
          },
          {
            tag: 'Fleet',
            title: 'Fleet Telemetry Map',
            description: 'Live fleet map with planned-vs-actual routes and deviation alerts.',
            image: EDITORIAL_IMAGES[0],
            href: '/join',
            variant: 'vertical',
          },
          {
            tag: 'Finance',
            title: 'Payment Integrity',
            description: 'Checkout through driver collection to supplier treasury — one reconciled flow.',
            image: EDITORIAL_IMAGES[1],
            href: '/join',
            variant: 'vertical',
          },
        ],
        features: [
          { tag: 'Performance', title: 'High Performance', description: 'Realtime refresh during peak dispatch.', image: EDITORIAL_IMAGES[2], href: '/join' },
          { tag: 'Design', title: 'Role-Ready UX', description: 'Interfaces tuned for ops teams.', image: EDITORIAL_IMAGES[3], href: '/join', tone: 'light' },
          { tag: 'Architecture', title: 'Shared Contracts', description: 'Portal and mobile read the same truth.', image: EDITORIAL_IMAGES[4], href: '/join' },
          { tag: 'Scale', title: 'Multi-Site Networks', description: 'Many warehouses on one platform.', image: EDITORIAL_IMAGES[5], href: '/join', tone: 'light' },
        ],
      }}
    />
  );
}
