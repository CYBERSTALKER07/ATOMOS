import SwiftUI

struct ShopClosedView: View {
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var viewModel = ShopClosedViewModel()

    var body: some View {
        Group {
            if viewModel.loading && viewModel.rows.isEmpty {
                SupplierLoadingView(title: "mobile_supplier.loading.shop_closed")
            } else if let error = viewModel.error, viewModel.rows.isEmpty {
                SupplierErrorView(message: error) { Task { await viewModel.load() } }
            } else if viewModel.rows.isEmpty {
                SupplierEmptyView(title: "mobile_supplier.empty.no_active_attempts", message: "mobile_supplier.empty.shop_closed_message")
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(viewModel.rows) { row in
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
                                Button("mobile_supplier.ui.wait") {
                                    Task { await viewModel.resolve(row.attemptId, action: "WAIT") }
                                }
                                .frame(minHeight: 44)
                                .disabled(viewModel.busyId == row.attemptId)
                                Button("mobile_supplier.ui.bypass") {
                                    Task { await viewModel.resolve(row.attemptId, action: "BYPASS") }
                                }
                                .frame(minHeight: 44)
                                .disabled(viewModel.busyId == row.attemptId)
                                Button("mobile_supplier.ui.return") {
                                    Task { await viewModel.resolve(row.attemptId, action: "RETURN_TO_DEPOT") }
                                }
                                .frame(minHeight: 44)
                                .disabled(viewModel.busyId == row.attemptId)
                            }
                            .buttonStyle(.bordered)
                        }
                    }
                }
            }
        }
        .navigationTitle("mobile_supplier.ui.shop_closed")
        .task { await viewModel.load() }
        .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
            Task { await viewModel.handleReconnect() }
        }
        .safeAreaInset(edge: .bottom) {
            if let statusMessage = viewModel.statusMessage {
                Text(statusMessage)
                    .font(.footnote)
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 8)
                    .background(.bar)
            }
        }
    }
}
