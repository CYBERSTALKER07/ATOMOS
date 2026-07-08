import SwiftUI

struct StockCommitmentsView: View {
    @State private var commitments: [StockCommitmentRow] = []
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        Group {
            if loading && commitments.isEmpty {
                WarehouseLoadingView(
                    title: "Loading stock commitments",
                    message: "Calculating pre-order allocations."
                )
            } else if let error {
                WarehouseErrorView(message: error) {
                    Task { await load() }
                }
            } else if commitments.isEmpty {
                WarehouseEmptyView(
                    title: "No commitments",
                    message: "No stock is currently reserved for pre-orders."
                )
            } else {
                List(commitments) { row in
                    StockCommitmentCard(row: row)
                        .listRowInsets(EdgeInsets(top: 8, leading: 16, bottom: 8, trailing: 16))
                        .listRowSeparator(.hidden)
                        .listRowBackground(Color.clear)
                }
                .listStyle(.plain)
            }
        }
        .navigationTitle("Stock Commitments")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("Refresh", systemImage: "arrow.clockwise") {
                    Task { await load() }
                }
                .disabled(loading)
            }
        }
        .task { await load() }
        .refreshable { await load(silent: true) }
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        do {
            let response = try await WarehouseService.stockCommitments()
            // Prefer items, fallback to skus depending on API shape returned
            commitments = response.items.isEmpty ? response.skus : response.items
        } catch {
            self.error = error.localizedDescription
        }
        if !silent { loading = false }
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
                                .cornerRadius(LabTheme.radiusSM)
                        case .failure:
                            Image(systemName: "photo")
                                .frame(width: 60, height: 60)
                                .foregroundColor(.secondary)
                        @unknown default:
                            EmptyView()
                        }
                    }
                } else {
                    Image(systemName: "shippingbox")
                        .resizable()
                        .aspectRatio(contentMode: .fit)
                        .frame(width: 40, height: 40)
                        .foregroundColor(.secondary)
                        .frame(width: 60, height: 60)
                        .background(Color(uiColor: .systemGray6))
                        .cornerRadius(LabTheme.radiusSM)
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
            .cornerRadius(LabTheme.radiusMD)
            .overlay(
                RoundedRectangle(cornerRadius: LabTheme.radiusMD)
                    .stroke(Color(uiColor: .systemGray5), lineWidth: 1)
            )
        }
        .padding()
        .background(Color(uiColor: .secondarySystemBackground))
        .cornerRadius(LabTheme.radiusLG)
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
