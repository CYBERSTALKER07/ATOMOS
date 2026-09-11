'use client';

import React, { useState } from 'react';
import Image from 'next/image';
import { User, FileText, Sparkles } from 'lucide-react';
import { useLanguage } from '@/app/context/LanguageContext';

type UserStoryItem = {
  id: string;
  name: string;
  role: string;
  avatar: string;
  personDescription: string;
  userStory: string;
};

const STORIES_RU: UserStoryItem[] = [
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
  {
    id: 'rustam',
    name: 'Рустам',
    role: 'Руководитель смены склада',
    avatar: '/Unknown-8.jpg',
    personDescription:
      'Начальник логистического узла и диспетчерской. Координирует утренние волны сборки, упаковку паллет и подготовку путевых листов для 60 грузовых фур.',
    userStory:
      'Рустам, как диспетчер склада, хочет автоматически кластеризовать 1 500 заказов по объему и временным окнам за 5 минут, чтобы автопарк выезжал на маршрут строго по расписанию.',
  },
  {
    id: 'alisher',
    name: 'Алишер',
    role: 'Водитель-экспедитор',
    avatar: '/Unknown-10.jpg',
    personDescription:
      'Водитель городской доставки. Развозит продукцию по розничным торговым точкам, принимает оплату наличными (COD) и подтверждает передачу груза электронным актом.',
    userStory:
      'Алишер, как водитель доставки, хочет видеть точный порядок выгрузки ящиков и фиксировать прием наличных даже при отсутствии связи в подвальных магазинах с последующей синхронизацией.',
  },
];

const STORIES_EN: UserStoryItem[] = [
  {
    id: 'alexey',
    name: 'Alexey',
    role: 'Business Owner',
    avatar: '/Unknown-5.jpg',
    personDescription:
      'Owner of a manufacturing and distribution business specializing in industrial apparel. His company supplies customized workwear to both wholesale and retail clients.',
    userStory:
      'Alexey, as a business owner, wants to view his real-time ledger balance and accounts settlement status to know the exact payout amount due to trade counterparties.',
  },
  {
    id: 'elena',
    name: 'Elena',
    role: 'Client Relationship Manager',
    avatar: '/Unknown-6.jpg',
    personDescription:
      'Client relationship manager handling commercial negotiations, contract approvals, and dispatch invoice generation across retail accounts.',
    userStory:
      'Elena, as a client manager, wants to issue verified invoices to retail stores instantly so they can settle payments promptly without slowing down the fulfillment pipeline.',
  },
  {
    id: 'rustam',
    name: 'Rustam',
    role: 'Warehouse Shift Lead',
    avatar: '/Unknown-8.jpg',
    personDescription:
      'Distribution center supervisor coordinating morning pick waves, pallet packing, and manifest issuance for a 60-truck delivery fleet.',
    userStory:
      'Rustam, as a warehouse lead, wants automated CVRP clustering to batch 1,500 morning retail orders in under 5 minutes so trucks roll out on schedule.',
  },
  {
    id: 'alisher',
    name: 'Alisher',
    role: 'Fleet Delivery Driver',
    avatar: '/Unknown-10.jpg',
    personDescription:
      'Last-mile delivery driver handling urban retail drops, cash-on-delivery (COD) till collection, and electronic proof-of-delivery signatures.',
    userStory:
      'Alisher, as a delivery driver, wants reverse-order loading guidance and offline cash-on-delivery logging so drops can be completed without cellular signal.',
  },
];

