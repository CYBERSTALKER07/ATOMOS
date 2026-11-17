import Foundation
import Observation

struct SellerMandateFormState: Equatable {
    var userId: String
    var memberName: String
    var memberRole: String = ""
    var canReceiveGoods: Bool = true
    var canPayCash: Bool = true
    var canUseCredit: Bool = true
    var dailyCreditCapSom: String = "0"
    var dailyCashCapSom: String = "0"
    var maxSingleOrderSom: String = "0"
    var dailyUsage: SellerDailyUsageDTO = SellerDailyUsageDTO()
}

@Observable
@MainActor
final class OwnerGovernanceViewModel {
    var isLoadingRole = true
    var isOwner = false
    var retailerRole = ""
    var statusFilter = "PENDING_OWNER"
    var approvals: [RetailerApprovalItem] = []
    var approvalsLoading = false
    var approvalsError: String?
    var decidingRequestId: String?
    var actionBanner: String?

    var mandateForm: SellerMandateFormState?
    var mandateLoading = false
    var mandateSaving = false
    var mandateError: String?

    private let meLoader: () async throws -> RetailerMeResponse
    private let approvalsLoader: (String, Int, String?) async throws -> RetailerApprovalsResponse
    private let approver: (String, String?) async throws -> ApprovalDecisionResponse
    private let rejector: (String, String?) async throws -> ApprovalDecisionResponse
    private let mandateLoader: (String) async throws -> SellerMandateResponse
    private let mandateUpdater: (String, SellerMandatePatchBody) async throws -> SellerMandateResponse

    convenience init(api: APIClient? = nil) {
        let client = api ?? .shared
        self.init(
            meLoader: { try await client.getRetailerMe() },
            approvalsLoader: { status, limit, cursor in
                try await client.getRetailerApprovals(status: status, limit: limit, cursor: cursor)
            },
            approver: { requestId, reason in
                try await client.approveRetailerRequest(requestId: requestId, reason: reason)
            },
            rejector: { requestId, reason in
                try await client.rejectRetailerRequest(requestId: requestId, reason: reason)
            },
            mandateLoader: { userId in
                try await client.getSellerMandate(userId: userId)
            },
            mandateUpdater: { userId, body in
                try await client.updateSellerMandate(userId: userId, body: body)
            }
        )
    }

    init(
        meLoader: @escaping () async throws -> RetailerMeResponse,
        approvalsLoader: @escaping (String, Int, String?) async throws -> RetailerApprovalsResponse,
        approver: @escaping (String, String?) async throws -> ApprovalDecisionResponse,
        rejector: @escaping (String, String?) async throws -> ApprovalDecisionResponse,
        mandateLoader: @escaping (String) async throws -> SellerMandateResponse,
        mandateUpdater: @escaping (String, SellerMandatePatchBody) async throws -> SellerMandateResponse
    ) {
        self.meLoader = meLoader
        self.approvalsLoader = approvalsLoader
        self.approver = approver
        self.rejector = rejector
        self.mandateLoader = mandateLoader
        self.mandateUpdater = mandateUpdater
    }

    func loadRoleAndApprovals() async {
        isLoadingRole = true
        approvalsError = nil
        do {
            let me = try await meLoader()
            retailerRole = me.retailerRole.uppercased()
            isOwner = me.resolvedIsOwner
            isLoadingRole = false
            if isOwner {
                await refreshApprovalsInternal(status: statusFilter)
            }
        } catch {
            isLoadingRole = false
            isOwner = false
            approvalsError = error.localizedDescription
        }
    }

    func selectStatusFilter(_ status: String) async {
        statusFilter = status
        actionBanner = nil
        guard isOwner else { return }
        await refreshApprovalsInternal(status: status)
    }

    func refreshApprovals() async {
        guard isOwner else {
            await loadRoleAndApprovals()
            return
        }
        await refreshApprovalsInternal(status: statusFilter)
    }

    private func refreshApprovalsInternal(status: String) async {
        approvalsLoading = true
        approvalsError = nil
        defer { approvalsLoading = false }
        do {
            let res = try await approvalsLoader(status, 20, nil)
            approvals = res.items
        } catch {
            approvalsError = error.localizedDescription
        }
    }

    func approveRequest(_ requestId: String, reason: String? = nil) async {
        await decide(requestId: requestId, approve: true, reason: reason)
    }

    func rejectRequest(_ requestId: String, reason: String? = nil) async {
        await decide(requestId: requestId, approve: false, reason: reason)
    }

    private func decide(requestId: String, approve: Bool, reason: String?) async {
        guard isOwner, decidingRequestId == nil else { return }
        decidingRequestId = requestId
        approvalsError = nil
        actionBanner = nil
        let trimmed = reason?.trimmingCharacters(in: .whitespacesAndNewlines)
        let cleanReason = (trimmed?.isEmpty == false) ? trimmed : nil
        do {
            if approve {
                _ = try await approver(requestId, cleanReason)
                actionBanner = "Order approved and dispatched."
            } else {
                _ = try await rejector(requestId, cleanReason)
                actionBanner = "Approval request rejected."
            }
            decidingRequestId = nil
            await refreshApprovalsInternal(status: statusFilter)
        } catch {
            decidingRequestId = nil
            approvalsError = error.localizedDescription
        }
    }

    func openMandateEditor(userId: String, memberName: String) async {
        guard isOwner else { return }
        mandateError = nil
        actionBanner = nil
        mandateLoading = true
        mandateForm = SellerMandateFormState(userId: userId, memberName: memberName)
        do {
            let res = try await mandateLoader(userId)
            mandateForm = SellerMandateFormState(
                userId: userId,
                memberName: memberName,
                memberRole: res.retailerRole,
                canReceiveGoods: res.mandate.canReceiveGoods,
                canPayCash: res.mandate.canPayCash,
                canUseCredit: res.mandate.canUseCredit,
                dailyCreditCapSom: RetailerMoney.minorToSomInput(res.mandate.dailyCreditCapMinor),
                dailyCashCapSom: RetailerMoney.minorToSomInput(res.mandate.dailyCashCapMinor),
                maxSingleOrderSom: RetailerMoney.minorToSomInput(res.mandate.maxSingleOrderMinor),
                dailyUsage: res.dailyUsage
            )
        } catch {
            mandateError = error.localizedDescription
        }
        mandateLoading = false
    }

    func saveMandate() async {
        guard isOwner, let form = mandateForm, !mandateSaving else { return }
        guard let creditMinor = RetailerMoney.somInputToMinor(form.dailyCreditCapSom),
              let cashMinor = RetailerMoney.somInputToMinor(form.dailyCashCapSom),
              let singleMinor = RetailerMoney.somInputToMinor(form.maxSingleOrderSom) else {
            mandateError = "Enter valid whole-number so'm amounts (0 or greater)."
            return
        }

        mandateSaving = true
        mandateError = nil
        defer { mandateSaving = false }
        do {
            let body = SellerMandatePatchBody(
                canReceiveGoods: form.canReceiveGoods,
                canPayCash: form.canPayCash,
                canUseCredit: form.canUseCredit,
                dailyCreditCapMinor: creditMinor,
                dailyCashCapMinor: cashMinor,
                maxSingleOrderMinor: singleMinor
            )
            _ = try await mandateUpdater(form.userId, body)
            mandateForm = nil
            actionBanner = "Saved seller mandate for \(form.memberName)"
        } catch {
            mandateError = error.localizedDescription
        }
    }
}
