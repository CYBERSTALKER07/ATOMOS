import Foundation
import Observation

@Observable
@MainActor
final class TeamViewModel {
    var members: [RetailerOrgMember] = []
    var joinRequests: [RetailerJoinRequestItem] = []
    var decidingRequestId: String?
    var loading = true
    var errorText: String?
    var banner: String?
    var name = ""
    var phone = ""
    var password = ""
    var role = "CASHIER"
    var busy = false

    private let getOrgMembersHandler: () async throws -> RetailerOrgMembersResponse
    private let createOrgMemberHandler: (String, String, String, String) async throws -> RetailerOrgMembersResponse
    private let deactivateOrgMemberHandler: (String) async throws -> RetailerOrgMembersResponse
    private let listJoinRequestsHandler: (String) async throws -> [RetailerJoinRequestItem]
    private let approveJoinRequestHandler: (String, String?, String) async throws -> RetailerJoinRequestItem
    private let rejectJoinRequestHandler: (String, String?, String) async throws -> RetailerJoinRequestItem

    init(
        getOrgMembers: @escaping () async throws -> RetailerOrgMembersResponse = { try await APIClient.shared.getOrgMembers() },
        createOrgMember: @escaping (String, String, String, String) async throws -> RetailerOrgMembersResponse = { name, phone, password, role in
            try await APIClient.shared.createOrgMember(name: name, phone: phone, password: password, role: role)
        },
        deactivateOrgMember: @escaping (String) async throws -> RetailerOrgMembersResponse = { userId in
            try await APIClient.shared.deactivateOrgMember(userId: userId)
        },
        listJoinRequests: @escaping (String) async throws -> [RetailerJoinRequestItem] = { status in
            try await APIClient.shared.listJoinRequests(status: status)
        },
        approveJoinRequest: @escaping (String, String?, String) async throws -> RetailerJoinRequestItem = { requestId, reason, idempotencyKey in
            try await APIClient.shared.approveJoinRequest(requestId: requestId, reason: reason, idempotencyKey: idempotencyKey)
        },
        rejectJoinRequest: @escaping (String, String?, String) async throws -> RetailerJoinRequestItem = { requestId, reason, idempotencyKey in
            try await APIClient.shared.rejectJoinRequest(requestId: requestId, reason: reason, idempotencyKey: idempotencyKey)
        }
    ) {
        self.getOrgMembersHandler = getOrgMembers
        self.createOrgMemberHandler = createOrgMember
        self.deactivateOrgMemberHandler = deactivateOrgMember
        self.listJoinRequestsHandler = listJoinRequests
        self.approveJoinRequestHandler = approveJoinRequest
        self.rejectJoinRequestHandler = rejectJoinRequest
    }

    func load() async {
        loading = true
        errorText = nil
        defer { loading = false }
        do {
            let res = try await getOrgMembersHandler()
            members = res.items
        } catch {
            errorText = error.localizedDescription
        }
        await loadJoinRequests()
    }

    func loadJoinRequests() async {
        do {
            joinRequests = try await listJoinRequestsHandler("PENDING")
        } catch {
            // Non-admin roles or early setup sessions may not hold pending join requests.
        }
    }

    func approveJoinRequest(requestId: String, reason: String? = nil) async -> Bool {
        decidingRequestId = requestId
        defer { decidingRequestId = nil }
        do {
            _ = try await approveJoinRequestHandler(
                requestId,
                reason,
                RetailerIdempotency.joinRequestDecide(requestId: requestId, decision: "approve")
            )
            joinRequests.removeAll { $0.requestId == requestId }
            banner = "Join request approved"
            if let res = try? await getOrgMembersHandler() {
                members = res.items
            }
            Haptics.success()
            return true
        } catch {
            banner = error.localizedDescription
            Haptics.error()
            return false
        }
    }

    func rejectJoinRequest(requestId: String, reason: String? = nil) async -> Bool {
        decidingRequestId = requestId
        defer { decidingRequestId = nil }
        do {
            _ = try await rejectJoinRequestHandler(
                requestId,
                reason,
                RetailerIdempotency.joinRequestDecide(requestId: requestId, decision: "reject")
            )
            joinRequests.removeAll { $0.requestId == requestId }
            banner = "Join request rejected"
            Haptics.selection()
            return true
        } catch {
            banner = error.localizedDescription
            Haptics.error()
            return false
        }
    }

    func invite() async {
        busy = true
        defer { busy = false }
        do {
            let res = try await createOrgMemberHandler(
                name.trimmingCharacters(in: .whitespacesAndNewlines),
                phone.trimmingCharacters(in: .whitespacesAndNewlines),
                password,
                role.trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
            )
            members = res.items
            banner = "Member created"
            name = ""
            phone = ""
            password = ""
            Haptics.success()
        } catch {
            banner = error.localizedDescription
            Haptics.error()
        }
    }

    func deactivate(_ userId: String) async {
        busy = true
        defer { busy = false }
        do {
            let res = try await deactivateOrgMemberHandler(userId)
            members = res.items
            banner = "Deactivated"
            Haptics.selection()
        } catch {
            banner = error.localizedDescription
            Haptics.error()
        }
    }
}
