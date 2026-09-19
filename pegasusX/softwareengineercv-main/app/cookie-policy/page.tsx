import type { Metadata } from 'next';
import Link from 'next/link';
import SiteNav from '@/app/components/explore/SiteNav';
import SubpageHero from '@/app/components/SubpageHero';
import Footer from '@/app/components/Footer';
import { getServerLanguage } from '@/app/lib/i18n/server';
import { breadcrumbJsonLd, jsonLdScript, pageMetadata } from '@/app/lib/seo';
import { COOKIE_INVENTORY } from '@/app/lib/cookies/cookieRegistry';
import CookiePolicyInteractivePanel from './CookiePolicyInteractivePanel';

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getServerLanguage();
  const isRu = lang === 'ru';

  return pageMetadata({
    title: isRu ? 'Политика использования файлов cookie' : 'Enterprise Cookie & Telemetry Policy',
    description: isRu
      ? 'Политика использования файлов cookie и техническое раскрытие телеметрии платформы Pegasus в соответствии с GDPR, ePrivacy, CCPA и Законом РУз ЗРУ-547.'
      : 'Comprehensive Enterprise Cookie Policy and Technical Telemetry Disclosure for Pegasus Logistics Operating System compliant with GDPR, ePrivacy, CCPA, and Law No. ZRU-547.',
    path: '/cookie-policy',
    language: lang,
  });
}

