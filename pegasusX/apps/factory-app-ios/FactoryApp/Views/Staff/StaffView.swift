import SwiftUI

struct StaffView: View {
    @State private var realtimeClient = FactoryRealtimeClient()
    @State private var staff: [StaffMember] = []
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        NavigationStack {
            Group {
                if loading {
                    FactoryLoadingView(
                        title: "Loading staff",
                        message: "Fetching factory operators and shift status."
                    )
                } else if let error {
                    FactoryErrorView(message: error, retry: { load() })
                } else if staff.isEmpty {
                    FactoryStateView(
                        kind: .empty,
                        headline: "No staff",
                        message: "No staff members are registered for this factory."
                    )
                } else {
                    StaffListContent(staff: staff)
                        .navigationDestination(for: String.self) { staffId in
                            StaffDetailView(staffId: staffId)
                        }
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Staff")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise", action: { load() })
                        .labelStyle(.iconOnly)
                }
            }
            .task { load() }
            .onAppear {
                realtimeClient.connect(
                    onStateChange: { _ in },
                    onEvent: { event in
                        guard let eventType = event.eventType else { return }
                        switch eventType {
                        case .supplyRequestUpdate, .transferUpdate, .manifestUpdate:
                            load()
                        case .outboxFailed:
                            break
                        }
                    }
                )
            }
            .onDisappear {
                realtimeClient.disconnect()
            }
        }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                staff = try await FactoryService.staff().staff
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }
}

private struct RequestActionSpec {
    let title: String
    let action: String
    let emphasized: Bool
}

private let supplyFilters = ["ALL", "SUBMITTED", "ACKNOWLEDGED", "IN_PRODUCTION", "READY", "FULFILLED", "CANCELLED"]

private func requestActions(for state: String) -> [RequestActionSpec] {
    switch state {
    case "SUBMITTED":
        return [
            RequestActionSpec(title: "Acknowledge", action: "ACKNOWLEDGE", emphasized: true),
            RequestActionSpec(title: "Cancel", action: "CANCEL", emphasized: false)
        ]
    case "ACKNOWLEDGED":
        return [
            RequestActionSpec(title: "Start Production", action: "START_PRODUCTION", emphasized: true),
            RequestActionSpec(title: "Cancel", action: "CANCEL", emphasized: false)
        ]
    case "IN_PRODUCTION":
        return [
            RequestActionSpec(title: "Mark Ready", action: "MARK_READY", emphasized: true)
        ]
    case "READY":
        return [
            RequestActionSpec(title: "Fulfill", action: "FULFILL", emphasized: true)
        ]
    default:
        return []
    }
}

struct SupplyRequestsView: View {
    @Environment(\.scenePhase) private var scenePhase
    @State private var realtimeClient = FactoryRealtimeClient()
    @State private var requests: [SupplyRequest] = []
    @State private var loading = true
    @State private var error: String?
    @State private var selectedFilter = "ALL"
    @State private var viewMode = "TABLE"
    @State private var fulfillModal: (SupplyRequest, SupplyFulfillOptions)?
    @State private var transitioningID: String?
    @State private var refreshing = false
    @State private var staleMessage: String?
    @State private var lastSyncedAt: Date?
    @State private var realtimeStatus: FactoryRealtimeStatus = .idle

    private var filteredRequests: [SupplyRequest] {
        selectedFilter == "ALL" ? requests : requests.filter { $0.state == selectedFilter }
    }

    private var runtimeStatus: String {
        if let staleMessage {
            return staleMessage
        }

        switch realtimeStatus {
        case .offline:
            return "Offline — showing last sync \(supplySyncText(lastSyncedAt))"
        case .reconnecting:
            return "Reconnecting live queue — last sync \(supplySyncText(lastSyncedAt))"
        case .connecting:
            return "Connecting to the live supply queue…"
        case .idle, .live:
            break
        }

        if refreshing {
            return "Refreshing live queue — last sync \(supplySyncText(lastSyncedAt))"
        }

        if lastSyncedAt != nil {
            return "Live sync active — last sync \(supplySyncText(lastSyncedAt))"
        }

        return "Waiting for first sync"
    }

