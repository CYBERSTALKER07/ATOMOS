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
                    ProgressView()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let error {
                    ContentUnavailableView {
                        Label("Error", systemImage: "exclamationmark.triangle")
                    } description: {
                        Text(error)
                    } actions: {
                        Button("Retry") { load() }
                    }
                } else if staff.isEmpty {
                    ContentUnavailableView("No Staff", systemImage: "person.2", description: Text("No staff members found"))
                } else {
                    List {
                        ForEach(Array(staff.enumerated()), id: \.element.id) { index, member in
                            HStack(spacing: LabTheme.spacingLG) {
                                Image(systemName: "person.circle")
                                    .font(.title2)
                                    .foregroundStyle(.secondary)
                                    .frame(width: 32)
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(member.name)
                                        .font(.subheadline.bold())
                                    Text(member.phone)
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                    Text(member.role)
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                                Spacer()
                                Text(member.status)
                                    .font(.caption2.bold())
                                    .padding(.horizontal, 8)
                                    .padding(.vertical, 3)
                                    .background(.quaternary)
                                    .clipShape(Capsule())
                            }
                            .staggeredAppear(index: index)
                        }
                    }
                    .listStyle(.plain)
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Staff")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button { load() } label: {
                        Image(systemName: "arrow.clockwise")
                    }
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
    @State private var transitioningID: String?
    @State private var refreshing = false
    @State private var staleMessage: String?
    @State private var lastSyncedAt: Date?

    private var filteredRequests: [SupplyRequest] {
        selectedFilter == "ALL" ? requests : requests.filter { $0.state == selectedFilter }
    }

    private var runtimeStatus: String {
        if refreshing {
            return "Refreshing live queue — last sync \(supplySyncText(lastSyncedAt))"
        }

        if let staleMessage {
            return staleMessage
        }

        if lastSyncedAt != nil {
            return "Live sync active — last sync \(supplySyncText(lastSyncedAt))"
        }

        return "Waiting for first sync"
    }

    var body: some View {
        NavigationStack {
            Group {
                if loading {
                    ProgressView()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let error {
                    ContentUnavailableView {
                        Label("Error", systemImage: "exclamationmark.triangle")
                    } description: {
                        Text(error)
                    } actions: {
                        Button("Retry") { Task { await load() } }
                    }
                } else if filteredRequests.isEmpty {
                    ContentUnavailableView(
                        selectedFilter == "ALL" ? "No Supply Requests" : "No \(selectedFilter.replacingOccurrences(of: "_", with: " ")) Requests",
                        systemImage: "checklist",
                        description: Text("Warehouse demand will appear here as soon as requests reach this factory queue.")
                    )
                } else {
                    ScrollView {
                        VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
                            SupplySummaryCard(
                                total: requests.count,
                                visible: filteredRequests.count,
                                runtimeStatus: runtimeStatus,
                                stale: staleMessage != nil
                            )
                            SupplyFilterRow(selectedFilter: $selectedFilter)

                            LazyVStack(spacing: LabTheme.spacingSM) {
                                ForEach(Array(filteredRequests.enumerated()), id: \.element.id) { index, request in
                                    SupplyRequestCard(
                                        request: request,
                                        transitioning: transitioningID == request.id,
                                        onAction: { action in
                                            Task { await transition(request: request, action: action) }
                                        }
                                    )
                                    .staggeredAppear(index: index)
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
                    onStateChange: { _ in },
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
        }
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
    private func transition(request: SupplyRequest, action: String) async {
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
    let stale: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            Text("Warehouse demand queue")
                .font(.title2.bold())
            Text("\(visible) requests in view, \(total) total across the factory queue.")
                .font(.body)
                .foregroundStyle(.secondary)
            Text(runtimeStatus)
                .font(.footnote.weight(.medium))
                .foregroundStyle(stale ? Color.red : .secondary)
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(.horizontal, LabTheme.spacingMD)
                .padding(.vertical, LabTheme.spacingSM)
                .background(stale ? Color.red.opacity(0.1) : LabTheme.tertiaryBackground, in: RoundedRectangle(cornerRadius: LabTheme.radiusMD))
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
