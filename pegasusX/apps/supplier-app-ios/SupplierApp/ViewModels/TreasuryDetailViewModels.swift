import Foundation
import Observation
#if canImport(UIKit)
import UIKit
#endif

@Observable
@MainActor
final class CashReconciliationsViewModel {
    var rows: [CashReconciliationRow] = []
    var loading = true
    private var loadGeneration = 0
    var error: String?
    var busyId: String?

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !rows.isEmpty
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let resp = try await SupplierOperationsService.cashReconciliations()
            guard generation == loadGeneration else { return }
            rows = resp.reconciliations
            guard generation == loadGeneration else { return }
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

    func accept(_ id: String) async {
        busyId = id
        defer { busyId = nil }
        do {
            try await SupplierOperationsService.acceptCashReconciliation(
                id: id,
                idempotencyKey: "supplier-cash-recon-accept:\(id)"
            )
            PegasusHaptics.confirm()
            await load()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: !rows.isEmpty, error: &self.error)
            PegasusHaptics.error()
        }
    }
}

@Observable
@MainActor
final class ChargebacksViewModel {
    var orderId = ""
    var retailerId = ""
    var gateway = "ADYEN"
    var amount = ""
    var currency = packCurrency(MarketPackStore.pack)
    var sessionId = ""
    var busy = false
    var chargebackMessage: String?
    var reversalMessage: String?
    var error: String?

    func handleReconnect() {
        if busy {
            busy = false
            error = L10n.format("mobile_supplier.chrome.connection_restored_chargeback")
        }
    }

    func submitChargeback() async {
        guard let amountMinor = Int64(amount) else {
            error = L10n.format("mobile_supplier.chrome.amount_must_be_number")
            PegasusHaptics.warning()
            return
        }
        busy = true
        error = nil
        chargebackMessage = nil
        defer { busy = false }
        do {
            let response = try await SupplierService.recordChargeback(
                PaymentChargebackRequest(
                    orderId: orderId,
                    retailerId: retailerId,
                    gateway: gateway,
                    amount: amountMinor,
                    currency: currency
                ),
                idempotencyKey: SupplierIdempotencyKeys.chargeback(orderId: orderId, reason: orderId)
            )
            chargebackMessage = L10n.format("mobile_supplier.chrome.chargeback_recorded", response.status)
            orderId = ""
            retailerId = ""
            amount = ""
            PegasusHaptics.confirm()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: false, error: &self.error)
            PegasusHaptics.error()
        }
    }

    func submitReversal() async {
        busy = true
        error = nil
        reversalMessage = nil
        defer { busy = false }
        do {
            let response = try await SupplierService.recordChargebackReversal(
                PaymentChargebackReversalRequest(sessionId: sessionId),
                idempotencyKey: SupplierIdempotencyKeys.chargebackReversal(chargebackId: sessionId, reason: sessionId)
            )
            reversalMessage = L10n.format("mobile_supplier.chrome.reversal_recorded", response.status)
            sessionId = ""
            PegasusHaptics.confirm()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: false, error: &self.error)
            PegasusHaptics.error()
        }
    }
}

@Observable
@MainActor
final class ClaimChargebacksViewModel {
    var items: [PaymentLedgerEntry] = []
    var orderFilter = ""
    var loading = true
    private var loadGeneration = 0
    var error: String?

    var total: Int64? {
        var sum: Int64 = 0
        for item in items {
            guard let amount = item.amountMinor else { return nil }
            sum += amount
        }
        return sum
    }

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !items.isEmpty || (error == nil && !loading)
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let resp = try await SupplierOperationsService.listClaimChargebacks(
                limit: 100,
                orderId: orderFilter.trimmingCharacters(in: .whitespacesAndNewlines)
            )
            guard generation == loadGeneration else { return }
            items = resp.items
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
}

@Observable
@MainActor
final class ComplianceAuditViewModel {
    var loading = true
    private var loadGeneration = 0
    var error: String?
    var summary: ComplianceSummary?
    var forceCompletes: [ComplianceForceCompleteRow] = []
    var openFiscal: [ComplianceFiscalOpenRow] = []

