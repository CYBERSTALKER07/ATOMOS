import AppsFamilyPage from '@/app/components/apps/AppsFamilyPage';
import { EDITORIAL_IMAGES } from '@/app/components/apps/AppsFamilyPage';
import { SITE_IMAGES } from '@/app/lib/siteAssets';

function DeviceVisual({ labels }: { labels: string[] }) {
  return (
    <div className="w-full max-w-md border border-white/20 bg-black p-4">
      <div className="mb-3 flex gap-1.5">
        <span className="h-2 w-2 rounded-full bg-white/30" />
        <span className="h-2 w-2 rounded-full bg-white/30" />
        <span className="h-2 w-2 rounded-full bg-white/30" />
      </div>
      <div className="grid grid-cols-3 gap-2">
        {labels.map((l) => (
          <div
            key={l}
            className="border border-white/15 p-3 text-center text-[10px] font-mono uppercase text-white/50"
          >
            {l}
          </div>
        ))}
      </div>
    </div>
  );
}

export default function WebAppsPage() {
  return (
    <AppsFamilyPage
      config={{
        surface: 'web',
        title: 'Operations Portals',
        subtitle:
          'Web portals for suppliers, warehouses, factories, and retailers — dispatch boards, treasury, and live ops.',
        laneLabel: 'Portals',
        deviceVisual: <DeviceVisual labels={['Dispatch', 'Fleet', 'Treasury']} />,
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
      configRu={{
        surface: 'web',
        title: 'Операционные порталы',
        subtitle:
          'Веб-порталы для поставщиков, складов, заводов и ритейлеров — доски диспетчеризации, казначейство и живые операции.',
        laneLabel: 'Порталы',
        deviceVisual: <DeviceVisual labels={['Диспетчер', 'Автопарк', 'Казна']} />,
        featured: {
          tag: 'Платформа',
          title: 'Панель управления поставщика',
          description:
            'Контроль сети для поставщиков — проверка заказов, превью диспетчеризации, топология и казначейство.',
          image: SITE_IMAGES.logisticsPlatformUi,
          href: '/join',
          ctaLabel: 'ЗАПРОСИТЬ ДЕМО',
        },
        apps: [
          {
            tag: 'Склад',
            title: 'Доска диспетчеризации склада',
            description:
              'Визуальная утренняя диспетчеризация с подбором грузовиков и заказов, планированием вместимости и пломбированием.',
            image: SITE_IMAGES.warehouseAutomation,
            href: '/join',
          },
          {
            tag: 'Ритейлер',
            title: 'Портал коммерции ритейлера',
            description: 'Каталог, оформление, планирование доставки и живое отслеживание заказов.',
            image: SITE_IMAGES.multimodalHub,
            href: '/join',
            tone: 'light',
          },
          {
            tag: 'Автопарк',
            title: 'Карта телеметрии автопарка',
            description: 'Живая карта автопарка с маршрутами «план vs факт» и алертами по отклонениям.',
            image: EDITORIAL_IMAGES[0],
            href: '/join',
            variant: 'vertical',
          },
          {
            tag: 'Финансы',
            title: 'Целостность платежей',
            description: 'От оформления через сбор водителем до казначейства поставщика — один сверенный поток.',
            image: EDITORIAL_IMAGES[1],
            href: '/join',
            variant: 'vertical',
          },
        ],
        features: [
          { tag: 'Производительность', title: 'Высокая производительность', description: 'Реалтайм-обновление в пик диспетчеризации.', image: EDITORIAL_IMAGES[2], href: '/join' },
          { tag: 'Дизайн', title: 'UX под роли', description: 'Интерфейсы, заточенные под операционные команды.', image: EDITORIAL_IMAGES[3], href: '/join', tone: 'light' },
          { tag: 'Архитектура', title: 'Общие контракты', description: 'Портал и мобильные читают одну правду.', image: EDITORIAL_IMAGES[4], href: '/join' },
          { tag: 'Масштаб', title: 'Мультисайтовые сети', description: 'Много складов на одной платформе.', image: EDITORIAL_IMAGES[5], href: '/join', tone: 'light' },
        ],
      }}
    />
  );
}
