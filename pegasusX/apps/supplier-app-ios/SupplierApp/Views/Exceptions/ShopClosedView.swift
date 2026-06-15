import SwiftUI

struct ShopClosedView: View {
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var rows: [ShopClosedAttemptRow] = []
    @State private var loading = true
    @State private var error: String?
    @State private var busyId: String?
    @State private var statusMessage: String?

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading shop-closed queue…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if rows.isEmpty {
                SupplierEmptyView(title: "No active attempts", message: "Driver-reported shop-closed cases appear here.")
            } else {
                List(rows) { row in
                    VStack(alignment: .leading, spacing: 8) {
                        Text(row.orderId).font(.headline)
                        Text("Driver \(row.driverId) · Retailer \(row.retailerId)").font(.caption)
                        HStack {
                            Button("Wait") { resolve(row.attemptId, action: "WAIT") }
                                .disabled(busyId == row.attemptId)
                            Button("Bypass") { resolve(row.attemptId, action: "BYPASS") }
                                .disabled(busyId == row.attemptId)
                            Button("Return") { resolve(row.attemptId, action: "RETURN_TO_DEPOT") }
                                .disabled(busyId == row.attemptId)
                        }
                        .buttonStyle(.bordered)
                    }
                }
                .listStyle(.insetGrouped)
            }
        }
        .navigationTitle("Shop closed")
        .task { await load() }
        .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
            if busyId != nil {
                busyId = nil
                statusMessage = "Connection restored — verify resolution status before retrying."
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
            rows = try await SupplierOperationsService.shopClosedActive()
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func resolve(_ attemptId: String, action: String) {
        busyId = attemptId
        statusMessage = nil
        Task {
            defer { busyId = nil }
            do {
                _ = try await SupplierOperationsService.resolveShopClosed(
                    ShopClosedResolveRequest(attemptId: attemptId, action: action),
                    idempotencyKey: "shop-closed-resolve:\(attemptId):\(action)"
                )
                statusMessage = "Resolved · \(action)"
                await load()
            } catch {
                statusMessage = error.localizedDescription
            }
        }
    }
}