    private var runtimeTone: FactoryRuntimeTone {
        if staleMessage != nil && realtimeStatus == .offline {
            return .offline
        }

        if staleMessage != nil {
            return .warning
        }

        switch realtimeStatus {
        case .offline:
            return .offline
        case .reconnecting, .connecting:
            return .refreshing
        case .idle, .live:
            break
        }

        return refreshing ? .refreshing : .live
    }

    var body: some View {
        NavigationStack {
            Group {
                if loading {
                    FactoryLoadingState(
                        title: "Loading supply requests",
                        message: "Fetching the live warehouse demand queue for this factory."
                    )
                } else if let error {
                    FactoryStateView(
                        kind: realtimeStatus == .offline ? .offline : .error,
                        headline: realtimeStatus == .offline ? "Supply queue unavailable offline" : "Unable to load supply requests",
                        message: error,
                        actionTitle: "Retry",
                        action: { Task { await load() } }
                    )
                } else if filteredRequests.isEmpty {
                    FactoryStateView(
                        kind: selectedFilter == "ALL" ? .empty : .noResults,
                        headline: selectedFilter == "ALL"
                            ? "No supply requests"
                            : "No \(selectedFilter.replacingOccurrences(of: "_", with: " ").lowercased()) requests",
                        message: selectedFilter == "ALL"
                            ? "Warehouse demand will appear here as soon as requests reach this factory queue."
                            : "Adjust the active filter or wait for the next queue refresh.",
                        actionTitle: selectedFilter == "ALL" ? nil : "Clear Filter",
                        action: selectedFilter == "ALL" ? nil : { selectedFilter = "ALL" }
                    )
                } else {
                    ScrollView {
                        VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
                            SupplySummaryCard(
                                total: requests.count,
                                visible: filteredRequests.count,
                                runtimeStatus: runtimeStatus,
                                runtimeTone: runtimeTone
                            )
                            SupplyFilterRow(selectedFilter: $selectedFilter)
                            SupplyViewModeRow(viewMode: $viewMode)

                            if viewMode == "BOARD" {
                                SupplyBoard(
                                    requests: filteredRequests,
                                    transitioningID: transitioningID,
                                    onAction: { request, action in
                                        Task { await handleAction(request: request, action: action) }
                                    }
                                )
                            } else {
                            LazyVStack(spacing: LabTheme.spacingSM) {
                                ForEach(Array(filteredRequests.enumerated()), id: \.element.id) { index, request in
                                    SupplyRequestCard(
                                        request: request,
                                        transitioning: transitioningID == request.id,
                                        onAction: { action in
                                            Task { await handleAction(request: request, action: action) }
                                        }
                                    )
                                    .staggeredAppear(index: index)
                                }
                            }
                            }
                        }
                        .padding()
                    }
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Supply Requests")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") {
                        Task { await load(background: !requests.isEmpty) }
                    }
                    .labelStyle(.iconOnly)
                }
            }
            .task { await load() }
            .task {
                while !Task.isCancelled {
                    try? await Task.sleep(for: .seconds(30))
                    if transitioningID == nil {
                        await load(background: true)
                    }
                }
            }
            .onChange(of: scenePhase) { _, newPhase in
                if newPhase == .active {
                    Task { await load(background: !requests.isEmpty) }
                }
            }
            .onAppear {
                realtimeClient.connect(
                    onStateChange: { status in
                        realtimeStatus = status
                    },
                    onEvent: { event in
                        guard event.eventType == .supplyRequestUpdate else { return }
                        if transitioningID == nil {
                            Task { await load(background: !requests.isEmpty) }
                        }
                    }
                )
            }
            .onDisappear {
                realtimeClient.disconnect()
            }
            .sheet(item: Binding(
                get: { fulfillModal.map { FulfillModalItem(request: $0.0, options: $0.1) } },
                set: { item in fulfillModal = item.map { ($0.request, $0.options) } }
            )) { item in
                NavigationStack {
                    Form {
                        Section("Fulfill decision") {
                            Text("\(item.options.warehouseName) · \(item.options.transferMode)")
                            Text(item.options.outcomeInternal)
                                .font(.footnote)
                                .foregroundStyle(.secondary)
                            Text(item.options.outcomeTruck)
                                .font(.footnote)
                                .foregroundStyle(.secondary)
                            if let eta = item.options.linkedDriverETA {
                                Text("Driver ETA: \(eta)")
                                    .font(.footnote)
                            }
                        }
                    }
                    .navigationTitle("Confirm fulfill")
                    .toolbar {
                        ToolbarItem(placement: .cancellationAction) {
                            Button("Cancel") { fulfillModal = nil }
                        }
                        ToolbarItem(placement: .confirmationAction) {
                            Button("Confirm") {
                                let request = item.request
                                fulfillModal = nil
                                Task { await runTransition(request: request, action: "FULFILL") }
                            }
                        }
                    }
                }
                .presentationDetents([.medium])
            }
        }
    }

    private struct FulfillModalItem: Identifiable {
        let request: SupplyRequest
        let options: SupplyFulfillOptions
        var id: String { request.id }
    }

    @MainActor
    private func handleAction(request: SupplyRequest, action: String) async {
        if action == "FULFILL" {
            do {
                let options = try await FactoryService.supplyFulfillOptions(id: request.id)
                fulfillModal = (request, options)
            } catch {
                self.error = error.localizedDescription
            }
            return
        }
        await runTransition(request: request, action: action)
    }

    @MainActor
    private func load(background: Bool = false) async {
        if background {
            refreshing = true
        } else if requests.isEmpty {
            loading = true
            error = nil
        }

        do {
            requests = try await FactoryService.supplyRequests()
            staleMessage = nil
            error = nil
            lastSyncedAt = Date()
        } catch {
            let message = error.localizedDescription
            if requests.isEmpty {
                self.error = message
            } else {
                staleMessage = "Showing last synced queue. \(message)"
            }
        }

        loading = false
        refreshing = false
    }

    @MainActor
    private func runTransition(request: SupplyRequest, action: String) async {
        transitioningID = request.id

        do {
            _ = try await FactoryService.transitionSupplyRequest(id: request.id, action: action)
            await load(background: true)
        } catch {
            self.error = error.localizedDescription
        }

        transitioningID = nil
    }
}

