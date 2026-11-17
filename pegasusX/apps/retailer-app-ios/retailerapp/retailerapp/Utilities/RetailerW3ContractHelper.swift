import Foundation

struct CreditRelationshipResolvedTermsWire: Decodable, Equatable, Sendable {
    let termsDays: Int?
    let gracePeriodDays: Int?

    enum CodingKeys: String, CodingKey {
        case termsDays = "terms_days"
        case gracePeriodDays = "grace_period_days"
    }
}

struct CreditRelationshipWire: Decodable, Identifiable, Equatable, Sendable {
    let supplierId: String
    let supplierName: String?
    let creditEnabled: Bool
    let termsDays: Int
    let gracePeriodDays: Int
    let creditLimitMinor: Int64
    let availableCreditMinor: Int64
    let currentBalanceMinor: Int64
    let profileStatus: String?
    let onHold: Bool
    let prepayOnlyLocked: Bool
    let resolvedTerms: CreditRelationshipResolvedTermsWire?

    var id: String { supplierId }

    var effectiveTermsDays: Int {
        termsDays > 0 ? termsDays : (resolvedTerms?.termsDays ?? termsDays)
    }

    var effectiveGraceDays: Int {
        gracePeriodDays > 0 ? gracePeriodDays : (resolvedTerms?.gracePeriodDays ?? gracePeriodDays)
    }

    enum CodingKeys: String, CodingKey {
        case supplierId = "supplier_id"
        case supplierName = "supplier_name"
        case creditEnabled = "credit_enabled"
        case termsDays = "terms_days"
        case gracePeriodDays = "grace_period_days"
        case creditLimitMinor = "credit_limit_minor"
        case availableCreditMinor = "available_credit_minor"
        case currentBalanceMinor = "current_balance_minor"
        case profileStatus = "profile_status"
        case onHold = "on_hold"
        case prepayOnlyLocked = "prepay_only_locked"
        case resolvedTerms = "resolved_terms"
    }

    init(
        supplierId: String,
        supplierName: String? = nil,
        creditEnabled: Bool = true,
        termsDays: Int = 0,
        gracePeriodDays: Int = 0,
        creditLimitMinor: Int64 = 0,
        availableCreditMinor: Int64 = 0,
        currentBalanceMinor: Int64 = 0,
        profileStatus: String? = nil,
        onHold: Bool = false,
        prepayOnlyLocked: Bool = false,
        resolvedTerms: CreditRelationshipResolvedTermsWire? = nil
    ) {
        self.supplierId = supplierId
        self.supplierName = supplierName
        self.creditEnabled = creditEnabled
        self.termsDays = termsDays
        self.gracePeriodDays = gracePeriodDays
        self.creditLimitMinor = creditLimitMinor
        self.availableCreditMinor = availableCreditMinor
        self.currentBalanceMinor = currentBalanceMinor
        self.profileStatus = profileStatus
        self.onHold = onHold
        self.prepayOnlyLocked = prepayOnlyLocked
        self.resolvedTerms = resolvedTerms
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        supplierId = try c.decodeIfPresent(String.self, forKey: .supplierId) ?? "—"
        supplierName = try c.decodeIfPresent(String.self, forKey: .supplierName)
        creditEnabled = try c.decodeIfPresent(Bool.self, forKey: .creditEnabled) ?? false
        termsDays = try c.decodeIfPresent(Int.self, forKey: .termsDays) ?? 0
        gracePeriodDays = try c.decodeIfPresent(Int.self, forKey: .gracePeriodDays) ?? 0
        creditLimitMinor = try c.decodeIfPresent(Int64.self, forKey: .creditLimitMinor) ?? 0
        availableCreditMinor = try c.decodeIfPresent(Int64.self, forKey: .availableCreditMinor) ?? 0
        currentBalanceMinor = try c.decodeIfPresent(Int64.self, forKey: .currentBalanceMinor) ?? 0
        profileStatus = try c.decodeIfPresent(String.self, forKey: .profileStatus)
        onHold = try c.decodeIfPresent(Bool.self, forKey: .onHold) ?? false
        prepayOnlyLocked = try c.decodeIfPresent(Bool.self, forKey: .prepayOnlyLocked) ?? false
        resolvedTerms = try c.decodeIfPresent(CreditRelationshipResolvedTermsWire.self, forKey: .resolvedTerms)
    }
}

struct CreditRelationshipsResponse: Decodable {
    let relationships: [CreditRelationshipWire]?
}

