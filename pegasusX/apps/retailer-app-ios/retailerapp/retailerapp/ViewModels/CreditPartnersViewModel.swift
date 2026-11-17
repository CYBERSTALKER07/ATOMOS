import Foundation
import Observation

@Observable
@MainActor
final class CreditPartnersViewModel {
    struct Service {
        var fetchRelationships: () async throws -> [CreditRelationshipWire]
        var fetchStatement: (_ supplierId: String?, _ limit: Int) async throws -> ArStatementWire
        var fetchInvoices: (_ status: String, _ limit: Int) async throws -> [ArInvoiceWire]
        var fetchPromises: (_ supplierId: String?) async throws -> ArPromisesResponse
        var submitPayFifo: (_ request: PayArFifoRequest, _ idempotencyKey: String) async throws -> PayArFifoResponse
        var submitPayInvoice: (_ invoiceId: String, _ request: PayArInvoiceRequest, _ idempotencyKey: String) async throws -> Void
        var submitPromise: (_ request: CreateArPromiseRequest, _ idempotencyKey: String) async throws -> ArPromiseWire
        var fetchOpeningBalances: () async throws -> [MigrationOpeningInvoiceWire]
        var submitAcknowledgeOpeningBalance: (_ invoiceId: String, _ idempotencyKey: String) async throws -> MigrationOpeningInvoiceWire
        var submitDisputeOpeningBalance: (_ invoiceId: String, _ reason: String, _ idempotencyKey: String) async throws -> MigrationOpeningInvoiceWire

        init(
            fetchRelationships: @escaping () async throws -> [CreditRelationshipWire],
            fetchStatement: @escaping (_ supplierId: String?, _ limit: Int) async throws -> ArStatementWire,
            fetchInvoices: @escaping (_ status: String, _ limit: Int) async throws -> [ArInvoiceWire],
            fetchPromises: @escaping (_ supplierId: String?) async throws -> ArPromisesResponse,
            submitPayFifo: @escaping (_ request: PayArFifoRequest, _ idempotencyKey: String) async throws -> PayArFifoResponse,
            submitPayInvoice: @escaping (_ invoiceId: String, _ request: PayArInvoiceRequest, _ idempotencyKey: String) async throws -> Void,
            submitPromise: @escaping (_ request: CreateArPromiseRequest, _ idempotencyKey: String) async throws -> ArPromiseWire,
            fetchOpeningBalances: @escaping () async throws -> [MigrationOpeningInvoiceWire] = { [] },
            submitAcknowledgeOpeningBalance: @escaping (_ invoiceId: String, _ idempotencyKey: String) async throws -> MigrationOpeningInvoiceWire = { id, _ in
                MigrationOpeningInvoiceWire(invoiceId: id, supplierId: "", principalMinor: 0, balanceMinor: 0, status: "ACKNOWLEDGED", dunningFrozen: false)
            },
            submitDisputeOpeningBalance: @escaping (_ invoiceId: String, _ reason: String, _ idempotencyKey: String) async throws -> MigrationOpeningInvoiceWire = { id, reason, _ in
                MigrationOpeningInvoiceWire(invoiceId: id, supplierId: "", principalMinor: 0, balanceMinor: 0, status: "DISPUTED", dunningFrozen: true, disputeReason: reason)
            }
        ) {
            self.fetchRelationships = fetchRelationships
            self.fetchStatement = fetchStatement
            self.fetchInvoices = fetchInvoices
            self.fetchPromises = fetchPromises
            self.submitPayFifo = submitPayFifo
            self.submitPayInvoice = submitPayInvoice
            self.submitPromise = submitPromise
            self.fetchOpeningBalances = fetchOpeningBalances
            self.submitAcknowledgeOpeningBalance = submitAcknowledgeOpeningBalance
            self.submitDisputeOpeningBalance = submitDisputeOpeningBalance
        }