private struct SupplySummaryCard: View {
    let total: Int
    let visible: Int
    let runtimeStatus: String
    let runtimeTone: FactoryRuntimeTone

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            Text("Warehouse demand queue")
                .font(.title2.bold())
            Text("\(visible) requests in view, \(total) total across the factory queue.")
                .font(.body)
                .foregroundStyle(.secondary)
            FactoryRuntimeBanner(tone: runtimeTone, message: runtimeStatus)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
    }
}

private struct SupplyFilterRow: View {
    @Binding var selectedFilter: String

    var body: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: LabTheme.spacingSM) {
                ForEach(supplyFilters, id: \.self) { filter in
                    Button {
                        selectedFilter = filter
                    } label: {
                        Text(filter.replacingOccurrences(of: "_", with: " "))
                            .font(.footnote.bold())
                            .padding(.horizontal, 12)
                            .padding(.vertical, 6)
                            .background(selectedFilter == filter ? LabTheme.label : Color.clear, in: Capsule())
                            .foregroundStyle(selectedFilter == filter ? Color(.systemBackground) : LabTheme.label)
                            .overlay(Capsule().stroke(.quaternary))
                    }
                    .buttonStyle(PressableButtonStyle())
                }
            }
        }
    }
}

private struct SupplyViewModeRow: View {
    @Binding var viewMode: String

    var body: some View {
        HStack(spacing: LabTheme.spacingSM) {
            modeButton("TABLE", label: "Table")
            modeButton("BOARD", label: "Board")
        }
    }

    private func modeButton(_ mode: String, label: String) -> some View {
        Button {
            viewMode = mode
        } label: {
            Text(label)
                .font(.footnote.bold())
                .padding(.horizontal, 12)
                .padding(.vertical, 6)
                .background(viewMode == mode ? LabTheme.label : Color.clear, in: Capsule())
                .foregroundStyle(viewMode == mode ? Color(.systemBackground) : LabTheme.label)
                .overlay(Capsule().stroke(.quaternary))
        }
        .buttonStyle(PressableButtonStyle())
    }
}

private struct SupplyBoard: View {
    let requests: [SupplyRequest]
    let transitioningID: String?
    let onAction: (SupplyRequest, String) -> Void

    private let lanes = ["SUBMITTED", "ACKNOWLEDGED", "IN_PRODUCTION", "READY"]

