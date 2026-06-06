import SwiftUI

struct EarningsView: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @State private var earnings: SupplierEarnings?
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        NavigationStack {
            ScrollView {
                Group {
                    if loading {
                        SupplierLoadingView(title: "Loading earnings…")
                    } else if let error {
                        SupplierErrorView(message: error) { Task { await load() } }
                    } else if let earnings {
                        LazyVGrid(
                            columns: [GridItem(.adaptive(minimum: horizontalSizeClass == .regular ? 220 : 160))],
                            spacing: SupplierTheme.spacingMD
                        ) {
                            KpiTile(
                                title: "Today",
                                value: MoneyFormat.minor(earnings.todayMinor, currency: earnings.currency),
                                systemImage: "sun.max",
                                tint: .accentColor
                            )
                            KpiTile(
                                title: "This week",
                                value: MoneyFormat.minor(earnings.weekMinor, currency: earnings.currency),
                                systemImage: "calendar",
                                tint: SupplierTheme.success
                            )
                            KpiTile(
                                title: "This month",
                                value: MoneyFormat.minor(earnings.monthMinor, currency: earnings.currency),
                                systemImage: "chart.bar",
                                tint: SupplierTheme.warning
                            )
                        }
                        if !earnings.authoritative {
                            Text("Scaffold data — treasury authority not connected.")
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
            .navigationTitle("Earnings")
            .task { await load() }
            .refreshable { await load(silent: true) }
        }
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        do {
            earnings = try await SupplierService.earnings()
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
        loading = false
    }
}