struct ArInvoiceWire: Decodable, Identifiable, Equatable, Sendable {
    let invoiceId: String
    let supplierId: String
    let orderId: String?
    let status: String
    let principalMinor: Int64?
    let balanceMinor: Int64
    let disputedMinor: Int64
    let undisputedMinor: Int64?
    let currency: String?
    let dueAt: String
    let createdAt: String?
    let termsDays: Int?
    let gracePeriodDays: Int?
    let agingBucket: String?
    let dunningStep: Int?

    var id: String { invoiceId }

    var effectiveUndisputedMinor: Int64 {
        undisputedMinor ?? max(0, balanceMinor - disputedMinor)
    }

    enum CodingKeys: String, CodingKey {
        case invoiceId = "invoice_id"
        case supplierId = "supplier_id"
        case orderId = "order_id"
        case status
        case principalMinor = "principal_minor"
        case balanceMinor = "balance_minor"
        case disputedMinor = "disputed_minor"
        case undisputedMinor = "undisputed_minor"
        case currency
        case dueAt = "due_at"
        case createdAt = "created_at"
        case termsDays = "terms_days"
        case gracePeriodDays = "grace_period_days"
        case agingBucket = "aging_bucket"
        case dunningStep = "dunning_step"
    }

    init(
        invoiceId: String,
        supplierId: String = "",
        orderId: String? = nil,
        status: String = "OPEN",
        principalMinor: Int64? = nil,
        balanceMinor: Int64 = 0,
        disputedMinor: Int64 = 0,
        undisputedMinor: Int64? = nil,
        currency: String? = "UZS",
        dueAt: String = "—",
        createdAt: String? = nil,
        termsDays: Int? = nil,
        gracePeriodDays: Int? = nil,
        agingBucket: String? = nil,
        dunningStep: Int? = nil
    ) {
        self.invoiceId = invoiceId
        self.supplierId = supplierId
        self.orderId = orderId
        self.status = status
        self.principalMinor = principalMinor
        self.balanceMinor = balanceMinor
        self.disputedMinor = disputedMinor
        self.undisputedMinor = undisputedMinor
        self.currency = currency
        self.dueAt = dueAt
        self.createdAt = createdAt
        self.termsDays = termsDays
        self.gracePeriodDays = gracePeriodDays
        self.agingBucket = agingBucket
        self.dunningStep = dunningStep
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        invoiceId = try c.decodeIfPresent(String.self, forKey: .invoiceId) ?? "—"
        supplierId = try c.decodeIfPresent(String.self, forKey: .supplierId) ?? ""
        orderId = try c.decodeIfPresent(String.self, forKey: .orderId)
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? "—"
        principalMinor = try c.decodeIfPresent(Int64.self, forKey: .principalMinor)
        balanceMinor = try c.decodeIfPresent(Int64.self, forKey: .balanceMinor) ?? 0
        disputedMinor = try c.decodeIfPresent(Int64.self, forKey: .disputedMinor) ?? 0
        undisputedMinor = try c.decodeIfPresent(Int64.self, forKey: .undisputedMinor)
        currency = try c.decodeIfPresent(String.self, forKey: .currency)
        dueAt = try c.decodeIfPresent(String.self, forKey: .dueAt) ?? "—"
        createdAt = try c.decodeIfPresent(String.self, forKey: .createdAt)
        termsDays = try c.decodeIfPresent(Int.self, forKey: .termsDays)
        gracePeriodDays = try c.decodeIfPresent(Int.self, forKey: .gracePeriodDays)
        agingBucket = try c.decodeIfPresent(String.self, forKey: .agingBucket)
        dunningStep = try c.decodeIfPresent(Int.self, forKey: .dunningStep)
    }
}

struct ArInvoicesResponse: Decodable {
    let invoices: [ArInvoiceWire]?
}

struct ArPromiseWire: Decodable, Identifiable, Equatable, Sendable {
    let promiseId: String
    let supplierId: String
    let invoiceId: String?
    let quarterKey: String
    let dueAt: String
    let promisedMinor: Int64
    let status: String
    let approvedBy: String?
    let createdAt: String?

    var id: String { promiseId }

    enum CodingKeys: String, CodingKey {
        case promiseId = "promise_id"
        case supplierId = "supplier_id"
        case invoiceId = "invoice_id"
        case quarterKey = "quarter_key"
        case dueAt = "due_at"
        case promisedMinor = "promised_minor"
        case status
        case approvedBy = "approved_by"
        case createdAt = "created_at"
    }

