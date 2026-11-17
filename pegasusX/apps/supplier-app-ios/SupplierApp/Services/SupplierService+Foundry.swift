//
//  SupplierService+Foundry.swift
//  SupplierAppIOS
//

import Foundation

extension SupplierService {

    // MARK: - Foundry Triad Operations

    static func applyScenarioPatch(scenarioId: String, req: ScenarioPatchRequest, idempotencyKey: String) async throws -> [String: String] {
        try await APIClient.shared.post(
            "v1/foundry/scenarios/\(scenarioId)/patches",
            body: req,
            idempotencyKey: idempotencyKey
        )
    }

    static func commitScenario(scenarioId: String, idempotencyKey: String) async throws -> ScenarioCommitResult {
        try await APIClient.shared.postEmpty(
            "v1/foundry/scenarios/\(scenarioId)/commit",
            idempotencyKey: idempotencyKey
        )
    }

    static func triggerLotRecall(req: LotRecallRequest, idempotencyKey: String) async throws -> RecallBlastRadiusReport {
        try await APIClient.shared.post(
            "v1/foundry/ontology/lot-recall",
            body: req,
            idempotencyKey: idempotencyKey
        )
    }

    static func executeAction(req: ActionExecutionRequest, idempotencyKey: String) async throws -> ActionExecutionRecord {
        try await APIClient.shared.post(
            "v1/foundry/kinetic/actions/execute",
            body: req,
            idempotencyKey: idempotencyKey
        )
    }
}
