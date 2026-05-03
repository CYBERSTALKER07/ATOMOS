import SwiftUI

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
    @State private var requests: [SupplyRequest] = []
    @State private var loading = true
    @State private var error: String?
    @State private var selectedFilter = "ALL"
    @State private var transitioningID: String?

    private var filteredRequests: [SupplyRequest] {
        selectedFilter == "ALL" ? requests : requests.filter { $0.state == selectedFilter }
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
                        Button("Retry") { load() }
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
                            SupplySummaryCard(total: requests.count, visible: filteredRequests.count)
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
                        load()
                    }
                    .labelStyle(.iconOnly)
                }
            }
            .task { load() }
        }
    }

    private func load() {
        loading = true
        error = nil

        Task {
            do {
                requests = try await FactoryService.supplyRequests()
            } catch {
                self.error = error.localizedDescription
            }

            loading = false
        }
    }

    @MainActor
    private func transition(request: SupplyRequest, action: String) async {
        transitioningID = request.id

        do {
            _ = try await FactoryService.transitionSupplyRequest(id: request.id, action: action)
            requests = try await FactoryService.supplyRequests()
        } catch {
            self.error = error.localizedDescription
        }

        transitioningID = nil
    }
}

private struct SupplySummaryCard: View {
    let total: Int
    let visible: Int

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            Text("Warehouse demand queue")
                .font(.title2.bold())
            Text("\(visible) requests in view, \(total) total across the factory queue.")
                .font(.body)
                .foregroundStyle(.secondary)
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
                SupplyMetric(label: "Volume", value: volumeLabel(request.totalVolumeVU))
                SupplyMetric(label: "Created", value: shortDate(request.createdAt))
                SupplyMetric(label: "Delivery", value: shortDate(request.requestedDeliveryDate))
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
                        Button(action.title) {
                            onAction(action.action)
                        }
                        .buttonStyle(action.emphasized ? .borderedProminent : .bordered)
                        .disabled(transitioning)
                        .frame(maxWidth: .infinity)
                    }
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
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

private func volumeLabel(_ value: Double) -> String {
    value.rounded(.towardZero) == value ? "\(Int(value)) VU" : String(format: "%.1f VU", value)
}

private func shortDate(_ value: String?) -> String {
    guard let value, !value.isEmpty else { return "Unscheduled" }
    return String(value.prefix { $0 != "T" })
}