    init(
        promiseId: String,
        supplierId: String = "",
        invoiceId: String? = nil,
        quarterKey: String = "",
        dueAt: String = "—",
        promisedMinor: Int64 = 0,
        status: String = "ACTIVE",
        approvedBy: String? = nil,
        createdAt: String? = nil
    ) {
        self.promiseId = promiseId
        self.supplierId = supplierId
        self.invoiceId = invoiceId
        self.quarterKey = quarterKey
        self.dueAt = dueAt
        self.promisedMinor = promisedMinor
        self.status = status
        self.approvedBy = approvedBy
        self.createdAt = createdAt
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        promiseId = try c.decodeIfPresent(String.self, forKey: .promiseId) ?? UUID().uuidString
        supplierId = try c.decodeIfPresent(String.self, forKey: .supplierId) ?? ""
        invoiceId = try c.decodeIfPresent(String.self, forKey: .invoiceId)
        quarterKey = try c.decodeIfPresent(String.self, forKey: .quarterKey) ?? ""
        dueAt = try c.decodeIfPresent(String.self, forKey: .dueAt) ?? "—"
        promisedMinor = try c.decodeIfPresent(Int64.self, forKey: .promisedMinor) ?? 0
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? "ACTIVE"
        approvedBy = try c.decodeIfPresent(String.self, forKey: .approvedBy)
        createdAt = try c.decodeIfPresent(String.self, forKey: .createdAt)
    }
}

struct ArPromisesResponse: Decodable, Equatable, Sendable {
    let promises: [ArPromiseWire]?
    let currentQuarterKey: String?
    let selfServePromiseUsed: Bool?
    let maxExtensionDays: Int?

    enum CodingKeys: String, CodingKey {
        case promises
        case currentQuarterKey = "current_quarter_key"
        case selfServePromiseUsed = "self_serve_promise_used"
        case maxExtensionDays = "max_extension_days"
    }
}

struct ArLedgerEntryWire: Decodable, Equatable, Sendable {
    let entryId: String?
    let id: String?
    let invoiceId: String?
    let supplierId: String?
    let entryType: String?
    let type: String?
    let amountMinor: Int64
    let balanceAfterMinor: Int64?
    let reference: String?
    let note: String?
    let createdAt: String?

    enum CodingKeys: String, CodingKey {
        case entryId = "entry_id"
        case id
        case invoiceId = "invoice_id"
        case supplierId = "supplier_id"
        case entryType = "entry_type"
        case type
        case amountMinor = "amount_minor"
        case balanceAfterMinor = "balance_after_minor"
        case reference
        case note
        case createdAt = "created_at"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        entryId = try c.decodeIfPresent(String.self, forKey: .entryId)
        id = try c.decodeIfPresent(String.self, forKey: .id)
        invoiceId = try c.decodeIfPresent(String.self, forKey: .invoiceId)
        supplierId = try c.decodeIfPresent(String.self, forKey: .supplierId)
        entryType = try c.decodeIfPresent(String.self, forKey: .entryType)
        type = try c.decodeIfPresent(String.self, forKey: .type)
        amountMinor = try c.decodeIfPresent(Int64.self, forKey: .amountMinor) ?? 0
        balanceAfterMinor = try c.decodeIfPresent(Int64.self, forKey: .balanceAfterMinor)
        reference = try c.decodeIfPresent(String.self, forKey: .reference)
        note = try c.decodeIfPresent(String.self, forKey: .note)
        createdAt = try c.decodeIfPresent(String.self, forKey: .createdAt)
    }
}

struct ArStatementWire: Decodable, Equatable, Sendable {
    let supplierId: String
    let currency: String
    let totalOpenMinor: Int64
    let totalDisputedMinor: Int64
    let totalUndisputedMinor: Int64
    let advanceBalanceMinor: Int64
    let currentQuarterKey: String
    let selfServePromiseUsed: Bool
    let maxExtensionDays: Int
    let invoices: [ArInvoiceWire]
    let promises: [ArPromiseWire]
    let ledgerEntries: [ArLedgerEntryWire]

    enum CodingKeys: String, CodingKey {
        case supplierId = "supplier_id"
        case currency
        case totalOpenMinor = "total_open_minor"
        case totalDisputedMinor = "total_disputed_minor"
        case totalUndisputedMinor = "total_undisputed_minor"
        case advanceBalanceMinor = "advance_balance_minor"
        case currentQuarterKey = "current_quarter_key"
        case selfServePromiseUsed = "self_serve_promise_used"
        case maxExtensionDays = "max_extension_days"
        case invoices
        case promises
        case ledgerEntries = "ledger_entries"
    }

