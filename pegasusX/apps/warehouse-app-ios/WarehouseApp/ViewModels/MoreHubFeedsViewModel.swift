import Foundation
import Observation

@Observable
@MainActor
final class WarehouseJSONFeedViewModel {
    typealias FetchJSON = @MainActor (String, [String: String]) async throws -> String

    var bodyText = ""
    var errorText: String?
    var loading = true

    private let fetchJSON: FetchJSON

    init(fetchJSON: FetchJSON? = nil) {
        self.fetchJSON = fetchJSON ?? { path, query in
            try await APIClient.shared.getJSONString(path, query: query)
        }
    }

    func load(path: String, query: [String: String] = [:]) async {
        loading = true
        defer { loading = false }
        do {
            bodyText = try await fetchJSON(path, query)
            errorText = nil
        } catch {
            errorText = error.localizedDescription
        }
    }
}

@Observable
@MainActor
final class WarehouseScoredExceptionsViewModel {
    typealias FetchScoredExceptions = @MainActor () async throws -> ScoredExceptionsResponse

    var rows: [ScoredException] = []
    var errorText: String?
    var loading = true

    private let fetchScoredExceptions: FetchScoredExceptions

    init(fetchScoredExceptions: FetchScoredExceptions? = nil) {
        self.fetchScoredExceptions = fetchScoredExceptions ?? {
            try await APIClient.shared.get(
                "v1/control-tower/exceptions/scored",
                query: ["limit": "50"]
            )
        }
    }

    func load() async {
        loading = true
        defer { loading = false }
        do {
            let resp = try await fetchScoredExceptions()
            rows = resp.exceptions
            errorText = nil
        } catch {
            errorText = error.localizedDescription
        }
    }
}