    var body: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(alignment: .top, spacing: LabTheme.spacingSM) {
                ForEach(lanes, id: \.self) { lane in
                    VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
                        Text(lane.replacingOccurrences(of: "_", with: " "))
                            .font(.footnote.bold())
                        ForEach(requests.filter { $0.state == lane }) { request in
                            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                Text(requestLabel(request))
                                    .font(.subheadline.bold())
                                Text(request.priority.isEmpty ? "NORMAL" : request.priority)
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                                ForEach(requestActions(for: request.state), id: \.action) { action in
                                    SupplyActionButton(action: action, transitioning: transitioningID == request.id) {
                                        onAction(request, action.action)
                                    }
                                }
                            }
                            .frame(width: 220, alignment: .leading)
                            .labCard()
                        }
                    }
                }
            }
        }
    }
}

private struct SupplyRequestCard: View {
    let request: SupplyRequest
    let transitioning: Bool
    let onAction: (String) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingMD) {
            HStack(alignment: .top, spacing: LabTheme.spacingMD) {
                VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                    Text(requestLabel(request))
                        .font(.subheadline.bold())
                    Text("Request \(request.id.prefix(8))")
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                VStack(alignment: .trailing, spacing: LabTheme.spacingXS) {
                    SupplyTag(text: request.state, emphasized: true)
                    SupplyTag(text: request.priority.isEmpty ? "NORMAL" : request.priority, emphasized: false)
                }
            }

            HStack(spacing: LabTheme.spacingSM) {
                SupplyMetric(label: "Volume", value: supplyVolumeLabel(request.totalVolumeVU))
                SupplyMetric(label: "Created", value: supplyShortDate(request.createdAt))
                SupplyMetric(label: "Delivery", value: supplyShortDate(request.requestedDeliveryDate))
            }

            if !request.notes.isEmpty {
                Text(request.notes)
                    .font(.footnote)
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(LabTheme.spacingMD)
                    .background(LabTheme.secondaryBackground, in: RoundedRectangle(cornerRadius: LabTheme.radiusMD))
            }

            let actions = requestActions(for: request.state)
            if actions.isEmpty {
                Text("No manual action is available for the current state.")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            } else {
                HStack(spacing: LabTheme.spacingSM) {
                    ForEach(actions, id: \.action) { action in
                        SupplyActionButton(action: action, transitioning: transitioning) {
                            onAction(action.action)
                        }
                    }
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
    }
}

private struct SupplyActionButton: View {
    let action: RequestActionSpec
    let transitioning: Bool
    let onTap: () -> Void

    var body: some View {
        Group {
            if action.emphasized {
                Button(action.title, action: onTap)
                    .buttonStyle(.borderedProminent)
            } else {
                Button(action.title, action: onTap)
                    .buttonStyle(.bordered)
            }
        }
        .disabled(transitioning)
        .frame(maxWidth: .infinity)
    }
}

private struct SupplyMetric: View {
    let label: String
    let value: String

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
            Text(value)
                .font(.subheadline.bold())
            Text(label)
                .font(.footnote)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(LabTheme.spacingMD)
        .background(LabTheme.tertiaryBackground, in: RoundedRectangle(cornerRadius: LabTheme.radiusMD))
    }
}

private struct SupplyTag: View {
    let text: String
    let emphasized: Bool

    var body: some View {
        Text(text.replacingOccurrences(of: "_", with: " "))
            .font(.footnote.bold())
            .padding(.horizontal, LabTheme.spacingSM)
            .padding(.vertical, LabTheme.spacingXS)
            .background(emphasized ? LabTheme.fill : LabTheme.tertiaryBackground, in: Capsule())
    }
}

private func requestLabel(_ request: SupplyRequest) -> String {
    if request.warehouseId.isEmpty { return "Warehouse" }
    return "Warehouse \(request.warehouseId.prefix(8))"
}

private func supplyVolumeLabel(_ value: Double) -> String {
    value.rounded(.towardZero) == value ? "\(Int(value)) VU" : String(format: "%.1f VU", value)
}

private func supplyShortDate(_ value: String?) -> String {
    guard let value, !value.isEmpty else { return "Unscheduled" }
    return String(value.prefix { $0 != "T" })
}

private func supplySyncText(_ value: Date?) -> String {
    guard let value else { return "waiting" }
    return value.formatted(date: .omitted, time: .shortened)
}
