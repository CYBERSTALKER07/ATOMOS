package foundry

import (
	"context"
	"fmt"
	"strings"
)

// Canonical Kinetic ActionType names corresponding to the 10-Step Master Operational Cascade.
const (
	ActionReleasePickWave     = "ACTION_RELEASE_PICK_WAVE"
	ActionSealVehicleManifest = "ACTION_SEAL_VEHICLE_MANIFEST"
	ActionStartDriverRoute    = "ACTION_START_DRIVER_ROUTE"
	ActionCompleteDoorstepEPOD = "ACTION_COMPLETE_DOORSTEP_EPOD"
	ActionBatchSupplierPayout = "ACTION_BATCH_SUPPLIER_PAYOUT"
)

func int64Param(params map[string]any, key string) (int64, bool) {
	if params == nil {
		return 0, false
	}
	v, ok := params[key]
	if !ok || v == nil {
		return 0, false
	}
	switch n := v.(type) {
	case int64:
		return n, true
	case int:
		return int64(n), true
	case float64:
		return int64(n), true
	default:
		return 0, false
	}
}

func stringParam(params map[string]any, key string) string {
	if params == nil {
		return ""
	}
	s, _ := params[key].(string)
	return strings.TrimSpace(s)
}

func boolParam(params map[string]any, key string) (bool, bool) {
	if params == nil {
		return false, false
	}
	b, ok := params[key].(bool)
	return b, ok
}

