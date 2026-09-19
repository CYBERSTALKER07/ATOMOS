import {
 CookieInventoryItem,
 ConsentCategories,
 CookieConsentRecord,
 CONSENT_STORAGE_KEY,
 CONSENT_COOKIE_NAME,
 CURRENT_POLICY_VERSION,
 JurisdictionCode,
} from './cookieTypes';

export const COOKIE_INVENTORY: CookieInventoryItem[] = [
 // 1. Strictly Necessary Cookies (Legal Basis: Art. 6(1)(f) GDPR / ePrivacy Directive Art. 5(3) exemption)
 {
 name: 'pegasus_session',
 category: 'necessary',
 provider: 'Pegasus Systems',
 party: 'First-Party',
 domain: '.pegasus.logistics',
 expiryEn: '30 days',
 expiryRu: '30 дней',
 purposeEn: 'Maintains authenticated supplier, warehouse, and dispatcher JWT sessions with cryptographic integrity.',
 purposeRu: 'Поддерживает аутентифицированную сессию поставщика, склада и диспетчера с криптографической защитой.',
 retentionType: 'Persistent',
 },
 {
 name: 'pegasus_csrf',
 category: 'necessary',
 provider: 'Pegasus Systems',
 party: 'First-Party',
 domain: '.pegasus.logistics',
 expiryEn: 'Session',
 expiryRu: 'Сессия',
 purposeEn: 'Protects mutation endpoints against Cross-Site Request Forgery (CSRF) attacks.',
 purposeRu: 'Защищает точки мутации данных от атак межсайтовой подделки запроса (CSRF).',
 retentionType: 'Session',
 },
 {
 name: 'pegasus_consent',
 category: 'necessary',
 provider: 'Pegasus Systems',
 party: 'First-Party',
 domain: '.pegasus.logistics',
 expiryEn: '12 months',
 expiryRu: '12 месяцев',
 purposeEn: 'Stores your granular cookie consent choices and timestamp to comply with GDPR Art. 7 audit requirements.',
 purposeRu: 'Хранит ваши параметры согласия на использование cookie и метку времени согласно ст. 7 GDPR.',
 retentionType: 'Persistent',
 },
 {
 name: 'pegasus-theme',
 category: 'necessary',
 provider: 'Pegasus Systems',
 party: 'First-Party',
 domain: '.pegasus.logistics',
 expiryEn: '12 months',
 expiryRu: '12 месяцев',
 purposeEn: 'Remembers tactical dark or operational light visual interface preference across page navigations.',
 purposeRu: 'Сохраняет визуальные настройки интерфейса (тёмная или светлая тема) между переходами по страницам.',
 retentionType: 'Persistent',
 },
 {
 name: 'pegasus_lang',
 category: 'necessary',
 provider: 'Pegasus Systems',
 party: 'First-Party',
 domain: '.pegasus.logistics',
 expiryEn: '12 months',
 expiryRu: '12 месяцев',
 purposeEn: 'Stores selected corridor interface language (English, Russian, Uzbek) to deliver statutory translations.',
 purposeRu: 'Сохраняет выбранный язык интерфейса (английский, русский, узбекский) для отображения переводов.',
 retentionType: 'Persistent',
 },
 {
 name: 'hasSeenSplash',
 category: 'necessary',
 provider: 'Pegasus Systems',
 party: 'First-Party',
 domain: '.pegasus.logistics',
 expiryEn: 'Session',
 expiryRu: 'Сессия',
 purposeEn: 'Prevents redundant display of the initial tactical initialization splash screen during a single visit.',
 purposeRu: 'Предотвращает повторное отображение экрана начальной загрузки в рамках одной рабочей сессии.',
 retentionType: 'Session',
 },

 // 2. Functional / Preference Cookies (Legal Basis: Art. 6(1)(a) GDPR Consent)
 {
 name: 'pegasus_ui_sidebar',
 category: 'functional',
 provider: 'Pegasus Systems',
 party: 'First-Party',
 domain: '.pegasus.logistics',
 expiryEn: '6 months',
 expiryRu: '6 месяцев',
 purposeEn: 'Remembers expanded or collapsed state of operations navigation rails and density panels.',
 purposeRu: 'Запоминает состояние навигационных панелей и плотности таблиц в рабочих экранах.',
 retentionType: 'Persistent',
 },
 {
 name: 'pegasus_corridor_filter',
 category: 'functional',
 provider: 'Pegasus Systems',
 party: 'First-Party',
 domain: '.pegasus.logistics',
 expiryEn: '3 months',
 expiryRu: '3 месяца',
 purposeEn: 'Persists active regional freight corridor filtering in tactical fleet and order monitoring views.',
 purposeRu: 'Сохраняет фильтры региональных коридоров доставки в мониторе автопарка и заказов.',
 retentionType: 'Persistent',
 },
 {
 name: 'pegasus_map_view',
 category: 'functional',
 provider: 'Pegasus Systems',
 party: 'First-Party',
 domain: '.pegasus.logistics',
 expiryEn: '3 months',
 expiryRu: '3 месяца',
 purposeEn: 'Stores preferred map zoom, tile layer, and geospatial perspective coordinates.',
 purposeRu: 'Сохраняет масштаб карты, слой отображения и координаты геопространственного мониторинга.',
 retentionType: 'Persistent',
 },

 // 3. Analytics & Performance Cookies (Legal Basis: Art. 6(1)(a) GDPR Consent)
 {
 name: '_pk_id',
 category: 'analytics',
 provider: 'Pegasus Telemetry (Self-Hosted)',
 party: 'First-Party',
 domain: '.pegasus.logistics',
 expiryEn: '13 months',
 expiryRu: '13 месяцев',
 purposeEn: 'Stores anonymized pseudonymized identifier to evaluate platform throughput, latency, and page speed.',
 purposeRu: 'Хранит обезличенный идентификатор для анализа производительности, задержек и скорости загрузки страниц.',
 retentionType: 'Persistent',
 },
 {
 name: '_pk_ses',
 category: 'analytics',
 provider: 'Pegasus Telemetry (Self-Hosted)',
 party: 'First-Party',
 domain: '.pegasus.logistics',
 expiryEn: '30 minutes',
 expiryRu: '30 минут',
 purposeEn: 'Temporary telemetry heartbeat session token measuring real-time API latency without user profiling.',
 purposeRu: 'Кратковременный телеметрический токен для замера задержек API в реальном времени без профилирования.',
 retentionType: 'Persistent',
 },
 {
 name: 'sentry_replay',
 category: 'analytics',
 provider: 'Sentry Diagnostics',
 party: 'Third-Party',
 domain: '.sentry.io',
 expiryEn: 'Session',
 expiryRu: 'Сессия',
 purposeEn: 'Captures anonymized client-side crash telemetry and call stack traces to debug UI exceptions.',
 purposeRu: 'Фиксирует обезличенные логи сбоев и трассировки для устранения программных ошибок в UI.',
 retentionType: 'Session',
 },

 // 4. Marketing & Targeting Cookies (Legal Basis: Art. 6(1)(a) GDPR Consent / CCPA Opt-Out)
 {
 name: '_li_fat_id',
 category: 'marketing',
 provider: 'LinkedIn Corporation',
 party: 'Third-Party',
 domain: '.linkedin.com',
 expiryEn: '30 days',
 expiryRu: '30 дней',
 purposeEn: 'Measures effectiveness of enterprise wholesale supply chain partner communications and recruitment.',
 purposeRu: 'Оценивает эффективность B2B-коммуникаций и привлечения корпоративных партнёров по цепочкам поставок.',
 retentionType: 'Persistent',
 },
 {
 name: '_gcl_au',
 category: 'marketing',
 provider: 'Google LLC',
 party: 'Third-Party',
 domain: '.google.com',
 expiryEn: '90 days',
 expiryRu: '90 дней',
 purposeEn: 'Attribution conversion tracker for verified enterprise supply chain procurement inquiries.',
 purposeRu: 'Атрибуция корпоративных заявок и обращений поставщиков в систему Pegasus.',
 retentionType: 'Persistent',
 },
];

