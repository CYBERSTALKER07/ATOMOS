import Foundation
import Observation

@Observable
@MainActor
final class ManifestsViewModel {
    typealias FetchManifests = @MainActor () async throws -> ManifestListResponse

    var manifests: [Manifest] = []
    var loading = true
    var error: String?

    private let fetchManifests: FetchManifests

    init(fetchManifests: FetchManifests? = nil) {
        self.fetchManifests = fetchManifests ?? {
            try await WarehouseService.manifests()
        }
    }

    func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let resp = try await fetchManifests()
            manifests = resp.manifests
        } catch {
            self.error = error.localizedDescription
        }
    }
}
