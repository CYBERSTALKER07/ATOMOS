export type AskPromptQueryToken = {
  type: 'keyword' | 'function' | 'string' | 'identifier' | 'plain';
  text: string;
};

export type AskPromptMetric = {
  id: string;
  label: string;
  title: string;
  description: string;
  verified?: boolean;
  queryLines: AskPromptQueryToken[];
  prompt: string;
  chartTitle: string;
  chartValue: string;
  chartLabels: string[];
  chartBars: number[];
};

export type AskPromptCardLayout = {
  desktop: string;
  mobileOrder?: number;
};

export type AskPromptCard = {
  id: string;
  category: string;
  question: string;
  featured?: boolean;
  layout: AskPromptCardLayout;
};

export type AskPromptSectionContent = {
  title: string;
  subtitle: string;
  metric?: AskPromptMetric;
  cards: AskPromptCard[];
};
