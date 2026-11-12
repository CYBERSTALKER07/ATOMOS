//
//  ExceptionsViewModel.swift
//  payload-app-ios
//
//  Phase 3 MVVM decomposition: exceptions tracking, dock damage, and escalation.
//

import Foundation
import Observation

@Observable
@MainActor
final class ExceptionsViewModel {
    var exceptions: [ManifestExceptionRow] = []
    var loading = false
    var error: String?
    var markingDamaged = false
    var damageReportedMessage: String?
    var overflowAttempts = 0
    var isEscalatedToDlq: Bool { overflowAttempts >= 3 }

    private let api: APIClient

    init(api: APIClient = .shared) {
        self.api = api
    }

    func loadExceptions() async {
        guard !loading else { return }
        loading = true
        error = nil
        defer { loading = false }
        do {
            let resp = try await api.manifestExceptionsList()
            exceptions = resp.exceptions
            overflowAttempts = exceptions.filter { $0.reason == "OVERFLOW" }.count
        } catch {
            self.error = error.localizedDescription
        }
    }

    func reportDockDamage(orderId: String, manifestId: String) async {
        guard !markingDamaged else { return }
        markingDamaged = true
        error = nil
        defer { markingDamaged = false }
        do {
            _ = try await api.manifestException(
                manifestId: manifestId,
                orderId: orderId,
                reason: "DAMAGED"
            )
            damageReportedMessage = "Dock damage reported."
            await loadExceptions()
        } catch {
            self.error = error.localizedDescription
        }
    }
}