export default async function CookiePolicyPage() {
  const lang = await getServerLanguage();
  const isRu = lang === 'ru';

  const homeLabel = isRu ? 'Главная' : 'Home';
  const cookiePolicyLabel = isRu ? 'Политика cookie' : 'Cookie Policy';

  const breadcrumbs = breadcrumbJsonLd([
    { name: homeLabel, path: '/' },
    { name: cookiePolicyLabel, path: '/cookie-policy' },
  ]);

  return (
    <>
      <script type="application/ld+json" dangerouslySetInnerHTML={jsonLdScript(breadcrumbs)} />
      <div className="min-h-screen bg-black text-white selection:bg-white selection:text-black">
        <SiteNav activeHref="/cookie-policy" />

        <SubpageHero
          badge={isRu ? 'ПРАВОВОЕ РАСКРЫТИЕ & КОМПЛАЕНС' : 'LEGAL DISCLOSURE & COMPLIANCE'}
          title={isRu ? 'Политика использования файлов cookie' : 'Enterprise Cookie & Telemetry Policy'}
          summary={isRu
            ? 'Политика использования файлов cookie и техническое раскрытие телеметрии платформы Pegasus в соответствии с GDPR, ePrivacy, CCPA и Законом РУз ЗРУ-547.'
            : 'Comprehensive Enterprise Cookie Policy and Technical Telemetry Disclosure for Pegasus Logistics Operating System compliant with GDPR, ePrivacy, CCPA, and Law No. ZRU-547.'}
          primaryCta={{
            label: isRu ? 'Панель согласия' : 'Consent Preferences',
            href: '#consent',
          }}
          secondaryCta={{
            label: isRu ? 'Главная' : 'Return Home',
            href: '/',
          }}
          widget={{
            title: 'GDPR · CCPA · ZRU-547',
            description: 'Zero unauthorized trackers. Cryptographically signed audit trail.',
            href: '#consent',
          }}
          breadcrumb={{
            homeLabel: homeLabel,
            currentPage: cookiePolicyLabel,
          }}
        />

        <main id="consent" className="max-w-5xl mx-auto px-6 sm:px-8 pt-10 pb-24">

          {/* Interactive Live Consent Control Box */}
          <section className="mb-14" aria-label="Consent Management">
            <CookiePolicyInteractivePanel />
          </section>

          {/* Legal Text Document */}
          <article className="prose prose-zinc dark:prose-invert max-w-none space-y-10 text-sm leading-relaxed text-zinc-700 dark:text-white/75">
            {/* Section 1 */}
            <section className="space-y-3 border-t border-black/10 dark:border-white/10 pt-8">
              <h2 className="text-xl font-semibold tracking-tight text-zinc-900 dark:text-white">
                {isRu ? '1. Нормативно-правовая база и сфера применения' : '1. Statutory Authority & Operational Scope'}
              </h2>
              <p>
                {isRu
                  ? 'Настоящая Политика использования файлов cookie («Политика») определяет принципы, технические механизмы и правовые основания применения файлов cookie и аналогичных технологий отслеживания на платформе Pegasus (включая веб-приложения, API-шлюзы и мобильные клиенты). Настоящий документ составлен в строгом соответствии с:'
                  : 'This Enterprise Cookie & Telemetry Policy ("Policy") governs the operational principles, technical mechanisms, and lawful bases for deploying HTTP cookies, local storage objects, and telemetry agents within the Pegasus Logistics Operating System. This disclosure strictly complies with:'}
              </p>
              <ul className="list-disc pl-5 space-y-1 font-mono text-xs">
                <li>
                  <strong>Regulation (EU) 2016/679 (GDPR)</strong>: Articles 6 (Lawfulness of processing), 7 (Conditions for consent), 12–14 (Transparency and data subject rights).
                </li>
                <li>
                  <strong>Directive 2002/58/EC (ePrivacy Directive)</strong>: Article 5(3) as amended by Directive 2009/136/EC regarding subscriber consent for terminal equipment access.
                </li>
                <li>
                  <strong>California Consumer Privacy Act (CCPA / CPRA)</strong>: Cal. Civ. Code § 1798.100 et seq. regarding notices at collection and opt-out of personal data sharing.
                </li>
                <li>
                  <strong>United Kingdom Data Protection Act 2018 & PECR</strong>: Regulatory guidance on consent verification and non-essential cookie gating.
                </li>
                <li>
                  <strong>Law of the Republic of Uzbekistan No. ZRU-547</strong>: Dated July 2, 2019 "On Personal Data" regulating the collection, systematization, and cross-border processing of digital identifiers.
                </li>
              </ul>
            </section>

            {/* Section 2 */}
            <section className="space-y-3 border-t border-black/10 dark:border-white/10 pt-8">
              <h2 className="text-xl font-semibold tracking-tight text-zinc-900 dark:text-white">
                {isRu ? '2. Что такое файлы cookie и технологии локального хранения' : '2. Technical Definitions: Cookies & Storage Technologies'}
              </h2>
              <p>
                {isRu
                  ? 'Файлы cookie — это небольшие текстовые фрагменты данных, передаваемые веб-сервером и сохраняемые браузером вашего устройства. Наряду с HTTP-cookie, платформа Pegasus использует смежные технологии браузерного хранения:'
                  : 'Cookies are standardized key-value text pairs generated by our web servers and stored locally by your browser client. In conjunction with traditional HTTP cookies, the Pegasus platform leverages adjacent client storage primitives:'}
              </p>
              <ul className="list-disc pl-5 space-y-2">
                <li>
                  <strong>HTTP Cookies (First-Party & Third-Party)</strong>: Small state containers transmitted with each network request via <code>Cookie</code> and <code>Set-Cookie</code> headers, protected by <code>SameSite=Lax</code> and <code>Secure</code> flags.
                </li>
                <li>
                  <strong>Web Storage API (localStorage & sessionStorage)</strong>: Sandboxed client-side key-value stores used to cache non-sensitive tactical UI state (such as dashboard table column widths and dark/light color tokens) without sending overhead bytes on every API trip.
                </li>
                <li>
                  <strong>Session Tokens</strong>: Cryptographically signed JWT identifiers maintaining zero-trust authorization across distributed fleet microservices.
                </li>
              </ul>
            </section>

            {/* Section 3 */}
            <section className="space-y-3 border-t border-black/10 dark:border-white/10 pt-8">
              <h2 className="text-xl font-semibold tracking-tight text-zinc-900 dark:text-white">
                {isRu ? '3. Классификация и правовые основания обработки' : '3. Cookie Classification & Lawful Grounds for Processing'}
              </h2>
              <p>
                {isRu
                  ? 'Все используемые технологии классифицируются по четырём изолированным категориям с независимым контролем доступа:'
                  : 'Pegasus partitions all browser identifiers into four distinct, audited architectural tiers with discrete consent lifecycles:'}
              </p>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4 pt-2">
                <div className="p-4 rounded-none border border-black/10 dark:border-white/10 bg-white dark:bg-black">
                  <h3 className="font-semibold text-zinc-900 dark:text-white text-sm mb-1">
                    {isRu ? 'A. Строго обязательные' : 'Tier A: Strictly Necessary'}
                  </h3>
                  <p className="text-xs text-zinc-600 dark:text-white/60 mb-2">
                    {isRu
                      ? 'Обеспечивают аутентификацию, предотвращение мошенничества, защиту от CSRF и маршрутизацию сессий.'
                      : 'Essential for cryptographic authentication, CSRF mutation gating, and distributed session routing.'}
                  </p>
                  <span className="text-[10px] font-mono text-zinc-600 dark:text-zinc-400">
                    Legal Basis: GDPR Art. 6(1)(f) (Legitimate Interest) / ePrivacy Art. 5(3) Exemption
                  </span>
                </div>

                <div className="p-4 rounded-none border border-black/10 dark:border-white/10 bg-white dark:bg-black">
                  <h3 className="font-semibold text-zinc-900 dark:text-white text-sm mb-1">
                    {isRu ? 'B. Функциональные настройки' : 'Tier B: Functional Preferences'}
                  </h3>
                  <p className="text-xs text-zinc-600 dark:text-white/60 mb-2">
                    {isRu
                      ? 'Запоминают рабочие фильтры автопарка, выбранные транспортные коридоры и масштабирование карт.'
                      : 'Preserves freight corridor filters, map perspective, and density settings across sessions.'}
                  </p>
                  <span className="text-[10px] font-mono text-zinc-600 dark:text-zinc-400">
                    Legal Basis: GDPR Art. 6(1)(a) (Explicit Prior Consent)
                  </span>
                </div>

                <div className="p-4 rounded-none border border-black/10 dark:border-white/10 bg-white dark:bg-black">
                  <h3 className="font-semibold text-zinc-900 dark:text-white text-sm mb-1">
                    {isRu ? 'C. Аналитика и производительность' : 'Tier C: Analytics & Telemetry'}
                  </h3>
                  <p className="text-xs text-zinc-600 dark:text-white/60 mb-2">
                    {isRu
                      ? 'Обезличенный мониторинг latency API, трассировка исключений и аудит пропускной способности.'
                      : 'Self-hosted pseudonymized latency monitoring and crash diagnostics without profiling.'}
                  </p>
                  <span className="text-[10px] font-mono text-zinc-600 dark:text-zinc-400">
                    Legal Basis: GDPR Art. 6(1)(a) (Explicit Prior Consent)
                  </span>
                </div>

                <div className="p-4 rounded-none border border-black/10 dark:border-white/10 bg-white dark:bg-black">
                  <h3 className="font-semibold text-zinc-900 dark:text-white text-sm mb-1">
                    {isRu ? 'D. Маркетинг и партнерская сеть' : 'Tier D: Marketing & Partner Attribution'}
                  </h3>
                  <p className="text-xs text-zinc-600 dark:text-white/60 mb-2">
                    {isRu
                      ? 'Атрибуция заявок на подключение корпоративных перевозчиков и оптовых поставщиков.'
                      : 'Attribution tracking for enterprise supply chain partner recruitment and wholesale inquiries.'}
                  </p>
                  <span className="text-[10px] font-mono text-zinc-600 dark:text-zinc-400">
                    Legal Basis: GDPR Art. 6(1)(a) Consent / CCPA Opt-Out Gated
                  </span>
                </div>
              </div>
            </section>

            {/* Section 4: Full Cookie Inventory Table */}
            <section className="space-y-3 border-t border-black/10 dark:border-white/10 pt-8">
              <h2 className="text-xl font-semibold tracking-tight text-zinc-900 dark:text-white">
                {isRu ? '4. Полный реестр используемых файлов cookie' : '4. Exhaustive Technical Cookie Inventory'}
              </h2>
              <p>
                {isRu
                  ? 'Ниже представлен детальный реестр всех файлов cookie и браузерных токенов, зарегистрированных в производственной инфраструктуре Pegasus:'
                  : 'The table below provides an audited, line-item disclosure of each identifier deployed within the Pegasus production runtime:'}
              </p>

              <div className="overflow-x-auto border border-black/10 dark:border-white/10 rounded-none bg-white dark:bg-black mt-4">
                <table className="w-full text-left border-collapse text-xs font-mono">
                  <thead>
                    <tr className="border-b border-black/10 dark:border-white/10 bg-zinc-50 dark:bg-zinc-950 text-zinc-500 dark:text-white/50 uppercase tracking-wider text-[10px]">
                      <th className="p-3">Identifier</th>
                      <th className="p-3">Category</th>
                      <th className="p-3">Provider / Host</th>
                      <th className="p-3">Retention</th>
                      <th className="p-3 font-sans">Purpose Description</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-black/5 dark:divide-white/5">
                    {COOKIE_INVENTORY.map((c) => (
                      <tr key={c.name} className="hover:bg-zinc-50/50 dark:hover:bg-white/5 transition-colors">
                        <td className="p-3 font-bold text-zinc-900 dark:text-white whitespace-nowrap">
                          {c.name}
                        </td>
                        <td className="p-3 whitespace-nowrap">
                          <span className="px-1.5 py-0.5 rounded text-[9px] uppercase tracking-wider font-semibold bg-white/10 text-white border border-white/20">
                            {c.category}
                          </span>
                        </td>
                        <td className="p-3 text-zinc-500 dark:text-white/60 whitespace-nowrap">
                          {c.provider}
                        </td>
                        <td className="p-3 text-zinc-500 dark:text-white/60 whitespace-nowrap">
                          {isRu ? c.expiryRu : c.expiryEn}
                        </td>
                        <td className="p-3 font-sans text-[11px] text-zinc-600 dark:text-white/70 max-w-xs sm:max-w-md">
                          {isRu ? c.purposeRu : c.purposeEn}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>

            {/* Section 5 */}
            <section className="space-y-3 border-t border-black/10 dark:border-white/10 pt-8">
              <h2 className="text-xl font-semibold tracking-tight text-zinc-900 dark:text-white">
                {isRu ? '5. Сигнал Global Privacy Control (GPC) и отказ от передачи данных' : '5. Global Privacy Control (GPC) & Do Not Sell / Share Rights'}
              </h2>
              <p>
                {isRu
                  ? 'Платформа Pegasus поддерживает и автоматически распознаёт спецификацию Global Privacy Control (GPC). Если в вашем браузере включен GPC (передаётся заголовок Sec-GPC: 1 или свойство navigator.globalPrivacyControl === true), маркетинговые и аналитические трекеры блокируются автоматически без необходимости ручного отключения.'
                  : 'Pegasus natively honors the Global Privacy Control (GPC) signal transmitted by compatible browsers (such as Brave, Firefox, or privacy extensions). When `navigator.globalPrivacyControl === true` or the `Sec-GPC: 1` header is detected, marketing and advertising cookies are programmatically locked in the disabled state in compliance with the California Consumer Privacy Act (CCPA/CPRA).'}
              </p>
            </section>

            {/* Section 6 */}
            <section className="space-y-3 border-t border-black/10 dark:border-white/10 pt-8">
              <h2 className="text-xl font-semibold tracking-tight text-zinc-900 dark:text-white">
                {isRu ? '6. Порядок изменения и отзыва согласия' : '6. Modifying or Revoking Consent'}
              </h2>
              <p>
                {isRu
                  ? 'В соответствии со ст. 7(3) GDPR и законодательством о защите персональных данных, вы имеете безусловное право в любой момент изменить или полностью отозвать ранее предоставленное согласие. Вы можете сделать это:'
                  : 'Pursuant to GDPR Article 7(3), you have the unconditional right to modify or revoke your consent at any time without penalty or loss of fundamental platform service. You may exercise this right by:'}
              </p>
              <ul className="list-disc pl-5 space-y-1">
                <li>
                  {isRu
                    ? 'Нажав кнопку «Изменить настройки cookie» в интерактивной панели в начале этой страницы или в нижней части сайта.'
                    : 'Clicking "Change Preferences" in the interactive control panel above or in the footer on any page.'}
                </li>
                <li>
                  {isRu
                    ? 'Очистив историю и файлы cookie в настройках вашего веб-браузера, что вызовет повторное отображение баннера выбора.'
                    : 'Clearing your browser cookies and site storage for pegasus.logistics, triggering a fresh consent prompt upon next visit.'}
                </li>
              </ul>
            </section>

            {/* Section 7 */}
            <section className="space-y-3 border-t border-black/10 dark:border-white/10 pt-8">
              <h2 className="text-xl font-semibold tracking-tight text-zinc-900 dark:text-white">
                {isRu ? '7. Контакты службы защиты данных (DPO)' : '7. Data Protection Officer (DPO) & Inquiries'}
              </h2>
              <p>
                {isRu
                  ? 'По всем вопросам, касающимся настоящей Политики, аудита согласий или реализации прав субъекта данных, вы можете обратиться к нашему сотруднику по защите данных:'
                  : 'For technical inquiries, consent audit verifications, or data subject access requests concerning browser telemetry, contact our Data Protection Office:'}
              </p>
              <div className="p-4 rounded-none border border-black/10 dark:border-white/10 bg-zinc-50 dark:bg-black font-mono text-xs space-y-1">
                <div><strong>Pegasus Legal & Compliance Office</strong></div>
                <div>Email: <a href="mailto:dpo@pegasus.logistics" className="underline text-zinc-900 dark:text-white">dpo@pegasus.logistics</a></div>
                <div>Jurisdiction: Tashkent, Uzbekistan / Global Transit Corridor Operations</div>
              </div>
            </section>

            {/* Legal Disclaimer Box */}
            <div className="p-4 rounded-none border border-black/10 dark:border-white/10 bg-zinc-100 dark:bg-black/60 text-xs text-zinc-500 dark:text-white/50 italic">
              {isRu
                ? 'Примечание: Данный документ является техническим описанием и стандартом комплаенса платформы Pegasus. По конкретным юридическим вопросам регулирования в вашей юрисдикции рекомендуется проконсультироваться с квалифицированным юристом.'
                : 'Disclaimer: This document serves as the operational technical disclosure for the Pegasus platform. Consult with qualified legal counsel regarding jurisdiction-specific organizational implementation.'}
            </div>
          </article>
        </main>

        <Footer />
      </div>
    </>
  );
}
