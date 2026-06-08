//
//  DeliveryCorrectionView.swift
//  driverappios
//

import SwiftUI

struct DeliveryCorrectionView: View {

    let orderId: String
    let driverId: String
    let onClose: () -> Void
    let onAmended: () -> Void

    @State private var vm = CorrectionViewModel()
    @State private var showConfirmAlert = false

    var body: some View {
        ZStack(alignment: .bottom) {
            VStack(alignment: .leading, spacing: 0) {
                // MARK: - Header
                headerView
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
                                lineItemCard(item)
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
                summaryBar
            }
        }
        .background(LabTheme.bg)
        .task {
            await vm.loadLineItems(orderId: orderId)
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

    // MARK: - Header

    private var headerView: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Button {
                    Haptics.light()
                    onClose()
                } label: {
                    HStack(spacing: 4) {
                        Image(systemName: "chevron.left")
                        Text("Back")
                    }
                    .font(.body.weight(.semibold))
                    .foregroundStyle(LabTheme.fg)
                }

                Spacer()

                StatusPill(
                    label: vm.hasRejections ? "\(vm.rejectedCount) REJECTED" : "ALL CLEAR",
                    color: vm.hasRejections ? LabTheme.warning : LabTheme.success
                )
            }

            Text("Delivery Correction")
                .font(.system(size: 22, weight: .bold))
                .foregroundStyle(LabTheme.fg)

            Text(orderId)
                .font(.system(size: 13, weight: .semibold, design: .monospaced))
                .foregroundStyle(LabTheme.fgSecondary)
        }
    }

    // MARK: - Line Item Card

    private func lineItemCard(_ item: LineItem) -> some View {
        let isRejected = item.status == .REJECTED_DAMAGED

        return VStack(alignment: .leading, spacing: 10) {
            HStack {
                VStack(alignment: .leading, spacing: 4) {
                    Text(item.sku_id)
                        .font(.system(size: 15, weight: .bold))
                        .strikethrough(isRejected)
                        .foregroundStyle(isRejected ? LabTheme.destructive : LabTheme.fg)

                    Text("\(item.quantity) × \(item.unit_price.formattedAmount)")
                        .font(.system(size: 13, weight: .medium))
                        .foregroundStyle(LabTheme.fgSecondary)
                }

                Spacer()

                Text(isRejected ? "Rejected" : "Delivered")
                    .font(.system(size: 11, weight: .bold))
                    .foregroundStyle(.white)
                    .padding(.horizontal, 12)
                    .padding(.vertical, 6)
                    .background(
                        isRejected ? LabTheme.destructive : LabTheme.fg,
                        in: Capsule()
                    )
            }

            Rectangle()
                .fill(LabTheme.separator)
                .frame(height: 0.5)

            HStack {
                Text("Line total")
                    .font(.caption)
                    .foregroundStyle(LabTheme.fgTertiary)
                Spacer()
                Text(item.lineTotal.formattedAmount)
                    .font(.system(size: 14, weight: .bold))
                    .strikethrough(isRejected)
                    .foregroundStyle(isRejected ? LabTheme.destructive.opacity(0.6) : LabTheme.fg)
            }

            if isRejected {
                HStack {
                    Text("Reason")
                        .font(.caption)
                        .foregroundStyle(LabTheme.fgTertiary)
                    Spacer()
                    Menu {
                        ForEach(RejectionReason.allCases, id: \.self) { reason in
                            Button {
                                vm.setReason(reason, for: item.id)
                            } label: {
                                if vm.reason(for: item.id) == reason {
                                    Label(reasonLabel(for: reason), systemImage: "checkmark")
                                } else {
                                    Text(reasonLabel(for: reason))
                                }
                            }
                        }
                    } label: {
                        HStack(spacing: 6) {
                            Text(reasonLabel(for: vm.reason(for: item.id)))
                            Image(systemName: "chevron.down")
                                .font(.system(size: 11, weight: .bold))
                        }
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(LabTheme.fg)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 8)
                        .background(LabTheme.fg.opacity(0.06), in: Capsule())
                    }
                }
            }

            RoundedRectangle(cornerRadius: 2)
                .fill(isRejected ? LabTheme.destructive : LabTheme.fg.opacity(0.15))
                .frame(height: 3)
        }
        .padding(LabTheme.s16)
        .labCard()
        .contentShape(Rectangle())
        .onTapGesture {
            vm.toggleStatus(for: item.id)
        }
    }

    private func reasonLabel(for reason: RejectionReason) -> String {
        reason.rawValue.replacingOccurrences(of: "_", with: " ").capitalized
    }

    // MARK: - Summary Bar

    private var summaryBar: some View {
        VStack(spacing: 10) {
            // Original total
            HStack {
                Text("Original total")
                    .font(.subheadline)
                    .foregroundStyle(LabTheme.fgSecondary)
                Spacer()
                Text(vm.originalTotal.formattedAmount)
                    .font(.subheadline.weight(.medium))
                    .foregroundStyle(LabTheme.fg)
            }

            // Refund delta
            if vm.refundDelta > 0 {
                HStack {
                    Text("Refund delta")
                        .font(.subheadline)
                        .foregroundStyle(LabTheme.destructive)
                    Spacer()
                    Text("−\(vm.refundDelta.formattedAmount)")
                        .font(.subheadline.weight(.bold))
                        .foregroundStyle(LabTheme.destructive)
                }
            }

            Rectangle()
                .fill(LabTheme.separator)
                .frame(height: 0.5)

            // Adjusted total
            HStack {
                Text("Adjusted total")
                    .font(.headline)
                    .foregroundStyle(LabTheme.fg)
                Spacer()
                Text(vm.adjustedTotal.formattedAmount)
                    .font(.headline)
                    .foregroundStyle(LabTheme.fg)
            }

            // Submit button
            Button {
                if vm.hasRejections { showConfirmAlert = true }
            } label: {
                Text(vm.hasRejections
                     ? "Submit Amendment (\(vm.rejectedCount) rejected)"
                     : "All Items Delivered")
                    .font(.system(size: 15, weight: .bold))
                    .foregroundStyle(vm.hasRejections ? .white : LabTheme.fgTertiary)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 16)
                    .background(
                        vm.hasRejections ? LabTheme.destructive : LabTheme.fg.opacity(0.06),
                        in: .rect(cornerRadius: LabTheme.buttonRadius)
                    )
            }
            .buttonStyle(.pressable)
            .disabled(!vm.hasRejections)
        }
        .padding(LabTheme.s16)
        .background(.ultraThinMaterial)
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
