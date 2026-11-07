//
//  SupplyTransfersView.swift
//  driverappios
//

import CoreLocation
import SwiftUI

@MainActor
@Observable
final class SupplyTransfersViewModel {
    var transfers: [SupplyTransferRow] = []
    var isLoading = false
    var isArriving: String?
    var errorMessage: String?
    var successMessage: String?

    private let arriveableStates: Set<String> = ["IN_TRANSIT", "IN_TRANSIT_TO_WAREHOUSE", "DISPATCHED"]

    var activeCount: Int {
        transfers.filter { $0.state.uppercased() != "ARRIVED" }.count
    }

    func refresh() async {
        isLoading = true
        errorMessage = nil
        defer { isLoading = false }
        do {
            let response = try await APIClient.shared.getSupplyTransfers()
            transfers = response.transfers
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func canArrive(_ transfer: SupplyTransferRow) -> Bool {
        arriveableStates.contains(transfer.state.uppercased())
    }

    func markArrived(_ transfer: SupplyTransferRow) async {
        isArriving = transfer.transferId
        errorMessage = nil
        successMessage = nil
        defer { isArriving = nil }

        let coordinate = await currentCoordinate()
        do {
            let response = try await APIClient.shared.arriveSupplyTransfer(
                transferId: transfer.transferId,
                latitude: coordinate.latitude,
                longitude: coordinate.longitude
            )
            successMessage = "Transfer \(response.transferId.suffix(6)) marked arrived"
            await refresh()
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func currentCoordinate() async -> CLLocationCoordinate2D {
        await withCheckedContinuation { continuation in
            let manager = CLLocationManager()
            manager.requestWhenInUseAuthorization()
            if let location = manager.location {
                continuation.resume(returning: location.coordinate)
            } else {
                continuation.resume(returning: CLLocationCoordinate2D(latitude: 0, longitude: 0))
            }
        }
    }
}

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
                            .font(.system(size: 12, weight: .medium))
                            .foregroundStyle(LabTheme.destructive)
                    }
                    if let success = vm.successMessage {
                        Text(success)
                            .font(.system(size: 12, weight: .medium))
                            .foregroundStyle(LabTheme.success)
                    }
                    if !vm.transfers.isEmpty {
                        Text(L10n.format("mobile_driver.ui.activecount_active_count_total", "\(vm.activeCount)", "\(vm.transfers.count)"))
                            .font(.system(size: 11, weight: .semibold, design: .monospaced))
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
                    .font(.system(size: 13, weight: .bold, design: .monospaced))
                Spacer()
                Text(transfer.state.replacingOccurrences(of: "_", with: " "))
                    .font(.system(size: 10, weight: .bold))
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(statusTint.opacity(0.15), in: Capsule())
                    .foregroundStyle(statusTint)
            }
            Text(L10n.format("mobile_driver.ui.warehouse_suffix", "\(transfer.warehouseId.suffix(6))"))
                .font(.system(size: 13, weight: .medium))
                .foregroundStyle(LabTheme.fgSecondary)
            if let supplyId = transfer.supplyRequestId, !supplyId.isEmpty {
                Text(L10n.format("mobile_driver.ui.supply_suffix", "\(supplyId.suffix(8))"))
                    .font(.system(size: 12, weight: .medium, design: .monospaced))
                    .foregroundStyle(LabTheme.fgTertiary)
            }
            Text(String(format: "Volume %.1f VU", transfer.totalVolumeVu))
                .font(.system(size: 12, weight: .medium))
                .foregroundStyle(LabTheme.fgTertiary)

            if canArrive {
                Button(action: onArrive) {
                    HStack {
                        if isArriving {
                            ProgressView()
                        }
                        Text("mobile_driver.ui.mark_arrived_at_warehouse")
                            .font(.system(size: 14, weight: .bold))
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 12)
                }
                .buttonStyle(.borderedProminent)
                .disabled(isArriving)
            }
        }
        .padding(.vertical, 8)
    }

    private var statusTint: Color {
        let state = transfer.state.uppercased()
        if state == "ARRIVED" { return LabTheme.success }
        if canArrive { return .orange }
        return LabTheme.fgTertiary
    }
}
