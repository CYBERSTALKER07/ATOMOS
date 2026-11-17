import SwiftUI

struct EarningsView: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @State private var viewModel = EarningsViewModel()

    var body: some View {
        ScrollView {
            Group {
                if viewModel.loading && viewModel.earnings == nil {
                    SupplierLoadingView(title: "mobile_supplier.loading.earnings")
                } else if let error = viewModel.error, viewModel.earnings == nil {
                    SupplierErrorView(message: error) { Task { await viewModel.load() } }
                } else if let earnings = viewModel.earnings {
                    LazyVGrid(
                        columns: [GridItem(.adaptive(minimum: horizontalSizeClass == .regular ? 220 : 160))],
                        spacing: SupplierTheme.spacingMD
                    ) {
                        KpiTile(
                            title: "mobile_supplier.chrome.today",
                            value: earningsLabel(earnings.todayMinor, earnings),
                            systemImage: "sun.max",
                            tint: SupplierTheme.primaryAction
                        )
                        KpiTile(
                            title: "mobile_supplier.chrome.this_week",
                            value: earningsLabel(earnings.weekMinor, earnings),
                            systemImage: "calendar",
                            tint: SupplierTheme.success
                        )
                        KpiTile(
                            title: "mobile_supplier.chrome.this_month",
                            value: earningsLabel(earnings.monthMinor, earnings),
                            systemImage: "chart.bar",
                            tint: SupplierTheme.warning
                        )
                    }
                    if !earnings.authoritative {
                        Text("mobile_supplier.ui.scaffold_data_treasury_authority_not_connected")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .padding(.top, SupplierTheme.spacingSM)
                    }
                }
            }
            .supplierReadableWidth()
            .padding()
        }
        .background(SupplierTheme.background)
        .navigationTitle("portal.nav.earnings")
        .task { await viewModel.load() }
        .refreshable { await viewModel.load(silent: true) }
    }

    private func earningsLabel(_ minor: Int64, _ earnings: SupplierEarnings) -> String {
        guard let amount = EarningsHonesty.minor(minor, authoritative: earnings.authoritative) else {
            return "—"
        }
        return MoneyFormat.minor(amount, currency: earnings.currency)
    }
}
