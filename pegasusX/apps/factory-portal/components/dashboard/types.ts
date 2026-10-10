import type { FactoryDashboardException } from '@pegasusx/types';

export interface FactoryStats {
  source: string;
  plane: string;
  pending_transfers: number;
  loading_transfers: number;
  active_manifests: number;
  dispatched_today: number;
  vehicles_total: number;
  vehicles_available: number;
  staff_on_shift: number;
  critical_insights: number;
  transfers_by_state: Record<string, number>;
  manifests_by_state: Record<string, number>;
  vehicles_by_state: Record<string, number>;
  driver_duty: Record<string, number>;
  sla_by_status: Record<string, number>;
  qc_by_result: Record<string, number>;
  qc_available: boolean;
  bay_loading_transfers: number;
  bay_loading_manifests: number;
  exceptions: FactoryDashboardException[];
}
