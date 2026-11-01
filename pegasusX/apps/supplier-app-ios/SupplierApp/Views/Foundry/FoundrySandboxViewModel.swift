//
//  FoundrySandboxViewModel.swift
//  SupplierAppIOS
//

import Foundation
import SwiftUI

public struct ScenarioPatchSummaryItem: Identifiable, Sendable {
    public var id: String { "\(objectType):\(objectId):\(key)" }
    public let objectType: String
    public let objectId: String
    public let key: String
    public let value: String
}

@Observable
final class FoundrySandboxViewModel {

    var supplierId: String = "supplier-tashkent-1"
    var cellId: String = "cell-uz"
    var scenarioId: String = "scenario-surge-plan-01"
    var objectType: String = "SKU"
    var objectId: String = "sku-uz-cola-1l"
    var patchKey: String = "price_minor"
    var patchValue: String = "1250000"

    var appliedPatches: [ScenarioPatchSummaryItem] = []
    var isCommitting: Bool = false
    var commitResult: ScenarioCommitResult? = nil
    var hasConflict: Bool = false
    var conflicts: [ScenarioWatermarkConflict] = []

    var lotBarcode: String = "LOT-SUGAR-2026-X8"
    var isAnalyzingRecall: Bool = false
    var recallReport: RecallBlastRadiusReport? = nil
    var isQuarantining: Bool = false
    var quarantineSuccess: Bool = false

    var statusMessage: String? = nil
    var errorMessage: String? = nil

    @MainActor
    func applyCowPatch() async {
        let key = patchKey.trimmingCharacters(in: .whitespaces)
        let val = patchValue.trimmingCharacters(in: .whitespaces)
        guard !key.isEmpty, !val.isEmpty else {
            errorMessage = "Patch key and value must not be empty"
            return
        }

        errorMessage = nil
        statusMessage = nil

        let req = ScenarioPatchRequest(
            cellId: cellId,
            supplierId: supplierId,
            objectType: objectType,
            objectId: objectId,
            operation: "SET",
            patchProperties: [key: val]
        )

        do {
            _ = try await SupplierService.applyScenarioPatch(scenarioId: scenarioId, req: req)
            let item = ScenarioPatchSummaryItem(
                objectType: objectType,
                objectId: objectId,
                key: key,
                value: val
            )
            appliedPatches.append(item)
            statusMessage = "CoW Patch staged: \(objectType):\(objectId) [\(key) = \(val)]"
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    @MainActor
    func commitScenario() async {
        isCommitting = true
        errorMessage = nil
        statusMessage = nil

        do {
            let res = try await SupplierService.commitScenario(scenarioId: scenarioId)
            isCommitting = false
            commitResult = res
            hasConflict = !res.conflicts.isEmpty
            conflicts = res.conflicts
            statusMessage = hasConflict ? "Commit Conflict (409): Live objects mutated concurrently." : "Scenario committed (\(res.committedObjects) objects updated)."
        } catch let apiError as APIError {
            isCommitting = false
            switch apiError {
            case .httpError(let code) where code == 409:
                hasConflict = true
                errorMessage = "Scenario Watermark Conflict (409): Baseline modified by another user."
            default:
                errorMessage = "Commit failed: \(apiError)"
            }
        } catch {
            isCommitting = false
            errorMessage = error.localizedDescription
        }
    }

    @MainActor
    func analyzeLotRecall() async {
        let lot = lotBarcode.trimmingCharacters(in: .whitespaces)
        guard !lot.isEmpty else {
            errorMessage = "Raw lot barcode is required"
            return
        }

        isAnalyzingRecall = true
        errorMessage = nil
        statusMessage = nil

        let req = LotRecallRequest(
            cellId: cellId,
            supplierId: supplierId,
            rawMaterialLotId: lot
        )

        do {
            let res = try await SupplierService.triggerLotRecall(req: req)
            isAnalyzingRecall = false
            recallReport = res
            quarantineSuccess = false
            statusMessage = "Blast Radius Evaluated: \(res.quarantinedSsccPallets.count) pallets & \(res.inTransitDriverDropBlocks.count) transit drops affected."
        } catch {
            isAnalyzingRecall = false
            errorMessage = error.localizedDescription
        }
    }

    @MainActor
    func executeQuarantineBlastRadius() async {
        guard let report = recallReport else {
            errorMessage = "No recall report available to quarantine"
            return
        }

        isQuarantining = true
        errorMessage = nil

        let req = ActionExecutionRequest(
            actionName: "EXECUTE_LOT_QUARANTINE_BLAST_RADIUS",
            actionType: "MUTATION",
            actorId: supplierId,
            actorRole: "SUPPLIER_QUALITY_DIRECTOR",
            targetObjectType: "RAW_MATERIAL_LOT",
            targetObjectId: report.contaminatedRawMaterialLot,
            idempotencyKey: UUID().uuidString,
            cellId: cellId,
            supplierId: supplierId
        )

        do {
            _ = try await SupplierService.executeAction(req: req)
            isQuarantining = false
            quarantineSuccess = true
            statusMessage = "EMERGENCY QUARANTINE ENFORCED: All downstream pallets, totes, and driver deliveries locked."
        } catch {
            isQuarantining = false
            errorMessage = error.localizedDescription
        }
    }
}
