import AppsFamilyPage from '@/app/components/apps/AppsFamilyPage';
import { EDITORIAL_IMAGES } from '@/app/components/apps/AppsFamilyPage';
import { SITE_IMAGES } from '@/app/lib/siteAssets';

const deviceVisual = (
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
);

export default function MobileAppsPage() {
  return (
    <AppsFamilyPage
      config={{
        surface: 'mobile',
        title: 'Field Mobile Apps',
        subtitle:
          'Native apps for drivers, warehouse floor teams, and gate operators — built for the field with offline tolerance.',
        laneLabel: 'Mobile',
        deviceVisual,
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
      configRu={{
        surface: 'mobile',
        title: 'Полевые мобильные приложения',
        subtitle:
          'Нативные приложения для водителей, складских команд и операторов ворот — для поля с устойчивостью к офлайну.',
        laneLabel: 'Мобильные',
        deviceVisual,
        featured: {
          tag: 'Водитель',
          title: 'Исполнение водителем',
          description:
            'Исполнение маршрута остановка за остановкой — пломбированные манифесты, подтверждение доставки, сбор наличных и живой прогресс.',
          image: SITE_IMAGES.deliveryDrone,
          href: '/join',
          ctaLabel: 'ЗАПРОСИТЬ ДЕМО',
        },
        apps: [
          {
            tag: 'Склад',
            title: 'Склад и ворота',
            description: 'Доски диспетчеризации, сканирование манифестов, пломбирование и живая видимость автопарка.',
            image: SITE_IMAGES.portCraneScene,
            href: '/join',
          },
          {
            tag: 'Офлайн',
            title: 'Устойчивость в поле',
            description: 'Очередь изменений офлайн и синхронизация при появлении сети — без потерянных доставок.',
            image: EDITORIAL_IMAGES[0],
            href: '/join',
            tone: 'light',
          },
        ],
        features: [
          { tag: 'GPS', title: 'Геофенс прибытия', description: 'Автостатус в зонах доставки.', image: EDITORIAL_IMAGES[1], href: '/join' },
          { tag: 'COD', title: 'Сбор наличных', description: 'Сбор водителем с аудит-следом.', image: EDITORIAL_IMAGES[2], href: '/join', tone: 'light' },
          { tag: 'Скан', title: 'Сканирование манифеста', description: 'Проверка пломбы на воротах.', image: EDITORIAL_IMAGES[3], href: '/join' },
        ],
      }}
    />
  );
}