    init(
        supplierId: String = "",
        currency: String = "UZS",
        totalOpenMinor: Int64 = 0,
        totalDisputedMinor: Int64 = 0,
        totalUndisputedMinor: Int64 = 0,
        advanceBalanceMinor: Int64 = 0,
        currentQuarterKey: String = "",
        selfServePromiseUsed: Bool = false,
        maxExtensionDays: Int = 14,
        invoices: [ArInvoiceWire] = [],
        promises: [ArPromiseWire] = [],
        ledgerEntries: [ArLedgerEntryWire] = []
    ) {
        self.supplierId = supplierId
        self.currency = currency
        self.totalOpenMinor = totalOpenMinor
        self.totalDisputedMinor = totalDisputedMinor
        self.totalUndisputedMinor = totalUndisputedMinor
        self.advanceBalanceMinor = advanceBalanceMinor
        self.currentQuarterKey = currentQuarterKey
        self.selfServePromiseUsed = selfServePromiseUsed
        self.maxExtensionDays = maxExtensionDays
        self.invoices = invoices
        self.promises = promises
        self.ledgerEntries = ledgerEntries
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        supplierId = try c.decodeIfPresent(String.self, forKey: .supplierId) ?? ""
        currency = try c.decodeIfPresent(String.self, forKey: .currency) ?? "UZS"
        totalOpenMinor = try c.decodeIfPresent(Int64.self, forKey: .totalOpenMinor) ?? 0
        totalDisputedMinor = try c.decodeIfPresent(Int64.self, forKey: .totalDisputedMinor) ?? 0
        totalUndisputedMinor = try c.decodeIfPresent(Int64.self, forKey: .totalUndisputedMinor) ?? 0
        advanceBalanceMinor = try c.decodeIfPresent(Int64.self, forKey: .advanceBalanceMinor) ?? 0
        currentQuarterKey = try c.decodeIfPresent(String.self, forKey: .currentQuarterKey) ?? ""
        selfServePromiseUsed = try c.decodeIfPresent(Bool.self, forKey: .selfServePromiseUsed) ?? false
        maxExtensionDays = try c.decodeIfPresent(Int.self, forKey: .maxExtensionDays) ?? 14
        invoices = try c.decodeIfPresent([ArInvoiceWire].self, forKey: .invoices) ?? []
        promises = try c.decodeIfPresent([ArPromiseWire].self, forKey: .promises) ?? []
        ledgerEntries = try c.decodeIfPresent([ArLedgerEntryWire].self, forKey: .ledgerEntries) ?? []
    }
}

struct PayArFifoRequest: Encodable, Equatable, Sendable {
    let supplierId: String
    let amountMinor: Int64
    let currency: String
    let paymentMethod: String
    let paymentReference: String?
    let earmarkInvoiceId: String?

    enum CodingKeys: String, CodingKey {
        case supplierId = "supplier_id"
        case amountMinor = "amount_minor"
        case currency
        case paymentMethod = "payment_method"
        case paymentReference = "payment_reference"
        case earmarkInvoiceId = "earmark_invoice_id"
    }
}

struct PayArFifoAllocationWire: Decodable, Identifiable, Equatable, Sendable {
    let invoiceId: String
    let orderId: String?
    let appliedMinor: Int64
    let remainingBalanceMinor: Int64
    let status: String

    var id: String { invoiceId }

    enum CodingKeys: String, CodingKey {
        case invoiceId = "invoice_id"
        case orderId = "order_id"
        case appliedMinor = "applied_minor"
        case remainingBalanceMinor = "remaining_balance_minor"
        case status
    }

    init(
        invoiceId: String,
        orderId: String? = nil,
        appliedMinor: Int64 = 0,
        remainingBalanceMinor: Int64 = 0,
        status: String = ""
    ) {
        self.invoiceId = invoiceId
        self.orderId = orderId
        self.appliedMinor = appliedMinor
        self.remainingBalanceMinor = remainingBalanceMinor
        self.status = status
    }
}

struct PayArFifoResponse: Decodable, Equatable, Sendable {
    let paidMinor: Int64
    let advanceAddedMinor: Int64
    let advanceBalanceMinor: Int64
    let allocations: [PayArFifoAllocationWire]

