import SwiftUI

struct OrderChecklistSection: View {
    @Bindable var viewModel: HomeViewModel
    let onShowException: (String) -> Void
    let onShowReDispatch: (String) -> Void
    let onScanProduct: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            PayloadSectionHeader(
                title: "ORDER CHECKLIST",
                trailing: "\(viewModel.sealedOrderIds.count) / \(viewModel.orders.count) SEALED",
                trailingTint: viewModel.allOrdersSealed ? TermTheme.live : TermTheme.accent
            )
            .padding(.horizontal, 4)

            if viewModel.loadingOrders {
                PayloadLoadingView(
                    title: "FETCHING_MANIFEST",
                    message: "Loading the checklist items assigned to this vehicle."
                )
            } else if viewModel.orders.isEmpty {
                PayloadStateView(
                    variant: .manifest,
                    title: "NO_ORDERS_ASSIGNED",
                    message: "This truck has no checklist items waiting to load.",
                    compact: true
                )
                .frame(maxWidth: .infinity, alignment: .center)
                .padding(32)
                .tacticalCard()
            } else {
                VStack(spacing: 8) {
                    ForEach(viewModel.orders) { order in
                        OrderChip(
                            order: order,
                            selected: order.orderId == viewModel.selectedOrderId,
                            sealed: viewModel.sealedOrderIds.contains(order.orderId),
                            dispatchCode: viewModel.dispatchCodes[order.orderId],
                            onTap: { viewModel.selectOrder(order.orderId) }
                        )
                    }
                }

                if let selected = viewModel.orders.first(where: { $0.orderId == viewModel.selectedOrderId }) {
                    VStack(alignment: .leading, spacing: 12) {
                        HStack {
                            Text("LINE_ITEMS")
                                .font(.system(.caption, design: .monospaced, weight: .black))
                                .foregroundStyle(TermTheme.secondary)
                            Spacer()
                            Button {
                                PegasusHaptics.light()
                                onScanProduct()
                            } label: {
                                Label("SCAN", systemImage: "barcode.viewfinder")
                                    .font(.system(.caption, design: .monospaced, weight: .bold))
                                    .frame(minWidth: 44, minHeight: 44)
                            }
                            .buttonStyle(.bordered)
                            .accessibilityLabel("Scan product barcode")

                            Text(L10n.format("mobile_payload.ui.ord_uppercased", "\(selected.orderId.suffix(6).uppercased())"))
                                .font(.system(.caption, design: .monospaced, weight: .black))
                                .foregroundStyle(TermTheme.accent)
                        }
                        .padding(.top, 8)

                        let items = selected.items ?? []
                        if items.isEmpty {
                            Text("NO_ITEMS_IN_ORDER")
                                .font(.system(.caption, design: .monospaced, weight: .bold))
                                .foregroundStyle(TermTheme.tertiary)
                        } else {
                            VStack(spacing: 2) {
                                ForEach(items) { item in
                                    ItemRow(
                                        checked: viewModel.checkedItems.contains(item.lineItemId),
                                        enabled: !viewModel.sealedOrderIds.contains(selected.orderId),
                                        label: item.skuName.isEmpty ? item.skuId : item.skuName,
                                        quantity: item.quantity,
                                        onToggle: { viewModel.toggleItem(item.lineItemId) }
                                    )
                                }
                            }
                        }

                        if viewModel.sealedOrderIds.contains(selected.orderId) {
                            HStack {
                                Image(systemName: "lock.fill")
                                Text(L10n.format("mobile_payload.ui.order_sealed_orderid", "\(viewModel.dispatchCodes[selected.orderId] ?? "")"))
                            }
                            .font(.system(.subheadline, design: .monospaced, weight: .black))
                            .foregroundStyle(TermTheme.live)
                            .padding()
                            .frame(maxWidth: .infinity, minHeight: 44)
                            .background(TermTheme.live.opacity(0.1), in: RoundedRectangle(cornerRadius: 12, style: .continuous))
                            .accessibilityElement(children: .combine)
                        } else {
                            Button {
                                PegasusHaptics.heavy()
                                Task { await viewModel.sealSelectedOrder() }
                            } label: {
                                HStack(spacing: 12) {
                                    if viewModel.sealingOrderId == selected.orderId {
                                        ProgressView().controlSize(.regular)
                                    } else {
                                        Image(systemName: "lock.shield.fill")
                                            .font(.system(.title3))
                                        Text("SEAL_ORDER")
                                            .font(.system(.callout, design: .monospaced, weight: .black))
                                    }
                                }
                                .padding()
                                .frame(maxWidth: .infinity, minHeight: 48)
                                .background(viewModel.canSealOrder(selected.orderId) ? TermTheme.accent : TermTheme.tertiary.opacity(0.3))
                                .foregroundStyle(TermTheme.card)
                                .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
                            }
                            .disabled(viewModel.sealingOrderId != nil || !viewModel.canSealOrder(selected.orderId))
                            .buttonStyle(.tactical)
                            .accessibilityLabel("Seal selected order")

                            HStack(spacing: 12) {
                                Button("REPORT_ISSUE") {
                                    PegasusHaptics.warning()
                                    onShowException(selected.orderId)
                                }
                                .font(.system(.caption, design: .monospaced, weight: .bold))
                                .frame(maxWidth: .infinity, minHeight: 44)
                                .background(TermTheme.warn.opacity(0.1), in: RoundedRectangle(cornerRadius: 12, style: .continuous))
                                .foregroundStyle(TermTheme.warn)
                                .buttonStyle(.tactical)
                                .accessibilityLabel("Report issue on order \(selected.orderId.suffix(6))")

                                Button("RE_DISPATCH") {
                                    PegasusHaptics.medium()
                                    onShowReDispatch(selected.orderId)
                                }
                                .font(.system(.caption, design: .monospaced, weight: .bold))
                                .frame(maxWidth: .infinity, minHeight: 44)
                                .background(TermTheme.accent.opacity(0.1), in: RoundedRectangle(cornerRadius: 12, style: .continuous))
                                .foregroundStyle(TermTheme.accent)
                                .buttonStyle(.tactical)
                                .accessibilityLabel("Re-dispatch order \(selected.orderId.suffix(6))")
                            }
                        }
                    }
                }
            }
        }
    }
}
