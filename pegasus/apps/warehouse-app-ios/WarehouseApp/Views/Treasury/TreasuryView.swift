import SwiftUI

struct TreasuryView: View {
    @State private var overview = TreasuryOverview.empty
    @State private var invoices: [Invoice] = []
    @State private var loading = true
    @State private var error: String?
    @State private var selectedSegment = 0

    private let columns = [
        GridItem(.flexible(), spacing: LabTheme.spacingMD),
        GridItem(.flexible(), spacing: LabTheme.spacingMD),
    ]

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                Picker("View", selection: $selectedSegment) {
                    Text("Overview").tag(0)
                    Text("Invoices").tag(1)
                    Text("Payment").tag(2)
                }
                .pickerStyle(.segmented)
                .padding()

                if loading {
                    ProgressView()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let error {
                    ContentUnavailableView {
                        Label("Error", systemImage: "exclamationmark.triangle")
                    } description: {
                        Text(error)
                    } actions: {
                        Button("Retry") { load() }
                    }
                } else {
                    switch selectedSegment {
                    case 0:
                        ScrollView {
                            LazyVGrid(columns: columns, spacing: LabTheme.spacingMD) {
                                TreasuryKpiCard(title: "Balance", value: "\(overview.balance.formatted()) UZS", icon: "banknote", index: 0)
                                TreasuryKpiCard(title: "Receivable", value: "\(overview.totalReceivable.formatted()) UZS", icon: "arrow.down.circle", index: 1)
                                TreasuryKpiCard(title: "Collected", value: "\(overview.totalCollected.formatted()) UZS", icon: "checkmark.circle", index: 2)
                                TreasuryKpiCard(title: "Overdue", value: "\(overview.overdueAmount.formatted()) UZS", icon: "exclamationmark.triangle", index: 3)
                            }
                            .padding()
                        }
                    case 1:
                        if invoices.isEmpty {
                            ContentUnavailableView("No Invoices", systemImage: "doc.text", description: Text("No invoices found"))
                        } else {
                            List(invoices) { inv in
                                HStack {
                                    VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                        Text(inv.retailerName)
                                            .font(.headline)
                                        Text("\(inv.amountUzs.formatted()) UZS · Due: \(inv.dueDate)")
                                            .font(.subheadline)
                                            .foregroundStyle(.secondary)
                                    }
                                    Spacer()
                                    Text(inv.status)
                                        .font(.caption.bold())
                                        .padding(.horizontal, LabTheme.spacingSM)
                                        .padding(.vertical, LabTheme.spacingXS)
                                        .background(.quaternary, in: Capsule())
                                }
                            }
                            .listStyle(.insetGrouped)
                        }
                    default:
                        PaymentConfigView()
                    }
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Treasury")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") { load() }
                }
            }
            .task { load() }
            .refreshable { load() }
        }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                async let o = WarehouseService.treasuryOverview()
                async let i = WarehouseService.treasuryInvoices()
                overview = try await o
                invoices = try await i.invoices
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }
}

private struct TreasuryKpiCard: View {
    let title: String
    let value: String
    let icon: String
    let index: Int

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            Image(systemName: icon)
                .font(.title3)
                .foregroundStyle(.secondary)
            Spacer(minLength: 0)
            Text(value)
                .font(.title3.bold())
            Text(title)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
        .staggeredAppear(index: index)
    }
}
