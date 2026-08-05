import AppsFamilyPage from '@/app/components/apps/AppsFamilyPage';
import { EDITORIAL_IMAGES } from '@/app/components/apps/AppsFamilyPage';
import { SITE_IMAGES } from '@/app/lib/siteAssets';

export default function MobileAppsPage() {
  return (
    <AppsFamilyPage
      config={{
        surface: 'mobile',
        title: 'Field Mobile Apps',
        subtitle:
          'Native apps for drivers, warehouse floor teams, and gate operators — built for the field with offline tolerance.',
        laneLabel: 'Mobile',
        deviceVisual: (
          <div className="flex gap-4">
            {[1, 2].map((i) => (
              <div
                key={i}
                className="h-48 w-24 rounded-2xl border-2 border-white/25 bg-black p-2"
              >
                <div className="h-full w-full border border-white/10 bg-[#111]" />
              </div>
            ))}
          </div>
        ),
        featured: {
          tag: 'Driver',
          title: 'Driver Execution',
          description:
            'Route execution stop by stop — sealed manifests, delivery confirmation, cash collection, and live progress.',
          image: SITE_IMAGES.deliveryDrone,
          href: '/join',
          ctaLabel: 'REQUEST DEMO',
        },
        apps: [
          {
            tag: 'Warehouse',
            title: 'Warehouse & Gate',
            description: 'Dispatch boards, manifest scanning, seal workflows, and live fleet visibility.',
            image: SITE_IMAGES.portCraneScene,
            href: '/join',
          },
          {
            tag: 'Offline',
            title: 'Field Resilience',
            description: 'Queue mutations offline and sync when connectivity returns — no lost deliveries.',
            image: EDITORIAL_IMAGES[0],
            href: '/join',
            tone: 'light',
          },
        ],
        features: [
          { tag: 'GPS', title: 'Geofenced Arrival', description: 'Auto status at delivery zones.', image: EDITORIAL_IMAGES[1], href: '/join' },
          { tag: 'COD', title: 'Cash Collection', description: 'Driver collection with audit trail.', image: EDITORIAL_IMAGES[2], href: '/join', tone: 'light' },
          { tag: 'Scan', title: 'Manifest Scanning', description: 'Seal verification at gate.', image: EDITORIAL_IMAGES[3], href: '/join' },
        ],
      }}
    />
  );
}
