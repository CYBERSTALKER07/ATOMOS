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
    private var loadGeneration = 0
    var error: String?

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = earnings != nil
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            async let earningsTask = SupplierService.earnings()
            async let ledgerTask = SupplierOperationsService.paymentLedger()
            async let authorityTask = SupplierOperationsService.paymentSettlementAuthority()
            async let mismatchTask = SupplierOperationsService.paymentReconciliationMismatches()

            let loadedEarnings = try await earningsTask
            let ledger = try await ledgerTask
            let authority = try await authorityTask
            let mismatches = try await mismatchTask
            guard generation == loadGeneration else { return }

            earnings = loadedEarnings
            if let month = EarningsHonesty.minor(loadedEarnings.monthMinor, authoritative: loadedEarnings.authoritative) {
                monthLabel = MoneyFormat.minor(month, currency: loadedEarnings.currency)
            } else {
                monthLabel = "—"
            }
            ledgerCount = ledger.count
            settlementRows = authority.items.count
            mismatchCount = mismatches.items.count
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
