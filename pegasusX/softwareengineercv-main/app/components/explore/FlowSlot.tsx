'use client';

import type { ComponentType } from 'react';
import dynamic from 'next/dynamic';
import type { FlowConfig, FlowVariant } from '@/app/data/topicTypes';

const flows: Record<FlowVariant, ComponentType<{ config?: FlowConfig }>> = {
  orderLifecycle: dynamic(() => import('@/app/components/flows/OrderLifecycleFlow')),
  controlPlane: dynamic(() => import('@/app/components/flows/ControlPlaneFlow')),
  mutatingHandler: dynamic(() => import('@/app/components/flows/MutatingHandlerFlow')),
  realtimePipeline: dynamic(() => import('@/app/components/flows/RealtimePipelineFlow')),
  dispatchBoard: dynamic(() => import('@/app/components/flows/DispatchBoardFlow')),
  fleetMap: dynamic(() => import('@/app/components/flows/FleetMapFlow')),
  paymentFlow: dynamic(() => import('@/app/components/flows/PaymentFlow')),
  roleJourney: dynamic(() => import('@/app/components/flows/RoleJourneyFlow')),
  topologyMap: dynamic(() => import('@/app/components/flows/TopologyMapFlow')),
  techStack: dynamic(() => import('@/app/components/flows/TechStackFlow')),
  aiAssist: dynamic(() => import('@/app/components/flows/AiAssistFlow')),
  exceptionPlaybook: dynamic(() => import('@/app/components/flows/ExceptionPlaybookFlow')),
  appsMatrix: dynamic(() => import('@/app/components/flows/AppsMatrixFlow')),
};

type FlowSlotProps = {
  variant: FlowVariant;
  config?: FlowConfig;
};

export default function FlowSlot({ variant, config }: FlowSlotProps) {
  const Flow = flows[variant];
  return <Flow config={config} />;
}