        static let live = Service(
            fetchRelationships: { try await APIClient.shared.getCreditRelationships() },
            fetchStatement: { try await APIClient.shared.getArStatement(supplierId: $0, limit: $1) },
            fetchInvoices: { try await APIClient.shared.getArInvoices(status: $0, limit: $1) },
            fetchPromises: { try await APIClient.shared.getArPromises(supplierId: $0) },
            submitPayFifo: { try await APIClient.shared.payArFifo(request: $0, idempotencyKey: $1) },
            submitPayInvoice: { try await APIClient.shared.payArInvoice(invoiceId: $0, request: $1, idempotencyKey: $2) },
            submitPromise: { try await APIClient.shared.createArPromise(request: $0, idempotencyKey: $1) },
            fetchOpeningBalances: { try await APIClient.shared.listNasiyaOpeningBalances() },
            submitAcknowledgeOpeningBalance: { try await APIClient.shared.acknowledgeOpeningBalance(invoiceId: $0, idempotencyKey: $1) },
            submitDisputeOpeningBalance: { try await APIClient.shared.disputeOpeningBalance(invoiceId: $0, reason: $1, idempotencyKey: $2) }
        )
    }

    var relationships: [CreditRelationshipWire] = []
    var selectedSupplierId: String = ""
    var statement: ArStatementWire?
    var invoices: [ArInvoiceWire] = []
    var openingBalances: [MigrationOpeningInvoiceWire] = []
    var openingBusyInvoiceId: String?
    var openingError: String?
    var openingNotice: String?
    var promises: [ArPromiseWire] = []
    var currentQuarterKey: String = ""
    var selfServePromiseUsed: Bool = false
    var maxExtensionDays: Int = 14

    var isLoading = false
    var isRefreshingDetail = false
    var isSubmittingPayment = false
    var isCreatingPromise = false

    var loadError: String?
    var payError: String?
    var paySuccessPaidMinor: Int64?
    var paySuccessAdvanceMinor: Int64 = 0
    var promiseError: String?
    var promiseCreated = false
    var quarterlyLimitError: QuarterlyPromiseLimitError?

    private let service: Service

    init(service: Service = .live) {
        self.service = service
    }

    var activeRelationship: CreditRelationshipWire? {
        relationships.first { $0.supplierId == selectedSupplierId } ?? relationships.first
    }

    var totalOpenMinor: Int64 {
        statement?.totalOpenMinor ?? invoices.reduce(0) { $0 + $1.balanceMinor }
    }

    var totalDisputedMinor: Int64 {
        statement?.totalDisputedMinor ?? invoices.reduce(0) { $0 + $1.disputedMinor }
    }

    var totalUndisputedMinor: Int64 {
        statement?.totalUndisputedMinor ?? invoices.reduce(0) { $0 + $1.effectiveUndisputedMinor }
    }

    var advanceBalanceMinor: Int64 {
        statement?.advanceBalanceMinor ?? 0
    }

    func load() async {
        isLoading = true
        loadError = nil
        openingError = nil
        defer { isLoading = false }

        do {
            openingBalances = try await service.fetchOpeningBalances()
        } catch {
            openingError = RetailerW3ContractHelper.parseArErrorMessage(
                from: error,
                fallback: String(localized: "mobile_retailer.credit.load_error")
            )
        }

        do {
            let rels = try await service.fetchRelationships()
            relationships = rels
            let nextSupplier = (!selectedSupplierId.isEmpty && rels.contains { $0.supplierId == selectedSupplierId })
                ? selectedSupplierId
                : (rels.first?.supplierId ?? "")
            selectedSupplierId = nextSupplier

            if !nextSupplier.isEmpty {
                await refreshSupplierDetail(supplierId: nextSupplier)
            } else {
                let openInvoices = (try? await service.fetchInvoices("OPEN,PARTIAL", 50)) ?? []
                statement = nil
                invoices = RetailerW3ContractHelper.sortInvoicesOldestFirst(openInvoices)
                promises = []
            }
        } catch {
            loadError = RetailerW3ContractHelper.parseArErrorMessage(
                from: error,
                fallback: String(localized: "mobile_retailer.credit.load_error")
            )
        }
    }

