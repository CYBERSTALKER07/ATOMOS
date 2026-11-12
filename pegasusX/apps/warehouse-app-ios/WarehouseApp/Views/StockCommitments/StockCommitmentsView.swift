import SwiftUI

struct StockCommitmentsView: View {
    @State private var vm = StockCommitmentsViewModel()

    var body: some View {
        Group {
            if vm.loading && vm.commitments.isEmpty {
                WarehouseLoadingView(
                    title: "Loading stock commitments",
                    message: "Calculating pre-order allocations."
                )
            } else if let error = vm.error {
                WarehouseErrorView(message: error) {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                }
            } else if vm.commitments.isEmpty {
                WarehouseEmptyView(
                    title: "No commitments",
                    message: "No stock is currently reserved for pre-orders."
                )
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(vm.commitments) { row in
                        StockCommitmentCard(row: row)
                            .listRowInsets(EdgeInsets(top: 8, leading: 16, bottom: 8, trailing: 16))
                            .listRowSeparator(.hidden)
                            .listRowBackground(Color.clear)
                    }
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .navigationTitle("mobile_warehouse.ui.stock_commitments")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                } label: {
                    Label("portal.page.orders.action.refresh", systemImage: "arrow.clockwise")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Refresh stock commitments")
                .disabled(vm.loading)
            }
        }
        .task { await vm.load() }
        .refreshable { await vm.load(silent: true) }
    }
}

private struct StockCommitmentCard: View {
    let row: StockCommitmentRow

    var body: some View {
        VStack(spacing: LabTheme.spacingMD) {
            HStack(alignment: .top, spacing: LabTheme.spacingMD) {
                if let urlString = row.imageUrl, let url = URL(string: urlString) {
                    AsyncImage(url: url) { phase in
                        switch phase {
                        case .empty:
                            ProgressView()
                                .frame(width: 60, height: 60)
                        case .success(let image):
                            image.resizable()
                                .aspectRatio(contentMode: .fit)
                                .frame(width: 60, height: 60)
                                .clipShape(RoundedRectangle(cornerRadius: LabTheme.radiusSM, style: .continuous))
                        case .failure:
                            Image(systemName: "photo")
                                .frame(width: 60, height: 60)
                                .foregroundStyle(.secondary)
                        @unknown default:
                            EmptyView()
                        }
                    }
                } else {
                    Image(systemName: "shippingbox")
                        .resizable()
                        .aspectRatio(contentMode: .fit)
                        .frame(width: 40, height: 40)
                        .foregroundStyle(.secondary)
                        .frame(width: 60, height: 60)
                        .background(Color(uiColor: .systemGray6))
                        .clipShape(RoundedRectangle(cornerRadius: LabTheme.radiusSM, style: .continuous))
                }

                VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                    Text(row.name ?? "SKU: \(row.skuId.prefix(8))")
                        .font(.headline)
                        .lineLimit(2)
                    Text(row.skuId)
                        .font(.caption.monospaced())
                        .foregroundStyle(.secondary)
                }
                Spacer()
            }

            VStack(spacing: 0) {
                metricRow("On Hand", "\(row.onHand)", isHeader: true)
                Divider()
                metricRow("Reserved (ASAP)", "\(row.reservedAsap)", valueColor: .orange)
                Divider()
                metricRow("Reserved (Scheduled)", "\(row.reservedScheduled)", valueColor: .blue)
                Divider()
                metricRow("Available", "\(row.availableQty)", valueColor: row.availableQty < 0 ? .red : .green)
                if row.deficitQty > 0 {
                    Divider()
                    metricRow("Deficit", "\(row.deficitQty)", valueColor: .red)
                }
            }
            .background(Color(uiColor: .systemBackground))
            .clipShape(RoundedRectangle(cornerRadius: LabTheme.radiusMD, style: .continuous))
            .overlay(
                RoundedRectangle(cornerRadius: LabTheme.radiusMD, style: .continuous)
                    .stroke(Color(uiColor: .systemGray5), lineWidth: 1)
            )
        }
        .padding()
        .background(Color(uiColor: .secondarySystemBackground))
        .clipShape(RoundedRectangle(cornerRadius: LabTheme.radiusLG, style: .continuous))
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(row.name ?? row.skuId), on hand \(row.onHand), available \(row.availableQty), deficit \(row.deficitQty)")
    }

    @ViewBuilder
    private func metricRow(_ label: String, _ value: String, isHeader: Bool = false, valueColor: Color = .primary) -> some View {
        HStack {
            Text(label)
                .font(isHeader ? .subheadline.bold() : .subheadline)
                .foregroundStyle(isHeader ? .primary : .secondary)
            Spacer()
            Text(value)
                .font(.subheadline.monospaced().bold())
                .foregroundStyle(valueColor)
        }
        .padding(.horizontal, LabTheme.spacingMD)
        .padding(.vertical, LabTheme.spacingSM)
    }
}
