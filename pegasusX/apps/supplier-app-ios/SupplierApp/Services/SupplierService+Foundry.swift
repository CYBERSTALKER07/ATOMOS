//
//  SupplierService+Foundry.swift
//  SupplierAppIOS
//

import Foundation

extension SupplierService {

    // MARK: - Foundry Triad Operations

    static func applyScenarioPatch(scenarioId: String, req: ScenarioPatchRequest) async throws -> [String: String] {
        try await APIClient.shared.post("v1/foundry/scenarios/\(scenarioId)/patches", body: req)
    }

    static func commitScenario(scenarioId: String) async throws -> ScenarioCommitResult {
        try await APIClient.shared.postEmpty("v1/foundry/scenarios/\(scenarioId)/commit")
    }

    static func triggerLotRecall(req: LotRecallRequest) async throws -> RecallBlastRadiusReport {
        try await APIClient.shared.post("v1/foundry/ontology/lot-recall", body: req)
    }

    static func executeAction(req: ActionExecutionRequest) async throws -> ActionExecutionRecord {
        try await APIClient.shared.post("v1/foundry/kinetic/actions/execute", body: req)
    }
}
