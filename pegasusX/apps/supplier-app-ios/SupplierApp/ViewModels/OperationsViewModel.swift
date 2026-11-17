import Foundation
import Observation

@Observable
@MainActor
final class OperationsViewModel {
    var loading: Bool = true
    var loadGeneration: Int = 0
    var error: String?
    var empathy: SupplierEmpathyAdoption?
    var title: String = ""
    var bodyText: String = ""
    var broadcastRole: String = "ALL"
    var orderId: String = ""
    var bypassReason: String = ""
    var bypassToken: String?
    var showBypassConfirm: Bool = false
    var broadcasting: Bool = false
    var replenishing: Bool = false
    var bypassing: Bool = false
    var statusMessage: String?
    var templateDate: String = ""

    func loadEmpathy(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = empathy != nil || (error == nil && !loading)
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let loaded = try await SupplierOperationsService.empathyAdoption()
            guard generation == loadGeneration else { return }
            empathy = loaded
            SupplierScreenLoad.succeed(error: &error)
        } catch {
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.fail(
                silent: silent,
                hasContent: hadContent,
                underlying: error,
                error: &self.error,
                noun: "data"
            )
        }
        if generation == loadGeneration {
            loading = false
        }
    }

    func sendBroadcast(supplierId: String?) async {
        broadcasting = true
        statusMessage = nil
        defer { broadcasting = false }
        do {
            let trimmedTitle = title.trimmingCharacters(in: .whitespacesAndNewlines)
            let trimmedBody = bodyText.trimmingCharacters(in: .whitespacesAndNewlines)
            let scopeId = supplierId ?? "supplier"
            let key = SupplierIdempotency.broadcast(
                scopeId: scopeId,
                role: broadcastRole,
                title: trimmedTitle,
                body: trimmedBody
            )
            let response = try await SupplierOperationsService.broadcast(
                SupplierBroadcastRequest(
                    title: trimmedTitle,
                    body: trimmedBody,
                    role: broadcastRole
                ),
                idempotencyKey: key
            )
            statusMessage = L10n.format("mobile_supplier.chrome.broadcast_status", response.status)
            title = ""
            bodyText = ""
        } catch {
            statusMessage = error.localizedDescription
            SupplierActionFailure.record(underlying: error, hasContent: true, error: &self.error)
        }
    }

    func runReplenishment() {
        replenishing = true
        statusMessage = nil
        Task {
            defer { replenishing = false }
            do {
                let scope = SupplierIdempotencyKeys.supplierScopeId()
                let response = try await SupplierOperationsService.triggerReplenishment(
                    idempotencyKey: SupplierIdempotencyKeys.replenishmentTrigger(
                        scopeId: scope,
                        day: SupplierIdempotencyKeys.utcDay(Date())
                    )
                )
                await MainActor.run {
                    statusMessage = L10n.format("mobile_supplier.chrome.replenishment_status", response.status)
                }
            } catch {
                await MainActor.run {
                    statusMessage = error.localizedDescription
                    SupplierActionFailure.record(underlying: error, hasContent: true, error: &self.error)
                }
            }
        }
    }

    func issueBypass() async {
        bypassing = true
        statusMessage = nil
        defer { bypassing = false }
        do {
            let trimmedOrderId = orderId.trimmingCharacters(in: .whitespacesAndNewlines)
            let reason = bypassReason.trimmingCharacters(in: .whitespacesAndNewlines)
            let key = SupplierIdempotency.paymentBypass(orderId: trimmedOrderId, reason: reason)
            let response = try await SupplierOperationsService.issuePaymentBypass(
                PaymentBypassRequest(
                    orderId: trimmedOrderId,
                    reason: reason
                ),
                idempotencyKey: key
            )
            bypassToken = response.bypassToken.isEmpty ? nil : response.bypassToken
            statusMessage = L10n.format("mobile_supplier.chrome.bypass_issued", response.orderId)
        } catch {
            statusMessage = error.localizedDescription
            SupplierActionFailure.record(underlying: error, hasContent: true, error: &self.error)
        }
    }
}
