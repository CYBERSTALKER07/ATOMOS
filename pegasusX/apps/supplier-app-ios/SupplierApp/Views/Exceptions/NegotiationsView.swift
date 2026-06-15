import SwiftUI

struct NegotiationsView: View {
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var rows: [NegotiationProposalRow] = []
    @State private var loading = true
    @State private var error: String?
    @State private var busyId: String?
    @State private var statusMessage: String?

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading negotiations…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if rows.isEmpty {
                SupplierEmptyView(title: "No pending negotiations", message: "Driver quantity proposals appear here.")
            } else {
                List(rows) { row in
                    VStack(alignment: .leading, spacing: 8) {
                        Text(row.orderId).font(.headline)
                        Text("\(row.items.count) line items · Driver \(row.driverId)").font(.caption)
                        HStack {
                            Button("Approve") { resolve(row.proposalId, action: "APPROVE") }
                                .disabled(busyId == row.proposalId)
                            Button("Reject") { resolve(row.proposalId, action: "REJECT") }
                                .disabled(busyId == row.proposalId)
                        }
                        .buttonStyle(.bordered)
                    }
                }
                .listStyle(.insetGrouped)
            }
        }
        .navigationTitle("Negotiations")
        .task { await load() }
        .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
            if busyId != nil {
                busyId = nil
                statusMessage = "Connection restored — verify status before retrying."
            }
            Task { await load() }
        }
        .safeAreaInset(edge: .bottom) {
            if let statusMessage {
                Text(statusMessage)
                    .font(.footnote)
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 8)
                    .background(.bar)
            }
        }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            rows = try await SupplierOperationsService.negotiationsPending()
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func resolve(_ proposalId: String, action: String) {
        busyId = proposalId
        statusMessage = nil
        Task {
            defer { busyId = nil }
            do {
                let key = "supplier-negotiate-resolve:\(proposalId):\(action)"
                _ = try await SupplierOperationsService.resolveNegotiation(
                    NegotiationResolveRequest(proposalId: proposalId, action: action, resolution: nil),
                    idempotencyKey: key
                )
                statusMessage = "Negotiation \(action)"
                await load()
            } catch {
                statusMessage = error.localizedDescription
            }
        }
    }
}
