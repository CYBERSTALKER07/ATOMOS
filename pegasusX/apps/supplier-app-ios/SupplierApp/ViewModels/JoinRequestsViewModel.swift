import Foundation
import Observation

@Observable
@MainActor
final class JoinRequestsViewModel {
    var statusFilter = "PENDING"
    var requests: [JoinRequestItem] = []
    var rejectionReasonById: [String: String] = [:]
    var loading = true
    var busyId: String?
    var error: String?
    var notice: String?

    func load(silent: Bool = false) async {
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        defer { loading = false }
        do {
            requests = try await MigrationService.listJoinRequests(status: statusFilter)
            SupplierScreenLoad.succeed(error: &error)
        } catch {
            SupplierScreenLoad.fail(
                silent: silent,
                hasContent: !requests.isEmpty,
                underlying: error,
                error: &self.error,
                noun: "join requests"
            )
        }
    }

    func decide(requestId: String, approve: Bool) async {
        guard busyId == nil else { return }
        busyId = requestId
        error = nil
        notice = nil
        defer { busyId = nil }
        let reason = rejectionReasonById[requestId]?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let decision = approve ? "approve" : "reject"
        let key = SupplierIdempotencyKeys.joinRequestDecide(requestId: requestId, decision: decision)
        do {
            if approve {
                try await MigrationService.approveJoinRequest(requestId: requestId, reason: reason, idempotencyKey: key)
                PegasusHaptics.confirm()
            } else {
                try await MigrationService.rejectJoinRequest(requestId: requestId, reason: reason, idempotencyKey: key)
                PegasusHaptics.warning()
            }
            notice = L10n.format("supplier_portal.join_requests.decided_notice")
            await load(silent: true)
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: !requests.isEmpty, error: &self.error)
            PegasusHaptics.error()
        }
    }
}
