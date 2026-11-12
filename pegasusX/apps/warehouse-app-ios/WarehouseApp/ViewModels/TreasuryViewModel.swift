import Foundation
import Observation

@Observable
@MainActor
final class TreasuryViewModel {
    typealias FetchOverview = @MainActor () async throws -> TreasuryOverview
    typealias FetchFallbackFinancials = @MainActor () async throws -> OpsFinancialsResponse
    typealias FetchInvoices = @MainActor () async throws -> InvoiceListResponse

    var overview = TreasuryOverview.empty
    var invoices: [Invoice] = []
    var loading = true
    var error: String?
    var selectedSegment = 0

    private let fetchOverview: FetchOverview
    private let fetchFallbackFinancials: FetchFallbackFinancials
    private let fetchInvoices: FetchInvoices

    init(
        fetchOverview: FetchOverview? = nil,
        fetchFallbackFinancials: FetchFallbackFinancials? = nil,
        fetchInvoices: FetchInvoices? = nil
    ) {
        self.fetchOverview = fetchOverview ?? {
            try await WarehouseService.treasuryOverview()
        }
        self.fetchFallbackFinancials = fetchFallbackFinancials ?? {
            try await WarehouseOperationsService.opsFinancials()
        }
        self.fetchInvoices = fetchInvoices ?? {
            try await WarehouseService.treasuryInvoices()
        }
    }

    func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            async let invoicesResponse = fetchInvoices()
            do {
                overview = try await fetchOverview()
            } catch {
                let financials = try await fetchFallbackFinancials()
                overview = TreasuryOverview(
                    balance: Int(financials.netPayout),
                    totalReceivable: Int(financials.totalRevenue),
                    totalCollected: Int(financials.cashCollected),
                    overdueAmount: Int(financials.cashPending)
                )
            }
            invoices = try await invoicesResponse.invoices
        } catch {
            self.error = error.localizedDescription
        }
    }
}
