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
                ResponsiveGridContentWrapper {
                    ForEach(rows) { row in
                        VStack(alignment: .leading, spacing: 8) {
                            Text(row.orderId).font(.headline)
                            Text(L10n.format("mobile_supplier.ui.driver_driverid_retailer_retailerid", "\(row.driverId)", "\(row.retailerId)")).font(.caption)
                            if let reason = row.shopClosedReason, !reason.isEmpty {
                                Text(L10n.format("mobile_supplier.ui.reason_reason_2", "\(reason)")).font(.caption2).foregroundStyle(.secondary)
                            }
                            if let grace = row.graceEndsAt, !grace.isEmpty {
                                Text(L10n.format("mobile_supplier.ui.grace_ends_grace_2", "\(grace)")).font(.caption2).foregroundStyle(.secondary)
                            }
                            if let res = row.shopClosedResolution, !res.isEmpty {
                                Text(L10n.format("mobile_supplier.ui.resolution_res_2", "\(res)")).font(.caption2)
                            }
                            HStack {
                                Button("mobile_supplier.ui.wait") { resolve(row.attemptId, action: "WAIT") }
                                    .disabled(busyId == row.attemptId)
                                Button("mobile_supplier.ui.bypass") { resolve(row.attemptId, action: "BYPASS") }
                                    .disabled(busyId == row.attemptId)
                                Button("mobile_supplier.ui.return") { resolve(row.attemptId, action: "RETURN_TO_DEPOT") }
                                    .disabled(busyId == row.attemptId)
                            }
                            .buttonStyle(.bordered)
                        }
                    }
                }
            }
        }
        .navigationTitle("mobile_supplier.ui.shop_closed")
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