    func selectSupplier(_ supplierId: String) async {
        selectedSupplierId = supplierId
        payError = nil
        paySuccessPaidMinor = nil
        paySuccessAdvanceMinor = 0
        promiseError = nil
        promiseCreated = false
        quarterlyLimitError = nil
        await refreshSupplierDetail(supplierId: supplierId)
    }

    func refreshSupplierDetail(supplierId: String) async {
        guard !supplierId.isEmpty else { return }
        isRefreshingDetail = true
        defer { isRefreshingDetail = false }

        let stmt = try? await service.fetchStatement(supplierId, 50)
        let invFallback = ((try? await service.fetchInvoices("OPEN,PARTIAL", 50)) ?? [])
            .filter { $0.supplierId.isEmpty || $0.supplierId == supplierId }
        let promEnvelope = try? await service.fetchPromises(supplierId)

        statement = stmt
        let rawInvoices = (stmt?.invoices.isEmpty == false) ? (stmt?.invoices ?? []) : invFallback
        invoices = RetailerW3ContractHelper.sortInvoicesOldestFirst(rawInvoices)
        let rawPromises = (stmt?.promises.isEmpty == false) ? (stmt?.promises ?? []) : (promEnvelope?.promises ?? [])
        promises = rawPromises

        currentQuarterKey = stmt?.currentQuarterKey.isEmpty == false
            ? (stmt?.currentQuarterKey ?? "")
            : (promEnvelope?.currentQuarterKey ?? "")
        selfServePromiseUsed = stmt?.selfServePromiseUsed ?? (promEnvelope?.selfServePromiseUsed ?? false)
        maxExtensionDays = stmt?.maxExtensionDays ?? (promEnvelope?.maxExtensionDays ?? 14)
    }

    func payFifo(
        supplierId: String,
        amountMinor: Int64,
        paymentMethod: String,
        paymentReference: String? = nil,
        earmarkInvoiceId: String? = nil,
        idempotencyKey: String = UUID().uuidString
    ) async -> Bool {
        isSubmittingPayment = true
        payError = nil
        paySuccessPaidMinor = nil
        paySuccessAdvanceMinor = 0
        defer { isSubmittingPayment = false }

        let cleanRef = paymentReference?.trimmingCharacters(in: .whitespacesAndNewlines)
        let cleanEarmark = earmarkInvoiceId?.trimmingCharacters(in: .whitespacesAndNewlines)
        let currency = (statement?.currency.isEmpty == false ? statement?.currency : nil) ?? "UZS"

        do {
            let result = try await service.submitPayFifo(
                PayArFifoRequest(
                    supplierId: supplierId,
                    amountMinor: amountMinor,
                    currency: currency,
                    paymentMethod: paymentMethod,
                    paymentReference: (cleanRef?.isEmpty == false) ? cleanRef : nil,
                    earmarkInvoiceId: (cleanEarmark?.isEmpty == false) ? cleanEarmark : nil
                ),
                idempotencyKey
            )
            paySuccessPaidMinor = result.paidMinor
            paySuccessAdvanceMinor = result.advanceAddedMinor
            await load()
            return true
        } catch {
            payError = RetailerW3ContractHelper.parseArErrorMessage(
                from: error,
                fallback: String(localized: "mobile_retailer.credit.pay_error")
            )
            return false
        }
    }

    func payInvoice(
        invoiceId: String,
        amountMinor: Int64,
        paymentMethod: String,
        paymentReference: String? = nil,
        idempotencyKey: String = UUID().uuidString
    ) async -> Bool {
        isSubmittingPayment = true
        payError = nil
        paySuccessPaidMinor = nil
        paySuccessAdvanceMinor = 0
        defer { isSubmittingPayment = false }

        let cleanRef = paymentReference?.trimmingCharacters(in: .whitespacesAndNewlines)
        do {
            try await service.submitPayInvoice(
                invoiceId,
                PayArInvoiceRequest(
                    amountMinor: amountMinor,
                    paymentMethod: paymentMethod,
                    paymentReference: (cleanRef?.isEmpty == false) ? cleanRef : nil
                ),
                idempotencyKey
            )
            paySuccessPaidMinor = amountMinor
            paySuccessAdvanceMinor = 0
            await load()
            return true
        } catch {
            payError = RetailerW3ContractHelper.parseArErrorMessage(
                from: error,
                fallback: String(localized: "mobile_retailer.credit.pay_error")
            )
            return false
        }
    }