export const DEFAULT_CONSENT: ConsentCategories = {
 necessary: true,
 functional: false,
 analytics: false,
 marketing: false,
};

export const ACCEPT_ALL_CONSENT: ConsentCategories = {
 necessary: true,
 functional: true,
 analytics: true,
 marketing: true,
};

export const REJECT_NON_ESSENTIAL_CONSENT: ConsentCategories = {
 necessary: true,
 functional: false,
 analytics: false,
 marketing: false,
};

/**
 * Checks if Global Privacy Control (GPC) is asserted by the user's browser or OS.
 * Recognized under CCPA/CPRA as a valid opt-out of sale/share/marketing.
 */
export function isGPCActive(): boolean {
 if (typeof window === 'undefined') return false;
 // eslint-disable-next-line @typescript-eslint/no-explicit-any
 const nav = navigator as any;
 return Boolean(nav.globalPrivacyControl === true || nav.globalPrivacyControl === '1');
}

/**
 * Reads stored consent record from localStorage and falls back to document.cookie.
 */
export function getStoredConsent(): CookieConsentRecord | null {
 if (typeof window === 'undefined') return null;

 try {
 const raw = localStorage.getItem(CONSENT_STORAGE_KEY);
 if (raw) {
 const parsed = JSON.parse(raw) as CookieConsentRecord;
 if (parsed && parsed.categories && typeof parsed.categories.necessary === 'boolean') {
 return parsed;
 }
 }
 } catch (e) {}

 // Fallback: check document.cookie
 try {
 const match = document.cookie
 .split('; ')
 .find((row) => row.startsWith(`${CONSENT_COOKIE_NAME}=`));
 if (match) {
 const value = decodeURIComponent(match.split('=')[1]);
 const parsed = JSON.parse(value) as CookieConsentRecord;
 if (parsed && parsed.categories) {
 return parsed;
 }
 }
 } catch (e) {}

 return null;
}