    enum CodingKeys: String, CodingKey {
        case paidMinor = "paid_minor"
        case advanceAddedMinor = "advance_added_minor"
        case advanceBalanceMinor = "advance_balance_minor"
        case allocations
    }

    init(
        paidMinor: Int64 = 0,
        advanceAddedMinor: Int64 = 0,
        advanceBalanceMinor: Int64 = 0,
        allocations: [PayArFifoAllocationWire] = []
    ) {
        self.paidMinor = paidMinor
        self.advanceAddedMinor = advanceAddedMinor
        self.advanceBalanceMinor = advanceBalanceMinor
        self.allocations = allocations
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        paidMinor = try c.decodeIfPresent(Int64.self, forKey: .paidMinor) ?? 0
        advanceAddedMinor = try c.decodeIfPresent(Int64.self, forKey: .advanceAddedMinor) ?? 0
        advanceBalanceMinor = try c.decodeIfPresent(Int64.self, forKey: .advanceBalanceMinor) ?? 0
        allocations = try c.decodeIfPresent([PayArFifoAllocationWire].self, forKey: .allocations) ?? []
    }
}

struct PayArInvoiceRequest: Encodable, Equatable, Sendable {
    let amountMinor: Int64
    let paymentMethod: String
    let paymentReference: String?

    enum CodingKeys: String, CodingKey {
        case amountMinor = "amount_minor"
        case paymentMethod = "payment_method"
        case paymentReference = "payment_reference"
    }
}

struct CreateArPromiseRequest: Encodable, Equatable, Sendable {
    let supplierId: String
    let invoiceId: String?
    let extensionDays: Int

    enum CodingKeys: String, CodingKey {
        case supplierId = "supplier_id"
        case invoiceId = "invoice_id"
        case extensionDays = "extension_days"
    }
}

struct QuarterlyPromiseLimitError: Equatable, Sendable {
    let code: String
    let quarterKey: String?
    let message: String?
}

enum RetailerW3ContractHelper {
    static func sortInvoicesOldestFirst(_ invoices: [ArInvoiceWire]) -> [ArInvoiceWire] {
        invoices.sorted { lhs, rhs in
            let lDue = sortableDateKey(lhs.dueAt)
            let rDue = sortableDateKey(rhs.dueAt)
            if lDue != rDue { return lDue < rDue }
            let lCreated = sortableDateKey(lhs.createdAt)
            let rCreated = sortableDateKey(rhs.createdAt)
            if lCreated != rCreated { return lCreated < rCreated }
            return lhs.invoiceId < rhs.invoiceId
        }
    }

    private static func sortableDateKey(_ raw: String?) -> String {
        let trimmed = raw?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if trimmed.isEmpty || trimmed == "—" {
            return "9999-99-99T99:99:99"
        }
        return trimmed
    }

    static func parseQuarterlyPromiseLimitError(from error: Error) -> QuarterlyPromiseLimitError? {
        guard let apiError = error as? APIError else { return nil }
        switch apiError {
        case .problemDetail(let problem):
            if let raw = problem.rawBody, let data = raw.data(using: .utf8),
               let dict = try? JSONSerialization.jsonObject(with: data) as? [String: Any] {
                let code = (dict["code"] as? String) ?? (dict["error"] as? String) ?? problem.code ?? ""
                let quarter = dict["quarter_key"] as? String
                let msg = (dict["message"] as? String) ?? (dict["detail"] as? String) ?? problem.detail
                if code == "quarterly_promise_limit_reached" || problem.status == 409 {
                    return QuarterlyPromiseLimitError(
                        code: code.isEmpty ? "quarterly_promise_limit_reached" : code,
                        quarterKey: quarter,
                        message: msg
                    )
                }
            }
            if problem.code == "quarterly_promise_limit_reached" || problem.error == "quarterly_promise_limit_reached" || problem.status == 409 {
                return QuarterlyPromiseLimitError(
                    code: problem.code ?? problem.error ?? "quarterly_promise_limit_reached",
                    quarterKey: nil,
                    message: problem.detail ?? problem.title
                )
            }
            return nil
        case .serverError(let statusCode, let body):
            if let data = body.data(using: .utf8),
               let dict = try? JSONSerialization.jsonObject(with: data) as? [String: Any] {
                let code = (dict["code"] as? String) ?? (dict["error"] as? String) ?? ""
                let quarter = dict["quarter_key"] as? String
                let msg = (dict["message"] as? String) ?? (dict["detail"] as? String) ?? body
                if code == "quarterly_promise_limit_reached" || statusCode == 409 {
                    return QuarterlyPromiseLimitError(
                        code: code.isEmpty ? "quarterly_promise_limit_reached" : code,
                        quarterKey: quarter,
                        message: msg
                    )
                }
            } else if statusCode == 409 || body.contains("quarterly_promise_limit_reached") {
                return QuarterlyPromiseLimitError(
                    code: "quarterly_promise_limit_reached",
                    quarterKey: nil,
                    message: body
                )
            }
            return nil
        default:
            return nil
        }
    }

