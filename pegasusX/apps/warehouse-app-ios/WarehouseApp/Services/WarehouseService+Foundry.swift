//
//  WarehouseService+Foundry.swift
//  WarehouseAppIOS
//

import Foundation

extension WarehouseService {

    // MARK: - Role 7: Cashier & Vault Operations

    static func startCashierSession(req: StartCashierSessionRequest) async throws -> CashierSessionView {
        try await APIClient.shared.post("v1/foundry/cashier/sessions", body: req)
    }

    static func submitBlindCount(sessionId: String, req: BlindCountRequest) async throws -> CashierSessionView {
        try await APIClient.shared.post("v1/foundry/cashier/sessions/\(sessionId)/blind-count", body: req)
    }

    static func reclassifyTender(req: RRNTenderReclassificationRequest) async throws -> CashierSessionView {
        try await APIClient.shared.post("v1/foundry/cashier/reclassify-tender", body: req)
    }

    static func executeCITHandover(req: CITHandoverRequest) async throws -> DepotVaultLedgerEntry {
        try await APIClient.shared.post("v1/foundry/cashier/vault/cit-handover", body: req)
    }

    static func checkAML(req: AMLCheckRequest) async throws -> AMLCheckResponse {
        try await APIClient.shared.post("v1/foundry/guards/aml-check", body: req)
    }

    // MARK: - Role 9: Yard Operations

    static func recordGateEvent(event: YardGateCheckEvent) async throws -> YardGateCheckEvent {
        try await APIClient.shared.post("v1/foundry/fleet/gate-events", body: event)
    }
}