    private let loadDashboardFn: @Sendable () async throws -> ComplianceDashboardResponse
    private let fetchReceiptFn: @Sendable (String) async throws -> OrderReceiptMeta
    private let openURLFn: @MainActor (URL) -> Void

    init(
        loadDashboard: @escaping @Sendable () async throws -> ComplianceDashboardResponse = {
            try await SupplierOperationsService.complianceDashboard()
        },
        fetchReceipt: @escaping @Sendable (String) async throws -> OrderReceiptMeta = { orderId in
            try await APIClient.shared.get(
                "v1/supplier/orders/\(orderId)/receipt",
                query: ["format": "json"]
            )
        },
        openURL: @escaping @MainActor (URL) -> Void = { url in
            #if canImport(UIKit)
            UIApplication.shared.open(url)
            #endif
        }
    ) {
        self.loadDashboardFn = loadDashboard
        self.fetchReceiptFn = fetchReceipt
        self.openURLFn = openURL
    }

    func openReceipt(orderId: String) async {
        do {
            let meta = try await fetchReceiptFn(orderId)
            let raw = [meta.htmlUrl, meta.qrUrl, meta.pdfUrl]
                .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
                .first { !$0.isEmpty }
            if let raw, let url = URL(string: raw) {
                PegasusHaptics.lightTap()
                openURLFn(url)
            }
        } catch {
            if ReceiptOpen.shouldStayQuiet(error) { return }
            let message = error.localizedDescription
            if summary == nil {
                self.error = message
            } else {
                SupplierStatusCenter.shared.notice = message
            }
            PegasusHaptics.error()
        }
    }

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = summary != nil || (error == nil && !loading)
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let response = try await loadDashboardFn()
            guard generation == loadGeneration else { return }
            summary = response.summary
            openFiscal = response.openFiscal
            forceCompletes = response.forceCompletes
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
}

@Observable
@MainActor
final class CreditNotesListViewModel {
    var rows: [CreditNoteRow] = []
    var loading = true
    private var loadGeneration = 0
    var error: String?
    var busyId: String?

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !rows.isEmpty
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let resp = try await SupplierOperationsService.creditNotes()
            guard generation == loadGeneration else { return }
            rows = resp.creditNotes
            guard generation == loadGeneration else { return }
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

    func issue(_ id: String) async {
        busyId = id
        defer { busyId = nil }
        do {
            try await SupplierOperationsService.issueCreditNote(
                id: id,
                idempotencyKey: "supplier-credit-note-issue:\(id)"
            )
            PegasusHaptics.confirm()
            await load()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: !rows.isEmpty, error: &self.error)
            PegasusHaptics.error()
        }
    }
}

@Observable
@MainActor
final class CreditProfilesViewModel {
    var rows: [CreditProfileRow] = []
    var loading = true
    private var loadGeneration = 0
    var error: String?
    var busyId: String?

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !rows.isEmpty
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let loadedRows = try await SupplierOperationsService.creditProfiles()
            guard generation == loadGeneration else { return }
            rows = loadedRows
            guard generation == loadGeneration else { return }
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

    func setStatus(_ row: CreditProfileRow, status: String) async {
        busyId = row.retailerId
        defer { busyId = nil }
        do {
            guard let creditLimitMinor = row.creditLimitMinor else {
                self.error = L10n.format("mobile_supplier.chrome.credit_limit_missing")
                return
            }
            try await SupplierOperationsService.patchRetailerCreditProfile(
                RetailerCreditProfilePatchRequest(
                    retailerId: row.retailerId,
                    creditLimitMinor: creditLimitMinor,
                    status: status,
                    reason: "collections_desk"
                ),
                idempotencyKey: "supplier-credit-profile:\(status):\(row.retailerId)"
            )
            PegasusHaptics.confirm()
            await load()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: !rows.isEmpty, error: &self.error)
            PegasusHaptics.error()
        }
    }
}

@Observable
@MainActor
final class LedgerViewModel {
    var rows: [PaymentLedgerEntry] = []
    var loading = true
    private var loadGeneration = 0
    var error: String?

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !rows.isEmpty
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let resp = try await SupplierOperationsService.paymentLedger()
            guard generation == loadGeneration else { return }
            rows = resp.items
            guard generation == loadGeneration else { return }
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
}