    static func parseArErrorMessage(from error: Error, fallback: String) -> String {
        guard let apiError = error as? APIError else {
            return error.localizedDescription.isEmpty ? fallback : error.localizedDescription
        }
        switch apiError {
        case .problemDetail(let problem):
            return problem.detail ?? problem.title ?? fallback
        case .serverError(_, let body):
            if let data = body.data(using: .utf8),
               let dict = try? JSONSerialization.jsonObject(with: data) as? [String: Any] {
                if let msg = dict["message"] as? String, !msg.isEmpty { return msg }
                if let detail = dict["detail"] as? String, !detail.isEmpty { return detail }
                if let err = dict["error"] as? String, !err.isEmpty { return err }
            }
            return body.isEmpty ? fallback : body
        default:
            return apiError.localizedDescription.isEmpty ? fallback : apiError.localizedDescription
        }
    }

    static func normalizedINNOrNil(_ raw: String) -> String? {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        guard trimmed.count == 9, trimmed.allSatisfy({ $0 >= "0" && $0 <= "9" }) else {
            return nil
        }
        return trimmed
    }

    static func parseJoinRequestConflict(from error: Error) -> RetailerJoinRequestConflict? {
        guard let apiError = error as? APIError else { return nil }
        switch apiError {
        case .problemDetail(let problem):
            if let raw = problem.rawBody, let data = raw.data(using: .utf8),
               let conflict = decodeJoinRequestConflict(data: data, statusCode: problem.status) {
                return conflict
            }
            if problem.status == 409,
               (problem.error == "join_request_pending" || problem.code == "join_request_pending") {
                return RetailerJoinRequestConflict(
                    error: "join_request_pending",
                    requestId: nil,
                    targetTenantId: nil,
                    legalName: nil,
                    normalizedInn: nil,
                    message: problem.detail ?? problem.title
                )
            }
            return nil
        case .serverError(let statusCode, let body):
            if let data = body.data(using: .utf8),
               let conflict = decodeJoinRequestConflict(data: data, statusCode: statusCode) {
                return conflict
            }
            if statusCode == 409 && body.contains("join_request_pending") {
                return RetailerJoinRequestConflict(
                    error: "join_request_pending",
                    requestId: nil,
                    targetTenantId: nil,
                    legalName: nil,
                    normalizedInn: nil,
                    message: body
                )
            }
            return nil
        default:
            return nil
        }
    }

    private static func decodeJoinRequestConflict(data: Data, statusCode: Int) -> RetailerJoinRequestConflict? {
        guard let dict = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            return nil
        }
        let err = (dict["error"] as? String) ?? (dict["code"] as? String) ?? ""
        guard err == "join_request_pending" || (statusCode == 409 && dict["request_id"] != nil) else {
            return nil
        }
        if let decoded = try? JSONDecoder().decode(RetailerJoinRequestConflict.self, from: data) {
            return decoded
        }
        return RetailerJoinRequestConflict(
            error: err.isEmpty ? "join_request_pending" : err,
            requestId: dict["request_id"] as? String,
            targetTenantId: dict["target_tenant_id"] as? String,
            legalName: dict["legal_name"] as? String,
            normalizedInn: dict["normalized_inn"] as? String,
            message: (dict["message"] as? String) ?? (dict["detail"] as? String)
        )
    }
}

struct RetailerINNLookupRequest: Encodable, Sendable {
    let inn: String
    let tenantType: String

    enum CodingKeys: String, CodingKey {
        case inn
        case tenantType = "tenant_type"
    }
}

struct RetailerINNLookupResponse: Decodable, Equatable, Sendable {
    let status: String
    let normalizedInn: String?
    let entityType: String?
    let legalName: String?
    let address: String?
    let registryState: String?
    let cached: Bool?
    let existingTenantId: String?
    let existingTenantType: String?
    let existingLegalName: String?
    let existingJoinEligible: Bool?

