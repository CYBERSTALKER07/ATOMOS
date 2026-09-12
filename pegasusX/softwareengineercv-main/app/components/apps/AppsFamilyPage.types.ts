import type { ReactNode } from 'react';

export type AppCard = {
  variant?: 'featured' | 'split' | 'vertical';
  tone?: 'dark' | 'light';
  tag: string;
  title: string;
  description: string;
  image: string;
  href: string;
  ctaLabel?: string;
  className?: string;
};

export type AppsFamilyConfig = {
  surface: 'web' | 'mobile' | 'desktop';
  title: string;
  subtitle: string;
  laneLabel: string;
  featured: AppCard;
  apps: AppCard[];
  features: AppCard[];
  deviceVisual: ReactNode;
};
