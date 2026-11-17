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
@MainActor
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
            errorMessage = L10n.format("mobile_supplier.chrome.patch_key_value_required")
            return
        }
        guard let pinnedSupplier = pinnedSupplierId() else { return }
        let scenario = scenarioId.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !scenario.isEmpty else {
            errorMessage = L10n.format("mobile_supplier.chrome.scenario_id_required")
            return
        }

        errorMessage = nil
        statusMessage = nil

        let req = ScenarioPatchRequest(
            cellId: cellId,
            supplierId: pinnedSupplier,
            objectType: objectType,
            objectId: objectId,
            operation: "SET",
            patchProperties: [key: val]
        )

        do {
            _ = try await SupplierService.applyScenarioPatch(
                scenarioId: scenario,
                req: req,
                idempotencyKey: FoundryIdempotency.scenarioPatch(
                    scenarioId: scenario,
                    supplierId: pinnedSupplier,
                    objectType: objectType,
                    objectId: objectId,
                    key: key,
                    value: val
                )
            )
            let item = ScenarioPatchSummaryItem(
                objectType: objectType,
                objectId: objectId,
                key: key,
                value: val
            )
            appliedPatches.append(item)
            statusMessage = L10n.format("mobile_supplier.chrome.cow_patch_staged", objectType, objectId, key, val)
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    @MainActor
    func commitScenario() async {
        guard pinnedSupplierId() != nil else { return }
        let scenario = scenarioId.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !scenario.isEmpty else {
            errorMessage = L10n.format("mobile_supplier.chrome.scenario_id_required")
            return
        }
        isCommitting = true
        defer { isCommitting = false }
        errorMessage = nil
        statusMessage = nil

        do {
            let res = try await SupplierService.commitScenario(
                scenarioId: scenario,
                idempotencyKey: FoundryIdempotency.scenarioCommit(scenarioId: scenario)
            )
            commitResult = res
            hasConflict = !res.conflicts.isEmpty
            conflicts = res.conflicts
            statusMessage = hasConflict ? L10n.format("mobile_supplier.chrome.commit_conflict") : L10n.format("mobile_supplier.chrome.scenario_committed", "\(res.committedObjects)")
        } catch let apiError as APIError {
            switch apiError {
            case .httpError(let code) where code == 409:
                hasConflict = true
                errorMessage = L10n.format("mobile_supplier.chrome.scenario_watermark_conflict")
            default:
                errorMessage = L10n.format("mobile_supplier.chrome.commit_failed", apiError.localizedDescription)
            }
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    @MainActor
    func analyzeLotRecall() async {
        let lot = lotBarcode.trimmingCharacters(in: .whitespaces)
        guard !lot.isEmpty else {
            errorMessage = L10n.format("mobile_supplier.chrome.raw_lot_required")
            return
        }
        guard let pinnedSupplier = pinnedSupplierId() else { return }

        isAnalyzingRecall = true
        defer { isAnalyzingRecall = false }
        errorMessage = nil
        statusMessage = nil

        let req = LotRecallRequest(
            cellId: cellId,
            supplierId: pinnedSupplier,
            rawMaterialLotId: lot
        )

        do {
            let res = try await SupplierService.triggerLotRecall(
                req: req,
                idempotencyKey: FoundryIdempotency.lotRecall(supplierId: pinnedSupplier, lot: lot)
            )
            recallReport = res
            quarantineSuccess = false
            statusMessage = L10n.format("mobile_supplier.chrome.blast_radius_evaluated", "\(res.quarantinedSsccPallets.count)", "\(res.inTransitDriverDropBlocks.count)")
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    @MainActor
    func executeQuarantineBlastRadius() async {
        guard let report = recallReport else {
            errorMessage = L10n.format("mobile_supplier.chrome.no_recall_report")
            return
        }
        guard let pinnedSupplier = pinnedSupplierId() else { return }

        isQuarantining = true
        defer { isQuarantining = false }
        errorMessage = nil

        let idempotencyKey = "foundry-lot-quarantine:\(pinnedSupplier):\(report.contaminatedRawMaterialLot)"
        let req = ActionExecutionRequest(
            actionName: "EXECUTE_LOT_QUARANTINE_BLAST_RADIUS",
            actionType: "MUTATION",
            actorId: pinnedSupplier,
            actorRole: "SUPPLIER_QUALITY_DIRECTOR",
            targetObjectType: "RAW_MATERIAL_LOT",
            targetObjectId: report.contaminatedRawMaterialLot,
            idempotencyKey: idempotencyKey,
            cellId: cellId,
            supplierId: pinnedSupplier
        )

        do {
            _ = try await SupplierService.executeAction(req: req, idempotencyKey: idempotencyKey)
            quarantineSuccess = true
            statusMessage = L10n.format("mobile_supplier.chrome.emergency_quarantine")
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func pinnedSupplierId() -> String? {
        let session = TokenStore.shared.supplierId?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !session.isEmpty else {
            errorMessage = L10n.format("mobile_supplier.chrome.sign_in_foundry")
            return nil
        }
        supplierId = session
        return session
    }
}

enum FoundryIdempotency {
    static func scenarioPatch(
        scenarioId: String,
        supplierId: String,
        objectType: String,
        objectId: String,
        key: String,
        value: String
    ) -> String {
        "foundry-scenario-patch:\(scenarioId):\(supplierId):\(objectType):\(objectId):\(key):\(value)"
    }

    static func scenarioCommit(scenarioId: String) -> String {
        "foundry-scenario-commit:\(scenarioId)"
    }

    static func lotRecall(supplierId: String, lot: String) -> String {
        "foundry-lot-recall:\(supplierId):\(lot)"
    }
}