@Observable
@MainActor
final class PaymentsViewModel {
    var loading = true
    private var loadGeneration = 0
    var error: String?
    var authority: SettlementAuthorityResponse?
    var mismatches: [ReconciliationMismatchRow] = []

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = authority != nil || (error == nil && !loading)
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            async let authorityCall = SupplierOperationsService.paymentSettlementAuthority()
            async let mismatchCall = SupplierOperationsService.paymentReconciliationMismatches()
            let loadedAuthority = try await authorityCall
            let loadedMismatches = try await mismatchCall
            guard generation == loadGeneration else { return }
            authority = loadedAuthority
            mismatches = loadedMismatches.items
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
}

@Observable
@MainActor
final class PayoutsViewModel {
    var rail: PayoutRailInfo?
    var policy: SupplierPayoutPolicy?
    var policyKnown = false
    var policyError: String?
    var draftMode = "HQ_SUPPLIER"
    var policyReason = ""
    var batches: [PayoutBatch] = []
    var loading = true
    private var loadGeneration = 0
    var error: String?
    var status: String?
    var periodStart = ""
    var periodEnd = ""
    var busy = false

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = rail != nil || !batches.isEmpty || policy != nil
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            async let railTask = SupplierOperationsService.payoutRail()
            async let listTask = SupplierOperationsService.payoutBatches()
            async let policyTask = SupplierOperationsService.payoutPolicy()
            let loadedRail = try await railTask
            let loadedBatches = try await listTask
            let previousPolicy = policyKnown ? policy : nil
            let policyOutcome: PulseHonesty.ObjectResult<SupplierPayoutPolicy>
            do {
                let loadedPolicy = try await policyTask
                policyOutcome = PulseHonesty.applyObject(
                    ok: true,
                    incoming: loadedPolicy,
                    previous: previousPolicy,
                    failure: "payout_policy_failed"
                )
            } catch {
                policyOutcome = PulseHonesty.applyObject(
                    ok: false,
                    incoming: nil,
                    previous: previousPolicy,
                    failure: error.localizedDescription
                )
            }
            guard generation == loadGeneration else { return }
            rail = loadedRail
            batches = loadedBatches
            if policyOutcome.error == nil, let loadedPolicy = policyOutcome.value {
                policy = loadedPolicy
                policyKnown = true
                if !loadedPolicy.payoutMode.isEmpty {
                    draftMode = loadedPolicy.payoutMode
                }
            } else if let kept = policyOutcome.value {
                policy = kept
                policyKnown = true
            } else {
                policyKnown = false
            }
            policyError = policyOutcome.error
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

    func savePolicy() async {
        busy = true
        defer { busy = false }
        do {
            let next = try await SupplierOperationsService.patchPayoutPolicy(
                SupplierPayoutPolicyPatch(payoutMode: draftMode, feePolicyVersion: nil, reason: policyReason)
            )
            policy = next
            draftMode = next.payoutMode.isEmpty ? draftMode : next.payoutMode
            policyReason = ""
            status = L10n.format("mobile_supplier.chrome.payout_mode_saved")
            PegasusHaptics.confirm()
        } catch {
            status = error.localizedDescription
            SupplierActionFailure.record(underlying: error, hasContent: true, error: &self.error)
            PegasusHaptics.error()
        }
    }

    func generate() async {
        busy = true
        defer { busy = false }
        do {
            let scope = SupplierIdempotencyKeys.supplierScopeId()
            _ = try await SupplierOperationsService.generatePayoutBatch(
                PayoutBatchGenerateRequest(periodStart: periodStart, periodEnd: periodEnd),
                idempotencyKey: SupplierIdempotencyKeys.payoutGenerate(
                    scopeId: scope,
                    periodStart: periodStart,
                    periodEnd: periodEnd
                )
            )
            status = L10n.format("mobile_supplier.chrome.batch_generated")
            PegasusHaptics.confirm()
            await load(silent: true)
        } catch {
            status = error.localizedDescription
            SupplierActionFailure.record(underlying: error, hasContent: true, error: &self.error)
            PegasusHaptics.error()
        }
    }

