import type { AskPromptSectionContent } from '@/app/components/ask-prompt/types';

export const PEGASUS_ASK_PROMPTS: AskPromptSectionContent = {
  title: 'Ask anything about your network',
  subtitle:
    'Give the AI approved logistics definitions to reuse across chat, charts, and dashboards.',
  metric: {
    id: 'otd',
    label: 'OTD',
    title: 'On-time delivery',
    description: 'Percentage of deliveries completed within the promised SLA window.',
    verified: true,
    queryLines: [
      { type: 'keyword', text: 'select' },
      { type: 'plain', text: '\n  ' },
      { type: 'function', text: 'date_trunc' },
      { type: 'plain', text: "('week', " },
      { type: 'identifier', text: 'delivered_at' },
      { type: 'plain', text: ')\n  ' },
      { type: 'keyword', text: 'as' },
      { type: 'plain', text: ' ' },
      { type: 'identifier', text: 'week' },
      { type: 'plain', text: ',\n  ' },
      { type: 'function', text: 'count' },
      { type: 'plain', text: '(*) ' },
      { type: 'keyword', text: 'filter' },
      { type: 'plain', text: ' (' },
      { type: 'keyword', text: 'where' },
      { type: 'plain', text: ' ' },
      { type: 'identifier', text: 'on_time' },
      { type: 'plain', text: ') * 100.0\n  / ' },
      { type: 'function', text: 'count' },
      { type: 'plain', text: '(*) ' },
      { type: 'keyword', text: 'as' },
      { type: 'plain', text: ' ' },
      { type: 'identifier', text: 'otd_pct' },
      { type: 'plain', text: '\n' },
      { type: 'keyword', text: 'from' },
      { type: 'plain', text: ' ' },
      { type: 'identifier', text: 'deliveries' },
      { type: 'plain', text: '\n' },
      { type: 'keyword', text: 'where' },
      { type: 'plain', text: ' ' },
      { type: 'identifier', text: 'status' },
      { type: 'plain', text: " = " },
      { type: 'string', text: "'completed'" },
      { type: 'plain', text: '\n' },
      { type: 'keyword', text: 'group by' },
      { type: 'plain', text: ' ' },
      { type: 'identifier', text: 'week' },
    ],
    prompt: 'How is our on-time delivery trending?',
    chartTitle: 'OTD by month',
    chartValue: '96.2%',
    chartLabels: ['Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug'],
    chartBars: [72, 81, 86, 84, 91, 96],
  },
  cards: [],
};

export const PEGASUS_ASK_PROMPTS_RU: AskPromptSectionContent = {
  title: 'Спросите что угодно о вашей сети',
  subtitle:
    'Дайте ИИ утверждённые логистические определения для повторного использования в чате, графиках и дашбордах.',
  metric: {
    ...PEGASUS_ASK_PROMPTS.metric!,
    title: 'Доставка вовремя',
    description: 'Доля доставок, завершённых в обещанном окне SLA.',
    prompt: 'Как меняется наша доставка вовремя?',
    chartTitle: 'OTD по месяцам',
    chartLabels: ['Мар', 'Апр', 'Май', 'Июн', 'Июл', 'Авг'],
  },
  cards: [],
};

export function getAskPromptContent(lang: 'en' | 'ru' = 'en'): AskPromptSectionContent {
  return lang === 'ru' ? PEGASUS_ASK_PROMPTS_RU : PEGASUS_ASK_PROMPTS;
}
