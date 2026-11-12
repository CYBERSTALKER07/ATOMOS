import SwiftUI

struct TreasuryView: View {
    @State private var vm = TreasuryViewModel()

    private var gridMin: CGFloat { 160 }

    var body: some View {
        @Bindable var vm = vm
        VStack(spacing: 0) {
            Picker("View", selection: $vm.selectedSegment) {
                Text("portal.nav.overview").tag(0)
                Text("warehouse_portal.treasury.text.invoices").tag(1)
                Text("mobile_warehouse.ui.payment").tag(2)
            }
            .pickerStyle(.segmented)
            .padding()
            .onChange(of: vm.selectedSegment) { _, _ in
                PegasusHaptics.selection()
            }

            if vm.loading {
                WarehouseLoadingView(
                    title: "Loading treasury",
                    message: "Fetching balance, receivables, and invoice status."
                )
            } else if let error = vm.error {
                WarehouseErrorView(message: error) {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                }
            } else {
                switch vm.selectedSegment {
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
                                KpiTile(
                                    title: "Balance",
                                    value: "\(vm.overview.balance.formatted()) \(packCurrency(MarketPackStore.pack))",
                                    systemImage: "banknote",
                                    tint: .accentColor
                                )
                                KpiTile(
                                    title: "Receivable",
                                    value: "\(vm.overview.totalReceivable.formatted()) \(packCurrency(MarketPackStore.pack))",
                                    systemImage: "arrow.down.circle",
                                    tint: LabTheme.warning
                                )
                                KpiTile(
                                    title: "Collected",
                                    value: "\(vm.overview.totalCollected.formatted()) \(packCurrency(MarketPackStore.pack))",
                                    systemImage: "checkmark.circle",
                                    tint: LabTheme.success
                                )
                                KpiTile(
                                    title: "Overdue",
                                    value: "\(vm.overview.overdueAmount.formatted()) \(packCurrency(MarketPackStore.pack))",
                                    systemImage: "exclamationmark.triangle",
                                    tint: LabTheme.destructive,
                                    chip: vm.overview.overdueAmount > 0 ? ("ALERT", LabTheme.destructive) : nil
                                )
                            }
                        }
                        .labReadableWidth()
                        .padding()
                    }
                    .scrollDismissesKeyboard(.interactively)
                case 1:
                    TreasuryTransactionList(invoices: vm.invoices)
                default:
                    PaymentConfigView()
                }
            }
        }
        .background(LabTheme.background)
        .navigationTitle("portal.nav.treasury")
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
                .accessibilityLabel("Refresh treasury")
            }
        }
        .task { await vm.load() }
        .refreshable { await vm.load() }
    }
}
