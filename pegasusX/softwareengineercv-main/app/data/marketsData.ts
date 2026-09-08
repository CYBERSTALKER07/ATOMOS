export interface MarketProfile {
  slug: string;
  name: string;
  nativeName: string;
  region: string;
  flag: string;
  currency: string;
  primaryLanguage: string;
  cellCluster: string;
  latency: string;
  headline: string;
  summary: string;
  marketOverview: string;
  complianceFrameworks: string[];
  logisticsBottlenecks: string[];
  pegasusSolution: string[];
  stats: { label: string; value: string }[];
  faqs: { question: string; answer: string }[];
}

export const MARKETS_DATA: MarketProfile[] = [
  {
    slug: 'united-states',
    name: 'United States',
    nativeName: 'United States',
    region: 'North America',
    flag: '🇺🇸',
    currency: 'USD ($)',
    primaryLanguage: 'English',
    cellCluster: 'cell-us (Google Cloud us-central1 & us-east4)',
    latency: '< 15ms continental',
    headline: 'Enterprise Logistics & Supply Chain Automation for the United States',
    summary: 'Power high-velocity multi-depot distribution, interstate freight compliance (FMCSA/ELD), and automated retail fulfillment across the US market.',
    marketOverview: 'The United States represents a $1.9T commercial logistics economy characterized by massive interstate freight lanes, strict Hours of Service (HOS) rules, and high carrier fragmentation.',
    complianceFrameworks: ['FMCSA ELD Mandate', 'DOT Hours of Service (HOS)', 'USMCA Trade Protocol', 'FDA FSMA Food Safety'],
    logisticsBottlenecks: [
      'High driver turnover and stringent FMCSA Hours of Service enforcement.',
      'Carrier spot rate volatility and fragmented multi-tier 3PL networks.',
      'Detention fee penalties at congested warehouse receiving docks.',
    ],
    pegasusSolution: [
      'Automated Google OR-Tools CVRP routing minimizing deadhead miles and honoring HOS rest breaks.',
      'Mobile driver execution app operating seamlessly on iOS and Android without proprietary hardware leases.',
      'Real-time automated proof-of-delivery (PoD) with instant digital bill of lading (eBOL) sign-offs.',
    ],
    stats: [
      { label: 'Dispatch Planning Reduction', value: '45%' },
      { label: 'Detention Time Eliminated', value: '38%' },
      { label: 'Fleet Fuel Savings', value: '18%' },
    ],
    faqs: [
      {
        question: 'Does Pegasus comply with US FMCSA and DOT regulations?',
        answer: 'Yes. Pegasus tracks driver duty status, geofenced rest periods, and electronic proof of delivery in compliance with FMCSA guidelines.',
      },
      {
        question: 'Can Pegasus replace legacy US TMS systems like Samsara or Rose Rocket?',
        answer: 'Yes. Pegasus provides hardware-agnostic mobile driver execution, avoiding 36-month proprietary hardware leases while synchronizing 6 enterprise roles from supplier to retail shelf.',
      },
    ],
  },
  {
    slug: 'germany',
    name: 'Germany',
    nativeName: 'Deutschland',
    region: 'Europe (DACH)',
    flag: '🇩🇪',
    currency: 'EUR (€)',
    primaryLanguage: 'German',
    cellCluster: 'cell-eu (Google Cloud europe-west3 Frankfurt)',
    latency: '< 8ms DACH region',
    headline: 'Globale Logistik- & Supply-Chain-Software für Deutschland',
    summary: 'Automatisieren Sie Flottendisposition, Lkw-Maut-Erfassung, grenzüberschreitenden EU-Transit und Just-in-Time-Industrielieferungen in Deutschland.',
    marketOverview: 'Deutschland ist die führende Logistikdrehscheibe Europas mit hochdichten Autobahnnetzen, anspruchsvollen Industrie-4.0-Lieferketten und strengen CO2-Maut-Vorgaben.',
    complianceFrameworks: ['EU Mobility Package', 'Toll Collect Lkw-Maut', 'Lieferkettensorgfaltspflichtengesetz (LkSG)', 'DSGVO / GDPR'],
    logisticsBottlenecks: [
      'Hohe Mautkosten und CO2-Emissionsabgaben auf Bundesautobahnen.',
      'Fahrermangel und strenge Kontrollen der Lenk- und Ruhezeiten.',
      'Rechtliche Dokumentationspflichten durch das deutsche Lieferkettengesetz.',
    ],
    pegasusSolution: [
      'Algorithmenbasierte Maut-optimierte Routenführung zur Senkung von Autobahngebühren.',
      'Automatische digitale Frachtbriefe (e-CMR) und barcodebasierte Rampensteuerung.',
      'Souveräne Datenverarbeitung in Frankfurter Cloud-Clustern mit voller DSGVO-Konformität.',
    ],
    stats: [
      { label: 'Maut- & Kraftstoffreduktion', value: '22%' },
      { label: 'Durchlaufzeit Rampe', value: '-35%' },
      { label: 'Pünktlichkeitsrate (On-Time)', value: '99.4%' },
    ],
    faqs: [
      {
        question: 'Ist Pegasus mit dem deutschen Lieferkettengesetz (LkSG) konform?',
        answer: 'Ja. Pegasus protokolliert die lückenlose digitale Chain-of-Custody mit unveränderlichen Zeitstempeln und auditierbaren Frachtbriefen.',
      },
      {
        question: 'Unterstützt Pegasus Schnittstellen zu SAP S/4HANA in Deutschland?',
        answer: 'Ja. Pegasus verfügt über native gRPC- und Kafka-Konnektoren für den bidirektionalen Echtzeit-Abgleich mit SAP-Systemen.',
      },
    ],
  },
  {
    slug: 'united-kingdom',
    name: 'United Kingdom',
    nativeName: 'United Kingdom',
    region: 'Europe (UK)',
    flag: '🇬🇧',
    currency: 'GBP (£)',
    primaryLanguage: 'English',
    cellCluster: 'cell-eu (Google Cloud europe-west2 London)',
    latency: '< 10ms UK national',
    headline: 'B2B Logistics Software & Fleet Dispatch for the United Kingdom',
    summary: 'Streamline post-Brexit customs documentation, port-to-warehouse drayage, and next-day wholesale distribution across the UK.',
    marketOverview: 'The UK logistics market demands agile multi-drop retail route optimization, clean air zone (ULEZ) routing, and seamless customs declaration integration.',
    complianceFrameworks: ['HMRC CDS Customs', 'London ULEZ Standards', 'DVSA Driver Hours', 'Earned Recognition Scheme'],
    logisticsBottlenecks: [
      'Customs declaration delays at Dover and maritime container ports.',
      'Urban delivery access penalties in London ULEZ and clean air zones.',
      'Retail delivery window penalties imposed by major UK supermarket chains.',
    ],
    pegasusSolution: [
      'Automated customs payload verification with digital seal checks at departure docks.',
      'Low-emission urban route planning avoiding congested and toll-heavy zones.',
      'Live geofenced customer tracking links with minute-accurate delivery ETAs.',
    ],
    stats: [
      { label: 'ULEZ Toll Optimization', value: '31%' },
      { label: 'Cross-Dock Transit Speed', value: '+40%' },
      { label: 'Dispute Reduction', value: '100%' },
    ],
    faqs: [
      {
        question: 'How does Pegasus handle UK Clean Air Zones and ULEZ routing?',
        answer: 'Pegasus evaluates vehicle Euro emissions classes during dispatch and routes non-compliant trucks outside high-tariff zones.',
      },
      {
        question: 'Does Pegasus integrate with HMRC customs declarations?',
        answer: 'Yes. Export/import manifests link directly to HMRC declaration reference numbers on digital bills of lading.',
      },
    ],
  },
  {
    slug: 'france',
    name: 'France',
    nativeName: 'France',
    region: 'Western Europe',
    flag: '🇫🇷',
    currency: 'EUR (€)',
    primaryLanguage: 'French',
    cellCluster: 'cell-eu (Google Cloud europe-west9 Paris)',
    latency: '< 9ms nationwide',
    headline: 'Logiciel de Logistique Mondiale & Gestion de Flotte en France',
    summary: 'Optimisez les tournées multi-arrêts, la chaîne du froid alimentaire et les flux de distribution B2B à travers la France.',
    marketOverview: 'Carrefour logistique de l’Europe du Sud, la France exige une traçabilité rigoureuse de la chaîne du froid et une gestion avancée des plateformes logistiques.',
    complianceFrameworks: ['Paquet Mobilité UE', 'Réglementation ATP Chaîne du Froid', 'Facturation Électronique B2B 2026', 'RGPD'],
    logisticsBottlenecks: [
      'Exigences strictes de maintien de la température en transport agroalimentaire.',
      'Temps d’attente élevés sur les quais de déchargement de la grande distribution.',
      'Obligations de facturation électronique et d’archivage légal.',
    ],
    pegasusSolution: [
      'Surveillance continue de la chaîne du froid avec alertes de déviation en temps réel.',
      'Planification de quais automatisée pour éliminer les retards de déchargement.',
      'Génération automatique de factures électroniques et preuve de livraison numérique signée.',
    ],
    stats: [
      { label: 'Gain de Temps Dispatch', value: '44%' },
      { label: 'Pertes Chaîne du Froid', value: '0%' },
      { label: 'Litiges Factures', value: '-85%' },
    ],
    faqs: [
      {
        question: 'Pegasus gère-t-il la chaîne du froid selon la norme ATP ?',
        answer: 'Oui. Pegasus intègre les données de sondes de température et verrouille le statut de livraison si des écarts critiques surviennent.',
      },
      {
        question: 'La solution est-elle prête pour la facturation électronique obligatoire en France ?',
        answer: 'Absolument. Pegasus génère automatiquement les flux Factur-X et les preuves de livraison numériques certifiées.',
      },
    ],
  },
  {
    slug: 'spain',
    name: 'Spain',
    nativeName: 'España',
    region: 'Southern Europe',
    flag: '🇪🇸',
    currency: 'EUR (€)',
    primaryLanguage: 'Spanish',
    cellCluster: 'cell-eu (Google Cloud europe-southwest1 Madrid)',
    latency: '< 8ms Iberian Peninsula',
    headline: 'Software de Logística Global y Gestión de Rutas en España',
    summary: 'Automatice el despacho de flotas, el transporte hortofrutícola y las rutas de distribución B2B en la Península Ibérica y Europa.',
    marketOverview: 'España es la principal huerta de Europa y un nodo marítimo estratégico con corredores intensivos hacia Francia, Alemania y el norte de África.',
    complianceFrameworks: ['CMR Electrónico (e-CMR)', 'Régimen de Tacógrafo Digital UE', 'Normativa DGT de Transporte', 'Ley de Cadena Alimentaria'],
    logisticsBottlenecks: [
      'Coordinación de exportaciones perecederas con tiempos de entrega ultrarrápidos.',
      'Pérdida de cobros y retrasos en la liquidación de facturas de porte.',
      'Falta de visibilidad de flotas subcontratadas en rutas internacionales.',
    ],
    pegasusSolution: [
      'Enrutamiento CVRP adaptado a camiones refrigerados y restricciones de peso por eje.',
      'Liquidación automática y cobro contra entrega (COD) con conciliación bancaria instantánea.',
      'Acceso móvil para transportistas autónomos y flotas propias sin hardware propietario.',
    ],
    stats: [
      { label: 'Ahorro de Combustible', value: '20%' },
      { label: 'Conciliación de Cobros', value: '100% Inmediata' },
      { label: 'Reducción de Esperas en Muelle', value: '37%' },
    ],
    faqs: [
      {
        question: '¿Soporta Pegasus el e-CMR y la carta de porte digital en España?',
        answer: 'Sí. Pegasus genera la carta de porte digital con firma biométrica en pantalla y validez legal probatoria.',
      },
      {
        question: '¿Permite gestionar pagos contra entrega (COD) para distribuidores mayoristas?',
        answer: 'Sí. Los transportistas concilian el efectivo cobrado en cada parada y el saldo se actualiza en el sistema central en tiempo real.',
      },
    ],
  },
  {
    slug: 'united-arab-emirates',
    name: 'United Arab Emirates',
    nativeName: 'الإمارات العربية المتحدة',
    region: 'Middle East (GCC)',
    flag: '🇦🇪',
    currency: 'AED (د.إ)',
    primaryLanguage: 'Arabic',
    cellCluster: 'cell-me (Google Cloud me-central2 Dammam & Dubai peering)',
    latency: '< 12ms GCC corridor',
    headline: 'Global Logistics & Supply Chain Platform for the United Arab Emirates',
    summary: 'Accelerate multimodal trade, bonded warehouse automation, and GCC cross-border freight from Jebel Ali to Dubai and Abu Dhabi.',
    marketOverview: 'The UAE is a premier global trade gateway connecting Asia, Europe, and Africa through hyper-modern ports, free trade zones, and high-volume re-export logistics.',
    complianceFrameworks: ['Dubai Customs Digital Protocols', 'FTA VAT Logistics Rules', 'GCC Customs Union Harmonization', 'DP World Gate Integration'],
    logisticsBottlenecks: [
      'Managing bonded free-zone customs handshakes and duty reconciliation.',
      'Extreme summer ambient temperatures requiring strict cold-chain safeguards.',
      'Multi-currency supplier invoicing (AED, USD, EUR) with complex trade credits.',
    ],
    pegasusSolution: [
      'Sub-second gate terminal checkpoints integrated with digital seal barcodes.',
      'Real-time dual-entry treasury ledger supporting AED, USD, and GCC currencies.',
      'High-frequency GPS tracking with temperature telemetry for UAE cold storage fleets.',
    ],
    stats: [
      { label: 'Gate Clearance Speed', value: '4x Faster' },
      { label: 'Cross-Border Transit Time', value: '-30%' },
      { label: 'Treasury Balancing Speed', value: 'Instant' },
    ],
    faqs: [
      {
        question: 'Can Pegasus connect to free zone operations like JAFZA and DAFZA?',
        answer: 'Yes. Pegasus tracks bonded cargo movements between free trade zones and the UAE domestic mainland with automated customs audit trails.',
      },
      {
        question: 'Does Pegasus support Arabic and English bilingual operations?',
        answer: 'Yes. The entire platform, mobile driver app, and customer tracking portals support full native Arabic (RTL) and English interfaces.',
      },
    ],
  },
  {
    slug: 'saudi-arabia',
    name: 'Saudi Arabia',
    nativeName: 'المملكة العربية السعودية',
    region: 'Middle East (GCC)',
    flag: '🇸🇦',
    currency: 'SAR (ر.س)',
    primaryLanguage: 'Arabic',
    cellCluster: 'cell-me (Google Cloud me-central2 Dammam & Riyadh peering)',
    latency: '< 15ms Kingdom-wide',
    headline: 'برمجيات سلاسل الإمداد والخدمات اللوجستية في المملكة العربية السعودية',
    summary: 'دعم رؤية السعودية 2030 بتحويل سلاسل الإمداد وتوزيع السلع وأتمتة حركة الشاحنات بين الرياض وجدة والدمام.',
    marketOverview: 'المملكة العربية السعودية تشهد أسرع نمو في البنية التحتية اللوجستية بالمنطقة، مع استثمارات استراتيجية كبرى في الموانئ والمراكز اللوجستية الإقليمية.',
    complianceFrameworks: ['هيئة الزكاة والضريبة والجمارك (ZATCA)', 'الهيئة العامة للنقل (TGA)', 'منصة وصل للربط الإلكتروني', 'نظام سابر'],
    logisticsBottlenecks: [
      'المسافات الطويلة بين المراكز الحضرية الرئيسية (الرياض، جدة، الدمام).',
      'متطلبات الربط الإلزامي مع منصات هيئة النقل وتتبع الشاحنات.',
      'تسوية مبالغ الدفع عند الاستلام (COD) في قطاع التوزيع والتجزئة.',
    ],
    pegasusSolution: [
      'تحسين مسارات الشاحنات CVRP لتغطية المسافات الطويلة وخفض استهلاك الوقود.',
      'تتبع المركبات عبر نظام GPS متوافق مع لوائح هيئة النقل وتحديثات حالة التسليم.',
      'سجل مالي متكامل للتسوية الفورية للمدفوعات النقدية والائتمانية عند التسليم.',
    ],
    stats: [
      { label: 'خفض تكاليف النقل الطويل', value: '25%' },
      { label: 'دقة مواعيد التسليم', value: '99.2%' },
      { label: 'مطابقة الفواتير ZATCA', value: '100% آلية' },
    ],
    faqs: [
      {
        question: 'هل يدعم بيغاسوس متطلبات هيئة النقل والربط مع المنصات السعودية؟',
        answer: 'نعم. المنصة مصممة للتوافق مع معايير هيئة النقل وتتبع الأساطيل وتوثيق الشحنات.',
      },
      {
        question: 'كيف يتعامل النظام مع الفوترة الإلكترونية ZATCA؟',
        answer: 'يقوم بيغاسوس بإنشاء الفواتير الإلكترونية المتوافقة مع متطلبات المرحلة الثانية من الفوترة بهيئة الزكاة والضريبة والجمارك.',
      },
    ],
  },
  {
    slug: 'china',
    name: 'China',
    nativeName: '中国',
    region: 'East Asia',
    flag: '🇨🇳',
    currency: 'CNY (¥)',
    primaryLanguage: 'Chinese',
    cellCluster: 'cell-apac (Hong Kong & Shanghai direct peering)',
    latency: '< 20ms coastal trade hubs',
    headline: '全球物流操作系统与供应链管理软件平台 — 中国',
    summary: '赋能高通量工厂出库调度、多式联运港口集疏运及全国多级批发分销网络。',
    marketOverview: '中国拥有全球规模最大的工业制造与出口供应链，对高并发调度吞吐量、毫秒级状态同步及大宗干线协同有着极高要求。',
    complianceFrameworks: ['全国道路货运车辆公共监管与服务平台', '网络货运经营管理暂行办法', '中国海关金关工程', '数据安全法'],
    logisticsBottlenecks: [
      '出厂装车高峰期月台拥堵与车辆派单混乱。',
      '跨省多式联运数据孤岛与电子运单脱节。',
      '海量批发零售网点高频小额订单的资金快速归集。',
    ],
    pegasusSolution: [
      '基于 Google OR-Tools 的自适应装载平衡与多车型路径优化引擎。',
      '数字条码防篡改道闸系统，实现装车清单与出厂即时比对。',
      '实时双向 ERP 集成，无缝对接企业 SAP、用友及金蝶系统。',
    ],
    stats: [
      { label: '月台装车周转率', value: '+45%' },
      { label: '干线空驶率降低', value: '26%' },
      { label: '对账周期缩短', value: '即时清算' },
    ],
    faqs: [
      {
        question: 'Pegasus 如何支持中国制造业高并发排产与发货？',
        answer: 'Pegasus 采用 Google Cloud Spanner 分布式事务架构与 Kafka 高性能事件流，支持每秒数万笔订单并发锁定与调度。',
      },
      {
        question: '是否支持与主流国产 ERP 系统的对接？',
        answer: '是的。Pegasus 提供开放的 gRPC 与 RESTful API，开箱即用支持金蝶、用友以及 SAP S/4HANA 的实时数据同步。',
      },
    ],
  },
  {
    slug: 'japan',
    name: 'Japan',
    nativeName: '日本',
    region: 'East Asia',
    flag: '🇯🇵',
    currency: 'JPY (¥)',
    primaryLanguage: 'Japanese',
    cellCluster: 'cell-apac (Google Cloud asia-northeast1 Tokyo)',
    latency: '< 7ms Tokyo/Osaka corridor',
    headline: 'グローバル物流＆サプライチェーン自動化システム — 日本',
    summary: '物流の「2024年問題」に対応。配車計画の自動化、待機時間の削減、精密JIT配送を実現。',
    marketOverview: '日本の物流市場はドライバーの時間外労働上限規制（2024年問題）に直面しており、極めて高精度な配車効率化と荷待ち時間の劇的削減が急務となっています。',
    complianceFrameworks: ['物流2024年問題（労働基準法改正）', '標準貨物自動車運送約款', '改正物流総合効率化法', 'デジタルタコグラフ規格'],
    logisticsBottlenecks: [
      'ドライバーの拘束時間規制による長距離輸送のキャパシティ逼迫。',
      '物流拠点における長時間の荷待ち・荷役作業による生産性低下。',
      '分刻みの時間指定配送（JIT）と誤配送ゼロの品質要求。',
    ],
    pegasusSolution: [
      'Google OR-Tools CVRP を用いた配車最適化で、拘束時間内での配送量を最大化。',
      'バース予約と事前デジタル検品による待機時間のゼロ化。',
      'iOS/Android スマートフォンで完結する高精度な電子受领サインと追跡。',
    ],
    stats: [
      { label: '荷待ち時間削減', value: '42%' },
      { label: '配車作成時間短縮', value: '75%' },
      { label: '時間遵守率 (JIT)', value: '99.8%' },
    ],
    faqs: [
      {
        question: '日本の「物流2024年問題」に対してどのような効果がありますか？',
        answer: 'Pegasusの自動配車アルゴリズムはドライバーの法定拘束時間を制約条件として組み込み、無理のない運行計画で稼働効率を最大化します。',
      },
      {
        question: '専用の車載器やハードウェアをトラックに設置する必要がありますか？',
        answer: '不要です。ドライバー自身のスマートフォン（iPhone/Android）で高精度なナビゲーション、検品、受領サインが完結します。',
      },
    ],
  },
  {
    slug: 'brazil',
    name: 'Brazil',
    nativeName: 'Brasil',
    region: 'South America (Mercosur)',
    flag: '🇧🇷',
    currency: 'BRL (R$)',
    primaryLanguage: 'Portuguese',
    cellCluster: 'cell-sa (Google Cloud southamerica-east1 São Paulo)',
    latency: '< 15ms Mercosur corridor',
    headline: 'Software de Logística Global e Gestão de Frotas no Brasil',
    summary: 'Otimize frete rodoviário, conformidade com CT-e / MDF-e e rastreamento de cargas de alto valor no Brasil.',
    marketOverview: 'O Brasil movimenta mais de 65% de suas cargas pelas rodovias, exigindo gerenciamento de risco rigoroso, prevenção de roubo de cargas e integração fiscal contínua.',
    complianceFrameworks: ['CT-e (Conhecimento de Transporte Eletrônico)', 'MDF-e (Manifesto Eletrônico)', 'Gerenciamento de Risco (GR)', 'LGPD'],
    logisticsBottlenecks: [
      'Complexidade tributária e validação fiscal instantânea em postos fiscais.',
      'Necessidade crítica de gerenciamento de risco e alertas de desvio de rota.',
      'Conciliação lenta de pagamentos de frete e adiantamentos de motoristas.',
    ],
    pegasusSolution: [
      'Roteirização inteligente com corredores seguros e zonas de parada homologadas.',
      'Sincronização digital automática com documentos fiscais eletrônicos.',
      'Conciliação imediata de comprovantes de entrega fotográficos e adiantamentos.',
    ],
    stats: [
      { label: 'Economia de Combustível', value: '18%' },
      { label: 'Conformidade Fiscal', value: '100% Digital' },
      { label: 'Tempo em Postos Fiscais', value: '-30%' },
    ],
    faqs: [
      {
        question: 'O Pegasus integra com CT-e e MDF-e no Brasil?',
        answer: 'Sim. O sistema vincula as chaves de acesso dos documentos fiscais eletrônicos diretamente aos manifestos digitais de carga.',
      },
      {
        question: 'Como funciona o rastreamento em áreas com sinal celular fraco?',
        answer: 'O app móvel do motorista armazena os eventos de telemetria e comprovantes offline e os sincroniza atomicamente assim que a conexão é restabelecida.',
      },
    ],
  },
  {
    slug: 'turkey',
    name: 'Turkey',
    nativeName: 'Türkiye',
    region: 'Eurasia',
    flag: '🇹🇷',
    currency: 'TRY (₺)',
    primaryLanguage: 'Turkish',
    cellCluster: 'cell-eu & cell-me (Istanbul direct low-latency peering)',
    latency: '< 12ms nationwide',
    headline: 'Türkiye için Global Lojistik ve Tedarik Zinciri Otomasyon Yazılımı',
    summary: 'Avrupa ve Asya arasındaki transit koridorları, TIR taşımacılığı ve yurt içi B2B dağıtım ağlarını otomatikleştirin.',
    marketOverview: 'Türkiye, Asya ile Avrupa arasında stratejik bir lojistik köprüdür. Yüksek hacimli uluslararası karayolu taşımacılığı ve gelişmiş iç dağıtım ağına sahiptir.',
    complianceFrameworks: ['TIR Karnesi Sözleşmesi', 'e-İrsaliye ve e-Fatura Sistemi', 'U-ETDS Ulaştırma Elektronik Takip Sistemi', 'KVKK'],
    logisticsBottlenecks: [
      'Sınır kapılarında (Kapıkule, Hamzabeyli) gümrük beklemeleri ve evrak karmaşası.',
      'U-ETDS sistemine anlık veri aktarımı ve bildirim zorunlulukları.',
      'Yurt içi toptan satışlarda kapıda tahsilat ve mutabakat gecikmeleri.',
    ],
    pegasusSolution: [
      'Gümrük bekleme sürelerini hesaba katan uluslararası CVRP rota optimizasyonu.',
      'U-ETDS ve e-İrsaliye ile tam uyumlu otomatik manifesto ve irsaliye aktarımı.',
      'Kapıda teslimatta anında dijital tahsilat ve muhasebe entegrasyonu.',
    ],
    stats: [
      { label: 'Sevkiyat Hazırlık Süresi', value: '-%45' },
      { label: 'U-ETDS Bildirim Doğruluğu', value: '%100' },
      { label: 'Rota Yakıt Tasarrufu', value: '%21' },
    ],
    faqs: [
      {
        question: 'Pegasus U-ETDS ve e-İrsaliye entegrasyonunu destekliyor mu?',
        answer: 'Evet. Pegasus, sefer ve yük bilgilerini yasal gereksinimlere uygun olarak elektronik ortamda üretir ve iletir.',
      },
      {
        question: 'Uluslararası TIR taşımacılığında kullanılabilir mi?',
        answer: 'Evet. Çok dilli sürücü uygulaması ve gümrük mühür kontrol modülü ile uluslararası taşımacılıkta tam denetim sağlar.',
      },
    ],
  },
  {
    slug: 'uzbekistan',
    name: 'Uzbekistan',
    nativeName: 'Oʻzbekiston',
    region: 'Central Asia',
    flag: '🇺🇿',
    currency: 'UZS (soʻm)',
    primaryLanguage: 'Uzbek',
    cellCluster: 'cell-uz (Tashkent Tier III Data Center, direct TAS-IX)',
    latency: '< 2ms TAS-IX network',
    headline: 'Oʻzbekiston uchun Global Logistika va Taʼminot Zanjiri Platformasi',
    summary: 'Toshkent va butun respublika boʻylab taʼminotchilar, yirik omborlar, haydovchilar va chakana doʻkonlarni yagona ekotizimga birlashtiring.',
    marketOverview: 'Oʻzbekiston Markaziy Osiyoning asosiy logistika markaziga aylanmoqda. Ichki ulgurji savdo va mintaqaviy tranzit yuk oqimlari jadal oʻsib bormoqda.',
    complianceFrameworks: ['E-Faktura milliy tizimi', 'Yagona elektron bojxona tizimi', 'TAS-IX milliy tarmoq qoidalari', 'Shaxsiy maʼlumotlarni himoya qilish qonuni'],
    logisticsBottlenecks: [
      'Ulgurji savdoda haydovchilar orqali naqd pul va toʻlovlarni yigʻishdagi noaniqliklar.',
      'Omborlarda ertalabki yuklash tirbandligi va yoʻnalishlarning samarasiz tuzilishi.',
      'Doʻkonlarning haqiqiy qarz balansi va tovar qoldiqlarining kechikib yangilanishi.',
    ],
    pegasusSolution: [
      'Servercore Tashkent Tier III infratuzilmasi orqali TAS-IX da 2ms tezkorlikda ishlash.',
      'Doʻkon eshigida tovarlarni skanerlash, toʻlovni qayd qilish va balansni bir zumda yangilash.',
      'Haydovchi uchun qulay oʻzbek va rus tillaridagi mobil ilova, tirbandliklarni hisobga oluvchi marshrut.',
    ],
    stats: [
      { label: 'TAS-IX Server Tezligi', value: '2ms' },
      { label: 'Toʻlovlar Kamomadi', value: '0%' },
      { label: 'Yetkazib Berish Tezligi', value: '+35%' },
    ],
    faqs: [
      {
        question: 'Pegasus serverlari Oʻzbekistonda joylashganmi?',
        answer: 'Ha! Pegasus suveren klasteri Toshkentdagi Tier III maʼlumotlar markazida oʻrnatilgan boʻlib, TAS-IX milliy trafigida 2 millisekundlik tezlikda ishlaydi.',
      },
      {
        question: 'Yetkazib beruvchilar va savdo agentlari uchun qanday qulaylik bor?',
        answer: 'Doʻkonlarga tovar yetkazilganda haydovchi mobil ilovada qabul qilingan toʻlovni kiritadi, hisob-kitoblar va qarz balansi real vaqtda yangilanadi.',
      },
    ],
  },
  {
    slug: 'singapore',
    name: 'Singapore',
    nativeName: 'Singapore',
    region: 'Southeast Asia (ASEAN)',
    flag: '🇸🇬',
    currency: 'SGD ($)',
    primaryLanguage: 'English',
    cellCluster: 'cell-apac (Google Cloud asia-southeast1 Singapore)',
    latency: '< 5ms island-wide',
    headline: 'Logistics Automation & Port-to-Distribution Software for Singapore',
    summary: 'Orchestrate bonded warehousing, cross-border Johor-Singapore trucking, and smart island-wide distribution.',
    marketOverview: 'Singapore is the world’s leading transshipment maritime hub with unmatched efficiency, smart port infrastructure, and intensive regional ASEAN logistics connections.',
    complianceFrameworks: ['Singapore Customs TradeNet', 'Portnet Terminal Gateway', 'IMDA Electronic Invoicing (Peppol)', 'PDPA'],
    logisticsBottlenecks: [
      'Tight urban congestion and ERP toll pricing across central business districts.',
      'Port clearance timing synchronization for ocean container drayage.',
      'Cross-border causeway congestion between Singapore and Malaysia.',
    ],
    pegasusSolution: [
      'Automated terminal staging minimizing container detention penalties.',
      'Peppol-compliant digital invoice generation upon physical delivery completion.',
      'Predictive cross-border dispatch planning optimizing customs transit windows.',
    ],
    stats: [
      { label: 'Drayage Turnaround Time', value: '-33%' },
      { label: 'ERP Toll Cost Reduction', value: '24%' },
      { label: 'Digital Invoicing Compliance', value: '100%' },
    ],
    faqs: [
      {
        question: 'Does Pegasus integrate with Singapore Peppol e-invoicing?',
        answer: 'Yes. Pegasus generates Peppol-ready invoices as soon as goods are checked off at destination.',
      },
      {
        question: 'How does Pegasus optimize container drayage around PSA terminals?',
        answer: 'Pegasus synchronizes driver arrival windows with container readiness alerts to minimize waiting time.',
      },
    ],
  },
  {
    slug: 'mexico',
    name: 'Mexico',
    nativeName: 'México',
    region: 'North America (USMCA)',
    flag: '🇲🇽',
    currency: 'MXN ($)',
    primaryLanguage: 'Spanish',
    cellCluster: 'cell-us & cell-latam (Querétaro & Dallas peering)',
    latency: '< 15ms nearshoring corridors',
    headline: 'Software de Cadena de Suministro y Logística Nearshoring en México',
    summary: 'Acelere el transporte transfronterizo USMCA, cumpla con Carta Porte SAT y controle frotas de manufactura en México.',
    marketOverview: 'México experimenta un auge histórico de nearshoring en manufactura automotriz y electrónica, exigiendo trazabilidad estricta y cumplimiento aduanero con EE.UU.',
    complianceFrameworks: ['Complemento Carta Porte SAT (CFDI)', 'Regulaciones SCT de Carga', 'Tratado T-MEC / USMCA', 'Esquema OEA Aduanero'],
    logisticsBottlenecks: [
      'Requisitos fiscales obligatorios del Complemento Carta Porte con riesgo de multas.',
      'Tiempos muertos prolongados en cruces fronterizos (Laredo, Tijuana, Ciudad Juárez).',
      'Seguridad de la carga en tramos carreteros vulnerables.',
    ],
    pegasusSolution: [
      'Generación automática de datos para Complemento Carta Porte antes del despacho.',
      'Rastreo continuo con botón de pánico y geocercas inteligentes para seguridad de ruta.',
      'Coordinación de transfer y cruce fronterizo en una sola plataforma operativa.',
    ],
    stats: [
      { label: 'Conformidad Carta Porte', value: '100%' },
      { label: 'Tiempo de Espera Fronterizo', value: '-28%' },
      { label: 'Eficiencia de Despacho', value: '+42%' },
    ],
    faqs: [
      {
        question: '¿El sistema genera la información del Complemento Carta Porte del SAT?',
        answer: 'Sí. Todos los campos obligatorios requeridos por el SAT se recopilan digitalmente al confirmar la asignación del camión.',
      },
      {
        question: '¿Cómo apoya Pegasus a las empresas de manufactura en nearshoring?',
        answer: 'Pegasus conecta plantas en México con almacenes y clientes en EE.UU. a través de un único estado unificado de pedido.',
      },
    ],
  },
  {
    slug: 'canada',
    name: 'Canada',
    nativeName: 'Canada',
    region: 'North America',
    flag: '🇨🇦',
    currency: 'CAD ($)',
    primaryLanguage: 'English / French',
    cellCluster: 'cell-us (Google Cloud northamerica-northeast1 Montreal)',
    latency: '< 12ms Trans-Canada corridor',
    headline: 'Logistics Operating System & Supply Chain Software for Canada',
    summary: 'Conquer long-haul freight corridors, cross-border US-Canada customs, and bilingual Canadian retail distribution.',
    marketOverview: 'Canada requires resilient freight planning across vast geographical expanses, harsh winter weather conditions, and mandatory ELD compliance.',
    complianceFrameworks: ['Transport Canada ELD Mandate', 'CBSA ACI eManifest', 'USMCA Rules of Origin', 'PIP Security Program'],
    logisticsBottlenecks: [
      'Severe winter weather causing unexpected route diversions and delays.',
      'Cross-border customs holds with CBSA and US CBP.',
      'High long-haul fuel expenses across trans-continental corridors.',
    ],
    pegasusSolution: [
      'Weather-aware route suggestions and dynamic dispatcher reassignment.',
      'Digital manifest pairing matching CBSA eManifest cargo control numbers.',
      'Multi-stop load consolidation maximizing trailer cubic utilization.',
    ],
    stats: [
      { label: 'Fuel Savings Long-Haul', value: '19%' },
      { label: 'Customs Hold Reduction', value: '60%' },
      { label: 'Driver Satisfaction', value: '+35%' },
    ],
    faqs: [
      {
        question: 'Does Pegasus comply with Transport Canada electronic logging device (ELD) mandates?',
        answer: 'Yes. Pegasus tracks driver duty cycles and enforces safety rest parameters.',
      },
      {
        question: 'Is the platform available in both official Canadian languages (English & French)?',
        answer: 'Yes. All interfaces and mobile apps switch instantly between English and Canadian French.',
      },
    ],
  },
  {
    slug: 'australia',
    name: 'Australia',
    nativeName: 'Australia',
    region: 'Oceania',
    flag: '🇦🇺',
    currency: 'AUD ($)',
    primaryLanguage: 'English',
    cellCluster: 'cell-apac (Google Cloud australia-southeast1 Sydney)',
    latency: '< 15ms national freight network',
    headline: 'B2B Logistics Software & Long-Haul Fleet Optimization for Australia',
    summary: 'Coordinate interstate road train logistics, port drayage, and metropolitan retail replenishment across Australia.',
    marketOverview: 'Australia combines massive interstate freight distances between capital cities with dense last-mile urban logistics in Sydney and Melbourne.',
    complianceFrameworks: ['National Heavy Vehicle Regulator (NHVR)', 'Heavy Vehicle National Law (HVNL)', 'Chain of Responsibility (CoR)', 'Fatigue Management Rules'],
    logisticsBottlenecks: [
      'Strict Chain of Responsibility (CoR) liability laws holding shippers accountable for driver fatigue.',
      'Excessive deadhead kilometers on long-haul routes between Sydney, Melbourne, and Brisbane.',
      'Complex multi-trailer road train staging and axle weight limits.',
    ],
    pegasusSolution: [
      'Automated Chain of Responsibility (CoR) audit logging protecting shippers and carriers.',
      'Multi-depot backhaul matching algorithms eliminating empty return runs.',
      'Heavy vehicle routing respecting bridge clearances and mass management limits.',
    ],
    stats: [
      { label: 'Empty Backhaul Reduction', value: '34%' },
      { label: 'CoR Compliance Rate', value: '100%' },
      { label: 'Dispatch Scheduling Speed', value: '3x Faster' },
    ],
    faqs: [
      {
        question: 'How does Pegasus protect operators under Australia’s Chain of Responsibility (CoR) laws?',
        answer: 'Pegasus maintains immutable digital records showing that schedules, load weights, and route times were planned strictly within legal NHVR limits.',
      },
      {
        question: 'Can Pegasus handle Australian road trains and B-Double combinations?',
        answer: 'Yes. Fleet assets are configured with specific vehicle configurations, gross mass ratings, and trailer lengths.',
      },
    ],
  },
];

export function getMarketBySlug(slug: string): MarketProfile | undefined {
  return MARKETS_DATA.find((m) => m.slug === slug);
}