    func createPromiseExtension(
        supplierId: String,
        extensionDays: Int,
        invoiceId: String? = nil,
        idempotencyKey: String = UUID().uuidString
    ) async -> Bool {
        isCreatingPromise = true
        promiseError = nil
        promiseCreated = false
        quarterlyLimitError = nil
        defer { isCreatingPromise = false }

        let cleanInv = invoiceId?.trimmingCharacters(in: .whitespacesAndNewlines)
        do {
            _ = try await service.submitPromise(
                CreateArPromiseRequest(
                    supplierId: supplierId,
                    invoiceId: (cleanInv?.isEmpty == false) ? cleanInv : nil,
                    extensionDays: extensionDays
                ),
                idempotencyKey
            )
            promiseCreated = true
            await refreshSupplierDetail(supplierId: supplierId)
            return true
        } catch {
            if let limitErr = RetailerW3ContractHelper.parseQuarterlyPromiseLimitError(from: error) {
                quarterlyLimitError = limitErr
            } else {
                promiseError = RetailerW3ContractHelper.parseArErrorMessage(
                    from: error,
                    fallback: String(localized: "mobile_retailer.credit.promise_error")
                )
            }
            return false
        }
    }

    @discardableResult
    func confirmOpeningBalance(
        invoiceId: String,
        idempotencyKey: String? = nil
    ) async -> Bool {
        openingBusyInvoiceId = invoiceId
        openingError = nil
        openingNotice = nil
        defer { openingBusyInvoiceId = nil }

        let key = idempotencyKey ?? RetailerIdempotency.nasiyaAcknowledge(invoiceId: invoiceId)
        do {
            let updated = try await service.submitAcknowledgeOpeningBalance(invoiceId, key)
            replaceOpeningInvoice(updated)
            openingNotice = String(localized: "retailer_desktop.nasiya_opening.acknowledged_status")
            return true
        } catch {
            openingError = RetailerW3ContractHelper.parseArErrorMessage(
                from: error,
                fallback: String(localized: "mobile_retailer.credit.pay_error")
            )
            return false
        }
    }

    @discardableResult
    func disputeOpeningBalance(
        invoiceId: String,
        reason: String,
        idempotencyKey: String? = nil
    ) async -> Bool {
        let trimmed = reason.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            openingError = String(localized: "mobile_retailer.nasiya_opening.dispute_hint")
            return false
        }

        openingBusyInvoiceId = invoiceId
        openingError = nil
        openingNotice = nil
        defer { openingBusyInvoiceId = nil }

        let key = idempotencyKey ?? RetailerIdempotency.nasiyaDispute(invoiceId: invoiceId, reason: trimmed)
        do {
            let updated = try await service.submitDisputeOpeningBalance(invoiceId, trimmed, key)
            replaceOpeningInvoice(updated)
            openingNotice = String(localized: "retailer_desktop.nasiya_opening.disputed_status")
            return true
        } catch {
            openingError = RetailerW3ContractHelper.parseArErrorMessage(
                from: error,
                fallback: String(localized: "mobile_retailer.credit.pay_error")
            )
            return false
        }
    }

    private func replaceOpeningInvoice(_ updated: MigrationOpeningInvoiceWire) {
        if let idx = openingBalances.firstIndex(where: { $0.invoiceId == updated.invoiceId }) {
            openingBalances[idx] = updated
        } else {
            openingBalances.insert(updated, at: 0)
        }
    }
}
