import AppsFamilyPage from '@/app/components/apps/AppsFamilyPage';
import { EDITORIAL_IMAGES } from '@/app/components/apps/AppsFamilyPage';
import { SITE_IMAGES } from '@/app/lib/siteAssets';

function DeviceVisual({ labels }: { labels: string[] }) {
  return (
    <div className="w-full max-w-lg border border-white/20 bg-[#111] p-3">
      <div className="mb-2 h-2 w-full bg-white/10" />
      <div className="grid grid-cols-4 gap-2">
        {labels.map((l) => (
          <div
            key={l}
            className="aspect-video border border-white/10 text-[9px] font-mono uppercase text-white/40 flex items-center justify-center"
          >
            {l}
          </div>
        ))}
      </div>
    </div>
  );
}

export default function DesktopAppsPage() {
  return (
    <AppsFamilyPage
      config={{
        surface: 'desktop',
        title: 'Desktop Command Centers',
        subtitle:
          'Electron and native desktop apps for control-room teams — multi-monitor dispatch, treasury, and network oversight.',
        laneLabel: 'Desktop',
        deviceVisual: <DeviceVisual labels={['Map', 'Board', 'Treasury', 'Alerts']} />,
        featured: {
          tag: 'Control room',
          title: 'Supplier Desktop Console',
          description:
            'Multi-monitor oversight for supplier ops — vetting queues, dispatch preview, and treasury in one workspace.',
          image: SITE_IMAGES.logisticsPlatformUi,
          href: '/join',
          ctaLabel: 'REQUEST DEMO',
        },
        apps: [
          {
            tag: 'Dispatch',
            title: 'Warehouse Desktop Board',
            description: 'Large-format dispatch for morning peak — drag trucks, override assignments, seal loads.',
            image: SITE_IMAGES.warehouseAutomation,
            href: '/join',
          },
          {
            tag: 'Analytics',
            title: 'Network Analytics',
            description: 'Desktop dashboards for SLA, OTD, and cohort retention across sites.',
            image: EDITORIAL_IMAGES[4],
            href: '/join',
            tone: 'light',
          },
          {
            tag: 'Integrations',
            title: 'ERP Bridge',
            description: 'Desktop agent for ERP sync and batch reconciliation.',
            image: EDITORIAL_IMAGES[5],
            href: '/join',
            variant: 'vertical',
          },
        ],
        features: [
          { tag: 'Multi-monitor', title: 'Control Room Layout', description: 'Span boards across displays.', image: EDITORIAL_IMAGES[0], href: '/join' },
          { tag: 'Keyboard', title: 'Power User Shortcuts', description: 'Dispatch without the mouse.', image: EDITORIAL_IMAGES[1], href: '/join', tone: 'light' },
          { tag: 'Offline', title: 'Local Cache', description: 'Keep working through outages.', image: EDITORIAL_IMAGES[2], href: '/join' },
          { tag: 'Security', title: 'Device Trust', description: 'Managed installs for enterprise.', image: EDITORIAL_IMAGES[3], href: '/join', tone: 'light' },
        ],
      }}
      configRu={{
        surface: 'desktop',
        title: 'Десктопные командные центры',
        subtitle:
          'Electron и нативные десктоп-приложения для команд в диспетчерской — мультимониторная диспетчеризация, казначейство и контроль сети.',
        laneLabel: 'Десктоп',
        deviceVisual: <DeviceVisual labels={['Карта', 'Доска', 'Казна', 'Алерты']} />,
        featured: {
          tag: 'Диспетчерская',
          title: 'Десктоп-консоль поставщика',
          description:
            'Мультимониторный контроль для операций поставщика — очереди проверки, превью диспетчеризации и казначейство в одном рабочем пространстве.',
          image: SITE_IMAGES.logisticsPlatformUi,
          href: '/join',
          ctaLabel: 'ЗАПРОСИТЬ ДЕМО',
        },
        apps: [
          {
            tag: 'Диспетчеризация',
            title: 'Десктоп-доска склада',
            description: 'Крупноформатная диспетчеризация для утреннего пика — перетаскивание грузовиков, переопределения, пломбирование.',
            image: SITE_IMAGES.warehouseAutomation,
            href: '/join',
          },
          {
            tag: 'Аналитика',
            title: 'Аналитика сети',
            description: 'Десктоп-дашборды SLA, OTD и удержания когорт по площадкам.',
            image: EDITORIAL_IMAGES[4],
            href: '/join',
            tone: 'light',
          },
          {
            tag: 'Интеграции',
            title: 'ERP-мост',
            description: 'Десктоп-агент для синхронизации с ERP и пакетной сверки.',
            image: EDITORIAL_IMAGES[5],
            href: '/join',
            variant: 'vertical',
          },
        ],
        features: [
          { tag: 'Мультимонитор', title: 'Раскладка диспетчерской', description: 'Доски на нескольких экранах.', image: EDITORIAL_IMAGES[0], href: '/join' },
          { tag: 'Клавиатура', title: 'Горячие клавиши', description: 'Диспетчеризация без мыши.', image: EDITORIAL_IMAGES[1], href: '/join', tone: 'light' },
          { tag: 'Офлайн', title: 'Локальный кэш', description: 'Работайте при сбоях сети.', image: EDITORIAL_IMAGES[2], href: '/join' },
          { tag: 'Безопасность', title: 'Доверие устройства', description: 'Управляемые установки для enterprise.', image: EDITORIAL_IMAGES[3], href: '/join', tone: 'light' },
        ],
      }}
    />
  );
}