/**
 * Persists cookie consent record to localStorage, document.cookie, and dispatches custom event.
 */
export function saveConsentRecord(
 categories: ConsentCategories,
 jurisdiction: JurisdictionCode = 'GLOBAL'
): CookieConsentRecord {
 const gpc = isGPCActive();

 // If GPC is active, marketing MUST be disabled (CCPA/CPRA statutory requirement)
 const finalCategories: ConsentCategories = {
 ...categories,
 necessary: true, // Always locked
 marketing: gpc ? false : categories.marketing,
 };

 const record: CookieConsentRecord = {
 version: CURRENT_POLICY_VERSION,
 consentId: generateConsentId(),
 timestamp: new Date().toISOString(),
 categories: finalCategories,
 jurisdiction,
 gpcApplied: gpc,
 };

 if (typeof window !== 'undefined') {
 try {
 localStorage.setItem(CONSENT_STORAGE_KEY, JSON.stringify(record));
 } catch (e) {}

 try {
 // 12-month expiry for consent cookies per GDPR EDPB guidelines
 const maxAge = 365 * 24 * 60 * 60;
 const secureFlag = window.location.protocol === 'https:' ? '; Secure' : '';
 document.cookie = `${CONSENT_COOKIE_NAME}=${encodeURIComponent(
 JSON.stringify(record)
 )}; path=/; max-age=${maxAge}; SameSite=Lax${secureFlag}`;
 } catch (e) {}

 // Dispatch custom event for telemetry/marketing script gating
 try {
 window.dispatchEvent(
 new CustomEvent('pegasus:consent-updated', { detail: record })
 );
 } catch (e) {}
 }

 return record;
}

/**
 * Resets consent state to prompt the user again.
 */
export function resetConsentRecord(): void {
 if (typeof window === 'undefined') return;

 try {
 localStorage.removeItem(CONSENT_STORAGE_KEY);
 } catch (e) {}

 try {
 document.cookie = `${CONSENT_COOKIE_NAME}=; path=/; max-age=0; SameSite=Lax`;
 } catch (e) {}

 try {
 window.dispatchEvent(new CustomEvent('pegasus:consent-reset'));
 } catch (e) {}
}

/**
 * Generates an anonymous cryptographic consent proof ID for audit trail.
 */
function generateConsentId(): string {
 if (typeof crypto !== 'undefined' && crypto.randomUUID) {
 return crypto.randomUUID();
 }
 return 'c-' + Math.random().toString(36).substring(2, 15) + Date.now().toString(36);
}
