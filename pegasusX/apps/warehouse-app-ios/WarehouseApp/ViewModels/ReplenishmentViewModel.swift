import Foundation
import Observation

@Observable
@MainActor
final class ReplenishmentViewModel {
    typealias FetchInsights = @MainActor () async throws -> ReplenishmentInsightsResponse
    typealias PerformInsightAction = @MainActor (String, String) async throws -> ReplenishmentInsightActionResponse

    var insights: [ReplenishmentInsight] = []
    var loading = true
    var error: String?
    var actingId: String?
    var statusMessage: String?

    private let fetchInsights: FetchInsights
    private let performInsightAction: PerformInsightAction

    init(
        fetchInsights: FetchInsights? = nil,
        performInsightAction: PerformInsightAction? = nil
    ) {
        self.fetchInsights = fetchInsights ?? {
            try await WarehouseService.replenishmentInsights()
        }
        self.performInsightAction = performInsightAction ?? { insightId, action in
            try await WarehouseService.replenishmentInsightAction(insightId: insightId, action: action)
        }
    }

    func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer {
            if !silent { loading = false }
        }
        do {
            let response = try await fetchInsights()
            insights = response.rows
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
    }

    func runAction(insightId: String, action: String) async {
        actingId = insightId
        defer { actingId = nil }
        do {
            let response = try await performInsightAction(insightId, action)
            if action == "approve", let transferId = response.transferId, !transferId.isEmpty {
                statusMessage = "Approved — transfer \(String(transferId.prefix(8)))"
            } else {
                statusMessage = action == "approve" ? "Insight approved" : "Insight dismissed"
            }
            PegasusHaptics.success()
            await load()
        } catch {
            statusMessage = error.localizedDescription
            PegasusHaptics.error()
        }
    }

    func handleReconnect() {
        if actingId != nil {
            actingId = nil
            statusMessage = "Connection restored — verify status before retrying."
        }
    }

    func autoClearStatusMessage() async {
        guard statusMessage != nil else { return }
        try? await Task.sleep(nanoseconds: 2_500_000_000)
        guard !Task.isCancelled else { return }
        statusMessage = nil
    }
}
