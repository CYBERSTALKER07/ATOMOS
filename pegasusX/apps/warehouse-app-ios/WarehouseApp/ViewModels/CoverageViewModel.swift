import Foundation
import Observation

@Observable
@MainActor
final class CoverageViewModel {
    typealias FetchCoverage = @MainActor () async throws -> WarehouseCoverageResponse
    typealias FetchFactory = @MainActor () async throws -> WarehouseSupplyFactoryResponse

    var coverage: WarehouseCoverageResponse?
    var factory: WarehouseSupplyFactoryResponse?
    var factoryError: String?
    var loading = true
    var error: String?

    private let fetchCoverage: FetchCoverage
    private let fetchFactory: FetchFactory

    init(
        fetchCoverage: FetchCoverage? = nil,
        fetchFactory: FetchFactory? = nil
    ) {
        self.fetchCoverage = fetchCoverage ?? {
            try await WarehouseService.opsCoverage()
        }
        self.fetchFactory = fetchFactory ?? {
            try await WarehouseService.opsSupplyFactory()
        }
    }

    func coverageModeLabel(_ mode: String?) -> String {
        switch (mode ?? "").uppercased() {
        case "PINNED": "Pinned"
        case "CITY_CELLS": "City cells"
        default: "Closest in country"
        }
    }

    func load() async {
        loading = true
        error = nil
        factoryError = nil
        defer { loading = false }
        do {
            coverage = try await fetchCoverage()
        } catch {
            self.error = error.localizedDescription
            return
        }
        do {
            factory = try await fetchFactory()
        } catch {
            factoryError = error.localizedDescription
        }
    }
}
