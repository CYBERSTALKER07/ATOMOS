import SwiftUI

struct TreasuryView: View {
    @State private var overview = TreasuryOverview.empty
    @State private var invoices: [Invoice] = []
    @State private var loading = true
    @State private var error: String?
    @State private var selectedSegment = 0

    private var gridMin: CGFloat { 160 }

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
                    WarehouseLoadingView(
                        title: "Loading treasury",
                        message: "Fetching balance, receivables, and invoice status."
                    )
                } else if let error {
                    WarehouseErrorView(message: error) { load() }
                } else {
                    switch selectedSegment {
                    case 0:
                        ScrollView {
                            VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
                                WarehouseSectionHeader(
                                    title: "Financial overview",
                                    subtitle: "Treasury KPIs for this warehouse"
                                )
                                LazyVGrid(
                                    columns: [GridItem(.adaptive(minimum: gridMin), spacing: LabTheme.spacingMD)],
                                    spacing: LabTheme.spacingMD
                                ) {
                                    KpiTile(title: "Balance", value: "\(overview.balance.formatted()) UZS", systemImage: "banknote", tint: .accentColor)
                                    KpiTile(title: "Receivable", value: "\(overview.totalReceivable.formatted()) UZS", systemImage: "arrow.down.circle", tint: LabTheme.warning)
                                    KpiTile(title: "Collected", value: "\(overview.totalCollected.formatted()) UZS", systemImage: "checkmark.circle", tint: LabTheme.success)
                                    KpiTile(
                                        title: "Overdue",
                                        value: "\(overview.overdueAmount.formatted()) UZS",
                                        systemImage: "exclamationmark.triangle",
                                        tint: LabTheme.destructive,
                                        chip: overview.overdueAmount > 0 ? ("ALERT", LabTheme.destructive) : nil
                                    )
                                }
                            }
                            .labReadableWidth()
                            .padding()
                        }
                    case 1:
                        if invoices.isEmpty {
                            WarehouseEmptyView(title: "No Invoices", message: "No invoices found for this warehouse.")
                        } else {
                            ResponsiveGridContentWrapper {
                                ForEach(invoices) { inv in
                                HStack(alignment: .top) {
                                    VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                        Text(inv.retailerName)
                                            .font(.headline)
                                        Text("\(inv.amountUzs.formatted()) \(inv.currency) · Due: \(inv.dueDate)")
                                            .font(.subheadline)
                                            .foregroundStyle(.secondary)
                                        let ownerType = inv.payoutOwnerType.isEmpty ? "SUPPLIER" : inv.payoutOwnerType
                                        let ownerID = inv.payoutOwnerId.isEmpty ? "" : String(inv.payoutOwnerId.prefix(8))
                                        Text("Owner \(ownerType)\(ownerID.isEmpty ? "" : ":\(ownerID)") · Fee \(inv.feeAmount.formatted()) · Net \(inv.netPayoutAmount.formatted())")
                                            .font(.caption)
                                            .foregroundStyle(.secondary)
                                    }
                                    Spacer()
                                    WarehouseStatusBadge(text: inv.status)
                                }
                            }
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
                async let invoicesResponse = WarehouseService.treasuryInvoices()
                do {
                    overview = try await WarehouseService.treasuryOverview()
                } catch {
                    let financials = try await WarehouseOperationsService.opsFinancials()
                    overview = TreasuryOverview(
                        balance: Int(financials.netPayout),
                        totalReceivable: Int(financials.totalRevenue),
                        totalCollected: Int(financials.cashCollected),
                        overdueAmount: Int(financials.cashPending)
                    )
                }
                invoices = try await invoicesResponse.invoices
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }
}
