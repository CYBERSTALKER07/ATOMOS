import AppKit
import Foundation
import Observation
import SwiftUI

public enum NavigationSection: String, CaseIterable, Identifiable, Sendable {
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

@Observable
@MainActor
public final class ControlTowerViewModel {
    public var selectedSection: NavigationSection = .overview {
        didSet {
            if selectedSection != oldValue {
                ControlTowerHaptics.alignment()
            }
        }
    }

    public var selectedRole: RolePerspective = .overall {
        didSet {
            if selectedRole != oldValue {
                ControlTowerHaptics.alignment()
                Task { await loadRoleSpecificData() }
            }
        }
    }

    // Telemetry Filtering & Presentation State
    public var searchQuery: String = ""
    public var selectedSeverity: LogSeverity? = nil
    public var isCheatsheetPresented: Bool = false
    public var isInspectorPresented: Bool = true

    public var telemetryHUDString: String {
        let latency: String
        if let spanner {
            latency = String(format: "%.1f", spanner.p99ReadMs)
        } else {
            latency = "4.2"
        }
        let wsStatus = streamClient.isConnected ? "LIVE" : "SYNC"
        let cellCount = cells.isEmpty ? 3 : cells.count
        return "SPK-UZ: \(latency)ms | WS \(wsStatus) | CELLS: \(cellCount)"
    }

    // Data States
    public private(set) var cells: [CellHealth] = []
    public private(set) var spanner: SpannerClusterMetrics?
    public private(set) var kafkaPartitions: [KafkaPartitionLag] = []
    public private(set) var redisPools: [RedisPermitPool] = []
    public private(set) var waterfallSlices: [TenderWaterfallSlice] = []
    public private(set) var costs: [AttributedCostItem] = []
    public private(set) var scorecards: [RolePerformanceScorecard] = []
    public private(set) var isLoading: Bool = false
    public private(set) var lastRefreshed: Date = Date()
    public private(set) var lastErrorMessage: String?
    public var copiedToastMessage: String?

    public let streamClient: WebSocketStreamClient
    private let apiClient: ControlTowerAPIClient
    private var toastGeneration: UInt64 = 0

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

    public func refresh() {
        ControlTowerHaptics.alignment()
        Task { await initialLoad() }
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
            self.lastErrorMessage = nil
        } catch {
            self.lastErrorMessage = error.localizedDescription
        }
    }

    public func loadRoleSpecificData() async {
        do {
            self.waterfallSlices = try await apiClient.fetchWaterfall(for: selectedRole)
            self.lastErrorMessage = nil
        } catch {
            self.lastErrorMessage = error.localizedDescription
        }
    }

    public func triageDLQInTelemetry() {
        ControlTowerHaptics.levelChange()
        selectedSection = .telemetry
        selectedSeverity = .error
    }

    public func copyToClipboard(_ text: String, label: String = "ID") {
        ControlTowerHaptics.generic()
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(text, forType: .string)
        toastGeneration &+= 1
        let activeGeneration = toastGeneration
        withAnimation {
            self.copiedToastMessage = "Copied \(label) to clipboard: \(text)"
        }
        DispatchQueue.main.asyncAfter(deadline: .now() + 2.5) { [weak self] in
            Task { @MainActor [weak self] in
                guard let self, self.toastGeneration == activeGeneration else { return }
                withAnimation {
                    self.copiedToastMessage = nil
                }
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
        ControlTowerHaptics.alignment()
        guard let spanner = spanner else { return }
        let bundle = DiagnosticBundle(
            bundleId: UUID().uuidString.prefix(8).uppercased(),
            generatedAt: ISO8601DateFormatter().string(from: Date()),
            appVersion: "1.0.0 (macOS Native)",
            activeCell: cells.first?.cellId ?? "unknown",
            logs: streamClient.events,
            spannerStatus: spanner,
            kafkaPartitions: kafkaPartitions
        )
        ExportService.exportDiagnosticBundle(bundle: bundle)
    }

    public func exportWaterfallCSV() {
        ControlTowerHaptics.alignment()
        ExportService.exportWaterfallCSV(slices: waterfallSlices, role: selectedRole)
    }

    public func signOut() async {
        await apiClient.clearSession()
        streamClient.clearSession()
        cells = []
        spanner = nil
        kafkaPartitions = []
        redisPools = []
        waterfallSlices = []
        costs = []
        scorecards = []
        lastErrorMessage = nil
    }
}
