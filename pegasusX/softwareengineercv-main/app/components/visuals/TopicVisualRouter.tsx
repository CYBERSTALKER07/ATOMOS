'use client';

import dynamic from 'next/dynamic';

const WorkflowCircuit = dynamic(() => import('./WorkflowCircuit'), { ssr: false });
const AgentNetworkHero = dynamic(() => import('./AgentNetworkHero'), { ssr: false });
const IntegrationsHubVisual = dynamic(() => import('./IntegrationsHubVisual'), { ssr: false });
const BridgeSwapVisual = dynamic(() => import('./BridgeSwapVisual'), { ssr: false });
const TransactionFlowCard = dynamic(() => import('./TransactionFlowCard'), { ssr: false });
const MarketShareDonut = dynamic(() => import('./MarketShareDonut'), { ssr: false });
const PixelDualHero = dynamic(() => import('./PixelDualHero'), { ssr: false });

export type TopicVisualContext = {
  categoryId: string;
  slug: string;
};

export function HubCategoryVisual({ hubId }: { hubId: string }) {
  switch (hubId) {
    case 'platform':
    case 'capabilities':
      return <IntegrationsHubVisual />;
    case 'technology':
      return <WorkflowCircuit />;
    case 'ai-vision':
      return <AgentNetworkHero />;
    case 'operations':
      return <TransactionFlowCard />;
    case 'apps-deploy':
      return <IntegrationsHubVisual />;
    case 'roles':
      return <PixelDualHero />;
    default:
      return <IntegrationsHubVisual />;
  }
}

export function TopicVisualBand({ categoryId, slug }: TopicVisualContext) {
  if (slug === 'finance' || slug.includes('treasury') || slug.includes('payment')) {
    return (
      <div className="topic-visual-band topic-visual-band--split">
        <MarketShareDonut />
        <TransactionFlowCard />
      </div>
    );
  }

  if (categoryId === 'technology') {
    return (
      <div className="topic-visual-band topic-visual-band--split">
        <WorkflowCircuit />
        <BridgeSwapVisual />
      </div>
    );
  }

  if (categoryId === 'ai-vision') {
    return <div className="topic-visual-band"><AgentNetworkHero /></div>;
  }

  if (categoryId === 'platform' || categoryId === 'capabilities') {
    return <div className="topic-visual-band"><IntegrationsHubVisual /></div>;
  }

  if (categoryId === 'operations') {
    return <div className="topic-visual-band"><TransactionFlowCard /></div>;
  }

  return null;
}
