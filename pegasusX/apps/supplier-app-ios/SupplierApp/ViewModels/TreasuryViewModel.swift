import Foundation

@Observable
@MainActor
final class TreasuryViewModel {
    var earnings: SupplierEarnings?
    var ledgerCount = 0
    var mismatchCount = 0
    var settlementRows = 0
    var monthLabel = "—"
    var loading = true
    var error: String?

    func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { loading = false }
        do {
            async let earningsTask = SupplierService.earnings()
            async let ledgerTask = SupplierOperationsService.paymentLedger()
            async let authorityTask = SupplierOperationsService.paymentSettlementAuthority()
            async let mismatchTask = SupplierOperationsService.paymentReconciliationMismatches()

            let loadedEarnings = try await earningsTask
            let ledger = try await ledgerTask
            let authority = try await authorityTask
            let mismatches = try await mismatchTask

            earnings = loadedEarnings
            monthLabel = MoneyFormat.minor(loadedEarnings.monthMinor, currency: loadedEarnings.currency)
            ledgerCount = ledger.count
            settlementRows = authority.items.count
            mismatchCount = mismatches.items.count
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
    }
}
