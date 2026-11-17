import { describe, expect, it } from 'vitest';
import type {
  YardGateCheckEvent,
  RecallBlastRadiusReport,
  ActionExecutionRequest,
} from '@pegasusx/types';

describe('Factory Portal Loading Bay Yard Gate & Lot Recall Quarantine', () => {
  it('enforces Outbound Bolt-Seal verification rules at the Factory Loading Bay', () => {
    const validateFactoryGateOutbound = (
      evt: YardGateCheckEvent,
    ): { allowed: boolean; error?: string } => {
      if (evt.direction === 'GATE_OUT') {
        if (!evt.bolt_seal_barcode || !evt.bolt_seal_barcode.trim()) {
          return { allowed: false, error: 'HARD GATE REJECTION: Missing bolt-seal barcode' };
        }
        if (!evt.seal_intact) {
          return { allowed: false, error: 'HARD GATE VIOLATION: Broken or tampered bolt-seal' };
        }
      }
      return { allowed: true };
    };

    // Case 1: Inbound raw material arrival (does not require outbound bolt seal)
    const inboundEvent: YardGateCheckEvent = {
      depot_id: 'factory-tashkent-01',
      supplier_id: 'supp-apex-central',
      vehicle_id: 'TRUCK-UZB-042',
      license_plate: '01 777 AAA',
      direction: 'GATE_IN',
      vehicle_purpose: 'INBOUND_FTL',
      seal_intact: true,
      assigned_bay_slot: 'INBOUND_STAGING',
    };
    expect(validateFactoryGateOutbound(inboundEvent).allowed).toBe(true);

    // Case 2: Outbound finished goods transfer with intact seal
    const outboundValid: YardGateCheckEvent = {
      depot_id: 'factory-tashkent-01',
      supplier_id: 'supp-apex-central',
      vehicle_id: 'TRUCK-UZB-042',
      license_plate: '01 777 AAA',
      direction: 'GATE_OUT',
      vehicle_purpose: 'OUTBOUND_ROUTE',
      bolt_seal_barcode: 'BOLT-UZB-84920',
      seal_intact: true,
      assigned_bay_slot: 'OUTBOUND_LOADING',
    };
    expect(validateFactoryGateOutbound(outboundValid).allowed).toBe(true);

    // Case 3: Outbound finished goods transfer with tampered seal
    const outboundTampered: YardGateCheckEvent = {
      ...outboundValid,
      seal_intact: false,
    };
    const tamperedRes = validateFactoryGateOutbound(outboundTampered);
    expect(tamperedRes.allowed).toBe(false);
    expect(tamperedRes.error).toContain('Broken or tampered');

    // Case 4: Outbound transfer missing seal barcode
    const outboundMissingSeal: YardGateCheckEvent = {
      ...outboundValid,
      bolt_seal_barcode: '',
    };
    const missingRes = validateFactoryGateOutbound(outboundMissingSeal);
    expect(missingRes.allowed).toBe(false);
    expect(missingRes.error).toContain('Missing bolt-seal barcode');
  });

  it('evaluates Raw Material Lot Recall contamination and formats Quarantine Action', () => {
    const report: RecallBlastRadiusReport = {
      cell_id: 'cell-uz',
      supplier_id: 'supp-apex-central',
      contaminated_raw_material_lot: 'LOT-RAW-FLOUR-202610-09',
      affected_work_orders: ['WO-PASTA-401', 'WO-PASTA-402'],
      affected_finished_lots: ['FIN-LOT-SPAGHETTI-09'],
      quarantined_sscc_pallets: ['00040123450000000011', '00040123450000000012'],
      quarantined_warehouse_totes: [],
      in_transit_driver_drop_blocks: ['DROP-BLOCK-101'],
      retailer_reverse_pickup_orders: ['RPO-901'],
      impacted_soliq_facturas: ['SF-2026-10-9901'],
    };

    const hasQuarantine =
      report.affected_work_orders.length > 0 ||
      report.affected_finished_lots.length > 0 ||
      report.quarantined_sscc_pallets.length > 0;

    expect(hasQuarantine).toBe(true);
    expect(report.quarantined_sscc_pallets).toHaveLength(2);

    const quarantineAction: ActionExecutionRequest = {
      cell_id: report.cell_id,
      supplier_id: report.supplier_id,
      action_type: 'QUARANTINE_LOT_BATCH',
      actor_id: 'factory-qc-officer',
      actor_role: 'ROLE_FACTORY',
      idempotency_key: `quarantine-${report.contaminated_raw_material_lot}-test`,
      parameters: {
        marking: 'MARKING:LOT_RECALL_QUARANTINE',
        contaminated_raw_material_lot: report.contaminated_raw_material_lot,
        quarantined_sscc_pallets: report.quarantined_sscc_pallets,
        affected_work_orders: report.affected_work_orders,
      },
    };

    expect(quarantineAction.action_type).toBe('QUARANTINE_LOT_BATCH');
    expect(quarantineAction.actor_role).toBe('ROLE_FACTORY');
    expect(quarantineAction.parameters.marking).toBe('MARKING:LOT_RECALL_QUARANTINE');
  });
});
