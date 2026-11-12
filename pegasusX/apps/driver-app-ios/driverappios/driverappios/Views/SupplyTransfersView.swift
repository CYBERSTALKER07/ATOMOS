//
//  SupplyTransfersView.swift
//  driverappios
//

import CoreLocation
import SwiftUI

struct SupplyTransfersView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var vm = SupplyTransfersViewModel()

    var body: some View {
        NavigationStack {
            Group {
                if vm.isLoading && vm.transfers.isEmpty {
                    ProgressView("Loading transfers…")
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if vm.transfers.isEmpty {
                    DriverEmptyView(
                        icon: "shippingbox.fill",
                        title: "No supply transfers",
                        message: "Assigned factory→warehouse legs will appear here."
                    )
                } else {
                    List(vm.transfers) { transfer in
                        SupplyTransferRowView(
                            transfer: transfer,
                            canArrive: vm.canArrive(transfer),
                            isArriving: vm.isArriving == transfer.transferId,
                            onArrive: {
                                Task { await vm.markArrived(transfer) }
                            }
                        )
                    }
                    .listStyle(.plain)
                }
            }
            .navigationTitle("mobile_driver.ui.supply_transfers")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("common.action.close") { dismiss() }
                }
                ToolbarItem(placement: .primaryAction) {
                    Button("portal.page.orders.action.refresh") {
                        Task { await vm.refresh() }
                    }
                }
            }
            .safeAreaInset(edge: .top) {
                VStack(spacing: 8) {
                    if let error = vm.errorMessage {
                        Text(error)
                            .font(.system(.caption, weight: .medium))
                            .foregroundStyle(LabTheme.destructive)
                    }
                    if let success = vm.successMessage {
                        Text(success)
                            .font(.system(.caption, weight: .medium))
                            .foregroundStyle(LabTheme.success)
                    }
                    if !vm.transfers.isEmpty {
                        Text(L10n.format("mobile_driver.ui.activecount_active_count_total", "\(vm.activeCount)", "\(vm.transfers.count)"))
                            .font(.system(.caption2, design: .monospaced, weight: .semibold))
                            .foregroundStyle(LabTheme.fgTertiary)
                    }
                }
                .padding(.horizontal, LabTheme.s16)
                .padding(.top, 8)
            }
            .task {
                await vm.refresh()
            }
        }
    }
}

private struct SupplyTransferRowView: View {
    let transfer: SupplyTransferRow
    let canArrive: Bool
    let isArriving: Bool
    let onArrive: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text(String(transfer.transferId.suffix(8)).uppercased())
                    .font(.system(.footnote, design: .monospaced, weight: .bold))
                Spacer()
                Text(transfer.state.replacingOccurrences(of: "_", with: " "))
                    .font(.system(.caption2, weight: .bold))
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(statusTint.opacity(0.15), in: Capsule())
                    .foregroundStyle(statusTint)
            }
            Text(L10n.format("mobile_driver.ui.warehouse_suffix", "\(transfer.warehouseId.suffix(6))"))
                .font(.system(.footnote, weight: .medium))
                .foregroundStyle(LabTheme.fgSecondary)
            if let supplyId = transfer.supplyRequestId, !supplyId.isEmpty {
                Text(L10n.format("mobile_driver.ui.supply_suffix", "\(supplyId.suffix(8))"))
                    .font(.system(.caption, design: .monospaced, weight: .medium))
                    .foregroundStyle(LabTheme.fgTertiary)
            }
            Text(String(format: "Volume %.1f VU", transfer.totalVolumeVu))
                .font(.system(.caption, weight: .medium))
                .foregroundStyle(LabTheme.fgTertiary)

            if canArrive {
                Button(action: onArrive) {
                    HStack {
                        if isArriving {
                            ProgressView()
                        }
                        Text("mobile_driver.ui.mark_arrived_at_warehouse")
                            .font(.system(.subheadline, weight: .bold))
                    }
                    .frame(maxWidth: .infinity, minHeight: 44)
                }
                .buttonStyle(.borderedProminent)
                .disabled(isArriving)
            }
        }
        .padding(.vertical, 8)
        .accessibilityElement(children: .combine)
    }

    private var statusTint: Color {
        let state = transfer.state.uppercased()
        if state == "ARRIVED" { return LabTheme.success }
        if canArrive { return .orange }
        return LabTheme.fgTertiary
    }
}
