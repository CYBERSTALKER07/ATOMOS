import type { Metadata } from 'next';
import { getServerLanguage } from '@/app/lib/i18n/server';
import {
  pageMetadata,
  faqPageJsonLd,
  breadcrumbJsonLd,
  softwareApplicationJsonLd,
  jsonLdGraphScript,
} from '@/app/lib/seo';
import BrandedSubpageLayout from '@/app/components/layout/BrandedSubpageLayout';

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getServerLanguage();
  const isRu = lang === 'ru';

  const title = isRu
    ? 'Автоматизация логистики: диспетчеризация и маршруты | Pegasus'
    : 'Logistics Automation & Automated Dispatch Platform | Pegasus';

  const description = isRu
    ? 'Автоматизируйте диспетчеризацию, расчет маршрутов (CVRP), КПП склада, подтверждение доставки и сверку платежей с системой автоматизации логистики Pegasus.'
    : 'Automate dispatch, vehicle routing (CVRP), warehouse gate check-ins, proof of delivery, and payment reconciliation with Pegasus logistics automation.';

  return pageMetadata({
    title,
    description,
    path: '/logistics-automation',
    language: lang,
  });
}

const LOGISTICS_AUTOMATION_FAQS = [
  {
    question: 'What is logistics automation in Pegasus?',
    answer:
      'Logistics automation in Pegasus is the end-to-end algorithmic orchestration of order dispatch, vehicle route optimization (CVRP), warehouse staging, security seal gate checks, and point-of-delivery payment reconciliation without manual spreadsheet coordination.',
  },
  {
    question: 'How does automated CVRP dispatching work?',
    answer:
      'Pegasus integrates Google OR-Tools algorithms to automatically group pending orders by delivery zones, vehicle payload capacities (GVW), volumetric cubic limits, driver working hours, and retailer receiving time windows within seconds.',
  },
  {
    question: 'How does automated gate security verification prevent loading errors?',
    answer:
      'Warehouse gate staff scan tamper-evident digital barcode seals attached to each outbound truck. Pegasus validates the manifest in real time against the assigned driver and destination stops, preventing incorrect trucks from leaving the yard.',
  },
  {
    question: 'How does automated proof-of-delivery (PoD) reduce disputes?',
    answer:
      'When a driver arrives at a retailer, geofencing triggers delivery mode. Drivers scan item barcodes, capture recipient signatures on glass, and record payment collection. The order status, invoice, and inventory ledgers are updated simultaneously in under 100 milliseconds.',
  },
  {
    question: 'What ROI can enterprises expect from logistics automation?',
    answer:
      'Enterprises running Pegasus typically observe a 42% reduction in morning dispatch planning time, a 19% reduction in total fleet fuel consumption through optimal routing, and 100% elimination of dropped handoffs and lost cash collections.',
  },
];

