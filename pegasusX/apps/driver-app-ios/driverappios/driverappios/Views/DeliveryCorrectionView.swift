//
//  DeliveryCorrectionView.swift
//  driverappios
//

import PhotosUI
import SwiftUI
import UIKit

struct DeliveryCorrectionView: View {

    let orderId: String
    let driverId: String
    var isPartial: Bool = false
    let onClose: () -> Void
    let onAmended: () -> Void

    @State private var vm = CorrectionViewModel()
    @State private var showConfirmAlert = false
    @State private var showStartTransitAlert = false
    @State private var pickedPhoto: PhotosPickerItem?

    var body: some View {
        ZStack(alignment: .bottom) {
            VStack(alignment: .leading, spacing: 0) {
                // MARK: - Header
                DeliveryCorrectionHeaderView(
                    orderId: orderId,
                    isPartial: isPartial,
                    hasRejections: vm.hasRejections,
                    rejectedCount: vm.rejectedCount,
                    onClose: onClose,
                    showStartTransitAlert: $showStartTransitAlert
                )
                .padding(.horizontal, LabTheme.s24)
                .padding(.top, LabTheme.s24)
                .padding(.bottom, LabTheme.s16)

                if vm.isLoading {
                    Spacer()
                    ProgressView("Loading line items...")
                        .tint(LabTheme.fg)
                        .frame(maxWidth: .infinity)
                    Spacer()
                } else {
                    // MARK: - Line Items
                    ScrollView {
                        VStack(alignment: .leading, spacing: 0) {
                            // Section header
                            Text("mobile_driver.ui.manifest_items")
                                .font(.system(size: 10, weight: .bold, design: .monospaced))
                                .foregroundStyle(LabTheme.fgTertiary)
                                .padding(.horizontal, LabTheme.s16)
                                .padding(.bottom, 12)

                            ForEach(Array(vm.lineItems.enumerated()), id: \.element.id) { index, item in
                                DeliveryCorrectionLineItemCard(
                                    item: item,
                                    currentReason: vm.reason(for: item.id),
                                    onToggleStatus: { vm.toggleStatus(for: item.id) },
                                    onSetReason: { reason in vm.setReason(reason, for: item.id) }
                                )
                                .padding(.horizontal, LabTheme.s16)
                                .padding(.bottom, 10)
                                .staggeredAppear(index: index)
                            }
                        }
                        .padding(.bottom, 200) // Space for summary overlay
                    }
                }
            }

            // MARK: - Bottom Summary Bar
            if !vm.isLoading {
                DeliveryCorrectionSummaryBar(
                    vm: vm,
                    pickedPhoto: $pickedPhoto,
                    showConfirmAlert: $showConfirmAlert
                )
            }
        }
        .background(LabTheme.bg)
        .task {
            await vm.loadLineItems(orderId: orderId)
        }
        .onChange(of: pickedPhoto) { _, item in
            guard let item else { return }
            Task {
                guard let data = try? await item.loadTransferable(type: Data.self),
                      let image = UIImage(data: data) else {
                    vm.submitError = "Could not read photo."
                    return
                }
                await vm.uploadEvidence(image: image, orderId: orderId)
            }
        }
        .alert("Start Transit?", isPresented: $showStartTransitAlert) {
            Button("common.action.cancel", role: .cancel) { }
            Button("mobile_driver.ui.confirm") {
                Task {
                    let success = await vm.startTransitForPartialOrder(orderId: orderId)
                    if success { onAmended() }
                }
            }
        } message: {
            Text("mobile_driver.ui.notify_the_other_driver_that_you_are_heading_to_this_route")
        }
        .alert("Confirm Amendment", isPresented: $showConfirmAlert) {
            Button("common.action.cancel", role: .cancel) { }
            Button("warehouse_portal.cycle_counts.text.submit", role: .destructive) {
                Task {
                    let success = await vm.submitAmendment(orderId: orderId, driverId: driverId)
                    if success { onAmended() }
                }
            }
        } message: {
            Text(L10n.format("mobile_driver.ui.rejectedcount_item_s_rejected_refund_formattedamount_proceed", "\(vm.rejectedCount)", "\(vm.refundDelta.formattedAmount)"))
        }
    }


}

#Preview {
    DeliveryCorrectionView(
        orderId: "ORD-TASH-0056",
        driverId: "DRV-AMIR-001",
        onClose: {},
        onAmended: {}
    )
}
