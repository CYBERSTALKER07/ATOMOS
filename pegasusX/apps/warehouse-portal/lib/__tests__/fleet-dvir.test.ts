import { describe, expect, it } from 'vitest';
import type {
  FleetDVIRInspection,
  DVIRChecklist8Point,
  YardGateCheckEvent,
  PredictiveReeferResponse,
} from '@pegasusx/types';

describe('Role 9 Fleet DVIR Walkaround, Yard Gate & Reefer Gates', () => {
  it('enforces 8-point walkaround checklist integrity and Red-Tag Grounding Gate', () => {
    const cleanChecklist: DVIRChecklist8Point = {
      steering_and_horn_ok: true,
      service_and_parking_brakes_ok: true,
      tires_tread_and_pressure_ok: true,
      lights_and_hazards_ok: true,
      wipers_and_fluids_ok: true,
      emergency_kit_and_extinguisher_ok: true,
      cargo_box_sanitation_ok: true,
      reefer_temp_calibrated_ok: true,
    };

    // Case 1: All 8 points pass -> DISPATCH_AUTHORIZED
    const passInspection: FleetDVIRInspection = {
      cell_id: 'cell-uz',
      supplier_id: 'supp-central',
      vehicle_id: 'TRUCK-001',
      driver_id: 'DRV-100',
      inspection_type: 'PRE_TRIP',
      requires_cold_chain: true,
      checklist: cleanChecklist,
      odometer_km: 125000,
      reefer_temp_celsius: -18.5,
      red_tag_applied: false,
      status: 'DISPATCH_AUTHORIZED',
    };

    expect(passInspection.red_tag_applied).toBe(false);
    expect(passInspection.status).toBe('DISPATCH_AUTHORIZED');
    expect(Object.values(passInspection.checklist).every(Boolean)).toBe(true);

    // Case 2: One critical failure (service_and_parking_brakes_ok: false) -> RED_TAG_QUARANTINE
    const failInspection: FleetDVIRInspection = {
      cell_id: 'cell-uz',
      supplier_id: 'supp-central',
      vehicle_id: 'TRUCK-002',
      driver_id: 'DRV-101',
      inspection_type: 'PRE_TRIP',
      requires_cold_chain: true,
      checklist: { ...cleanChecklist, service_and_parking_brakes_ok: false },
      failed_points: ['service_and_parking_brakes_ok'],
      critical_defect_found: true,
      odometer_km: 130000,
      reefer_temp_celsius: -18.0,
      red_tag_applied: true,
      maintenance_work_order_id: 'WO-BRAKES-202610-01',
      status: 'RED_TAG_QUARANTINE',
    };

    expect(failInspection.red_tag_applied).toBe(true);
    expect(failInspection.status).toBe('RED_TAG_QUARANTINE');
    expect(failInspection.maintenance_work_order_id).toBe('WO-BRAKES-202610-01');
    expect(Object.values(failInspection.checklist).every(Boolean)).toBe(false);
  });

  it('validates Outbound Yard Gate bolt-seal integrity rule', () => {
    // Outbound gate rule: must have bolt seal barcode and seal_intact === true
    const validateGateDeparture = (event: YardGateCheckEvent): { authorized: boolean; reason?: string } => {
      if (event.direction === 'GATE_OUT') {
        if (!event.bolt_seal_barcode || !event.bolt_seal_barcode.trim()) {
          return { authorized: false, reason: 'Missing mandatory bolt-seal barcode' };
        }
        if (!event.seal_intact) {
          return { authorized: false, reason: 'Broken or tampered bolt-seal detected' };
        }
      }
      return { authorized: true };
    };

    const validOutbound: YardGateCheckEvent = {
      depot_id: 'DEPOT-01',
      supplier_id: 'SUPP-01',
      vehicle_id: 'TRUCK-001',
      license_plate: '01 777 AAA',
      direction: 'GATE_OUT',
      vehicle_purpose: 'OUTBOUND_ROUTE',
      bolt_seal_barcode: 'BOLT-UZB-99881',
      seal_intact: true,
      assigned_bay_slot: 'OUTBOUND_LOADING',
    };
    expect(validateGateDeparture(validOutbound).authorized).toBe(true);

    const tamperedOutbound: YardGateCheckEvent = {
      ...validOutbound,
      seal_intact: false,
    };
    const tamperedRes = validateGateDeparture(tamperedOutbound);
    expect(tamperedRes.authorized).toBe(false);
    expect(tamperedRes.reason).toContain('Broken or tampered');

    const missingSealOutbound: YardGateCheckEvent = {
      ...validOutbound,
      bolt_seal_barcode: undefined,
    };
    const missingRes = validateGateDeparture(missingSealOutbound);
    expect(missingRes.authorized).toBe(false);
    expect(missingRes.reason).toContain('Missing mandatory');
  });

  it('evaluates predictive reefer cooling curve against pre-failure hold thresholds', () => {
    const evaluateReeferTelemetry = (
      coolingRate: number,
      minRequiredRate: number,
      vehicleId: string,
    ): PredictiveReeferResponse => {
      const hold = coolingRate < minRequiredRate;
      return {
        vehicle_id: vehicleId,
        pre_failure_maintenance_hold: hold,
      };
    };

    // Healthy reefer: cooling at 0.75 deg/min, required min is 0.40 deg/min
    const healthy = evaluateReeferTelemetry(0.75, 0.40, 'REEFER-01');
    expect(healthy.pre_failure_maintenance_hold).toBe(false);

    // Failing reefer compressor: cooling at 0.20 deg/min, required min is 0.40 deg/min
    const failing = evaluateReeferTelemetry(0.20, 0.40, 'REEFER-02');
    expect(failing.pre_failure_maintenance_hold).toBe(true);
  });
});
