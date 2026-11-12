import SwiftUI

struct ClaimsView: View {
    @State private var viewModel = ClaimsViewModel()

    private let settlementModes: [(String, String)] = [
        ("LEDGER_ONLY", "mobile_supplier.chrome.ledger_only"),
        ("STORE_CREDIT", "mobile_supplier.chrome.store_credit"),
        ("GATEWAY_REFUND", "mobile_supplier.chrome.card_refund"),
    ]

    var body: some View {
        @Bindable var vm = viewModel
        List {
            Section {
                Picker("mobile_supplier.chrome.status", selection: $vm.statusFilter) {
                    Text("OPEN").tag("OPEN")
                    Text("UNDER_REVIEW").tag("UNDER_REVIEW")
                    Text("supplier_portal.demand.signals.text.all").tag("")
                }
                .onChange(of: viewModel.statusFilter) { _, _ in
                    Task { await viewModel.load() }
                }

                Picker("mobile_supplier.chrome.settlement", selection: $vm.settlementMode) {
                    ForEach(settlementModes, id: \.0) { mode in
                        Text(LocalizedStringKey(mode.1)).tag(mode.0)
                    }
                }

                if let lastSettlement = viewModel.lastSettlement {
                    Text(L10n.format("mobile_supplier.ui.last_settle_lastsettlement_2", "\(lastSettlement)"))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }

                NavigationLink {
                    ClaimChargebacksView()
                } label: {
                    Label("mobile_supplier.ui.claim_chargebacks_ledger", systemImage: "list.bullet.rectangle")
                }
            }

            if viewModel.loading && viewModel.claims.isEmpty {
                Section {
                    ProgressView("mobile_supplier.loading.claims")
                }
            } else if let error = viewModel.error, viewModel.claims.isEmpty {
                Section {
                    Text(error).foregroundStyle(SupplierTheme.destructive)
                    Button("common.action.retry") { Task { await viewModel.load() } }
                }
            } else if viewModel.claims.isEmpty {
                Section {
                    Text("mobile_supplier.ui.no_claims_in_this_filter_retailer_post_delivery_claims_appear_wi")
                        .foregroundStyle(.secondary)
                }
            } else {
                Section("portal.nav.claims") {
                    ForEach(viewModel.claims) { claim in
                        VStack(alignment: .leading, spacing: 6) {
                            HStack {
                                Text(claim.claimType)
                                    .font(.caption.weight(.semibold))
                                Text(claim.status)
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                            Text(claim.claimId)
                                .font(.caption.monospaced())
                                .foregroundStyle(.tint)
                            Text(L10n.format("mobile_supplier.ui.order_orderid_retailer_retailerid", "\(claim.orderId)", "\(claim.retailerId)"))
                                .font(.caption)
                            Text(Self.claimAmountLabel(claim.amountMinor, currency: claim.currency))
                                .font(.subheadline.weight(.medium))
                            if let description = claim.description, !description.isEmpty {
                                Text(description).font(.caption).foregroundStyle(.secondary)
                            }
                            if claim.status == "OPEN" || claim.status == "UNDER_REVIEW" {
                                HStack {
                                    Button(L10n.format("mobile_supplier.ui.approve_n_1_settlementmode", L10n.format(settlementModes.first { $0.0 == viewModel.settlementMode }?.1 ?? viewModel.settlementMode))) {
                                        Task { await viewModel.approve(claim.claimId) }
                                    }
                                    .buttonStyle(.borderedProminent)
                                    .frame(minHeight: 44)
                                    .disabled(viewModel.busyId == claim.claimId)

                                    Button("mobile_supplier.ui.reject", role: .destructive) {
                                        Task { await viewModel.reject(claim.claimId) }
                                    }
                                    .frame(minHeight: 44)
                                    .disabled(viewModel.busyId == claim.claimId)
                                }
                                .padding(.top, 4)
                            }
                        }
                        .padding(.vertical, 4)
                    }
                }
            }
        }
        .navigationTitle("portal.nav.claims")
        .refreshable { await viewModel.load(silent: true) }
        .task { await viewModel.load() }
    }

    static func claimAmountLabel(_ amount: Int64?, currency: String?) -> String {
        guard let amount else { return MissingNumber.label(nil as Int64?) }
        return MoneyFormat.minor(amount, currency: displayPackCurrency(currency))
    }
}
