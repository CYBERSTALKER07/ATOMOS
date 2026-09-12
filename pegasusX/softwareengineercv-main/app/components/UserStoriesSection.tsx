'use client';

import React from 'react';
import Image from 'next/image';
import { useLanguage } from '@/app/context/LanguageContext';

type UserStoryItem = {
  id: string;
  name: string;
  role: string;
  avatar: string;
  personDescription: string;
  userStory: string;
};

const MULTILINGUAL_STORIES: Record<string, { title: string; stories: UserStoryItem[] }> = {
  ru: {
    title: 'User stories',
    stories: [
      {
        id: 'alexey',
        name: 'Алексей',
        role: 'Владелец бизнеса',
        avatar: '/Unknown-5.jpg',
        personDescription:
          'Владелец небольшого бизнеса, специализирующегося на производстве и продаже спецодежды. Его компания предлагает разнообразные модели спецодежды для оптовых и розничных покупателей.',
        userStory:
          'Алексей, как владелец бизнеса хочет увидеть баланс своего счета и информацию о состоянии расчетов, чтобы понять, какую сумму ему нужно перевести контрагентам по совершенным сделкам.',
      },
      {
        id: 'elena',
        name: 'Елена',
        role: 'Менеджер по работе с клиентами',
        avatar: '/Unknown-6.jpg',
        personDescription:
          'Менеджер по работе с клиентами. Она проводит переговоры, заключает договоры и оформляет все необходимые документы.',
        userStory:
          'Елена, как менеджер по работе с клиентами, хочет оперативно выставлять счета клиентам, чтобы они могли быстрее оплатить их и не тормозить рабочий процесс.',
      },
    ],
  },
  en: {
    title: 'User stories',
    stories: [
      {
        id: 'alexey',
        name: 'Alexey',
        role: 'Business Owner',
        avatar: '/Unknown-5.jpg',
        personDescription:
          'Owner of a manufacturing and distribution business specializing in industrial apparel. His company supplies customized workwear to wholesale and retail buyers.',
        userStory:
          'Alexey, as a business owner, wants to view his real-time account balance and settlement status to clearly understand how much he needs to transfer to trade counterparties for completed transactions.',
      },
      {
        id: 'elena',
        name: 'Elena',
        role: 'Client Relationship Manager',
        avatar: '/Unknown-6.jpg',
        personDescription:
          'Client relationship manager handling commercial negotiations, contractual agreements, and generating all necessary documentation for client accounts.',
        userStory:
          'Elena, as a client manager, wants to promptly issue invoices to clients so they can settle payments faster without stalling operational workflows.',
      },
    ],
  },
  uz: {
    title: 'Foydalanuvchi hikoyalari',
    stories: [
      {
        id: 'alexey',
        name: 'Aleksey',
        role: 'Biznes egasi',
        avatar: '/Unknown-5.jpg',
        personDescription:
          'Maxsus ish kiyimlari ishlab chiqarish va sotishga ixtisoslashgan kichik biznes egasi. Uning kompaniyasi ulgurji va chakana xaridorlar uchun turli modellarni taklif etadi.',
        userStory:
          'Aleksey, biznes egasi sifatida, hisob-kitob holati va hisob balansini aniq ko‘rishni xohlaydi, shunda u tuzilgan bitimlar bo‘yicha kontragentlarga qancha mablag‘ o‘tkazishi kerakligini biladi.',
      },
      {
        id: 'elena',
        name: 'Yelena',
        role: 'Mijozlar bilan ishlash menejeri',
        avatar: '/Unknown-6.jpg',
        personDescription:
          'Mijozlar bilan ishlash menejeri. U muzokaralar olib boradi, shartnomalar tuzadi va barcha zarur hujjatlarni rasmiylashtiradi.',
        userStory:
          'Yelena, mijozlar menejeri sifatida, mijozlarga hisob-fakturalarni tezkorlik bilan chiqarishni xohlaydi, shunda ular to‘lovni tezroq amalga oshirib, ish jarayonini to‘xtatib qo‘ymaydi.',
      },
    ],
  },
  es: {
    title: 'Historias de usuario',
    stories: [
      {
        id: 'alexey',
        name: 'Alexey',
        role: 'Propietario de negocio',
        avatar: '/Unknown-5.jpg',
        personDescription:
          'Propietario de una pequeña empresa especializada en la confección y venta de ropa de trabajo para compradores mayoristas y minoristas.',
        userStory:
          'Alexey, como propietario de una empresa, quiere ver el saldo de su cuenta y el estado de liquidación para saber cuánto transferir a las contrapartes por acuerdos concluidos.',
      },
      {
        id: 'elena',
        name: 'Elena',
        role: 'Gerente de cuentas',
        avatar: '/Unknown-6.jpg',
        personDescription:
          'Gerente de relaciones con clientes. Conduce negociaciones comerciales, formaliza contratos y emite toda la documentación requerida.',
        userStory:
          'Elena, como gerente de cuentas, quiere emitir facturas a los clientes de inmediato para que puedan pagar más rápido y no retrasar el flujo de trabajo.',
      },
    ],
  },
  de: {
    title: 'User Stories',
    stories: [
      {
        id: 'alexey',
        name: 'Alexey',
        role: 'Geschäftsinhaber',
        avatar: '/Unknown-5.jpg',
        personDescription:
          'Inhaber eines mittelständischen Unternehmens für Berufsbekleidung. Beliefert Groß- und Einzelhandelskunden mit modernen Arbeitskleidungsmodellen.',
        userStory:
          'Alexey möchte als Geschäftsinhaber seinen Kontostand und den aktuellen Abrechnungsstatus einsehen, um zu verstehen, welche Beträge an Geschäftspartner zu überweisen sind.',
      },
      {
        id: 'elena',
        name: 'Elena',
        role: 'Kundenbetreuerin',
        avatar: '/Unknown-6.jpg',
        personDescription:
          'Kundenbetreuerin für Handelspartner. Führt Verhandlungen, schließt Verträge ab und erstellt alle erforderlichen Versanddokumente.',
        userStory:
          'Elena möchte Rechnungen schnell an Kunden ausstellen, damit diese zügig bezahlen und der operative Prozess nicht ins Stocken gerät.',
      },
    ],
  },
  fr: {
    title: 'Histoires utilisateurs',
    stories: [
      {
        id: 'alexey',
        name: 'Alexeï',
        role: 'Chef d’entreprise',
        avatar: '/Unknown-5.jpg',
        personDescription:
          'Propriétaire d’une entreprise spécialisée dans la fabrication et la distribution de vêtements professionnels pour clients grossistes et détaillants.',
        userStory:
          'Alexeï souhaite visualiser le solde de son compte et l’état des règlements pour savoir quel montant transférer à ses contreparties pour les transactions conclues.',
      },
      {
        id: 'elena',
        name: 'Elena',
        role: 'Responsable clientèle',
        avatar: '/Unknown-6.jpg',
        personDescription:
          'Responsable clientèle menant les négociations, la signature des contrats et l’émission de l’ensemble des pièces administratives.',
        userStory:
          'Elena souhaite facturer rapidement les clients afin d’accélérer les paiements et d’assurer la fluidité continue des opérations.',
      },
    ],
  },
  zh: {
    title: '用户故事',
    stories: [
      {
        id: 'alexey',
        name: '阿列克谢',
        role: '企业所有者',
        avatar: '/Unknown-5.jpg',
        personDescription:
          '一家专注于工业工作服制造与销售的企业负责人，为批发和零售买家提供丰富的产品线。',
        userStory:
          '阿列克谢希望实时查看其账户余额与清算状态，以便清楚掌握各项交易所产生应付账款，及时向交易方结算。',
      },
      {
        id: 'elena',
        name: '埃琳娜',
        role: '客户关系经理',
        avatar: '/Unknown-6.jpg',
        personDescription:
          '负责商务洽谈、签署合作合同并出具各项必要交易凭据的客户关系经理。',
        userStory:
          '埃琳娜希望能够迅速向客户开具账单发票，以便客户加快付款速度，不阻碍订单履约与生产流程。',
      },
    ],
  },
  ja: {
    title: 'ユーザーストーリー',
    stories: [
      {
        id: 'alexey',
        name: 'アレクセイ',
        role: '事業主',
        avatar: '/Unknown-5.jpg',
        personDescription:
          '作業着の製造・販売を専門とするビジネスのオーナー。卸売および小売の顧客に向けて製品を展開しています。',
        userStory:
          'アレクセイは事業主として、完了した取引に対して取引先へ送金すべき金額を把握するため、口座残高と決済状況を即座に確認したいと考えています。',
      },
      {
        id: 'elena',
        name: 'エレーナ',
        role: 'アカウントマネージャー',
        avatar: '/Unknown-6.jpg',
        personDescription:
          '顧客関係を担当するマネージャー。商談を行い、契約を締結し、必要なすべての書類を発行します。',
        userStory:
          'エレーナは顧客マネージャーとして、顧客が迅速に支払いを行い業務の流れを滞らせないよう、速やかに請求書を発行したいと考えています。',
      },
    ],
  },
  ar: {
    title: 'قصص المستخدمين',
    stories: [
      {
        id: 'alexey',
        name: 'أليكسي',
        role: 'صاحب عمل',
        avatar: '/Unknown-5.jpg',
        personDescription:
          'صاحب عمل متخصص في تصنيع وبيع ملابس العمل للشركات وتجار الجملة والتجزئة.',
        userStory:
          'يرغب أليكسي كصاحب عمل في الاطلاع على رصيد حسابه وحالة التسويات ليعرف بدقة المبالغ المستحقة للأطراف المقابلة.',
      },
      {
        id: 'elena',
        name: 'إيلينا',
        role: 'مديرة علاقات العملاء',
        avatar: '/Unknown-6.jpg',
        personDescription:
          'مديرة حسابات وعلاقات العملاء. تجري المفاوضات وتبرم العقود وتصدر كافة المستندات المطلوبة.',
        userStory:
          'ترغب إيلينا في إصدار الفواتير بسرعة للعملاء حتى يتمكنوا من الدفع دون تأخير سير العمل التشغيلي.',
      },
    ],
  },
  pt: {
    title: 'Histórias de usuário',
    stories: [
      {
        id: 'alexey',
        name: 'Alexey',
        role: 'Proprietário',
        avatar: '/Unknown-5.jpg',
        personDescription:
          'Proprietário de uma empresa especializada na produção e venda de vestuário de trabalho para atacado e varejo.',
        userStory:
          'Alexey, como proprietário, quer ver o saldo de sua conta e o status dos acertos para saber o valor exato a ser transferido aos parceiros comerciais.',
      },
      {
        id: 'elena',
        name: 'Elena',
        role: 'Gerente de contas',
        avatar: '/Unknown-6.jpg',
        personDescription:
          'Gerente de contas comerciais. Conduz negociações, fecha contratos e emite toda a documentação necessária.',
        userStory:
          'Elena quer emitir faturas aos clientes com agilidade para que eles possam pagar mais rápido sem travar os fluxos operacionais.',
      },
    ],
  },
  tr: {
    title: 'Kullanıcı hikayeleri',
    stories: [
      {
        id: 'alexey',
        name: 'Aleksey',
        role: 'İşletme Sahibi',
        avatar: '/Unknown-5.jpg',
        personDescription:
          'İş kıyafetleri üretimi ve toptan/perakende satışı konusunda uzmanlaşmış işletme sahibi.',
        userStory:
          'Aleksey, tamamlanan işlemler doğrultusunda iş ortaklarına ne kadar ödeme yapacağını bilmek için hesap bakiyesini ve mutabakat durumunu net görmek istiyor.',
      },
      {
        id: 'elena',
        name: 'Elena',
        role: 'Müşteri Yöneticisi',
        avatar: '/Unknown-6.jpg',
        personDescription:
          'Müşteri ilişkileri yöneticisi. Görüşmeleri yürütür, sözleşmeleri imzalar ve tüm operasyonel belgeleri hazırlar.',
        userStory:
          'Elena, müşterilerin ödemeleri geciktirmeden yapabilmesi ve iş akışının aksamaması için faturaları hızlıca kesmek istiyor.',
      },
    ],
  },
};

