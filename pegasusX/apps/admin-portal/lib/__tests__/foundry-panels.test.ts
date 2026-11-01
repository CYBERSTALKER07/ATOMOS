import { describe, expect, it } from 'vitest';
import {
  TIYIN_PER_UZS,
  type ActionExecutionRequest,
  type PreconditionResult,
  type DepotVaultLedgerEntry,
  type FleetDVIRInspection,
  type DVIRChecklist8Point,
} from '@pegasusx/types';

describe('Admin Portal Foundry Triad, Vault Monitor & Fleet Yard Monitors', () => {
  it('evaluates Kinetic Action pre-execution guards and rubrics', () => {
    const preconditions: PreconditionResult[] = [
      { rule_name: 'ROLE_RBAC_AUTHORIZED', passed: true },
      { rule_name: 'FISCAL_INVENTORY_COMMITTED', passed: true },
      { rule_name: 'SOLIQ_CREDIT_LIMIT_UNEXHAUSTED', passed: false, reason: 'Credit ceiling exceeded' },
    ];

    const evaluateGuard = (rules: PreconditionResult[]): { canExecute: boolean; blockedRule?: string } => {
      const failed = rules.find((r) => !r.passed);
      if (failed) {
        return { canExecute: false, blockedRule: failed.rule_name };
      }
      return { canExecute: true };
    };

    const guardResult = evaluateGuard(preconditions);
    expect(guardResult.canExecute).toBe(false);
    expect(guardResult.blockedRule).toBe('SOLIQ_CREDIT_LIMIT_UNEXHAUSTED');

    const passedPreconditions: PreconditionResult[] = [
      { rule_name: 'ROLE_RBAC_AUTHORIZED', passed: true },
      { rule_name: 'FISCAL_INVENTORY_COMMITTED', passed: true },
      { rule_name: 'SOLIQ_CREDIT_LIMIT_UNEXHAUSTED', passed: true },
    ];
    expect(evaluateGuard(passedPreconditions).canExecute).toBe(true);

    const actionReq: ActionExecutionRequest = {
      cell_id: 'cell-uz',
      supplier_id: 'sup_admin_hq',
      action_type: 'WAVE_RELEASE_ALLOCATION',
      actor_id: 'admin_officer_01',
      actor_role: 'ROLE_PLATFORM_ADMIN',
      idempotency_key: 'wave-rel-202610-001',
      parameters: {
        wave_id: 'WAVE-992',
        tier: 'EXPRESS',
      },
    };

    expect(actionReq.actor_role).toBe('ROLE_PLATFORM_ADMIN');
    expect(actionReq.idempotency_key).toBe('wave-rel-202610-001');
  });

  it('aggregates Central Depot Vault ledger entries strictly in integer Tiyin', () => {
    const vaultEntries: DepotVaultLedgerEntry[] = [
      {
        vault_entry_id: 'v-001',
        depot_id: 'depot-tashkent-central',
        supplier_id: 'supp-central',
        entry_type: 'DRIVER_DROP_INFLOW',
        amount_minor: 120000000, // 1,200,000 UZS
        running_balance_minor: 120000000,
        created_at: new Date().toISOString(),
      },
      {
        vault_entry_id: 'v-002',
        depot_id: 'depot-samarkand',
        supplier_id: 'supp-central',
        entry_type: 'DRIVER_DROP_INFLOW',
        amount_minor: 85000000, // 850,000 UZS
        running_balance_minor: 85000000,
        created_at: new Date().toISOString(),
      },
      {
        vault_entry_id: 'v-003',
        depot_id: 'depot-tashkent-central',
        supplier_id: 'supp-central',
        entry_type: 'CIT_ARMORED_OUTFLOW',
        amount_minor: -100000000, // -1,000,000 UZS
        running_balance_minor: 20000000,
        created_at: new Date().toISOString(),
      },
    ];

    const totalVaultNetMinor = vaultEntries.reduce((sum, entry) => sum + entry.amount_minor, 0);
    // 120M + 85M - 100M = 105M Tiyin = 1,050,000 UZS
    expect(totalVaultNetMinor).toBe(105000000);
    expect(Number.isInteger(totalVaultNetMinor)).toBe(true);

    const formattedUZS = (totalVaultNetMinor / TIYIN_PER_UZS).toLocaleString('en-US', {
      minimumFractionDigits: 2,
    });
    expect(formattedUZS).toBe('1,050,000.00');
  });

  it('aggregates Fleet Yard DVIR inspections and isolates Red-Tag Grounded assets', () => {
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

    const fleetInspections: FleetDVIRInspection[] = [
      {
        cell_id: 'cell-uz',
        supplier_id: 'supp-central',
        vehicle_id: 'TRK-01',
        driver_id: 'D-01',
        inspection_type: 'PRE_TRIP',
        requires_cold_chain: true,
        checklist: cleanChecklist,
        odometer_km: 50000,
        reefer_temp_celsius: -19.0,
        red_tag_applied: false,
        status: 'DISPATCH_AUTHORIZED',
      },
      {
        cell_id: 'cell-uz',
        supplier_id: 'supp-central',
        vehicle_id: 'TRK-02',
        driver_id: 'D-02',
        inspection_type: 'PRE_TRIP',
        requires_cold_chain: true,
        checklist: { ...cleanChecklist, service_and_parking_brakes_ok: false },
        failed_points: ['service_and_parking_brakes_ok'],
        critical_defect_found: true,
        odometer_km: 75000,
        reefer_temp_celsius: -10.0,
        red_tag_applied: true,
        maintenance_work_order_id: 'WO-BRAKES-02',
        status: 'RED_TAG_QUARANTINE',
      },
      {
        cell_id: 'cell-uz',
        supplier_id: 'supp-central',
        vehicle_id: 'TRK-03',
        driver_id: 'D-03',
        inspection_type: 'POST_TRIP',
        requires_cold_chain: true,
        checklist: cleanChecklist,
        odometer_km: 99000,
        reefer_temp_celsius: -18.2,
        red_tag_applied: false,
        preventative_service_due: true,
        status: 'DISPATCH_AUTHORIZED',
      },
    ];

    const redTaggedVehicles = fleetInspections.filter((v) => v.red_tag_applied);
    const authorizedVehicles = fleetInspections.filter((v) => v.status === 'DISPATCH_AUTHORIZED');
    const serviceDueVehicles = fleetInspections.filter((v) => v.preventative_service_due);

    expect(redTaggedVehicles).toHaveLength(1);
    expect(redTaggedVehicles[0].vehicle_id).toBe('TRK-02');
    expect(redTaggedVehicles[0].maintenance_work_order_id).toBe('WO-BRAKES-02');

    expect(authorizedVehicles).toHaveLength(2);
    expect(serviceDueVehicles).toHaveLength(1);
    expect(serviceDueVehicles[0].vehicle_id).toBe('TRK-03');
  });
});
