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
                            Text("MANIFEST ITEMS")
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
            Button("Cancel", role: .cancel) { }
            Button("Confirm") {
                Task {
                    let success = await vm.startTransitForPartialOrder(orderId: orderId)
                    if success { onAmended() }
                }
            }
        } message: {
            Text("Notify the other driver that you are heading to this route?")
        }
        .alert("Confirm Amendment", isPresented: $showConfirmAlert) {
            Button("Cancel", role: .cancel) { }
            Button("Submit", role: .destructive) {
                Task {
                    let success = await vm.submitAmendment(orderId: orderId, driverId: driverId)
                    if success { onAmended() }
                }
            }
        } message: {
            Text("\(vm.rejectedCount) item(s) rejected. Refund: \(vm.refundDelta.formattedAmount). Proceed?")
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
