import Foundation
import Observation

@Observable
@MainActor
final class PaymentConfigViewModel {
    typealias FetchPaymentConfig = @MainActor () async throws -> PaymentConfigResponse

    var listings: [PSPListing] = []
    var currency = ""
    var loading = true
    var error: String?

    private let fetchPaymentConfig: FetchPaymentConfig

    init(fetchPaymentConfig: FetchPaymentConfig? = nil) {
        self.fetchPaymentConfig = fetchPaymentConfig ?? {
            try await WarehouseService.paymentConfig()
        }
    }

    func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let resp = try await fetchPaymentConfig()
            listings = resp.catalog.isEmpty
                ? resp.gateways.map { PSPListing(code: $0.code, status: $0.status, selectable: $0.selectable) }
                : resp.catalog
            currency = resp.currencyCode
        } catch {
            self.error = error.localizedDescription
        }
    }
}
