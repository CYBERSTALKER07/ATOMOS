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
                                .font(.system(size: 12, weight: .black, design: .monospaced))
                                .foregroundStyle(TermTheme.secondary)
                            Spacer()
                            Button(action: onScanProduct) {
                                Label("SCAN", systemImage: "barcode.viewfinder")
                                    .font(.system(size: 11, weight: .bold, design: .monospaced))
                            }
                            .buttonStyle(.bordered)
                            Text(L10n.format("mobile_payload.ui.ord_uppercased", "\(selected.orderId.suffix(6).uppercased())"))
                                .font(.system(size: 12, weight: .black, design: .monospaced))
                                .foregroundStyle(TermTheme.accent)
                        }
                        .padding(.top, 8)
                        
                        let items = selected.items ?? []
                        if items.isEmpty {
                            Text("NO_ITEMS_IN_ORDER")
                                .font(.system(size: 12, weight: .bold, design: .monospaced))
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
                            .font(.system(size: 14, weight: .black, design: .monospaced))
                            .foregroundStyle(TermTheme.live)
                            .padding()
                            .frame(maxWidth: .infinity)
                            .background(TermTheme.live.opacity(0.1), in: RoundedRectangle(cornerRadius: 12))
                        } else {
                            Button {
                                Task { await viewModel.sealSelectedOrder() }
                            } label: {
                                HStack(spacing: 12) {
                                    if viewModel.sealingOrderId == selected.orderId {
                                        ProgressView().controlSize(.regular)
                                    } else {
                                        Image(systemName: "lock.shield.fill")
                                            .font(.system(size: 20))
                                        Text("SEAL_ORDER")
                                            .font(.system(size: 16, weight: .black, design: .monospaced))
                                    }
                                }
                                .padding()
                                .frame(maxWidth: .infinity)
                                .background(viewModel.canSealOrder(selected.orderId) ? TermTheme.accent : TermTheme.tertiary.opacity(0.3))
                                .foregroundStyle(TermTheme.card)
                                .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
                            }
                            .disabled(viewModel.sealingOrderId != nil || !viewModel.canSealOrder(selected.orderId))
                            .buttonStyle(.tactical)

                            HStack(spacing: 12) {
                                Button("REPORT_ISSUE") { onShowException(selected.orderId) }
                                    .font(.system(size: 12, weight: .bold, design: .monospaced))
                                    .padding(.vertical, 10)
                                    .frame(maxWidth: .infinity)
                                    .background(TermTheme.warn.opacity(0.1), in: RoundedRectangle(cornerRadius: 12))
                                    .foregroundStyle(TermTheme.warn)
                                    .buttonStyle(.tactical)

                                Button("RE_DISPATCH") { onShowReDispatch(selected.orderId) }
                                    .font(.system(size: 12, weight: .bold, design: .monospaced))
                                    .padding(.vertical, 10)
                                    .frame(maxWidth: .infinity)
                                    .background(TermTheme.accent.opacity(0.1), in: RoundedRectangle(cornerRadius: 12))
                                    .foregroundStyle(TermTheme.accent)
                                    .buttonStyle(.tactical)
                            }
                        }
                    }
                }
            }
        }
    }
}

struct OrderChip: View {
    let order: LiveOrder
    let selected: Bool
    let sealed: Bool
    let dispatchCode: String?
    let onTap: () -> Void

    var body: some View {
        Button(action: onTap) {
            HStack(spacing: 12) {
                if sealed {
                    Image(systemName: "checkmark.seal.fill") // Tactical seal icon
                        .font(.system(size: 20))
                        .foregroundStyle(TermTheme.live)
                } else {
                    Image(systemName: "circle.dotted")
                        .font(.system(size: 20))
                        .foregroundStyle(selected ? TermTheme.accent : TermTheme.tertiary)
                }

                VStack(alignment: .leading, spacing: 4) {
                    Text(L10n.format("mobile_payload.ui.ord_uppercased", "\(order.orderId.suffix(6).uppercased())"))
                        .font(.system(size: 14, weight: .black, design: .monospaced))
                        .foregroundStyle(TermTheme.accent)
                    
                    Text(L10n.format("mobile_payload.ui.count_units", "\((order.items ?? []).count)"))
                        .font(.system(size: 10, weight: .bold, design: .monospaced))
                        .foregroundStyle(TermTheme.secondary)
                }
                
                Spacer()
                
                if sealed, let code = dispatchCode, !code.isEmpty {
                    Text(code)
                        .font(.system(size: 16, weight: .black, design: .monospaced))
                        .foregroundStyle(TermTheme.live)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 4)
                        .background(TermTheme.live.opacity(0.12), in: Capsule())
                }
            }
            .padding(14)
            .background {
                RoundedRectangle(cornerRadius: 16, style: .continuous)
                    .fill(background)
                    .overlay {
                        if selected {
                            RoundedRectangle(cornerRadius: 16, style: .continuous)
                                .stroke(TermTheme.accent.opacity(0.2), lineWidth: 2)
                        } else {
                            RoundedRectangle(cornerRadius: 16, style: .continuous)
                                .stroke(TermTheme.separator.opacity(0.08), lineWidth: 1)
                        }
                    }
            }
        }
        .buttonStyle(.tactical)
    }

    private var background: Color {
        if sealed { return TermTheme.live.opacity(0.04) }
        if selected { return TermTheme.accent.opacity(0.08) }
        return TermTheme.card
    }
}

struct ItemRow: View {
    let checked: Bool
    let enabled: Bool
    let label: String
    let quantity: Int
    let onToggle: () -> Void

    var body: some View {
        Button(action: { if enabled { onToggle() } }) {
            HStack(spacing: 14) {
                ZStack {
                    RoundedRectangle(cornerRadius: 8, style: .continuous)
                        .fill(checked ? TermTheme.accent : TermTheme.accent.opacity(0.06))
                        .frame(width: 32, height: 32)
                    
                    if checked {
                        Image(systemName: "checkmark")
                            .font(.system(size: 14, weight: .black))
                            .foregroundStyle(TermTheme.card)
                    }
                }
                
                Text(label.uppercased())
                    .font(.system(size: 13, weight: .bold, design: .monospaced))
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .foregroundStyle(checked ? TermTheme.accent : TermTheme.secondary)
                
                Text(L10n.format("mobile_payload.ui.qty_quantity_2", "\(quantity)"))
                    .font(.system(size: 13, weight: .black, design: .monospaced))
                    .foregroundStyle(TermTheme.accent)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 4)
                    .background(TermTheme.accent.opacity(0.06), in: RoundedRectangle(cornerRadius: 6))
            }
            .padding(.vertical, 8)
            .padding(.horizontal, 12)
            .background {
                if checked {
                    RoundedRectangle(cornerRadius: 12, style: .continuous)
                        .fill(TermTheme.accent.opacity(0.03))
                }
            }
            .contentShape(Rectangle())
        }
        .buttonStyle(.tactical)
        .opacity(enabled ? 1 : 0.5)
        .disabled(!enabled)
    }
}