    enum CodingKeys: String, CodingKey {
        case status
        case normalizedInn = "normalized_inn"
        case entityType = "entity_type"
        case legalName = "legal_name"
        case address
        case registryState = "registry_state"
        case cached
        case existingTenantId = "existing_tenant_id"
        case existingTenantType = "existing_tenant_type"
        case existingLegalName = "existing_legal_name"
        case existingJoinEligible = "existing_join_eligible"
    }

    var isVerifiedFound: Bool { status.uppercased() == "FOUND" }
    var isUnavailable: Bool { status.uppercased() == "UNAVAILABLE" }
    var isInvalidOrMissing: Bool {
        let upper = status.uppercased()
        return upper == "INVALID" || upper == "NOT_FOUND"
    }
}

struct RetailerJoinRequestConflict: Decodable, Equatable, Sendable {
    let error: String?
    let requestId: String?
    let targetTenantId: String?
    let legalName: String?
    let normalizedInn: String?
    let message: String?

    enum CodingKeys: String, CodingKey {
        case error
        case requestId = "request_id"
        case targetTenantId = "target_tenant_id"
        case legalName = "legal_name"
        case normalizedInn = "normalized_inn"
        case message
    }
}

struct RetailerTaxIdentityRequest: Encodable, Sendable {
    let taxId: String
    let informalShop: Bool
    let legalName: String?
    let roleSought: String?

    enum CodingKeys: String, CodingKey {
        case taxId = "tax_id"
        case informalShop = "informal_shop"
        case legalName = "legal_name"
        case roleSought = "role_sought"
    }
}

struct RetailerTaxIdentityResponse: Decodable, Equatable, Sendable {
    let id: String?
    let taxId: String?
    let normalizedTaxId: String?
    let legalEntityStatus: String?
    let innVerifiedAt: String?

    enum CodingKeys: String, CodingKey {
        case id
        case taxId = "tax_id"
        case normalizedTaxId = "normalized_tax_id"
        case legalEntityStatus = "legal_entity_status"
        case innVerifiedAt = "inn_verified_at"
    }
}

struct MigrationOpeningInvoiceWire: Codable, Identifiable, Equatable, Sendable {
    let invoiceId: String
    let supplierId: String
    let supplierName: String?
    let retailerId: String?
    let orderId: String?
    let externalId: String?
    let principalMinor: Int64
    let balanceMinor: Int64
    let currency: String?
    let status: String
    let source: String?
    let dunningFrozen: Bool
    let dueAt: String?
    let acknowledgedAt: String?
    let disputedAt: String?
    let disputeReason: String?
    let updatedAt: String?

    var id: String { invoiceId }
    var effectiveCurrency: String { currency ?? "UZS" }

    var isPendingConfirmation: Bool {
        let upper = status.uppercased()
        return upper != "ACKNOWLEDGED" && upper != "ACCEPTED" && upper != "DISPUTED"
    }

    var isDisputed: Bool { status.uppercased() == "DISPUTED" }
    var isAcknowledged: Bool {
        let upper = status.uppercased()
        return upper == "ACKNOWLEDGED" || upper == "ACCEPTED"
    }

    enum CodingKeys: String, CodingKey {
        case invoiceId = "invoice_id"
        case supplierId = "supplier_id"
        case supplierName = "supplier_name"
        case retailerId = "retailer_id"
        case orderId = "order_id"
        case externalId = "external_id"
        case principalMinor = "principal_minor"
        case balanceMinor = "balance_minor"
        case currency
        case status
        case source
        case dunningFrozen = "dunning_frozen"
        case dueAt = "due_at"
        case acknowledgedAt = "acknowledged_at"
        case disputedAt = "disputed_at"
        case disputeReason = "dispute_reason"
        case updatedAt = "updated_at"
    }

