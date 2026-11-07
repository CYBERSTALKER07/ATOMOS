import AppKit
import Combine
import Foundation
import SwiftUI

public enum NavigationSection: String, CaseIterable, Identifiable {
    case overview = "Overview"
    case infra = "Infrastructure"
    case finops = "FinOps & Billing"
    case telemetry = "Live Telemetry"
    case analytics = "Scorecards"

    public var id: String { rawValue }

    public var icon: String {
        switch self {
        case .overview: return "gauge.with.needle.fill"
        case .infra: return "server.rack"
        case .finops: return "chart.bar.xaxis"
        case .telemetry: return "waveform.path.ecg"
        case .analytics: return "target"
        }
    }
}

@MainActor
public final class ControlTowerViewModel: ObservableObject {
    @Published public var selectedSection: NavigationSection = .overview
    @Published public var selectedRole: RolePerspective = .overall {
        didSet {
            Task { await loadRoleSpecificData() }
        }
    }

    // Telemetry Filtering
    @Published public var searchQuery: String = ""
    @Published public var selectedSeverity: LogSeverity? = nil

    // Data States
    @Published public private(set) var cells: [CellHealth] = []
    @Published public private(set) var spanner: SpannerClusterMetrics?
    @Published public private(set) var kafkaPartitions: [KafkaPartitionLag] = []
    @Published public private(set) var redisPools: [RedisPermitPool] = []
    @Published public private(set) var waterfallSlices: [TenderWaterfallSlice] = []
    @Published public private(set) var costs: [AttributedCostItem] = []
    @Published public private(set) var scorecards: [RolePerformanceScorecard] = []
    @Published public private(set) var isLoading: Bool = false
    @Published public private(set) var lastRefreshed: Date = Date()
    @Published public var copiedToastMessage: String?

    public let streamClient: WebSocketStreamClient
    private let apiClient: ControlTowerAPIClient

    public init(
        apiClient: ControlTowerAPIClient,
        streamClient: WebSocketStreamClient
    ) {
        self.apiClient = apiClient
        self.streamClient = streamClient
    }

    public convenience init(apiClient: ControlTowerAPIClient = .shared) {
        self.init(apiClient: apiClient, streamClient: WebSocketStreamClient())
    }

    public func initialLoad() async {
        isLoading = true
        defer { isLoading = false }
        do {
            async let fetchedCells = apiClient.fetchCellHealth()
            async let fetchedSpanner = apiClient.fetchSpannerMetrics()
            async let fetchedKafka = apiClient.fetchKafkaLags()
            async let fetchedRedis = apiClient.fetchRedisPools()
            async let fetchedWaterfall = apiClient.fetchWaterfall(for: selectedRole)
            async let fetchedCosts = apiClient.fetchAttributedCosts()
            async let fetchedScorecards = apiClient.fetchScorecards()

            self.cells = try await fetchedCells
            self.spanner = try await fetchedSpanner
            self.kafkaPartitions = try await fetchedKafka
            self.redisPools = try await fetchedRedis
            self.waterfallSlices = try await fetchedWaterfall
            self.costs = try await fetchedCosts
            self.scorecards = try await fetchedScorecards
            self.lastRefreshed = Date()
        } catch {
            print("ControlTowerViewModel initial load error: \(error)")
        }
    }

    public func loadRoleSpecificData() async {
        do {
            self.waterfallSlices = try await apiClient.fetchWaterfall(for: selectedRole)
        } catch {
            print("Failed to load role waterfall: \(error)")
        }
    }

    public func copyToClipboard(_ text: String, label: String = "ID") {
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(text, forType: .string)
        withAnimation {
            self.copiedToastMessage = "Copied \(label) to clipboard: \(text)"
        }
        DispatchQueue.main.asyncAfter(deadline: .now() + 2.5) { [weak self] in
            withAnimation {
                self?.copiedToastMessage = nil
            }
        }
    }

    public var filteredTelemetryEvents: [TelemetryLogEvent] {
        streamClient.events.filter { event in
            let matchesSeverity = selectedSeverity == nil || event.severity == selectedSeverity
            let matchesQuery = searchQuery.isEmpty ||
                event.message.localizedCaseInsensitiveContains(searchQuery) ||
                event.entityId.localizedCaseInsensitiveContains(searchQuery) ||
                event.service.localizedCaseInsensitiveContains(searchQuery) ||
                event.traceId.localizedCaseInsensitiveContains(searchQuery)

            let matchesRole: Bool
            switch selectedRole {
            case .overall: matchesRole = true
            case .supplier: matchesRole = event.role.localizedCaseInsensitiveContains("Supplier")
            case .warehouse: matchesRole = event.role.localizedCaseInsensitiveContains("Warehouse")
            case .retailer: matchesRole = event.role.localizedCaseInsensitiveContains("Retailer")
            case .fleet: matchesRole = event.role.localizedCaseInsensitiveContains("Fleet")
            }

            return matchesSeverity && matchesQuery && matchesRole
        }
    }

    public func exportDiagnosticBundle() {
        guard let spanner = spanner else { return }
        let bundle = DiagnosticBundle(
            bundleId: UUID().uuidString.prefix(8).uppercased(),
            generatedAt: ISO8601DateFormatter().string(from: Date()),
            appVersion: "1.0.0 (macOS Native)",
            activeCell: "cell-uz-tashkent",
            logs: streamClient.events,
            spannerStatus: spanner,
            kafkaPartitions: kafkaPartitions
        )
        ExportService.exportDiagnosticBundle(bundle: bundle)
    }

    public func exportWaterfallCSV() {
        ExportService.exportWaterfallCSV(slices: waterfallSlices, role: selectedRole)
    }
}
