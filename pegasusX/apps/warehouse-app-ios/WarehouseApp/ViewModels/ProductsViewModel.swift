import Foundation
import Observation

@Observable
@MainActor
final class ProductsViewModel {
    typealias FetchProducts = @MainActor () async throws -> ProductListResponse

    var products: [Product] = []
    var loading = true
    var error: String?

    private let fetchProducts: FetchProducts

    init(fetchProducts: FetchProducts? = nil) {
        self.fetchProducts = fetchProducts ?? {
            try await WarehouseService.products()
        }
    }

    func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let resp = try await fetchProducts()
            products = resp.products
        } catch {
            self.error = error.localizedDescription
        }
    }
}