export default function UserStoriesSection() {
  const { language } = useLanguage();
  const isRu = language === 'ru';
  const stories = isRu ? STORIES_RU : STORIES_EN;
  const [activeDot, setActiveDot] = useState(2);

  return (
    <section className="relative w-full bg-[#000000] text-white py-20 lg:py-28 overflow-hidden font-sans border-t border-[#1C1C24]">
      {/* Background ambient lighting */}
      <div className="absolute top-0 right-0 w-[500px] h-[500px] bg-[radial-gradient(ellipse_at_top_right,_rgba(50,50,70,0.25)_0%,_transparent_70%)] pointer-events-none" />
      <div className="absolute bottom-10 left-10 w-[400px] h-[400px] bg-[radial-gradient(circle,_rgba(25,25,35,0.4)_0%,_transparent_70%)] pointer-events-none" />

      <div className="max-w-[1240px] mx-auto px-6 sm:px-8 lg:px-12">
        {/* Top Header Metadata Bar (Matching Reference Image) */}
        <div className="border-t border-b border-[#22222E] py-4 grid grid-cols-2 md:grid-cols-4 gap-4 text-[11px] font-mono tracking-wider text-[#8A8A9E] items-center">
          <div>
            <span className="block text-[#525266] uppercase text-[10px]">
              {isRu ? 'АВТОР:' : 'AUTHOR:'}
            </span>
            <span className="text-white font-medium uppercase">
              {isRu ? 'АНАСТАСИЯ СЕМЕНОВА' : 'ANASTASIA SEMENOVA'}
            </span>
          </div>

          <div>
            <span className="block text-[#525266] uppercase text-[10px]">
              {isRu ? 'РАЗДЕЛ:' : 'SECTION:'}
            </span>
            <span className="text-white font-medium uppercase">
              {isRu ? 'ОБОБЩЕНИЕ' : 'GENERALIZATION'}
            </span>
          </div>

          {/* Center Pagination Dots */}
          <div className="flex items-center justify-start md:justify-center space-x-1.5">
            {[0, 1, 2, 3, 4, 5, 6, 7].map((dotIdx) => (
              <button
                key={dotIdx}
                onClick={() => setActiveDot(dotIdx)}
                className={`w-1.5 h-1.5 rounded-full transition-all ${
                  dotIdx === activeDot
                    ? 'bg-white scale-125 shadow-[0_0_8px_rgba(255,255,255,0.8)]'
                    : 'bg-[#3A3A4C] hover:bg-[#6E6E82]'
                }`}
                aria-label={`Slide ${dotIdx + 1}`}
              />
            ))}
          </div>

          <div className="text-left md:text-right">
            <span className="block text-[#525266] uppercase text-[10px]">
              IOS APP
            </span>
            <span className="text-white font-medium uppercase">
              {isRu ? 'INVOICE CREATION FLOW' : 'INVOICE CREATION FLOW'}
            </span>
          </div>
        </div>

        {/* Big Bold Section Title */}
        <div className="pt-16 pb-16">
          <h2 className="text-6xl sm:text-7xl lg:text-8xl font-medium tracking-tight text-white select-none">
            User stories
          </h2>
        </div>

        {/* Stories Flow List */}
        <div className="space-y-24">
          {stories.map((story, index) => (
            <div key={story.id} className="relative">
              {/* Persona (Left Side) */}
              <div className="max-w-xl">
                <div className="flex items-center space-x-4">
                  <div className="relative w-14 h-14 sm:w-16 sm:h-16 rounded-full overflow-hidden border border-white/20 shrink-0 bg-[#161622] shadow-lg">
                    <Image
                      src={story.avatar}
                      alt={story.name}
                      fill
                      className="object-cover"
                      sizes="(max-width: 640px) 56px, 64px"
                    />
                  </div>
                  <div>
                    <h3 className="text-2xl sm:text-3xl font-medium text-white tracking-tight">
                      {story.name}
                    </h3>
                    <p className="text-xs font-mono text-[#8E8EA0] uppercase tracking-wider">
                      {story.role}
                    </p>
                  </div>
                </div>

                <p className="mt-4 text-xs sm:text-sm text-[#8E8EA8] leading-relaxed max-w-lg">
                  {story.personDescription}
                </p>

                {/* Person Pill Tag */}
                <div className="mt-4 inline-flex items-center space-x-1.5 px-3 py-1 rounded-full bg-[#121218] border border-[#272736] text-xs text-[#E0E0E8] font-medium shadow-sm">
                  <span className="text-sm">👤</span>
                  <span>Person</span>
                </div>
              </div>

              {/* Curved Connector Hairline (Connecting Person to User Story) */}
              <div className="hidden md:block absolute left-24 top-[150px] w-48 h-24 border-l border-b border-[#2A2A38] rounded-bl-3xl pointer-events-none opacity-60" />

              {/* User Story Floating White Bubble (Right Side) */}
              <div className="mt-6 md:mt-2 md:ml-auto max-w-xl lg:max-w-2xl flex flex-col items-end">
                <div className="relative w-full bg-white text-[#111116] p-6 sm:p-7 rounded-2xl shadow-[0_12px_40px_rgba(0,0,0,0.6)]">
                  <p className="text-sm sm:text-base leading-relaxed font-normal text-[#1A1A24]">
                    {story.userStory}
                  </p>
                </div>

                {/* User Story Pill Tag */}
                <div className="mt-3 inline-flex items-center space-x-1.5 px-3 py-1 rounded-full bg-[#121218] border border-[#272736] text-xs text-[#E0E0E8] font-medium shadow-sm">
                  <span className="text-sm">📝</span>
                  <span>User story</span>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