    func exportCSV(_ batchId: String) async {
        busy = true
        defer { busy = false }
        do {
            let csv = try await SupplierOperationsService.exportPayoutBatch(
                batchId,
                idempotencyKey: SupplierIdempotencyKeys.payoutExport(batchId: batchId)
            )
            #if canImport(UIKit)
            UIPasteboard.general.string = csv
            status = L10n.format("mobile_supplier.chrome.csv_copied", "\(csv.count)")
            #else
            status = L10n.format("mobile_supplier.chrome.csv_exported", "\(csv.count)")
            #endif
            PegasusHaptics.confirm()
            await load(silent: true)
        } catch {
            status = error.localizedDescription
            SupplierActionFailure.record(underlying: error, hasContent: true, error: &self.error)
            PegasusHaptics.error()
        }
    }

    func markPaid(_ batchId: String) async {
        busy = true
        defer { busy = false }
        do {
            let resp = try await SupplierOperationsService.markPayoutBatchPaid(
                batchId,
                idempotencyKey: SupplierIdempotencyKeys.payoutMarkPaid(batchId: batchId)
            )
            status = resp.message.isEmpty ? L10n.format("mobile_supplier.chrome.marked_paid") : resp.message
            PegasusHaptics.confirm()
            await load(silent: true)
        } catch {
            status = error.localizedDescription
            SupplierActionFailure.record(underlying: error, hasContent: true, error: &self.error)
            PegasusHaptics.error()
        }
    }

    func dispatchLive(_ batchId: String) async {
        busy = true
        defer { busy = false }
        do {
            let resp = try await SupplierOperationsService.dispatchPayoutBatch(
                batchId,
                live: true,
                idempotencyKey: SupplierIdempotencyKeys.payoutDispatch(batchId: batchId, live: true)
            )
            if resp.code == "no_live_rail" || resp.error == "no_live_rail" {
                status = resp.message.isEmpty ? L10n.format("mobile_supplier.chrome.no_live_rail_hint") : resp.message
            } else {
                status = resp.message.isEmpty ? L10n.format("mobile_supplier.chrome.dispatch_attempted") : resp.message
            }
            PegasusHaptics.confirm()
            await load(silent: true)
        } catch {
            status = error.localizedDescription
            SupplierActionFailure.record(underlying: error, hasContent: true, error: &self.error)
            PegasusHaptics.error()
        }
    }
}

@Observable
@MainActor
final class ReconciliationViewModel {
    var loading = true
    private var loadGeneration = 0
    var hasLoaded = false
    var error: String?
    var netMinor: Int64?
    var currency = packCurrency(MarketPackStore.pack)
    var mismatchCount = 0

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = hasLoaded
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            async let authority = SupplierOperationsService.paymentSettlementAuthority()
            async let mismatches = SupplierOperationsService.paymentReconciliationMismatches()
            let authorityResp = try await authority
            let mismatchResp = try await mismatches
            guard generation == loadGeneration else { return }
            if let primary = authorityResp.totalsByCurrency.first {
                currency = primary.currency
                netMinor = primary.amountMinorTotal
            } else {
                netMinor = nil
            }
            mismatchCount = mismatchResp.items.count
            hasLoaded = true
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
}

@Observable
@MainActor
final class RoutePerformanceListViewModel {
    var rows: [RoutePerformanceRow] = []
    var loading = true
    private var loadGeneration = 0
    var error: String?

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !rows.isEmpty
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        let outcome: PulseHonesty.Result<RoutePerformanceRow>
        do {
            let loaded = try await SupplierOperationsService.routePerformance().routes
            guard generation == loadGeneration else { return }
            outcome = PulseHonesty.apply(ok: true, incoming: loaded, previous: rows)
        } catch {
            guard generation == loadGeneration else { return }
            outcome = PulseHonesty.apply(ok: false, incoming: nil, previous: rows, failure: error.localizedDescription)
        }
        rows = outcome.events
        if let failure = outcome.error {
            SupplierScreenLoad.fail(
                silent: silent,
                hasContent: hadContent,
                underlyingMessage: failure,
                error: &error,
                noun: "data"
            )
        } else {
            SupplierScreenLoad.succeed(error: &error)
        }
        if generation == loadGeneration {
            loading = false
        }
    }
}