    init(
        invoiceId: String,
        supplierId: String,
        supplierName: String? = nil,
        retailerId: String? = nil,
        orderId: String? = nil,
        externalId: String? = nil,
        principalMinor: Int64,
        balanceMinor: Int64,
        currency: String? = "UZS",
        status: String = "PENDING_ACK",
        source: String? = "MIGRATION",
        dunningFrozen: Bool = true,
        dueAt: String? = nil,
        acknowledgedAt: String? = nil,
        disputedAt: String? = nil,
        disputeReason: String? = nil,
        updatedAt: String? = nil
    ) {
        self.invoiceId = invoiceId
        self.supplierId = supplierId
        self.supplierName = supplierName
        self.retailerId = retailerId
        self.orderId = orderId
        self.externalId = externalId
        self.principalMinor = principalMinor
        self.balanceMinor = balanceMinor
        self.currency = currency
        self.status = status
        self.source = source
        self.dunningFrozen = dunningFrozen
        self.dueAt = dueAt
        self.acknowledgedAt = acknowledgedAt
        self.disputedAt = disputedAt
        self.disputeReason = disputeReason
        self.updatedAt = updatedAt
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        invoiceId = try c.decode(String.self, forKey: .invoiceId)
        supplierId = try c.decode(String.self, forKey: .supplierId)
        supplierName = try c.decodeIfPresent(String.self, forKey: .supplierName)
        retailerId = try c.decodeIfPresent(String.self, forKey: .retailerId)
        orderId = try c.decodeIfPresent(String.self, forKey: .orderId)
        externalId = try c.decodeIfPresent(String.self, forKey: .externalId)
        principalMinor = try c.decodeIfPresent(Int64.self, forKey: .principalMinor) ?? 0
        balanceMinor = try c.decodeIfPresent(Int64.self, forKey: .balanceMinor) ?? 0
        currency = try c.decodeIfPresent(String.self, forKey: .currency)
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? "PENDING_ACK"
        source = try c.decodeIfPresent(String.self, forKey: .source)
        dunningFrozen = try c.decodeIfPresent(Bool.self, forKey: .dunningFrozen) ?? true
        dueAt = try c.decodeIfPresent(String.self, forKey: .dueAt)
        acknowledgedAt = try c.decodeIfPresent(String.self, forKey: .acknowledgedAt)
        disputedAt = try c.decodeIfPresent(String.self, forKey: .disputedAt)
        disputeReason = try c.decodeIfPresent(String.self, forKey: .disputeReason)
        updatedAt = try c.decodeIfPresent(String.self, forKey: .updatedAt)
    }
}

struct MigrationOpeningBalancesResponse: Decodable, Sendable {
    let retailerId: String?
    let invoices: [MigrationOpeningInvoiceWire]

    enum CodingKeys: String, CodingKey {
        case retailerId = "retailer_id"
        case invoices
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        retailerId = try c.decodeIfPresent(String.self, forKey: .retailerId)
        invoices = try c.decodeIfPresent([MigrationOpeningInvoiceWire].self, forKey: .invoices) ?? []
    }
}

struct MigrationOpeningInvoiceMutationResponse: Decodable, Sendable {
    let retailerId: String?
    let invoice: MigrationOpeningInvoiceWire

    enum CodingKeys: String, CodingKey {
        case retailerId = "retailer_id"
        case invoice
    }
}

struct DisputeOpeningBalanceRequest: Encodable, Sendable {
    let reason: String
}

struct RetailerJoinRequestItem: Codable, Identifiable, Equatable, Sendable {
    let requestId: String
    let homeCell: String?
    let targetTenantType: String?
    let targetTenantId: String?
    let normalizedInn: String?
    let requesterPhone: String
    let requesterName: String?
    let requesterEmail: String?
    let roleSought: String
    let status: String
    let decidedBy: String?
    let decidedAt: String?
    let decisionReason: String?
    let createdAt: String?

    var id: String { requestId }

    enum CodingKeys: String, CodingKey {
        case requestId = "request_id"
        case homeCell = "home_cell"
        case targetTenantType = "target_tenant_type"
        case targetTenantId = "target_tenant_id"
        case normalizedInn = "normalized_inn"
        case requesterPhone = "requester_phone"
        case requesterName = "requester_name"
        case requesterEmail = "requester_email"
        case roleSought = "role_sought"
        case status
        case decidedBy = "decided_by"
        case decidedAt = "decided_at"
        case decisionReason = "decision_reason"
        case createdAt = "created_at"
    }
}

struct RetailerJoinRequestsResponse: Decodable, Sendable {
    let retailerId: String?
    let requests: [RetailerJoinRequestItem]

    enum CodingKeys: String, CodingKey {
        case retailerId = "retailer_id"
        case requests
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        retailerId = try c.decodeIfPresent(String.self, forKey: .retailerId)
        requests = try c.decodeIfPresent([RetailerJoinRequestItem].self, forKey: .requests) ?? []
    }
}

struct RetailerJoinRequestDecisionRequest: Encodable, Sendable {
    let reason: String?
}

struct RetailerJoinRequestDecisionResponse: Decodable, Sendable {
    let retailerId: String?
    let request: RetailerJoinRequestItem

    enum CodingKeys: String, CodingKey {
        case retailerId = "retailer_id"
        case request
    }
}