export default function UserStoriesSection() {
  const { language } = useLanguage();
  const currentLocale = MULTILINGUAL_STORIES[language] || MULTILINGUAL_STORIES.en;
  const { title, stories } = currentLocale;

  return (
    <section className="relative w-full bg-[#000000] text-white py-20 lg:py-28 overflow-hidden font-sans border-t border-[#1C1C24]">
      {/* Subtle ambient lighting */}
      <div className="absolute top-0 right-0 w-[500px] h-[500px] bg-[radial-gradient(ellipse_at_top_right,_rgba(50,50,70,0.2)_0%,_transparent_70%)] pointer-events-none" />
      <div className="absolute bottom-10 left-10 w-[400px] h-[400px] bg-[radial-gradient(circle,_rgba(25,25,35,0.35)_0%,_transparent_70%)] pointer-events-none" />

      <div className="max-w-[1240px] mx-auto px-6 sm:px-8 lg:px-12">
        {/* Clean Big Section Title */}
        <div className="pb-16 sm:pb-20">
          <h2 className="text-6xl sm:text-7xl lg:text-8xl font-medium tracking-tight text-white select-none">
            {title}
          </h2>
        </div>

        {/* Stories Flow: Z-Layout (Alternating Left & Right) */}
        <div className="space-y-24 lg:space-y-32">
          {/* Row 1: Left Persona (Алексей) -> Right Story Card */}
          <div className="relative">
            {/* Persona 1 (Left) */}
            <div className="max-w-xl">
              <div className="flex items-center space-x-4">
                <div className="relative w-14 h-14 sm:w-16 sm:h-16 rounded-full overflow-hidden border border-white/20 shrink-0 bg-[#161622] shadow-lg">
                  <Image
                    src={stories[0].avatar}
                    alt={stories[0].name}
                    fill
                    className="object-cover"
                    sizes="(max-width: 640px) 56px, 64px"
                  />
                </div>
                <div>
                  <h3 className="text-2xl sm:text-3xl font-medium text-white tracking-tight">
                    {stories[0].name}
                  </h3>
                  <p className="text-xs font-mono text-[#8E8EA0] uppercase tracking-wider mt-0.5">
                    {stories[0].role}
                  </p>
                </div>
              </div>

              <p className="mt-4 text-xs sm:text-sm text-[#8E8EA8] leading-relaxed max-w-lg">
                {stories[0].personDescription}
              </p>
            </div>

            {/* Curved Connector Hairline (Flowing Left to Right) */}
            <div className="hidden md:block absolute left-20 top-[135px] w-48 h-20 border-l border-b border-[#2A2A38] rounded-bl-3xl pointer-events-none opacity-60" />

            {/* User Story Card 1 (Right) */}
            <div className="mt-6 md:mt-2 md:ml-auto max-w-xl lg:max-w-2xl flex flex-col items-end">
              <div className="relative w-full bg-white text-[#111116] p-6 sm:p-7 rounded-2xl shadow-[0_12px_40px_rgba(0,0,0,0.6)]">
                <p className="text-sm sm:text-base leading-relaxed font-normal text-[#1A1A24]">
                  {stories[0].userStory}
                </p>
              </div>
            </div>
          </div>

          {/* Row 2: Right Persona (Елена) -> Left Story Card (Z-Pattern Mirror) */}
          <div className="relative">
            {/* Persona 2 (Right on Desktop) */}
            <div className="max-w-xl md:ml-auto md:flex md:flex-col md:items-end md:text-right">
              <div className="flex items-center space-x-4 md:flex-row-reverse md:space-x-reverse">
                <div className="relative w-14 h-14 sm:w-16 sm:h-16 rounded-full overflow-hidden border border-white/20 shrink-0 bg-[#161622] shadow-lg">
                  <Image
                    src={stories[1].avatar}
                    alt={stories[1].name}
                    fill
                    className="object-cover"
                    sizes="(max-width: 640px) 56px, 64px"
                  />
                </div>
                <div>
                  <h3 className="text-2xl sm:text-3xl font-medium text-white tracking-tight">
                    {stories[1].name}
                  </h3>
                  <p className="text-xs font-mono text-[#8E8EA0] uppercase tracking-wider mt-0.5">
                    {stories[1].role}
                  </p>
                </div>
              </div>

              <p className="mt-4 text-xs sm:text-sm text-[#8E8EA8] leading-relaxed max-w-lg">
                {stories[1].personDescription}
              </p>
            </div>

            {/* Curved Connector Hairline (Flowing Right to Left) */}
            <div className="hidden md:block absolute right-20 top-[135px] w-48 h-20 border-r border-b border-[#2A2A38] rounded-br-3xl pointer-events-none opacity-60" />

            {/* User Story Card 2 (Left on Desktop) */}
            <div className="mt-6 md:mt-2 md:mr-auto max-w-xl lg:max-w-2xl flex flex-col items-start">
              <div className="relative w-full bg-white text-[#111116] p-6 sm:p-7 rounded-2xl shadow-[0_12px_40px_rgba(0,0,0,0.6)]">
                <p className="text-sm sm:text-base leading-relaxed font-normal text-[#1A1A24]">
                  {stories[1].userStory}
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
