import Foundation

/// Supplier discovery for retailer procurement — catalog search + preference mutations.
enum RetailerSupplierDiscoveryService {
    static func searchSuppliers(api: APIClient, query: String) async throws -> [Supplier] {
        let trimmed = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard trimmed.count >= 2 else { return [] }
        let encoded = trimmed.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? trimmed
        return try await api.get(path: "/v1/catalog/suppliers/search?q=\(encoded)")
    }

    static func addSupplier(api: APIClient, supplierId: String) async throws {
        let _: APIResponse<String> = try await api.post(
            path: "/v1/retailer/suppliers/\(supplierId)/add",
            body: EmptyRequest(),
            headers: ["Idempotency-Key": "retailer-supplier-add:\(supplierId)"]
        )
    }

    static func removeSupplier(api: APIClient, supplierId: String) async throws {
        let _: APIResponse<String> = try await api.post(
            path: "/v1/retailer/suppliers/\(supplierId)/remove",
            body: EmptyRequest(),
            headers: ["Idempotency-Key": "retailer-supplier-remove:\(supplierId)"]
        )
    }

    static func filterNewCandidates(_ results: [Supplier], existing: [Supplier]) -> [Supplier] {
        let existingIds = Set(existing.map(\.id))
        return results.filter { !existingIds.contains($0.id) }
    }
}

private struct EmptyRequest: Encodable {}
