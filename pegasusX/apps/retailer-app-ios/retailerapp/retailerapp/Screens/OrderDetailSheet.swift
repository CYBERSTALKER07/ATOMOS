import SwiftUI

// MARK: - Order Detail Sheet (3/4 height)

struct OrderDetailSheet: View {
    let order: Order
    var onCancelled: (() -> Void)? = nil
    @Environment(\.dismiss) private var dismiss
    @State private var vm = OrderDetailViewModel()

    private var claimCandidate: Bool {
        vm.isClaimCandidate(order)
    }

    var body: some View {
        @Bindable var vm = vm
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: AppTheme.spacingLG) {
                    // Status header
                    HStack {
                        VStack(alignment: .leading, spacing: 4) {
                            Text(L10n.format("mobile_retailer.ui.order_suffix", "\(order.id.suffix(3))"))
                                .font(.system(.title3, design: .rounded, weight: .bold))
                                .foregroundStyle(AppTheme.textPrimary)
                            Text(order.status.displayName)
                                .font(.system(.subheadline, design: .rounded, weight: .medium))
                                .foregroundStyle(AppTheme.textSecondary)
                            if order.isMigrationOrigin {
                                Text("mobile_retailer.nasiya_opening.migration_badge")
                                    .font(.system(.caption2, design: .rounded, weight: .bold))
                                    .padding(.horizontal, 8)
                                    .padding(.vertical, 3)
                                    .background(AppTheme.warning.opacity(0.15))
                                    .foregroundStyle(AppTheme.warning)
                                    .clipShape(Capsule())
                            }
                            if order.autoConfirmOptIn {
                                Text("Auto-confirm opt-in enabled")
                                    .font(.system(.caption2, design: .rounded, weight: .semibold))
                                    .foregroundStyle(AppTheme.accent)
                            }
                        }
                        Spacer()
                        Text(order.displayTotal)
                            .font(.system(.title2, design: .rounded, weight: .bold))
                            .foregroundStyle(AppTheme.textPrimary)
                    }
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("Order \(order.id.suffix(3)), \(order.status.displayName), total \(order.displayTotal)")
                    .slideIn(delay: 0)

                    // ETA
                    if let eta = order.estimatedDelivery {
                        LabCard {
                            VStack(spacing: AppTheme.spacingSM) {
                                Text("mobile_retailer.ui.estimated_arrival")
                                    .font(.system(.caption, design: .rounded))
                                    .foregroundStyle(AppTheme.textTertiary)
                                CountdownText(targetISO: eta, font: .system(.title2, design: .monospaced, weight: .bold), color: AppTheme.textPrimary)
                            }
                            .frame(maxWidth: .infinity)
                            .padding(AppTheme.spacingLG)
                        }
                        .accessibilityElement(children: .combine)
                        .slideIn(delay: 0.05)
                    }

                    // Line items
                    LabCardWithHeader(title: "Items", subtitle: "\(order.itemCount) items", icon: "list.bullet") {
                        VStack(spacing: 0) {
                            ForEach(Array(order.items.enumerated()), id: \.element.id) { index, item in
                                if index > 0 {
                                    Rectangle().fill(AppTheme.separator.opacity(0.3)).frame(height: AppTheme.separatorHeight)
                                }
                                VStack(alignment: .leading, spacing: 6) {
                                    HStack {
                                        Text(item.productName)
                                            .font(.system(.subheadline, design: .rounded))
                                            .foregroundStyle(AppTheme.textPrimary)
                                        Spacer()
                                        Text(L10n.format("mobile_retailer.ui.quantity", "\(item.quantity)"))
                                            .font(.system(.caption, design: .rounded, weight: .bold))
                                            .foregroundStyle(AppTheme.textTertiary)
                                            .padding(.horizontal, 8).padding(.vertical, 3)
                                            .background(AppTheme.surfaceElevated)
                                            .clipShape(.capsule)
                                    }

                                    if item.mustHave || item.allowSubstitutes {
                                        HStack(spacing: 6) {
                                            if item.mustHave {
                                                Text("Must-have")
                                                    .font(.system(.caption2, design: .rounded, weight: .semibold))
                                                    .foregroundStyle(AppTheme.warning)
                                                    .padding(.horizontal, 6).padding(.vertical, 2)
                                                    .background(AppTheme.warning.opacity(0.14))
                                                    .clipShape(.capsule)
                                            }
                                            if item.allowSubstitutes {
                                                Text("Subs allowed")
                                                    .font(.system(.caption2, design: .rounded, weight: .semibold))
                                                    .foregroundStyle(AppTheme.accent)
                                                    .padding(.horizontal, 6).padding(.vertical, 2)
                                                    .background(AppTheme.accentSoft.opacity(0.35))
                                                    .clipShape(.capsule)
                                            }
                                        }
                                    }

                                    if item.substituteSku != nil || item.substituteName != nil || item.priceGapMinor != nil {
                                        VStack(alignment: .leading, spacing: 2) {
                                            Text("Substituted with \(item.substituteName ?? item.substituteSku ?? "alternative SKU")")
                                                .font(.system(.caption, design: .rounded, weight: .semibold))
                                                .foregroundStyle(AppTheme.info)
                                            if let orig = item.originalSku, !orig.isEmpty {
                                                Text("Original SKU: \(orig)")
                                                    .font(.system(.caption2, design: .rounded))
                                                    .foregroundStyle(AppTheme.textTertiary)
                                            }
                                            if let gapMinor = item.priceGapMinor {
                                                Text("Price gap: \(RetailerMoney.formatPriceGapSom(gapMinor))")
                                                    .font(.system(.caption, design: .rounded, weight: .bold))
                                                    .foregroundStyle(gapMinor > 0 ? AppTheme.warning : AppTheme.success)
                                            }
                                        }
                                        .padding(8)
                                        .frame(maxWidth: .infinity, alignment: .leading)
                                        .background(AppTheme.infoSoft.opacity(0.25))
                                        .clipShape(.rect(cornerRadius: AppTheme.radiusSM, style: .continuous))
                                    }

                                    if let shortageReason = item.shortageReason, !shortageReason.isEmpty {
                                        Text(shortageReason)
                                            .font(.system(.caption2, design: .rounded))
                                            .foregroundStyle(AppTheme.warning)
                                    }
                                }
                                .padding(.vertical, AppTheme.spacingSM)
                                .accessibilityElement(children: .combine)
                                .accessibilityLabel("\(item.productName), quantity \(item.quantity)")
                            }
                        }
                    }
                    .slideIn(delay: 0.1)

                    // Logistics fee
                    LabCard {
                        VStack(spacing: AppTheme.spacingMD) {
                            summaryRow("Subtotal", value: order.displayTotal)
                            summaryRow("Logistics Fee", value: "$2.50")
                            Rectangle().fill(AppTheme.separator.opacity(0.3)).frame(height: AppTheme.separatorHeight)
                            HStack {
                                Text("retailer_desktop.pos.text.total")
                                    .font(.system(.headline, design: .rounded))
                                    .foregroundStyle(AppTheme.textPrimary)
                                Spacer()
                                Text(order.displayTotal)
                                    .font(.system(.title3, design: .rounded, weight: .bold))
                                    .foregroundStyle(AppTheme.textPrimary)
                            }
                            .accessibilityElement(children: .combine)
                        }
                        .padding(AppTheme.spacingLG)
                    }
                    .slideIn(delay: 0.15)

                    // Show QR — JIT: only after dispatch
                    if order.status.hasDeliveryToken, let qrData = order.deliveryQRCodePayload {
                        VStack(spacing: AppTheme.spacingSM) {
                            QRCodeView(data: qrData, size: 180)
                            Text("mobile_retailer.ui.show_this_qr_code_to_the_driver")
                                .font(.system(.caption, design: .rounded))
                                .foregroundStyle(AppTheme.textTertiary)
                        }
                        .frame(maxWidth: .infinity)
                        .padding(AppTheme.spacingLG)
                        .background(AppTheme.cardBackground)
                        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
                        .shadow(color: AppTheme.shadowColor, radius: 4, y: 2)
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel("Delivery QR code ready for driver scan")
                        .slideIn(delay: 0.2)
                    } else if !order.status.hasDeliveryToken {
                        VStack(spacing: AppTheme.spacingSM) {
                            Image(systemName: "lock.fill")
                                .font(.title2)
                                .foregroundStyle(AppTheme.textTertiary)
                            Text("mobile_retailer.ui.awaiting_dispatch")
                                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                                .foregroundStyle(AppTheme.textSecondary)
                            Text("mobile_retailer.ui.qr_code_will_appear_when_your_order_is_on_the_way")
                                .font(.system(.caption, design: .rounded))
                                .foregroundStyle(AppTheme.textTertiary)
                                .multilineTextAlignment(.center)
                        }
                        .frame(maxWidth: .infinity)
                        .padding(AppTheme.spacingLG)
                        .background(AppTheme.surfaceElevated)
                        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
                        .accessibilityElement(children: .combine)
                        .slideIn(delay: 0.2)
                    }

                    OrderStatusHistorySection(orderId: order.id)
                        .slideIn(delay: 0.18)

                    // Post-delivery claims — gated by claim-eligibility countdown (G2)
                    if claimCandidate {
                        if let claimElig = vm.claimElig, claimElig.eligible {
                            Text(L10n.format("mobile_retailer.ui.eligible_until_formatclaimendsat_hoursremainingh_left", "\(formatClaimEndsAt(claimElig.endsAt))", "\(claimElig.hoursRemaining)"))
                                .font(.system(.caption, design: .rounded))
                                .foregroundStyle(AppTheme.textTertiary)
                            Button {
                                Haptics.light()
                                vm.showFileClaim = true
                            } label: {
                                HStack(spacing: AppTheme.spacingSM) {
                                    Image(systemName: "exclamationmark.bubble")
                                    Text("mobile_retailer.ui.report_damage_or_missing_items")
                                        .font(.system(.subheadline, design: .rounded, weight: .semibold))
                                }
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, AppTheme.spacingMD)
                                .foregroundStyle(AppTheme.textPrimary)
                                .overlay(
                                    RoundedRectangle(cornerRadius: AppTheme.radiusCard)
                                        .stroke(AppTheme.separator.opacity(0.5), lineWidth: 1)
                                )
                                .pegasusMinimumHitTarget()
                            }
                            .accessibilityLabel("Report damage or missing items")
                            .slideIn(delay: 0.22)
                        } else if let claimElig = vm.claimElig, !claimElig.eligible {
                            Text(
                                "Window closed — claim filing unavailable" +
                                    (claimElig.endsAt.map { " (ended \(formatClaimEndsAt($0)))" } ?? "")
                            )
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                            .slideIn(delay: 0.22)
                        } else {
                            Button {
                                Haptics.light()
                                vm.showFileClaim = true
                            } label: {
                                HStack(spacing: AppTheme.spacingSM) {
                                    Image(systemName: "exclamationmark.bubble")
                                    Text("mobile_retailer.ui.report_damage_or_missing_items")
                                        .font(.system(.subheadline, design: .rounded, weight: .semibold))
                                }
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, AppTheme.spacingMD)
                                .foregroundStyle(AppTheme.textPrimary)
                                .overlay(
                                    RoundedRectangle(cornerRadius: AppTheme.radiusCard)
                                        .stroke(AppTheme.separator.opacity(0.5), lineWidth: 1)
                                )
                                .pegasusMinimumHitTarget()
                            }
                            .accessibilityLabel("Report damage or missing items")
                            .slideIn(delay: 0.22)
                        }
                    }

                    // Cancel action — permitted for cancellable states
                    if order.status.canCancel {
                        Button {
                            Haptics.warning()
                            vm.showCancelConfirm = true
                        } label: {
                            HStack(spacing: AppTheme.spacingSM) {
                                if vm.isCancelling {
                                    ProgressView()
                                        .tint(AppTheme.destructive)
                                        .controlSize(.small)
                                } else {
                                    Image(systemName: "xmark.circle")
                                }
                                Text(vm.isCancelling ? "Cancelling\u{2026}" : "Cancel Order")
                                    .font(.system(.subheadline, design: .rounded, weight: .semibold))
                            }
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, AppTheme.spacingMD)
                            .foregroundStyle(AppTheme.destructive)
                            .overlay(
                                RoundedRectangle(cornerRadius: AppTheme.radiusCard)
                                    .stroke(AppTheme.destructive.opacity(0.4), lineWidth: 1)
                            )
                            .pegasusMinimumHitTarget()
                        }
                        .disabled(vm.isCancelling)
                        .accessibilityLabel(vm.isCancelling ? "Cancelling order" : "Cancel Order")
                        .slideIn(delay: 0.25)
                    } else if order.status != .cancelled && order.status != .completed {
                        HStack(spacing: AppTheme.spacingSM) {
                            Image(systemName: "exclamationmark.triangle")
                                .font(.system(.caption))
                            Text("mobile_retailer.ui.order_in_progress_cannot_be_cancelled")
                                .font(.system(.caption, design: .rounded))
                        }
                        .foregroundStyle(AppTheme.textTertiary)
                        .frame(maxWidth: .infinity, alignment: .center)
                        .padding(.vertical, AppTheme.spacingSM)
                        .accessibilityElement(children: .combine)
                        .slideIn(delay: 0.25)
                    }
                }
                .padding(AppTheme.spacingLG)
                .padding(.bottom, AppTheme.spacingXXL)
            }
            .scrollDismissesKeyboard(.interactively)
            .scrollIndicators(.hidden)
            .background(AppTheme.background)
            .navigationTitle("mobile_retailer.ui.order_details")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("warehouse_portal.kpi_stat_card.text.done") { dismiss() }
                        .font(.system(.subheadline, design: .rounded, weight: .semibold))
                        .pegasusMinimumHitTarget()
                }
            }
            .alert("Cancel Order", isPresented: $vm.showCancelConfirm) {
                Button("mobile_retailer.ui.cancel_order", role: .destructive) {
                    Task {
                        if await vm.cancelOrder(order) {
                            onCancelled?()
                            dismiss()
                        }
                    }
                }
                Button("mobile_retailer.ui.keep_order", role: .cancel) {}
            } message: {
                Text(L10n.format("mobile_retailer.ui.cancel_order_suffix_this_cannot_be_undone", "\(order.id.suffix(3))"))
            }
            .alert("Cancellation Failed", isPresented: $vm.cancelError) {
                Button("OK", role: .cancel) {}
            } message: {
                Text("mobile_retailer.ui.could_not_cancel_the_order_please_try_again")
            }
            .sheet(isPresented: $vm.showFileClaim) {
                FileClaimView(order: order)
            }
            .task(id: order.id) {
                await vm.loadClaimEligibility(for: order)
            }
        }
    }

    private static let claimIsoFormatter = ISO8601DateFormatter()

    private func formatClaimEndsAt(_ raw: String?) -> String {
        guard let raw, let date = Self.claimIsoFormatter.date(from: raw) else {
            return raw ?? "window end"
        }
        return date.formatted(date: .abbreviated, time: .shortened)
    }

    private func summaryRow(_ title: String, value: String) -> some View {
        HStack {
            Text(title)
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
            Spacer()
            Text(value)
                .font(.system(.subheadline, design: .rounded, weight: .medium))
                .foregroundStyle(AppTheme.textPrimary)
        }
        .accessibilityElement(children: .combine)
    }
}

