import { describe, expect, it } from 'vitest';
import {
  STATUTORY_AML_CASH_CEILING_MINOR,
  TIYIN_PER_UZS,
  type LotRecallRequest,
  type RecallBlastRadiusReport,
  type ScenarioPatchRequest,
  type ScenarioPatch,
  type ScenarioCommitResult,
} from '@pegasusx/types';

describe('Supplier Portal Foundry Scenario Sandbox & Lot Recall', () => {
  it('structures Lot Recall request and blast radius report across all supply-chain tiers', () => {
    const request: LotRecallRequest = {
      cell_id: 'cell-uz',
      supplier_id: 'sup_samarkand_dairy',
      raw_material_lot_id: 'rm_milk_batch_889',
    };

    expect(request.cell_id).toBe('cell-uz');
    expect(request.supplier_id).toBe('sup_samarkand_dairy');
    expect(request.raw_material_lot_id).toBe('rm_milk_batch_889');

    const report: RecallBlastRadiusReport = {
      cell_id: request.cell_id,
      supplier_id: request.supplier_id,
      contaminated_raw_material_lot: request.raw_material_lot_id,
      affected_work_orders: ['WO-MILK-001', 'WO-MILK-002'],
      affected_finished_lots: ['LOT-YOGURT-889-A', 'LOT-CHEESE-889-B'],
      quarantined_sscc_pallets: ['00040123451234567891', '00040123451234567892'],
      quarantined_warehouse_totes: ['TOTE-TASH-101', 'TOTE-TASH-102'],
      in_transit_driver_drop_blocks: ['DROP-ROUTE-42-STOP-3'],
      retailer_reverse_pickup_orders: ['RET-PICKUP-501', 'RET-PICKUP-502'],
      impacted_soliq_facturas: ['SF-2026-10-88912', 'SF-2026-10-88913'],
    };

    expect(report.affected_work_orders).toHaveLength(2);
    expect(report.affected_finished_lots).toHaveLength(2);
    expect(report.quarantined_sscc_pallets).toHaveLength(2);
    expect(report.quarantined_warehouse_totes).toHaveLength(2);
    expect(report.in_transit_driver_drop_blocks).toHaveLength(1);
    expect(report.retailer_reverse_pickup_orders).toHaveLength(2);
    expect(report.impacted_soliq_facturas).toHaveLength(2);
  });

  it('stages Scenario Sandbox Copy-on-Write patches with prices strictly in integer Tiyin', () => {
    // Supplier price increase scenario: 45,000 UZS -> 4,500,000 Tiyin
    const newPriceUZS = 45000;
    const newPriceMinor = newPriceUZS * TIYIN_PER_UZS;
    expect(newPriceMinor).toBe(4500000);
    expect(Number.isInteger(newPriceMinor)).toBe(true);

    const patchReq: ScenarioPatchRequest = {
      cell_id: 'cell-uz',
      supplier_id: 'sup_samarkand_dairy',
      object_type: 'WholesaleProduct',
      object_id: 'prod_kefir_1l',
      operation: 'UPSERT',
      patch_properties: {
        wholesale_price_minor: newPriceMinor,
        min_order_qty: 50,
      },
    };

    expect(patchReq.patch_properties.wholesale_price_minor).toBe(4500000);
    expect(patchReq.object_type).toBe('WholesaleProduct');

    const stagedPatch: ScenarioPatch = {
      ...patchReq,
      scenario_id: 'scen-dairy-202610-01',
      base_version: 12,
      updated_at: new Date().toISOString(),
    };
    expect(stagedPatch.base_version).toBe(12);
  });

  it('verifies optimistic watermark commit conflict detection', () => {
    const simulateWatermarkCommit = (
      scenarioId: string,
      baseVersion: number,
      currentBranchVersion: number,
    ): ScenarioCommitResult => {
      if (baseVersion !== currentBranchVersion) {
        return {
          scenario_id: scenarioId,
          committed_objects: 0,
          conflicts: [
            {
              cell_id: 'cell-uz',
              object_type: 'WholesaleProduct',
              object_id: 'prod_kefir_1l',
              base_version: baseVersion,
              live_version: currentBranchVersion,
            },
          ],
        };
      }
      return {
        scenario_id: scenarioId,
        committed_objects: 1,
      };
    };

    // Case 1: Version matches -> succeeds with 1 committed object and no conflicts
    const okRes = simulateWatermarkCommit('scen-01', 12, 12);
    expect(okRes.committed_objects).toBe(1);
    expect(okRes.conflicts).toBeUndefined();

    // Case 2: Version drift -> conflict returned
    const conflictRes = simulateWatermarkCommit('scen-01', 12, 13);
    expect(conflictRes.committed_objects).toBe(0);
    expect(conflictRes.conflicts).toHaveLength(1);
    expect(conflictRes.conflicts?.[0].live_version).toBe(13);
  });

  it('evaluates Supplier AML exposure dynamically and supports uncapped policy', () => {
    expect(STATUTORY_AML_CASH_CEILING_MINOR).toBe(2500000000);

    const evaluateSupplierTurnIn = (amountMinor: number, ceilingMinor: number = STATUTORY_AML_CASH_CEILING_MINOR) => {
      const exceedsCeiling = ceilingMinor > 0 && amountMinor > ceilingMinor;
      const ratio = ceilingMinor > 0 ? amountMinor / ceilingMinor : 0;
      return { exceedsCeiling, ratio, ceilingMinor };
    };

    const turnInUnder = evaluateSupplierTurnIn(2_000_000_000); // 20M UZS
    expect(turnInUnder.exceedsCeiling).toBe(false);
    expect(turnInUnder.ratio).toBe(0.8);

    const turnInOver = evaluateSupplierTurnIn(3_000_000_000); // 30M UZS
    expect(turnInOver.exceedsCeiling).toBe(true);
    expect(turnInOver.ratio).toBe(1.2);

    const turnInUncapped = evaluateSupplierTurnIn(100_000_000_000, 0); // 1B UZS under uncapped policy
    expect(turnInUncapped.exceedsCeiling).toBe(false);
  });
});
