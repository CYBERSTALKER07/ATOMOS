import SwiftUI

struct ClaimsView: View {
    @State private var vm = ClaimsViewModel()

    var body: some View {
        VStack(spacing: 0) {
            ScrollView(.horizontal, showsIndicators: false) {
                HStack {
                    ForEach(vm.statusFilters, id: \.self) { s in
                        Button(s.isEmpty ? "ALL" : s) {
                            PegasusHaptics.selection()
                            vm.status = s
                        }
                        .buttonStyle(.bordered)
                        .tint(vm.status == s ? .accentColor : .secondary)
                        .frame(minHeight: 44)
                        .contentShape(Rectangle())
                        .accessibilityLabel("Filter \(s.isEmpty ? "ALL" : s)")
                    }
                }
                .padding(.horizontal, LabTheme.spacingMD)
                .padding(.vertical, LabTheme.spacingSM)
            }
            HStack(spacing: LabTheme.spacingMD) {
                NavigationLink("Returns inbound") { ReturnsView() }
                    .frame(minHeight: 44)
                NavigationLink("Exception triage") { ExceptionsView() }
                    .frame(minHeight: 44)
            }
            .font(.subheadline)
            .padding(.horizontal, LabTheme.spacingMD)
            Text("mobile_warehouse.ui.read_only_prep_queue_approve_reject_stays_supplier_hq")
                .font(.caption)
                .foregroundStyle(.secondary)
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(.horizontal, LabTheme.spacingMD)
                .padding(.bottom, LabTheme.spacingSM)

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
                } else if vm.claims.isEmpty {
                    ContentUnavailableView("No claims in this filter", systemImage: "doc.text")
                } else {
                    List(vm.claims) { c in
                        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                            Text(L10n.format("mobile_warehouse.ui.claimtype_status", "\(c.claimType)", "\(c.status)"))
                                .font(.caption)
                                .foregroundStyle(.secondary)
                            Text(c.claimId)
                                .font(.headline.monospaced())
                            if !c.orderId.isEmpty {
                                NavigationLink("Order \(c.orderId)") {
                                    OrderDetailView(orderId: c.orderId)
                                }
                                .frame(minHeight: 44)
                            }
                            Text(L10n.format("mobile_warehouse.ui.retailer_retailerid_amountminor_currency", "\(c.retailerId)", "\(c.amountMinor)", "\(c.currency)"))
                                .font(.subheadline)
                                .foregroundStyle(.secondary)
                            ForEach(c.lineItems) { li in
                                Text(L10n.format("mobile_warehouse.ui.sku_quantity", "\(li.sku)", "\(li.quantity)") + (li.reason.isEmpty ? "" : " (\(li.reason))"))
                                    .font(.caption)
                            }
                            if !c.description.isEmpty {
                                Text(c.description)
                                    .font(.subheadline)
                            }
                        }
                        .padding(.vertical, 4)
                        .frame(minHeight: 44)
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel("Claim \(c.claimId), type \(c.claimType), status \(c.status), amount \(c.amountMinor) \(c.currency)")
                    }
                    .listStyle(.plain)
                    .scrollDismissesKeyboard(.interactively)
                }
            }
        }
        .background(LabTheme.background)
        .navigationTitle("portal.nav.claims")
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
                .accessibilityLabel("Refresh claims")
            }
        }
        .task(id: vm.status) { await vm.load() }
        .refreshable { await vm.load() }
    }
}
