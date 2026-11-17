import SwiftUI

struct PaymentConfigView: View {
    @State private var vm = PaymentConfigViewModel()

    var body: some View {
        Group {
            if vm.loading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error = vm.error {
                ContentUnavailableView {
                    Label("mobile_warehouse.ui.error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("common.action.retry") {
                        PegasusHaptics.light()
                        Task { await vm.load() }
                    }
                    .frame(minWidth: 44, minHeight: 44)
                    .contentShape(Rectangle())
                }
            } else if vm.listings.isEmpty {
                ContentUnavailableView(
                    "No Gateways",
                    systemImage: "creditcard",
                    description: Text("warehouse_portal.payment_config.text.no_payment_gateways_configured")
                )
            } else {
                List {
                    if !vm.currency.isEmpty {
                        Section {
                            LabeledContent("Pack currency", value: vm.currency)
                        }
                    }
                    Section("Pack payment catalog") {
                        ForEach(vm.listings) { listing in
                            HStack {
                                VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                    Text(listing.code)
                                        .font(.headline)
                                    Text(listing.status)
                                        .font(.subheadline)
                                        .foregroundStyle(.secondary)
                                }
                                Spacer()
                                Image(systemName: listing.selectable ? "checkmark.circle.fill" : "xmark.circle")
                                    .foregroundStyle(listing.selectable ? .green : .secondary)
                            }
                            .frame(minHeight: 44)
                            .accessibilityElement(children: .combine)
                            .accessibilityLabel("Gateway \(listing.code), status \(listing.status), \(listing.selectable ? "selectable" : "not selectable")")
                        }
                    }
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .navigationTitle("warehouse_portal.treasury.text.payment_config")
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
                .accessibilityLabel("Refresh payment config")
            }
        }
        .task { await vm.load() }
        .refreshable { await vm.load() }
    }
}