// registerDefaultPreExecutionGuards wires the mandatory pre-execution rubrics across
// the 10-Step Operational Cascade so every execution is validated before state mutation.
func (e *Engine) registerDefaultPreExecutionGuards() {
	// Step 5: Pick Wave Release Pre-Execution Guards (Guard #3 Fiscal Readiness + Guard #9 FX Corridor).
	e.RegisterPrecondition(ActionReleasePickWave, func(ctx context.Context, req ActionExecutionRequest) PreconditionResult {
		if certHours, ok := int64Param(req.Parameters, "pkcs12_cert_hours_remaining"); ok && certHours < 72 {
			return PreconditionResult{
				RuleName: "Guard3_PreWavePKCS12CertWindow",
				Passed:   false,
				Reason:   fmt.Sprintf("PKCS#12 signing certificate expires in %dh (< 72h minimum)", certHours),
			}
		}
		if ikpuValid, ok := boolParam(req.Parameters, "all_skus_ikpu_valid"); ok && !ikpuValid {
			return PreconditionResult{
				RuleName: "Guard3_PreWaveIKPUValidation",
				Passed:   false,
				Reason:   "one or more SKUs lack a valid IKPU/MXIK tax classification code",
			}
		}
		fxPolicy := strings.ToUpper(stringParam(req.Parameters, "fx_policy"))
		if fxPolicy == "FLOAT_WITH_CORRIDOR_CAP" {
			deltaBps, hasDelta := int64Param(req.Parameters, "fx_delta_bps")
			capBps, hasCap := int64Param(req.Parameters, "fx_corridor_cap_bps")
			if hasDelta && hasCap {
				absDelta := deltaBps
				if absDelta < 0 {
					absDelta = -absDelta
				}
				if absDelta > capBps {
					return PreconditionResult{
						RuleName: "Guard9_FXVolatilityCorridor",
						Passed:   false,
						Reason:   fmt.Sprintf("FX delta %d bps exceeds corridor cap %d bps; retailer reconfirmation required", deltaBps, capBps),
					}
				}
			}
		}
		return PreconditionResult{RuleName: "Step5_WaveReleasePreFlight", Passed: true}
	})

	// Step 6: Payloader Vehicle Manifest Sealing Pre-Execution Guards (Guard #6 3D Crushability/RTI + Guard #12 DVIR).
	e.RegisterPrecondition(ActionSealVehicleManifest, func(ctx context.Context, req ActionExecutionRequest) PreconditionResult {
		vehicleID := stringParam(req.Parameters, "vehicle_id")
		if vehicleID != "" {
			if err := e.AssertVehicleDispatchAuthorized(ctx, vehicleID); err != nil {
				return PreconditionResult{
					RuleName: "Guard12_VehicleDVIRAuthorization",
					Passed:   false,
					Reason:   err.Error(),
				}
			}
		}
		if crushViolation, ok := boolParam(req.Parameters, "crush_violation"); ok && crushViolation {
			return PreconditionResult{
				RuleName: "Guard6_3DVolumetricCrushability",
				Passed:   false,
				Reason:   "heavy SKU layer stacked above fragile carton exceeding MaxStackCrushKg",
			}
		}
		if seal := stringParam(req.Parameters, "bolt_seal_barcode"); seal == "" {
			return PreconditionResult{
				RuleName: "Step6_CryptographicBoltSealRequired",
				Passed:   false,
				Reason:   "bolt_seal_barcode is required before sealing vehicle manifest",
			}
		}
		return PreconditionResult{RuleName: "Step6_PayloaderManifestSealPreFlight", Passed: true}
	})

	// Step 7: Driver Route Start Pre-Execution Guards (Guard #12 DVIR Red-Tag Grounding Gate).
	e.RegisterPrecondition(ActionStartDriverRoute, func(ctx context.Context, req ActionExecutionRequest) PreconditionResult {
		vehicleID := stringParam(req.Parameters, "vehicle_id")
		if vehicleID == "" {
			return PreconditionResult{
				RuleName: "Guard12_VehicleDVIRAuthorization",
				Passed:   false,
				Reason:   "vehicle_id is required to start a driver route",
			}
		}
		if err := e.AssertVehicleDispatchAuthorized(ctx, vehicleID); err != nil {
			return PreconditionResult{
				RuleName: "Guard12_VehicleDVIRAuthorization",
				Passed:   false,
				Reason:   err.Error(),
			}
		}
		return PreconditionResult{RuleName: "Step7_DriverRouteStartPreFlight", Passed: true}
	})

	// Step 8: Doorstep ePOD Completion Pre-Execution Guards (Guard #2 AML Ceiling & Anti-Smurfing).
	e.RegisterPrecondition(ActionCompleteDoorstepEPOD, func(ctx context.Context, req ActionExecutionRequest) PreconditionResult {
		cashMinor, hasCash := int64Param(req.Parameters, "cash_collected_minor")
		if hasCash && cashMinor > 0 && e.IsAMLCeilingEnforced() {
			ceiling := e.AMLCashCeilingMinor()
			if ceiling > 0 && cashMinor > ceiling {
				return PreconditionResult{
					RuleName: "Guard2_StatutoryAMLCeiling",
					Passed:   false,
					Reason:   fmt.Sprintf("cash leg %d Tiyin exceeds statutory AML ceiling (%d Tiyin)", cashMinor, ceiling),
				}
			}
			tin := stringParam(req.Parameters, "legal_entity_tin")
			calDate := stringParam(req.Parameters, "calendar_date")
			if tin != "" && calDate != "" {
				e.mu.RLock()
				existing := e.amlDailyExposure[amlExposureKey(tin, req.SupplierID, calDate)]
				e.mu.RUnlock()
				if ceiling > 0 && existing+cashMinor > ceiling {
					return PreconditionResult{
						RuleName: "Guard2_TINAntiSmurfingAggregation",
						Passed:   false,
						Reason:   fmt.Sprintf("accumulated daily cash (%d + %d Tiyin) for TIN %s exceeds statutory AML ceiling (%d Tiyin)", existing, cashMinor, tin, ceiling),
					}
				}
			}
		}
		return PreconditionResult{RuleName: "Step8_DoorstepEPODPreFlight", Passed: true}
	})

	// Step 10: Supplier Payout Batch Pre-Execution Guards (Guard #3 Soliq Registration Lock).
	e.RegisterPrecondition(ActionBatchSupplierPayout, func(ctx context.Context, req ActionExecutionRequest) PreconditionResult {
		fiscalState := strings.ToUpper(stringParam(req.Parameters, "fiscal_state"))
		if fiscalState != "REGISTERED_SOLIQ" {
			return PreconditionResult{
				RuleName: "Guard3_SoliqFiscalRegistrationLock",
				Passed:   false,
				Reason:   fmt.Sprintf("settlement slice cannot be batched while fiscal_state=%q (must be REGISTERED_SOLIQ)", fiscalState),
			}
		}
		return PreconditionResult{RuleName: "Step10_PayoutBatchPreFlight", Passed: true}
	})
}