// MARK: - QR Overlay (Full Screen Blurred)

struct QROverlay: View {
    let order: Order
    let onDismiss: () -> Void

    @State private var appeared = false

    var body: some View {
        ZStack {
            // Blurred background
            Color.clear
                .background(.ultraThinMaterial)
                .ignoresSafeArea()
                .onTapGesture { onDismiss() }

            // QR content
            VStack(spacing: AppTheme.spacingXL) {
                Text(L10n.format("mobile_retailer.ui.order_suffix", "\(order.id.suffix(3))"))
                    .font(.system(.headline, design: .rounded))
                    .foregroundStyle(AppTheme.textPrimary)

                if let qrData = order.deliveryQRCodePayload {
                    QRCodeView(data: qrData, size: 240)
                        .padding(AppTheme.spacingXL)
                        .background(AppTheme.cardBackground)
                        .clipShape(.rect(cornerRadius: AppTheme.radiusXL))
                        .shadow(color: AppTheme.shadowColor, radius: 16, y: 8)
                } else {
                    // No token yet — locked state
                    VStack(spacing: AppTheme.spacingSM) {
                        Image(systemName: "lock.fill")
                            .font(.largeTitle)
                            .foregroundStyle(AppTheme.textTertiary)
                        Text("mobile_retailer.ui.token_not_yet_generated")
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                    }
                    .frame(width: 240, height: 240)
                    .padding(AppTheme.spacingXL)
                    .background(AppTheme.cardBackground)
                    .clipShape(.rect(cornerRadius: AppTheme.radiusXL))
                    .shadow(color: AppTheme.shadowColor, radius: 16, y: 8)
                }

                Text("mobile_retailer.ui.show_to_driver_for_delivery_confirmation")
                    .font(.system(.subheadline, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)

                Button {
                    Haptics.light()
                    onDismiss()
                } label: {
                    Text("factory_portal.toast.text.dismiss")
                        .font(.system(.subheadline, design: .rounded, weight: .semibold))
                        .foregroundStyle(AppTheme.textSecondary)
                        .padding(.horizontal, AppTheme.spacingXL)
                        .padding(.vertical, AppTheme.spacingMD)
                        .background(AppTheme.surfaceElevated)
                        .clipShape(.capsule)
                        .pegasusMinimumHitTarget()
                }
                .accessibilityLabel("Dismiss QR code overlay")
            }
            .scaleEffect(appeared ? 1 : 0.85)
            .opacity(appeared ? 1 : 0)
        }
        .onAppear {
            withAnimation(AnimationConstants.fluid) { appeared = true }
        }
    }
}
