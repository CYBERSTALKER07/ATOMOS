'use client';

import type { ReactNode } from 'react';
import dynamic from 'next/dynamic';
import FleekSection from './FleekSection';
import LogisticsAnalyticsDashboard from '@/app/components/logistics/LogisticsAnalyticsDashboard';
import ImpactMetricCard from './cards/ImpactMetricCard';
import BlobStatPanel from './cards/BlobStatPanel';
import AxiomStatsBar from './cards/AxiomStatsBar';

const IntegrationsHubVisual = dynamic(() => import('@/app/components/visuals/IntegrationsHubVisual'), { ssr: false });
const WorkflowCircuit = dynamic(() => import('@/app/components/visuals/WorkflowCircuit'), { ssr: false });
const BridgeSwapVisual = dynamic(() => import('@/app/components/visuals/BridgeSwapVisual'), { ssr: false });
const AgentNetworkHero = dynamic(() => import('@/app/components/visuals/AgentNetworkHero'), { ssr: false });
const TransactionFlowCard = dynamic(() => import('@/app/components/visuals/TransactionFlowCard'), { ssr: false });
const MarketShareDonut = dynamic(() => import('@/app/components/visuals/MarketShareDonut'), { ssr: false });
const PixelDualHero = dynamic(() => import('@/app/components/visuals/PixelDualHero'), { ssr: false });

type FleekDataSectionProps = {
  hubId?: string;
  categoryId?: string;
  slug?: string;
  extra?: ReactNode;
};

function hubCards(hubId?: string, categoryId?: string, slug?: string): ReactNode[] {
  const cards: ReactNode[] = [];

  if (slug === 'finance' || slug?.includes('treasury') || slug?.includes('payment')) {
    cards.push(<MarketShareDonut key="donut" />, <TransactionFlowCard key="flow" />);
    return cards;
  }

  const id = hubId ?? categoryId;

  switch (id) {
    case 'platform':
      cards.push(<BlobStatPanel key="blob" />, <AxiomStatsBar key="axiom" />);
      break;
    case 'technology':
      cards.push(<WorkflowCircuit key="circuit" />, <BridgeSwapVisual key="bridge" />, <AxiomStatsBar key="axiom" />);
      break;
    case 'ai-vision':
      cards.push(<AgentNetworkHero key="agent" />, <BlobStatPanel key="blob" />);
      break;
    case 'operations':
      cards.push(<TransactionFlowCard key="flow" />, <ImpactMetricCard key="impact" />);
      break;
    case 'capabilities':
      cards.push(<IntegrationsHubVisual key="hub" />, <ImpactMetricCard key="impact" />);
      break;
    case 'apps-deploy':
      cards.push(<BlobStatPanel key="blob" />, <IntegrationsHubVisual key="hub" />);
      break;
    case 'roles':
      cards.push(<PixelDualHero key="pixel" />, <AxiomStatsBar key="axiom" />);
      break;
    case 'technology' as string:
      break;
    default:
      if (categoryId === 'technology') {
        cards.push(<WorkflowCircuit key="circuit" />, <BridgeSwapVisual key="bridge" />);
      } else if (categoryId === 'ai-vision') {
        cards.push(<AgentNetworkHero key="agent" />);
      } else if (categoryId === 'operations') {
        cards.push(<TransactionFlowCard key="flow" />);
      } else if (categoryId === 'platform' || categoryId === 'capabilities') {
        cards.push(<IntegrationsHubVisual key="hub" />);
      }
  }

  return cards;
}

export default function FleekDataSection({ hubId, categoryId, slug, extra }: FleekDataSectionProps) {
  const cards = hubCards(hubId, categoryId, slug);

  return (
    <FleekSection id="fleek-section-04" number="04" title="NETWORK DATA & VISUALS">
      <LogisticsAnalyticsDashboard />
      {cards.length > 0 ? (
        <div className={`fleek-data-cards ${cards.length > 1 ? 'fleek-data-cards--grid' : ''}`}>
          {cards}
        </div>
      ) : null}
      {extra}
    </FleekSection>
  );
}
