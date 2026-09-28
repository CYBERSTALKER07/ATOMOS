export type InfraLayer = 'all' | 'data' | 'compute' | 'edge' | 'security';

export type InfraNode = {
  id: string;
  name: string;
  subtitle: string;
  category: InfraLayer;
  tech: string[];
  status: 'active' | 'synced' | 'locked' | 'optimal';
  metrics: {
    label: string;
    value: string;
    unit?: string;
  }[];
  description: string;
  repoPath: string;
  gridX: number;
  gridY: number;
  height: number;
};

export type HudTelemetryCard = {
  id: string;
  title: string;
  status: string;
  statusColor?: string;
  metrics: {
    icon?: string;
    label: string;
    value: string;
    color?: string;
  }[];
  anchorNodeId: string;
  cardPos: { x: number; y: number };
  leaderLine: {
    startX: number;
    startY: number;
    endX: number;
    endY: number;
  };
};