export default async function LogisticsAutomationPage() {
  const lang = await getServerLanguage();
  const isRu = lang === 'ru';

  const structuredData = [
    softwareApplicationJsonLd(lang),
    faqPageJsonLd(LOGISTICS_AUTOMATION_FAQS, lang),
    breadcrumbJsonLd([
      { name: 'Home', path: '/' },
      { name: 'Solutions', path: '/solutions' },
      { name: 'Logistics Automation', path: '/logistics-automation' },
    ]),
  ];

  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={jsonLdGraphScript(structuredData)}
      />

      <BrandedSubpageLayout
        activeHref="/logistics-automation"
        breadcrumb={{
          categoryLabel: isRu ? 'РЕШЕНИЯ' : 'SOLUTIONS',
          categoryHref: '/solutions',
          currentPage: isRu ? 'АВТОМАТИЗАЦИЯ ЛОГИСТИКИ' : 'LOGISTICS AUTOMATION',
          telemetryCode: 'NODE-04 // ONLINE',
        }}
        badge={
          isRu
            ? 'АЛГОРИТМИЧЕСКАЯ ОРКЕСТРАЦИЯ АВТОПАРКА'
            : 'TERAFAB-MATCHED ALGORITHMIC FLEET & YARD ORCHESTRATION'
        }
        title={
          isRu
            ? 'Интеллектуальная автоматизация логистики'
            : 'Intelligent Logistics Automation'
        }
        summary={
          isRu
            ? 'Автоматизируйте маршрутизацию транспорта, балансировку диспетчеризации, досмотр на КПП складов и закрытие казначейства без ручных задержек.'
            : 'Automate vehicle routing, visual dispatch load balancing, warehouse gate inspections, and treasury reconciliation with deterministic sub-second calculation.'
        }
        primaryCta={{
          label: isRu ? 'Смотреть автоматизацию' : 'See Automation in Action',
          href: '/join',
        }}
        secondaryCta={{
          label: isRu ? 'Гайд по оптимизации' : 'Dispatch Optimization Guide',
          href: '/capabilities/smarter-dispatch',
        }}
        heroImageSrc="/images/logistics-automation-hero.jpg"
        heroImageAlt="Pegasus Autonomous Logistics and Freight Terminal"
        metrics={[
          {
            value: '42%',
            label: isRu ? 'Сокращение времени планирования' : 'Reduction in Dispatch Prep Time',
            sublabel: isRu ? 'Алгоритмическая группировка' : 'Algorithmic multi-depot CVRP clustering',
          },
          {
            value: '19%',
            label: isRu ? 'Экономия топлива по маршрутам' : 'Fuel Savings via CVRP Routing',
            sublabel: isRu ? 'Минимизация холостого пробега' : 'Volumetric constraint optimization',
          },
          {
            value: '< 100ms',
            label: isRu ? 'Синхронизация состояния сети' : 'Real-time State Synchronization',
            sublabel: isRu ? 'Телеметрия автопарка и складов' : 'Distributed fleet & yard telemetry sync',
          },
          {
            value: '100%',
            label: isRu ? 'Исключение расхождений кассы' : 'Cash & Cargo Ledger Reconciliation',
            sublabel: isRu ? 'Цифровой PoD и казначейство' : 'ACID ledger updates at point of handoff',
          },
        ]}
        capabilitiesKicker={isRu ? 'МОДУЛИ АВТОМАТИЗАЦИИ' : 'AUTOMATION MODULES'}
        capabilitiesTitle={
          isRu
            ? 'Шесть автоматизированных систем Pegasus'
            : 'Six Automated Engines Powering Pegasus'
        }
        capabilitiesDescription={
          isRu
            ? 'От утреннего расчета маршрутов до вечернего закрытия казначейства — каждый операционный этап управляется детерминированной логикой.'
            : 'From morning route calculation to evening treasury closure, every operational step is governed by deterministic business logic.'
        }
        capabilities={[
          {
            id: 'cvrp',
            tag: 'ENGINE 01',
            badge: 'OR-TOOLS SOLVER',
            title: isRu ? '1. Автоматический расчет CVRP' : '1. Automated CVRP Routing',
            description: isRu
              ? 'Математическая оптимизация многоточечных маршрутов с учетом веса, объема, окон доставки и пробок.'
              : 'Mathematical multi-stop route optimization balancing vehicle weight limits, cubic space, customer time windows, and traffic patterns with Google OR-Tools.',
            href: '/capabilities/smarter-dispatch',
            imageSrc: '/Gemini_Generated_Image_un3te4un3te4un3t.webp',
          },
          {
            id: 'staging',
            tag: 'ENGINE 02',
            badge: 'GATE CHECKPOINT',
            title: isRu ? '2. КПП и цифровые пломбы' : '2. Automated Staging & Gate Check',
            description: isRu
              ? 'Цифровые штрихкод-пломбы верифицируют каждую паллету и предотвращают ошибки отправки до выезда с территории.'
              : 'Digital barcode seals verify every pallet and prevent outbound truck dispatch errors before vehicles exit the loading terminal gate.',
            href: '/platform/warehouse-ops',
            imageSrc: '/Gemini_Generated_Image_y7jkmqy7jkmqy7jk.webp',
          },
          {
            id: 'geofence',
            tag: 'ENGINE 03',
            badge: 'TELEMETRY MESH',
            title: isRu ? '3. Автоматический геофенсинг' : '3. Automated Geofence Status',
            description: isRu
              ? 'Телеметрия в реальном времени фиксирует прибытие на склад, уведомляет приемку и бронирует разгрузочный док.'
              : 'Live vehicle telemetry triggers automatic arrival status, notifying receiving managers and preparing loading bays the moment a truck enters the perimeter.',
            href: '/technology/iot-telemetry',
            imageSrc: '/electric_semi_truck_tesla_with_trailer_rigged_362-1.jpg',
          },
          {
            id: 'pod',
            tag: 'ENGINE 04',
            badge: 'DIGITAL POD',
            title: isRu ? '4. Подтверждение доставки (PoD)' : '4. Automated Proof of Delivery',
            description: isRu
              ? 'Сканирование на стойке клиента фиксирует приемку товаров, цифровую подпись на стекле и фотоотчет с мгновенной синхронизацией.'
              : 'Barcode scan verification at the retailer counter captures item-level receipt, glass signature, and photo validation with instant cloud sync.',
            href: '/capabilities/proof-of-delivery',
            imageSrc: '/images/pegasus_handheld_os.jpg',
          },
          {
            id: 'treasury',
            tag: 'ENGINE 05',
            badge: 'ACID SETTLEMENT',
            title: isRu ? '5. Автоматическое казначейство' : '5. Automated Treasury Balancing',
            description: isRu
              ? 'Прием наличных, торговый кредит и урегулирование споров сразу разносятся по главной книге без ручной сверки.'
              : 'Point-of-delivery cash collection, commercial trade credit, and dispute offsets are immediately posted to the general ledger with zero manual reconciliation.',
            href: '/capabilities/instant-treasury',
            imageSrc: '/Gemini_Generated_Image_1y7rbo1y7rbo1y7r.webp',
          },
          {
            id: 'override',
            tag: 'GOVERNANCE',
            badge: 'OPERATOR PROTOCOL',
            title: isRu ? '6. Архитектура ручного контроля' : '6. Human Override Architecture',
            description: isRu
              ? 'Автоматизация помогает оператору. Диспетчеры и начальники складов могут в 1 клик переназначить водителя или изменить приоритет.'
              : 'Automation serves the floor. Dispatchers and warehouse managers retain 1-click manual override capability for rush orders and sick driver substitutions.',
            href: '/roles/dispatch',
            imageSrc: '/Gemini_Generated_Image_xvlgisxvlgisxvlg.webp',
          },
        ]}
        faqsKicker={isRu ? 'ПРОТОКОЛЫ И СПЕЦИФИКАЦИИ' : 'SYSTEM PROTOCOLS'}
        faqsTitle={
          isRu
            ? 'Часто задаваемые вопросы по платформе'
            : 'Logistics Automation Platform FAQ'
        }
        faqs={LOGISTICS_AUTOMATION_FAQS}
        conversionBand={{
          kicker: isRu ? 'ГОТОВО К ВНЕДРЕНИЮ' : 'ENTERPRISE PRODUCTION READY',
          title: isRu
            ? 'Готовы автоматизировать логистику автопарка?'
            : 'Ready to Automate Your Freight & Yard Operations?',
          description: isRu
            ? 'Интегрируйте платформу Pegasus с вашей ERP, WMS и бортовыми телематическими терминалами менее чем за 14 дней.'
            : 'Integrate Pegasus into your logistics network with sub-second CVRP calculation, automated gate verification, and real-time ledger accounting in under two weeks.',
          primaryCta: {
            label: isRu ? 'Запросить демонстрацию →' : 'Request Enterprise Demo →',
            href: '/join',
          },
          secondaryCta: {
            label: isRu ? 'Документация API' : 'Explore API & Docs',
            href: '/docs',
          },
        }}
        showPartnerStrip={true}
      />
    </>
  );
}
